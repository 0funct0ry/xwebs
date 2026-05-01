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

func TestOllamaChatBuiltin(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req ollamaChatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Simple echo-back response to verify context
		content := "I received: " + req.Messages[len(req.Messages)-1].Content
		if len(req.Messages) > 1 && req.Messages[0].Role == "system" {
			content += " (system: " + req.Messages[0].Content + ")"
		}
		
		resp := ollamaChatResponse{
			Message: OllamaChatMessage{Role: "assistant", Content: content},
			Done:    true,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	engine := template.New(false)
	reg := NewRegistry(ClientMode)
	mockConn := &mockConn{}

	d := NewDispatcher(reg, mockConn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, server.URL+"/api/chat")

	builtin := &OllamaChatBuiltin{}

	t.Run("first turn", func(t *testing.T) {
		tmplCtx := template.NewContext()
		action := &Action{
			Command: "ollama-chat",
			Model:   "llama2",
			Prompt:  "Hello",
			System:  "You are a helpful assistant",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, "I received: Hello (system: You are a helpful assistant)", tmplCtx.OllamaReply)
		
		// Verify history
		val, ok := builtin.histories.Load("mock-conn-id")
		require.True(t, ok)
		history := val.([]OllamaChatMessage)
		assert.Equal(t, 2, len(history))
		assert.Equal(t, "user", history[0].Role)
		assert.Equal(t, "Hello", history[0].Content)
		assert.Equal(t, "assistant", history[1].Role)
	})

	t.Run("second turn with history", func(t *testing.T) {
		tmplCtx := template.NewContext()
		action := &Action{
			Command: "ollama-chat",
			Model:   "llama2",
			Prompt:  "How are you?",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, "I received: How are you?", tmplCtx.OllamaReply)
		
		// Verify history grew
		val, ok := builtin.histories.Load("mock-conn-id")
		require.True(t, ok)
		history := val.([]OllamaChatMessage)
		assert.Equal(t, 4, len(history))
	})

	t.Run("max history limit", func(t *testing.T) {
		tmplCtx := template.NewContext()
		action := &Action{
			Command:    "ollama-chat",
			Model:      "llama2",
			Prompt:     "One more",
			MaxHistory: 2, // Only keep last turn (user + assistant)
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		
		// Verify history truncated
		val, ok := builtin.histories.Load("mock-conn-id")
		require.True(t, ok)
		history := val.([]OllamaChatMessage)
		assert.Equal(t, 2, len(history))
		assert.Equal(t, "user", history[0].Role)
		assert.Equal(t, "One more", history[0].Content)
	})
}

func TestOllamaEmbedBuiltin(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/embeddings", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req ollamaEmbedRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		resp := ollamaEmbedResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	engine := template.New(false)
	reg := NewRegistry(ClientMode)
	mockConn := &mockConn{}

	d := NewDispatcher(reg, mockConn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, server.URL+"/api/embeddings")

	builtin := &OllamaEmbedBuiltin{}

	t.Run("default input", func(t *testing.T) {
		tmplCtx := template.NewContext()
		tmplCtx.Message = "Hello"
		action := &Action{
			Command: "ollama-embed",
			Model:   "all-minilm",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, []float64{0.1, 0.2, 0.3}, tmplCtx.Embedding)
	})

	t.Run("custom input", func(t *testing.T) {
		tmplCtx := template.NewContext()
		action := &Action{
			Command: "ollama-embed",
			Model:   "all-minilm",
			Input:   "Custom text",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, []float64{0.1, 0.2, 0.3}, tmplCtx.Embedding)
	})
}
