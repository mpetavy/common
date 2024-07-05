package common

import (
	"bytes"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"
)

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

func GoRoutineName() string {
	buf := make([]byte, 100)
	runtime.Stack(buf, true)
	buf = bytes.Split(buf, []byte{'\n'})[0]
	buf = buf[:len(buf)-1]
	return string(bytes.TrimSuffix(buf, []byte("[running]")))
}

type TimeoutRegister[T comparable] struct {
	mutex    sync.Mutex
	timeout  time.Duration
	closeCh  chan struct{}
	register map[T]time.Time
	ticker   *time.Ticker
}

func NewTimeoutRegister[T comparable](timeout time.Duration) *TimeoutRegister[T] {
	tr := &TimeoutRegister[T]{
		mutex:    sync.Mutex{},
		timeout:  timeout,
		closeCh:  make(chan struct{}),
		register: make(map[T]time.Time),
		ticker:   time.NewTicker(time.Second),
	}

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))
	loop:
		for {
			select {
			case <-tr.ticker.C:
				tr.clean()
			case <-tr.closeCh:
				tr.ticker.Stop()
				break loop
			}
		}
	}()

	return tr
}

func (tr *TimeoutRegister[T]) clean() {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	modified := false

	now := time.Now()
	for k, v := range tr.register {
		if v.Add(tr.timeout).Before(now) {
			modified = true

			delete(tr.register, k)

			DebugFunc(k)
		}
	}

	if modified {
		DebugFunc("Remain: %d", len(tr.register))
	}
}

func (tr *TimeoutRegister[T]) IsRegistered(item T) bool {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	_, ok := tr.register[item]

	DebugFunc("%s: %v", item, ok)

	return ok
}

func (tr *TimeoutRegister[T]) Register(item T) {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	DebugFunc(item)

	tr.register[item] = time.Now()
}

func (tr *TimeoutRegister[T]) Close() {
	DebugFunc()

	tr.closeCh <- struct{}{}
}

func Background(timeout time.Duration, fn func() error) error {
	errCh := make(chan error, 2)

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		errCh <- fn()
	}()

	ti := time.AfterFunc(timeout, func() {
		errCh <- nil
	})

	err := <-errCh

	ti.Stop()

	return err
}

type GoRoutinesRegister struct {
	list []uint64
	mu   sync.Mutex
}

func NewGoRoutinesRegister() *GoRoutinesRegister {
	return &GoRoutinesRegister{}
}

func (tr *GoRoutinesRegister) Register() {
	mu.Lock()
	defer func() {
		mu.Unlock()
	}()

	id := GoRoutineId()

	if !slices.Contains(tr.list, id) {
		tr.list = append(tr.list, id)
	}
}
func (tr *GoRoutinesRegister) IsRegistered() bool {
	mu.Lock()
	defer func() {
		mu.Unlock()
	}()

	return slices.Contains(tr.list, GoRoutineId())
}

func (tr *GoRoutinesRegister) Deregister() {
	mu.Lock()
	defer func() {
		mu.Unlock()
	}()

	id := GoRoutineId()

	p := slices.Index(tr.list, id)

	if p != -1 {
		tr.list = slices.Delete(tr.list, p, p+1)
	}
}
