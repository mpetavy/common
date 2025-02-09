package common

import (
	"testing"
	"time"
)

func TestLoopNotifier(t *testing.T) {
	t.Skip()

	rm := NewReentrantMutex()

	rm.Lock()

	go func() {
		rm.Lock()
	}()

	time.Sleep(3 * time.Second)

	rm.Unlock()
}
