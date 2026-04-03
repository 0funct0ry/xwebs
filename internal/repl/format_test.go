package repl

import (
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormattingState_FormatMessage(t *testing.T) {
	state := NewFormattingState()
	state.Color = "off" // Disable color for predictable output

	msg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(`{"foo": "bar", "num": 123}`),
		Metadata: ws.MessageMetadata{
			Timestamp: time.Unix(1711996800, 0), // 2024-04-01T18:40:00Z
			Direction: "received",
			Length:    27,
		},
	}

	t.Run("Default Raw", func(t *testing.T) {
		state.Color = "off"
		state.Format = FormatRaw
		formatted, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
		assert.Equal(t, "⬇ {\"foo\": \"bar\", \"num\": 123}", formatted)
	})

	t.Run("Raw Highlighted", func(t *testing.T) {
		state.Color = "on"
		state.Format = FormatRaw
		formatted, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
		// Check for some ANSI codes (Cyan for key, Green for string, Yellow for number)
		assert.Contains(t, formatted, "\x1b[36m\"foo\":\x1b[0m")
		assert.Contains(t, formatted, "\x1b[32m\"bar\"\x1b[0m")
		assert.Contains(t, formatted, "\x1b[33m123\x1b[0m")
	})

	t.Run("JSON Pretty", func(t *testing.T) {
		state.Color = "off"
		state.Format = FormatJSON
		formatted, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
		assert.Contains(t, formatted, "⬇ {")
		assert.Contains(t, formatted, "  \"foo\": \"bar\"")
	})

	t.Run("Hex Dump", func(t *testing.T) {
		state.Color = "off"
		state.Format = FormatHex
		formatted, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
		assert.Contains(t, formatted, "⬇ 00000000  7b 22 66 6f 6f 22 3a 20  22 62 61 72 22 2c 20 22")
	})

	t.Run("Filtering - Match", func(t *testing.T) {
		err := state.SetFilter(".num == 123")
		require.NoError(t, err)
		_, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
	})

	t.Run("Filtering - No Match", func(t *testing.T) {
		err := state.SetFilter(".num == 456")
		require.NoError(t, err)
		_, ok := state.FormatMessage(msg, nil, nil)
		assert.False(t, ok)
	})

	t.Run("Filtering - Regex Match", func(t *testing.T) {
		err := state.SetFilter("/bar/")
		require.NoError(t, err)
		_, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
	})

	t.Run("Verbose Mode", func(t *testing.T) {
		state.Verbose = true
		state.Format = FormatRaw
		err := state.SetFilter("off")
		require.NoError(t, err)
		formatted, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
		assert.Contains(t, formatted, "[text len=27 compress=false]")
	})

	t.Run("Timestamps", func(t *testing.T) {
		state.Timestamps = true
		state.TimestampUTC = true
		formatted, ok := state.FormatMessage(msg, nil, nil)
		assert.True(t, ok)
		assert.Contains(t, formatted, "2024-04-01T18:40:00.000Z")
	})
}
