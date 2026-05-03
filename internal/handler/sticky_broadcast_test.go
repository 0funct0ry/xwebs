package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStickyBroadcastBuiltin(t *testing.T) {
	registry := NewRegistry(ServerMode)
	conn := &mockConn{}
	tm := new(mockTopicManager)
	engine := template.New(false)
	d := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, nil, tm, nil, nil, nil, nil, nil, nil, "")

	ctx := context.Background()
	msg := &ws.Message{Data: []byte("trigger"), Metadata: ws.MessageMetadata{Direction: "received"}}
	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)

	// Test Sticky Broadcast
	tm.On("PublishSticky", "dashboard", mock.MatchedBy(func(m *ws.Message) bool {
		return string(m.Data) == "State: trigger"
	})).Return(2, nil).Once()

	action := &Action{
		Type:    "builtin",
		Command: "sticky-broadcast",
		Topic:   "dashboard",
		Message: "State: {{.Message}}",
	}

	err := d.ExecuteAction(ctx, action, tmplCtx, msg)
	assert.NoError(t, err)
	assert.Equal(t, "State: trigger", tmplCtx.Retained)

	tm.AssertExpectations(t)
}
