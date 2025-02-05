package common

import (
	"fmt"
	"slices"
	"sync"
)

type Event interface{}

type EventFunc func(Event)

type EventManager struct {
	listeners    map[string][]*EventFunc
	currentEmits []string
	sync.Mutex
}

var (
	Events = NewEventManager()
)

func NewEventManager() *EventManager {
	return &EventManager{
		listeners: make(map[string][]*EventFunc),
	}
}

func (eventManager *EventManager) AddListener(event interface{}, eventFunc EventFunc) *EventFunc {
	eventType := fmt.Sprintf("%T", event)

	DebugFunc("%T", event)

	if _, ok := eventManager.listeners[eventType]; ok {
		eventManager.listeners[eventType] = append(eventManager.listeners[eventType], &eventFunc)
	} else {
		eventManager.listeners[eventType] = []*EventFunc{&eventFunc}
	}

	return &eventFunc
}

func (eventManager *EventManager) RemoveListener(eventFunc *EventFunc) {
	DebugFunc()

	for eventType, funcs := range eventManager.listeners {
		for i := range funcs {
			if funcs[i] == eventFunc {
				funcs = SliceDelete(funcs, i)
				break
			}
		}

		if len(funcs) == 0 {
			delete(eventManager.listeners, eventType)
		} else {
			eventManager.listeners[eventType] = funcs
		}
	}
}

func (eventManager *EventManager) registerEventType(eventType string) bool {
	eventManager.Lock()
	defer eventManager.Unlock()

	if slices.Contains(eventManager.currentEmits, eventType) {
		return false
	}

	eventManager.currentEmits = append(eventManager.currentEmits, eventType)

	return true
}

func (eventManager *EventManager) unregisterEventType(eventType string) {
	eventManager.Lock()
	defer eventManager.Unlock()

	p := slices.Index(eventManager.currentEmits, eventType)
	if p != -1 {
		eventManager.currentEmits = slices.Delete(eventManager.currentEmits, p, p+1)
	}
}

func (eventManager *EventManager) Emit(event interface{}, reverse bool) {
	eventType := fmt.Sprintf("%T", event)

	if !eventManager.registerEventType(eventType) {
		return
	}

	defer eventManager.unregisterEventType(eventType)

	funcs, ok := eventManager.listeners[eventType]
	if !ok && len(funcs) == 0 {
		return
	}

	//DebugFunc(eventType)

	funcs = slices.Clone(funcs)

	if reverse {
		funcs = ReverseSlice(funcs)
	}

	for _, receiver := range funcs {
		(*receiver)(event)
	}
}
