package handler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
)

func TestHandler_ConcurrencyControl(t *testing.T) {
	engine := template.New(false)
	reg := NewRegistry(ServerMode)
	conn := &mockConn{}
	d := NewDispatcher(reg, conn, engine, false, nil, nil, false, nil, nil, nil, nil, nil)

	// Handler that sleeps for 100ms
	// Use a shell command to simulate work
	sleepCmd := "sleep 0.1"

	t.Run("Default (Concurrent)", func(t *testing.T) {
		h := &Handler{
			Name: "concurrent-handler",
			Run:  sleepCmd,
		}

		msg := &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}

		start := time.Now()
		var wg sync.WaitGroup
		numRequests := 3
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				defer wg.Done()
				err := d.Execute(context.Background(), h, msg, nil)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		// If concurrent, should take ~100ms (definitely less than 250ms)
		assert.Less(t, duration, 250*time.Millisecond, "Concurrent handlers took too long: %v", duration)
	})

	t.Run("Serialized (concurrent: false)", func(t *testing.T) {
		isConcurrent := false
		h := &Handler{
			Name:       "serialized-handler",
			Concurrent: &isConcurrent,
			Run:        sleepCmd,
		}

		msg := &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}

		start := time.Now()
		var wg sync.WaitGroup
		numRequests := 3
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				defer wg.Done()
				err := d.Execute(context.Background(), h, msg, nil)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		// If serialized, should take at least 300ms
		assert.GreaterOrEqual(t, duration, 300*time.Millisecond, "Serialized handlers were too fast: %v", duration)
	})

	t.Run("Independent Handlers", func(t *testing.T) {
		isConcurrent := false
		h1 := &Handler{
			Name:       "h1",
			Concurrent: &isConcurrent,
			Run:        sleepCmd,
		}
		h2 := &Handler{
			Name:       "h2",
			Concurrent: &isConcurrent,
			Run:        sleepCmd,
		}

		msg := &ws.Message{Data: []byte("test"), Metadata: ws.MessageMetadata{Direction: "received"}}

		start := time.Now()
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			err := d.Execute(context.Background(), h1, msg, nil)
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			err := d.Execute(context.Background(), h2, msg, nil)
			assert.NoError(t, err)
		}()

		wg.Wait()
		duration := time.Since(start)

		// Even if serialized individually, different handlers should run in parallel
		// So total time should be ~100ms
		assert.Less(t, duration, 250*time.Millisecond, "Independent serialized handlers blocked each other: %v", duration)
	})
}
