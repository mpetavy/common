package common

import (
	"fmt"
	"net"
	"strings"
)

var (
	suppressErrors = []string{
		"http2",
		"wsasend",
		"tls: unknown certificate",
		"tls handshake error",
		"use of closed network connection",
		"eof",
	}
)

func IsErrExit(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*ErrExit)

	return ok
}

func IsErrNetClosing(err error) bool {
	if err == nil {
		return false
	}

	return strings.Index(strings.ToLower(err.Error()), "use of closed network connection") != -1
}

func IsErrUnexpectedEOF(err error) bool {
	if err == nil {
		return false
	}

	return strings.Index(strings.ToLower(err.Error()), "unexpected eof") != -1
}

func IsSuppressedError(err error) bool {
	if err == nil {
		return false
	}

	return IsSuppressedErrorMessage(err.Error())
}

func IsSuppressedErrorMessage(err string) bool {
	msg := strings.ToLower(err)
	for _, se := range suppressErrors {
		if strings.Contains(msg, se) {
			return true
		}
	}

	return false
}

func HandleCopyBufferError(written int64, err error) (int64, error) {
	if err == nil {
		return written, nil
	}

	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return written, fmt.Errorf("Timeout error: %s", neterr.Error())
	}

	if operr, ok := err.(*net.OpError); ok {
		return written, fmt.Errorf("Operation error: %s", operr.Error())
	}

	if IsErrNetClosing(err) || IsErrUnexpectedEOF(err) {
		return written, nil
	}

	return written, err
}
