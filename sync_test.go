package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestSync(t *testing.T) {
	syncErr := NewSync[error]()

	assert.Nil(t, syncErr.Get())

	err := fmt.Errorf("error")

	syncErr.Set(err)

	e := syncErr.Get()

	assert.NotNil(t, syncErr.Get())
	assert.True(t, syncErr.IsSet())
	assert.Equal(t, "error", e.Error())
}

func TestSyncSame(t *testing.T) {
	str := "Hello world"

	s := NewSyncOf(&str)

	assert.True(t, &str == s.Get())
}

func TestSynOf(t *testing.T) {
	var counter int

	s := NewSyncOf(&counter)

	count := 1000
	wg := sync.WaitGroup{}

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			s.RunSynchronized(func(v *int) {
				*v++
			})
		}()
	}

	wg.Wait()

	assert.Equal(t, count, counter)
}
