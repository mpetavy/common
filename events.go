package common

import (
	"fmt"
	"reflect"
)

type EventInfo reflect.Type
type EventListener chan interface{}
type EventType int
type Event struct {
	infoType  EventInfo
	listeners map[EventType][]EventListener
}

func NewEvent(eventInfo interface{}) *Event {
	return &Event{infoType: reflect.TypeOf(eventInfo), listeners: make(map[EventType][]EventListener)}
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
func (this *Event) EmitEvent(eventType EventType, infoType interface{}) {
	if this.infoType != reflect.TypeOf(infoType) {
		panic(fmt.Errorf("event expects typeof %v but tried to emit is typeof %v", this.infoType, reflect.TypeOf(infoType)))
	}

	if listeners, ok := this.listeners[eventType]; ok {
		for _, listener := range listeners {
			listener <- infoType
		}
	}
}
