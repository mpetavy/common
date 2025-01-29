package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIniFile(t *testing.T) {
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
