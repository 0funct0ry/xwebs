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

func TestFileWriteBuiltin(t *testing.T) {
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, nil, "")

	// Setup temporary directory for files
	tmpDir, err := os.MkdirTemp("", "xwebs-fwrite-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("Overwrite Mode (Default)", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "overwrite.txt")
		err := os.WriteFile(filePath, []byte("original content"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "fwrite-overwrite",
			Builtin: "file-write",
			Path:    filePath,
			Content: "new content",
		}
		msg := &ws.Message{
			Data:     []byte("trigger"),
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(content))
	})

	t.Run("Append Mode", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "append.txt")
		err := os.WriteFile(filePath, []byte("line1\n"), 0644)
		require.NoError(t, err)

		h := &Handler{
			Name:    "fwrite-append",
			Builtin: "file-write",
			Path:    filePath,
			Content: "line2\n",
			Mode:    "append",
		}
		msg := &ws.Message{
			Data:     []byte("trigger"),
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "line1\nline2\n", string(content))
	})

	t.Run("Auto-create Directory", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nested/dir/file.txt")

		h := &Handler{
			Name:    "fwrite-mkdir",
			Builtin: "file-write",
			Path:    filePath,
			Content: "nested content",
		}
		msg := &ws.Message{
			Data:     []byte("trigger"),
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "nested content", string(content))
	})

	t.Run("Template Rendering", func(t *testing.T) {
		filePathTmpl := filepath.Join(tmpDir, "{{.Message}}.txt")

		h := &Handler{
			Name:    "fwrite-tmpl",
			Builtin: "file-write",
			Path:    filePathTmpl,
			Content: "Content for {{.Message}}",
		}
		msg := &ws.Message{
			Data:     []byte("hello"),
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		expectedPath := filepath.Join(tmpDir, "hello.txt")
		content, err := os.ReadFile(expectedPath)
		require.NoError(t, err)
		assert.Equal(t, "Content for hello", string(content))
	})

	t.Run("Raw Message Content", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "raw.bin")
		data := []byte{0xDE, 0xAD, 0xBE, 0xEF}

		h := &Handler{
			Name:    "fwrite-raw",
			Builtin: "file-write",
			Path:    filePath,
			// Content omitted
		}
		msg := &ws.Message{
			Data:     data,
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("Relative Path with BaseDir", func(t *testing.T) {
		fileName := "rel_write.txt"
		h := &Handler{
			Name:    "fwrite-relative",
			Builtin: "file-write",
			Path:    fileName,
			Content: "relative content",
			BaseDir: tmpDir,
		}
		msg := &ws.Message{
			Data:     []byte("test"),
			Metadata: ws.MessageMetadata{Direction: "received"},
		}

		err = d.Execute(context.Background(), h, msg, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, fileName))
		require.NoError(t, err)
		assert.Equal(t, "relative content", string(content))
	})
}
