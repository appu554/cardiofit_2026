package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector tracks KB-20 service metrics using Prometheus.
type Collector struct {
	RequestDuration *prometheus.HistogramVec
	RequestTotal    *prometheus.CounterVec
	CacheHits       prometheus.Counter
	CacheMisses     prometheus.Counter
	LabValidations *prometheus.CounterVec
	EGFRComputed    prometheus.Counter
	StratumQueries  prometheus.Counter
	EventsPublished *prometheus.CounterVec
}

// NewCollector creates and registers all KB-20 Prometheus metrics.
func NewCollector() *Collector {
	return &Collector{
		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "kb20_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "endpoint", "status"}),

		RequestTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb20_requests_total",
			Help: "Total HTTP requests",
		}, []string{"method", "endpoint", "status"}),

		CacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb20_cache_hits_total",
			Help: "Total cache hits",
		}),

		CacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb20_cache_misses_total",
			Help: "Total cache misses",
		}),

		LabValidations: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb20_lab_validations_total",
			Help: "Lab value validations by result",
		}, []string{"lab_type", "result"}),

		EGFRComputed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb20_egfr_computations_total",
			Help: "Total eGFR computations performed",
		}),

		StratumQueries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb20_stratum_queries_total",
			Help: "Total stratum activation queries",
		}),

		EventsPublished: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb20_events_published_total",
			Help: "Events published to event bus",
		}, []string{"event_type"}),
	}
}

// Timer helps measure operation duration.
type Timer struct {
	start time.Time
}

// StartTimer begins a timing measurement.
func StartTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Duration returns elapsed time in seconds.
func (t *Timer) Duration() float64 {
	return time.Since(t.start).Seconds()
}
