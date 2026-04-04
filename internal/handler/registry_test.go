package handler

import (
	"testing"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_PrioritySorting(t *testing.T) {
	reg := NewRegistry()
	reg.AddHandlers([]Handler{
		{Name: "low", Priority: 1, Match: Matcher{Type: "text", Pattern: "ping"}},
		{Name: "high", Priority: 10, Match: Matcher{Type: "text", Pattern: "ping"}},
		{Name: "medium", Priority: 5, Match: Matcher{Type: "text", Pattern: "ping"}},
		{Name: "default", Priority: 0, Match: Matcher{Type: "text", Pattern: "ping"}},
	})

	handlers := reg.Handlers()
	require.Len(t, handlers, 4)
	assert.Equal(t, "high", handlers[0].Name)
	assert.Equal(t, "medium", handlers[1].Name)
	assert.Equal(t, "low", handlers[2].Name)
	assert.Equal(t, "default", handlers[3].Name)
}

func TestRegistry_Match(t *testing.T) {
	reg := NewRegistry()
	reg.AddHandlers([]Handler{
		{Name: "text_match", Match: Matcher{Type: "text", Pattern: "hello"}},
		{Name: "regex_match", Match: Matcher{Type: "regex", Pattern: "^foo.*"}},
		{Name: "glob_match", Match: Matcher{Type: "glob", Pattern: "*.txt"}},
		{Name: "json_match", Match: Matcher{Type: "json", Pattern: ".id == 1"}},
	})

	tests := []struct {
		name    string
		input   string
		expect []string
	}{
		{"text", "hello", []string{"text_match"}},
		{"regex", "foobar", []string{"regex_match"}},
		{"glob", "test.txt", []string{"glob_match"}},
		{"json", `{"id": 1}`, []string{"json_match"}},
		{"no match", "no match", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ws.Message{Data: []byte(tt.input)}
			matches, err := reg.Match(msg)
			require.NoError(t, err)
			
			var names []string
			for _, m := range matches {
				names = append(names, m.Name)
			}
			assert.Equal(t, tt.expect, names)
		})
	}
}
