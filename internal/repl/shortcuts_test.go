package repl

import (
	"testing"
)

func TestParseShortcut(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    rune
		wantErr bool
	}{
		{"Raw number", "15", 15, false},
		{"Ctrl+A", "Ctrl+A", 1, false},
		{"Ctrl-O", "Ctrl-O", 15, false},
		{"Ctrl+Z", "Ctrl+Z", 26, false},
		{"Lowercase ctrl", "ctrl+q", 17, false},
		{"Lowercase Ctrl-O", "Ctrl-o", 15, false},
		{"Empty", "", 0, true},
		{"Invalid Ctrl+", "Ctrl++", 0, true},
		{"Invalid char", "Ctrl+1", 0, true},
		{"Unsupported", "Alt+K", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShortcut(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseShortcut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseShortcut() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortcutExecution(t *testing.T) {
	r, _ := New(ClientMode, &Config{
		Terminal: true,
		Shortcuts: map[string]string{
			"Ctrl+K": ":clear",
			"Ctrl+S": ":send ",
		},
	})

	t.Run("Ctrl+K mapping", func(t *testing.T) {
		// Ctrl+K is 11
		newLine, newPos, ok := r.OnChange(nil, 0, 11)
		if !ok {
			t.Errorf("Expected shortcut to be handled")
		}
		if string(newLine) != ":clear" {
			t.Errorf("Expected :clear, got %s", string(newLine))
		}
		if newPos != len(":clear") {
			t.Errorf("Expected pos %d, got %d", len(":clear"), newPos)
		}
	})

	t.Run("Ctrl+S mapping with space", func(t *testing.T) {
		// Ctrl+S is 19
		newLine, newPos, ok := r.OnChange(nil, 0, 19)
		if !ok {
			t.Errorf("Expected shortcut to be handled")
		}
		if string(newLine) != ":send " {
			t.Errorf("Expected ':send ', got %q", string(newLine))
		}
		if newPos != len(":send ") {
			t.Errorf("Expected pos %d, got %d", len(":send "), newPos)
		}
	})

	t.Run("Unhandled key", func(t *testing.T) {
		_, _, ok := r.OnChange(nil, 0, 'A')
		if ok {
			t.Errorf("Expected key 'A' not to be handled")
		}
	})
}
