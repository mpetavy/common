package common

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCatch(t *testing.T) {
	err := Catch(func() error {
		panic("panic")
	})

	require.Error(t, err)
	require.Equal(t, "panic", err.Error())
}

func TestToBool(t *testing.T) {
	require.True(t, ToBool("1"))
	require.True(t, ToBool("T"))
	require.True(t, ToBool("T"))
	require.True(t, ToBool("true"))
	require.True(t, ToBool("Y"))
	require.True(t, ToBool("Yes"))
	require.True(t, ToBool("yes"))
	require.True(t, ToBool("J"))
	require.True(t, ToBool("ja"))

	require.False(t, ToBool(""))
	require.False(t, ToBool("0"))
	require.False(t, ToBool("N"))
	require.False(t, ToBool("no"))
}

func TestToSlice(t *testing.T) {
	require.Equal(t, ToSlice([]string{}...), []string{})
	require.Equal(t, ToSlice([]string{"1", "2", "3"}...), []string{"1", "2", "3"})
}

func TestToAnySlice(t *testing.T) {
	require.Equal(t, ToAnySlice([]string{}...), []any{})
	require.Equal(t, ToAnySlice([]string{"1", "2", "3"}...), []any{"1", "2", "3"})
}

func TestEval(t *testing.T) {
	require.Equal(t, "a", Eval(true, "a", "b"))
	require.Equal(t, "b", Eval(false, "a", "b"))
}

func TestSleep(t *testing.T) {
	start := time.Now()
	Sleep(20 * time.Millisecond)
	require.True(t, time.Since(start) >= 20*time.Millisecond)
}

func TestSleepWithChannel(t *testing.T) {
	ch := make(chan struct{})

	go func() {
		time.AfterFunc(20*time.Millisecond, func() {
			close(ch)
		})
	}()

	start := time.Now()
	SleepWithChannel(time.Second, ch)
	require.True(t, time.Since(start) <= 30*time.Millisecond)
}

func TestRnd(t *testing.T) {
	m := make(map[int]bool)

	for i := 0; i < 10; i++ {
		v := Rnd(10000)
		_, ok := m[v]
		require.False(t, ok)
		m[v] = true
	}
}

func TestRndBytes(t *testing.T) {
	m := make(map[string]bool)

	for i := 0; i < 10; i++ {
		v := string(RndBytes(10))
		_, ok := m[v]
		require.False(t, ok)
		m[v] = true
	}
}

func TestRndString(t *testing.T) {
	m := make(map[string]bool)

	for i := 0; i < 10; i++ {
		v := RndString(10)
		_, ok := m[v]
		require.False(t, ok)
		m[v] = true
	}
}
