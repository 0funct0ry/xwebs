package repl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

type mockServerContext struct {
	handlers []handler.Handler
	updated  handler.Handler
	applied  []handler.Handler
	vars     map[string]interface{}
}

func (m *mockServerContext) GetClientCount() int                            { return 0 }
func (m *mockServerContext) GetUptime() time.Duration                       { return 0 }
func (m *mockServerContext) GetClients() []template.ClientInfo              { return nil }
func (m *mockServerContext) GetClient(id string) (template.ClientInfo, bool) { return template.ClientInfo{}, false }
func (m *mockServerContext) Broadcast(msg *ws.Message) error                { return nil }
func (m *mockServerContext) Send(id string, msg *ws.Message) error          { return nil }
func (m *mockServerContext) Kick(id string, code int, reason string) error  { return nil }
func (m *mockServerContext) GetStatus() string                              { return "running" }
func (m *mockServerContext) GetTemplateEngine() *template.Engine            { return nil }
func (m *mockServerContext) GetHandlers() []handler.Handler                 { return m.handlers }
func (m *mockServerContext) EnableHandler(name string) error                { return nil }
func (m *mockServerContext) DisableHandler(name string) error               { return nil }
func (m *mockServerContext) ReloadHandlers() error                          { return nil }
func (m *mockServerContext) GetHandlerStats(name string) (uint64, time.Duration, uint64, bool) {
	return 0, 0, 0, false
}
func (m *mockServerContext) IsHandlerDisabled(name string) bool { return false }
func (m *mockServerContext) AddHandler(h handler.Handler) error {
	m.handlers = append(m.handlers, h)
	return nil
}
func (m *mockServerContext) UpdateHandler(h handler.Handler) error {
	m.updated = h
	return nil
}
func (m *mockServerContext) DeleteHandler(name string) error { return nil }
func (m *mockServerContext) RenameHandler(oldName, newName string) error {
	for i, h := range m.handlers {
		if h.Name == oldName {
			m.handlers[i].Name = newName
			return nil
		}
	}
	return fmt.Errorf("handler %q not found", oldName)
}
func (m *mockServerContext) ApplyHandlers(handlers []handler.Handler, variables map[string]interface{}) error {
	m.applied = handlers
	m.vars = variables
	return nil
}

func TestHandlerEdit(t *testing.T) {
	r, _ := New(ServerMode, &Config{Terminal: false})
	msc := &mockServerContext{
		handlers: []handler.Handler{
			{Name: "test-handler", Match: handler.Matcher{Pattern: "foo"}},
		},
	}
	r.RegisterServerCommands(msc)

	// Mock EDITOR to just succeed (no changes)
	os.Setenv("EDITOR", "cat")
	defer os.Unsetenv("EDITOR")

	t.Run("edit existing handler (no change)", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":handler edit test-handler")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if msc.updated.Name != "" {
			t.Errorf("Expected no update when no changes made")
		}
	})

	t.Run("edit full config (no change)", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":handler edit")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if msc.applied != nil {
			t.Errorf("Expected no apply when no changes made")
		}
	})

	t.Run("edit non-existent handler", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":handler edit non-existent")
		if err == nil {
			t.Errorf("Expected error for non-existent handler")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("rename handler", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":handler rename test-handler new-echo")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify rename
		handlers := msc.GetHandlers()
		found := false
		for _, h := range handlers {
			if h.Name == "new-echo" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Handler was not renamed to 'new-echo'")
		}
	})
}
