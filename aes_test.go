package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncrypt(t *testing.T) {
	key := RndBytes(16)

	txt := "Hello world!"
	enc, err := EncryptString(key, txt)
	if err != nil {
		Error(err)
	}

	dec, err := DecryptString(key, enc)
	if err != nil {
		Error(err)
	}

	require.Equal(t, txt, dec)
}
