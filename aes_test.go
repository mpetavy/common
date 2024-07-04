package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncrypt(t *testing.T) {
	key, err := RndBytes(16)
	if err != nil {
		Error(err)
	}

	txt := "Hello world!"
	enc, err := EncryptString(key, txt)
	if err != nil {
		Error(err)
	}

	dec, err := DecryptString(key, enc)
	if err != nil {
		Error(err)
	}

	assert.Equal(t, txt, dec)
}
