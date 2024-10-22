package common

import (
	"slices"
	"time"
)

const (
	goVarslastLogEntry = "LAST_LOG_ENTRY"
)

type goRoutineVars map[uint64]map[string]any

var (
	vars          = make(goRoutineVars)
	GoRoutineVars = NewSyncOf(&vars)
)

func init() {
	t := time.NewTicker(time.Second)
	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		for {
			select {
			case <-t.C:
				GoRoutineVars.Get().cleanup()
			}
		}
	}()
}

func (g *goRoutineVars) cleanup() {
	GoRoutineVars.RunSynchronized(func(g *goRoutineVars) {
		ids := GoRoutineIds()

		for id := range *g {
			if !slices.Contains(ids, id) {
				delete(*g, id)
			}
		}
	})
}

func (g *goRoutineVars) Set(name string, value any) {
	GoRoutineVars.RunSynchronized(func(g *goRoutineVars) {
		id := GoRoutineId()

		values, ok := (*g)[id]

		if !ok {
			values = make(map[string]any)
		}

		values[name] = value
		(*g)[id] = values
	})
}

func (g *goRoutineVars) GetById(id uint64, key string) (value any, ok bool) {
	GoRoutineVars.RunSynchronized(func(g *goRoutineVars) {
		m, found := (*g)[id]

		if !found {
			return
		}

		value, ok = m[key]
	})

	return value, ok
}

func (g *goRoutineVars) Get(key string) (value any, ok bool) {
	return g.GetById(GoRoutineId(), key)
}
