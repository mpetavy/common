package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

type boolEvent struct {
	value bool
}

func TestEventsFuncReceiver(t *testing.T) {
	eventManager := NewEventManager()

	var listener1Received boolEvent
	var listener2Received boolEvent

	ef1 := eventManager.AddListener(boolEvent{}, func(e Event) {
		listener1Received = e.(boolEvent)
	})
	eventManager.AddListener(boolEvent{}, func(e Event) {
		listener2Received = e.(boolEvent)
	})

	eventManager.Emit(boolEvent{true}, false)

	// check that listeners are modified by Emit

	require.Equal(t, true, listener1Received.value)
	require.Equal(t, true, listener2Received.value)

	// remove only listener1, listener2 should still be notified

	eventManager.RemoveListener(ef1)

	eventManager.Emit(boolEvent{false}, false)

	// listener1 must not be notified, listener2 still be notified

	require.Equal(t, true, listener1Received.value)
	require.Equal(t, false, listener2Received.value)
}

type intEvent struct {
	value int
}

func TestEventsPreventRepeating(t *testing.T) {
	eventManager := NewEventManager()

	eventManager.AddListener(&intEvent{}, func(e Event) {
		ie := e.(*intEvent)
		ie.value++

		eventManager.Emit(e, false)
	})

	e := &intEvent{0}

	eventManager.Emit(e, false)

	require.Equal(t, e.value, 1)
}
