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

func TestConnectionPingPong(t *testing.T) {
	upgrader := websocket.Upgrader{}
	
	t.Run("automatic ping is sent", func(t *testing.T) {
		pingReceived := make(chan bool, 1)
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			conn.SetPingHandler(func(appData string) error {
				pingReceived <- true
				return nil
			})
			
			// Keep connection alive for a bit
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}))
		defer server.Close()

		url := strings.Replace(server.URL, "http", "ws", 1)
		opts := []DialOption{
			WithPingInterval(100 * time.Millisecond),
			WithPongWait(1 * time.Second),
		}

		conn, err := Dial(context.Background(), url, opts...)
		require.NoError(t, err)
		defer conn.Close()

		select {
		case <-pingReceived:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("ping not received by server")
		}
	})

	t.Run("connection closed after pong timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			// Do NOT respond with pong automatically
			conn.SetPingHandler(func(appData string) error {
				return nil
			})
			
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}))
		defer server.Close()

		url := strings.Replace(server.URL, "http", "ws", 1)
		opts := []DialOption{
			WithPingInterval(100 * time.Millisecond),
			WithPongWait(200 * time.Millisecond),
		}

		conn, err := Dial(context.Background(), url, opts...)
		require.NoError(t, err)
		defer conn.Close()

		// Wait for connection to be closed by engine
		time.Sleep(1 * time.Second)
		
		assert.True(t, conn.IsClosed(), "connection should be closed after pong timeout")
	})

	t.Run("manual ping work", func(t *testing.T) {
		pingPayload := "hello-ping"
		pingReceived := make(chan string, 1)
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			conn.SetPingHandler(func(appData string) error {
				pingReceived <- appData
				return nil
			})
			
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}))
		defer server.Close()

		url := strings.Replace(server.URL, "http", "ws", 1)
		conn, err := Dial(context.Background(), url)
		require.NoError(t, err)
		defer conn.Close()

		err = conn.Write(&Message{
			Type: PingMessage,
			Data: []byte(pingPayload),
		})
		require.NoError(t, err)

		select {
		case data := <-pingReceived:
			assert.Equal(t, pingPayload, data)
		case <-time.After(1 * time.Second):
			t.Fatal("manual ping not received by server")
		}
	})

	t.Run("received ping and pong visible on read channel", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			
			// Send a ping to the client
			_ = conn.WriteControl(websocket.PingMessage, []byte("server-ping"), time.Now().Add(time.Second))
			
			// Send a pong to the client
			_ = conn.WriteControl(websocket.PongMessage, []byte("server-pong"), time.Now().Add(time.Second))
			
			// Stay alive
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}))
		defer server.Close()

		url := strings.Replace(server.URL, "http", "ws", 1)
		conn, err := Dial(context.Background(), url)
		require.NoError(t, err)
		defer conn.Close()

		// Expect ping on read channel
		select {
		case msg := <-conn.Read():
			assert.Equal(t, PingMessage, msg.Type)
			assert.Equal(t, []byte("server-ping"), msg.Data)
		case <-time.After(1 * time.Second):
			t.Fatal("ping not received on read channel")
		}

		// Expect pong on read channel
		select {
		case msg := <-conn.Read():
			assert.Equal(t, PongMessage, msg.Type)
			assert.Equal(t, []byte("server-pong"), msg.Data)
		case <-time.After(1 * time.Second):
			t.Fatal("pong not received on read channel")
		}
	})
}
