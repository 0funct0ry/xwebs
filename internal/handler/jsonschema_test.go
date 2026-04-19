package handler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchemaMatcher(t *testing.T) {
	// Create a temporary schema file
	tmpDir, err := os.MkdirTemp("", "xwebs-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{
		"type": "object",
		"properties": {
			"type": { "enum": ["query"] },
			"id": { "type": "integer" }
		},
		"required": ["type", "id"]
	}`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	reg := NewRegistry(ServerMode)
	h := Handler{
		Name: "test-schema",
		Match: Matcher{
			JSONSchema: "schema.json",
		},
		Respond: "ok",
		BaseDir: tmpDir,
	}
	_ = reg.AddHandlers([]Handler{h})

	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "valid message",
			message: `{"type": "query", "id": 123}`,
			want:    true,
		},
		{
			name:    "missing required field",
			message: `{"type": "query"}`,
			want:    false,
		},
		{
			name:    "wrong type",
			message: `{"type": "query", "id": "not-an-int"}`,
			want:    false,
		},
		{
			name:    "invalid JSON",
			message: `{"type": "query", id: 123}`, // missing quotes around id
			want:    false,
		},
		{
			name:    "not an object",
			message: `123`,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ws.Message{
				Data: []byte(tt.message),
			}

			matches, err := reg.Match(msg, nil, template.NewContext())
			require.NoError(t, err)
			if tt.want {
				assert.Len(t, matches, 1)
				assert.Equal(t, "test-schema", matches[0].Handler.Name)
			} else {
				assert.Len(t, matches, 0)
			}
		})
	}
}

func TestJSONSchemaMatcherAbsolute(t *testing.T) {
	// Create a temporary schema file
	tmpDir, err := os.MkdirTemp("", "xwebs-test-abs")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{"type": "number"}`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	reg := NewRegistry(ServerMode)
	h := Handler{
		Name: "test-abs",
		Match: Matcher{
			JSONSchema: schemaPath,
		},
		Respond: "ok",
	}
	_ = reg.AddHandlers([]Handler{h})

	msg := &ws.Message{Data: []byte("123")}
	engine := template.New(false)
	ctx := template.NewContext()
	ctx.Message = "123"

	matches, err := reg.Match(msg, engine, ctx)
	require.NoError(t, err)
	assert.Len(t, matches, 1)
}
