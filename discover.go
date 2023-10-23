package common

import (
	"fmt"
	"net"
	"sort"
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
		lifecycle: NewNotice(true),
		listener:  nil,
	}, nil
}

func (server *DiscoverServer) Start() error {
	server.mu.Lock()
	defer server.mu.Unlock()

	DebugFunc(server)

	b := make([]byte, maxInfoLength)

	var err error

	server.listener, err = net.ListenPacket("udp4", server.address)
	if Error(err) {
		return err
	}

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		lifecycleCh := server.lifecycle.NewChannel()
		defer server.lifecycle.RemoveChannel(lifecycleCh)

	loop:
		for server.lifecycle.IsSet() {
			select {
			case <-lifecycleCh:
				break loop
			default:
				err := server.listener.SetReadDeadline(CalcDeadline(time.Now(), server.timeout))
				if Error(err) {
					break
				}

				n, peer, err := server.listener.ReadFrom(b)
				if err != nil {
					if IsErrTimeout(err) {
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

				receivedUID := string(b[:n])

				Debug("received UDP broadcast from %+v: %s\n", peer, receivedUID)

				b, err := EqualWildcards(server.uid, receivedUID)
				if Error(err) {
					break
				}

				if !b {
					Debug("not matching uid, expected: %s received: %s -> ignore", server.uid, receivedUID)

					break
				}

				_, _, hostInfos, err := GetHostInfos()
				if Error(err) {
					break
				}

				var local net.IP

				for _, hostInfo := range hostInfos {
					if hostInfo.IPNet.Contains(remote) {
						local = hostInfo.IPNet.IP

						break
					}
				}

				info := server.info
				if local != nil {
					info = strings.ReplaceAll(info, "<host>", local.String())
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

	DebugFunc(server)

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

func Discover(address string, timeout time.Duration, uid string) ([]string, error) {
	DebugFunc("discover uid: %s", uid)

	_, discoverPort, err := net.SplitHostPort(address)
	if Error(err) {
		return nil, err
	}

	list := make([]string, 0)

	_, _, hostInfos, err := GetHostInfos()
	if Error(err) {
		return nil, err
	}

	var wg sync.WaitGroup
	errs := NewSync(new(error))

	c, err := net.ListenPacket("udp4", ":0")
	if Error(err) {
		return nil, err
	}
	defer func() {
		DebugError(c.Close())
	}()

	for _, hostInfo := range hostInfos {
		if hostInfo.Intf.Flags&net.FlagBroadcast == 0 {
			continue
		}

		ip := hostInfo.IPNet.IP
		if !IsIPv4(ip) {
			continue
		}

		ip = ip.To4()

		wg.Add(1)

		go func(hostInfo HostInfo) {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			defer wg.Done()

			ones, bits := hostInfo.IPNet.Mask.Size()
			mask := net.CIDRMask(ones, bits)

			broadcast := net.IP(make([]byte, 4))
			for i := range ip {
				broadcast[i] = ip[i] | ^mask[i]
			}

			Debug("UDP broadcast: %v for ip: %v on port: %s", broadcast.String(), hostInfo.IPNet, discoverPort)

			dst, err := net.ResolveUDPAddr("udp4", broadcast.String()+":"+discoverPort)
			if err != nil {
				errs.Set(&err)

				return
			}

			if _, err := c.WriteTo([]byte(uid), dst); err != nil {
				errs.Set(&err)

				return
			}
		}(hostInfo)
	}

	wg.Wait()

	if errs.Get() != nil {
		return nil, *errs.Get()
	}

	Debug("reading answers ...")

	b := make([]byte, maxInfoLength)
	for {
		err := c.SetReadDeadline(CalcDeadline(time.Now(), timeout))
		if Error(err) {
			break
		}

		n, peer, err := c.ReadFrom(b)
		if err != nil {
			if IsErrTimeout(err) {
				break
			} else {
				return list, err
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

		list = append(list, info)

		Debug("%d bytes read from %s: %s\n", n, peer.String(), info)
	}

	sort.Strings(list)

	return list, nil
}
