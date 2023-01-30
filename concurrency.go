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
	Duration time.Duration
	Err      error
}

func (e *ErrTimeout) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Timeout error after: %+v, error: %+v", e.Duration, e.Err)
	} else {
		return fmt.Sprintf("Timeout error after: %+v", e.Duration)
	}
}

func (e *ErrTimeout) Timeout() bool {
	return true
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

func GoRoutineName() string {
	buf := make([]byte, 100)
	runtime.Stack(buf, true)
	buf = bytes.Split(buf, []byte{'\n'})[0]
	buf = buf[:len(buf)-1]
	return string(bytes.TrimSuffix(buf, []byte("[running]")))
}

type Channel[K any] struct {
	mu   sync.RWMutex
	ch   chan K
	open bool
}

func NewChannel[T any](len int) *Channel[T] {
	return &Channel[T]{
		ch:   make(chan T, len),
		open: true,
	}
}

func (ch *Channel[T]) isOpen() error {
	if !ch.open {
		return fmt.Errorf("channel closed")
	}

	return nil
}

func (ch *Channel[T]) IsOpen() error {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.isOpen()
}

func (ch *Channel[T]) Put(value T) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	err := ch.isOpen()

	if Error(err) {
		return err
	}

	ch.ch <- value

	return nil
}

func (ch *Channel[T]) Get() (T, bool) {
	err := ch.IsOpen()

	if err != nil {
		return *new(T), false
	}

	value, ok := <-ch.ch

	return value, ok
}

func (ch *Channel[T]) Close() error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	err := ch.isOpen()

	if Error(err) {
		return err
	}

	close(ch.ch)

	return nil
}

type Chrono struct {
	ticker *time.Ticker
	done   chan struct{}
	run    func(*Chrono)
}

func NewChrono(d time.Duration, run func(*Chrono)) *Chrono {
	c := &Chrono{
		ticker: time.NewTicker(d),
		done:   make(chan struct{}),
		run:    run,
	}

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		for {
			select {
			case <-c.done:
				return
			case _ = <-c.ticker.C:
				c.run(c)
			}
		}
	}()

	return c
}

func (c *Chrono) Stop() {
	close(c.done)
}

type TimeoutRegister[T comparable] struct {
	mutex    sync.Mutex
	timeout  time.Duration
	quit     chan struct{}
	register map[T]time.Time
	ticker   *time.Ticker
}

func NewTimeoutRegister[T comparable](timeout time.Duration) *TimeoutRegister[T] {
	tr := &TimeoutRegister[T]{
		mutex:    sync.Mutex{},
		timeout:  timeout,
		quit:     make(chan struct{}),
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
			case <-tr.quit:
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

func (tr *TimeoutRegister[T]) Quit() {
	DebugFunc()

	tr.quit <- struct{}{}
}
