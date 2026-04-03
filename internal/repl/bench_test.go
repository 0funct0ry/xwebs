package repl

import (
	"sort"
	"testing"
	"time"
)

func TestLatencyStats(t *testing.T) {
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}
	
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	mean := total / time.Duration(len(latencies))
	
	if mean != 30*time.Millisecond {
		t.Errorf("Expected mean 30ms, got %v", mean)
	}
	
	p50 := latencies[int(float64(len(latencies))*0.5)]
	if p50 != 30*time.Millisecond {
		t.Errorf("Expected P50 30ms, got %v", p50)
	}
}
