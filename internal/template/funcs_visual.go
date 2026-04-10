package template

import (
	"math/rand"
	"sync/atomic"
	"time"
)

var (
	emojis = []string{"🚀", "✨", "🔥", "🛠️", "💡", "📡", "🔌", "🛡️", "⚙️", "🧪", "⚡", "🌈", "💎", "🎯", "🔔"}
	colors = []string{"red", "green", "yellow", "blue", "magenta", "cyan", "white", "grey"}
)

func (e *Engine) registerVisualFuncs() {
	e.funcs["reqCounter"] = func() uint64 {
		return e.incrementCounter("req")
	}

	e.funcs["msgCounter"] = func() uint64 {
		return e.incrementCounter("msg")
	}

	e.funcs["errorCount"] = func() uint64 {
		return e.incrementCounter("error")
	}

	e.funcs["randomEmoji"] = func() string {
		return emojis[rand.Intn(len(emojis))]
	}

	e.funcs["randomColor"] = func() string {
		return colors[rand.Intn(len(colors))]
	}

	e.funcs["sessionAge"] = func() time.Duration {
		return time.Since(e.sessionStartTime)
	}
}

func (e *Engine) incrementCounter(name string) uint64 {
	e.countersMu.Lock()
	ptr, ok := e.counters[name]
	if !ok {
		var val uint64
		ptr = &val
		e.counters[name] = ptr
	}
	e.countersMu.Unlock()

	return atomic.AddUint64(ptr, 1)
}
