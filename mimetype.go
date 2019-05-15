package common

import (
	"os"
	"strings"
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
)

const MIME_TYPE_HEADER_LEN = 1024

func DetectMimeType(buf []byte) (string, string) {
	t, err := filetype.Match(buf)
	if err == nil {
		return t.MIME.Value, t.Extension
	}

	mime, ext := mimetype.Detect(buf)

	if len(mime) > 0 {
		return mime, ext
	}

	return "", ""
}

func ReadHeader(path string) ([]byte,error) {
	byteSlice := make([]byte, MIME_TYPE_HEADER_LEN)

	file, err := os.Open(path)
	if err != nil {
		return byteSlice[0:0],err
	}
	defer file.Close()

	bytesRead, err := file.Read(byteSlice)
	if err != nil {
		return byteSlice[0:0],err
	}

	return byteSlice[:bytesRead],nil
}

func IsImageMimeType(s string) bool {
	return strings.HasPrefix(s,"image/") 
}