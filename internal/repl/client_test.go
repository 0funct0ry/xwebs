package repl

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/0funct0ry/xwebs/internal/template"
)

type mockClientContext struct {
	conn       *ws.Connection
	dialURL    string
	closed     bool
	tmplEngine *template.Engine
}

func (m *mockClientContext) GetConnection() *ws.Connection { return m.conn }
func (m *mockClientContext) SetConnection(conn *ws.Connection) { m.conn = conn }
func (m *mockClientContext) Dial(ctx context.Context, url string) error {
	m.dialURL = url
	return nil
}
func (m *mockClientContext) CloseConnection() error {
	m.closed = true
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}
func (m *mockClientContext) GetTemplateEngine() *template.Engine { return m.tmplEngine }

func TestClientCommands(t *testing.T) {
	r, _ := New(ClientMode, nil)
	mcc := &mockClientContext{}
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
}

func TestBinaryCommand(t *testing.T) {
	r, _ := New(ClientMode, nil)
	mcc := &mockClientContext{}
	r.RegisterClientCommands(mcc)
	
	// Create a dummy channel for the mock connection to avoid nil dereference if Write is called
	// Actually we should mock the connection too if we want to verify Write
}
