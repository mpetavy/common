package common

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestNewRestURL(t *testing.T) {
	u := NewRestURL(http.MethodGet, "/patient/{id}/doc/{id2}")

	require.Equal(t, "/patient/123/doc/456", u.Format(123, 456))
}
