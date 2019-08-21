package common

var events = make(map[string][]chan interface{})

// ListEvents list all current registered

// AddListener adds an event listener to the Dog struct instance
func AddListener(name string, ch chan interface{}) {
	if _, ok := events[name]; ok {
		events[name] = append(events[name], ch)
	} else {
		events[name] = []chan interface{}{ch}
	}
}

// RemoveListener removes an event listener from the Dog struct instance
func RemoveListener(name string, ch chan interface{}) {
	if _, ok := events[name]; ok {
		for i := range events[name] {
			if events[name][i] == ch {
				events[name] = append(events[name][:i], events[name][i+1:]...)
				break
			}
		}
	}
}

// EmitEvent emits an event on the Dog struct instance
func EmitEvent(name string, event interface{}) {
	if _, ok := events[name]; ok {
		for _, handler := range events[name] {
			go func(handler chan interface{}) {
				handler <- event
			}(handler)
		}
	}
}
