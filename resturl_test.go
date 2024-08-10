package common

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestNewRestURL(t *testing.T) {
	u := NewRestURL(http.MethodGet, "/patient/{id}/doc/{id2}")
	require.Equal(t, "/patient/123/doc/456", u.Format(123, 456))

	// wrong method

	req, err := http.NewRequest(http.MethodPost, "/patient/123/doc/456", nil)
	require.NoError(t, err)
	require.Error(t, u.Validate(req))

	// correct method

	req, err = http.NewRequest(http.MethodGet, "/patient/123/doc/456", nil)
	require.NoError(t, err)
	require.NoError(t, u.Validate(req))

	// require "offset" value

	u.Params = []RestURLField{{
		Name:        "offset",
		Description: "offset to start from",
		Default:     "",
	}}

	req, err = http.NewRequest(http.MethodGet, "/patient/123/doc/456", nil)
	require.NoError(t, err)
	require.Error(t, u.Validate(req))

	// "offset" value with default value

	u.Params = []RestURLField{{
		Name:        "offset",
		Description: "offset to start from",
		Default:     "123",
	}}

	req, err = http.NewRequest(http.MethodGet, "/patient/123/doc/456", nil)
	require.NoError(t, err)
	require.NoError(t, u.Validate(req))
	require.Equal(t, "123", u.Param(req, "offset"))

	// "offset" value with given value

	req, err = http.NewRequest(http.MethodGet, "/patient/123/doc/456?offset=99", nil)
	require.NoError(t, err)
	require.NoError(t, u.Validate(req))
	require.Equal(t, "99", u.Param(req, "offset"))

	u.Params = []RestURLField{{
		Name:        "offset",
		Description: "offset to start from",
		Default:     "123",
	}}
}
