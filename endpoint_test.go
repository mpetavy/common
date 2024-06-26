package common

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func test(t *testing.T, address string, tlsConfig *tls.Config, txt string, isClient bool) error {
	ep, connector, err := NewEndpoint(address, isClient, tlsConfig)
	if Error(err) {
		return err
	}

	err = ep.Start()
	if Error(err) {
		return err
	}

	defer func() {
		Error(ep.Stop())
	}()

	conn, err := connector()
	if Error(err) {
		return err
	}

	defer func() {
		Error(conn.Close())
	}()

	if !isClient {
		ba := make([]byte, 1000)

		conn := NewTimeoutReader(conn, true, time.Second)

		buf := bytes.Buffer{}

		for {
			nr, err := conn.Read(ba)
			if IsErrTimeout(err) {
				break
			}
			if Error(err) {
				return err
			}

			nw, err := buf.Write(ba[:nr])
			if Error(err) {
				return err
			}

			assert.Equal(t, nr, nw)
		}

		assert.Equal(t, txt, buf.String())
	} else {
		n, err := conn.Write([]byte(txt))
		if Error(err) {
			return err
		}

		assert.Equal(t, len(txt), n)

		Sleep(time.Second * 2)
	}

	return nil
}

func TestEndpoint(t *testing.T) {
	SetTesting(t)

	txt, err := RndString(100)
	if Error(err) {
		return
	}

	port, err := FindFreePort("tcp", 1024, nil)
	if Error(err) {
		return
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := test(t, fmt.Sprintf(":%d", port), nil, txt, false)
		if IsErrNetClosed(err) || Error(err) {
			return
		}
	}()

	Sleep(time.Second)

	err = test(t, fmt.Sprintf(":%d", port), nil, txt, true)
	if Error(err) {
		return
	}

	wg.Wait()
}
