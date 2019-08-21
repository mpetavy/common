package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	listener1 := make(chan interface{})
	listener2 := make(chan interface{})

	AddListener("listener", listener1)
	AddListener("listener", listener2)

	ticker := time.NewTicker(time.Second)
	timer := time.NewTimer(time.Second * 5)
	i := 0

	resultListener1 := ""
	resultListener2 := ""
loop:
	for {
		select {
		case ev := <-listener1:
			resultListener1 = resultListener1 + ev.(string)
			fmt.Printf("listener111: %v\n", ev)
		case ev := <-listener2:
			resultListener2 = resultListener2 + ev.(string)
			fmt.Printf("listener222: %v\n", ev)
		case <-ticker.C:
			if i%2 == 0 {
				EmitEvent("listener", "0")
			} else {
				EmitEvent("listener", "1")
			}
			i++
		case <-timer.C:
			break loop
		}
	}

	assert.Equal(t, "0101", resultListener1)
	assert.Equal(t, "0101", resultListener2)
}
