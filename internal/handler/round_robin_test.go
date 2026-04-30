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

type rrMockServerStats struct {
	clients  []template.ClientInfo
	sentMsgs map[string][]*ws.Message
	paused   bool
}

func (m *rrMockServerStats) GetClientCount() int                                 { return len(m.clients) }
func (m *rrMockServerStats) GetUptime() time.Duration                            { return time.Hour }
func (m *rrMockServerStats) GetClients() []template.ClientInfo                   { return m.clients }
func (m *rrMockServerStats) IsPaused() bool                                      { return m.paused }
func (m *rrMockServerStats) WaitIfPaused()                                       {}
func (m *rrMockServerStats) Broadcast(msg *ws.Message, excludeIDs ...string) int { return 0 }
func (m *rrMockServerStats) Send(id string, msg *ws.Message) error {
	if m.sentMsgs == nil {
		m.sentMsgs = make(map[string][]*ws.Message)
	}
	m.sentMsgs[id] = append(m.sentMsgs[id], msg)
	return nil
}

func (m *rrMockServerStats) SendToSSE(stream, event, data, id string) error { return nil }
func (m *rrMockServerStats) UpdateSSEStreamConfig(stream, onNoConsumers string, bufferSize int) error {
	return nil
}
func (m *rrMockServerStats) RegisterHTTPMock(path string, mock template.HTTPMockResponse) error { return nil }
func (m *rrMockServerStats) GetGlobalStats() interface{} { return nil }
func (m *rrMockServerStats) GetRegistryStats() (uint64, uint64) { return 0, 0 }
func (m *rrMockServerStats) GetTopics() []template.TopicInfo { return nil }
func (m *rrMockServerStats) GetKVStore() map[string]interface{} { return nil }

func TestRoundRobinBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	stats := &rrMockServerStats{
		clients: []template.ClientInfo{
			{ID: "c1"},
			{ID: "c2"},
			{ID: "c3"},
		},
	}

	conn := &mockConn{}
	engine := template.New(false)
	d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, stats, nil, nil, nil, "")

	handlerName := "rr-test"
	pool := `["c1", "c2", "c3"]`

	action := &Action{
		Type:        "builtin",
		Command:     "round-robin",
		Pool:        pool,
		HandlerName: handlerName,
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, &ws.Message{Data: []byte("test")})

	builtin, ok := GetBuiltin("round-robin")
	require.True(t, ok)

	// 1. First execution should go to c1
	err := builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 1)
	assert.Len(t, stats.sentMsgs["c2"], 0)
	assert.Len(t, stats.sentMsgs["c3"], 0)

	// 2. Second execution should go to c2
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 1)
	assert.Len(t, stats.sentMsgs["c2"], 1)
	assert.Len(t, stats.sentMsgs["c3"], 0)

	// 3. Third execution should go to c3
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 1)
	assert.Len(t, stats.sentMsgs["c2"], 1)
	assert.Len(t, stats.sentMsgs["c3"], 1)

	// 4. Fourth execution should cycle back to c1
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 2)
}

func TestRoundRobinSkipDisconnected(t *testing.T) {
	reg := NewRegistry(ServerMode)
	stats := &rrMockServerStats{
		clients: []template.ClientInfo{
			{ID: "c1"},
			// c2 is "disconnected" (not in list)
			{ID: "c3"},
		},
	}

	conn := &mockConn{}
	engine := template.New(false)
	d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, stats, nil, nil, nil, "")

	handlerName := "rr-skip-test"
	pool := `["c1", "c2", "c3"]`

	action := &Action{
		Type:        "builtin",
		Command:     "round-robin",
		Pool:        pool,
		HandlerName: handlerName,
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, &ws.Message{Data: []byte("test")})

	builtin, _ := GetBuiltin("round-robin")

	// 1. First goes to c1
	err := builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 1)

	// 2. Second should skip c2 and go to c3
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c2"], 0)
	assert.Len(t, stats.sentMsgs["c3"], 1)

	// Next one should start after c3, so cycle to c1
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 2)
}

func TestRoundRobinOnEmpty(t *testing.T) {
	reg := NewRegistry(ServerMode)
	stats := &rrMockServerStats{
		clients: []template.ClientInfo{}, // No one connected
	}

	conn := &mockConn{}
	engine := template.New(false)
	d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, stats, nil, nil, nil, "")

	handlerName := "rr-empty-test"
	pool := `["c1", "c2"]`

	action := &Action{
		Type:        "builtin",
		Command:     "round-robin",
		Pool:        pool,
		OnEmpty:     "No backend available",
		HandlerName: handlerName,
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, &ws.Message{Data: []byte("test")})

	builtin, _ := GetBuiltin("round-robin")

	err := builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)

	// Should have sent on_empty to sender
	require.Len(t, conn.messages, 1)
	assert.Equal(t, "No backend available", string(conn.messages[0].Data))
}

func TestRoundRobinTemplatePool(t *testing.T) {
	reg := NewRegistry(ServerMode)
	stats := &rrMockServerStats{
		clients: []template.ClientInfo{
			{ID: "c1"},
		},
	}

	conn := &mockConn{}
	engine := template.New(false)

	// Set a variable for the pool
	vars := map[string]interface{}{
		"backends": []string{"c1", "c2"},
	}

	d := NewDispatcher(reg, conn, engine, true, vars, nil, false, nil, stats, nil, nil, nil, "")

	action := &Action{
		Type:        "builtin",
		Command:     "round-robin",
		Pool:        `{{ .Vars.backends | toJSON }}`,
		HandlerName: "rr-tmpl-test",
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, &ws.Message{Data: []byte("test")})

	builtin, _ := GetBuiltin("round-robin")

	err := builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)
	assert.Len(t, stats.sentMsgs["c1"], 1)
}

func TestRoundRobinCustomMessage(t *testing.T) {
	reg := NewRegistry(ServerMode)
	stats := &rrMockServerStats{
		clients: []template.ClientInfo{
			{ID: "c1"},
		},
	}

	conn := &mockConn{}
	engine := template.New(false)
	d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, stats, nil, nil, nil, "")

	action := &Action{
		Type:        "builtin",
		Command:     "round-robin",
		Pool:        `["c1"]`,
		Message:     "Custom: {{.Message}}",
		HandlerName: "rr-custom-msg",
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, &ws.Message{Type: ws.TextMessage, Data: []byte("original")})

	builtin, _ := GetBuiltin("round-robin")

	err := builtin.Execute(context.Background(), d, action, tmplCtx)
	assert.NoError(t, err)

	require.Len(t, stats.sentMsgs["c1"], 1)
	assert.Equal(t, "Custom: original", string(stats.sentMsgs["c1"][0].Data))
}
