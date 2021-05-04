package common

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type TimeoutError error

func NewTimeoutError(maxDuration time.Duration) TimeoutError {
	return TimeoutError(fmt.Errorf("Timeout error after: %+v", time.Duration(maxDuration)))
}

func NewTimeoutOperation(checkDuration time.Duration, maxDuration time.Duration, fn func() (error)) error {
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
			return NewTimeoutError(maxDuration)
		}
	}

	return nil
}

var (
	routines        = make(map[int]RuntimeInfo)
	routinesCounter = 0
	routinesMutex   = sync.Mutex{}
)

func RegisterGoRoutine() int {
	routinesMutex.Lock()
	defer routinesMutex.Unlock()

	ri := GetRuntimeInfo(1)
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
	for k, _ := range routines {
		ks = append(ks, k)
	}

	sort.Ints(ks)

	for _, k := range ks {
		f(k, routines[k])
	}
}
