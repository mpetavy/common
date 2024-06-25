package common

import (
	"fmt"
	"golang.org/x/exp/slices"
)

type Event interface{}

type EventFunc func(Event)

type EventManager struct {
	listeners map[string][]*EventFunc
}

var (
	Events       = NewEventManager()
	currentEmits []string
)

func NewEventManager() *EventManager {
	return &EventManager{
		listeners: make(map[string][]*EventFunc),
	}
}

func (this *EventManager) AddListener(event interface{}, eventFunc EventFunc) *EventFunc {
	eventType := fmt.Sprintf("%T", event)

	DebugFunc("%T", event)

	if _, ok := this.listeners[eventType]; ok {
		this.listeners[eventType] = append(this.listeners[eventType], &eventFunc)
	} else {
		this.listeners[eventType] = []*EventFunc{&eventFunc}
	}

	return &eventFunc
}

func (this *EventManager) RemoveListener(eventFunc *EventFunc) {
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
	eventType := fmt.Sprintf("%T", event)

	if slices.Contains(currentEmits, eventType) {
		return
	}

	currentEmits = append(currentEmits, eventType)
	defer func() {
		p := slices.Index(currentEmits, eventType)
		currentEmits = slices.Delete(currentEmits, p, p+1)
	}()

	DebugFunc(eventType)

	funcs, ok := this.listeners[eventType]

	if !ok {
		return
	}

	funcs = SliceClone(funcs)

	if reverse {
		funcs = ReverseSlice(funcs)
	}

	for _, receiver := range funcs {
		(*receiver)(event)
	}
}
