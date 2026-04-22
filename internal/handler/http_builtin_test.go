package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpBuiltin(t *testing.T) {
	// 1. Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created " + r.Header.Get("X-Test")))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
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
	tmplCtx.Vars["server_url"] = server.URL

	builtin := &HttpBuiltin{}

	t.Run("GET request", func(t *testing.T) {
		action := &Action{
			Type:    "builtin",
			Command: "http",
			URL:     "{{.Vars.server_url}}/get",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, tmplCtx.HttpStatus)
		assert.Equal(t, "ok", tmplCtx.HttpBody)
	})

	t.Run("POST request with headers and body", func(t *testing.T) {
		action := &Action{
			Type:    "builtin",
			Command: "http",
			Method:  "POST",
			URL:     "{{.Vars.server_url}}/post",
			Headers: map[string]string{
				"X-Test": "val123",
			},
			Body: "test body",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, tmplCtx.HttpStatus)
		assert.Equal(t, "created val123", tmplCtx.HttpBody)
	})

	t.Run("Template in URL and Headers", func(t *testing.T) {
		tmplCtx.Vars["path"] = "dynamic-path"
		tmplCtx.Vars["hdr_val"] = "templated-header"
		
		action := &Action{
			Type:    "builtin",
			Command: "http",
			Method:  "POST",
			URL:     "{{.Vars.server_url}}/{{.Vars.path}}",
			Headers: map[string]string{
				"X-Test": "{{.Vars.hdr_val}}",
			},
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, tmplCtx.HttpStatus)
		assert.Equal(t, "created templated-header", tmplCtx.HttpBody)
	})

	t.Run("Timeout handling", func(t *testing.T) {
		// Server that hangs
		hangingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer hangingServer.Close()

		action := &Action{
			Type:    "builtin",
			Command: "http",
			URL:     hangingServer.URL,
			Timeout: "10ms",
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Respond template uses http results", func(t *testing.T) {
		reg := NewRegistry(ServerMode)
		// Register a mock connection that captures written messages
		mockConn := &httpMockConnection{
			writeFunc: func(m *ws.Message) error {
				return nil
			},
		}
		
		d := &Dispatcher{
			registry:       reg,
			templateEngine: tmplEngine,
			conn:           mockConn,
			verbose:        true,
			Log:            t.Logf,
			Error:          t.Logf,
		}

		h := &Handler{
			Name:    "http-test",
			Builtin: "http",
			URL:     server.URL + "/get",
			Respond: "Status: {{.HttpStatus}}, Body: {{.HttpBody}}",
		}

		err := d.Execute(context.Background(), h, &ws.Message{Data: []byte("hello")}, nil)
		require.NoError(t, err)
		
		require.NotEmpty(t, mockConn.msgs)
		assert.Equal(t, "Status: 200, Body: ok", string(mockConn.msgs[0].Data))
	})

	t.Run("Non-2xx status doesn't error", func(t *testing.T) {
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		}))
		defer errServer.Close()

		action := &Action{
			Type:    "builtin",
			Command: "http",
			URL:     errServer.URL,
		}

		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, tmplCtx.HttpStatus)
		assert.Equal(t, "server error", tmplCtx.HttpBody)
	})
}

type httpMockConnection struct {
	writeFunc func(m *ws.Message) error
	msgs      []*ws.Message
}

func (m *httpMockConnection) Write(msg *ws.Message) error {
	m.msgs = append(m.msgs, msg)
	if m.writeFunc != nil {
		return m.writeFunc(msg)
	}
	return nil
}
func (m *httpMockConnection) CloseWithCode(code int, reason string) error { return nil }
func (m *httpMockConnection) Subscribe() <-chan *ws.Message             { return nil }
func (m *httpMockConnection) Unsubscribe(ch <-chan *ws.Message)         {}
func (m *httpMockConnection) Done() <-chan struct{}                     { return nil }
func (m *httpMockConnection) IsCompressionEnabled() bool                { return false }
func (m *httpMockConnection) GetID() string                             { return "test-id" }
func (m *httpMockConnection) GetURL() string                            { return "ws://localhost" }
func (m *httpMockConnection) GetSubprotocol() string                    { return "" }
func (m *httpMockConnection) RemoteAddr() string                        { return "127.0.0.1" }
func (m *httpMockConnection) LocalAddr() string                         { return "127.0.0.1" }
func (m *httpMockConnection) ConnectedAt() time.Time                    { return time.Now() }
func (m *httpMockConnection) MessageCount() uint64                      { return 0 }
func (m *httpMockConnection) MsgsIn() uint64                            { return 0 }
func (m *httpMockConnection) MsgsOut() uint64                           { return 0 }
func (m *httpMockConnection) LastMsgReceivedAt() time.Time              { return time.Time{} }
func (m *httpMockConnection) LastMsgSentAt() time.Time                  { return time.Time{} }
func (m *httpMockConnection) RTT() time.Duration                        { return 0 }
func (m *httpMockConnection) AvgRTT() time.Duration                     { return 0 }
