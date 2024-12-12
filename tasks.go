package common

import (
	"context"
	"fmt"
	"sync"
)

type Tasks struct {
	sync.Mutex
	Wg   sync.WaitGroup
	Ctx  context.Context
	Errs []error
}

func NewTasks(ctx context.Context) *Tasks {
	return &Tasks{
		Ctx: ctx,
	}
}

type TaskFunc func(ctx context.Context) error

func (tasks *Tasks) Add(fn TaskFunc) {
	tasks.Wg.Add(1)

	go func() {
		defer tasks.Wg.Done()

		err := fn(tasks.Ctx)

		if err != nil {
			tasks.Lock()
			defer tasks.Unlock()

			tasks.Errs = append(tasks.Errs, err)
		}
	}()
}

func (tasks *Tasks) Wait() error {
	tasks.Wg.Wait()

	if len(tasks.Errs) > 0 {
		return fmt.Errorf("%d errors occured", len(tasks.Errs))
	}

	return nil
}
