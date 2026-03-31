package ws

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestConnection_GracefulFlush(t *testing.T) {
	upgrader := websocket.Upgrader{}
	messagesReceived := make([]string, 0)
	var mu sync.Mutex
	var closeCode int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				if ce, ok := err.(*websocket.CloseError); ok {
					mu.Lock()
					closeCode = ce.Code
					mu.Unlock()
				}
				return
			}
			if mt == websocket.TextMessage {
				mu.Lock()
				messagesReceived = append(messagesReceived, string(message))
				mu.Unlock()
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

	c := NewConnection(conn, url, nil, &DialOptions{Verbose: true})
	c.Start()

	// Send multiple messages
	numMsgs := 5
	for i := 0; i < numMsgs; i++ {
		_ = c.Write(&Message{Type: TextMessage, Data: []byte(fmt.Sprintf("msg-%d", i))})
	}

	// Close immediately after writing
	if err := c.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Give some time for messages to process on server
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(messagesReceived) != numMsgs {
		t.Errorf("expected %d messages, got %d: %v", numMsgs, len(messagesReceived), messagesReceived)
	}
	if closeCode != websocket.CloseNormalClosure {
		t.Errorf("expected close code %d, got %d", websocket.CloseNormalClosure, closeCode)
	}
}

func TestConnection_CustomClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var closeCode int
	var closeReason string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if ce, ok := err.(*websocket.CloseError); ok {
					mu.Lock()
					closeCode = ce.Code
					closeReason = ce.Text
					mu.Unlock()
				}
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

	c := NewConnection(conn, url, nil, &DialOptions{Verbose: true})
	c.Start()

	customCode := 4000
	customReason := "custom closure"
	if err := c.CloseWithCode(customCode, customReason); err != nil {
		t.Errorf("CloseWithCode() failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if closeCode != customCode {
		t.Errorf("expected close code %d, got %d", customCode, closeCode)
	}
	if closeReason != customReason {
		t.Errorf("expected close reason %q, got %q", customReason, closeReason)
	}
}

func TestConnection_RemoteClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	customCode := 4001
	customReason := "server directed close"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Send close frame and exit
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(customCode, customReason))
		conn.Close()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	var capturedCode int
	var capturedReason string
	var mu sync.Mutex
	done := make(chan struct{})

	c := NewConnection(conn, url, nil, &DialOptions{
		Verbose: true,
		OnDisconnect: func(code int, reason string) {
			mu.Lock()
			capturedCode = code
			capturedReason = reason
			mu.Unlock()
			close(done)
		},
	})
	c.Start()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for OnDisconnect")
	}

	mu.Lock()
	defer mu.Unlock()
	if capturedCode != customCode {
		t.Errorf("expected captured code %d, got %d", customCode, capturedCode)
	}
	if capturedReason != customReason {
		t.Errorf("expected captured reason %q, got %q", customReason, capturedReason)
	}
}

func TestConnection_AbnormalClose(t *testing.T) {
	upgrader := websocket.Upgrader{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Hard close without close frame
		conn.UnderlyingConn().Close()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	var capturedCode int
	var mu sync.Mutex
	done := make(chan struct{})

	c := NewConnection(conn, url, nil, &DialOptions{
		Verbose: true,
		OnDisconnect: func(code int, reason string) {
			mu.Lock()
			capturedCode = code
			mu.Unlock()
			close(done)
		},
	})
	c.Start()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for OnDisconnect")
	}

	mu.Lock()
	defer mu.Unlock()
	if capturedCode != websocket.CloseAbnormalClosure {
		t.Errorf("expected captured code %d (Abnormal), got %d", websocket.CloseAbnormalClosure, capturedCode)
	}
}
