package common

import (
	"reflect"
	"sync"
)

type Event interface{}

type EventFunc func(Event)

type EventType reflect.Type

type EventManager struct {
	mu        sync.Mutex
	listeners map[EventType][]*EventFunc
}

var (
	Events = NewEventManager()
)

func NewEventManager() *EventManager {
	return &EventManager{
		listeners: make(map[EventType][]*EventFunc),
	}
}

func (this *EventManager) AddListener(event interface{}, eventFunc EventFunc) *EventFunc {
	this.mu.Lock()
	defer this.mu.Unlock()

	eventType := reflect.TypeOf(event)

	DebugFunc()

	if _, ok := this.listeners[eventType]; ok {
		this.listeners[eventType] = append(this.listeners[eventType], &eventFunc)
	} else {
		this.listeners[eventType] = []*EventFunc{&eventFunc}
	}

	return &eventFunc
}

func (this *EventManager) RemoveListener(eventFunc *EventFunc) {
	this.mu.Lock()
	defer this.mu.Unlock()

	DebugFunc()

	for eventType, funcs := range this.listeners {
		for i := range funcs {
			if funcs[i] == eventFunc {
				funcs = append(funcs[:i], funcs[i+1:]...)
				break
			}
		}

		if len(funcs) == 0 {
			delete(this.listeners, eventType)
		} else {
			this.listeners[eventType] = funcs
		}
	}
}

func (this *EventManager) Emit(event interface{}, reverse bool) bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	b := false

	eventType := reflect.TypeOf(event)

	DebugFunc(eventType)

	if funcs, ok := this.listeners[eventType]; ok {
		if reverse {
			funcs = ReverseSlice(funcs)
		}

		for _, receiver := range funcs {
			(*receiver)(event)
			b = true
		}
	}

	return b
}
