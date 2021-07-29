package common

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	LOCALHOST_IP4 = "127.0.0.1"
	LOCALHOST_IP6 = "::1"
)

const (
	FlagNameIoPrimaryIface     = "io.primary.iface"
	FlagNameIoNetworkTimeout   = "io.network.timeout"
	FlagNameIoConnectTimeout   = "io.connect.timeout"
	FlagNameIoReadwriteTimeout = "io.readwrite.timeout"
)

var (
	FlagIoPrimaryIface     = flag.String(FlagNameIoPrimaryIface, "", "ethernet interface holding primary ip")
	FlagIoNetworkTimeout   = flag.Int(FlagNameIoNetworkTimeout, 10*1000, "network server and client dial timeout")
	FlagIoConnectTimeout   = flag.Int(FlagNameIoConnectTimeout, 3*1000, "network server and client dial timeout")
	FlagIoReadwriteTimeout = flag.Int(FlagNameIoReadwriteTimeout, 30*60*1000, "network read/write timeout")
)

func DeadlineByMsec(msec int) time.Time {
	if msec > 0 {
		return time.Now().Add(time.Duration(msec) * time.Millisecond)
	} else {
		return time.Time{}
	}
}

func DeadlineByDuration(duration time.Duration) time.Time {
	if duration > 0 {
		return time.Now().Add(duration)
	} else {
		return time.Time{}
	}
}

type HostInfo struct {
	IFace net.Interface
	Mac   string
	IP    string
	IPNet *net.IPNet
	Addr  net.Addr
}

func GetHostInfo() (*HostInfo, string, error) {
	DebugFunc("...")

	hostName, err := os.Hostname()
	if Error(err) {
		return nil, "", err
	}

	if *FlagIoPrimaryIface != "" {
		DebugFunc("try to get ip by iface %v...", *FlagIoPrimaryIface)

		addrs, err := GetHostInfos(true, false, nil)
		if !DebugError(err) {
			for _, addr := range addrs {
				if addr.IFace.Name == *FlagIoPrimaryIface {
					DebugFunc(addr)

					return &addr, hostName, nil
				}
			}
		}
	}

	if IsLinuxOS() {
		DebugFunc("try to get ip by ip routing to 1.1.1.1...")

		cmd := exec.Command("ip", "-o", "route", "get", "to", "1.1.1.1")

		ba, err := WatchdogCmd(cmd, time.Second)
		if !DebugError(err) {
			output := string(ba)

			p := strings.Index(output, "src ")
			if p != -1 {
				output = output[p+4:]
				p := strings.Index(output, " ")
				output = output[:p]

				hostAddress, err := GetHostInfo4IP(net.ParseIP(output))

				if !DebugError(err) {
					DebugFunc(hostAddress)

					return hostAddress, hostName, nil
				}
			}
		}
	}

	addrs, err := net.LookupHost(hostName)
	if err == nil {
		for _, addr := range addrs {
			ip := net.ParseIP(addr)
			if ip != nil {
				hostAddress, err := GetHostInfo4IP(ip)

				if !DebugError(err) {
					DebugFunc(hostAddress)

					return hostAddress, hostName, nil
				}
			}
		}
	}

	intfs, err := net.Interfaces()
	if intfs != nil {
		for _, intf := range intfs {
			if intf.Flags&net.FlagUp == 0 {
				continue
			}

			mac := intf.HardwareAddr.String()

			addrs, _ := intf.Addrs()
			if addrs != nil {
				for _, addr := range addrs {
					ip, ok := addr.(*net.IPNet)

					if !ok {
						continue
					}

					if ip.IP.To4() != nil && IsLocalhost(ip.IP) {
						return &HostInfo{
							IFace: intf,
							Mac:   mac,
							IP:    ip.IP.To4().String(),
							IPNet: ip,
							Addr:  addr,
						}, hostName, nil
					}
				}
			}
		}
	}

	return nil, "", fmt.Errorf("cannot determine primary address")
}

func GetHostInfos(inclLocalhost bool, onlyBroadcastIface bool, remote net.IP) ([]HostInfo, error) {
	DebugFunc("...")

	var list []HostInfo

	intfs, err := net.Interfaces()
	if Error(err) {
		return nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
			continue
		}

		if onlyBroadcastIface && (intf.Flags&net.FlagBroadcast) == 0 {
			continue
		}

		mac := intf.HardwareAddr.String()

		addrs, err := intf.Addrs()
		if Error(err) {
			return nil, err
		}

		for _, addr := range addrs {
			ip, ok := addr.(*net.IPNet)
			if !ok || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() || (!inclLocalhost && IsLocalhost(ip.IP)) {
				continue
			}

			if remote != nil && ip.IP.To4() != nil {
				if len(ip.IP) != len(remote) {
					continue
				}

				localIP := ip.IP.To4()
				remoteIP := remote.To4()

				subnet, err := hex.DecodeString(ip.IP.DefaultMask().String())
				if Error(err) {
					continue
				}

				found := false
				for i := 0; i < len(subnet); i++ {
					found = localIP[i]&subnet[i] == remoteIP[i]&subnet[i]

					if !found {
						break
					}
				}

				if !found {
					continue
				}

				DebugFunc("Local IP for Remote IP %v: %v", remoteIP.String(), localIP.String())
			}

			list = append(list, HostInfo{
				IFace: intf,
				Mac:   mac,
				IP:    ip.IP.To4().String(),
				IPNet: ip,
				Addr:  addr,
			})
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return strings.ToUpper(list[i].Addr.String()) < strings.ToUpper(list[j].Addr.String())
	})

	DebugFunc("%+v", list)

	return list, nil
}

func GetHostInfo4IP(ip net.IP) (*HostInfo, error) {
	DebugFunc()

	intfs, err := net.Interfaces()
	if Error(err) {
		return nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := intf.Addrs()
		if Error(err) {
			return nil, err
		}

		for _, addr := range addrs {
			if strings.Contains(addr.String(), ip.String()) {
				_, ipnet, err := net.ParseCIDR(addr.String())
				if Error(err) {
					return nil, err
				}

				return &HostInfo{
					IFace: intf,
					Mac:   intf.HardwareAddr.String(),
					IP:    ip.To4().String(),
					IPNet: ipnet,
					Addr:  addr,
				}, nil
			}
		}
	}

	return nil, nil
}

func IsPortAvailable(network string, port int) (bool, error) {
	DebugFunc("network: %s, port: %d", network, port)

	switch network {
	case "tcp":
		if network == "tcp" {
			tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if tcpListener != nil {
				Error(tcpListener.Close())
			}
			if err != nil {
				return false, err
			}

		}
	case "udp":
		if network == "udp" {
			udpListener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
			if udpListener != nil {
				Error(udpListener.Close())
			}
			if err != nil {
				return false, err
			}
		}
	default:
		return false, fmt.Errorf("unknown network: %s", network)
	}

	return true, nil
}

func FindFreePort(network string, startPort int, excludedPorts []int) (int, error) {
	DebugFunc()

	for port := startPort; port < 65536; port++ {
		if IndexOf(excludedPorts, port) == -1 {
			b, _ := IsPortAvailable(network, port)

			if !b {
				continue
			}

			DebugFunc("found: %d", port)

			return port, nil
		}
	}

	return -1, fmt.Errorf("cannot find free port")
}

func IsLocalhost(ip net.IP) bool {
	list := []string{LOCALHOST_IP6, LOCALHOST_IP4, "localhost"}

	b := false
	for _, k := range list {
		if ip.String() == k {
			b = true

			break
		}
	}

	if !b {
		_, localhostNet, err := net.ParseCIDR("127.0.0.0/8")
		if err == nil {
			b = localhostNet.Contains(ip)
		}
	}

	DebugFunc("%s: %v", ip, b)

	return b
}

func IsPrivateIP(ip string) (bool, error) {
	var err error

	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return false, fmt.Errorf("Invalid IP: %v", ip)
	}
	_, private24BitBlock, _ := net.ParseCIDR("10.0.0.0/8")
	_, private20BitBlock, _ := net.ParseCIDR("172.16.0.0/12")
	_, private16BitBlock, _ := net.ParseCIDR("192.168.0.0/16")

	private := private24BitBlock.Contains(parsedIp) || private20BitBlock.Contains(parsedIp) || private16BitBlock.Contains(parsedIp)

	DebugFunc("%s: %v", ip, private)

	return private, err
}

func WaitUntilNetworkIsAvailable(lookupIp string) error {
	if lookupIp != "" {
		DebugFunc(lookupIp)
	} else {
		DebugFunc()
	}

	return NewTimeoutOperation(time.Millisecond*500, MillisecondToDuration(*FlagIoNetworkTimeout), func() error {
		addrs, err := GetHostInfos(lookupIp == "", false, nil)

		if DebugError(err) {
			return err
		}

		if len(addrs) == 0 {
			return fmt.Errorf("host networking is down")
		}

		if lookupIp != "" {
			for _, ip := range addrs {
				if ip.IP == lookupIp {
					DebugFunc("host networking with ip %s is available: %+v", lookupIp, addrs)

					return nil
				}
			}

			return fmt.Errorf("host networking with ip %s is not available: %+v", lookupIp, addrs)
		}

		DebugFunc("host networking is available: %+v", addrs)

		return nil
	})
}

func SplitHost(addr string) (string, error) {
	if !strings.Contains(addr, ":") {
		p := strings.Index(addr, "]")

		if p != -1 {
			addr = addr[0:p] + ":]"
		} else {
			addr = addr + ":"
		}
	}

	host, _, err := net.SplitHostPort(addr)

	return host, err
}
