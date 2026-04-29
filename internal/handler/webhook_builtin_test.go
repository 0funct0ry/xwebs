package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookBuiltin(t *testing.T) {
	// 1. Create a mock HTTP server
	var mu sync.Mutex
	var lastMethod string
	var lastBody string
	var lastHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		lastMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		lastBody = string(body)
		lastHeaders = r.Header.Clone()

		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("received: " + lastBody))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("only POST allowed"))
		}
	}))
	defer server.Close()

	// 2. Setup Dispatcher and Context
	reg := NewRegistry(ServerMode)
	tmplEngine := template.New(false)
	d := &Dispatcher{
		registry:       reg,
		templateEngine: tmplEngine,
		verbose:        true,
		Log:            t.Logf,
		Error:          t.Logf,
	}

	tmplCtx := template.NewContext()
	tmplCtx.Message = "hello raw message"
	tmplCtx.Vars["server_url"] = server.URL

	builtin := &WebhookBuiltin{}

	t.Run("Default POST with raw message body", func(t *testing.T) {
		action := &Action{
			Type:    "builtin",
			Command: "webhook",
			URL:     "{{.Vars.server_url}}/callback",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)

		mu.Lock()
		method := lastMethod
		body := lastBody
		mu.Unlock()

		assert.Equal(t, "POST", method)
		assert.Equal(t, "hello raw message", body)
		assert.Equal(t, http.StatusOK, tmplCtx.HttpStatus)
		assert.Equal(t, "received: hello raw message", tmplCtx.HttpBody)
	})

	t.Run("Custom body template", func(t *testing.T) {
		action := &Action{
			Type:    "builtin",
			Command: "webhook",
			URL:     server.URL,
			Body:    "{\"msg\": \"{{.Message}}\"}",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)

		mu.Lock()
		method := lastMethod
		body := lastBody
		headers := lastHeaders.Clone()
		mu.Unlock()

		assert.Equal(t, "POST", method)
		assert.Equal(t, "{\"msg\": \"hello raw message\"}", body)
		assert.Equal(t, "application/json", headers.Get("Content-Type"))
	})

	t.Run("Custom headers", func(t *testing.T) {
		action := &Action{
			Type:    "builtin",
			Command: "webhook",
			URL:     server.URL,
			Headers: map[string]string{
				"X-Webhook-Source": "xwebs-test",
				"Content-Type":     "text/plain",
			},
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)

		mu.Lock()
		headers := lastHeaders.Clone()
		mu.Unlock()

		assert.Equal(t, "xwebs-test", headers.Get("X-Webhook-Source"))
		assert.Equal(t, "text/plain", headers.Get("Content-Type"))
	})

	t.Run("Timeout handling", func(t *testing.T) {
		hangingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer hangingServer.Close()

		action := &Action{
			Type:    "builtin",
			Command: "webhook",
			URL:     hangingServer.URL,
			Timeout: "10ms",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Non-2xx status doesn't return error", func(t *testing.T) {
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		defer errServer.Close()

		action := &Action{
			Type:    "builtin",
			Command: "webhook",
			URL:     errServer.URL,
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, tmplCtx.HttpStatus)
		assert.Equal(t, "not found", tmplCtx.HttpBody)
	})

	t.Run("Network failure returns error", func(t *testing.T) {
		action := &Action{
			Type:    "builtin",
			Command: "webhook",
			URL:     "http://localhost:12345/unreachable", // Assuming nothing is on this port
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "webhook request failed")
	})
}
