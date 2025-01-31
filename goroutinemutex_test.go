package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGoRoutineMutex(t *testing.T) {
	m := GoRoutineMutex{EnterIfSame: false}

	require.True(t, m.TryLock())
	require.False(t, m.TryLock())
	require.False(t, m.TryLock())

	m.Unlock()

	require.True(t, m.TryLock())

	m = GoRoutineMutex{EnterIfSame: true}

	require.True(t, m.TryLock())
	require.True(t, m.TryLock())
}
