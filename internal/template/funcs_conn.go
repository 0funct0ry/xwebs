package template

import (
	"fmt"
	"time"
)

func (e *Engine) registerConnFuncs() {
	// These are placeholder functions for the documentation/metadata.
	// The actual implementation is injected dynamically in Execute()
	// to ensure thread-safety and context-awareness.
	
	e.funcs["connID"] = func() string { return "🔌" }
	e.funcs["shortConnID"] = func() string { return "🔌" }
	e.funcs["sessionID"] = func() string { return "" }
	e.funcs["clientIP"] = func() string { return "❓" }
	e.funcs["remoteAddr"] = func() string { return "❓" }
	e.funcs["localAddr"] = func() string { return "❓" }
	e.funcs["subprotocol"] = func() string { return "" }
	e.funcs["connectedSince"] = func() time.Time { return time.Time{} }
	e.funcs["uptime"] = func() time.Duration { return 0 }
	e.funcs["messageCount"] = func() uint64 { return 0 }
}

// FormatUptime returns a concise uptime string.
func FormatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
