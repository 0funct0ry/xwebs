package ws

import (
	"math"
	"time"
)

// ExponentialBackoff calculates the backoff duration for a given attempt.
// initial: the starting backoff duration.
// max: the maximum backoff duration.
// attempt: the current attempt number (starting at 0).
func ExponentialBackoff(initial, max time.Duration, attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate 2^attempt * initial
	backoff := float64(initial) * math.Pow(2, float64(attempt))

	// Cap at max
	if backoff > float64(max) {
		backoff = float64(max)
	}

	return time.Duration(backoff)
}
