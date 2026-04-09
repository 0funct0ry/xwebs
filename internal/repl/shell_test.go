package repl

import (
	"context"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

func TestShellCommand(t *testing.T) {
	r, err := New(ClientMode, &Config{Terminal: true})
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	// Mocking output to capture it
	var sb strings.Builder
	r.config.Stdout = &testWriter{Writer: &sb}
	r.IsInteractive = false

	t.Run("Basic shell execution", func(t *testing.T) {
		sb.Reset()
		err := r.ExecuteCommand(context.Background(), ":! echo hello")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := sb.String()
		if !strings.Contains(output, "hello") {
			t.Errorf("Expected output to contain 'hello', got %q", output)
		}
	})

	t.Run("Shell execution with pipes", func(t *testing.T) {
		sb.Reset()
		// Use a simple pipe
		err := r.ExecuteCommand(context.Background(), ":! echo foo | grep foo")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := sb.String()
		if !strings.Contains(output, "foo") {
			t.Errorf("Pipeline failed, got output: %q", output)
		}
	})

	t.Run("Shell execution with quoted arguments", func(t *testing.T) {
		sb.Reset()
		// If we use :! echo "a   b", it should preserve spaces because we intercept raw string
		err := r.ExecuteCommand(context.Background(), ":! echo \"a   b\"")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := sb.String()
		// The echo command will output a   b (quotes removed by shell)
		if !strings.Contains(output, "a   b") {
			t.Errorf("Expected 'a   b' with preserved spaces, got %q", output)
		}
	})

	t.Run("Command failure", func(t *testing.T) {
		sb.Reset()
		err := r.ExecuteCommand(context.Background(), ":! exit 42")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		output := sb.String()
		if !strings.Contains(output, "code 42") {
			t.Errorf("Expected error message for exit code 42, got %q", output)
		}
	})
}


