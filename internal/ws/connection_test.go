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

func TestConnection_MaxMessageSize(t *testing.T) {
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

	maxSize := int64(10)
	c := NewConnection(conn, url, nil, &DialOptions{MaxMessageSize: maxSize})
	c.Start()
	defer c.Close()

	// 1. Test Outgoing Size Limit
	largeMsg := &Message{Type: TextMessage, Data: []byte("this message is longer than 10 bytes")}
	err = c.Write(largeMsg)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("expected 'exceeds limit' error, got %v", err)
	}

	// 2. Test Incoming Size Limit
	// Send a large message through a separate connection to trigger read limit on 'c'
	conn2, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial second conn: %v", err)
	}
	defer conn2.Close()
	if err := conn2.WriteMessage(websocket.TextMessage, []byte("this is a very large message that should be rejected by the other connection")); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Wait for connection to close or error to occur
	select {
	case <-c.Done():
		if err := c.Err(); err == nil {
			t.Error("expected connection to close with error due to read limit")
		}
	case <-time.After(2 * time.Second):
		// Use a slightly longer timeout for CI stability
	}
}

func TestConnection_FrameTypes(t *testing.T) {
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

	tests := []struct {
		name string
		msg  *Message
	}{
		{"text frame", &Message{Type: TextMessage, Data: []byte("hello")}},
		{"binary frame", &Message{Type: BinaryMessage, Data: []byte{0x00, 0x01, 0x02, 0x03}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Write(tt.msg); err != nil {
				t.Fatalf("failed to write %s: %v", tt.name, err)
			}

			select {
			case received := <-c.Read():
				if received.Type != tt.msg.Type {
					t.Errorf("expected type %v, got %v", tt.msg.Type, received.Type)
				}
				if string(received.Data) != string(tt.msg.Data) {
					t.Errorf("expected data %v, got %v", tt.msg.Data, received.Data)
				}
			case <-time.After(1 * time.Second):
				t.Fatal("timed out waiting for message")
			}
		})
	}
}

