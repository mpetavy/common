package reentrant

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"

	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	rl := New()
	pass := NewPass()

	rl.Lock(pass)

	select {
	case <-time.After(time.Millisecond * 100):
		t.Fatal("test did not finish in time")
	default:
		rl.Lock(pass)
		rl.Lock(pass)
		rl.Unlock(pass)

		assert.Equal(t, 2, pass.c)

		rl.UnlockNow()

		assert.Equal(t, 2, pass.c)
	}
}

func TestReentrantLock(t *testing.T) {
	countGoroutines := 10
	countLoop := 10

	var c uint64 = 0

	list := make([]string, 0)
	mu := sync.Mutex{}

	f := func() {
		rl := New()
		wg := sync.WaitGroup{}

		for goroutine := 0; goroutine < countGoroutines; goroutine++ {

			wg.Add(1)
			go func(goroutine int) {
				defer func() {
					wg.Done()
				}()

				pass := NewPass()

				for i := 0; i < countLoop; i++ {
					n := fmt.Sprintf("GO routine #%d", goroutine)

					rl.Lock(pass)

					// just check that we can reentrant ...
					rl.Lock(pass)

					atomic.AddUint64(&c, 1)

					mu.Lock()
					list = append(list, n)
					mu.Unlock()

					rl.Unlock(pass)

					rl.Unlock(pass)
				}
			}(goroutine)
		}

		wg.Wait()
	}

	select {
	case <-time.After(time.Second):
		t.Fatal("test did not finish in time")
	default:
		f()
	}

	for i, e := range list {
		t.Logf("#%4d %s\n", i, e)
	}

	t.Logf("%d\n", len(list))

	//if c != uint64(countGoroutines * countLoop) {
	if len(list) != countGoroutines*countLoop {
		t.Fatal("unexpected len of generated list entries")
	}
}
