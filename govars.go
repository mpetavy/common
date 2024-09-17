package common

import (
	"slices"
	"sync"
	"time"
)

type goVars struct {
	sync.RWMutex
	register map[uint64]map[string]any
	lastTime time.Time
}

var (
	GoVars = &goVars{
		register: make(map[uint64]map[string]any),
	}
)

func (g *goVars) Set(name string, value any) {
	g.Lock()
	defer func() {
		g.Unlock()
	}()

	id := GoRoutineId()

	values, ok := g.register[id]

	if !ok {
		values = make(map[string]any)
	}

	values[name] = value
	g.register[id] = values
}

func (g *goVars) GetById(id uint64) map[string]any {
	g.RLock()
	defer func() {
		g.RUnlock()
	}()

	if g.lastTime.IsZero() || g.lastTime.Before(time.Now()) {
		g.lastTime = time.Now()

		ids := GoRoutineIds()

		for id := range g.register {
			if !slices.Contains(ids, id) {
				delete(g.register, id)
			}
		}
	}

	values, ok := g.register[id]

	if !ok {
		return nil
	}

	return values
}

func (g *goVars) Get(id uint64) map[string]any {
	return g.GetById(GoRoutineId())
}
