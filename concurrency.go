package common

import (
	"fmt"
	"time"
)

type TimeoutError error

func NewTimeoutError(maxDuration time.Duration) TimeoutError {
	return TimeoutError(fmt.Errorf("Timeout error after: %+v", time.Duration(maxDuration)))
}

func NewTimeoutOperation(maxDuration time.Duration, checkDuration time.Duration, fn func() (bool, error)) error {
	start := time.Now()
	loop := true

	ti := time.NewTicker(checkDuration)
	defer ti.Stop()

	for loop {
		<-ti.C

		if time.Since(start) > maxDuration {
			return NewTimeoutError(maxDuration)
		}

		var err error

		loop, err = fn()
		if err != nil {
			return err
		}
	}

	return nil
}

type limitedGo struct {
	ch chan bool
}

func NewLimitedGo(amount int) limitedGo {
	return limitedGo{ch: make(chan bool, amount)}
}

func (this *limitedGo) Go(fn func()) {
	//this.ch <- true
	//defer func() {
	//	<-this.ch
	//}()

	Warn("no GO routine!!")

	fn()
}

func (this *limitedGo) Wait() {
	close(this.ch)

	for {
		_, ok := <-this.ch

		if !ok {
			break
		}

	}
}
