package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShadowBuiltin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xwebs-shadow-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "shadow.log")

	// 1. Setup registry with two handlers
	registry := NewRegistry(ServerMode)

	// Primary handler that shadows the secondary
	primary := Handler{
		Name:    "primary",
		Match:   Matcher{Pattern: "*"},
		Builtin: "shadow",
		Target:  "secondary",
		Respond: "Primary OK",
	}

	// Secondary handler that writes to a file
	secondary := Handler{
		Name:    "secondary",
		Match:   Matcher{Pattern: "DISABLED"}, // Match condition doesn't matter for shadow execution
		Builtin: "file-write",
		Path:    logPath,
		Content: "Shadowed: {{.Message}}",
	}

	require.NoError(t, registry.Add(primary))
	require.NoError(t, registry.Add(secondary))

	// 2. Setup dispatcher
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(registry, conn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")

	// 3. Execute primary handler
	msg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte("Hello Shadow"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
			Timestamp: time.Now(),
		},
	}

	tmplCtx := template.NewContext()
	d.populateTemplateContext(tmplCtx, msg)

	action := Action{
		Type:    "builtin",
		Command: "shadow",
		Target:  "secondary",
		Respond: "Primary OK",
	}

	err = d.ExecuteAction(context.Background(), &action, tmplCtx, msg)
	require.NoError(t, err)

	// 4. Verify primary response was sent
	conn.mu.Lock()
	foundPrimary := false
	for _, m := range conn.messages {
		if string(m.Data) == "Primary OK" {
			foundPrimary = true
		}
	}
	conn.mu.Unlock()
	assert.True(t, foundPrimary, "Primary response should be sent")

	// 5. Wait for async shadow execution and verify file write
	// We need to give it a moment as it's async
	deadline := time.Now().Add(2 * time.Second)
	var content []byte
	for time.Now().Before(deadline) {
		content, err = os.ReadFile(logPath)
		if err == nil && len(content) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NoError(t, err, "Shadow log file should be created")
	assert.Equal(t, "Shadowed: Hello Shadow", string(content))

	// 6. Verify that secondary handler's direct response (if it had one) would be discarded
	// Let's update secondary to also try to send a message
	secondary.Respond = "Secondary Response"
	require.NoError(t, registry.UpdateHandler(secondary))

	// Reset mock conn
	conn.mu.Lock()
	conn.messages = nil
	conn.mu.Unlock()

	// Clear log file
	_ = os.Remove(logPath)

	err = d.ExecuteAction(context.Background(), &action, tmplCtx, msg)
	require.NoError(t, err)

	// Wait for shadow execution
	for time.Now().Before(time.Now().Add(1 * time.Second)) {
		content, _ = os.ReadFile(logPath)
		if len(content) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	conn.mu.Lock()
	for _, m := range conn.messages {
		if string(m.Data) == "Secondary Response" {
			t.Errorf("Secondary response should have been discarded by silentConn")
		}
	}
	conn.mu.Unlock()
}
