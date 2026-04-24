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

type mockServerStats struct {
	clients []template.ClientInfo
	msgs    map[string][]*ws.Message
}

func (m *mockServerStats) GetClientCount() int { return len(m.clients) }
func (m *mockServerStats) GetUptime() time.Duration { return 0 }
func (m *mockServerStats) GetClients() []template.ClientInfo { return m.clients }
func (m *mockServerStats) IsPaused() bool { return false }
func (m *mockServerStats) WaitIfPaused() {}
func (m *mockServerStats) Broadcast(msg *ws.Message, excludeIDs ...string) int { return 0 }
func (m *mockServerStats) Send(id string, msg *ws.Message) error {
	m.msgs[id] = append(m.msgs[id], msg)
	return nil
}

func TestThrottleBroadcastBuiltin(t *testing.T) {
	registry := NewRegistry(ServerMode)
	stats := &mockServerStats{
		clients: []template.ClientInfo{
			{ID: "c1", RemoteAddr: "1.1.1.1"},
			{ID: "c2", RemoteAddr: "2.2.2.2"},
		},
		msgs: make(map[string][]*ws.Message),
	}

	engine := template.New(false)
	d := &Dispatcher{
		registry:       registry,
		templateEngine: engine,
		serverStats:    stats,
	}

	builtin := &ThrottleBroadcastBuiltin{}
	a := &Action{
		HandlerName: "test-handler",
		Window:      "1s",
		Message:     "hello",
	}

	tmplCtx := template.NewContext()

	// 1. First broadcast - both should receive
	err := builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 2, tmplCtx.DeliveredCount)
	assert.Equal(t, 0, tmplCtx.SkippedCount)
	assert.Len(t, stats.msgs["c1"], 1)
	assert.Len(t, stats.msgs["c2"], 1)

	// 2. Immediate second broadcast - both should be skipped
	tmplCtx = template.NewContext()
	err = builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 0, tmplCtx.DeliveredCount)
	assert.Equal(t, 2, tmplCtx.SkippedCount)
	assert.Len(t, stats.msgs["c1"], 1)
	assert.Len(t, stats.msgs["c2"], 1)

	// 3. Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	tmplCtx = template.NewContext()
	err = builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 2, tmplCtx.DeliveredCount)
	assert.Equal(t, 0, tmplCtx.SkippedCount)
	assert.Len(t, stats.msgs["c1"], 2)
	assert.Len(t, stats.msgs["c2"], 2)
}

func TestThrottleBroadcastBuiltin_TemplateWindow(t *testing.T) {
	registry := NewRegistry(ServerMode)
	stats := &mockServerStats{
		clients: []template.ClientInfo{
			{ID: "c1", RemoteAddr: "1.1.1.1"},
		},
		msgs: make(map[string][]*ws.Message),
	}

	engine := template.New(false)
	d := &Dispatcher{
		registry:       registry,
		templateEngine: engine,
		serverStats:    stats,
	}

	builtin := &ThrottleBroadcastBuiltin{}
	a := &Action{
		HandlerName: "test-handler",
		Window:      "{{.Vars.dur}}",
		Message:     "hello",
	}

	tmplCtx := template.NewContext()
	tmplCtx.Vars["dur"] = "500ms"

	// 1. First broadcast
	err := builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 1, tmplCtx.DeliveredCount)

	// 2. Immediate skip
	err = builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 0, tmplCtx.DeliveredCount)
	assert.Equal(t, 1, tmplCtx.SkippedCount)

	// 3. Wait 600ms
	time.Sleep(600 * time.Millisecond)
	err = builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 1, tmplCtx.DeliveredCount)
}
