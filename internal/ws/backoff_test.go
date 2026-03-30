package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoff(t *testing.T) {
	initial := 1 * time.Second
	max := 10 * time.Second

	tests := []struct {
		attempt int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 10 * time.Second}, // Capped
		{10, 10 * time.Second}, // Capped
	}

	for _, tt := range tests {
		got := ExponentialBackoff(initial, max, tt.attempt)
		assert.Equal(t, tt.expected, got, "attempt %d", tt.attempt)
	}
}
