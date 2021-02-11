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
		if Error(err) {
			return err
		}
	}

	return nil
}
