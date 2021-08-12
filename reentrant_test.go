package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"

	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	remutex := NewRentrantMutex()
	pass := remutex.NewPass()

	pass.Lock()

	select {
	case <-time.After(time.Millisecond * 100):
		t.Fatal("test did not finish in time")
	default:
		pass.Lock()
		pass.Lock()
		pass.Unlock()

		assert.Equal(t, 2, pass.c)

		remutex.UnlockNow()

		assert.Equal(t, 2, pass.c)
	}
}

func TestBlocking(t *testing.T) {
	remutex := NewRentrantMutex()
	pass0 := remutex.NewPass()
	pass1 := remutex.NewPass()

	pass0.Lock()

	start := time.Now()
	d := time.Millisecond * 100

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		time.Sleep(d)
		pass0.Unlock()
	}()

	pass1.Lock()

	assert.True(t, time.Now().Sub(start) >= d)
}

func TestLock(t *testing.T) {
	countGoroutines := 10
	countLoop := 10

	var c uint64 = 0

	list := make([]string, 0)
	mu := sync.Mutex{}

	f := func() {
		rl := NewRentrantMutex()
		wg := sync.WaitGroup{}

		for goroutine := 0; goroutine < countGoroutines; goroutine++ {

			wg.Add(1)
			go func(goroutine int) {
				defer UnregisterGoRoutine(RegisterGoRoutine(1))

				defer func() {
					wg.Done()
				}()

				pass := rl.NewPass()

				for i := 0; i < countLoop; i++ {
					n := fmt.Sprintf("GO routine #%d", goroutine)

					pass.Lock()

					// just check that we can reentrant ...
					pass.Lock()

					atomic.AddUint64(&c, 1)

					mu.Lock()
					list = append(list, n)
					mu.Unlock()

					pass.Unlock()

					pass.Unlock()
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

	//for i, e := range list {
	//	t.Logf("#%4d %s\n", i, e)
	//}
	//
	//t.Logf("%d\n", len(list))

	//if c != uint64(countGoroutines * countLoop) {
	if len(list) != countGoroutines*countLoop {
		t.Fatal("unexpected len of generated list entries")
	}
}
