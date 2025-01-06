package common

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestParseDateTime(t *testing.T) {
	//txt := "19.05.1970 12:34:56 789"
	local, err := time.LoadLocation("Local")
	require.NoError(t, err)

	ti, err := ParseDateTime(DateMask, "19.05.1970")
	require.NoError(t, err)
	require.Equal(t, time.Date(1970, time.May, 19, 0, 0, 0, 0, local), ti)

	ti, err = ParseDateTime(TimeMask, "12:34:56")
	require.NoError(t, err)
	require.Equal(t, time.Date(0, 1, 1, 12, 34, 56, 0, local), ti)

	ti, err = ParseDateTime(DateTimeMask, "19.05.1970 12:34:56")
	require.NoError(t, err)
	require.Equal(t, time.Date(1970, time.May, 19, 12, 34, 56, 0, local), ti)

	ti, err = ParseDateTime(SortedDateMask, "1970-05-19")
	require.NoError(t, err)
	require.Equal(t, time.Date(1970, time.May, 19, 0, 0, 0, 0, local), ti)

	ti, err = ParseDateTime(SortedDateTimeMilliMask, "1970-05-19 12:34:56.789")
	require.NoError(t, err)
	require.Equal(t, time.Date(1970, time.May, 19, 12, 34, 56, 789000000, local), ti)
}

func TestCompareDate(t *testing.T) {
	t0, err := ParseDateTime(SortedDateTimeMilliMask, "1970-01-01 01:01:01.001")
	require.NoError(t, err)

	t1, err := ParseDateTime(SortedDateTimeMilliMask, "1970-12-31 23:23:23.999")
	require.NoError(t, err)

	require.Equal(t, time.Duration(0), CompareDate(t0, t0))
	require.True(t, CompareDate(t0, t1) < time.Duration(0))
	require.True(t, CompareDate(t1, t0) > time.Duration(0))
}

func TestCompareTime(t *testing.T) {
	t0, err := ParseDateTime(SortedDateTimeMilliMask, "1970-01-01 01:01:01.001")
	require.NoError(t, err)

	t1, err := ParseDateTime(SortedDateTimeMilliMask, "1970-12-31 23:23:23.999")
	require.NoError(t, err)

	require.Equal(t, time.Duration(0), CompareTime(t0, t0))
	require.True(t, CompareTime(t0, t1) < time.Duration(0))
	require.True(t, CompareTime(t1, t0) > time.Duration(0))
}

func TestClearTime(t *testing.T) {
	t0, err := ParseDateTime(SortedDateTimeMilliMask, "1970-05-19 12:34:56.789")
	require.NoError(t, err)

	t1 := ClearTime(t0)

	require.Equal(t, 0, t1.Hour())
	require.Equal(t, 0, t1.Minute())
	require.Equal(t, 0, t1.Second())
}

func TestClearDate(t *testing.T) {
	t0, err := ParseDateTime(SortedDateTimeMilliMask, "1970-05-19 12:34:56.789")
	require.NoError(t, err)

	t1 := ClearDate(t0)

	require.Equal(t, 0, t1.Year())
	require.Equal(t, time.Month(1), t1.Month())
	require.Equal(t, 1, t1.Day())
}

func TestTruncateTime(t *testing.T) {
	t0, err := ParseDateTime(SortedDateTimeMilliMask, "1970-05-19 12:34:56.789")
	require.NoError(t, err)

	t1 := TruncateTime(t0, Day)

	require.Equal(t, 0, t1.Second())
}
