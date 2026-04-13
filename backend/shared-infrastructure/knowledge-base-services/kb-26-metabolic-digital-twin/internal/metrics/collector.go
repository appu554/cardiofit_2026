package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector provides Prometheus metrics for KB-26 Metabolic Digital Twin.
type Collector struct {
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	twinUpdate         *prometheus.HistogramVec
	simulationDuration *prometheus.HistogramVec
	calibrationDuration *prometheus.HistogramVec

	// BP context classification metrics (Phase 2)
	BPPhenotypeTotal  *prometheus.CounterVec
	BPClassifyLatency prometheus.Histogram
	BPClassifyErrors  prometheus.Counter

	// BP context batch metrics (Phase 3)
	BPBatchDuration      prometheus.Histogram
	BPBatchPatientsTotal *prometheus.CounterVec // labels: outcome (success|error)
	BPBatchErrors        prometheus.Counter

	// KB-19 publisher metrics (Phase 3)
	KB19PublishLatency prometheus.Histogram
	KB19PublishErrors  prometheus.Counter
}

func NewCollector() *Collector {
	return &Collector{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb26_requests_total",
				Help: "Total HTTP requests to KB-26 Metabolic Digital Twin",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb26_request_duration_seconds",
				Help:    "HTTP request latency",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		twinUpdate: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb26_twin_update_duration_seconds",
				Help:    "Twin state update latency by source",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"source"},
		),
		simulationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb26_simulation_duration_seconds",
				Help:    "Coupled simulation latency by intervention count",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
			},
			[]string{"intervention_count"},
		),
		calibrationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kb26_calibration_duration_seconds",
				Help:    "Bayesian calibration latency",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0},
			},
			[]string{},
		),
		BPPhenotypeTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb26_bp_phenotype_total",
				Help: "Total number of BP context classifications by phenotype",
			},
			[]string{"phenotype"},
		),
		BPClassifyLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kb26_bp_classify_latency_seconds",
				Help:    "Latency of BP context classification end-to-end",
				Buckets: prometheus.DefBuckets,
			},
		),
		BPClassifyErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "kb26_bp_classify_errors_total",
				Help: "Total number of BP context classification failures",
			},
		),
		BPBatchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb26_bp_batch_duration_seconds",
			Help:    "End-to-end duration of one BP context daily batch run",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s -> 1024s
		}),
		BPBatchPatientsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kb26_bp_batch_patients_total",
				Help: "Patients processed by the BP context batch, by outcome",
			},
			[]string{"outcome"},
		),
		BPBatchErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb26_bp_batch_errors_total",
			Help: "Number of fatal BP batch failures (does not include per-patient errors)",
		}),
		KB19PublishLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb26_kb19_publish_latency_seconds",
			Help:    "Latency of POST to KB-19 /api/v1/events from KB-26",
			Buckets: prometheus.DefBuckets,
		}),
		KB19PublishErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb26_kb19_publish_errors_total",
			Help: "Number of failed KB-19 event publishes",
		}),
	}
}

func (c *Collector) RecordRequest(method, path string, status int, duration float64) {
	c.requestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
	c.requestDuration.WithLabelValues(method, path).Observe(duration)
}

func (c *Collector) RecordTwinUpdate(source string, duration float64) {
	c.twinUpdate.WithLabelValues(source).Observe(duration)
}

func (c *Collector) RecordSimulation(interventionCount string, duration float64) {
	c.simulationDuration.WithLabelValues(interventionCount).Observe(duration)
}

func (c *Collector) RecordCalibration(duration float64) {
	c.calibrationDuration.WithLabelValues().Observe(duration)
}
