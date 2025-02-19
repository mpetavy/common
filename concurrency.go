package common

import (
	"bufio"
	"bytes"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	FlagNameConcurrentLimit   = "concurrent.limit"
	FlagNameConcurrentTimeout = "concurrent.timeout"
)

var (
	FlagConcurrentLimit   = SystemFlagInt(FlagNameConcurrentLimit, Max(4, runtime.NumCPU()*2), "Limit of maximum current running tasks")
	FLagConcurrentTimeout = SystemFlagInt(FlagNameConcurrentTimeout, 10000, "Tinmeout waiting for running a current running tasks")

	routines        = make(map[int]RuntimeInfo)
	routinesCounter = 0
	routinesMutex   = sync.Mutex{}

	onceConcurrentLimit sync.Once
	concurrentLimitCh   chan struct{}
)

func RegisterConcurrentLimit() bool {
	onceConcurrentLimit.Do(func() {
		concurrentLimitCh = make(chan struct{}, *FlagConcurrentLimit)
	})

	if *FlagConcurrentLimit == 0 {
		return false
	}

	DebugFunc("Register...")

	defer func() {
		DebugFunc("Run")
	}()

	select {
	case concurrentLimitCh <- struct{}{}:
		return true
	case <-time.After(MillisecondToDuration(*FLagConcurrentTimeout)):
		Warn("Concurrent limit exceeded")

		return false
	}
}

func UnregisterConcurrentLimit(fromChannel bool) {
	if *FlagConcurrentLimit == 0 {
		return
	}

	DebugFunc("Unregister")

	if fromChannel {
		<-concurrentLimitCh
	}
}

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

func getRoutineId(s string) uint64 {
	s = strings.TrimPrefix(s, "goroutine ")

	i := strings.Index(s, " ")
	if i == -1 {
		i = len(s)
	}

	n, err := strconv.ParseUint(s[:i], 10, 64)

	Panic(err)

	return n
}

func GoRoutineId() uint64 {
	b := make([]byte, 64*1024)
	b = b[:runtime.Stack(b, false)]

	return getRoutineId(string(b))
}

func GoRoutineIds() []uint64 {
	ids := make([]uint64, 0)

	b := make([]byte, 1024*1024)
	b = b[:runtime.Stack(b, true)]

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "goroutine ") {
			continue
		}

		id := getRoutineId(line)

		ids = append(ids, id)
	}

	return ids
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

type BackgroundTask struct {
	fn      func(task *BackgroundTask)
	wg      sync.WaitGroup
	aliveCh chan struct{}
	isAlive atomic.Bool
}

func NewBackgroundTask(fn func(task *BackgroundTask)) *BackgroundTask {
	return &BackgroundTask{fn: fn}
}

func (bt *BackgroundTask) Start() {
	DebugFunc()

	if bt.isAlive.Load() {
		return
	}

	bt.isAlive.Store(true)
	bt.aliveCh = make(chan struct{})

	bt.wg.Add(1)
	go func() {
		defer func() {
			bt.wg.Done()
		}()

		bt.fn(bt)
	}()
}

func (bt *BackgroundTask) IsAlive() bool {
	return bt.isAlive.Load()
}

func (bt *BackgroundTask) Channel() chan struct{} {
	return bt.aliveCh
}

func (bt *BackgroundTask) Stop(waitFor bool) {
	DebugFunc()

	if !bt.isAlive.Load() {
		return
	}

	bt.isAlive.Store(false)
	close(bt.aliveCh)

	if waitFor {
		bt.wg.Wait()
	}
}

type AlignedTicker struct {
	IsFirstTicker bool
	SleepTime     time.Duration
	now           time.Time
}

func NewAlignedTicker(sleepTime time.Duration) *AlignedTicker {
	alignedTicker := &AlignedTicker{
		IsFirstTicker: false,
		SleepTime:     sleepTime,
		now:           time.Time{},
	}

	return alignedTicker
}

func (at *AlignedTicker) SleepUntilNextTicker() time.Duration {
	var delta time.Duration

	if at.now.IsZero() {
		at.now = time.Now()
	}

	if !at.IsFirstTicker {
		nextTick := time.Date(at.now.Year(), at.now.Month(), at.now.Day(), 0, 0, 0, 0, at.now.Location())
		for nextTick.Before(at.now) || nextTick.Equal(at.now) {
			nextTick = nextTick.Add(at.SleepTime)
		}

		delta = nextTick.Sub(at.now)

		Debug("Next ticker: %v sleep: %v\n", nextTick, delta)
	}

	at.IsFirstTicker = false

	return delta
}
