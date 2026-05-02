package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestABTestBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, "")

	// Register two mock handlers
	handlerA := Handler{Name: "a", Match: Matcher{Pattern: "a"}, Respond: "Response A"}
	handlerB := Handler{Name: "b", Match: Matcher{Pattern: "b"}, Respond: "Response B"}
	require.NoError(t, reg.Add(handlerA))
	require.NoError(t, reg.Add(handlerB))

	tests := []struct {
		name     string
		field    string
		split    int
		input    string
		expected string
	}{
		{
			name:     "route to A (split 100)",
			field:    ".id",
			split:    100,
			input:    `{"id": "1"}`,
			expected: "Response A",
		},
		{
			name:     "route to B (split 0)",
			field:    ".id",
			split:    0,
			input:    `{"id": "1"}`,
			expected: "Response B",
		},
		{
			name:  "deterministic routing - value 1",
			field: ".user_id",
			split: 50,
			input: `{"user_id": "user-123"}`,
			// We'll check consistency by running multiple times
			expected: "", // Will be determined in first run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			split := tt.split
			action := &Action{
				Type:     "builtin",
				Command:  "ab-test",
				Field:    tt.field,
				Split:    &split,
				HandlerA: "a",
				HandlerB: "b",
			}

			tmplCtx := template.NewContext()
			tmplCtx.MessageBytes = []byte(tt.input)
			tmplCtx.Session = make(map[string]interface{})

			conn.mu.Lock()
			conn.messages = nil
			conn.lastWritten = ""
			conn.mu.Unlock()

			err := d.ExecuteAction(context.Background(), action, tmplCtx, nil)
			require.NoError(t, err)

			conn.mu.Lock()
			lastWritten := conn.lastWritten
			conn.mu.Unlock()

			if tt.expected != "" {
				assert.Equal(t, tt.expected, lastWritten)
			} else {
				// Deterministic check: run again and ensure same result
				firstResult := lastWritten
				require.NotEmpty(t, firstResult)

				conn.mu.Lock()
				conn.messages = nil
				conn.lastWritten = ""
				conn.mu.Unlock()

				err := d.ExecuteAction(context.Background(), action, tmplCtx, nil)
				require.NoError(t, err)

				conn.mu.Lock()
				finalWritten := conn.lastWritten
				conn.mu.Unlock()

				assert.Equal(t, firstResult, finalWritten, "Routing should be deterministic")
			}
		})
	}
}

func TestABTestBuiltin_NonExistentHandler(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, "")

	split := 50
	action := &Action{
		Type:     "builtin",
		Command:  "ab-test",
		Field:    ".id",
		Split:    &split,
		HandlerA: "non-existent",
		HandlerB: "b",
	}

	tmplCtx := template.NewContext()
	tmplCtx.MessageBytes = []byte(`{"id": "1"}`)

	err := d.ExecuteAction(context.Background(), action, tmplCtx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chosen handler \"non-existent\" not found")
}

func TestABTestBuiltin_InvalidJQ(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, "")

	split := 50
	action := &Action{
		Type:     "builtin",
		Command:  "ab-test",
		Field:    "!!! invalid !!!",
		Split:    &split,
		HandlerA: "a",
		HandlerB: "b",
	}

	tmplCtx := template.NewContext()
	tmplCtx.MessageBytes = []byte(`{"id": "1"}`)

	err := d.ExecuteAction(context.Background(), action, tmplCtx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid jq expression")
}
