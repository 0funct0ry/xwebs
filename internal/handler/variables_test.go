package handler

import (
	"context"
	"os"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerVariables(t *testing.T) {
	// Set an environment variable for testing
	os.Setenv("XWEBS_TEST_ENV", "env-value")
	defer os.Unsetenv("XWEBS_TEST_ENV")

	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}

	globalVars := map[string]interface{}{
		"app":      "xwebs",
		"version":  "v1",
		"fullname": "{{.Vars.app}}-{{.Vars.version}}",
		"env_val":  "{{.Env.XWEBS_TEST_ENV}}",
	}

	sessionVars := map[string]interface{}{
		"user": "alice",
	}

	d := NewDispatcher(reg, conn, engine, true, globalVars, sessionVars)

	h := &Handler{
		Name: "var-test",
		Variables: map[string]interface{}{
			"local": "local-val",
			"msg":   "{{.Vars.fullname}}-{{.Vars.local}}-{{.Session.user}}",
		},
		Match: Matcher{
			Type:     "template",
			Template: `{{eq .Vars.local "local-val"}}`,
		},
		Run: "echo {{.Vars.msg}}",
	}
	reg.AddHandlers([]Handler{*h})

	ctx := context.Background()
	msg := &ws.Message{
		Data: []byte("trigger"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
		},
	}

	// 1. Test Matching with variables
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)
	matches, err := reg.Match(msg, engine, tmplCtx)
	require.NoError(t, err)
	assert.Len(t, matches, 1, "Handler should match because it has access to its own variables")

	// 2. Test Execution with variables
	err = d.Execute(ctx, matches[0], msg)
	require.NoError(t, err)

	// Verify the output (which is stored in tmplCtx.Stdout by ExecuteAction/executeShell)
	// Wait, Execute creates its own tmplCtx. We need to check what was "sent" or "logged" or "run".
	// Our mockConn doesn't capture ExecuteAction results directly unless it's a "send".
	
	// Let's use a "send" action instead of "run" for easier verification
	h.Actions = []Action{{Type: "send", Message: "Result: {{.Vars.msg}} - Env: {{.Vars.env_val}}"}}
	h.Run = ""
	reg.handlers[0] = *h

	err = d.Execute(ctx, &reg.handlers[0], msg)
	require.NoError(t, err)

	conn.mu.Lock()
	lastSent := conn.lastWritten
	conn.mu.Unlock()
	assert.Equal(t, "Result: xwebs-v1-local-val-alice - Env: env-value", lastSent)
}

func TestHandlerVariableOverride(t *testing.T) {
	reg := NewRegistry()
	engine := template.New(false)
	conn := &mockConn{}

	globalVars := map[string]interface{}{
		"key": "global",
	}

	d := NewDispatcher(reg, conn, engine, true, globalVars, nil)

	h := &Handler{
		Name: "override-test",
		Variables: map[string]interface{}{
			"key": "local",
		},
		Match:   Matcher{Pattern: "*"},
		Respond: "Value: {{.Vars.key}}",
	}
	reg.AddHandlers([]Handler{*h})

	ctx := context.Background()
	msg := &ws.Message{
		Data: []byte("hello"),
		Metadata: ws.MessageMetadata{Direction: "received"},
	}

	err := d.Execute(ctx, &reg.handlers[0], msg)
	require.NoError(t, err)

	conn.mu.Lock()
	lastSent := conn.lastWritten
	conn.mu.Unlock()
	assert.Equal(t, "Value: local", lastSent)
}
