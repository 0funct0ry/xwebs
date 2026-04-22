package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoBuiltin_Behavior(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil)

	t.Run("Verbatim Echo", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		h := &Handler{
			Name:    "echo-verbatim",
			Builtin: "echo",
		}
		msg := &ws.Message{
			Data:     []byte("verbatim message"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err := d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, "verbatim message", conn.lastWritten)
		assert.Len(t, conn.messages, 1)
		assert.Equal(t, ws.TextMessage, conn.messages[0].Type)
	})

	t.Run("Respond Override", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		h := &Handler{
			Name:    "echo-respond",
			Builtin: "echo",
			Respond: "Overridden: {{.Message}}",
		}
		msg := &ws.Message{
			Data:     []byte("foo"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err := d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, "Overridden: foo", conn.lastWritten)
		assert.Len(t, conn.messages, 1)
	})

	t.Run("Delay", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		h := &Handler{
			Name:    "echo-delay",
			Builtin: "echo",
			Delay:   "100ms",
		}
		msg := &ws.Message{
			Data:     []byte("delayed"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		start := time.Now()
		err := d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)
		duration := time.Since(start)

		assert.GreaterOrEqual(t, duration, 100*time.Millisecond)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, "delayed", conn.lastWritten)
	})

	t.Run("Binary Echo", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		h := &Handler{
			Name:    "echo-binary",
			Builtin: "echo",
		}
		binaryData := []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}
		msg := &ws.Message{
			Data:     binaryData,
			Type:     ws.BinaryMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err := d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, binaryData, []byte(conn.lastWritten))
		assert.Len(t, conn.messages, 1)
		assert.Equal(t, ws.BinaryMessage, conn.messages[0].Type)
	})
}
