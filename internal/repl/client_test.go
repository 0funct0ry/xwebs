package repl

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

type mockClientContext struct {
	conn        *ws.Connection
	dialURL     string
	closed      bool
	closeCode   int
	closeReason string
	tmplEngine  *template.Engine
}

func (m *mockClientContext) GetConnection() *ws.Connection     { return m.conn }
func (m *mockClientContext) SetConnection(conn *ws.Connection) { m.conn = conn }
func (m *mockClientContext) Dial(ctx context.Context, url string) error {
	m.dialURL = url
	return nil
}
func (m *mockClientContext) CloseConnectionWithCode(code int, reason string) error {
	m.closed = true
	m.closeCode = code
	m.closeReason = reason
	// Don't call m.conn.CloseWithCode as it might panic if not fully initialized (e.g. in tests)
	return nil
}
func (m *mockClientContext) CloseConnection() error {
	m.closed = true
	return nil
}
func (m *mockClientContext) GetTemplateEngine() *template.Engine { return m.tmplEngine }

func TestClientCommands(t *testing.T) {
	r, _ := New(ClientMode, nil)
	mcc := &mockClientContext{
		tmplEngine: template.New(false),
	}
	r.RegisterClientCommands(mcc)

	t.Run("connect command", func(t *testing.T) {
		err := r.executeCommand(context.Background(), ":connect ws://example.com")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mcc.dialURL != "ws://example.com" {
			t.Errorf("Expected dial URL 'ws://example.com', got %q", mcc.dialURL)
		}
	})

	t.Run("connect command with template", func(t *testing.T) {
		r.SetVar("host", "example.org")
		err := r.executeCommand(context.Background(), ":connect ws://{{.Session.host}}")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mcc.dialURL != "ws://example.org" {
			t.Errorf("Expected dial URL 'ws://example.org', got %q", mcc.dialURL)
		}
	})

	t.Run("reconnect command", func(t *testing.T) {
		err := r.executeCommand(context.Background(), ":reconnect")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mcc.dialURL != "" {
			t.Errorf("Expected empty dial URL for reconnect, got %q", mcc.dialURL)
		}
	})

	t.Run("disconnect command", func(t *testing.T) {
		err := r.executeCommand(context.Background(), ":disconnect")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mcc.closed {
			t.Errorf("Expected connection to be closed")
		}
	})

	t.Run("close command defaults", func(t *testing.T) {
		mcc.closed = false
		mcc.closeCode = 0
		mcc.closeReason = ""

		// We need a dummy connection to satisfy GetConnection() != nil
		mcc.conn = &ws.Connection{}

		err := r.executeCommand(context.Background(), ":close")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mcc.closed {
			t.Error("Expected connection to be closed")
		}
		if mcc.closeCode != 1000 {
			t.Errorf("Expected code 1000, got %d", mcc.closeCode)
		}
		if mcc.closeReason != "Normal Closure" {
			t.Errorf("Expected reason 'Normal Closure', got %q", mcc.closeReason)
		}
	})

	t.Run("close command custom code", func(t *testing.T) {
		mcc.closed = false
		err := r.executeCommand(context.Background(), ":close 1001")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mcc.closeCode != 1001 {
			t.Errorf("Expected code 1001, got %d", mcc.closeCode)
		}
		if mcc.closeReason != "Normal Closure" {
			t.Errorf("Expected default reason, got %q", mcc.closeReason)
		}
	})

	t.Run("close command custom code and reason", func(t *testing.T) {
		mcc.closed = false
		err := r.executeCommand(context.Background(), ":close 4000 Custom Reason")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mcc.closeCode != 4000 {
			t.Errorf("Expected code 4000, got %d", mcc.closeCode)
		}
		if mcc.closeReason != "Custom Reason" {
			t.Errorf("Expected reason 'Custom Reason', got %q", mcc.closeReason)
		}
	})

	t.Run("close command reason only", func(t *testing.T) {
		mcc.closed = false
		err := r.executeCommand(context.Background(), ":close Bye for now")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mcc.closeCode != 1000 {
			t.Errorf("Expected default code 1000, got %d", mcc.closeCode)
		}
		if mcc.closeReason != "Bye for now" {
			t.Errorf("Expected reason 'Bye for now', got %q", mcc.closeReason)
		}
	})
}

func TestBinaryCommand(t *testing.T) {
	r, _ := New(ClientMode, nil)
	mcc := &mockClientContext{}
	r.RegisterClientCommands(mcc)

	// Create a dummy channel for the mock connection to avoid nil dereference if Write is called
	// Actually we should mock the connection too if we want to verify Write
}
