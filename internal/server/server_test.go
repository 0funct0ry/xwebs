package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ServeStatus(t *testing.T) {
	s := New(WithPaths([]string{"/ws"}))
	
	// Test HTML response
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.serveStatus(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "xwebs server")
	assert.Contains(t, w.Body.String(), "RUNNING")
	assert.Contains(t, w.Body.String(), "/ws")

	// Test JSON response
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	s.serveStatus(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status": "running"`)
	assert.Contains(t, w.Body.String(), `"/ws"`)
}

func TestServer_WebSocketUpgrade(t *testing.T) {
	s := New(WithPaths([]string{"/ws"}))
	
	server := httptest.NewServer(http.HandlerFunc(s.serveWS))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Connect to the server
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
}

func TestServer_NonWSRequestToWSPath(t *testing.T) {
	s := New(WithPaths([]string{"/ws"}))
	
	server := httptest.NewServer(http.HandlerFunc(s.serveWS))
	defer server.Close()

	// Direct HTTP GET to /ws path should return status page
	resp, err := http.Get(server.URL + "/ws")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}

func TestServer_StartAndStop(t *testing.T) {
	s := New(WithPort(0)) // Random port
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Start(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	cancel()
	
	err := <-errChan
	assert.NoError(t, err)
}
