package common

import (
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestURLBuilder_parse(t *testing.T) {
	tests := []string{
		"",
		"http://www.google.de",
		"http://www.google.de:9999",
		"http://user@www.google.de:9999",
		"http://user:password@www.google.de:9999",
		"http://user:password@www.google.de:9999/path",
		"ftp://ftp.is.co.za/rfc/rfc1808.txt",
		"http://www.ietf.org/rfc/rfc2396.txt",
		"ldap://[2001:db8::7]/c=GB?objectClass?one",
		"ldap://[2001:db8::7]:443/c=GB?objectClass?one",
		"mailto:John.Doe@example.com",
		"news:comp.infosystems.www.servers.unix",
		"tel:+1-816-555-1212",
		"telnet://192.0.2.16:80/",
		"urn:oasis:names:specification:docbook:dtd:xml:4.1.22",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			uri := &URI{}
			assert.NoError(t, uri.Parse(tt))
			assert.Equal(t, tt, uri.String())

			u, err := url.Parse(tt)
			assert.NoError(t, err)
			assert.Equal(t, tt, u.String())
		})
	}
}
