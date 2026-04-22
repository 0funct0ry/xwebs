package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerRouting(t *testing.T) {
	opts := DefaultOptions()
	opts.Paths = []string{"/ws1", "/ws2"}
	opts.Port = 0 // Auto-assign

	srv, _ := New(
		WithPaths(opts.Paths),
	)

	// Manually setup the mux to test routing without starting the full server if needed,
	// but testing Start() is better for integration.
	// However, Start() blocks, so we run it in a goroutine.

	// Use httptest.NewServer to wrap the server's handler
	mux := http.NewServeMux()
	for _, path := range opts.Paths {
		pattern := path
		if path == "/" {
			pattern = "/{$}"
		}
		mux.HandleFunc(pattern, srv.serveWS)
	}
	mux.HandleFunc("/{$}", srv.serveStatus)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	url1 := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws1"
	url2 := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws2"
	urlUnknown := "ws" + strings.TrimPrefix(ts.URL, "http") + "/unknown"

	t.Run("Connect to /ws1", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(url1, nil)
		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	})

	t.Run("Connect to /ws2", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(url2, nil)
		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	})

	t.Run("Connect to unconfigured path /unknown", func(t *testing.T) {
		_, resp, err := websocket.DefaultDialer.Dial(urlUnknown, nil)
		assert.Error(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Visit root / for status page", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	})

	t.Run("Visit /ws1 with browser (non-ws)", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/ws1")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	})
}
