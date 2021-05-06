package common

import (
	"encoding/json"
	"fmt"
)

type echoLogger struct{}

func (this echoLogger) Write(p []byte) (int, error) {
	msg := string(p)
	isError := false
	m := make(map[string]interface{})

	err := json.Unmarshal(p, &m)
	if err == nil {
		v, ok := m["error"]

		isError = ok && fmt.Sprintf("%v", v) != ""
	}

	if isError {
		DebugError(fmt.Errorf("Echo: %s", msg))
	} else {
		Debug("Echo: %s", msg)
	}

	return len(p), nil
}

func NewEchoLogger() echoLogger {
	return echoLogger{}
}
