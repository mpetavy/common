package common

import (
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
		"http: Server closed",
	}
)

func IsErrExit(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*ErrExit)

	return ok
}

func IsErrTimeout(err error) bool {
	type withTimeout interface {
		Timeout() bool
	}

	if err == nil {
		return false
	}

	errTimeout, ok := err.(withTimeout)

	return ok && errTimeout.Timeout()
}

func IsErrNetOperation(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*net.OpError)

	return ok
}

func IsErrNetClosed(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), "use of closed network connection") ||
		strings.Contains(strings.ToLower(err.Error()), " connection was forcibly closed")
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
