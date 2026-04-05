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
		{Name: "regex_shorthand", Match: Matcher{Regex: "^user:[a-z0-9]+$"}},
		{Name: "regex_complex", Match: Matcher{Regex: `\d{3}-\d{3}-\d{4}`}},
		{Name: "glob_txt", Match: Matcher{Type: "glob", Pattern: "*.txt"}},
		{Name: "glob_deploy", Match: Matcher{Type: "glob", Pattern: "*deploy*"}},
		{Name: "glob_exact", Match: Matcher{Type: "glob", Pattern: "exact_match"}},
		{Name: "json_match", Match: Matcher{Type: "json", Pattern: ".id == 1"}},
		{Name: "jq_type_match", Match: Matcher{Type: "jq", Pattern: `.type == "release" and .env == "production"`}},
		{Name: "jq_shorthand_match", Match: Matcher{JQ: `.user.role == "admin"`}},
		{Name: "jq_array_match", Match: Matcher{JQ: `.tags | contains(["urgent"])`}},
	})

	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"text", "hello", []string{"text_match"}},
		{"regex", "foobar", []string{"regex_match"}},
		{"regex_shorthand match", "user:alex123", []string{"regex_shorthand"}},
		{"regex_shorthand with newline", "user:alex123\n", []string{"regex_shorthand"}},
		{"regex_shorthand no match", "user:ALEX123", nil},
		{"regex_complex match", "123-456-7890", []string{"regex_complex"}},
		{"regex_complex no match", "12-456-7890", nil},
		{"glob standard", "test.txt", []string{"glob_txt"}},
		{"glob with slash", "some/path/to/test.txt", []string{"glob_txt"}},
		{"glob substring match", "system is ready to deploy now", []string{"glob_deploy"}},
		{"glob newline match", "line1\ndeploying system\nline3", []string{"glob_deploy"}},
		{"glob no match substring", "system is ready", nil},
		{"glob exact match", "exact_match", []string{"glob_exact"}},
		{"glob exact no partial match", "exact_match_extra", nil},
		{"json", `{"id": 1}`, []string{"json_match"}},
		{"jq type match", `{"type": "release", "env": "production"}`, []string{"jq_type_match"}},
		{"jq type no match", `{"type": "release", "env": "staging"}`, nil},
		{"jq shorthand match", `{"user": {"role": "admin"}}`, []string{"jq_shorthand_match"}},
		{"jq shorthand no match", `{"user": {"role": "guest"}}`, nil},
		{"jq array match", `{"tags": ["urgent", "backup"]}`, []string{"jq_array_match"}},
		{"jq array no match", `{"tags": ["backup"]}`, nil},
		{"jq non-json input", `not json`, nil},
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
