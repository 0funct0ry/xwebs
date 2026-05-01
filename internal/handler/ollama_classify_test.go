package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaClassifyBuiltin(t *testing.T) {
	b := &OllamaClassifyBuiltin{}
	assert.Equal(t, "ollama-classify", b.Name())

	t.Run("Validation", func(t *testing.T) {
		err := b.Validate(Action{Labels: FlexLabels{List: []string{}}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing labels list")

		err = b.Validate(Action{Labels: FlexLabels{List: []string{"label1"}}})
		assert.NoError(t, err)
	})

	t.Run("Execute Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/generate", r.URL.Path)

			var req ollamaRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "json", req.Format)
			assert.Contains(t, req.Prompt, "label1, label2")

			resp := ollamaResponse{
				Response: `{"label": "label1", "confidence": 0.99}`,
				Done:     true,
			}
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer server.Close()

		reg := NewRegistry(ServerMode)
		engine := template.New(false)
		d := NewDispatcher(reg, nil, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")
		tmplCtx := template.NewContext()
		tmplCtx.Message = "hello"

		action := &Action{
			Type:      "builtin",
			Command:   "ollama-classify",
			OllamaURL: server.URL,
			Model:     "test-model",
			Labels:    FlexLabels{List: []string{"label1", "label2"}},
		}

		err := b.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, "label1", tmplCtx.Label)
		assert.Equal(t, 0.99, tmplCtx.Confidence)
	})

	t.Run("Execute Fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ollamaResponse{
				Response: "This message is about label2.",
				Done:     true,
			}
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer server.Close()

		reg := NewRegistry(ServerMode)
		engine := template.New(false)
		d := NewDispatcher(reg, nil, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")
		tmplCtx := template.NewContext()
		tmplCtx.Message = "hello"

		action := &Action{
			Type:      "builtin",
			Command:   "ollama-classify",
			OllamaURL: server.URL,
			Labels:    FlexLabels{List: []string{"label1", "label2"}},
		}

		err := b.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, "label2", tmplCtx.Label)
		assert.Equal(t, 0.5, tmplCtx.Confidence)
	})
}
