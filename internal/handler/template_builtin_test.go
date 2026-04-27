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

func TestTemplateBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil)

	// Setup temporary directory for templates
	tmpDir, err := os.MkdirTemp("", "xwebs-templates-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("Basic Rendering", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		tmplPath := filepath.Join(tmpDir, "hello.tmpl")
		err := os.WriteFile(tmplPath, []byte("Hello {{.Message}}!"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "template-basic",
			Builtin: "template",
			File:    tmplPath,
		}
		msg := &ws.Message{
			Data:     []byte("World"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		conn.mu.Lock()
		defer conn.mu.Unlock()
		assert.Equal(t, "Hello World!", conn.lastWritten)
	})

	t.Run("Dynamic File Path", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		// Create two template files
		err := os.WriteFile(filepath.Join(tmpDir, "a.tmpl"), []byte("Template A"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "b.tmpl"), []byte("Template B"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "template-dynamic",
			Builtin: "template",
			File:    filepath.Join(tmpDir, "{{.Message}}.tmpl"),
		}

		// Test with A
		msgA := &ws.Message{
			Data:     []byte("a"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}
		err = d.Execute(context.Background(), h, msgA, nil)
		require.NoError(t, err)
		assert.Equal(t, "Template A", conn.lastWritten)

		// Test with B
		msgB := &ws.Message{
			Data:     []byte("b"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}
		err = d.Execute(context.Background(), h, msgB, nil)
		require.NoError(t, err)
		assert.Equal(t, "Template B", conn.lastWritten)
	})

	t.Run("Relative Path with BaseDir", func(t *testing.T) {
		conn.mu.Lock()
		conn.lastWritten = ""
		conn.messages = nil
		conn.mu.Unlock()

		tmplName := "relative.tmpl"
		err := os.WriteFile(filepath.Join(tmpDir, tmplName), []byte("Relative Content"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "template-relative",
			Builtin: "template",
			File:    tmplName,
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
		assert.Equal(t, "Relative Content", conn.lastWritten)
	})

	t.Run("Missing File Error", func(t *testing.T) {
		h := &Handler{
			Name:    "template-missing",
			Builtin: "template",
			File:    "non-existent.tmpl",
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
		assert.Contains(t, err.Error(), "non-existent.tmpl")
	})

	t.Run("Template Error in File Content", func(t *testing.T) {
		tmplPath := filepath.Join(tmpDir, "error.tmpl")
		err := os.WriteFile(tmplPath, []byte("Hello {{.InvalidField}}!"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "template-render-error",
			Builtin: "template",
			File:    tmplPath,
		}
		msg := &ws.Message{
			Data:     []byte("test"),
			Type:     ws.TextMessage,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rendering template file")
	})
}
