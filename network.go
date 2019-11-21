package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type TimeoutSocket struct {
	io.ReadWriter
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Socket       *net.Conn
}

func (this *TimeoutSocket) Read(p []byte) (n int, err error) {
	err = (*this.Socket).SetReadDeadline(DeadlineByDuration(this.ReadTimeout))
	if err != nil {
		return 0, err
	}

	return (*this.Socket).Read(p)
}

func (this *TimeoutSocket) Write(p []byte) (n int, err error) {
	err = (*this.Socket).SetWriteDeadline(DeadlineByDuration(this.WriteTimeout))
	if err != nil {
		return 0, err
	}

	return (*this.Socket).Write(p)
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

func GetMainIP() (string, error) {
	ips, err := GetActiveIPs()
	if len(ips) == 1 {
		DebugFunc(ips[0])

		return ips[0], nil
	}

	hostname, err := os.Hostname()
	if Error(err) {
		return "", err
	}

	path, err := exec.LookPath("nslookup")

	if path != "" {
		cmd := exec.Command(path, hostname)

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := Watchdog(cmd, time.Second*3)
		if Error(err) {
			return "", nil
		}
		output := string(stdout.Bytes())

		scanner := bufio.NewScanner(strings.NewReader(output))
		line := ""
		found := false

		for scanner.Scan() {
			line = strings.TrimSpace(scanner.Text())

			found = strings.Index(line, "Name:") != -1

			if found {
				break
			}
		}

		if found && scanner.Scan() {
			line = strings.TrimSpace(scanner.Text())

			line = strings.TrimSpace(line[10:])

			DebugFunc(line)

			return line, nil
		}
	}

	path, err = exec.LookPath("host")

	if path != "" {
		cmd := exec.Command(path, hostname)

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = Watchdog(cmd, time.Second*3)
		if Error(err) {
			return "", nil
		}
		output := string(stdout.Bytes())

		ss := strings.Split(output, " ")

		if len(ss) > 0 {
			ip := strings.TrimSpace(ss[len(ss)-1])

			DebugFunc(ip)

			return ip, nil
		}
	}

	return "", fmt.Errorf("cannot find main ip for %s", hostname)
}

func GetActiveIPs() ([]string, error) {
	var addresses []string

	intfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, intf := range intfs {
		if intf.Flags&net.FlagUp == 0 {
			continue
		}

		if intf.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := intf.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			if addr == nil || addr.(*net.IPNet).IP.IsLoopback() {
				continue
			}

			addresses = append(addresses, addr.String())
		}
	}
	return addresses, nil
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
			WarnError(tcpListener.Close())
		}
	case "udp":
		if network == "udp" {
			udpListener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
			if err != nil {
				return false, err
			}
			WarnError(udpListener.Close())
		}
	default:
		return false, fmt.Errorf("unknown network: %s", network)
	}

	return true, nil
}

func FindFreePort(network string, startPort int, excludedPorts []int) (int, error) {
	DebugFunc()

	for port := startPort; port < 65536; port++ {
		index, err := IndexOf(excludedPorts, port)
		if Error(err) {
			return -1, err
		}

		if index == -1 {
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
