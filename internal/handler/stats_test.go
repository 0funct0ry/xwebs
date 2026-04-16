package handler

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegistryStats(t *testing.T) {
	r := NewRegistry()

	// Record some executions
	r.RecordExecution("h1", 100*time.Millisecond, nil)
	r.RecordExecution("h2", 200*time.Millisecond, errors.New("fail"))
	r.RecordExecution("h3", 50*time.Millisecond, nil)

	total, errs := r.GetGlobalStats()
	assert.Equal(t, uint64(3), total)
	assert.Equal(t, uint64(1), errs)

	slowLog := r.GetSlowLog(10)
	assert.Equal(t, 3, len(slowLog))
	assert.Equal(t, "h2", slowLog[0].HandlerName)
	assert.Equal(t, 200*time.Millisecond, slowLog[0].Duration)
	assert.Equal(t, "fail", slowLog[0].Error)
	assert.Equal(t, "h1", slowLog[1].HandlerName)
	assert.Equal(t, "h3", slowLog[2].HandlerName)
}

func TestSlowLogLimit(t *testing.T) {
	r := NewRegistry()
	r.global.maxSlowLog = 2

	r.RecordExecution("h1", 100*time.Millisecond, nil)
	r.RecordExecution("h2", 200*time.Millisecond, nil)
	r.RecordExecution("h3", 150*time.Millisecond, nil)

	slowLog := r.GetSlowLog(10)
	assert.Equal(t, 2, len(slowLog))
	assert.Equal(t, "h2", slowLog[0].HandlerName)
	assert.Equal(t, "h3", slowLog[1].HandlerName)
}
