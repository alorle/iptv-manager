package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// StreamsActive tracks the number of active streams
	StreamsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "iptv_streams_active",
		Help: "Number of active streams",
	})

	// ClientsConnected tracks the total number of connected clients across all streams
	ClientsConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "iptv_clients_connected",
		Help: "Number of total clients connected",
	})

	// UpstreamReconnections tracks total reconnections per content ID
	UpstreamReconnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "iptv_upstream_reconnections_total",
		Help: "Total number of upstream reconnections",
	}, []string{"content_id"})

	// UpstreamErrors tracks upstream errors by content ID and error type
	UpstreamErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "iptv_upstream_errors_total",
		Help: "Total number of upstream errors",
	}, []string{"content_id", "error_type"})

	// CircuitBreakerState tracks the current state of circuit breakers
	// 0=closed, 1=open, 2=half-open
	CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "iptv_circuit_breaker_state",
		Help: "Current state of circuit breaker (0=closed, 1=open, 2=half-open)",
	}, []string{"content_id"})

	// CircuitBreakerTrips tracks how many times a circuit breaker transitioned to OPEN
	CircuitBreakerTrips = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "iptv_circuit_breaker_trips_total",
		Help: "Total number of times circuit breaker transitioned to OPEN state",
	}, []string{"content_id"})

	// HealthCheckFailures tracks health check failures
	HealthCheckFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "iptv_health_check_failures_total",
		Help: "Total number of health check failures",
	})
)

// SetCircuitBreakerState updates the circuit breaker state metric
// state should be one of: "CLOSED" (0), "OPEN" (1), "HALF-OPEN" (2)
func SetCircuitBreakerState(contentID, state string) {
	var value float64
	switch state {
	case "CLOSED":
		value = 0
	case "OPEN":
		value = 1
	case "HALF-OPEN":
		value = 2
	}
	CircuitBreakerState.WithLabelValues(contentID).Set(value)
}

// RecordUpstreamReconnection increments the reconnection counter for a content ID
func RecordUpstreamReconnection(contentID string) {
	UpstreamReconnections.WithLabelValues(contentID).Inc()
}

// RecordUpstreamError increments the error counter for a content ID and error type
func RecordUpstreamError(contentID, errorType string) {
	UpstreamErrors.WithLabelValues(contentID, errorType).Inc()
}

// RecordCircuitBreakerTrip increments the circuit breaker trip counter
func RecordCircuitBreakerTrip(contentID string) {
	CircuitBreakerTrips.WithLabelValues(contentID).Inc()
}

// RecordHealthCheckFailure increments the health check failure counter
func RecordHealthCheckFailure() {
	HealthCheckFailures.Inc()
}

// SetStreamsActive sets the number of active streams
func SetStreamsActive(count int) {
	StreamsActive.Set(float64(count))
}

// SetClientsConnected sets the total number of connected clients
func SetClientsConnected(count int) {
	ClientsConnected.Set(float64(count))
}
