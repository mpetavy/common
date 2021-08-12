package common

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

type ErrTimeout struct {
	Timeout time.Duration
	Err     error
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("Timeout error after: %+v, error: %+v", e.Timeout, e.Err)
}

func NewTimeoutOperation(checkDuration time.Duration, maxDuration time.Duration, fn func() error) error {
	start := time.Now()

	err := fn()

	if err == nil {
		return nil
	}

	ti := time.NewTicker(checkDuration)
	defer ti.Stop()

	for {
		<-ti.C

		err := fn()

		if err == nil {
			return nil
		}

		if time.Since(start) > maxDuration {
			return &ErrTimeout{maxDuration, err}
		}
	}
}

var (
	routines        = make(map[int]RuntimeInfo)
	routinesCounter = 0
	routinesMutex   = sync.Mutex{}
)

func RegisterGoRoutine(index int) int {
	routinesMutex.Lock()
	defer routinesMutex.Unlock()

	ri := GetRuntimeInfo(index)
	id := routinesCounter
	routinesCounter++

	routines[id] = ri

	return id
}

func UnregisterGoRoutine(id int) {
	routinesMutex.Lock()
	defer routinesMutex.Unlock()

	delete(routines, id)
}

func RegisteredGoRoutines(f func(id int, ri RuntimeInfo)) {
	routinesMutex.Lock()
	defer routinesMutex.Unlock()

	ks := make([]int, 0)
	for k := range routines {
		ks = append(ks, k)
	}

	sort.Ints(ks)

	for _, k := range ks {
		f(k, routines[k])
	}
}

func GoRoutineId() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)

	return n
}
