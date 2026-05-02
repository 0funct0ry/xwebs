package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

type mockSSEProvider struct {
	sentToSSE             map[string][]string
	updatedConfigs        map[string]struct{ onNoConsumers string; bufferSize int }
}

func (m *mockSSEProvider) SendToSSE(stream, event, data, id string) error {
	if m.sentToSSE == nil {
		m.sentToSSE = make(map[string][]string)
	}
	m.sentToSSE[stream] = append(m.sentToSSE[stream], event+":"+data)
	return nil
}

func (m *mockSSEProvider) UpdateSSEStreamConfig(stream, onNoConsumers string, bufferSize int) error {
	if m.updatedConfigs == nil {
		m.updatedConfigs = make(map[string]struct{ onNoConsumers string; bufferSize int })
	}
	m.updatedConfigs[stream] = struct{ onNoConsumers string; bufferSize int }{onNoConsumers, bufferSize}
	return nil
}

// Implement other methods of ServerStatProvider if needed
func (m *mockSSEProvider) GetClientCount() int                                 { return 0 }
func (m *mockSSEProvider) GetUptime() time.Duration                            { return 0 }
func (m *mockSSEProvider) GetClients() []template.ClientInfo                   { return nil }
func (m *mockSSEProvider) IsPaused() bool                                      { return false }
func (m *mockSSEProvider) WaitIfPaused()                                       {}
func (m *mockSSEProvider) Broadcast(msg *ws.Message, excludeIDs ...string) int { return 0 }
func (m *mockSSEProvider) Send(id string, msg *ws.Message) error               { return nil }
func (m *mockSSEProvider) RegisterHTTPMock(path string, mock template.HTTPMockResponse) error { return nil }
func (m *mockSSEProvider) GetTopics() []template.TopicInfo                     { return nil }
func (m *mockSSEProvider) GetKVStore() map[string]interface{}                  { return nil }
func (m *mockSSEProvider) GetGlobalStats() interface{}                         { return nil }
func (m *mockSSEProvider) GetRegistryStats() (uint64, uint64)                  { return 0, 0 }

func TestSSEForwardBuiltin(t *testing.T) {
	registry := NewRegistry(ServerMode)
	te := template.New(false)
	provider := &mockSSEProvider{}
	
	dispatcher := NewDispatcher(registry, nil, te, true, nil, nil, false, nil, provider, nil, nil, nil, nil, "")
	
	h := &Handler{
		Name:    "test-sse",
		Builtin: "sse-forward",
		Stream:  "alerts",
		Event:   "critical",
		Message: "ALERT: {{.Message}}",
	}
	
	msg := &ws.Message{
		Data: []byte("fire!"),
		Type: ws.TextMessage,
	}
	
	ctx := context.Background()
	err := dispatcher.Execute(ctx, h, msg, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if len(provider.sentToSSE["alerts"]) != 1 {
		t.Fatalf("expected 1 message sent to SSE, got %d", len(provider.sentToSSE["alerts"]))
	}
	
	expected := "critical:ALERT: fire!"
	if provider.sentToSSE["alerts"][0] != expected {
		t.Errorf("expected %q, got %q", expected, provider.sentToSSE["alerts"][0])
	}
}

func TestSSEForwardBuiltin_ConfigUpdate(t *testing.T) {
	registry := NewRegistry(ServerMode)
	te := template.New(false)
	provider := &mockSSEProvider{}
	
	dispatcher := NewDispatcher(registry, nil, te, true, nil, nil, false, nil, provider, nil, nil, nil, nil, "")
	
	h := &Handler{
		Name:          "test-sse-config",
		Builtin:       "sse-forward",
		Stream:        "logs",
		OnNoConsumers: "buffer",
		BufferSize:    50,
	}
	
	msg := &ws.Message{
		Data: []byte("log line"),
		Type: ws.TextMessage,
	}
	
	ctx := context.Background()
	err := dispatcher.Execute(ctx, h, msg, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	cfg, ok := provider.updatedConfigs["logs"]
	if !ok {
		t.Fatal("expected config update for stream 'logs'")
	}
	if cfg.onNoConsumers != "buffer" || cfg.bufferSize != 50 {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
