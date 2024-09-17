package common

import (
	"slices"
	"sync"
	"time"
)

type goVars struct {
	sync.RWMutex
	register map[uint64]map[string]any
}

var (
	GoVars *goVars
)

func init() {
	GoVars = &goVars{
		register: make(map[uint64]map[string]any),
	}

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)

			GoVars.garbadge()
		}
	}()
}

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

	values, ok := g.register[id]

	if !ok {
		return nil
	}

	return values
}

func (g *goVars) Get(id uint64) map[string]any {
	return g.GetById(GoRoutineId())
}

func (g *goVars) garbadge() {
	g.Lock()
	defer func() {
		g.Unlock()
	}()

	ids := GoRoutineIds()

	for k := range g.register {
		if !slices.Contains(ids, k) {
			delete(g.register, k)
		}
	}
}
