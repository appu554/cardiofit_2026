package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Collector struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	chainTraversal  *prometheus.HistogramVec
	safetyCheck     *prometheus.HistogramVec
	comparison      *prometheus.HistogramVec
	cacheHits       *prometheus.CounterVec
}

func NewCollector() *Collector {
	return &Collector{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb25_requests_total",
				Help: "Total HTTP requests to KB-25 Lifestyle Knowledge Graph",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb25_request_duration_seconds",
				Help:    "HTTP request latency",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		chainTraversal: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb25_chain_traversal_duration_seconds",
				Help:    "Causal chain traversal latency",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5},
			},
			[]string{"target"},
		),
		safetyCheck: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb25_safety_check_duration_seconds",
				Help:    "Safety rule evaluation latency",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1},
			},
			[]string{},
		),
		comparison: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb25_comparison_duration_seconds",
				Help:    "Intervention comparison latency",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{},
		),
		cacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb25_cache_hits_total",
				Help: "Cache hit/miss counts",
			},
			[]string{"operation", "result"},
		),
	}
}

func (c *Collector) RecordRequest(method, path string, status int, duration float64) {
	c.requestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	c.requestDuration.WithLabelValues(method, path).Observe(duration)
}

func (c *Collector) RecordChainTraversal(target string, duration float64) {
	c.chainTraversal.WithLabelValues(target).Observe(duration)
}

func (c *Collector) RecordSafetyCheck(duration float64) {
	c.safetyCheck.WithLabelValues().Observe(duration)
}

func (c *Collector) RecordComparison(duration float64) {
	c.comparison.WithLabelValues().Observe(duration)
}

func (c *Collector) RecordCacheResult(operation, result string) {
	c.cacheHits.WithLabelValues(operation, result).Inc()
}
