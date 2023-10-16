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

	DebugFunc("%T", event)

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
				funcs = SliceDelete(funcs, i)
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

func (this *EventManager) Emit(event interface{}, reverse bool) {
	this.mu.Lock()

	eventType := reflect.TypeOf(event)

	DebugFunc(eventType)

	funcs, ok := this.listeners[eventType]

	if !ok {
		this.mu.Unlock()

		return
	}

	funcs = SliceClone(funcs)

	this.mu.Unlock()

	if reverse {
		funcs = ReverseSlice(funcs)
	}

	for _, receiver := range funcs {
		(*receiver)(event)
	}
}
