package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type tickEvent struct {
	value bool
}

func TestFuncReceiver(t *testing.T) {
	eventManager := NewEventManager()

	var listener1Received tickEvent
	var listener2Received tickEvent

	ef1 := eventManager.AddListener(tickEvent{}, func(e Event) {
		listener1Received = e.(tickEvent)
	})
	eventManager.AddListener(tickEvent{}, func(e Event) {
		listener2Received = e.(tickEvent)
	})

	eventManager.Emit(tickEvent{true}, false)

	// check that listeners are modified by Emit

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, true, listener2Received.value)

	// remove only listener1, listener2 should still be notified

	eventManager.RemoveListener(ef1)

	eventManager.Emit(tickEvent{false}, false)

	// listener1 must not be notified, listener2 still be notified

	assert.Equal(t, true, listener1Received.value)
	assert.Equal(t, false, listener2Received.value)
}
