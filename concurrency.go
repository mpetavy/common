package common

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
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

	registeredGoRoutines      = make(map[int]RuntimeInfo)
	registeredGoRoutinesMutex = sync.Mutex{}

	onceConcurrentLimit sync.Once
	concurrentLimitCh   chan struct{}
)

func RegisterConcurrentLimit() bool {
	if *FlagConcurrentLimit == 0 {
		return false
	}

	onceConcurrentLimit.Do(func() {
		concurrentLimitCh = make(chan struct{}, *FlagConcurrentLimit)
	})

	ti := time.NewTimer(MillisecondToDuration(*FLagConcurrentTimeout))
	defer func() {
		ti.Stop()
	}()

	select {
	case concurrentLimitCh <- struct{}{}:
		if !ti.Stop() {
			<-ti.C
		}

		return true
	case <-ti.C:
		Info(fmt.Sprintf("Concurrent limit exceeded. Limit: %d timeout: %d", *FlagConcurrentLimit, *FLagConcurrentTimeout))

		return false
	}
}

func UnregisterConcurrentLimit(fromChannel bool) {
	if *FlagConcurrentLimit == 0 {
		return
	}

	if fromChannel {
		<-concurrentLimitCh
	}
}

func RegisterGoRoutine(index int) int {
	registeredGoRoutinesMutex.Lock()
	defer registeredGoRoutinesMutex.Unlock()

	ri := GetRuntimeInfo(index)
	id := int(GoRoutineId())

	registeredGoRoutines[id] = ri

	return id
}

func UnregisterGoRoutine(id int) {
	registeredGoRoutinesMutex.Lock()
	defer registeredGoRoutinesMutex.Unlock()

	delete(registeredGoRoutines, id)
}

func NumRegisteredGoRoutines() int {
	registeredGoRoutinesMutex.Lock()
	defer registeredGoRoutinesMutex.Unlock()

	return len(registeredGoRoutines)
}

func RegisteredGoRoutines(f func(id int, ri RuntimeInfo)) {
	registeredGoRoutinesMutex.Lock()
	defer registeredGoRoutinesMutex.Unlock()

	ks := make([]int, 0)
	for k := range registeredGoRoutines {
		ks = append(ks, k)
	}

	sort.Ints(ks)

	for _, k := range ks {
		f(k, registeredGoRoutines[k])
	}
}

func getRoutineId(s string) uint64 {
	s = strings.TrimPrefix(s, "goroutine ")

	i := strings.Index(s, " ")
	if i == -1 {
		i = len(s)
	}

	n, err := strconv.ParseUint(s[:i], 10, 64)
	if err != nil {
		WarnError(errors.Wrap(err, fmt.Sprintf("parsing goroutine id from: %s", s)))

		return 0
	}

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
	TickerTime time.Duration
	next       time.Time
	now        time.Time
}

func NewAlignedTicker(tickerTime time.Duration) *AlignedTicker {
	return &AlignedTicker{
		TickerTime: tickerTime,
		next:       time.Now().Truncate(24 * time.Hour),
	}
}

func (at *AlignedTicker) NextTicker() time.Duration {
	current := at.now
	if at.now.IsZero() {
		current = time.Now()
	}

	for at.next.Before(current) || at.next.Equal(current) {
		at.next = at.next.Add(at.TickerTime)
	}

	delta := at.next.Sub(current)

	Debug("Next ticker: %v sleep: %v\n", at.next, delta)

	return delta
}
