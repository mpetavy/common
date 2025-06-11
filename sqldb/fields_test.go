package sqldb

import (
	"encoding/json"
	"fmt"
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	common.RunTests(m)
}

func asJson(t *testing.T, v any) string {
	ba, err := json.Marshal(v)
	require.NoError(t, err)

	return string(ba)
}

func TestFieldString(t *testing.T) {
	v := NewFieldString()
	require.False(t, v.Valid)
	require.Equal(t, "", v.String())
	require.Equal(t, "null", asJson(t, v))

	v = NewFieldString("")
	require.True(t, v.Valid)
	require.Equal(t, "", v.String())
	require.Equal(t, "\"\"", asJson(t, v))

	v = NewFieldString("a")
	require.True(t, v.Valid)
	require.Equal(t, "a", v.String())
	require.Equal(t, "\"a\"", asJson(t, v))

	v.SetNull()
	require.False(t, v.Valid)
	require.Equal(t, "", v.String())
	require.Equal(t, "null", asJson(t, v))
}

func TestFieldInt64(t *testing.T) {
	v := NewFieldInt64()
	require.False(t, v.Valid)
	require.Equal(t, int64(0), v.Int64())
	require.Equal(t, "null", asJson(t, v))

	v = NewFieldInt64(0)
	require.True(t, v.Valid)
	require.Equal(t, int64(0), v.Int64())
	require.Equal(t, "0", asJson(t, v))

	v = NewFieldInt64(1)
	require.True(t, v.Valid)
	require.Equal(t, int64(1), v.Int64())
	require.Equal(t, "1", asJson(t, v))

	v.SetNull()
	require.False(t, v.Valid)
	require.Equal(t, int64(0), v.Int64())
	require.Equal(t, "null", asJson(t, v))
}

func TestFieldTime(t *testing.T) {
	v := NewFieldTime()
	require.False(t, v.Valid)
	require.Equal(t, time.Time{}, v.Time())
	require.Equal(t, "null", asJson(t, v))

	v = NewFieldTime(time.Time{})
	require.True(t, v.Valid)
	require.Equal(t, time.Time{}, v.Time())
	require.Equal(t, "\"0001-01-01T00:00:00Z\"", asJson(t, v))

	x := time.Now().UTC()
	xstr := x.Format("2006-01-02T15:04:05.999999999Z")

	v = NewFieldTime(x)
	require.True(t, v.Valid)
	require.Equal(t, x, v.Time())
	require.Equal(t, fmt.Sprintf("\"%s\"", xstr), asJson(t, v))

	v.SetNull()
	require.False(t, v.Valid)
	require.Equal(t, time.Time{}, v.Time())
	require.Equal(t, "null", asJson(t, v))
}

func TestFieldBool(t *testing.T) {
	v := NewFieldBool()
	require.False(t, v.Valid)
	require.Equal(t, false, v.Bool())
	require.Equal(t, "null", asJson(t, v))

	v = NewFieldBool(true)
	require.True(t, v.Valid)
	require.Equal(t, true, v.Bool())
	require.Equal(t, "true", asJson(t, v))

	v = NewFieldBool(false)
	require.True(t, v.Valid)
	require.Equal(t, false, v.Bool())
	require.Equal(t, "false", asJson(t, v))

	v.SetNull()
	require.False(t, v.Valid)
	require.Equal(t, false, v.Bool())
	require.Equal(t, "null", asJson(t, v))
}

func TestFieldsAsJSON(t *testing.T) {
	type foostruct struct {
		S FieldString `json:"s"`
		I FieldInt64  `json:"i"`
		T FieldTime   `json:"t"`
		B FieldBool   `json:"b"`
	}

	foo := &foostruct{
		S: NewFieldString("a"),
		I: NewFieldInt64(123),
		T: NewFieldTime(time.Time{}),
		B: NewFieldBool(true),
	}

	ba, err := json.MarshalIndent(foo, "", "    ")
	require.NoError(t, err)

	newFoo := &foostruct{}
	err = json.Unmarshal(ba, newFoo)
	require.NoError(t, err)

	require.Equal(t, foo, newFoo)
}
