package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
)

func TestSSEManager(t *testing.T) {
	configs := []handler.SSEStreamConfig{
		{
			Name: "test",
		},
	}

	mgr := NewSSEManager(configs, true, t.Logf)
	if mgr == nil {
		t.Fatal("failed to create SSEManager")
	}

	// Configure the stream via UpdateStreamConfig
	err := mgr.UpdateStreamConfig("test", "buffer", 10)
	if err != nil {
		t.Fatalf("UpdateStreamConfig failed: %v", err)
	}

	// Test SendToSSE with buffer
	err = mgr.SendToSSE("test", "event1", "data1", "id1")
	if err != nil {
		t.Fatalf("SendToSSE failed: %v", err)
	}

	info, ok := mgr.GetStreamInfo("test")
	if !ok {
		t.Fatal("stream not found")
	}
	if info.BufferDepth != 1 {
		t.Errorf("expected buffer depth 1, got %d", info.BufferDepth)
	}

	// Test HTTP consumption
	req := httptest.NewRequest("GET", "/sse/test", nil)
	w := httptest.NewRecorder()
	
	// We need a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	
	done := make(chan bool)
	go func() {
		mgr.HandleSSE("test")(w, req)
		done <- true
	}()

	// Wait a bit for the connection to be established and buffer to be flushed
	time.Sleep(100 * time.Millisecond)

	// Send another message
	err = mgr.SendToSSE("test", "event2", "data2", "id2")
	if err != nil {
		t.Fatalf("SendToSSE failed: %v", err)
	}

	// Give it some time to be delivered
	time.Sleep(100 * time.Millisecond)
	cancel() // Stop the handler
	<-done

	body := w.Body.String()
	if !strings.Contains(body, "event: event1") || !strings.Contains(body, "data: data1") {
		t.Errorf("missing first event in SSE output: %q", body)
	}
	if !strings.Contains(body, "event: event2") || !strings.Contains(body, "data: data2") {
		t.Errorf("missing second event in SSE output: %q", body)
	}
}

func TestSSEManager_NoStream(t *testing.T) {
	mgr := NewSSEManager(nil, false, nil)
	err := mgr.SendToSSE("nonexistent", "e", "d", "i")
	if err == nil {
		t.Error("expected error for nonexistent stream")
	}
}

func TestSSEManager_DropStrategy(t *testing.T) {
	configs := []handler.SSEStreamConfig{
		{
			Name: "drop",
		},
	}
	mgr := NewSSEManager(configs, false, nil)
	_ = mgr.UpdateStreamConfig("drop", "drop", 10)
	
	err := mgr.SendToSSE("drop", "e", "d", "i")
	if err != nil {
		t.Fatalf("SendToSSE failed: %v", err)
	}
	
	info, _ := mgr.GetStreamInfo("drop")
	if info.BufferDepth != 0 {
		t.Errorf("expected buffer depth 0 for drop strategy, got %d", info.BufferDepth)
	}
}
