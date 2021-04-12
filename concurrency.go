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

func NewTimeoutOperation(maxDuration time.Duration, checkDuration time.Duration, fn func() (bool, error)) error {
	start := time.Now()
	loop := true

	ti := time.NewTicker(checkDuration)
	defer ti.Stop()

	for loop {
		<-ti.C

		if time.Since(start) > maxDuration {
			return NewTimeoutError(maxDuration)
		}

		var err error

		loop, err = fn()
		if Error(err) {
			return err
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
