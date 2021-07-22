package common

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type DiscoverServer struct {
	mu        sync.Mutex
	address   string
	timeout   time.Duration
	uid       string
	info      string
	lifecycle *Notice
	listener  net.PacketConn
}

const (
	maxInfoLength = 1024
)

func NewDiscoverServer(address string, timeout time.Duration, uid string, info string) (*DiscoverServer, error) {
	if len(info) > maxInfoLength {
		return nil, fmt.Errorf("max UDP info length exceeded. max length expected: %d received: %d", maxInfoLength, len(info))
	}

	return &DiscoverServer{
		mu:        sync.Mutex{},
		address:   address,
		timeout:   timeout,
		uid:       uid,
		info:      info,
		lifecycle: NewNotice(),
		listener:  nil,
	}, nil
}

func (server *DiscoverServer) Start() error {
	server.mu.Lock()
	defer server.mu.Unlock()

	DebugFunc(*server)

	b := make([]byte, maxInfoLength)

	var err error

	server.listener, err = net.ListenPacket("udp4", server.address)
	if Error(err) {
		return err
	}

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine())

	loop:
		for server.lifecycle.isSet {
			select {
			case <-server.lifecycle.NewChannel():
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
						if server.lifecycle.IsSet() {
							Error(err)
						}

						break
					}
				}

				host, err := SplitHost(peer.String())
				if Error(err) {
					break
				}

				remote := net.ParseIP(host)
				if remote == nil {
					Error(fmt.Errorf("cannot parse ip: %s", host))

					break
				}

				addrs, err := GetHostInfos(true, false, remote)
				if Error(err) {
					break
				}

				info := server.info

				if len(addrs) > 0 {
					host, _, err := net.ParseCIDR(addrs[0].Addr.String())
					if Error(err) {
						break
					}

					info = strings.Replace(info, "<host>", host.String(), 1)
				}

				receivedUID := string(b[:n])

				Debug("received UDP broadcast from %+v: %s\n", peer, receivedUID)

				b, err := EqualWildcards(server.uid, receivedUID)
				if Error(err) {
					continue
				}

				if !b {
					Debug("not matching uid, expected: %s received: %s -> ignore", server.uid, receivedUID)

					break
				}

				Debug("answer positive discover with info %s to %+v", info, peer)

				if _, err := server.listener.WriteTo([]byte(info), peer); err != nil {
					Error(err)
				}
			}
		}
	}()

	return nil
}

func (server *DiscoverServer) Stop() error {
	server.mu.Lock()
	defer server.mu.Unlock()

	DebugFunc(*server)

	if server.lifecycle != nil {
		server.lifecycle.Unset()
	}

	if server.listener != nil {
		err := server.listener.Close()
		if Error(err) {
			return err
		}
	}

	return nil
}

func Discover(address string, timeout time.Duration, uid string) (map[string]string, error) {
	DebugFunc("discover uid: %s", uid)

	discoverIp, discoverPort, err := net.SplitHostPort(address)
	if Error(err) {
		return nil, err
	}

	discoveredIps := make(map[string]string)

	addrs, err := GetHostInfos(false, true, nil)
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

	for _, addr := range addrs {
		ip, ipNet, err := net.ParseCIDR(addr.Addr.String())
		if Error(err) {
			return nil, err
		}

		ip = ip.To4()

		if ip == nil {
			continue
		}

		if discoverIp != "" && ip.String() != discoverIp {
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
		for _, e := range errs.GetAll() {
			Error(e)
		}

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
