package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockTopicManager struct {
	mock.Mock
}

func (m *mockTopicManager) Subscribe(connID string, conn Connection, topic string) {
	m.Called(connID, conn, topic)
}

func (m *mockTopicManager) Unsubscribe(connID, topic string) int {
	args := m.Called(connID, topic)
	return args.Int(0)
}

func (m *mockTopicManager) Publish(topic string, msg *ws.Message) (int, error) {
	args := m.Called(topic, msg)
	return args.Int(0), args.Error(1)
}

func (m *mockTopicManager) PublishSticky(topic string, msg *ws.Message) (int, error) {
	args := m.Called(topic, msg)
	return args.Int(0), args.Error(1)
}

func (m *mockTopicManager) ClearRetained(topic string) {
	m.Called(topic)
}

func TestPubSubBuiltins(t *testing.T) {
	registry := NewRegistry(ServerMode)
	conn := &mockConn{}

	tm := new(mockTopicManager)
	engine := template.New(false)
	d := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, nil, tm, nil, nil, nil, "")

	ctx := context.Background()
	msg := &ws.Message{Data: []byte("trigger"), Metadata: ws.MessageMetadata{Direction: "received"}}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)

	// Set up all expectations
	tm.On("Subscribe", "mock-conn-id", conn, "chat-127.0.0.1:12345").Return().Once()
	tm.On("Unsubscribe", "mock-conn-id", "chat-127.0.0.1:12345").Return(0).Once()
	tm.On("Publish", "alerts", mock.Anything).Return(5, nil).Once()
	tm.On("Publish", "logs", mock.Anything).Return(1, nil).Once()

	// 1. Test Subscribe
	subAction := &Action{
		Type:    "builtin",
		Command: "subscribe",
		Topic:   "chat-{{.Conn.RemoteAddr}}",
	}
	err := d.ExecuteAction(ctx, subAction, tmplCtx, msg)
	assert.NoError(t, err)

	// 2. Test Unsubscribe
	unsubAction := &Action{
		Type:    "builtin",
		Command: "unsubscribe",
		Topic:   "chat-127.0.0.1:12345",
	}
	err = d.ExecuteAction(ctx, unsubAction, tmplCtx, msg)
	assert.NoError(t, err)

	// 3. Test Publish with Message
	pubAction := &Action{
		Type:    "builtin",
		Command: "publish",
		Topic:   "alerts",
		Message: "Alert from {{.ConnectionID}}: {{.Message}}",
	}
	err = d.ExecuteAction(ctx, pubAction, tmplCtx, msg)
	assert.NoError(t, err)

	// 4. Test Publish with Respond
	pubRespondAction := &Action{
		Type:    "builtin",
		Command: "publish",
		Topic:   "logs",
		Respond: "LOG: {{.Message}}",
	}
	err = d.ExecuteAction(ctx, pubRespondAction, tmplCtx, msg)
	assert.NoError(t, err)

	tm.AssertExpectations(t)
}
