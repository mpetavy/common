package common

import (
	"reflect"
	"sync"
)

type Event interface{}
type EventChan chan Event
type EventFunc func(Event)
type EventType reflect.Type
type EventManager struct {
	mu    sync.Mutex
	chans map[EventType][]EventChan
	funcs map[EventType][]*EventFunc
}

var (
	Events *EventManager
)

func init() {
	Events = NewEventManager()
}

func NewEventManager() *EventManager {
	return &EventManager{
		chans: make(map[EventType][]EventChan),
		funcs: make(map[EventType][]*EventFunc),
	}
}

// CreateChanReceiver adds an event listener to the Dog struct instance
func (this *EventManager) CreateChanReceiver(event interface{}) EventChan {
	this.mu.Lock()
	defer this.mu.Unlock()

	eventType := reflect.TypeOf(event)
	eventListener := make(EventChan)

	if _, ok := this.chans[eventType]; ok {
		this.chans[eventType] = append(this.chans[eventType], eventListener)
	} else {
		this.chans[eventType] = []EventChan{eventListener}
	}

	return eventListener
}

// CreateFuncReceiver adds an event listener to the Dog struct instance
func (this *EventManager) CreateFuncReceiver(event interface{}, eventFunc EventFunc) {
	this.mu.Lock()
	defer this.mu.Unlock()

	eventType := reflect.TypeOf(event)

	if _, ok := this.funcs[eventType]; ok {
		this.funcs[eventType] = append(this.funcs[eventType], &eventFunc)
	} else {
		this.funcs[eventType] = []*EventFunc{&eventFunc}
	}
}

// DestroyFuncReceiver removes an event listener from the Dog struct instance
func (this *EventManager) DestroyChanReceiver(eventChans EventChan) {
	this.mu.Lock()
	defer this.mu.Unlock()

	for _, chans := range this.chans {
		for i := range chans {
			if chans[i] == eventChans {
				close(chans[i])
				chans = append(chans[:i], chans[i+1:]...)
				break
			}
		}
	}
}

// DestroyFuncReceiver removes an event listener from the Dog struct instance
func (this *EventManager) DestroyFuncReceiver(eventChans EventChan) {
	this.mu.Lock()
	defer this.mu.Unlock()

	for _, chans := range this.chans {
		for i := range chans {
			if chans[i] == eventChans {
				chans = append(chans[:i], chans[i+1:]...)
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

	if chans, ok := this.chans[eventType]; ok {
		for _, receiver := range chans {
			receiver <- event
		}
	}

	if funcs, ok := this.funcs[eventType]; ok {
		for _, receiver := range funcs {
			(*receiver)(event)
		}
	}
}
