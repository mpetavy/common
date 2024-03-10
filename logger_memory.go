package common

import (
	"io"
	"sync"
)

type memoryWriter struct {
	io.Writer

	mu   sync.Mutex
	msgs []string
}

func newMemoryWriter() *memoryWriter {
	return &memoryWriter{}
}

func (mw *memoryWriter) Write(msg []byte) (int, error) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if len(mw.msgs) >= *FlagLogCount {
		mw.msgs = mw.msgs[len(mw.msgs)-*FlagLogCount:]
	}

	mw.msgs = append(mw.msgs, string(msg))

	return len(msg), nil
}

func (mw *memoryWriter) GetLogs() []string {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	return mw.msgs
}

func (mw *memoryWriter) Clearlogs() {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	mw.msgs = nil
}
