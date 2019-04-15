package common

import (
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
