package scripting

import (
	"bytes"
	"crypto/tls"
	"github.com/dop251/goja"
	"github.com/mpetavy/common"
	"io"
	"net/http"
	"strings"
)

type gojaHttp struct{}

func (c *gojaHttp) execute(method string, url string, username string, password string, header map[string][]string, body []byte) (*http.Response, error) {
	var tr *http.Transport

	if strings.Contains(url, "https") {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, _, err := common.HTTPRequest(tr, common.MillisecondToDuration(*common.FlagHTTPTimeout), method, url, header, nil, username, password, bytes.NewReader(body), 0)
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
