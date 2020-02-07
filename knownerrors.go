package common

func IsErrNetClosing(err error) bool {
	return err.Error() == "use of closed network connection"
}
