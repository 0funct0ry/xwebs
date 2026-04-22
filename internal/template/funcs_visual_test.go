package template

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisualFuncs(t *testing.T) {
	e := New(false)

	t.Run("counters", func(t *testing.T) {
		// Individual counters
		require.Equal(t, uint64(1), e.incrementCounter("req"))
		require.Equal(t, uint64(2), e.incrementCounter("req"))
		require.Equal(t, uint64(1), e.incrementCounter("msg"))
		require.Equal(t, uint64(1), e.incrementCounter("error"))
		require.Equal(t, uint64(1), e.incrementCounter("seq"))

		// Template execution
		res, err := e.Execute("test", "{{reqCounter}} {{reqCounter}} {{msgCounter}}", nil)
		require.NoError(t, err)
		assert.Equal(t, "3 4 2", res)
	})

	t.Run("randomEmoji", func(t *testing.T) {
		funcName := "randomEmoji"
		f, ok := e.funcs[funcName].(func() string)
		require.True(t, ok)

		val := f()
		assert.NotEmpty(t, val)

		found := false
		for _, e := range emojis {
			if e == val {
				found = true
				break
			}
		}
		assert.True(t, found, "emoji should be from the list")
	})

	t.Run("randomColor", func(t *testing.T) {
		funcName := "randomColor"
		f, ok := e.funcs[funcName].(func() string)
		require.True(t, ok)

		val := f()
		assert.NotEmpty(t, val)

		found := false
		for _, c := range colors {
			if c == val {
				found = true
				break
			}
		}
		assert.True(t, found, "color should be from the list")
	})

	t.Run("sessionAge", func(t *testing.T) {
		funcName := "sessionAge"
		f, ok := e.funcs[funcName].(func() time.Duration)
		require.True(t, ok)

		// Wait a bit to ensure age > 0
		time.Sleep(10 * time.Millisecond)
		val := f()
		assert.Greater(t, val, time.Duration(0))
	})

	t.Run("template integration", func(t *testing.T) {
		tmpl := `{{seq}} {{randomEmoji}} {{sessionAge | duration}}`
		res, err := e.Execute("test", tmpl, nil)
		require.NoError(t, err)

		parts := strings.Fields(res)
		require.Len(t, parts, 3)
		assert.Equal(t, "2", parts[0]) // seq was 1, now 2
		assert.NotEmpty(t, parts[1])
		assert.NotEmpty(t, parts[2])
	})
}
