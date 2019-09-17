package common

import (
	"reflect"
)

type EventInfo reflect.Type
type EventListener chan interface{}
type EventType int
type Event struct {
	listeners map[EventType][]EventListener
}

func NewEvent() *Event {
	return &Event{listeners: make(map[EventType][]EventListener)}
}

// AddListener adds an event listener to the Dog struct instance
func (this *Event) AddListener(eventType EventType) EventListener {
	eventListener := make(EventListener)

	if _, ok := this.listeners[eventType]; ok {
		this.listeners[eventType] = append(this.listeners[eventType], eventListener)
	} else {
		this.listeners[eventType] = []EventListener{eventListener}
	}

	return eventListener
}

// RemoveListener removes an event listener from the Dog struct instance
func (this *Event) RemoveListener(eventType EventType, eventListener EventListener) {
	if _, ok := this.listeners[eventType]; ok {
		for i := range this.listeners[eventType] {
			if this.listeners[eventType][i] == eventListener {
				this.listeners[eventType] = append(this.listeners[eventType][:i], this.listeners[eventType][i+1:]...)
				break
			}
		}
	}
}

// EmitEvent emits an event on the Dog struct instance
func (this *Event) EmitEvent(eventType EventType, eventInfo interface{}) {
	if listeners, ok := this.listeners[eventType]; ok {
		for _, listener := range listeners {
			listener <- eventInfo
		}
	}
}
