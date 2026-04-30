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

func TestOllamaGenerateBuiltin(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/generate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req ollamaRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		if req.Stream {
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)
			
			resp1 := ollamaResponse{Response: "Hello", Done: false}
			_ = json.NewEncoder(w).Encode(resp1)
			
			resp2 := ollamaResponse{Response: " world", Done: true}
			_ = json.NewEncoder(w).Encode(resp2)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := ollamaResponse{Response: "Hello world", Done: true}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	engine := template.New(false)
	reg := NewRegistry(ClientMode)
	// Mock connection
	mockConn := &mockConn{}

	d := NewDispatcher(reg, mockConn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, server.URL+"/api/generate")

	t.Run("non-streaming", func(t *testing.T) {
		tmplCtx := template.NewContext()
		action := &Action{
			Command: "ollama-generate",
			Model:   "llama2",
			Prompt:  "Hi",
		}

		builtin := &OllamaGenerateBuiltin{}
		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, "Hello world", tmplCtx.OllamaReply)
	})

	t.Run("streaming", func(t *testing.T) {
		tmplCtx := template.NewContext()
		action := &Action{
			Command: "ollama-generate",
			Model:   "llama2",
			Prompt:  "Hi",
			Stream:  "true",
			Respond: "AI: {{.OllamaReply}}",
		}

		builtin := &OllamaGenerateBuiltin{}
		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, "Hello world", tmplCtx.OllamaReply)
		
		// Check that two messages were "sent"
		assert.Equal(t, 2, len(mockConn.messages))
		assert.Equal(t, "AI: Hello", string(mockConn.messages[0].Data))
		assert.Equal(t, "AI:  world", string(mockConn.messages[1].Data))
	})
}
