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
	if err == nil || *FlagLogVerbose {
		return false
	}

	msg := strings.ToLower(err.Error())
	for _, se := range suppressErrors {
		if strings.Index(msg, se) != -1 {
			return true
		}
	}

	return false
}

func IsSuppressedErrorMessage(err string) bool {
	return IndexOf(suppressErrors, err) != -1
}

func CopyBufferError(written int64, err error) (int64, error) {
	if err == nil {
		return written, nil
	}

	DebugError(err)

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
