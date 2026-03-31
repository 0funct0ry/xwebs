package template

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// Engine is a wrapper around text/template that provides standard functions.
type Engine struct {
	funcs template.FuncMap
}

// New creates a new template engine with the standard functions registered.
func New() *Engine {
	e := &Engine{
		funcs: make(template.FuncMap),
	}
	e.registerDefaults()
	return e
}

// registerDefaults adds the standard functions to the engine's function map.
func (e *Engine) registerDefaults() {
	e.funcs["now"] = func() time.Time {
		return time.Now()
	}
}

// Execute renders the template string with the provided data.
func (e *Engine) Execute(name, text string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Funcs(e.funcs).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}

	return buf.String(), nil
}
