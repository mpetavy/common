package common

import (
	"strings"
	"sync"
	"time"
)

type LoopNotifier struct {
	NextTime time.Time
	mu       sync.Mutex
}

func (loopNotifier *LoopNotifier) Reset() {
	loopNotifier.mu.Lock()
	defer loopNotifier.mu.Unlock()

	loopNotifier.NextTime = time.Time{}
}

func (loopNotifier *LoopNotifier) Notify() {
	loopNotifier.mu.Lock()
	defer loopNotifier.mu.Unlock()

	current := time.Now()

	if loopNotifier.NextTime.IsZero() {
		loopNotifier.NextTime = current.Add(time.Second)

		return
	}

	if current.After(loopNotifier.NextTime) {
		loopNotifier.NextTime = current.Add(time.Second)

		logEntry := formatLog(LevelDebug, 2, strings.TrimSpace("Loop notify()"), false)

		logWarnPrint(logEntry.PrintMsg)
	}
}
