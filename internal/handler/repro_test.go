package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReproduction_NestedVariables(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}

	globalVars := map[string]interface{}{
		"app":     "xwebs",
		"version": "v1.0",
	}

	d := NewDispatcher(reg, conn, engine, true, globalVars, nil)

	h := &Handler{
		Name: "version_check",
		Variables: map[string]interface{}{
			"local_suffix": "dev",
			"fullname":     "{{.Vars.app}}-{{.Vars.version}}-{{.Vars.local_suffix}}",
		},
		Match: Matcher{
			Pattern: "*ping*",
			// Type: "glob", // Missing in user's config
		},
		Respond: "Running {{.Vars.fullname}}",
	}
	reg.AddHandlers([]Handler{*h})

	ctx := context.Background()
	msg := &ws.Message{
		Data: []byte("ping"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
		},
	}

	// 1. Check matching (should now work even without explicit Type: glob)
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)
	matches, err := reg.Match(msg, engine, tmplCtx)
	require.NoError(t, err)
	assert.Len(t, matches, 1, "Should now match even without Type: glob because of wildcards")

	// 2. Test if it works with Type: glob
	h.Match.Type = "glob"
	reg.handlers[0] = *h
	matches, err = reg.Match(msg, engine, tmplCtx)
	require.NoError(t, err)
	assert.Len(t, matches, 1, "Should match with Type: glob")

	// 3. Check execution
	err = d.Execute(ctx, matches[0], msg)
	require.NoError(t, err)

	conn.mu.Lock()
	lastSent := conn.lastWritten
	conn.mu.Unlock()
	assert.Equal(t, "Running xwebs-v1.0-dev", lastSent)
}
