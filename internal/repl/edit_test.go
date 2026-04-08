package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

func TestREPLConfigManagement(t *testing.T) {
	r, err := New(ClientMode, &Config{Terminal: false})
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}
	r.TemplateEngine = template.New(false)

	t.Run("AddConfigPath", func(t *testing.T) {
		path := "test.yaml"
		abs, _ := filepath.Abs(path)
		r.AddConfigPath(path)
		
		found := false
		for _, p := range r.configPaths {
			if p == abs {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Config path %s not found in r.configPaths", abs)
		}

		// Test deduplication
		r.AddConfigPath(path)
		count := 0
		for _, p := range r.configPaths {
			if p == abs {
				count++
			}
		}
		if count != 1 {
			t.Errorf("Expected 1 instance of %s, got %d", abs, count)
		}
	})

	t.Run("ReloadConfig", func(t *testing.T) {
		// Prepare a dummy handlers file
		tmpDir, err := os.MkdirTemp("", "xwebs-reload-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		handlerFile := filepath.Join(tmpDir, "handlers.yaml")
		content := `
variables:
  reloaded: true
handlers:
  - name: test-handler
    match: "ping"
    run: "echo pong"
`
		if err := os.WriteFile(handlerFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write handlers file: %v", err)
		}

		err = r.ReloadConfig(handlerFile)
		if err != nil {
			t.Fatalf("ReloadConfig failed: %v", err)
		}

		// Verify variables
		if r.GetVar("reloaded") != true {
			t.Errorf("Expected variable 'reloaded' to be true, got %v", r.GetVar("reloaded"))
		}

		// Verify handlers
		handlers := r.Handlers.Handlers()
		if len(handlers) != 1 {
			t.Errorf("Expected 1 handler, got %d", len(handlers))
		}
		if handlers[0].Name != "test-handler" {
			t.Errorf("Expected handler name 'test-handler', got %s", handlers[0].Name)
		}
	})
}
