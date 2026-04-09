package repl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHighlighter_Paint(t *testing.T) {
	s := NewFormattingState()
	s.Color = "on"
	h := NewHighlighter(s)

	tests := []struct {
		name     string
		input    string
		contains []string // ANSI codes it should contain
	}{
		{
			name:  "JSON Object",
			input: `{"id": 1, "name": "alice", "ok": true}`,
			contains: []string{
				"\033[36m\"id\":\033[0m",
				"\033[33m1\033[0m",
				"\033[32m\"alice\"\033[0m",
				"\033[31mtrue\033[0m",
			},
		},
		{
			name:  "Go Template",
			input: `Hello {{ uuid }} world`,
			contains: []string{
				"\033[35m{{ uuid }}\033[0m",
			},
		},
		{
			name:  "Command with JSON",
			input: `:sendj {"a": 1}`,
			contains: []string{
				":sendj ",
				"\033[36m\"a\":\033[0m",
				"\033[33m1\033[0m",
			},
		},
		{
			name:  "Command with Template",
			input: `:set greeting {{ .user }}`,
			contains: []string{
				"\033[35m{{ .user }}\033[0m",
			},
		},
		{
			name:  "Bare message with mixed content",
			input: `{"msg": "{{ .name }}"}`,
			contains: []string{
				"\033[36m\"msg\":\033[0m",
				"\033[32m\"{{ .name }}\"\033[0m", // Template is inside a string, so it's treated as string by current logic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(h.Paint([]rune(tt.input), 0))
			for _, c := range tt.contains {
				assert.Contains(t, got, c)
			}
		})
	}
}

func TestHighlighter_ColorDisabled(t *testing.T) {
	s := NewFormattingState()
	s.Color = "off"
	h := NewHighlighter(s)

	input := `{"a": 1}`
	got := string(h.Paint([]rune(input), 0))
	assert.Equal(t, input, got)
}
