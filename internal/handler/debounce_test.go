package handler

import (
	"sync"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
)

func TestRegistry_Debounce(t *testing.T) {
	reg := NewRegistry()
	msg1 := &ws.Message{Data: []byte("msg1")}
	msg2 := &ws.Message{Data: []byte("msg2")}
	msg3 := &ws.Message{Data: []byte("msg3")}

	t.Run("Single execution after delay", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		var received *ws.Message
		
		reg.Debounce("h1", 50*time.Millisecond, msg1, func(m *ws.Message) {
			received = m
			wg.Done()
		})
		
		wg.Wait()
		assert.Equal(t, msg1, received)
	})

	t.Run("Multiple messages reset timer (trailing-edge)", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		var received *ws.Message
		count := 0
		var mu sync.Mutex

		handlerFn := func(m *ws.Message) {
			mu.Lock()
			received = m
			count++
			mu.Unlock()
			wg.Done()
		}

		// Send 3 messages with 20ms gap, debounce is 50ms
		reg.Debounce("h2", 50*time.Millisecond, msg1, handlerFn)
		time.Sleep(20 * time.Millisecond)
		reg.Debounce("h2", 50*time.Millisecond, msg2, handlerFn)
		time.Sleep(20 * time.Millisecond)
		reg.Debounce("h2", 50*time.Millisecond, msg3, handlerFn)

		wg.Wait()
		
		mu.Lock()
		assert.Equal(t, 1, count, "Should only execute once")
		assert.Equal(t, msg3, received, "Should receive the most recent message")
		mu.Unlock()
	})

	t.Run("Independent debouncers", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)
		
		var m1, m2 *ws.Message
		
		reg.Debounce("h3", 50*time.Millisecond, msg1, func(m *ws.Message) {
			m1 = m
			wg.Done()
		})
		reg.Debounce("h4", 50*time.Millisecond, msg2, func(m *ws.Message) {
			m2 = m
			wg.Done()
		})
		
		wg.Wait()
		assert.Equal(t, msg1, m1)
		assert.Equal(t, msg2, m2)
	})
}
