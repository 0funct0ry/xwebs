package repl

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

func TestMkdirCommand(t *testing.T) {
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

	// Create a temporary directory for testing root
	testRoot, err := os.MkdirTemp("", "xwebs-mkdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testRoot)

	t.Run("Create simple directory", func(t *testing.T) {
		buf.Reset()
		dirName := "simple_dir"
		path := filepath.Join(testRoot, dirName)

		err := r.ExecuteCommand(context.Background(), ":mkdir "+path)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", path)
		}

		output := buf.String()
		if !strings.Contains(output, "Directory created") || !strings.Contains(output, dirName) {
			t.Errorf("Unexpected output: %s", output)
		}
	})

	t.Run("Create nested directory without -p (should fail)", func(t *testing.T) {
		buf.Reset()
		path := filepath.Join(testRoot, "nested", "fail")

		err := r.ExecuteCommand(context.Background(), ":mkdir "+path)
		if err == nil {
			t.Errorf("Expected error when creating nested directory without -p")
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Directory %s should not have been created", path)
		}
	})

	t.Run("Create nested directory with -p", func(t *testing.T) {
		buf.Reset()
		dirName := filepath.Join("nested", "success")
		path := filepath.Join(testRoot, dirName)

		err := r.ExecuteCommand(context.Background(), ":mkdir -p "+path)
		if err != nil {
			t.Errorf("Unexpected error with -p: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created with -p", path)
		}

		output := buf.String()
		if !strings.Contains(output, "Directory created") || !strings.Contains(output, dirName) {
			t.Errorf("Unexpected output: %s", output)
		}
	})

	t.Run("Create existing directory without -p (should fail)", func(t *testing.T) {
		buf.Reset()
		path := filepath.Join(testRoot, "existing")
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatalf("Failed to create existing dir: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":mkdir "+path)
		if err == nil {
			t.Errorf("Expected error when creating already existing directory")
		}
	})

	t.Run("Create existing directory with -p (should succeed)", func(t *testing.T) {
		buf.Reset()
		path := filepath.Join(testRoot, "existing_p")
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatalf("Failed to create existing dir: %v", err)
		}

		err := r.ExecuteCommand(context.Background(), ":mkdir -p "+path)
		if err != nil {
			t.Errorf("Unexpected error with -p on existing directory: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Directory created") {
			t.Errorf("Unexpected output: %s", output)
		}
	})
}
