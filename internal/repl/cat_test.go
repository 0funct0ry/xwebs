package repl

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
)

func TestCatCommand(t *testing.T) {
	var buf bytes.Buffer
	r, err := New(ClientMode, &Config{
		Terminal: false,
		Stdout:   &nopCloser{Writer: &buf},
	})
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "xwebs-cat-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("Basic cat", func(t *testing.T) {
		buf.Reset()
		filename := filepath.Join(tmpDir, "test.txt")
		content := "Hello World\nLine 2"
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":cat "+filename)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Hello World")
		assert.Contains(t, buf.String(), "Line 2")
	})

	t.Run("JSON cat with highlighting", func(t *testing.T) {
		buf.Reset()
		r.Display.Color = "on"
		filename := filepath.Join(tmpDir, "test.json")
		content := `{"key": "value", "num": 123}`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":cat "+filename)
		assert.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "\"key\":")
		assert.Contains(t, output, "\033[36m") // Cyan for keys
		assert.Contains(t, output, "123")
		assert.Contains(t, output, "\033[33m") // Yellow for numbers
	})

	t.Run("YAML cat with highlighting", func(t *testing.T) {
		buf.Reset()
		r.Display.Color = "on"
		filename := filepath.Join(tmpDir, "test.yaml")
		content := "name: test\nversion: 1.0"
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":cat "+filename)
		assert.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "name")
		assert.Contains(t, output, "\033[36m") // Cyan for keys
	})

	t.Run("Log cat with highlighting", func(t *testing.T) {
		buf.Reset()
		r.Display.Color = "on"
		filename := filepath.Join(tmpDir, "test.log")
		content := "2024-04-08T12:00:00Z INFO starting up\n2024-04-08T12:00:01Z ERROR failed to connect"
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":cat "+filename)
		assert.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "\033[1;36m") // Bold Cyan for INFO
		assert.Contains(t, output, "ERROR")
		assert.Contains(t, output, "\033[1;31m") // Bold Red for ERROR
		assert.Contains(t, output, "\033[2m")    // Dim for timestamps
	})

	t.Run("Large file truncation", func(t *testing.T) {
		buf.Reset()
		filename := filepath.Join(tmpDir, "large.txt")
		var content strings.Builder
		for i := 0; i < 600; i++ {
			content.WriteString("line\n")
		}
		if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":cat "+filename)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "[Output truncated.")
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		// 500 lines + truncation message
		assert.LessOrEqual(t, len(lines), 505)
	})

	t.Run("Non-existent file", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":cat /non/existent/file")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "opening file")
	})

	t.Run("Is directory error", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":cat "+tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is a directory")
	})
}
