package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all KB-22 Prometheus metrics.
type Collector struct {
	SessionsStarted       prometheus.Counter
	QuestionsAsked        prometheus.Counter
	SafetyFlagsRaised     *prometheus.CounterVec
	DifferentialConverged  prometheus.Counter
	PatanahiRate          *prometheus.GaugeVec
	CalibrationConcordance *prometheus.GaugeVec

	AnswerLatency       prometheus.Histogram
	SessionInitLatency  prometheus.Histogram
	KB20FetchDuration   prometheus.Histogram
	KB21FetchDuration   prometheus.Histogram
	KB23FetchDuration   prometheus.Histogram
	EntropyComputation  prometheus.Histogram

	SessionsByStatus *prometheus.GaugeVec
}

func NewCollector() *Collector {
	return &Collector{
		SessionsStarted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb22_sessions_started_total",
			Help: "Total HPI sessions created",
		}),
		QuestionsAsked: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb22_questions_asked_total",
			Help: "Total questions answered across all sessions",
		}),
		SafetyFlagsRaised: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb22_safety_flags_raised_total",
			Help: "Safety flags fired by severity",
		}, []string{"severity", "flag_id"}),
		DifferentialConverged: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb22_differential_converged_total",
			Help: "Sessions reaching convergence threshold",
		}),
		PatanahiRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kb22_patanahi_rate",
			Help: "Rolling pata-nahi rate per node",
		}, []string{"node_id"}),
		CalibrationConcordance: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kb22_calibration_concordance",
			Help: "Top-1 concordance per node and stratum",
		}, []string{"node_id", "stratum"}),

		AnswerLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb22_answer_latency_ms",
			Help:    "Answer processing time in milliseconds",
			Buckets: []float64{5, 10, 20, 30, 40, 50, 75, 100, 200},
		}),
		SessionInitLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb22_session_init_latency_ms",
			Help:    "Session initialisation time including KB fetches",
			Buckets: []float64{10, 20, 30, 40, 50, 75, 100, 200, 500},
		}),
		KB20FetchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb22_kb20_fetch_duration_ms",
			Help:    "KB-20 stratum query latency",
			Buckets: []float64{5, 10, 20, 30, 40, 50, 100},
		}),
		KB21FetchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb22_kb21_fetch_duration_ms",
			Help:    "KB-21 adherence/reliability query latency",
			Buckets: []float64{5, 10, 20, 30, 40, 50, 100},
		}),
		KB23FetchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb22_kb23_fetch_duration_ms",
			Help:    "KB-23 treatment perturbation query latency",
			Buckets: []float64{5, 10, 20, 30, 40, 50, 100},
		}),
		EntropyComputation: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb22_entropy_computation_ms",
			Help:    "Question ordering entropy computation time",
			Buckets: []float64{1, 2, 5, 10, 20, 50},
		}),

		SessionsByStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kb22_sessions_by_status",
			Help: "Current session count by status",
		}, []string{"status"}),
	}
}
