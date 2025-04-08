package common

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	AUTHORIZATION = "Authorization"
	BEARER        = "Bearer"

	ACCEPT              = "Accept"
	CONTENT_TYPE        = "Content-Type"
	CONTENT_LENGTH      = "Content-Length"
	CONTENT_MD5         = "Content-MD5"
	CONTENT_DISPOSITION = "Content-Disposition"
	CONTENT_ENCODING    = "Content-Encoding"

	ACCEPT_ENCODING = "Accept-Encoding"

	HEADER_LOCATION = "Location"

	FlagNameHTTPHeaderLimit = "http.headerlimit"
	FlagNameHTTPBodyLimit   = "http.bodylimit"
	FlagNameHTTPTLSInsecure = "http.tlsinsecure"
	FlagNameHTTPTimeout     = "http.timeout"
	FlagNameHTTPGzip        = "http.gzip"
)

var (
	FlagHTTPHeaderLimit = SystemFlagInt64(FlagNameHTTPHeaderLimit, 1024*1024, "HTTP header limit")
	FlagHTTPBodyLimit   = SystemFlagInt64(FlagNameHTTPBodyLimit, 5*1024*1024*1024, "HTTP body limit")
	FlagHTTPTLSInsecure = SystemFlagBool(FlagNameHTTPTLSInsecure, true, "HTTP default TLS insecure")
	FlagHTTPTimeout     = SystemFlagInt(FlagNameHTTPTimeout, 120000, "HTTP default request timeout")
	FlagHTTPGzip        = SystemFlagBool(FlagNameHTTPGzip, true, "HTTP GZip support")

	httpServer *http.Server

	ErrUnauthorized  = fmt.Errorf("Unauthorized")
	ErrNoBodyContent = fmt.Errorf("no HTTP body provided")
)

type BasicAuthFunc func(username string, password string) error

type HTTPError struct {
	StatusCode int   // HTTP status code
	Err        error // Original error
}

func (he *HTTPError) Error() string {
	return fmt.Sprintf("status %d: %v", he.StatusCode, he.Err)
}

func (he *HTTPError) Unwrap() error {
	return he.Err
}

func NewHTTPError(statusCode int, err error) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Err:        err,
	}
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

type StatusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *StatusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func NewStatusResponseWriter(w http.ResponseWriter) *StatusResponseWriter {
	// Set statusCode to 200 by default in case WriteHeader is not explicitly called.
	return &StatusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func TelemetryHandler(next http.HandlerFunc) http.HandlerFunc {
	DebugFunc()

	return func(w http.ResponseWriter, r *http.Request) {
		eventTelemetry := EventTelemetry{
			IsTelemetryRequest: true,
			Ctx:                r.Context(),
			Title:              fmt.Sprintf("%s %s", r.Method, r.URL.String()),
			Start:              time.Now(),
		}
		defer func() {
			eventTelemetry.End = time.Now()

			Events.Emit(eventTelemetry, false)
		}()

		sw := NewStatusResponseWriter(w)

		next.ServeHTTP(sw, r)

		eventTelemetry.Code = sw.statusCode
		if eventTelemetry.Code == 0 {
			eventTelemetry.Code = http.StatusOK
		}
	}
}

func BasicAuthHandler(mandatory bool, authFunc BasicAuthFunc, next http.HandlerFunc) http.HandlerFunc {
	DebugFunc()

	return func(w http.ResponseWriter, r *http.Request) {
		status, err := func() (int, error) {
			username, password, ok := r.BasicAuth()
			if !ok {
				if mandatory {
					return http.StatusUnauthorized, ErrUnauthorized
				} else {
					return http.StatusOK, nil
				}
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

func HTTPServerStart(port int, tlsConfig *tls.Config, mux *http.ServeMux) error {
	DebugFunc()

	err := IsPortAvailable("tcp", port)
	if Error(err) {
		return err
	}

	protocolInfo := "HTTP"
	if tlsConfig != nil {
		protocolInfo = "HTTPS"
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

	StartInfo(fmt.Sprintf("%s server %s", protocolInfo, httpServer.Addr))

	ln, err := net.Listen("tcp", httpServer.Addr)
	if Error(err) {
		return err
	}

	go func() {
		defer func() {
			WarnError(ln.Close())
		}()

		if tlsConfig != nil {
			WarnError(httpServer.ServeTLS(ln, "", ""))
		} else {
			WarnError(httpServer.Serve(ln))
		}
	}()

	return nil
}

func HTTPServerStop() error {
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

	protocolInfo := "HTTP"
	if httpServer.TLSConfig != nil {
		protocolInfo = "HTTPS"
	}

	StopInfo(fmt.Sprintf("%s server %s", protocolInfo, httpServer.Addr))

	httpServer = nil

	return nil
}

func HashAlgorithm(s string) (crypto.Hash, error) {
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

func IsHashedValue(s string) bool {
	_, err := HashAlgorithm(s)

	return err == nil
}

func HashValue(algorithm crypto.Hash, s string) (string, error) {
	current, err := HashAlgorithm(s)
	if err == nil {
		if current != algorithm {
			return "", fmt.Errorf("Different hash algorithm used %s. Expected %s, current %s", s, algorithm.String(), current.String())
		}

		return s, nil
	}

	p := strings.Index(s, ":")
	if p != -1 && algorithm.String() == s[:p] {
		return s, nil
	}

	hash, err := HashBytes(algorithm, strings.NewReader(s))
	if Error(err) {
		return "", err
	}

	return fmt.Sprintf("%s:%s", strings.ToUpper(algorithm.String()), hex.EncodeToString(hash)), nil
}

func CompareHashes(expected string, actual string) error {
	err := func() error {
		hashAlgorithm, err := HashAlgorithm(expected)
		if Error(err) {
			return err
		}

		actualHashed, err := HashValue(hashAlgorithm, actual)
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

type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (g *GzipResponseWriter) Write(data []byte) (int, error) {
	return g.Writer.Write(data)
}

func HTTPResponse(w http.ResponseWriter, r *http.Request, status int, mimeType string, bodyLen int, body io.Reader) error {
	w.Header().Set(CONTENT_TYPE, mimeType)

	useGZip := *FlagHTTPGzip && strings.Contains(r.Header.Get(ACCEPT_ENCODING), "gzip")

	if useGZip {
		w.Header().Set(CONTENT_ENCODING, "gzip")
	} else {
		if bodyLen >= 0 {
			w.Header().Set(CONTENT_LENGTH, strconv.Itoa(bodyLen))
		}
	}

	w.WriteHeader(status)

	if useGZip {
		gzipWriter := gzip.NewWriter(w)

		defer func() {
			Error(gzipWriter.Flush())
			Error(gzipWriter.Close())
		}()

		w = &GzipResponseWriter{Writer: gzipWriter, ResponseWriter: w}
	}

	if body != nil {
		_, err := io.Copy(w, body)
		if Error(err) {
			return err
		}
	}

	return nil
}

func HTTPRequest(httpTransport *http.Transport, timeout time.Duration, method string, address string, headers http.Header, formdata url.Values, username string, password string, body io.Reader, expectedCode int) (*http.Response, []byte, error) {
	DebugFunc()

	start := time.Now()

	eventTelemetry := EventTelemetry{
		IsTelemetryRequest: false,
		Title:              fmt.Sprintf("%s %s", method, address),
		Start:              start,
	}
	defer func() {
		eventTelemetry.End = time.Now()

		Events.Emit(eventTelemetry, false)
	}()

	client := &http.Client{}

	if headers == nil {
		headers = make(http.Header)
	}

	if httpTransport != nil {
		client.Transport = httpTransport
	} else {
		if *FlagHTTPTLSInsecure {
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		for key, val := range via[0].Header {
			req.Header[key] = val
		}
		return nil
	}

	if formdata != nil && body == nil {
		body = strings.NewReader(formdata.Encode())
	}

	req, err := http.NewRequest(method, address, body)
	if Error(err) {
		return nil, nil, err
	}

	// GO won't transmit Content-Length with PATCH HTTP method, even when defined in the headers...
	if method == http.MethodPatch {
		headerContentLength := headers.Get(CONTENT_LENGTH)
		if headerContentLength != "" {
			contentLength, err := strconv.Atoi(headerContentLength)
			if Error(err) {
				return nil, nil, err
			}

			Debug("Explicit set %s with HTTP PATCH method", CONTENT_LENGTH)

			// https://stackoverflow.com/questions/63537645/content-length-header-is-not-getting-set-for-patch-requests-with-empty-nil-paylo
			req.TransferEncoding = []string{"identity"}
			req.ContentLength = int64(contentLength)
		}
	}

	if formdata != nil {
		if headers.Get(CONTENT_TYPE) == "" {
			headers.Set("CONTENT_TYPE", "MimetypeApplicationXWWWFormUrlencoded.MimeType")
		}

		req.PostForm = formdata
	}

	req.Header = headers

	if username != "" || password != "" {
		if username == BEARER {
			req.Header[AUTHORIZATION] = []string{fmt.Sprintf("%s %s", BEARER, password)}
		} else {
			if username == "" {
				username = "dummy"
			}

			req.SetBasicAuth(username, password)
		}
	}

	if IsLogVerboseEnabled() {
		ba, err := httputil.DumpRequestOut(req, IsTextMimeType(req.Header.Get(CONTENT_TYPE)))
		if Error(err) {
			return nil, nil, err
		}

		s := HideSecrets(string(ba))

		Debug("HTTP Request: %s %s Username: %s Password: %s\n%s\n", method, address, username, strings.Repeat("X", 5)+"...", s)
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

	timeNeeded := time.Since(start)

	if IsLogVerboseEnabled() {
		ba, err := httputil.DumpResponse(resp, IsTextMimeType(resp.Header.Get(CONTENT_TYPE)))
		if Error(err) {
			return nil, nil, err
		}

		s := HideSecrets(string(ba))

		Debug("HTTP Response (after %v): %s %s Username: %s Password: %s\n%s\n", timeNeeded, method, address, username, strings.Repeat("X", 5)+"...", s)
	}

	ba, err := ReadBody(resp.Body)
	if Error(err) {
		return nil, nil, err
	}

	if expectedCode > 0 && resp.StatusCode != expectedCode {
		return nil, nil, NewHTTPError(resp.StatusCode, fmt.Errorf("Unexpected HTTP status code, expected %d got %d\nResponseBody\n%s", expectedCode, resp.StatusCode, ba))
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

func ConcurrentLimitHandler(next http.HandlerFunc) http.HandlerFunc {
	DebugFunc()

	return func(w http.ResponseWriter, r *http.Request) {
		defer UnregisterConcurrentLimit(RegisterConcurrentLimit())

		next.ServeHTTP(w, r)
	}
}
