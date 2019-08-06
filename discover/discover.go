package discover

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/mpetavy/common"
)

type Server struct {
	address     string
	readTimeout time.Duration
	uid         string
	info        string
	quitCh      chan struct{}
}

var (
	discoverAddress     *string
	discoverReadTimeout *time.Duration
	discoverUID         *string
	discoverInfo        *string
)

func init() {
	discoverAddress = flag.String("discover.address", ":9999", "discover address")
	discoverReadTimeout = flag.Duration("discover.readtimeout", time.Millisecond*1000, "discover read timeout")
	discoverUID = flag.String("discover.uid", "my-uid", "discover uid")
	discoverInfo = flag.String("discover.info", "my-info", "discover info")
}

func NewServer(address string, readTimeout time.Duration, uid string, info string) Server {
	address = common.Eval(len(address) != 0, address, *discoverAddress).(string)
	readTimeout = common.Eval(readTimeout != 0, readTimeout, *discoverReadTimeout).(time.Duration)
	uid = common.Eval(len(uid) != 0, uid, *discoverUID).(string)
	info = common.Eval(len(info) != 0, info, *discoverInfo).(string)

	return Server{address: address, readTimeout: readTimeout, uid: uid, info: info}
}

func (server *Server) Start() error {
	common.DebugFunc("discover server: %+v", *server)

	if server.quitCh != nil {
		return fmt.Errorf("Server already started")
	}

	b := make([]byte, len(server.uid))

	c, err := net.ListenPacket("udp4", server.address)
	if err != nil {
		return err
	}

	server.quitCh = make(chan struct{})

	go func() {
		for {
			select {
			case <-server.quitCh:
				break
			default:
				c.SetReadDeadline(time.Now().Add(*discoverReadTimeout))

				n, peer, err := c.ReadFrom(b)
				if err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						break
					} else {
						common.DebugError(err)

						break
					}
				}

				receivedUID := string(b[:n])

				common.Debug("received UDP broadcast from %+v: %s\n", peer, receivedUID)

				if receivedUID != server.uid {
					common.Debug("not matching uid, expected: %s received:%s -> ignore", server.uid, receivedUID)

					break
				}

				common.Debug("answer positive discover with info %s to %+v", server.info, peer)

				if _, err := c.WriteTo([]byte(server.info), peer); err != nil {
					common.Error(err)
				}
			}
		}
	}()

	return nil
}

func (server *Server) Stop() error {
	if server.quitCh == nil {
		return fmt.Errorf("Server already stopped")
	}

	close(server.quitCh)

	server.quitCh = nil

	common.DebugFunc("discover server: %+v", *server)

	return nil
}

func Discover(uid string) (map[string]string, error) {
	uid = common.Eval(len(uid) != 0, uid, *discoverUID).(string)

	common.DebugFunc("discover uid: %s", uid)

	localIps, err := common.FindActiveIPs()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	discoveredIps := make(map[string]string)

	c, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return discoveredIps, err
	}
	defer c.Close()

	_, discoverPort, err := net.SplitHostPort(*discoverAddress)
	if err != nil {
		return discoveredIps, err
	}

	for _, localIp := range localIps {
		ipv4Addr, ipv4Net, err := net.ParseCIDR(localIp)
		if err != nil {
			panic(err)
		}

		ipv4Addr = ipv4Addr.To4()

		if ipv4Addr == nil {
			continue
		}

		wg.Add(1)

		go func(ipv4Addr net.IP, ipv4Net *net.IPNet) {
			defer wg.Done()

			ones, bits := ipv4Net.Mask.Size()
			mask := net.CIDRMask(ones, bits)

			broadcast := net.IP(make([]byte, 4))
			for i := range ipv4Addr {
				broadcast[i] = ipv4Addr[i] | ^mask[i]
			}

			common.Debug("UDP broadcast: %v for ip: %v on port: %s", broadcast.String(), ipv4Net, discoverPort)

			dst, err := net.ResolveUDPAddr("udp4", broadcast.String()+":"+discoverPort)
			if err != nil {
				log.Fatal(err)
			}

			if _, err := c.WriteTo([]byte(uid), dst); err != nil {
				log.Fatal(err)
			}
		}(ipv4Addr, ipv4Net)
	}

	wg.Wait()

	common.Debug("reading answers ...")

	b := make([]byte, 512)
	c.SetReadDeadline(time.Now().Add(*discoverReadTimeout))
	for {
		n, peer, err := c.ReadFrom(b)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				break
			} else {
				log.Fatal(err)
			}
		}

		client := peer.String()
		info := string(b[:n])

		discoveredIps[client] = info

		common.Debug("%d bytes read from %+v: %s\n", n, client, info)
	}

	return discoveredIps, nil
}
