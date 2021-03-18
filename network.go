package common

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
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

type TimeoutSocketReader struct {
	ReadTimeout time.Duration
	Socket      net.Conn
}

type TimeoutSocketWriter struct {
	WriteTimeout time.Duration
	Socket       net.Conn
}

func (this TimeoutSocketReader) Read(p []byte) (n int, err error) {
	if this.Socket == nil {
		return 0, io.EOF
	}

	err = this.Socket.SetReadDeadline(DeadlineByDuration(this.ReadTimeout))
	if Error(err) {
		return 0, err
	}

	return this.Socket.Read(p)
}

func (this TimeoutSocketWriter) Write(p []byte) (n int, err error) {
	if this.Socket == nil {
		return 0, nil
	}

	err = this.Socket.SetWriteDeadline(DeadlineByDuration(this.WriteTimeout))
	if Error(err) {
		return 0, err
	}

	return this.Socket.Write(p)
}

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

func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if Error(err) {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

func assignIP(currentIp, newIp net.IP) net.IP {
	if (currentIp == nil) || (IsLocalhost(currentIp) && !IsLocalhost(newIp)) {
		return newIp
	}

	return currentIp
}

func GetHost() (net.IP, string, error) {
	var ip net.IP
	var hostname string

	addrs, err := GetHostAddrs(true, nil)
	for _, addr := range addrs {
		newIp, _, err := net.ParseCIDR(addr.Addr.String())
		if Error(err) {
			continue
		}

		ip = assignIP(ip, newIp)
	}

	hostname, err = os.Hostname()
	if Error(err) {
		return nil, "", err
	}

	newIp, err := GetOutboundIP()
	if err == nil {
		ip = assignIP(ip, newIp)
	}

	if ip == nil || IsLocalhost(ip) {
		path, err := exec.LookPath("nslookup")
		if err == nil {
			cmd := exec.Command(path, hostname)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := WatchdogCmd(cmd, MillisecondToDuration(*FlagIoNetworkTimeout))
			if err == nil {
				output := string(stdout.Bytes())

				scanner := bufio.NewScanner(strings.NewReader(output))
				line := ""

				nslookupHostnameFound := false

				for scanner.Scan() {
					line = strings.TrimSpace(scanner.Text())

					if !nslookupHostnameFound {
						nslookupHostnameFound = strings.HasPrefix(line, "Name:")

						if nslookupHostnameFound {
							hostname = strings.TrimSpace(line[5:])
						}
					} else {
						if strings.HasPrefix(line, "Address:") {
							addr := strings.TrimSpace(line[8:])
							newIp = net.ParseIP(addr)

							if ip == nil {
								return nil, "", fmt.Errorf("cannot parse IP: %s", addr)
							}

							ip = assignIP(ip, newIp)

							break
						}
					}
				}

				Debug("nslookup result: hostname: %s ip: %s", hostname, ip)
			}
		}
	}

	if ip == nil || IsLocalhost(ip) {
		path, err := exec.LookPath("host")
		if err == nil {
			cmd := exec.Command(path, hostname)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err = WatchdogCmd(cmd, MillisecondToDuration(*FlagIoNetworkTimeout))
			if Error(err) {
				return nil, "", err
			}

			output := string(stdout.Bytes())

			ss := strings.Split(output, " ")

			if len(ss) > 0 {
				addr := strings.TrimSpace(ss[len(ss)-1])
				newIp = net.ParseIP(addr)

				if ip == nil {
					return nil, "", fmt.Errorf("cannot parse IP: %s", addr)
				}

				ip = assignIP(ip, newIp)
			}

			Debug("host result: ip: %s", ip)
		}
	}

	if ip == nil || IsLocalhost(ip) {
		addrs, err := net.LookupHost(hostname)
		if err == nil {
			newIp := net.ParseIP(addrs[0])
			if newIp == nil {
				return nil, "", fmt.Errorf("cannot parse IP: %s", addrs[0])
			}

			ip = assignIP(ip, newIp)
		}
	}

	if ip == nil {
		return nil, "", fmt.Errorf("cannot find main ip for %s", hostname)
	}

	DebugFunc("IP: %s, FQDN: %s", ip, hostname)

	return ip, hostname, nil
}

type hostAddress struct {
	Mac  string
	Addr net.Addr
}

func GetHostAddrs(inclLocalhost bool, remote net.IP) ([]hostAddress, error) {
	var list []hostAddress

	intfs, err := net.Interfaces()
	if Error(err) {
		return nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
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

			list = append(list, hostAddress{
				Mac:  mac,
				Addr: addr,
			})
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return strings.ToUpper(list[i].Addr.String()) < strings.ToUpper(list[j].Addr.String())
	})

	return list, nil
}

func GetHostInterface(ip net.IP) (*net.Interface, net.Addr, error) {
	intfs, err := net.Interfaces()
	if Error(err) {
		return nil, nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := intf.Addrs()
		if Error(err) {
			return nil, nil, err
		}

		for _, addr := range addrs {
			if strings.Contains(addr.String(), ip.String()) {
				return &intf, addr, nil
			}
		}
	}

	return nil, nil, nil
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
