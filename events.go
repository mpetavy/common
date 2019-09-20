package common

import (
	"reflect"
	"sync"
)

type EventListener chan interface{}

type EventType reflect.Type

type EventManager struct {
	mu            sync.Mutex
	listenerTypes map[EventType][]EventListener
}

var (
	Events *EventManager
)

func init() {
	Events = NewEventManager()
}

func NewEventManager() *EventManager {
	return &EventManager{listenerTypes: make(map[EventType][]EventListener)}
}

// CreateListener adds an event listener to the Dog struct instance
func (this *EventManager) CreateListener(event interface{}) EventListener {
	this.mu.Lock()
	defer this.mu.Unlock()

	eventType := reflect.TypeOf(event)
	eventListener := make(EventListener)

	if _, ok := this.listenerTypes[eventType]; ok {
		this.listenerTypes[eventType] = append(this.listenerTypes[eventType], eventListener)
	} else {
		this.listenerTypes[eventType] = []EventListener{eventListener}
	}

	return eventListener
}

// DestroyListener removes an event listener from the Dog struct instance
func (this *EventManager) DestroyListener(eventListener EventListener) {
	this.mu.Lock()
	defer this.mu.Unlock()

	for _, listenerType := range this.listenerTypes {
		for i := range listenerType {
			if listenerType[i] == eventListener {
				listenerType = append(listenerType[:i], listenerType[i+1:]...)
				break
			}
		}
	}
}

// Emit emits an event on the Dog struct instance
func (this *EventManager) Emit(event interface{}) {
	this.mu.Lock()
	defer this.mu.Unlock()

	eventType := reflect.TypeOf(event)

	if listenerType, ok := this.listenerTypes[eventType]; ok {
		for _, listener := range listenerType {
			listener <- event
		}
	}
}
