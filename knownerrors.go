package common

import "strings"

func IsErrNetClosing(err error) bool {
	return strings.Index(strings.ToLower(err.Error()), "use of closed network connection") != -1
}
