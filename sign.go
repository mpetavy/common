package common

import "sync"

type Sign struct {
	mu sync.Mutex

	isSet bool
}

func (this *Sign) Set() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	if !this.isSet {
		this.isSet = true

		return true
	}

	return false
}

func (this *Sign) Unset() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.isSet {
		this.isSet = false

		return true
	}

	return false
}

func (this *Sign) IsSet() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	return this.isSet
}
