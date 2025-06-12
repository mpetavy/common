package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

const (
	content = `
path = "c:\temp"
port=8855
log.verbose=      true
secret = "-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQClszcOVRuQXRhM
AxaCHI1VWi87Df3rWNgtRniouQgWJ3LmovgYTBsMjb1gxSUkLRFbVqqFy3gwxWq6
hT3gW2quscMkdxLGphRiBV5/RzxxmMwSqnCBqgyCvN6k1rQQlKCIZ+x4h7d9Jc6H
KIGeF7mNRuy1i95V+q6mXpF7GljH1nItf9RIcIbScqHi6XX/fRSa/SLD1+QX0BLg
Ln8tdpakHFPq1M+hLNlopVZUP7YYAKypr7zZTvT/Rj1H7BaG1a3U0vs8DrvjWEjm
VKWw+9ufMp9JpIYi+dsyFLxDQOVr9KTh4xTjsPvacxuKPRA79JZdo/nCeWH0n4QO
OIDpfAgDAgMBAAECggEAFmCJ/l8C/m98EQPXvdGCSrUHrNd6Y5aXdyHNuKdoTqmc
LEZ0778TZhIcMZ3eIrENZ4LgO3pbbGa0v6Sv0wU1doseGeYUvIwAM66a8OBbatHi
OWEYGYKv9tXv0V4HajfQKCu0tSBK6NU6u2j+fC2jCs+5ttjBOWZFwMUDq5bGx5HX
29NrFimQD9NVlPqLsBHHc7LSN8BS2Ju476qjLwu7qS1adTjIvdyfcwSfQmxUAo2k
V24KdNIJi/nolWuxFwHaauJcO+M5ZEfG7XWLp6RDmMsaxToHrD3R++RRWZhhBB4K
V20b8M1XdNu0W7nB2bcMTyo5wMnnHwL6Dy1Utm76KQKBgQDliogtkJOmyg24ekFC
XORQOa9T/jsIbx0lb5GIE9iV06ZfszN3+eF26Xoigh0ChU4PIBdl7BbukNUlfc2X
gUiu8+pJwdXydLArHT+wB4WqXRQvMCM2pXmZXTEig/zD6n1QeoeRkPS+dirkwgRy
D4L5En6CtOYtfPtOVVmpqXitaQKBgQC4zNBQ3eFaHRfM870jdnkFeez0fOijdbd/
EMwWmBCcuOL0INv9HUEUF3iKJV1OJrdy6h71nN2YXyf/d/biGSYnvhqvw4EbA82c
isdDlI0jL0S7jh8oHZYPXa5FPtjq3d16r9SNm8qfWspBESM2Etki0bulRk6GfIbk
dADfyirgiwKBgBExkEPBeZ3bsq1n0u2SobN0rrJe77MRB6DfO4py2h1W7jZq6OcK
u525nWFqV5vxukgdwkLrLUiPZrfZNYYss/IO6TS/JTR1EyEXnsajuZpqQHHMbEbS
nEolleGc+1j9foeBthfsQLjnhwz9j3GvwcLAZOOLg1ZS70wNzpqLzDNJAoGAOblI
LKpR9Or3fz53Svd7r/k4ydmmdUCU86zUgw42yi16PtVwwex8YoE+VrB7J6kyTkPR
Ldk04p5+iO75AADpCSr5fQNtdXnHpOk4euSQ/XeLWaZ4Fvi+4cfaYqjR6vModmUr
2JvcO9CJMq/etspGZvjqSyLd7mZBYGTXzQ+COycCgYAtJOU902nPCiK4qBtAvcGW
OXFucRCTTe3BL3E/aGCJlnv0MECxffL9yUu8foTXfQJT+pT01ZMpsRJhozHjg5gp
b0QhrRXjnl8LxA00S2MsF/5aWtRzYOgk2GSZQYQL/dFdi2kUrxSqd5LSpvrbdoXK
P5GwlxdeUtD8sl3mJpYE2Q==
-----END PRIVATE KEY-----
"
`
)

func TestIniFileContent(t *testing.T) {
	ini := NewIniFile()

	err := ini.Load([]byte(content))
	require.NoError(t, err)

	require.Equal(t, "c:\\temp", ini.Get("path"))
	require.Equal(t, "8855", ini.Get("port"))
	require.Equal(t, "true", ini.Get("log.verbose"))
	fmt.Printf("***%s***\n", ini.Get("secret"))
	require.True(t, strings.HasPrefix(ini.Get("secret"), "-----BEGIN PRIVATE KEY-----"))
	require.True(t, strings.HasSuffix(ini.Get("secret"), "-----END PRIVATE KEY-----\n"))
}

func TestIniFileOperate(t *testing.T) {
	ini := NewIniFile()

	require.Equal(t, 0, len(ini.Sections()))
	require.Equal(t, 0, len(ini.Keys()))
	require.Equal(t, 0, len(ini.Keys(DEFAULT_SECTION)))

	ini.Set("foo", "bar")
	require.Equal(t, 1, len(ini.Sections()))
	require.Equal(t, DEFAULT_SECTION, ini.Sections()[0])
	require.Equal(t, "bar", ini.Get("foo"))

	require.Equal(t, []byte("[default]\nfoo=bar\n"), ini.Save())

	ini.Remove("foo")
	require.Equal(t, 0, len(ini.Keys()))
	require.Equal(t, 0, len(ini.Keys(DEFAULT_SECTION)))
	require.Equal(t, "", ini.Get("foo"))

	require.Equal(t, []byte(nil), ini.Save())

	ini.Set("foo", "additional:bar", "additional")
	require.Equal(t, 1, len(ini.Sections()))
	require.Equal(t, []string{"additional"}, ini.Sections())
	require.Equal(t, "", ini.Get("foo"))
	require.Equal(t, "additional:bar", ini.Get("foo", "additional"))

	require.Equal(t, []byte("[additional]\nfoo=additional:bar\n"), ini.Save())

	ini.Set("foo", "bar")
	require.Equal(t, 2, len(ini.Sections()))
	require.Equal(t, []string{DEFAULT_SECTION, "additional"}, ini.Sections())
	require.Equal(t, "bar", ini.Get("foo"))
	require.Equal(t, "bar", ini.Get("foo", DEFAULT_SECTION))
	require.Equal(t, "additional:bar", ini.Get("foo", DEFAULT_SECTION, "additional"))

	require.Equal(t, []byte("[default]\nfoo=bar\n[additional]\nfoo=additional:bar\n"), ini.Save())

	ini.Remove("foo", ini.Sections()...)
	require.Equal(t, 0, len(ini.Keys()))
	require.Equal(t, 0, len(ini.Keys(DEFAULT_SECTION)))
	require.Equal(t, "", ini.Get("foo"))

	require.Equal(t, []byte(nil), ini.Save())

	ini.Set("foo", "bar")
	ini.Set("foo", "additional:bar", "additional")

	require.Equal(t, []string{DEFAULT_SECTION, "additional"}, ini.Sections())

	ini.RemoveSection()
	require.Equal(t, []string{"additional"}, ini.Sections())

	require.Equal(t, []byte("[additional]\nfoo=additional:bar\n"), ini.Save())

	ini.RemoveSection("asdf")
	require.Equal(t, []string{"additional"}, ini.Sections())

	ini.RemoveSection("additional")
	require.Equal(t, 0, len(ini.Sections()))

	require.Equal(t, []byte(nil), ini.Save())

	ini.Set("foo", "\nHello\nworld!")
	require.Equal(t, "\nHello\nworld!", ini.Get("foo"))

	ini.Clear()

	err := ini.LoadFile("./testdata/sample.ini")
	require.NoError(t, err)

	secret := `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQClszcOVRuQXRhM
AxaCHI1VWi87Df3rWNgtRniouQgWJ3LmovgYTBsMjb1gxSUkLRFbVqqFy3gwxWq6
-----END PRIVATE KEY-----
`

	require.Equal(t, "true", ini.Get("log.verbose"))
	require.Equal(t, "jsmith", ini.Get("user.name"))
	require.Equal(t, secret, ini.Get("secret"))
}
