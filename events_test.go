package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	tick = iota
)

func TestEvent(t *testing.T) {
	event := NewEvent()

	listener1 := event.AddListener(tick)
	listener2 := event.AddListener(tick)

	var listener1Received bool
	var listener2Received bool

	go func() {
		for {
			ev := <-listener1
			listener1Received = ev.(bool)
		}
	}()

	go func() {
		for {
			ev := <-listener2
			listener2Received = ev.(bool)
		}
	}()

	event.EmitEvent(tick, true)

	time.Sleep(time.Millisecond * 100)

	// check that listeners are modified by EmitEvent

	assert.Equal(t, true, listener1Received)
	assert.Equal(t, true, listener2Received)

	// remove only listener1, listener2 should still be notified

	event.RemoveListener(tick, listener1)

	event.EmitEvent(tick, false)

	time.Sleep(time.Millisecond * 100)

	// listener1 must not be notified, listener2 still be notified

	assert.Equal(t, true, listener1Received)
	assert.Equal(t, false, listener2Received)
}
