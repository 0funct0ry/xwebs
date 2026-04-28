package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
)

type ruleEngineMockConn struct {
	mockConn
	sent []*ws.Message
}

func (m *ruleEngineMockConn) Write(msg *ws.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

func (m *ruleEngineMockConn) GetID() string { return "test-conn" }

func TestRuleEngineBuiltin(t *testing.T) {
	registry := NewRegistry(ServerMode)
	engine := template.New(false)
	builtin := &RuleEngineBuiltin{}

	tests := []struct {
		name     string
		rules    []Rule
		def      string
		msg      string
		expected string
	}{
		{
			name: "first match wins",
			rules: []Rule{
				{When: Matcher{Pattern: "ping"}, Respond: "pong1"},
				{When: Matcher{Pattern: "ping"}, Respond: "pong2"},
			},
			msg:      "ping",
			expected: "pong1",
		},
		{
			name: "second match wins",
			rules: []Rule{
				{When: Matcher{Pattern: "hello"}, Respond: "hi"},
				{When: Matcher{Pattern: "ping"}, Respond: "pong"},
			},
			msg:      "ping",
			expected: "pong",
		},
		{
			name: "jq match",
			rules: []Rule{
				{When: Matcher{Type: "jq", Pattern: ".type == \"login\""}, Respond: "welcome"},
			},
			msg:      `{"type": "login"}`,
			expected: "welcome",
		},
		{
			name: "default wins",
			rules: []Rule{
				{When: Matcher{Pattern: "ping"}, Respond: "pong"},
			},
			def:      "fallback",
			msg:      "unknown",
			expected: "fallback",
		},
		{
			name: "no match no default",
			rules: []Rule{
				{When: Matcher{Pattern: "ping"}, Respond: "pong"},
			},
			msg:      "unknown",
			expected: "",
		},
		{
			name: "template respond",
			rules: []Rule{
				{When: Matcher{Pattern: "echo *"}, Respond: "echoed: {{index .Matches 0}}"},
			},
			msg:      "echo hello",
			expected: "echoed: hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &ruleEngineMockConn{}
			dispatcher := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, nil, nil, nil, nil)

			tmplCtx := template.NewContext()
			msg := &ws.Message{Data: []byte(tt.msg), Metadata: ws.MessageMetadata{Direction: "received"}}
			dispatcher.populateTemplateContext(tmplCtx, msg)

			action := &Action{
				Rules:   tt.rules,
				Default: tt.def,
			}

			err := builtin.Execute(context.Background(), dispatcher, action, tmplCtx)
			assert.NoError(t, err)

			if tt.expected == "" {
				assert.Empty(t, conn.sent)
			} else {
				assert.Len(t, conn.sent, 1)
				assert.Equal(t, tt.expected, string(conn.sent[0].Data))
			}
		})
	}
}
