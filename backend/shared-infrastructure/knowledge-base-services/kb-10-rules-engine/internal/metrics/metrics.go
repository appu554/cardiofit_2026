// Package metrics provides Prometheus metrics collection for the rules engine
package metrics

import (
	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

// Collector provides metrics collection
type Collector struct {
	config *config.MetricsConfig

	// Rule evaluation metrics
	rulesEvaluated *prometheus.CounterVec
	rulesTriggered *prometheus.CounterVec

	// Performance metrics
	evaluationDuration prometheus.Histogram
	cachingHits        prometheus.Counter
	cachingMisses      prometheus.Counter

	// Alert metrics
	alertsCreated   *prometheus.CounterVec
	activeAlerts    prometheus.Gauge

	// Store metrics
	totalRules      prometheus.Gauge
	activeRules     prometheus.Gauge
	rulesByType     *prometheus.GaugeVec
	rulesByCategory *prometheus.GaugeVec

	// Health metrics
	healthStatus prometheus.Gauge
	lastReloadTime prometheus.Gauge
}

// NewCollector creates a new metrics collector
func NewCollector(cfg *config.MetricsConfig) *Collector {
	namespace := cfg.Namespace
	subsystem := cfg.Subsystem

	return &Collector{
		config: cfg,

		rulesEvaluated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rules_evaluated_total",
				Help:      "Total number of rules evaluated",
			},
			[]string{"type", "category"},
		),

		rulesTriggered: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rules_triggered_total",
				Help:      "Total number of rules triggered",
			},
			[]string{"rule_id", "severity"},
		),

		evaluationDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "evaluation_duration_ms",
				Help:      "Time taken to evaluate rules in milliseconds",
				Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
		),

		cachingHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
		),

		cachingMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
		),

		alertsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "alerts_created_total",
				Help:      "Total number of alerts created",
			},
			[]string{"severity", "category"},
		),

		activeAlerts: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_alerts",
				Help:      "Number of active (unacknowledged) alerts",
			},
		),

		totalRules: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "total_rules",
				Help:      "Total number of rules loaded",
			},
		),

		activeRules: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_rules",
				Help:      "Number of active rules",
			},
		),

		rulesByType: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rules_by_type",
				Help:      "Number of rules by type",
			},
			[]string{"type"},
		),

		rulesByCategory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rules_by_category",
				Help:      "Number of rules by category",
			},
			[]string{"category"},
		),

		healthStatus: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "health_status",
				Help:      "Health status (1 = healthy, 0 = unhealthy)",
			},
		),

		lastReloadTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "last_reload_timestamp",
				Help:      "Unix timestamp of last rules reload",
			},
		),
	}
}

// Register registers all metrics with Prometheus
func (c *Collector) Register() {
	if !c.config.Enabled {
		return
	}

	prometheus.MustRegister(
		c.rulesEvaluated,
		c.rulesTriggered,
		c.evaluationDuration,
		c.cachingHits,
		c.cachingMisses,
		c.alertsCreated,
		c.activeAlerts,
		c.totalRules,
		c.activeRules,
		c.rulesByType,
		c.rulesByCategory,
		c.healthStatus,
		c.lastReloadTime,
	)
}

// RecordRuleEvaluation records a rule evaluation
func (c *Collector) RecordRuleEvaluation(ruleType, category string, triggered bool) {
	if !c.config.Enabled {
		return
	}

	c.rulesEvaluated.WithLabelValues(ruleType, category).Inc()
}

// RecordRuleTriggered records a triggered rule
func (c *Collector) RecordRuleTriggered(ruleID, severity string) {
	if !c.config.Enabled {
		return
	}

	c.rulesTriggered.WithLabelValues(ruleID, severity).Inc()
}

// RecordEvaluationDuration records the evaluation duration
func (c *Collector) RecordEvaluationDuration(durationMs float64) {
	if !c.config.Enabled {
		return
	}

	c.evaluationDuration.Observe(durationMs)
}

// RecordCacheHit records a cache hit
func (c *Collector) RecordCacheHit() {
	if !c.config.Enabled {
		return
	}

	c.cachingHits.Inc()
}

// RecordCacheMiss records a cache miss
func (c *Collector) RecordCacheMiss() {
	if !c.config.Enabled {
		return
	}

	c.cachingMisses.Inc()
}

// RecordAlertCreated records an alert creation
func (c *Collector) RecordAlertCreated(severity, category string) {
	if !c.config.Enabled {
		return
	}

	c.alertsCreated.WithLabelValues(severity, category).Inc()
}

// SetActiveAlerts sets the number of active alerts
func (c *Collector) SetActiveAlerts(count int) {
	if !c.config.Enabled {
		return
	}

	c.activeAlerts.Set(float64(count))
}

// SetRuleStats updates rule statistics metrics
func (c *Collector) SetRuleStats(total, active int, byType, byCategory map[string]int) {
	if !c.config.Enabled {
		return
	}

	c.totalRules.Set(float64(total))
	c.activeRules.Set(float64(active))

	for t, count := range byType {
		c.rulesByType.WithLabelValues(t).Set(float64(count))
	}

	for cat, count := range byCategory {
		c.rulesByCategory.WithLabelValues(cat).Set(float64(count))
	}
}

// SetHealthStatus sets the health status
func (c *Collector) SetHealthStatus(healthy bool) {
	if !c.config.Enabled {
		return
	}

	if healthy {
		c.healthStatus.Set(1)
	} else {
		c.healthStatus.Set(0)
	}
}

// SetLastReloadTime sets the last reload timestamp
func (c *Collector) SetLastReloadTime(timestamp float64) {
	if !c.config.Enabled {
		return
	}

	c.lastReloadTime.Set(timestamp)
}
