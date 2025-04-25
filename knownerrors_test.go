package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsSpecificError(t *testing.T) {
	httperr := &ErrHTTPRequest{
		StatusCode: 401,
	}

	require.False(t, IsSpecificError[*ErrHTTPRequest](fmt.Errorf("dummy"), nil))
	require.True(t, IsSpecificError[*ErrHTTPRequest](httperr, nil))
	require.True(t, IsSpecificError[*ErrHTTPRequest](httperr, nil))
	require.False(t, IsSpecificError[*ErrHTTPRequest](httperr, func(request *ErrHTTPRequest) bool {
		return request.StatusCode == 500
	}))
	require.True(t, IsSpecificError[*ErrHTTPRequest](httperr, func(request *ErrHTTPRequest) bool {
		return request.StatusCode == 401
	}))
}
