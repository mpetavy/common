package common

import (
	"strings"
	"time"
)

type LoopNotifier struct {
	NextTime time.Time
}

func (loopNotifier *LoopNotifier) Reset() {
	loopNotifier.NextTime = time.Time{}
}

func (loopNotifier *LoopNotifier) Notify() {
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
