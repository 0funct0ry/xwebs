package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

type mockMulticastServerStats struct {
	sentMessages map[string][]*ws.Message
	clients      []template.ClientInfo
}

func (m *mockMulticastServerStats) GetClientCount() int                                 { return len(m.clients) }
func (m *mockMulticastServerStats) GetUptime() time.Duration                            { return 0 }
func (m *mockMulticastServerStats) GetClients() []template.ClientInfo                   { return m.clients }
func (m *mockMulticastServerStats) IsPaused() bool                                      { return false }
func (m *mockMulticastServerStats) WaitIfPaused()                                       {}
func (m *mockMulticastServerStats) Broadcast(msg *ws.Message, excludeIDs ...string) int { return 0 }
func (m *mockMulticastServerStats) Send(id string, msg *ws.Message) error {
	found := false
	for _, c := range m.clients {
		if c.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("connection not found")
	}
	m.sentMessages[id] = append(m.sentMessages[id], msg)
	return nil
}

func (m *mockMulticastServerStats) SendToSSE(stream, event, data, id string) error { return nil }
func (m *mockMulticastServerStats) UpdateSSEStreamConfig(stream, onNoConsumers string, bufferSize int) error {
	return nil
}
func (m *mockMulticastServerStats) RegisterHTTPMock(path string, mock template.HTTPMockResponse) error { return nil }
func (m *mockMulticastServerStats) GetGlobalStats() interface{} { return nil }
func (m *mockMulticastServerStats) GetRegistryStats() (uint64, uint64) { return 0, 0 }
func (m *mockMulticastServerStats) GetTopics() []template.TopicInfo { return nil }
func (m *mockMulticastServerStats) GetKVStore() map[string]interface{} { return nil }

func TestMulticastBuiltin(t *testing.T) {
	stats := &mockMulticastServerStats{
		sentMessages: make(map[string][]*ws.Message),
		clients: []template.ClientInfo{
			{ID: "client1"},
			{ID: "client2"},
		},
	}

	registry := NewRegistry(ServerMode)
	conn := &mockMulticastConnection{id: "sender"}
	engine := template.New(false)
	d := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, stats, nil, nil, nil, nil, "")

	tests := []struct {
		name          string
		action        Action
		expectedSent  map[string]int
		expectedDeliv int
		expectedSkip  int
		wantErr       bool
	}{
		{
			name: "static list",
			action: Action{
				Type:    "builtin",
				Command: "multicast",
				Targets: `["client1", "client2"]`,
				Message: "hello",
			},
			expectedSent:  map[string]int{"client1": 1, "client2": 1},
			expectedDeliv: 2,
			expectedSkip:  0,
		},
		{
			name: "single id (fallback)",
			action: Action{
				Type:    "builtin",
				Command: "multicast",
				Targets: `client1`,
				Message: "hello",
			},
			expectedSent:  map[string]int{"client1": 1},
			expectedDeliv: 1,
			expectedSkip:  0,
		},
		{
			name: "comma separated (fallback)",
			action: Action{
				Type:    "builtin",
				Command: "multicast",
				Targets: `client1, client2, non-existent`,
				Message: "hello",
			},
			expectedSent:  map[string]int{"client1": 1, "client2": 1},
			expectedDeliv: 2,
			expectedSkip:  1,
		},
		{
			name: "template expression",
			action: Action{
				Type:    "builtin",
				Command: "multicast",
				Targets: `["{{.Vars.target}}"]`,
				Message: "hello {{.Vars.name}}",
			},
			expectedSent:  map[string]int{"client2": 1},
			expectedDeliv: 1,
			expectedSkip:  0,
		},
		{
			name: "missing targets",
			action: Action{
				Type:    "builtin",
				Command: "multicast",
				Message: "hello",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset stats
			stats.sentMessages = make(map[string][]*ws.Message)

			tmplCtx := template.NewContext()
			tmplCtx.Vars = map[string]interface{}{
				"target": "client2",
				"name":   "world",
			}

			builtin, ok := GetBuiltin("multicast")
			if !ok {
				t.Fatal("multicast builtin not found")
			}

			if tt.wantErr {
				if err := builtin.Validate(tt.action); err == nil {
					t.Error("expected validation error")
				}
				return
			}

			err := builtin.Execute(context.Background(), d, &tt.action, tmplCtx)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}

			if tmplCtx.DeliveredCount != tt.expectedDeliv {
				t.Errorf("DeliveredCount = %d, want %d", tmplCtx.DeliveredCount, tt.expectedDeliv)
			}
			if tmplCtx.SkippedCount != tt.expectedSkip {
				t.Errorf("SkippedCount = %d, want %d", tmplCtx.SkippedCount, tt.expectedSkip)
			}

			for id, count := range tt.expectedSent {
				if len(stats.sentMessages[id]) != count {
					t.Errorf("client %s received %d messages, want %d", id, len(stats.sentMessages[id]), count)
				}
			}
		})
	}
}

type mockMulticastConnection struct {
	id string
}

func (m *mockMulticastConnection) Write(msg *ws.Message) error                 { return nil }
func (m *mockMulticastConnection) CloseWithCode(code int, reason string) error { return nil }
func (m *mockMulticastConnection) Subscribe() <-chan *ws.Message               { return nil }
func (m *mockMulticastConnection) Unsubscribe(ch <-chan *ws.Message)           {}
func (m *mockMulticastConnection) Done() <-chan struct{}                       { return nil }
func (m *mockMulticastConnection) IsCompressionEnabled() bool                  { return false }
func (m *mockMulticastConnection) GetID() string                               { return m.id }
func (m *mockMulticastConnection) GetURL() string                              { return "" }
func (m *mockMulticastConnection) GetSubprotocol() string                      { return "" }
func (m *mockMulticastConnection) RemoteAddr() string                          { return "" }
func (m *mockMulticastConnection) LocalAddr() string                           { return "" }
func (m *mockMulticastConnection) ConnectedAt() time.Time                      { return time.Now() }
func (m *mockMulticastConnection) MessageCount() uint64                        { return 0 }
func (m *mockMulticastConnection) MsgsIn() uint64                              { return 0 }
func (m *mockMulticastConnection) MsgsOut() uint64                             { return 0 }
func (m *mockMulticastConnection) LastMsgReceivedAt() time.Time                { return time.Now() }
func (m *mockMulticastConnection) LastMsgSentAt() time.Time                    { return time.Now() }
func (m *mockMulticastConnection) RTT() time.Duration                          { return 0 }
func (m *mockMulticastConnection) AvgRTT() time.Duration                       { return 0 }
