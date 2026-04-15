package repl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

type mockServerContext struct {
	handlers     []handler.Handler
	updated      handler.Handler
	applied      []handler.Handler
	vars         map[string]interface{}
	handlersFile string
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
func (m *mockServerContext) GetVariables() map[string]interface{}           { return m.vars }
func (m *mockServerContext) GetHandlersFile() string                        { return m.handlersFile }
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
func (m *mockServerContext) GetTopics() []template.TopicInfo                             { return nil }
func (m *mockServerContext) GetTopic(name string) (template.TopicInfo, bool)             { return template.TopicInfo{}, false }
func (m *mockServerContext) PublishToTopic(topic string, msg *ws.Message) (int, error)   { return 0, nil }
func (m *mockServerContext) SubscribeClientToTopic(clientID, topic string) error         { return nil }
func (m *mockServerContext) UnsubscribeClientFromTopic(clientID, topic string) (int, error) { return 0, nil }
func (m *mockServerContext) UnsubscribeClientFromAllTopics(clientID string) ([]string, error) { return nil, nil }
func (m *mockServerContext) ListKV() map[string]interface{} { return nil }
func (m *mockServerContext) GetKV(key string) (interface{}, bool) { return nil, false }
func (m *mockServerContext) SetKV(key string, val interface{}) {}
func (m *mockServerContext) DeleteKV(key string) {}

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

	t.Run("save handlers to file", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "handlers.yaml")
		err := r.ExecuteCommand(context.Background(), ":handler save "+outFile)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		raw, readErr := os.ReadFile(outFile)
		if readErr != nil {
			t.Fatalf("failed reading saved file: %v", readErr)
		}
		content := string(raw)
		if !strings.Contains(content, "handlers:") {
			t.Fatalf("expected handlers yaml content, got: %s", content)
		}
		if !strings.Contains(content, "name: new-echo") {
			t.Fatalf("expected renamed handler in saved file, got: %s", content)
		}
	})

	t.Run("save refuses overwrite without force", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "handlers.yaml")
		if err := os.WriteFile(outFile, []byte("existing: true\n"), 0644); err != nil {
			t.Fatalf("failed to seed existing file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":handler save "+outFile)
		if err == nil {
			t.Fatalf("Expected overwrite error")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("save overwrites with force", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "handlers.yaml")
		if err := os.WriteFile(outFile, []byte("existing: true\n"), 0644); err != nil {
			t.Fatalf("failed to seed existing file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":handler save "+outFile+" --force")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		raw, readErr := os.ReadFile(outFile)
		if readErr != nil {
			t.Fatalf("failed reading saved file: %v", readErr)
		}
		content := string(raw)
		if strings.Contains(content, "existing: true") {
			t.Fatalf("expected file overwrite with handler yaml, got: %s", content)
		}
		if !strings.Contains(content, "handlers:") {
			t.Fatalf("expected handlers yaml content, got: %s", content)
		}
	})

	t.Run("save without filename uses --handlers path", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "handlers.yaml")
		msc.handlersFile = outFile

		err := r.ExecuteCommand(context.Background(), ":handler save")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		raw, readErr := os.ReadFile(outFile)
		if readErr != nil {
			t.Fatalf("failed reading saved file: %v", readErr)
		}
		content := string(raw)
		if !strings.Contains(content, "handlers:") {
			t.Fatalf("expected handlers yaml content, got: %s", content)
		}
	})

	t.Run("save without filename fails when --handlers is missing", func(t *testing.T) {
		msc.handlersFile = ""
		err := r.ExecuteCommand(context.Background(), ":handler save")
		if err == nil {
			t.Fatalf("Expected usage error when --handlers is unavailable")
		}
		if !strings.Contains(err.Error(), "start with --handlers") {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("save without filename supports force overwrite", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "handlers.yaml")
		msc.handlersFile = outFile
		if err := os.WriteFile(outFile, []byte("existing: true\n"), 0644); err != nil {
			t.Fatalf("failed to seed existing file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":handler save --force")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		raw, readErr := os.ReadFile(outFile)
		if readErr != nil {
			t.Fatalf("failed reading saved file: %v", readErr)
		}
		content := string(raw)
		if strings.Contains(content, "existing: true") {
			t.Fatalf("expected file overwrite with handler yaml, got: %s", content)
		}
	})
}
