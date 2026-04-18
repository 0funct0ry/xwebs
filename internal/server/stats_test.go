package server

import (
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/observability"
	"github.com/stretchr/testify/assert"
)

func TestServerObservability(t *testing.T) {
	observability.ResetStats()
	s, _ := New(WithPort(0)) // random port

	// Verify initial stats
	stats := s.GetGlobalStats()
	assert.Equal(t, uint64(0), stats.TotalConnections)

	// Simulate a connection increment (manual for testing logic)
	// In real server, this happens in serveWS
	// We can't easily trigger serveWS without a full test server setup,
	// so we'll test the Registry integration which we've mostly already done.
	
	h := handler.Handler{
		Name: "slow-test",
		Run:  "sleep 0.1",
		Match: handler.Matcher{
			Pattern: "test",
		},
	}
	err := s.registry.Add(h)
	assert.NoError(t, err)
	
	s.registry.RecordExecution("slow-test", 150*time.Millisecond, nil)
	
	slowLog := s.GetSlowLog(10)
	assert.Equal(t, 1, len(slowLog))
	assert.Equal(t, "slow-test", slowLog[0].HandlerName)
	
	hHits, hErrs := s.GetRegistryStats()
	assert.Equal(t, uint64(1), hHits)
	assert.Equal(t, uint64(0), hErrs)
}
