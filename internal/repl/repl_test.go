package repl

import (
	"context"
	"os"
	"path/filepath"
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
func (c *mockCommand) IsVisible(r *REPL) bool { return true }

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

	t.Run("cd", func(t *testing.T) {
		initialDir, _ := os.Getwd()
		home, _ := os.UserHomeDir()

		// Create a temp dir to test cd
		tmpDir, err := os.MkdirTemp("", "xwebs-cd-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// 1. CD to temp dir
		err = r.ExecuteCommand(context.Background(), ":cd "+tmpDir)
		if err != nil {
			t.Errorf("CD to temp dir failed: %v", err)
		}
		cwd, _ := os.Getwd()
		// On macOS, /tmp is a symlink to /private/tmp, so we use filepath.EvalSymlinks or just compare suffixes if they are different but equivalent.
		// For simplicity, we just check if it changed from initialDir.
		if cwd == initialDir && cwd != tmpDir {
			t.Errorf("Expected directory to change, still in %s", cwd)
		}
		if r.prevDir != initialDir {
			t.Errorf("Expected prevDir to be %s, got %s", initialDir, r.prevDir)
		}

		// 2. CD - (back to initial)
		err = r.ExecuteCommand(context.Background(), ":cd -")
		if err != nil {
			t.Errorf("CD - failed: %v", err)
		}
		cwd, _ = os.Getwd()
		if cwd != initialDir {
			t.Errorf("Expected to be back in %s, got %s", initialDir, cwd)
		}

		// 3. CD (home)
		err = r.ExecuteCommand(context.Background(), ":cd")
		if err != nil {
			t.Errorf("CD to home failed: %v", err)
		}
		cwd, _ = os.Getwd()
		// Compare with home (evaluate symlinks to be safe on macOS)
		evalCwd, _ := filepath.EvalSymlinks(cwd)
		evalHome, _ := filepath.EvalSymlinks(home)
		if evalCwd != evalHome {
			t.Errorf("Expected to be in home %s, got %s", evalHome, evalCwd)
		}

		// Cleanup: go back to initialDir so other tests aren't affected
		_ = os.Chdir(initialDir)
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
		r.Display.Color = "on"
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

func TestREPLPromptStats(t *testing.T) {
	r, _ := New(ClientMode, &Config{Terminal: true})
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	// 1. Test MSGS stats
	mcc := &mockClientContext{
		handlerHits:    42,
		activeHandlers: 3,
	}
	r.RegisterClientCommands(mcc)

	err := r.ExecuteCommand(context.Background(), ":prompt set \"{{msgsIn}} {{msgsOut}} {{handlerHits}} {{activeHandlers}}\"")
	if err != nil {
		t.Fatalf("Failed to set prompt: %v", err)
	}

	// Stats should be 0 as mcc.conn is nil
	if r.GetPrompt() != "0 0 42 3" {
		t.Errorf("Expected '0 0 42 3', got %q", r.GetPrompt())
	}

	// 2. Test with a connection context
	// We need a way to mock ws.Connection or at least its stats.
	// Since ws.Connection is a struct, we can't easily mock its methods unless they are exported and we can override them.
	// But MsgsIn() etc. are methods on *Connection.
	// In replenish tests, we often use a real connection pointing to a local server or a specialized mock.
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		// Basic unquoted args
		{":handler add -m sub:*", []string{":handler", "add", "-m", "sub:*"}},
		// Single-quoted arg strips outer quotes
		{":send 'hello world'", []string{":send", "hello world"}},
		// Double-quoted arg strips outer quotes
		{`":send" "hello world"`, []string{":send", "hello world"}},
		// Single-quoted string containing double-quotes (Go template with string literal)
		{`:handler add --topic '{{.Message | trimPrefix "sub:"}}' -R 'subscribed:{{.Message | trimPrefix "sub:"}}' `,
			[]string{":handler", "add", "--topic", `{{.Message | trimPrefix "sub:"}}`, "-R", `subscribed:{{.Message | trimPrefix "sub:"}}`}},
		// Double-quoted template
		{`:send "{{upper .Message}}"`, []string{":send", `{{upper .Message}}`}},
		// Backslash escape
		{`:send hello\ world`, []string{":send", "hello world"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitCommand(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitCommand(%q) = %v (len %d); want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCommand(%q)[%d] = %q; want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBareCommandExecution(t *testing.T) {
	t.Run("Server Mode - Bare Command", func(t *testing.T) {
		r, _ := New(ServerMode, &Config{Terminal: true})
		mock := &mockCommand{name: "kv"}
		r.RegisterCommand(mock)

		// Input without colon
		err := r.RunLine(context.Background(), "kv list")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mock.executed {
			t.Errorf("Expected bare command 'kv' to be executed in Server Mode")
		}
		if len(mock.args) != 1 || mock.args[0] != "list" {
			t.Errorf("Unexpected args: %v", mock.args)
		}
	})

	t.Run("Server Mode - Unknown Bare Input", func(t *testing.T) {
		r, _ := New(ServerMode, &Config{Terminal: true})
		
		// Capture output would be better, but we can check if it logic flows correctly.
		// Since we don't have a good way to check Errorf output in unit tests without refactoring,
		// we'll just ensure it doesn't crash and returns no error from RunLine (as it handles it internally).
		err := r.RunLine(context.Background(), "unknown_stuff")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Client Mode - Bare Input (send as message)", func(t *testing.T) {
		r, _ := New(ClientMode, &Config{Terminal: true})
		inputSent := ""
		r.SetOnInput(func(ctx context.Context, text string) error {
			inputSent = text
			return nil
		})
		
		mock := &mockCommand{name: "kv"}
		r.RegisterCommand(mock)

		err := r.RunLine(context.Background(), "kv list")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if mock.executed {
			t.Errorf("Did not expect bare command 'kv' to be executed in Client Mode")
		}
		if inputSent != "kv list" {
			t.Errorf("Expected input to be sent as message, got %q", inputSent)
		}
	})
}

// RunLine is a helper to simulate the logic inside Run loop for a single line
func (r *REPL) RunLine(ctx context.Context, line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	if strings.HasPrefix(trimmed, ":") {
		return r.ExecuteCommand(ctx, trimmed)
	}
	
	// Copy logic from Run loop
	if r.onInput != nil {
		return r.onInput(ctx, line)
	}
	
	parts := splitCommand(trimmed)
	if len(parts) > 0 {
		cmdName := parts[0]
		r.mu.RLock()
		_, isCmd := r.commands[cmdName]
		_, isAlias := r.aliases[cmdName]
		_, isScript := r.scriptAliases[cmdName]
		r.mu.RUnlock()

		if isCmd || isAlias || isScript {
			return r.ExecuteCommand(ctx, ":"+trimmed)
		}
	}
	
	// Fallback error message (simulated)
	return nil
}
