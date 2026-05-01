package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatBuiltin_Execute_NonStreaming(t *testing.T) {
	// Mock OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req openAIChatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "gpt-3.5-turbo", req.Model)
		assert.Len(t, req.Messages, 2) // System + User

		resp := openAIChatResponse{
			ID: "chatcmpl-123",
			Choices: []struct {
				Message      OpenAIChatMessage `json:"message"`
				Delta        OpenAIChatMessage `json:"delta"`
				FinishReason string            `json:"finish_reason"`
			}{
				{
					Message: OpenAIChatMessage{Role: "assistant", Content: "Hello! I am OpenAI."},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	builtin := &OpenAIChatBuiltin{}
	d := &Dispatcher{
		templateEngine: template.New(false),
		conn:           &mockConn{},
	}
	a := &Action{
		Model:  "gpt-3.5-turbo",
		APIURL: server.URL,
		APIKey: "test-key",
		System: "You are a helpful assistant.",
	}
	tmplCtx := template.NewContext()
	tmplCtx.Message = "Hi"

	err := builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)

	assert.Equal(t, "Hello! I am OpenAI.", tmplCtx.OpenAIReply)
	assert.Equal(t, "Hello! I am OpenAI.", tmplCtx.OllamaReply)
}

func TestOpenAIChatBuiltin_Execute_Streaming(t *testing.T) {
	// Mock OpenAI API with streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		chunks := []string{"Hello", "!", " I", " am", " OpenAI", "."}
		for _, chunk := range chunks {
			resp := openAIChatResponse{
				Choices: []struct {
					Message      OpenAIChatMessage `json:"message"`
					Delta        OpenAIChatMessage `json:"delta"`
					FinishReason string            `json:"finish_reason"`
				}{
					{
						Delta: OpenAIChatMessage{Content: chunk},
					},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			w.(http.Flusher).Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	builtin := &OpenAIChatBuiltin{}
	mc := &mockConn{}
	d := &Dispatcher{
		templateEngine: template.New(false),
		conn:           mc,
	}
	a := &Action{
		Model:   "gpt-3.5-turbo",
		APIURL:  server.URL,
		Stream:  "true",
		Respond: "{{.OpenAIReply}}",
	}
	tmplCtx := template.NewContext()
	tmplCtx.Message = "Hi"

	err := builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)

	assert.Equal(t, "Hello! I am OpenAI.", tmplCtx.OpenAIReply)
	
	// Check if all chunks were written to connection
	var fullText strings.Builder
	for _, m := range mc.messages {
		fullText.Write(m.Data)
	}
	assert.Equal(t, "Hello! I am OpenAI.", fullText.String())
}

func TestOpenAIChatBuiltin_History(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		
		resp := openAIChatResponse{
			Choices: []struct {
				Message      OpenAIChatMessage `json:"message"`
				Delta        OpenAIChatMessage `json:"delta"`
				FinishReason string            `json:"finish_reason"`
			}{
				{
					Message: OpenAIChatMessage{Role: "assistant", Content: fmt.Sprintf("Response to: %s", req.Messages[len(req.Messages)-1].Content)},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	builtin := &OpenAIChatBuiltin{}
	d := &Dispatcher{
		templateEngine: template.New(false),
		conn:           &mockConn{},
	}
	a := &Action{
		Model:      "gpt-3.5-turbo",
		APIURL:     server.URL,
		MaxHistory: 4, // 2 turns (user+assistant)
	}
	tmplCtx := template.NewContext()

	// First Turn
	tmplCtx.Message = "Msg 1"
	err := builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)

	// Second Turn
	tmplCtx.Message = "Msg 2"
	err = builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)

	// Third Turn - Should evict first turn
	tmplCtx.Message = "Msg 3"
	err = builtin.Execute(context.Background(), d, a, tmplCtx)
	require.NoError(t, err)

	// Verify history for conn1
	val, ok := builtin.histories.Load("mock-conn-id")
	require.True(t, ok)
	history := val.([]OpenAIChatMessage)
	
	// 6 - 4 = 2. index 2 to 5.
	// messages[2] = user="Msg 2"
	// messages[3] = assistant="Response to: Msg 2"
	// messages[4] = user="Msg 3"
	// messages[5] = assistant="Response to: Msg 3"
	
	assert.Len(t, history, 4)
	assert.Equal(t, "Msg 2", history[0].Content)
	assert.Equal(t, "Msg 3", history[2].Content)
}
