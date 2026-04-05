package handler

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	content := `
variables:
  foo: bar
handlers:
  - name: "test_handler"
    match:
      type: "text"
      pattern: "hello"
    actions:
      - action: "shell"
        command: "echo hello"
    on_connect:
      - action: "send"
        message: "welcome"
`
	tmpfile, err := os.CreateTemp("", "handlers.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(content))
	require.NoError(t, err)
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, "bar", cfg.Variables["foo"])
	require.Len(t, cfg.Handlers, 1)
	assert.Equal(t, "test_handler", cfg.Handlers[0].Name)
	assert.Equal(t, "text", cfg.Handlers[0].Match.Type)
	assert.Equal(t, "hello", cfg.Handlers[0].Match.Pattern)
	require.Len(t, cfg.Handlers[0].Actions, 1)
	assert.Equal(t, "shell", cfg.Handlers[0].Actions[0].Type)
	assert.Equal(t, "echo hello", cfg.Handlers[0].Actions[0].Command)
	require.Len(t, cfg.Handlers[0].OnConnect, 1)
	assert.Equal(t, "send", cfg.Handlers[0].OnConnect[0].Type)
}

func TestValidateConfigErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing name",
			content: "handlers: [{match: {pattern: '*'}}]",
			wantErr: "missing a name",
		},
		{
			name:    "missing pattern, regex, jq, json_path, json_schema, template, binary, all, or any",
			content: "handlers: [{name: 'foo', actions: [{action: 'shell', command: 'ls'}]}]",
			wantErr: "missing a match condition (pattern, regex, jq, json_path, json_schema, template, binary, all, or any)",
		},
		{
			name:    "binary shorthand",
			content: `
handlers:
  - name: "binary_shorthand"
    match:
      binary: true
    actions:
      - action: "log"
        message: "matched"
`,
			wantErr: "", // Should not error
		},
		{
			name:    "template shorthand",
			content: `
handlers:
  - name: "template_shorthand"
    match:
      template: "{{ eq .Message 'ping' }}"
    actions:
      - action: "log"
        message: "matched"
`,
			wantErr: "", // Should not error
		},
		{
			name:    "jq shorthand",
			content: `
handlers:
  - name: "jq_shorthand"
    match:
      jq: ".type == 'ping'"
    actions:
      - action: "log"
        message: "matched"
`,
			wantErr: "", // Should not error
		},
		{
			name:    "json_path shorthand",
			content: `
handlers:
  - name: "json_path_shorthand"
    match:
      json_path: "user.id"
      equals: 123
    actions:
      - action: "log"
        message: "matched"
`,
			wantErr: "", // Should not error
		},
		{
			name:    "regex shorthand",
			content: `
handlers:
  - name: "regex_shorthand"
    match:
      regex: "^user:.*"
    actions:
      - action: "log"
        message: "matched"
`,
			wantErr: "", // Should not error
		},
		{
			name:    "json_schema shorthand",
			content: `
handlers:
  - name: "json_schema_shorthand"
    match:
      json_schema: "schema.json"
    actions:
      - action: "log"
        message: "matched"
`,
			wantErr: "", // Should not error
		},
		{
			name:    "missing actions and lifecycle",
			content: "handlers: [{name: 'foo', match: {pattern: '*'}}]",
			wantErr: "no actions or lifecycle events",
		},
		{
			name:    "invalid action type",
			content: "handlers: [{name: 'foo', match: {pattern: '*'}, actions: [{action: 'invalid'}]}]",
			wantErr: "unknown action type: invalid",
		},
		{
			name:    "missing shell command",
			content: "handlers: [{name: 'foo', match: {pattern: '*'}, actions: [{action: 'shell'}]}]",
			wantErr: "shell action missing command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, _ := os.CreateTemp("", "invalid_handlers.yaml")
			defer os.Remove(tmpfile.Name())
			_, _ = tmpfile.Write([]byte(tt.content))
			tmpfile.Close()

			_, err := LoadConfig(tmpfile.Name())
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
