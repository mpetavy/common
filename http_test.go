package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

const (
	port = 9999
)

func startServer() error {
	mux := &http.ServeMux{}

	mux.HandleFunc("/noanswer", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Hour * 24)
	})

	err := HTTPServerStart(port, nil, mux)
	if Error(err) {
		return err
	}

	return err
}

func serverUrl(query string) string {
	return fmt.Sprintf("http://localhost:%d%s", port, query)
}

func stopServer() {
	Error(HTTPServerStop())
}

func TestHttpRequest(t *testing.T) {
	backupFlagHTTPTimeout := *FlagHTTPTimeout
	*FlagHTTPTimeout = 1000

	require.NoError(t, startServer())
	defer func() {
		stopServer()

		*FlagHTTPTimeout = backupFlagHTTPTimeout
	}()

	_, _, err := HTTPRequest(nil, MillisecondToDuration(*FlagHTTPTimeout), http.MethodGet, serverUrl("/noanswer"), nil, nil, "", "", nil, http.StatusOK)

	require.True(t, IsErrTimeout(err))
}
