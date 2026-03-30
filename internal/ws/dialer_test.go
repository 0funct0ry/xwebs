package ws

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

var upgrader = websocket.Upgrader{}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		err = conn.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func subprotocolHandler(w http.ResponseWriter, r *http.Request) {
	upgraderWithSub := websocket.Upgrader{
		Subprotocols: []string{"mqtt", "v1.xwebs"},
	}
	conn, err := upgraderWithSub.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
}

func TestDial(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("Successful Connection", func(t *testing.T) {
		conn, err := Dial(ctx, u)
		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, u, conn.URL)
		defer conn.Close()
	})

	t.Run("Invalid Scheme", func(t *testing.T) {
		_, err := Dial(ctx, "http://example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scheme")
	})

	t.Run("Unreachable URL", func(t *testing.T) {
		_, err := Dial(ctx, "ws://localhost:9999")
		require.Error(t, err)
	})
}

func TestDialWithHeaders(t *testing.T) {
	headerKey := "X-Test-Header"
	headerValue := "X-Test-Value"

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerKey) != headerValue {
			http.Error(w, "missing header", http.StatusBadRequest)
			return
		}
		echoHandler(w, r)
	}))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	headers := make(http.Header)
	headers.Add(headerKey, headerValue)

	t.Run("Inject Headers", func(t *testing.T) {
		conn, err := Dial(ctx, u, WithHeaders(headers))
		require.NoError(t, err)
		defer conn.Close()
	})
}

func TestDialWithSubprotocols(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(subprotocolHandler))
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("Negotiate Subprotocol", func(t *testing.T) {
		conn, err := Dial(ctx, u, WithSubprotocols("v1.xwebs"))
		require.NoError(t, err)
		assert.Equal(t, "v1.xwebs", conn.NegotiatedSubprotocol)
		defer conn.Close()
	})
}
