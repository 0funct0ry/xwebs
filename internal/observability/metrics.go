package observability

import (
	"net/http"
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// ActiveConnections tracks the number of currently connected clients.
	ActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xwebs_active_connections",
		Help: "Current number of active WebSocket connections.",
	})

	// MessagesReceived tracks the total number of messages received from clients.
	MessagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xwebs_messages_received_total",
		Help: "Total number of messages received from clients.",
	})

	// MessagesSent tracks the total number of messages sent to clients.
	MessagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xwebs_messages_sent_total",
		Help: "Total number of messages sent to clients.",
	})

	// HandlerExecutions tracks the total number of times handlers have been executed.
	HandlerExecutions = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "xwebs_handler_executions_total",
		Help: "Total number of handler executions.",
	})
)

func init() {
	// Register metrics with the default registry
	prometheus.MustRegister(ActiveConnections)
	prometheus.MustRegister(MessagesReceived)
	prometheus.MustRegister(MessagesSent)
	prometheus.MustRegister(HandlerExecutions)
}

// Handler returns the Prometheus HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

var (
	dynamicCounters = make(map[string]*prometheus.CounterVec)
	dynamicMu       sync.RWMutex
)

// IncrementCounter increments a dynamic counter by name and labels.
// The set of label keys for a given metric name must be consistent across calls.
func IncrementCounter(name string, labels map[string]string) {
	dynamicMu.RLock()
	cv, ok := dynamicCounters[name]
	dynamicMu.RUnlock()

	if !ok {
		dynamicMu.Lock()
		// Check again
		cv, ok = dynamicCounters[name]
		if !ok {
			labelKeys := make([]string, 0, len(labels))
			for k := range labels {
				labelKeys = append(labelKeys, k)
			}
			sort.Strings(labelKeys)
			cv = prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: name,
				Help: "Dynamic counter created by handler builtin action",
			}, labelKeys)
			// Using MustRegister might panic if the name is already used by a non-vec counter
			// or if label keys are inconsistent. We'll let it panic for now as it's a developer error
			// in the handler configuration.
			prometheus.MustRegister(cv)
			dynamicCounters[name] = cv
		}
		dynamicMu.Unlock()
	}

	cv.With(labels).Inc()
}
