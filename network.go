package common

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	LOCALHOST_IP4 = "127.0.0.1"
	LOCALHOST_IP6 = "::1"
)

const (
	FlagNameIoNetworkIface     = "io.network.iface"
	FlagNameIoNetworkTimeout   = "io.network.timeout"
	FlagNameIoConnectTimeout   = "io.connect.timeout"
	FlagNameIoReadwriteTimeout = "io.readwrite.timeout"
)

var (
	FlagIoPrimaryIface     = SystemFlagString(FlagNameIoNetworkIface, "", "primary ethernet interface")
	FlagIoNetworkTimeout   = SystemFlagInt(FlagNameIoNetworkTimeout, 10*1000, "network ready timeout")
	FlagIoConnectTimeout   = SystemFlagInt(FlagNameIoConnectTimeout, 3*1000, "network server and client dial timeout")
	FlagIoReadwriteTimeout = SystemFlagInt(FlagNameIoReadwriteTimeout, 30*60*1000, "network read/write timeout")
)

type HostInfo struct {
	Intf  net.Interface
	IPNet *net.IPNet
}

func GetHostInfos() (string, net.IP, []HostInfo, error) {
	var hostInfos []HostInfo
	var hostAddress net.IP

	hostName, err := os.Hostname()
	if Error(err) {
		return "", nil, nil, err
	}
	intfs, err := net.Interfaces()
	if Error(err) {
		return "", nil, nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := intf.Addrs()
		if Error(err) {
			return "", nil, nil, err
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || !IsIPv4(ipNet.IP) || ipNet.IP.IsLinkLocalUnicast() || ipNet.IP.IsLinkLocalMulticast() {
				continue
			}

			hostInfos = append(hostInfos, HostInfo{
				Intf:  intf,
				IPNet: ipNet,
			})

			if hostAddress == nil || *FlagIoPrimaryIface == intf.Name {
				hostAddress = ipNet.IP
			}

			break
		}
	}

	if hostAddress == nil {
		r := net.Resolver{}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		ipStrings, err := r.LookupHost(ctx, hostName)
		if !DebugError(err) {
			for _, ipString := range ipStrings {
				ip := net.ParseIP(ipString)
				if ip != nil && !ip.IsLoopback() {
					hostAddress = ip

					break
				}
			}
		}
	}

	sort.SliceStable(hostInfos, func(i, j int) bool {
		return hostInfos[i].Intf.Index < hostInfos[j].Intf.Index
	})

	return hostName, hostAddress, hostInfos, nil
}

func IsPortAvailable(network string, port int) error {
	err := func() error {
		switch network {
		case "tcp":
			if network == "tcp" {
				tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
				if err != nil {
					return err
				}

				err = tcpListener.Close()
				if err != nil {
					return err
				}

				return nil
			}
		case "udp":
			if network == "udp" {
				udpListener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
				if err != nil {
					return err
				}

				err = udpListener.Close()
				if err != nil {
					return err
				}

				return nil
			}
		}
		return fmt.Errorf("invalid network type: %s", network)
	}()

	DebugFunc("%d/%s : %v", port, network, err == nil)

	return err
}

func FindFreePort(network string, startPort int, excludedPorts []int) (int, error) {
	DebugFunc()

	for port := startPort; port < 65536; port++ {
		if IndexOf(excludedPorts, port) == -1 {
			if IsPortAvailable(network, port) != nil {
				continue
			}

			DebugFunc("found: %d", port)

			return port, nil
		}
	}

	return -1, fmt.Errorf("cannot find free port")
}

func IsIPv4(ip net.IP) bool {
	if ip == nil {
		return false
	}

	ip = ip.To4()

	return ip != nil && len(ip) == net.IPv4len
}

func IsLocalhost(ip net.IP) bool {
	b := ip.IsLoopback()

	DebugFunc("%s: %v", ip, b)

	return b
}

func IsPrivateIP(ip net.IP) bool {
	_, private24BitBlock, _ := net.ParseCIDR("10.0.0.0/8")
	_, private20BitBlock, _ := net.ParseCIDR("172.16.0.0/12")
	_, private16BitBlock, _ := net.ParseCIDR("192.168.0.0/16")

	private := private24BitBlock.Contains(ip) || private20BitBlock.Contains(ip) || private16BitBlock.Contains(ip)

	DebugFunc("%s: %v", ip, private)

	return private
}

func WaitUntilNetworkIsAvailable(lookupIp net.IP) error {
	if lookupIp != nil {
		DebugFunc(lookupIp)
	} else {
		DebugFunc()
	}

	return NewWatchdogRetry(time.Millisecond*500, MillisecondToDuration(*FlagIoNetworkTimeout), func() error {
		_, _, hostInfos, err := GetHostInfos()

		if DebugError(err) {
			return err
		}

		if lookupIp != nil {
			for _, ip := range hostInfos {
				if reflect.DeepEqual(ip.IPNet.IP, lookupIp) {
					DebugFunc("host networking with ip %s is available: %+v", lookupIp, hostInfos)

					return nil
				}
			}

			return fmt.Errorf("host networking with ip %s is not available: %+v", lookupIp, hostInfos)
		}

		DebugFunc("host networking is available: %+v", hostInfos)

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

func FormatIP(ip net.IP) string {
	if ip.To4() != nil {
		return ip.To4().String()
	} else {
		return fmt.Sprintf("[%s]", ip.To16().String())
	}
}

func IsLinkUp(nic string) (bool, error) {
	// https://linuxconfig.org/how-to-detect-whether-a-physical-cable-is-connected-to-network-card-slot-on-linux

	if IsWindows() {
		return true, fmt.Errorf("not supported on Windows")
	}

	ba, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", nic))
	if Error(err) {
		return false, err
	}

	s := strings.TrimSpace(string(ba))

	return s == "up", nil
}

func IsLinkConnected(nic string) (bool, error) {
	// https://linuxconfig.org/how-to-detect-whether-a-physical-cable-is-connected-to-network-card-slot-on-linux

	if IsWindows() {
		return true, fmt.Errorf("not supported on Windows")
	}

	ba, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/carrier", nic))
	if Error(err) {
		return false, err
	}

	s := strings.TrimSpace(string(ba))
	v, err := strconv.Atoi(s)
	if Error(err) {
		return false, err
	}

	return v == 1, nil
}
