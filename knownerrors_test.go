package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsError(t *testing.T) {
	httperr := &ErrHTTPRequest{
		StatusCode: 401,
	}

	require.False(t, IsError[*ErrHTTPRequest](nil))
	require.False(t, IsError[*ErrHTTPRequest](fmt.Errorf("dummy")))
	require.True(t, IsError[*ErrHTTPRequest](httperr))
	require.True(t, IsError[*ErrHTTPRequest](httperr))
	require.False(t, IsError[*ErrHTTPRequest](httperr, func(request *ErrHTTPRequest) bool {
		return request.StatusCode == 500
	}))
	require.True(t, IsError[*ErrHTTPRequest](httperr, func(request *ErrHTTPRequest) bool {
		return request.StatusCode == 401
	}))
}
