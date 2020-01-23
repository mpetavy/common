package common

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type DiscoverServer struct {
	address  string
	timeout  time.Duration
	uid      string
	info     string
	quitCh   chan struct{}
	listener net.PacketConn
}

const (
	maxInfoLength = 1024
)

func NewDiscoverServer(address string, timeout time.Duration, uid string, info string) (*DiscoverServer, error) {
	if len(info) > maxInfoLength {
		return nil, fmt.Errorf("max UDP info length exceeded. max length expected: %d received: %d", maxInfoLength, len(info))
	}

	return &DiscoverServer{address: address, timeout: timeout, uid: uid, info: info}, nil
}

func (server *DiscoverServer) Start() error {
	DebugFunc(*server)

	if server.quitCh != nil {
		return fmt.Errorf("DiscoverServer already started")
	}

	b := make([]byte, maxInfoLength)

	var err error

	server.listener, err = net.ListenPacket("udp4", server.address)
	if Error(err) {
		return err
	}

	server.quitCh = make(chan struct{})

	go func() {
	loop:
		for AppLifecycle().IsSet() {
			select {
			case <-server.quitCh:
				break loop
			default:
				err := server.listener.SetDeadline(DeadlineByDuration(server.timeout))
				if Error(err) {
					break
				}

				n, peer, err := server.listener.ReadFrom(b)
				if err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						break
					} else {
						_, isOpen := <-server.quitCh

						if AppLifecycle().IsSet() && isOpen {
							Error(err)
						}

						break
					}
				}

				receivedUID := string(b[:n])

				Debug("received UDP broadcast from %+v: %s\n", peer, receivedUID)

				if receivedUID != server.uid {
					Debug("not matching uid, expected: %s received: %s -> ignore", server.uid, receivedUID)

					break
				}

				Debug("answer positive discover with info %s to %+v", server.info, peer)

				if _, err := server.listener.WriteTo([]byte(server.info), peer); err != nil {
					Error(err)
				}
			}
		}
	}()

	return nil
}

func (server *DiscoverServer) Stop() error {
	DebugFunc(*server)

	close(server.quitCh)

	Error(server.listener.Close())

	return nil
}

func Discover(address string, timeout time.Duration, uid string) (map[string]string, error) {
	DebugFunc("discover uid: %s", uid)

	_, discoverPort, err := net.SplitHostPort(address)
	if Error(err) {
		return nil, err
	}

	discoveredIps := make(map[string]string)

	ips, err := GetActiveIPs(true)
	if Error(err) {
		return nil, err
	}

	var wg sync.WaitGroup
	var errs ChannelError

	c, err := net.ListenPacket("udp4", ":0")
	if Error(err) {
		return nil, err
	}
	defer func() {
		Ignore(c.Close())
	}()

	for _, localIp := range ips {
		ip, ipNet, err := net.ParseCIDR(localIp)
		if Error(err) {
			return nil, err
		}

		ip = ip.To4()

		if ip == nil {
			continue
		}

		wg.Add(1)

		go func(ip net.IP, ipNet *net.IPNet) {
			defer wg.Done()

			ones, bits := ipNet.Mask.Size()
			mask := net.CIDRMask(ones, bits)

			broadcast := net.IP(make([]byte, 4))
			for i := range ip {
				broadcast[i] = ip[i] | ^mask[i]
			}

			Debug("UDP broadcast: %v for ip: %v on port: %s", broadcast.String(), ipNet, discoverPort)

			dst, err := net.ResolveUDPAddr("udp4", broadcast.String()+":"+discoverPort)
			if err != nil {
				errs.Add(err)

				return
			}

			if _, err := c.WriteTo([]byte(uid), dst); err != nil {
				errs.Add(err)

				return
			}
		}(ip, ipNet)
	}

	wg.Wait()

	if errs.Exists() {
		Error(errs.Get())

		return nil, errs.Get()
	}

	Debug("reading answers ...")

	b := make([]byte, maxInfoLength)
	for {
		err := c.SetDeadline(DeadlineByDuration(timeout))
		if Error(err) {
			break
		}

		n, peer, err := c.ReadFrom(b)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				break
			} else {
				return discoveredIps, err
			}
		}

		host, port, err := net.SplitHostPort(peer.String())
		if Error(err) {
			continue
		}

		info := string(b[:n])

		info = strings.ReplaceAll(info, "$host", host)
		info = strings.ReplaceAll(info, "$port", port)
		info = strings.ReplaceAll(info, "$address", peer.String())

		discoveredIps[peer.String()] = info

		Debug("%d bytes read from %s: %s\n", n, peer.String(), info)
	}

	return discoveredIps, nil
}
