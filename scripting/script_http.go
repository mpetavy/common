package scripting

import (
	"bytes"
	"crypto/tls"
	"github.com/dop251/goja"
	"github.com/mpetavy/common"
	"io"
	"net/http"
)

type gojaHttp struct{}

func (c *gojaHttp) execute(method string, url string, username string, password string, header map[string][]string, body []byte) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if header != nil {
		req.Header = header
	}
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if common.Error(err) {
		return resp, err
	}

	return resp, nil
}

func (c *gojaHttp) body(resp *http.Response) ([]byte, error) {
	ba, err := io.ReadAll(resp.Body)

	defer func() {
		common.Error(resp.Body.Close())
	}()

	if common.Error(err) {
		return nil, err
	}

	return ba, nil
}

func registerHttp(vm *goja.Runtime) error {
	h := &gojaHttp{}

	obj := vm.NewObject()

	err := obj.Set("execute", h.execute)
	if common.Error(err) {
		return err
	}

	err = obj.Set("body", h.body)
	if common.Error(err) {
		return err
	}

	err = vm.Set("http", obj)
	if common.Error(err) {
		return err
	}

	return nil
}
