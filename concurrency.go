package common

import (
	"fmt"
	"time"
)

type TimeoutError struct {
	maxDuration time.Duration
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("Timeout after: %+v", time.Duration(e.maxDuration))
}

func NewTimeout(maxDuration time.Duration, checkDuration time.Duration, fn func() (bool, error)) error {
	start := time.Now()
	loop := true

	ti := time.NewTicker(checkDuration)
	defer ti.Stop()

	for loop {
		<-ti.C

		if time.Since(start) > maxDuration {
			return &TimeoutError{
				maxDuration: maxDuration,
			}
		}

		var err error

		loop, err = fn()
		if err != nil {
			return err
		}
	}

	return nil
}
