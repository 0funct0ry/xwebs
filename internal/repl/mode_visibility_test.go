package repl

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestServerModeVisibility(t *testing.T) {
	// Create a REPL in ServerMode
	out := &bytes.Buffer{}
	r, err := New(ServerMode, &Config{
		Terminal: false,
		Stdout:   nopWriteCloser{out},
	})
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}
	r.RegisterCommonCommands()

	t.Run("Hidden from help", func(t *testing.T) {
		out.Reset()
		err := r.ExecuteCommand(context.Background(), ":help")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		helpOutput := out.String()
		forbidden := []string{":set", ":get", ":vars"}
		for _, cmd := range forbidden {
			if strings.Contains(helpOutput, cmd) {
				t.Errorf("Help output should NOT contain %q in server mode", cmd)
			}
		}
	})

	t.Run("Hidden from completion", func(t *testing.T) {
		prefixes := []string{":se", ":ge", ":va"}
		for _, p := range prefixes {
			sugg, _ := r.DoContext([]rune(p), len(p))
			if len(sugg) > 0 {
				t.Errorf("Should have no suggestions for %q in server mode, got %v", p, sugg)
			}
		}
	})

	t.Run("Behavior diverted", func(t *testing.T) {
		cmds := []string{":set foo bar", ":get foo", ":vars"}
		for _, cmd := range cmds {
			out.Reset()
			err := r.ExecuteCommand(context.Background(), cmd)
			if err != nil {
				t.Errorf("Unexpected error for %q: %v", cmd, err)
			}
			msg := out.String()
			if !strings.Contains(msg, "is not available in server mode") {
				t.Errorf("Expected redirection message for %q, got: %q", cmd, msg)
			}
			if !strings.Contains(msg, "Use :kv set / :kv get") {
				t.Errorf("Expected mention of :kv in message for %q", cmd)
			}
		}
	})
}

func TestClientModeVisibility(t *testing.T) {
	// Create a REPL in ClientMode
	out := &bytes.Buffer{}
	r, err := New(ClientMode, &Config{
		Terminal: false,
		Stdout:   nopWriteCloser{out},
	})
	if err != nil {
		t.Fatalf("Failed to create REPL: %v", err)
	}
	r.RegisterCommonCommands()

	t.Run("Visible in help", func(t *testing.T) {
		out.Reset()
		err := r.ExecuteCommand(context.Background(), ":help")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		helpOutput := out.String()
		expected := []string{":set", ":get", ":vars"}
		for _, cmd := range expected {
			if !strings.Contains(helpOutput, cmd) {
				t.Errorf("Help output SHOULD contain %q in client mode", cmd)
			}
		}
	})

	t.Run("Visible in completion", func(t *testing.T) {
		prefixes := []struct {
			p    string
			want string
		}{
			{":se", "t"},
			{":ge", "t"},
			{":va", "rs"},
		}
		for _, tc := range prefixes {
			sugg, _ := r.DoContext([]rune(tc.p), len(tc.p))
			found := false
			for _, s := range sugg {
				if string(s) == tc.want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Should have suggestion %q for %q in client mode", tc.want, tc.p)
			}
		}
	})

	t.Run("Behavior normal", func(t *testing.T) {
		out.Reset()
		err := r.ExecuteCommand(context.Background(), ":set foo bar")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if r.GetVar("foo") != "bar" {
			t.Errorf("Expected variable 'foo' to be 'bar'")
		}

		out.Reset()
		err = r.ExecuteCommand(context.Background(), ":get foo")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(out.String(), "bar") {
			t.Errorf("Expected output to contain 'bar', got %q", out.String())
		}
	})
}

// nopWriteCloser is a wrapper for bytes.Buffer to implement io.WriteCloser
type nopWriteCloser struct {
	*bytes.Buffer
}

func (n nopWriteCloser) Close() error { return nil }
