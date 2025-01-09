package common

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestEncrypt(t *testing.T) {
	key := RndBytes(16)

	txt := "Hello world!"
	encrypted, err := EncryptString(key, txt)
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(encrypted, SECRET_PREFIX))

	decrypted, err := DecryptString(key, encrypted)
	require.NoError(t, err)

	require.Equal(t, txt, decrypted)
}
