package common

import (
	"bufio"
	"bytes"
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
	if err != nil {
		return 0, err
	}

	return this.Socket.Read(p)
}

func (this TimeoutSocketWriter) Write(p []byte) (n int, err error) {
	if this.Socket == nil {
		return 0, nil
	}

	err = this.Socket.SetWriteDeadline(DeadlineByDuration(this.WriteTimeout))
	if err != nil {
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

func GetHost() (string, string, error) {
	var ip, hostname string

	addrs, err := GetActiveAddrs(true)
	for _, addr := range addrs {
		addrIp, _, err := net.ParseCIDR(addr.String())
		if Error(err) {
			continue
		}

		if !IsLocalhost(addrIp.String()) {
			ip = addrIp.String()
			break
		}
	}

	hostname, err = os.Hostname()
	if Error(err) {
		return "", "", err
	}

	path, err := exec.LookPath("nslookup")

	if path != "" {
		cmd := exec.Command(path, hostname)

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := WatchdogCmd(cmd, time.Second*3)
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
						p := strings.LastIndex(line, " ")
						if p != -1 {
							hostname = strings.TrimSpace(line[p+1:])
						}
					}
				} else {
					if strings.HasPrefix(line, "Address:") {
						p := strings.LastIndex(line, " ")
						if p != -1 {
							ip = strings.TrimSpace(line[p+1:])
						}
					}
				}
			}
		}
	}

	if ip == "" {
		path, err = exec.LookPath("host")

		if path != "" {
			cmd := exec.Command(path, hostname)

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err = WatchdogCmd(cmd, time.Second*3)
			if Error(err) {
				return "", "", nil
			}

			output := string(stdout.Bytes())

			ss := strings.Split(output, " ")

			if len(ss) > 0 {
				ip = strings.TrimSpace(ss[len(ss)-1])
			}
		}
	}

	DebugFunc("IP: %s, FQDN: %s", ip, hostname)

	if ip == "" {
		return "", "", fmt.Errorf("cannot find main ip for %s", hostname)
	}

	return ip, hostname, nil
}

func GetActiveAddrs(inclLocalhost bool) ([]net.Addr, error) {
	var list []net.Addr

	intfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := intf.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ip, ok := addr.(*net.IPNet)
			if !ok || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() || (!inclLocalhost && IsLocalhost(ip.String())) {
				continue
			}

			list = append(list, addr)
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		return strings.ToUpper(list[i].String()) < strings.ToUpper(list[j].String())
	})

	return list, nil
}

func IsPortAvailable(network string, port int) (bool, error) {
	DebugFunc("network: %s, port: %d", network, port)

	switch network {
	case "tcp":
		if network == "tcp" {
			tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				return false, err
			}
			Error(tcpListener.Close())
		}
	case "udp":
		if network == "udp" {
			udpListener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
			if err != nil {
				return false, err
			}
			Error(udpListener.Close())
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

func IsLocalhost(ip string) bool {
	list := []string{LOCALHOST_IP6, LOCALHOST_IP4, "localhost"}

	for _, k := range list {
		if ip == k {
			return true
		}
	}

	return false

}
