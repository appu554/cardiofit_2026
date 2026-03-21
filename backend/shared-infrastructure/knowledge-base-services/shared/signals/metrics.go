package signals

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// SignalMetrics holds Prometheus metrics for the clinical signal pipeline.
type SignalMetrics struct {
	ConsumerMessagesTotal      *prometheus.CounterVec
	ConsumerProcessingDuration *prometheus.HistogramVec
	ConsumerErrorsTotal        *prometheus.CounterVec
	ConsumerLag                *prometheus.GaugeVec
	DLQMessagesTotal           *prometheus.CounterVec
	OutboxRelayPublishedTotal  *prometheus.CounterVec
	OutboxRelayPendingCount    prometheus.Gauge
}

// NewSignalMetrics registers and returns the pipeline metrics.
func NewSignalMetrics(reg prometheus.Registerer) *SignalMetrics {
	m := &SignalMetrics{
		ConsumerMessagesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "signal_consumer_messages_total",
			Help: "Total messages processed by signal consumers",
		}, []string{"consumer_group", "topic", "signal_type"}),
		ConsumerProcessingDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "signal_consumer_processing_duration_seconds",
			Help:    "Processing latency per signal",
			Buckets: prometheus.DefBuckets,
		}, []string{"consumer_group", "signal_type"}),
		ConsumerErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "signal_consumer_errors_total",
			Help: "Total consumer processing errors",
		}, []string{"consumer_group", "signal_type", "error_type"}),
		ConsumerLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "signal_consumer_lag",
			Help: "Consumer lag in messages",
		}, []string{"consumer_group", "topic", "partition"}),
		DLQMessagesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "signal_dlq_messages_total",
			Help: "Messages sent to dead letter queue",
		}, []string{"consumer_group", "signal_type"}),
		OutboxRelayPublishedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "outbox_relay_published_total",
			Help: "Outbox relay published events by topic",
		}, []string{"topic"}),
		OutboxRelayPendingCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "outbox_relay_pending_count",
			Help: "Unpublished outbox rows awaiting Kafka relay",
		}),
	}
	collectors := []prometheus.Collector{
		m.ConsumerMessagesTotal, m.ConsumerProcessingDuration,
		m.ConsumerErrorsTotal, m.ConsumerLag, m.DLQMessagesTotal,
		m.OutboxRelayPublishedTotal, m.OutboxRelayPendingCount,
	}
	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			var are prometheus.AlreadyRegisteredError
			if errors.As(err, &are) {
				// Re-use the previously registered collector (safe in tests).
				continue
			}
			panic(err)
		}
	}
	return m
}
