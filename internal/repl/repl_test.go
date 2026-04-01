package repl

import (
	"context"
	"strings"
	"testing"
)

type mockCommand struct {
	name    string
	help    string
	executed bool
	args    []string
}

func (c *mockCommand) Name() string { return c.name }
func (c *mockCommand) Help() string { return c.help }
func (c *mockCommand) Execute(ctx context.Context, r *REPL, args []string) error {
	c.executed = true
	c.args = args
	return nil
}

func TestREPLCommands(t *testing.T) {
	r, err := New(ClientMode, nil)
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}

	mock := &mockCommand{name: "test", help: "test command"}
	r.RegisterCommand(mock)

	t.Run("Execute command", func(t *testing.T) {
		err := r.executeCommand(context.Background(), ":test arg1 arg2")
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
		err := r.executeCommand(context.Background(), ":unknown")
		if err == nil {
			t.Errorf("Expected error for unknown command")
		}
		if !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestREPLVars(t *testing.T) {
	r, _ := New(ClientMode, nil)
	
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
	r, _ := New(ClientMode, nil)
	mock := &mockCommand{name: "real"}
	r.RegisterCommand(mock)
	r.RegisterAlias("fake", "real")

	err := r.executeCommand(context.Background(), ":fake arg1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !mock.executed {
		t.Errorf("Aliased command was not executed")
	}
}

func TestREPLCompletion(t *testing.T) {
	r, _ := New(ClientMode, nil)
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
	r, _ := New(ClientMode, nil)
	r.RegisterCommonCommands()

	t.Run("set and get", func(t *testing.T) {
		err := r.executeCommand(context.Background(), ":set mykey myvalue")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if r.GetVar("mykey") != "myvalue" {
			t.Errorf("Expected 'myvalue', got %v", r.GetVar("mykey"))
		}

		err = r.executeCommand(context.Background(), ":get mykey")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("vars", func(t *testing.T) {
		r.SetVar("a", "1")
		r.SetVar("b", "2")
		err := r.executeCommand(context.Background(), ":vars")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("env", func(t *testing.T) {
		err := r.executeCommand(context.Background(), ":env")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
