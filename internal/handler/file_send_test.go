package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSendBuiltin(t *testing.T) {
	reg := NewRegistry(ClientMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")

	// Setup temporary directory for files
	tmpDir, err := os.MkdirTemp("", "xwebs-files-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("Basic Text Send", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		filePath := filepath.Join(tmpDir, "hello.txt")
		err := os.WriteFile(filePath, []byte("Hello World!"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "file-send-text",
			Builtin: "file-send",
			File:    filePath,
			Mode:    "text",
		}
		msg := &ws.Message{
			Data:     []byte("trigger"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, "Hello World!", string(conn.lastWritten))
		require.Len(t, conn.messages, 1)
		assert.Equal(t, ws.TextMessage, conn.messages[0].Type)
	})

	t.Run("Binary Send", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		filePath := filepath.Join(tmpDir, "data.bin")
		data := []byte{0x00, 0x01, 0x02, 0x03}
		err := os.WriteFile(filePath, data, 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "file-send-binary",
			Builtin: "file-send",
			File:    filePath,
			Mode:    "binary",
		}
		msg := &ws.Message{
			Data:     []byte("trigger"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		require.Len(t, conn.messages, 1)
		assert.Equal(t, data, conn.messages[0].Data)
		assert.Equal(t, ws.BinaryMessage, conn.messages[0].Type)
	})

	t.Run("Dynamic File Path", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		err := os.WriteFile(filepath.Join(tmpDir, "A.txt"), []byte("Content A"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "B.txt"), []byte("Content B"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "file-send-dynamic",
			Builtin: "file-send",
			File:    filepath.Join(tmpDir, "{{.Message}}.txt"),
		}

		// Test with A
		msgA := &ws.Message{
			Data:     []byte("A"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}
		err = d.Execute(context.Background(), h, msgA, nil)
		require.NoError(t, err)
		assert.Equal(t, "Content A", string(conn.lastWritten))

		// Test with B
		msgB := &ws.Message{
			Data:     []byte("B"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}
		err = d.Execute(context.Background(), h, msgB, nil)
		require.NoError(t, err)
		assert.Equal(t, "Content B", string(conn.lastWritten))
	})

	t.Run("Relative Path with BaseDir", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		fileName := "relative_file.txt"
		err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte("Relative File Content"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "file-send-relative",
			Builtin: "file-send",
			File:    fileName,
			BaseDir: tmpDir,
		}
		msg := &ws.Message{
			Data:     []byte("test"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, "Relative File Content", string(conn.lastWritten))
	})

	t.Run("Missing File Error", func(t *testing.T) {
		h := &Handler{
			Name:    "file-send-missing",
			Builtin: "file-send",
			File:    "missing.txt",
			BaseDir: tmpDir,
		}
		msg := &ws.Message{
			Data:     []byte("test"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("ServerMode Scope Check", func(t *testing.T) {
		regServer := NewRegistry(ServerMode)
		h := Handler{
			Name:    "file-send-server",
			Match:   Matcher{Pattern: "*"},
			Builtin: "file-send",
			File:    "test.txt",
		}

		err := regServer.Add(h)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "builtin \"file-send\" is only available in client mode")
	})
}
