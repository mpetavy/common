package common

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	FlagNameHTTPHeaderLimit = "http.headerlimit"
	FlagNameHTTPBodyLimit   = "http.bodylimit"
)

var (
	FlagHTTPHeaderLimit = flag.Int64(FlagNameHTTPHeaderLimit, 1024*1024, "HTTP header limit")
	FlagHTTPBodyLimit   = flag.Int64(FlagNameHTTPBodyLimit, 5*1024*1024*1024, "HTTP body limit")
)

var (
	httpServer      *http.Server
	ctxServer       context.Context
	ctxServerCancel context.CancelFunc

	ErrUnauthorized  = fmt.Errorf("Unauthorized")
	ErrNoBodyContent = fmt.Errorf("no HTTP body provided")
)

const (
	CONTENT_TYPE        = "Content-Type"
	CONTENT_LENGTH      = "Content-Length"
	CONTENT_DISPOSITION = "Content-Disposition"

	HEADER_LOCATION = "Location"
)

type BasicAuthFunc func(username string, password string) error

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.Message)
}

func Header(r *http.Request, name string) (string, error) {
	DebugFunc()

	for k, v := range r.Header {
		if strings.ToLower(k) == strings.ToLower(name) {
			return v[len(v)-1], nil
		}
	}

	v := r.URL.Query().Get(name)
	if v != "" {
		return v, nil
	}

	return "", fmt.Errorf("missing header '%s'", name)
}

func BasicAuthHandler(authFunc BasicAuthFunc, next http.HandlerFunc) http.HandlerFunc {
	DebugFunc()

	return func(w http.ResponseWriter, r *http.Request) {
		status, err := func() (int, error) {
			username, password, ok := r.BasicAuth()
			if !ok {
				return http.StatusUnauthorized, ErrUnauthorized
			}

			err := authFunc(username, password)
			if Error(err) {
				return http.StatusUnauthorized, err
			}

			return http.StatusOK, nil
		}()

		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

			http.Error(w, err.Error(), status)

			return
		}

		next.ServeHTTP(w, r)
	}
}

func StartHTTPServer(port int, tlsConfig *tls.Config, mux *http.ServeMux) error {
	DebugFunc()

	tlsInfo := ""
	if tlsConfig != nil {
		tlsInfo = " [TLS]"
	}

	httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadTimeout:       MillisecondToDuration(*FlagIoReadwriteTimeout),
		ReadHeaderTimeout: MillisecondToDuration(*FlagIoReadwriteTimeout),
		WriteTimeout:      MillisecondToDuration(*FlagIoReadwriteTimeout),
		ErrorLog:          LogDebug,
		ConnState: func(conn net.Conn, cs http.ConnState) {
			if cs == http.StateNew {
				Error(conn.SetReadDeadline(time.Now().Add(MillisecondToDuration(*FlagIoConnectTimeout))))
			}
		},
	}
	httpServer.SetKeepAlivesEnabled(false)
	httpServer.MaxHeaderBytes = int(*FlagHTTPHeaderLimit)

	Info(fmt.Sprintf("HTTP server%s started on port: %d", tlsInfo, port))

	ctxServer, ctxServerCancel = context.WithCancel(context.Background())

	var err error

	if tlsConfig != nil {
		err = httpServer.ListenAndServeTLS("", "")
	} else {
		err = httpServer.ListenAndServe()
	}

	if err != nil && err == http.ErrServerClosed {
		<-ctxServer.Done()

		err = nil
	}

	if Error(err) {
		return err
	}

	return nil
}

func StopHTTPServer() error {
	DebugFunc()

	if httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := httpServer.Shutdown(ctx)
	if Error(err) {
		return err
	}

	tlsInfo := ""
	if httpServer.TLSConfig != nil {
		tlsInfo = " [TLS]"
	}

	Info(fmt.Sprintf("HTTP server%s closed", tlsInfo))

	httpServer = nil

	ctxServerCancel()

	return nil
}

func InitHashAlgorithm(s string) (crypto.Hash, error) {
	DebugFunc()

	p := strings.Index(s, ":")
	if p != -1 {
		s = s[:p]
	}

	var algorithm crypto.Hash

	switch strings.ToUpper(s) {
	case crypto.MD5.String():
		algorithm = crypto.MD5
	case crypto.SHA224.String():
		algorithm = crypto.SHA224
	case crypto.SHA256.String():
		algorithm = crypto.SHA256
	case crypto.SHA512.String():
		algorithm = crypto.SHA512
	default:
		return 0, fmt.Errorf("unknown hash algorithm: %s", s)
	}

	return algorithm, nil
}

func HashBytes(crypto crypto.Hash, r io.Reader) ([]byte, error) {
	DebugFunc()

	hasher := crypto.New()

	_, err := io.Copy(hasher, r)
	if Error(err) {
		return nil, err
	}

	return hasher.Sum(nil), nil
}

func HashString(crypto crypto.Hash, s string) (string, error) {
	p := strings.Index(s, ":")
	if p != -1 && crypto.String() == s[:p] {
		return s, nil
	}

	hash, err := HashBytes(crypto, strings.NewReader(s))
	if Error(err) {
		return "", err
	}

	return fmt.Sprintf("%s:%s", strings.ToUpper(crypto.String()), hex.EncodeToString(hash)), nil
}

func CompareHashes(expected string, actual string) error {
	err := func() error {
		hashAlgorithm, err := InitHashAlgorithm(expected)
		if Error(err) {
			return err
		}

		actualHashed, err := HashString(hashAlgorithm, actual)
		if Error(err) {
			return err
		}

		if expected != actualHashed {
			return fmt.Errorf("expected and actual hashes don't match")
		}

		return nil
	}()

	DebugFunc(strconv.FormatBool(err == nil))

	return err
}

func HTTPWriteJson(status int, w http.ResponseWriter, ba []byte) error {
	w.Header().Set(CONTENT_TYPE, MimetypeApplicationJson.MimeType)
	w.Header().Set(CONTENT_LENGTH, strconv.Itoa(len(ba)))
	w.WriteHeader(status)

	_, err := w.Write(ba)
	if Error(err) {
		return err
	}

	return nil
}

func HTTPRequest(timeout time.Duration, method string, address string, headers map[string]string, username string, password string, body io.Reader, expectedCode int) (*http.Response, []byte, error) {
	DebugFunc("Method: %s URL: %s Username: %s Password: %s", method, address, username, strings.Repeat("X", len(password)))

	client := &http.Client{}

	if strings.Contains(address, "https") {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	req, err := http.NewRequest(method, address, body)
	if Error(err) {
		return nil, nil, err
	}

	if username != "" || password != "" {
		if username == "" {
			username = "dummy"
		}

		req.SetBasicAuth(username, password)
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	var resp *http.Response

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer func() {
			cancel()
		}()

		resp, err = client.Do(req.WithContext(ctx))
		if Error(err) {
			return nil, nil, err
		}
	} else {
		resp, err = client.Do(req)
		if Error(err) {
			return nil, nil, err
		}
	}

	if expectedCode > 0 && resp.StatusCode != expectedCode {
		return nil, nil, fmt.Errorf("unexpected HTTP staus code, expected %d got %d", expectedCode, resp.StatusCode)
	}

	// Caution: if read the body then respect then take this with  your TIMEOUT parameter into account

	ba, err := ReadBody(resp.Body)
	if Error(err) {
		return nil, nil, err
	}

	return resp, ba, nil
}

func ReadBody(r io.ReadCloser) ([]byte, error) {
	defer func() {
		Error(r.Close())
	}()

	buf := bytes.Buffer{}

	_, err := io.Copy(&buf, io.LimitReader(r, *FlagHTTPBodyLimit))
	if Error(err) {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ReadBodyJSON[T any](r io.ReadCloser) ([]T, bool, error) {
	var records []T

	ba, err := ReadBody(r)
	if Error(err) {
		return nil, false, err
	}

	if len(ba) == 0 {
		return nil, false, ErrNoBodyContent
	}

	isArray := strings.HasPrefix(string(ba), "[")

	decoder := json.NewDecoder(bytes.NewReader(ba))

	if isArray {
		err = decoder.Decode(&records)
		if Error(err) {
			return nil, false, err
		}
	} else {
		records = make([]T, 1)

		err = decoder.Decode(&records[0])
		if Error(err) {
			return nil, false, err
		}
	}

	return records, isArray, nil
}

type RestURL struct {
	Method   string
	Resource string
}

func NewRestURL(method string, resource string) *RestURL {
	return &RestURL{method, resource}
}

func (u *RestURL) String() string {
	return fmt.Sprintf("%s %s", u.Method, u.Resource)
}

func (u *RestURL) Format(args ...any) string {
	s := u.Resource

	if strings.Contains(s, "{") {
		regex := regexp.MustCompile("{.*?\\}")

		s = regex.ReplaceAllString(s, "%v")
		s = fmt.Sprintf(s, args...)
	}

	return s
}

func (u *RestURL) URL(address string, args ...any) string {
	return fmt.Sprintf("%s%s", address, u.Format(args...))
}
