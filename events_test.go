package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type tickEvent struct {
	value bool
}

func TestChanReceiver(t *testing.T) {
	eventManager := NewEventManager()

	listener1 := eventManager.NewChanReceiver(tickEvent{})
	listener2 := eventManager.NewChanReceiver(tickEvent{})

	var listener1Received tickEvent
	var listener2Received tickEvent

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine())

		for event := range listener1 {
			listener1Received = event.(tickEvent)
		}
	}()

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine())

		for event := range listener2 {
			listener2Received = event.(tickEvent)
		}
	}()

	eventManager.Emit(tickEvent{true})

	time.Sleep(time.Millisecond * 100)

	// check that listeners are modified by Emit

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, true, listener2Received.value)

	// remove only listener1, listener2 should still be notified

	eventManager.DestroyChanReceiver(listener1)

	eventManager.Emit(tickEvent{false})

	time.Sleep(time.Millisecond * 100)

	// listener1 must not be notified, listener2 still be notified

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, false, listener2Received.value)
}

func TestFuncReceiver(t *testing.T) {
	eventManager := NewEventManager()

	var listener1Received tickEvent
	var listener2Received tickEvent

	ef1 := eventManager.NewFuncReceiver(tickEvent{}, func(e Event) {
		listener1Received = e.(tickEvent)
	})
	eventManager.NewFuncReceiver(tickEvent{}, func(e Event) {
		listener2Received = e.(tickEvent)
	})

	eventManager.Emit(tickEvent{true})

	// check that listeners are modified by Emit

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, true, listener2Received.value)

	// remove only listener1, listener2 should still be notified

	eventManager.DestroyFuncReceiver(ef1)

	eventManager.Emit(tickEvent{false})

	// listener1 must not be notified, listener2 still be notified

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, false, listener2Received.value)
}
