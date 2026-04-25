package handler

import (
	"context"
	"os"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLuaBuiltin_Execute(t *testing.T) {
	// Setup dispatcher with mock connection
	conn := &mockConn{}

	registry := NewRegistry(ServerMode)
	engine := template.New(false)

	d := &Dispatcher{
		conn:           conn,
		registry:       registry,
		templateEngine: engine,
	}

	lua := &LuaBuiltin{}

	t.Run("Inline script return string", func(t *testing.T) {
		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "test",
			Script:      "return 'hello from lua'",
		}
		tmplCtx := template.NewContext()
		tmplCtx.Message = "input"

		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		require.Len(t, conn.messages, 1)
		assert.Equal(t, "hello from lua", string(conn.messages[0].Data))
		conn.mu.Unlock()
	})

	t.Run("Inline script return false (drop)", func(t *testing.T) {
		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "test",
			Script:      "return false",
		}
		tmplCtx := template.NewContext()

		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.ErrorIs(t, err, ErrDrop)

		conn.mu.Lock()
		assert.Empty(t, conn.messages)
		conn.mu.Unlock()
	})

	t.Run("State persistence", func(t *testing.T) {
		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "stateful",
			Script:      "state.count = (state.count or 0) + 1; return tostring(state.count)",
		}
		tmplCtx := template.NewContext()

		// Call 1
		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "1", string(conn.messages[0].Data))
		conn.mu.Unlock()

		// Call 2
		err = lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "2", string(conn.messages[1].Data))
		conn.mu.Unlock()
	})

	t.Run("Access globals", func(t *testing.T) {
		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "globals",
			Script:      "return message .. ' ' .. connection_id",
		}
		tmplCtx := template.NewContext()
		tmplCtx.Message = "hello"
		tmplCtx.ConnectionID = "conn123"

		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "hello conn123", string(conn.messages[0].Data))
		conn.mu.Unlock()
	})

	t.Run("JSON module", func(t *testing.T) {
		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "json",
			Script:      `local data = json.decode(message); data.foo = "bar"; return json.encode(data)`,
		}
		tmplCtx := template.NewContext()
		tmplCtx.Message = `{"val": 123}`

		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Contains(t, string(conn.messages[0].Data), `"foo":"bar"`)
		assert.Contains(t, string(conn.messages[0].Data), `"val":123`)
		conn.mu.Unlock()
	})

	t.Run("Regex module", func(t *testing.T) {
		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "regex",
			Script:      `if re.match("^foo", message) then return "matched" else return "no match" end`,
		}
		tmplCtx := template.NewContext()

		tmplCtx.Message = "foobar"
		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "matched", string(conn.messages[0].Data))
		conn.mu.Unlock()

		tmplCtx.Message = "barfoo"
		err = lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "no match", string(conn.messages[1].Data))
		conn.mu.Unlock()
	})

	t.Run("Timeout", func(t *testing.T) {
		a := &Action{
			HandlerName: "slow",
			Script:      `while true do end`,
			Timeout:     "100ms",
		}
		tmplCtx := template.NewContext()

		err := lua.Execute(context.Background(), d, a, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("File script", func(t *testing.T) {
		content := "return 'hello from file'"
		tmpFile, err := os.CreateTemp("", "test*.lua")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		tmpFile.Close()

		conn.mu.Lock()
		conn.messages = nil
		conn.mu.Unlock()

		a := &Action{
			HandlerName: "file-test",
			File:        tmpFile.Name(),
		}
		tmplCtx := template.NewContext()

		err = lua.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "hello from file", string(conn.messages[0].Data))
		conn.mu.Unlock()
	})
}
