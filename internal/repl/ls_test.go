package repl

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

type nopCloser struct {
	io.Writer
}

func (n *nopCloser) Close() error { return nil }

func TestLSCommand(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "xwebs-ls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files and a subdirectory
	files := []string{"file1.txt", "file2.log"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}
	subDirName := "subdir"
	subDir := filepath.Join(tmpDir, subDirName)
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	t.Run("Basic ls", func(t *testing.T) {
		buf.Reset()
		err := r.ExecuteCommand(context.Background(), ":ls "+tmpDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := buf.String()
		for _, f := range files {
			if !strings.Contains(output, f) {
				t.Errorf("Expected output to contain %s, got:\n%s", f, output)
			}
		}
		if !strings.Contains(output, subDirName) {
			t.Errorf("Expected output to contain %s, got:\n%s", subDirName, output)
		}
	})

	t.Run("Long format ls", func(t *testing.T) {
		buf.Reset()
		err := r.ExecuteCommand(context.Background(), ":ls -l "+tmpDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Mode") || !strings.Contains(output, "Size") || !strings.Contains(output, "Modified") {
			t.Errorf("Expected output to contain headers, got:\n%s", output)
		}
		for _, f := range files {
			if !strings.Contains(output, f) {
				t.Errorf("Expected output to contain %s, got:\n%s", f, output)
			}
		}
	})

	t.Run("Non-existent directory", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":ls /non/existent/path/for/xwebs/test")
		if err == nil {
			t.Errorf("Expected error for non-existent directory")
		}
	})
}
