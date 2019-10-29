package common

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	TLSPemFile *string
	TLSKeyFile *string
)

func init() {
	TLSPemFile = flag.String("tls.pem", AppFilename(".server.pem"), "TLS server PEM file")
	TLSKeyFile = flag.String("tls.key", AppFilename(".server.key"), "TLS server KEY file")
}

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

func FindMainIP() (string, error) {
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
			return strings.TrimSpace(ss[len(ss)-1]), nil
		}
	}

	return "", fmt.Errorf("cannot find main ip for %s", hostname)
}

func FindActiveIPs() ([]string, error) {
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

func CreateTLSConfig() (*tls.Config, error) {
	DebugFunc()

	b, err := FileExists(*TLSPemFile)
	if Error(err) || !b {
		return nil, nil
	}

	b, err = FileExists(*TLSKeyFile)
	if Error(err) || !b {
		return nil, nil
	}

	Debug("generate TLS config from pem %s and key %s", *TLSPemFile, *TLSKeyFile)

	cert, err := tls.LoadX509KeyPair(*TLSPemFile, *TLSKeyFile)
	if Error(err) {
		return nil, err
	}

	var tlsConfig tls.Config

	tlsConfig = tls.Config{Certificates: []tls.Certificate{cert}}
	tlsConfig.Rand = rand.Reader

	return &tlsConfig, nil
}
