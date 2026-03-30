package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestConnection_ReadWrite(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(mt, message); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	c := NewConnection(conn, url, nil, &DialOptions{})
	c.Start()
	defer c.Close()

	msg := &Message{Type: TextMessage, Data: []byte("hello")}
	if err := c.Write(msg); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	select {
	case received := <-c.Read():
		if string(received.Data) != "hello" {
			t.Errorf("expected 'hello', got %q", string(received.Data))
		}
		if received.Type != TextMessage {
			t.Errorf("expected TextMessage, got %v", received.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestConnection_ConcurrentWrite(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	c := NewConnection(conn, url, nil, &DialOptions{})
	c.Start()
	defer c.Close()

	numMsgs := 100
	errCh := make(chan error, numMsgs)

	for i := 0; i < numMsgs; i++ {
		go func(id int) {
			err := c.Write(&Message{Type: TextMessage, Data: []byte("msg")})
			errCh <- err
		}(i)
	}

	for i := 0; i < numMsgs; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent write failed: %v", err)
		}
	}
}

func TestConnection_Close(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	c := NewConnection(conn, url, nil, &DialOptions{})
	c.Start()

	if err := c.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if !c.IsClosed() {
		t.Error("expected connection to be marked closed")
	}

	// Verify writing to closed connection fails
	err = c.Write(&Message{Type: TextMessage, Data: []byte("test")})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected error writing to closed connection, got %v", err)
	}
}
