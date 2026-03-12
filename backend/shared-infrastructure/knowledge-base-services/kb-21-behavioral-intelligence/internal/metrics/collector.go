package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector provides Prometheus metrics for KB-21.
type Collector struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	adherenceCompute *prometheus.HistogramVec
	loopTrustCompute *prometheus.HistogramVec
	hypoRiskEvents   *prometheus.CounterVec
	correlationCompute *prometheus.HistogramVec
	phenotypeChanges *prometheus.CounterVec
}

func NewCollector() *Collector {
	return &Collector{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb21_requests_total",
				Help: "Total HTTP requests to KB-21 Behavioral Intelligence",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb21_request_duration_seconds",
				Help:    "HTTP request latency",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		adherenceCompute: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb21_adherence_compute_duration_seconds",
				Help:    "Adherence score computation latency",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
			},
			[]string{"drug_class"},
		),
		loopTrustCompute: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb21_loop_trust_compute_duration_seconds",
				Help:    "Loop trust score computation latency",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05},
			},
			[]string{},
		),
		hypoRiskEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb21_hypo_risk_events_total",
				Help: "HYPO_RISK_ELEVATED events published",
			},
			[]string{"risk_level"},
		),
		correlationCompute: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb21_correlation_compute_duration_seconds",
				Help:    "OutcomeCorrelation computation latency",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{"response_class"},
		),
		phenotypeChanges: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb21_phenotype_changes_total",
				Help: "Phenotype transitions",
			},
			[]string{"from", "to"},
		),
	}
}

func (c *Collector) RecordRequest(method, path string, status int, duration float64) {
	c.requestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	c.requestDuration.WithLabelValues(method, path).Observe(duration)
}

func (c *Collector) RecordAdherenceCompute(drugClass string, duration float64) {
	c.adherenceCompute.WithLabelValues(drugClass).Observe(duration)
}

func (c *Collector) RecordHypoRiskEvent(riskLevel string) {
	c.hypoRiskEvents.WithLabelValues(riskLevel).Inc()
}

func (c *Collector) RecordCorrelationCompute(responseClass string, duration float64) {
	c.correlationCompute.WithLabelValues(responseClass).Observe(duration)
}

func (c *Collector) RecordPhenotypeChange(from, to string) {
	c.phenotypeChanges.WithLabelValues(from, to).Inc()
}
