package common

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

func FindMainIP() (string, error) {
	host, err := os.Hostname()
	if CheckError(err) {
		return "", err
	}

	if IsWindowsOS() {
		cmd := exec.Command("nslookup", host)

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := Watchdog(cmd, time.Second*3)
		if CheckError(err) {
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

			line = line[10:]

			return line, nil
		}
	} else {
		cmd := exec.Command("host", host)

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := Watchdog(cmd, time.Second*3)
		if CheckError(err) {
			return "", nil
		}
		output := string(stdout.Bytes())

		ss := strings.Split(output, " ")

		if len(ss) > 0 {
			return ss[len(ss)-1], nil
		}
	}

	return "", fmt.Errorf("cannot find main ip for %s", host)
}

func FindActiveIPs() ([]string, error) {
	var addresses []string
	// list system network interfaces
	// https://golang.org/pkg/net/#Interfaces
	intfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	// mapping between network interface name and index
	// https://golang.org/pkg/net/#Interface
	for _, intf := range intfs {
		// skip down interface & check next intf
		if intf.Flags&net.FlagUp == 0 {
			continue
		}
		// skip loopback & check next intf
		if intf.Flags&net.FlagLoopback != 0 {
			continue
		}
		// list of unicast interface addresses for specific interface
		// https://golang.org/pkg/net/#Interface.Addrs
		addrs, err := intf.Addrs()
		if err != nil {
			return nil, err
		}
		// network end point address
		// https://golang.org/pkg/net/#Addr
		for _, addr := range addrs {
			// if for windows may need to type switch

			// type assertion to access Addr interface
			// underlying IPNet IP method
			if addr == nil || addr.(*net.IPNet).IP.IsLoopback() {
				continue
			}
			// append active interfaces
			addresses = append(addresses, addr.String())
		}
	}
	return addresses, nil
}
