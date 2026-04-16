package observability

import (
	"sync/atomic"
)

// GlobalStats tracks server-wide metrics.
type GlobalStats struct {
	TotalConnections uint64
	MessagesReceived uint64
	MessagesSent     uint64
	TotalErrors      uint64
}

var stats GlobalStats

// IncrementTotalConnections increments the total connection counter.
func IncrementTotalConnections() {
	atomic.AddUint64(&stats.TotalConnections, 1)
}

// IncrementMessagesReceived increments the received message counter.
func IncrementMessagesReceived() {
	atomic.AddUint64(&stats.MessagesReceived, 1)
}

// IncrementMessagesSent increments the sent message counter.
func IncrementMessagesSent() {
	atomic.AddUint64(&stats.MessagesSent, 1)
}

// IncrementTotalErrors increments the total error counter.
func IncrementTotalErrors() {
	atomic.AddUint64(&stats.TotalErrors, 1)
}

// GetGlobalStats returns a snapshot of global statistics.
func GetGlobalStats() GlobalStats {
	return GlobalStats{
		TotalConnections: atomic.LoadUint64(&stats.TotalConnections),
		MessagesReceived: atomic.LoadUint64(&stats.MessagesReceived),
		MessagesSent:     atomic.LoadUint64(&stats.MessagesSent),
		TotalErrors:      atomic.LoadUint64(&stats.TotalErrors),
	}
}

// ResetStats resets all global statistics (useful for tests).
func ResetStats() {
	atomic.StoreUint64(&stats.TotalConnections, 0)
	atomic.StoreUint64(&stats.MessagesReceived, 0)
	atomic.StoreUint64(&stats.MessagesSent, 0)
	atomic.StoreUint64(&stats.TotalErrors, 0)
}
