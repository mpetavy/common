package common

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestSync(t *testing.T) {
	var counter int

	s := NewSync(&counter)

	count := 1000
	wg := sync.WaitGroup{}

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			s.Run(func(v *int) {
				*v++
			})
		}()
	}

	wg.Wait()

	assert.Equal(t, count, counter)
}
