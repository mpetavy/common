package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKeyValueList(t *testing.T) {
	kvl := make(KeyValueList, 0)

	kvl.Put("a", "1")
	assert.Equal(t, 1, len(kvl))
	kvl.Put("b", "2")
	assert.Equal(t, 2, len(kvl))
	kvl.Put("c", "3")
	assert.Equal(t, 3, len(kvl))

	a, _ := kvl.Get("a")
	b, _ := kvl.Get("b")
	c, _ := kvl.Get("c")

	keys := kvl.Keys()
	assert.Equal(t, "a", keys[0])
	assert.Equal(t, "b", keys[1])
	assert.Equal(t, "c", keys[2])

	assert.Equal(t, "1", a)
	assert.Equal(t, "2", b)
	assert.Equal(t, "3", c)

	x, err := kvl.Get("x")
	assert.Equal(t, "", x)
	assert.NotEqual(t, nil, err)

	kvl.Put("b", "99")
	b, _ = kvl.Get("b")
	assert.Equal(t, "99", b)
	assert.Equal(t, 3, len(kvl))

	c, _ = kvl.Remove("c")
	assert.Equal(t, "3", c)
	assert.Equal(t, 2, len(kvl))
	b, _ = kvl.Remove("b")
	assert.Equal(t, "99", b)
	assert.Equal(t, 1, len(kvl))
	a, _ = kvl.Remove("a")
	assert.Equal(t, "1", a)
	assert.Equal(t, 0, len(kvl))
}
