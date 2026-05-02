package handler

import (
	"context"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDispatcher_Sandbox(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}

	t.Run("Sandbox Enabled - Allowed", func(t *testing.T) {
		d := NewDispatcher(reg, conn, engine, false, nil, nil, true, []string{"echo"}, nil, nil, nil, nil, nil, "")
		h := &Handler{
			Name: "test-allowed",
			Run:  "echo 'hello'",
		}

		err := d.Execute(context.Background(), h, &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}, nil)
		assert.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Contains(t, conn.lastWritten, "") // Just ensuring it didn't crash
	})

	t.Run("Sandbox Enabled - Disallowed", func(t *testing.T) {
		d := NewDispatcher(reg, conn, engine, false, nil, nil, true, []string{"echo"}, nil, nil, nil, nil, nil, "")
		h := &Handler{
			Name: "test-disallowed",
			Run:  "ls",
		}

		err := d.Execute(context.Background(), h, &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in the allowlist")
	})

	t.Run("Sandbox Enabled - Empty Allowlist", func(t *testing.T) {
		d := NewDispatcher(reg, conn, engine, false, nil, nil, true, []string{}, nil, nil, nil, nil, nil, "")
		h := &Handler{
			Name: "test-deny-all",
			Run:  "echo 'forbidden'",
		}

		err := d.Execute(context.Background(), h, &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "allowlist is empty")
	})

	t.Run("Sandbox Disabled - Allowed All", func(t *testing.T) {
		d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, "")
		h := &Handler{
			Name: "test-no-sandbox",
			Run:  "ls",
		}

		err := d.Execute(context.Background(), h, &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}, nil)
		assert.NoError(t, err)
	})
}
