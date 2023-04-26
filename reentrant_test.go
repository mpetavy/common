package common

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	rm := NewRentrantMutex()

	rm.Lock()
	rm.Lock()
	rm.Lock()

	assert.Equal(t, rm.count, 3)

	rm.Unlock()

	assert.Equal(t, rm.count, 2)

	rm.UnlockNow()

	assert.Equal(t, rm.count, 0)
	assert.Equal(t, rm.current, uint64(0))
}

func TestBlocking(t *testing.T) {
	rm := NewRentrantMutex()

	start := time.Now()
	wg := sync.WaitGroup{}

	rm.Lock()

	wg.Add(1)
	go func() {
		defer wg.Done()

		rm.Lock()
	}()

	time.Sleep(time.Millisecond * 100)

	rm.Unlock()

	wg.Wait()

	assert.LessOrEqual(t, int64(100), time.Since(start).Milliseconds())
}

func TestBlockingBlock(t *testing.T) {
	rm := NewRentrantMutex()
	list := []int{}
	wg := sync.WaitGroup{}

	for id := 0; id < 100; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			rm.Lock()
			defer rm.Unlock()

			for i := 0; i < 10; i++ {
				list = append(list, id)
			}
		}(id)
	}

	wg.Wait()

	last := -1
	c := 0
	for i := 0; i < len(list); i++ {
		if last != -1 {
			if last == list[i] {
				c++
			} else {
				if i > 0 {
					assert.Equal(t, 10, c)
				}
			}
		}
	}
}
