package replay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		// Artificial delay to test grace period (600ms)
		time.Sleep(600 * time.Millisecond)
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func TestReplayTimingAndReliability(t *testing.T) {
	// Start a local echo server
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create a recording with 10 seconds of leading silence
	tmpfile, err := os.CreateTemp("", "xwebs-recording-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	header := map[string]interface{}{
		"xwebs_recording": 1,
		"url":             wsURL,
		"started":         "2026-04-02T14:00:00Z",
		"version":         "0.1.0",
	}
	hData, _ := json.Marshal(header)
	if _, err := tmpfile.Write(append(hData, '\n')); err != nil {
		t.Fatalf("Failed to write to tmpfile: %v", err)
	}

	// Message with 10s delay (represented by ts: 10000)
	msg1 := map[string]interface{}{
		"ts":  10000,
		"dir": "send",
		"msg": `{"ping": "latency_test"}`,
	}
	mData1, _ := json.Marshal(msg1)
	if _, err := tmpfile.Write(append(mData1, '\n')); err != nil {
		t.Fatalf("Failed to write to tmpfile: %v", err)
	}

	// Another message 1s later
	msg2 := map[string]interface{}{
		"ts":  11000,
		"dir": "send",
		"msg": `{"ping": "grace_period_test"}`,
	}
	mData2, _ := json.Marshal(msg2)
	if _, err := tmpfile.Write(append(mData2, '\n')); err != nil {
		t.Fatalf("Failed to write to tmpfile: %v", err)
	}
	tmpfile.Close()

	// Dial the server
	ctx := context.Background()
	conn, err := ws.Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	rep := NewReplayer()

	start := time.Now()
	var events []int64

	// 0. Start a parallel reader to verify broadcast (multi-subscriber)
	// This mimics the main REPL read loop.
	var parallelRecv atomic.Int32
	parallelDone := make(chan struct{})
	go func() {
		defer close(parallelDone)
		readCh := conn.Read() // Primary subscriber
		for {
			select {
			case _, ok := <-readCh:
				if !ok {
					return
				}
				parallelRecv.Add(1)
			case <-ctx.Done():
				return
			}
		}
	}()

	sent, recv, err := rep.Replay(ctx, conn, tmpfile.Name(), 1.0, func(elapsed int64, dir string, msg string) {
		events = append(events, time.Since(start).Milliseconds())
	})

	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	duration := time.Since(start)

	// 1. Verify leading silence skip
	// The first message should happen almost immediately (< 500ms), not after 10s.
	if duration > 5*time.Second {
		t.Errorf("Replay took too long (%v), leading silence not skipped?", duration)
	}
	if len(events) > 0 && events[0] > 500 {
		t.Errorf("First message delayed by %dms, want < 500ms", events[0])
	}

	// 2. Verify relative timing
	// msg2 should be ~1s after msg1.
	if len(events) == 2 {
		diff := events[1] - events[0]
		if diff < 800 || diff > 1500 {
			t.Errorf("Relative timing off: %dms, want ~1000ms", diff)
		}
	}

	// 3. Verify recvCount accuracy (Grace Period)
	// We sent 2 messages to an echo server, we expect 2 received.
	if sent != 2 {
		t.Errorf("Expected 2 sent messages, got %d", sent)
	}
	if recv != 2 {
		t.Errorf("Expected 2 received messages (echoes) in Replayer, got %d", recv)
	}

	// 4. Verify broadcast success
	// The parallel reader should also have received both messages.
	select {
	case <-parallelDone:
	case <-time.After(100 * time.Millisecond):
		// This might happen if the connection wasn't closed yet.
		// In the test, we'll wait a bit.
	}
	if parallelRecv.Load() != 2 {
		t.Errorf("Expected 2 received messages in parallel reader, got %d", parallelRecv.Load())
	}
}
