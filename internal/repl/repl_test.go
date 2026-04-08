package repl

import (
	"context"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

type mockCommand struct {
	name     string
	help     string
	executed bool
	args     []string
}

func (c *mockCommand) Name() string { return c.name }
func (c *mockCommand) Help() string { return c.help }
func (c *mockCommand) Execute(ctx context.Context, r *REPL, args []string) error {
	c.executed = true
	c.args = args
	return nil
}

func TestREPLCommands(t *testing.T) {
	r, err := New(ClientMode, &Config{Terminal: true})
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}
	r.TemplateEngine = template.New(false)

	mock := &mockCommand{name: "test", help: "test command"}
	r.RegisterCommand(mock)

	t.Run("Execute command", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":test arg1 arg2")
		if err != nil {
			t.Errorf("Unexpected error executing command: %v", err)
		}
		if !mock.executed {
			t.Errorf("Command was not executed")
		}
		if len(mock.args) != 2 || mock.args[0] != "arg1" || mock.args[1] != "arg2" {
			t.Errorf("Unexpected args: %v", mock.args)
		}
	})

	t.Run("Unknown command", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":unknown")
		if err == nil {
			t.Errorf("Expected error for unknown command")
		}
		if !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestREPLVars(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true})
	r.TemplateEngine = template.New(false)

	r.SetVar("foo", "bar")
	val := r.GetVar("foo")
	if val != "bar" {
		t.Errorf("Expected 'bar', got %v", val)
	}

	vars := r.GetVars()
	if vars["foo"] != "bar" {
		t.Errorf("Expected 'bar' in vars map")
	}
}

func TestRegisterAlias(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true})
	r.TemplateEngine = template.New(false)
	mock := &mockCommand{name: "real"}
	r.RegisterCommand(mock)
	r.RegisterAlias("fake", "real")

	err := r.ExecuteCommand(context.Background(), ":fake arg1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !mock.executed {
		t.Errorf("Aliased command was not executed")
	}
}

func TestREPLCompletion(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true})
	r.TemplateEngine = template.New(false)
	r.RegisterCommand(&mockCommand{name: "status"})
	r.RegisterCommand(&mockCommand{name: "send"})
	r.RegisterAlias("quit", "exit")

	tests := []struct {
		name     string
		line     string
		pos      int
		wantSugg []string
	}{
		{
			name:     "Complete status",
			line:     ":st",
			pos:      3,
			wantSugg: []string{"atus"},
		},
		{
			name:     "Complete send",
			line:     ":se",
			pos:      3,
			wantSugg: []string{"nd"},
		},
		{
			name:     "Complete alias",
			line:     ":qu",
			pos:      3,
			wantSugg: []string{"it"},
		},
		{
			name:     "No colon",
			line:     "st",
			pos:      2,
			wantSugg: nil,
		},
		{
			name:     "Empty line",
			line:     "",
			pos:      0,
			wantSugg: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sugg, _ := r.Do([]rune(tt.line), tt.pos)
			if len(sugg) != len(tt.wantSugg) {
				t.Fatalf("Got %d suggestions, want %d", len(sugg), len(tt.wantSugg))
			}
			for i, s := range sugg {
				if string(s) != tt.wantSugg[i] {
					t.Errorf("Got suggestion %q, want %q", string(s), tt.wantSugg[i])
				}
			}
		})
	}
}

func TestSessionCommands(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true})
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	t.Run("set and get", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":set mykey myvalue")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if r.GetVar("mykey") != "myvalue" {
			t.Errorf("Expected 'myvalue', got %v", r.GetVar("mykey"))
		}

		err = r.ExecuteCommand(context.Background(), ":get mykey")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("vars", func(t *testing.T) {
		r.SetVar("a", "1")
		r.SetVar("b", "2")
		err := r.ExecuteCommand(context.Background(), ":vars")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("env", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":env")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("pwd", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":pwd")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		err = r.ExecuteCommand(context.Background(), ":pwd mydir")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		val := r.GetVar("mydir")
		if val == nil {
			t.Errorf("Expected variable 'mydir' to be set")
		}
		if _, ok := val.(string); !ok {
			t.Errorf("Expected 'mydir' to be a string, got %T", val)
		}
	})
}

func TestScriptingCommands(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true})
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()
	mock := &mockCommand{name: "say"}
	r.RegisterCommand(mock)

	t.Run("Script Alias with positional args", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":alias greet :say hello $1")
		if err != nil {
			t.Fatalf("Failed to register alias: %v", err)
		}
		
		err = r.ExecuteCommand(context.Background(), ":greet world")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mock.executed || len(mock.args) != 2 || mock.args[1] != "world" {
			t.Errorf("Alias expansion failed, got args: %v", mock.args)
		}
	})

	t.Run("Assert success", func(t *testing.T) {
		r.SetVar("val", "100")
		err := r.ExecuteCommand(context.Background(), ":assert {{eq .Vars.val \"100\"}}")
		if err != nil {
			t.Errorf("Assert failed: %v", err)
		}
	})

	t.Run("Assert failure", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":assert {{eq \"1\" \"2\"}} \"mismatch\"")
		if err == nil {
			t.Errorf("Expected assert to fail")
		}
	})

	t.Run("Wait command", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":wait 10ms")
		if err != nil {
			t.Errorf("Wait failed: %v", err)
		}
	})

	t.Run("Exit and Quit", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":exit")
		if err != ErrExit {
			t.Errorf("Expected ErrExit, got %v", err)
		}

		err = r.ExecuteCommand(context.Background(), ":quit")
		if err != ErrExit {
			t.Errorf("Expected ErrExit, got %v", err)
		}
	})
}

func TestREPLPrompt(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true, Prompt: "> "})
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	t.Run("Default prompt", func(t *testing.T) {
		r.renderPrompt()
		if r.GetPrompt() != "> " {
			t.Errorf("Expected '> ', got %q", r.GetPrompt())
		}
	})

	t.Run("Custom template prompt", func(t *testing.T) {
		r.SetVar("env", "prod")
		err := r.ExecuteCommand(context.Background(), ":prompt set \"[{{.Vars.env}}] > \"")
		if err != nil {
			t.Fatalf("Failed to set prompt: %v", err)
		}
		if r.GetPrompt() != "[prod] > " {
			t.Errorf("Expected '[prod] > ', got %q", r.GetPrompt())
		}
	})

	t.Run("Color template prompt", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":prompt set \"{{red \\\"!\\\"}} > \"")
		if err != nil {
			t.Fatalf("Failed to set prompt: %v", err)
		}
		// red "!" -> \033[31m!\033[0m
		expected := "\033[31m!\033[0m > "
		if r.GetPrompt() != expected {
			t.Errorf("Expected %q, got %q", expected, r.GetPrompt())
		}
	})

	t.Run("Reset prompt", func(t *testing.T) {
		err := r.ExecuteCommand(context.Background(), ":prompt reset")
		if err != nil {
			t.Fatalf("Failed to reset prompt: %v", err)
		}
		if r.GetPrompt() != "> " {
			t.Errorf("Expected '> ', got %q", r.GetPrompt())
		}
	})
}
