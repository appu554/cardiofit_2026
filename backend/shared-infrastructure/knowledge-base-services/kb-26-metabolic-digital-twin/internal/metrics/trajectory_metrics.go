package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TrajectoryMetrics groups all Prometheus metrics for the MHRI trajectory engine.
type TrajectoryMetrics struct {
	ComputeDuration         prometheus.Histogram
	ConcordantDeterioration *prometheus.CounterVec
	DivergenceTotal         *prometheus.CounterVec
	LeadingIndicatorTotal   *prometheus.CounterVec
	DomainCrossingTotal     *prometheus.CounterVec
	InsufficientData        prometheus.Counter
	PersistTotal            *prometheus.CounterVec
}

// NewTrajectoryMetrics registers all metrics with the global registry and
// returns the collector. Call once at server init.
func NewTrajectoryMetrics() *TrajectoryMetrics {
	return &TrajectoryMetrics{
		ComputeDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb26_trajectory_compute_duration_ms",
			Help:    "Latency of ComputeDecomposedTrajectory in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		ConcordantDeterioration: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_concordant_deterioration_total",
			Help: "Number of patients flagged with concordant multi-domain deterioration",
		}, []string{"domains_count"}),
		DivergenceTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_divergence_total",
			Help: "Number of divergence pairs detected",
		}, []string{"improving_domain", "declining_domain"}),
		LeadingIndicatorTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_leading_indicator_total",
			Help: "Behavioral leading indicator fires by lagging domain",
		}, []string{"lagging_domain"}),
		DomainCrossingTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_domain_crossing_total",
			Help: "Domain category crossings by domain and direction",
		}, []string{"domain", "direction"}),
		InsufficientData: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb26_trajectory_insufficient_data_total",
			Help: "Trajectory requests blocked by <2 data points",
		}),
		PersistTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_persist_total",
			Help: "Trajectory history persistence outcomes",
		}, []string{"result"}),
	}
}
