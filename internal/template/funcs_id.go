package template

import (
	"sync"

	"github.com/google/uuid"
	"github.com/matoous/go-nanoid/v2"
	"github.com/oklog/ulid/v2"
	"github.com/teris-io/shortid"
)

var (
	counterMap = make(map[string]int)
	counterMu  sync.Mutex
)

func (e *Engine) registerIDFuncs() {
	e.funcs["uuid"] = func() string {
		return uuid.New().String()
	}

	e.funcs["ulid"] = func() string {
		return ulid.Make().String()
	}

	e.funcs["nanoid"] = func() (string, error) {
		return gonanoid.New()
	}

	e.funcs["shortid"] = func() (string, error) {
		return shortid.Generate()
	}

	e.funcs["counter"] = func(name string) int {
		counterMu.Lock()
		defer counterMu.Unlock()
		counterMap[name]++
		return counterMap[name]
	}
}
