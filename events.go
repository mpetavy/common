package common

import "reflect"

type EventListener chan interface{}

type EventType reflect.Type

type EventManager struct {
	listeners map[EventType][]EventListener
}

var (
	Events *EventManager
)

func init() {
	Events = NewEventManager()
}

func NewEventManager() *EventManager {
	return &EventManager{listeners: make(map[EventType][]EventListener)}
}

// AddListener adds an event listener to the Dog struct instance
func (this *EventManager) AddListener(event interface{}) EventListener {
	eventType := reflect.TypeOf(event)
	eventListener := make(EventListener)

	if _, ok := this.listeners[eventType]; ok {
		this.listeners[eventType] = append(this.listeners[eventType], eventListener)
	} else {
		this.listeners[eventType] = []EventListener{eventListener}
	}

	return eventListener
}

// RemoveListener removes an event listener from the Dog struct instance
func (this *EventManager) RemoveListener(event interface{}, eventListener EventListener) {
	eventType := reflect.TypeOf(event)

	if _, ok := this.listeners[eventType]; ok {
		for i := range this.listeners[eventType] {
			if this.listeners[eventType][i] == eventListener {
				this.listeners[eventType] = append(this.listeners[eventType][:i], this.listeners[eventType][i+1:]...)
				break
			}
		}
	}
}

// Emit emits an event on the Dog struct instance
func (this *EventManager) Emit(event interface{}) {
	eventType := reflect.TypeOf(event)

	if listeners, ok := this.listeners[eventType]; ok {
		for _, listener := range listeners {
			listener <- event
		}
	}
}
