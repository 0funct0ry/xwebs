package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKVBuiltins_Integrated(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	kvm := &mockKVManager{store: make(map[string]interface{})}

	d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, nil, nil, kvm, nil, nil, "")

	t.Run("KV Set with Capture Groups", func(t *testing.T) {
		h := &Handler{
			Name: "kv-set-test",
			Match: Matcher{
				Regex: "^set:([^:]+):(.+)$",
			},
			Actions: []Action{
				{
					Type:    "builtin",
					Command: "kv-set",
					Key:     "{{index .Matches 1}}",
					Value:   "{{index .Matches 2}}",
				},
			},
		}
		err := reg.AddHandlers([]Handler{*h})
		require.NoError(t, err)

		msg := &ws.Message{
			Data:     []byte("set:user:bob"),
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		// 1. Match
		matches, err := reg.Match(msg, engine, template.NewContext())
		require.NoError(t, err)
		require.Len(t, matches, 1)
		assert.Equal(t, []string{"set:user:bob", "user", "bob"}, matches[0].Matches)

		// 2. Execute
		err = d.Execute(context.Background(), matches[0].Handler, msg, matches[0].Matches)
		require.NoError(t, err)

		// 3. Verify KV
		val, ok := kvm.GetKV("user")
		assert.True(t, ok)
		assert.Equal(t, "bob", val)
	})

	t.Run("KV Get with Capture Groups", func(t *testing.T) {
		kvm.SetKV("color", "red", 0)

		h := &Handler{
			Name: "kv-get-test",
			Match: Matcher{
				Regex: "^get:([^:]+)$",
			},
			Actions: []Action{
				{
					Type:    "builtin",
					Command: "kv-get",
					Key:     "{{index .Matches 1}}",
					Default: "none",
				},
			},
			Respond: "Result: {{.KvValue}}",
		}
		err := reg.AddHandlers([]Handler{*h})
		require.NoError(t, err)

		// Get existing
		msg := &ws.Message{Data: []byte("get:color"), Metadata: ws.MessageMetadata{Direction: "received"}}
		matches, _ := reg.Match(msg, engine, template.NewContext())
		require.NotEmpty(t, matches, "Should find get:color match")

		err = d.Execute(context.Background(), matches[0].Handler, msg, matches[0].Matches)
		require.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "Result: red", conn.lastWritten)
		conn.mu.Unlock()

		// Get non-existent
		msg = &ws.Message{Data: []byte("get:size"), Metadata: ws.MessageMetadata{Direction: "received"}}
		matches, _ = reg.Match(msg, engine, template.NewContext())
		require.NotEmpty(t, matches, "Should find get:size match")

		err = d.Execute(context.Background(), matches[0].Handler, msg, matches[0].Matches)
		require.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "Result: none", conn.lastWritten)
		conn.mu.Unlock()
	})

	t.Run("KV List", func(t *testing.T) {
		kvm.store = map[string]interface{}{"a": "val1", "b": "val2"}

		h := &Handler{
			Name:    "kv-list-test",
			Match:   Matcher{Pattern: "list"},
			Actions: []Action{{Type: "builtin", Command: "kv-list"}},
			Respond: "{{range .KvKeys}}{{.}} {{end}}",
		}
		err := reg.AddHandlers([]Handler{*h})
		require.NoError(t, err)

		msg := &ws.Message{Data: []byte("list"), Metadata: ws.MessageMetadata{Direction: "received"}}
		matches, _ := reg.Match(msg, engine, template.NewContext())
		require.NotEmpty(t, matches, "Should find list match")

		err = d.Execute(context.Background(), matches[0].Handler, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		assert.Equal(t, "a b ", conn.lastWritten)
		conn.mu.Unlock()
	})
}
