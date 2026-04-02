package replay

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
)

// Replayer handles playback of recorded WebSocket sessions.
type Replayer struct{}

// NewReplayer creates a new Replayer instance.
func NewReplayer() *Replayer {
	return &Replayer{}
}

// Replay reads a JSONL recording and re-sends messages with timing.
// It returns the number of messages sent and the number of messages received during playback.
func (rep *Replayer) Replay(ctx context.Context, conn *ws.Connection, filename string, speed float64, onProgress func(elapsed int64, dir string, msg string)) (int, int, error) {
	if conn == nil {
		return 0, 0, fmt.Errorf("no active connection")
	}

	f, err := os.Open(filename)
	if err != nil {
		return 0, 0, fmt.Errorf("opening replay file: %w", err)
	}
	defer f.Close()

	if speed <= 0 {
		speed = 1.0
	}

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, 0, fmt.Errorf("empty replay file")
	}

	// Validate header
	var header map[string]interface{}
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return 0, 0, fmt.Errorf("invalid recording header: %w", err)
	}
	if header["xwebs_recording"] == nil {
		return 0, 0, fmt.Errorf("missing recording format version")
	}

	startTime := time.Now()
	sentCount := 0
	var recvCount atomic.Int32
	var firstTS int64 = -1

	// Monitor incoming messages during replay using a dedicated subscription
	// to avoid competing with the main REPL read loop.
	done := make(chan struct{})
	readCh := conn.Subscribe()
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-readCh:
				if !ok {
					return
				}
				recvCount.Add(1)
			}
		}
	}()
	defer conn.Unsubscribe(readCh)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return sentCount, int(recvCount.Load()), ctx.Err()
		case <-conn.Done():
			return sentCount, int(recvCount.Load()), fmt.Errorf("connection closed during replay")
		default:
		}

		var entry struct {
			TS      int64  `json:"ts"`
			Dir     string `json:"dir"`
			Msg     string `json:"msg"`
			MsgBase64 string `json:"msg_b64"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // skip malformed entries
		}

		// Replay strictly sends messages from the recording marked as "send" (or "sent")
		if entry.Dir != "send" && entry.Dir != "sent" {
			continue
		}

		if firstTS == -1 {
			firstTS = entry.TS
		}

		// Calculate wait time relative to the first message's timestamp
		targetElapsed := time.Duration(float64(entry.TS-firstTS)/speed) * time.Millisecond
		actualElapsed := time.Since(startTime)
		wait := targetElapsed - actualElapsed

		if wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return sentCount, int(recvCount.Load()), ctx.Err()
			}
		}

		// Send message
		var msg *ws.Message
		if entry.MsgBase64 != "" {
			data, err := base64.StdEncoding.DecodeString(entry.MsgBase64)
			if err != nil {
				continue
			}
			msg = &ws.Message{Type: ws.BinaryMessage, Data: data}
		} else {
			msg = &ws.Message{Type: ws.TextMessage, Data: []byte(entry.Msg)}
		}

		if onProgress != nil {
			onProgress(entry.TS, "send", entry.Msg)
		}

		if err := conn.Write(msg); err != nil {
			return sentCount, int(recvCount.Load()), fmt.Errorf("write error: %w", err)
		}
		sentCount++
	}

	if err := scanner.Err(); err != nil {
		return sentCount, int(recvCount.Load()), fmt.Errorf("reading replay file: %w", err)
	}

	// Wait for a grace period to capture responses to the last messages.
	// Increased to 1000ms to handle higher latency or jitter.
	select {
	case <-time.After(1000 * time.Millisecond):
	case <-ctx.Done():
	}

	return sentCount, int(recvCount.Load()), nil
}
