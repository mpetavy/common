package common

import (
	"strings"
)

var (
	suppressError = []string{
		"http2",
		"wsasend",
		"remote error: tls: unknown certificate",
	}
)

func IsErrNetClosing(err error) bool {
	return strings.Index(strings.ToLower(err.Error()), "use of closed network connection") != -1
}

func IsErrUnexpectedEOF(err error) bool {
	return strings.Index(strings.ToLower(err.Error()), "unexpected eof") != -1
}

func IsSuppressedError(err error) bool {
	return IndexOf(suppressError, err.Error()) != -1
}

func IsSuppressedErrorMessage(err string) bool {
	return IndexOf(suppressError, err) != -1
}
