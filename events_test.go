package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type tickEvent struct {
	value bool
}

func TestEvent(t *testing.T) {
	event := NewEventManager()

	listener1 := event.CreateListener(tickEvent{})
	listener2 := event.CreateListener(tickEvent{})

	var listener1Received tickEvent
	var listener2Received tickEvent

	go func() {
		for {
			ev := <-listener1
			listener1Received = ev.(tickEvent)
		}
	}()

	go func() {
		for {
			ev := <-listener2
			listener2Received = ev.(tickEvent)
		}
	}()

	event.Emit(tickEvent{true})

	time.Sleep(time.Millisecond * 100)

	// check that listeners are modified by Emit

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, true, listener2Received.value)

	// remove only listener1, listener2 should still be notified

	event.DestroyListener(listener1)

	event.Emit(tickEvent{false})

	time.Sleep(time.Millisecond * 100)

	// listener1 must not be notified, listener2 still be notified

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, false, listener2Received.value)
}
