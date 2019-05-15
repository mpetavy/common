package common

type EventType string

type EventHandler struct {
	Listeners map[EventType][]chan interface{}
}

func NewEventHandler() *EventHandler {
	return &EventHandler{make(map[EventType][]chan interface{})}
}

func (eh *EventHandler) AddListener(name EventType, ch chan interface{}) {
	if eh.Listeners == nil {
		eh.Listeners = make(map[EventType][]chan interface{})
	}
	if _, ok := eh.Listeners[name]; ok {
		eh.Listeners[name] = append(eh.Listeners[name], ch)
	} else {
		eh.Listeners[name] = []chan interface{}{ch}
	}
}

func (eh *EventHandler) RemoveListener(name EventType, ch chan interface{}) {
	if _, ok := eh.Listeners[name]; ok {
		for i := range eh.Listeners[name] {
			if eh.Listeners[name][i] == ch {
				eh.Listeners[name] = append(eh.Listeners[name][:i], eh.Listeners[name][i+1:]...)
				break
			}
		}
	}
}

func (eh *EventHandler) RemoveListenesr(name EventType) {
	delete(eh.Listeners, name)
}

func (eh *EventHandler) Emit(name EventType, response string) {
	if _, ok := eh.Listeners[name]; ok {
		for _, handler := range eh.Listeners[name] {
			go func(handler chan interface{}) {
				handler <- response
			}(handler)
		}
	}
}
