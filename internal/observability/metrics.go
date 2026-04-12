package observability

import (
	"net/http"

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
