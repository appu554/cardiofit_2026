// Package tests provides comprehensive test utilities for KB-17 Population Registry
// metrics_observability_test.go - Tests for Prometheus metrics and observability
// This validates metrics accuracy critical for clinical quality dashboards and alerting
package tests

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

// =============================================================================
// METRICS REGISTRY
// =============================================================================

// MetricType represents the type of Prometheus metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// MetricValue represents a metric value with labels
type MetricValue struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

// HistogramValue represents histogram bucket data
type HistogramValue struct {
	Name       string
	Labels     map[string]string
	Sum        float64
	Count      uint64
	Buckets    map[float64]uint64 // le -> count
	Quantiles  map[float64]float64
	Timestamp  time.Time
}

// MockMetricsRegistry simulates Prometheus metrics registry
type MockMetricsRegistry struct {
	mu         sync.RWMutex
	counters   map[string]*MetricValue
	gauges     map[string]*MetricValue
	histograms map[string]*HistogramValue
}

// NewMockMetricsRegistry creates a new mock registry
func NewMockMetricsRegistry() *MockMetricsRegistry {
	return &MockMetricsRegistry{
		counters:   make(map[string]*MetricValue),
		gauges:     make(map[string]*MetricValue),
		histograms: make(map[string]*HistogramValue),
	}
}

// metricKey creates a unique key for metrics with labels
func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	var parts []string
	for k, v := range labels {
		// Prometheus format requires quoted label values: key="value"
		parts = append(parts, k+"=\""+v+"\"")
	}
	sort.Strings(parts)
	return name + "{" + strings.Join(parts, ",") + "}"
}

// IncCounter increments a counter
func (r *MockMetricsRegistry) IncCounter(name string, labels map[string]string) {
	r.AddCounter(name, 1, labels)
}

// AddCounter adds to a counter
func (r *MockMetricsRegistry) AddCounter(name string, value float64, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := metricKey(name, labels)
	if existing, ok := r.counters[key]; ok {
		existing.Value += value
		existing.Timestamp = time.Now()
	} else {
		r.counters[key] = &MetricValue{
			Name:      name,
			Type:      MetricTypeCounter,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// SetGauge sets a gauge value
func (r *MockMetricsRegistry) SetGauge(name string, value float64, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := metricKey(name, labels)
	r.gauges[key] = &MetricValue{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

// IncGauge increments a gauge
func (r *MockMetricsRegistry) IncGauge(name string, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := metricKey(name, labels)
	if existing, ok := r.gauges[key]; ok {
		existing.Value++
		existing.Timestamp = time.Now()
	} else {
		r.gauges[key] = &MetricValue{
			Name:      name,
			Type:      MetricTypeGauge,
			Value:     1,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// DecGauge decrements a gauge
func (r *MockMetricsRegistry) DecGauge(name string, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := metricKey(name, labels)
	if existing, ok := r.gauges[key]; ok {
		existing.Value--
		existing.Timestamp = time.Now()
	}
}

// ObserveHistogram records a histogram observation
func (r *MockMetricsRegistry) ObserveHistogram(name string, value float64, labels map[string]string, buckets []float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := metricKey(name, labels)
	if existing, ok := r.histograms[key]; ok {
		existing.Sum += value
		existing.Count++
		for _, bucket := range buckets {
			if value <= bucket {
				existing.Buckets[bucket]++
			}
		}
		existing.Timestamp = time.Now()
	} else {
		h := &HistogramValue{
			Name:      name,
			Labels:    labels,
			Sum:       value,
			Count:     1,
			Buckets:   make(map[float64]uint64),
			Timestamp: time.Now(),
		}
		for _, bucket := range buckets {
			if value <= bucket {
				h.Buckets[bucket] = 1
			} else {
				h.Buckets[bucket] = 0
			}
		}
		r.histograms[key] = h
	}
}

// GetCounter returns counter value
func (r *MockMetricsRegistry) GetCounter(name string, labels map[string]string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := metricKey(name, labels)
	if m, ok := r.counters[key]; ok {
		return m.Value
	}
	return 0
}

// GetGauge returns gauge value
func (r *MockMetricsRegistry) GetGauge(name string, labels map[string]string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := metricKey(name, labels)
	if m, ok := r.gauges[key]; ok {
		return m.Value
	}
	return 0
}

// GetHistogram returns histogram data
func (r *MockMetricsRegistry) GetHistogram(name string, labels map[string]string) *HistogramValue {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := metricKey(name, labels)
	return r.histograms[key]
}

// GetAllMetrics returns all metrics in Prometheus format
func (r *MockMetricsRegistry) GetAllMetrics() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lines []string

	// Counters
	for _, m := range r.counters {
		lines = append(lines, fmt.Sprintf("# TYPE %s counter", m.Name))
		lines = append(lines, fmt.Sprintf("%s %v", metricKey(m.Name, m.Labels), m.Value))
	}

	// Gauges
	for _, m := range r.gauges {
		lines = append(lines, fmt.Sprintf("# TYPE %s gauge", m.Name))
		lines = append(lines, fmt.Sprintf("%s %v", metricKey(m.Name, m.Labels), m.Value))
	}

	// Histograms
	for _, h := range r.histograms {
		lines = append(lines, fmt.Sprintf("# TYPE %s histogram", h.Name))
		for le, count := range h.Buckets {
			labels := make(map[string]string)
			for k, v := range h.Labels {
				labels[k] = v
			}
			labels["le"] = fmt.Sprintf("%v", le)
			lines = append(lines, fmt.Sprintf("%s_bucket%s %v", h.Name, formatLabels(labels), count))
		}
		lines = append(lines, fmt.Sprintf("%s_sum%s %v", h.Name, formatLabels(h.Labels), h.Sum))
		lines = append(lines, fmt.Sprintf("%s_count%s %v", h.Name, formatLabels(h.Labels), h.Count))
	}

	return strings.Join(lines, "\n")
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	sort.Strings(parts)
	return "{" + strings.Join(parts, ",") + "}"
}

// =============================================================================
// POPULATION METRICS COLLECTOR
// =============================================================================

// PopulationMetricsCollector collects KB-17 specific metrics
type PopulationMetricsCollector struct {
	registry *MockMetricsRegistry
}

// NewPopulationMetricsCollector creates new collector
func NewPopulationMetricsCollector(registry *MockMetricsRegistry) *PopulationMetricsCollector {
	return &PopulationMetricsCollector{registry: registry}
}

// RecordEnrollment records enrollment metric
func (c *PopulationMetricsCollector) RecordEnrollment(registryCode models.RegistryCode, source models.EnrollmentSource) {
	c.registry.IncCounter("kb17_enrollments_total", map[string]string{
		"registry": string(registryCode),
		"source":   string(source),
	})
	c.registry.IncGauge("kb17_active_enrollments", map[string]string{
		"registry": string(registryCode),
	})
}

// RecordDisenrollment records disenrollment metric
func (c *PopulationMetricsCollector) RecordDisenrollment(registryCode models.RegistryCode, reason string) {
	c.registry.IncCounter("kb17_disenrollments_total", map[string]string{
		"registry": string(registryCode),
		"reason":   reason,
	})
	c.registry.DecGauge("kb17_active_enrollments", map[string]string{
		"registry": string(registryCode),
	})
}

// RecordRiskTierChange records risk tier change
func (c *PopulationMetricsCollector) RecordRiskTierChange(registryCode models.RegistryCode, fromTier, toTier models.RiskTier) {
	c.registry.IncCounter("kb17_risk_tier_changes_total", map[string]string{
		"registry":  string(registryCode),
		"from_tier": string(fromTier),
		"to_tier":   string(toTier),
	})
}

// RecordAPILatency records API latency
func (c *PopulationMetricsCollector) RecordAPILatency(endpoint string, method string, durationMs float64) {
	buckets := []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000}
	c.registry.ObserveHistogram("kb17_api_request_duration_ms", durationMs, map[string]string{
		"endpoint": endpoint,
		"method":   method,
	}, buckets)
}

// RecordKafkaConsumerLag records consumer lag
func (c *PopulationMetricsCollector) RecordKafkaConsumerLag(topic string, partition int, lag int64) {
	c.registry.SetGauge("kb17_kafka_consumer_lag", float64(lag), map[string]string{
		"topic":     topic,
		"partition": fmt.Sprintf("%d", partition),
	})
}

// RecordCacheHitRate records cache statistics
func (c *PopulationMetricsCollector) RecordCacheHit(cacheName string) {
	c.registry.IncCounter("kb17_cache_hits_total", map[string]string{"cache": cacheName})
}

// RecordCacheMiss records cache miss
func (c *PopulationMetricsCollector) RecordCacheMiss(cacheName string) {
	c.registry.IncCounter("kb17_cache_misses_total", map[string]string{"cache": cacheName})
}

// SetPopulationSize sets current population size
func (c *PopulationMetricsCollector) SetPopulationSize(registryCode models.RegistryCode, riskTier models.RiskTier, count int64) {
	c.registry.SetGauge("kb17_population_size", float64(count), map[string]string{
		"registry":  string(registryCode),
		"risk_tier": string(riskTier),
	})
}

// RecordError records error metric
func (c *PopulationMetricsCollector) RecordError(operation string, errorType string) {
	c.registry.IncCounter("kb17_errors_total", map[string]string{
		"operation":  operation,
		"error_type": errorType,
	})
}

// =============================================================================
// PROMETHEUS SCRAPE TESTS
// =============================================================================

// TestMetrics_PrometheusScrapableFormat tests scrape format
func TestMetrics_PrometheusScrapableFormat(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Generate some metrics
	collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)
	collector.RecordAPILatency("/v1/enrollments", "POST", 45.5)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierHigh, 1500)

	// Get Prometheus format output
	output := registry.GetAllMetrics()

	// Verify Prometheus format compliance
	assert.Contains(t, output, "# TYPE", "Should have TYPE annotations")
	assert.Contains(t, output, "kb17_enrollments_total", "Should have enrollment counter")
	assert.Contains(t, output, "kb17_api_request_duration_ms", "Should have latency histogram")
	assert.Contains(t, output, "kb17_population_size", "Should have population gauge")

	// Verify label format
	assert.Contains(t, output, `registry="DIABETES"`, "Should have registry label")
}

// TestMetrics_AllRequiredMetricsExposed tests required metrics
func TestMetrics_AllRequiredMetricsExposed(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Generate all metric types
	collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)
	collector.RecordDisenrollment(models.RegistryDiabetes, "patient_request")
	collector.RecordRiskTierChange(models.RegistryDiabetes, models.RiskTierModerate, models.RiskTierHigh)
	collector.RecordAPILatency("/v1/enrollments", "POST", 50)
	collector.RecordKafkaConsumerLag("clinical.events", 0, 100)
	collector.RecordCacheHit("enrollments")
	collector.RecordCacheMiss("enrollments")
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierHigh, 500)
	collector.RecordError("enrollment", "validation")

	output := registry.GetAllMetrics()

	// Required metrics for clinical quality dashboards
	requiredMetrics := []string{
		"kb17_enrollments_total",
		"kb17_disenrollments_total",
		"kb17_risk_tier_changes_total",
		"kb17_api_request_duration_ms",
		"kb17_kafka_consumer_lag",
		"kb17_cache_hits_total",
		"kb17_cache_misses_total",
		"kb17_population_size",
		"kb17_errors_total",
	}

	for _, metric := range requiredMetrics {
		assert.Contains(t, output, metric,
			"Required metric %s should be exposed", metric)
	}
}

// =============================================================================
// ENROLLMENT METRICS TESTS
// =============================================================================

// TestMetrics_EnrollmentCountByRegistry tests enrollment counting per registry
func TestMetrics_EnrollmentCountByRegistry(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Enroll in different registries
	registries := []models.RegistryCode{
		models.RegistryDiabetes,
		models.RegistryDiabetes,
		models.RegistryHypertension,
		models.RegistryCKD,
		models.RegistryDiabetes,
	}

	for _, reg := range registries {
		collector.RecordEnrollment(reg, models.EnrollmentSourceDiagnosis)
	}

	// Verify counts
	diabetesCount := registry.GetCounter("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryDiabetes),
		"source":   string(models.EnrollmentSourceDiagnosis),
	})
	assert.Equal(t, float64(3), diabetesCount, "Diabetes enrollments should be 3")

	htnCount := registry.GetCounter("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryHypertension),
		"source":   string(models.EnrollmentSourceDiagnosis),
	})
	assert.Equal(t, float64(1), htnCount, "Hypertension enrollments should be 1")
}

// TestMetrics_EnrollmentSourceTracking tests source attribution
func TestMetrics_EnrollmentSourceTracking(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Different enrollment sources
	sources := []models.EnrollmentSource{
		models.EnrollmentSourceDiagnosis,
		models.EnrollmentSourceDiagnosis,
		models.EnrollmentSourceLabResult,
		models.EnrollmentSourceManual,
		models.EnrollmentSourceBulk,
	}

	for _, src := range sources {
		collector.RecordEnrollment(models.RegistryDiabetes, src)
	}

	// Verify by source
	diagnosisCount := registry.GetCounter("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryDiabetes),
		"source":   string(models.EnrollmentSourceDiagnosis),
	})
	assert.Equal(t, float64(2), diagnosisCount)

	manualCount := registry.GetCounter("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryDiabetes),
		"source":   string(models.EnrollmentSourceManual),
	})
	assert.Equal(t, float64(1), manualCount)
}

// TestMetrics_ActiveEnrollmentGauge tests active enrollment tracking
func TestMetrics_ActiveEnrollmentGauge(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Enroll 5 patients
	for i := 0; i < 5; i++ {
		collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)
	}

	active := registry.GetGauge("kb17_active_enrollments", map[string]string{
		"registry": string(models.RegistryDiabetes),
	})
	assert.Equal(t, float64(5), active)

	// Disenroll 2
	collector.RecordDisenrollment(models.RegistryDiabetes, "patient_request")
	collector.RecordDisenrollment(models.RegistryDiabetes, "exclusion_met")

	active = registry.GetGauge("kb17_active_enrollments", map[string]string{
		"registry": string(models.RegistryDiabetes),
	})
	assert.Equal(t, float64(3), active)
}

// =============================================================================
// RISK TIER METRICS TESTS
// =============================================================================

// TestMetrics_RiskTierDistribution tests risk tier tracking
func TestMetrics_RiskTierDistribution(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Set population sizes by risk tier
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierLow, 1000)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierModerate, 500)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierHigh, 200)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierCritical, 50)

	// Verify distribution
	low := registry.GetGauge("kb17_population_size", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"risk_tier": string(models.RiskTierLow),
	})
	assert.Equal(t, float64(1000), low)

	critical := registry.GetGauge("kb17_population_size", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"risk_tier": string(models.RiskTierCritical),
	})
	assert.Equal(t, float64(50), critical)
}

// TestMetrics_RiskTierChangeTracking tests tier change counting
func TestMetrics_RiskTierChangeTracking(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Record tier changes
	changes := []struct {
		from models.RiskTier
		to   models.RiskTier
	}{
		{models.RiskTierModerate, models.RiskTierHigh},
		{models.RiskTierModerate, models.RiskTierHigh},
		{models.RiskTierHigh, models.RiskTierCritical},
		{models.RiskTierHigh, models.RiskTierModerate}, // Improvement
	}

	for _, change := range changes {
		collector.RecordRiskTierChange(models.RegistryDiabetes, change.from, change.to)
	}

	// Verify escalations
	modToHigh := registry.GetCounter("kb17_risk_tier_changes_total", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"from_tier": string(models.RiskTierModerate),
		"to_tier":   string(models.RiskTierHigh),
	})
	assert.Equal(t, float64(2), modToHigh)

	// Verify improvements (de-escalation)
	highToMod := registry.GetCounter("kb17_risk_tier_changes_total", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"from_tier": string(models.RiskTierHigh),
		"to_tier":   string(models.RiskTierModerate),
	})
	assert.Equal(t, float64(1), highToMod)
}

// =============================================================================
// API LATENCY METRICS TESTS
// =============================================================================

// TestMetrics_APILatencyHistogram tests latency distribution
func TestMetrics_APILatencyHistogram(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Simulate various latencies
	latencies := []float64{5, 12, 25, 48, 75, 150, 250, 500, 1200, 3000}
	for _, lat := range latencies {
		collector.RecordAPILatency("/v1/enrollments", "POST", lat)
	}

	histogram := registry.GetHistogram("kb17_api_request_duration_ms", map[string]string{
		"endpoint": "/v1/enrollments",
		"method":   "POST",
	})

	require.NotNil(t, histogram)
	assert.Equal(t, uint64(10), histogram.Count)

	// Verify bucket distribution
	assert.True(t, histogram.Buckets[100] >= 5, "Should have at least 5 requests under 100ms")
	assert.True(t, histogram.Buckets[1000] >= 8, "Should have at least 8 requests under 1000ms")
}

// TestMetrics_APILatencyPerEndpoint tests per-endpoint tracking
func TestMetrics_APILatencyPerEndpoint(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Different endpoints
	collector.RecordAPILatency("/v1/enrollments", "POST", 50)
	collector.RecordAPILatency("/v1/enrollments", "GET", 20)
	collector.RecordAPILatency("/v1/registries", "GET", 10)
	collector.RecordAPILatency("/v1/stats", "GET", 100)

	// Verify separate tracking
	enrollPost := registry.GetHistogram("kb17_api_request_duration_ms", map[string]string{
		"endpoint": "/v1/enrollments",
		"method":   "POST",
	})
	assert.NotNil(t, enrollPost)
	assert.Equal(t, uint64(1), enrollPost.Count)

	enrollGet := registry.GetHistogram("kb17_api_request_duration_ms", map[string]string{
		"endpoint": "/v1/enrollments",
		"method":   "GET",
	})
	assert.NotNil(t, enrollGet)
	assert.Equal(t, uint64(1), enrollGet.Count)
}

// =============================================================================
// ALERT THRESHOLD TESTS
// =============================================================================

// AlertRule represents a Prometheus alert rule
type AlertRule struct {
	Name        string
	Expression  string
	For         time.Duration
	Severity    string
	Labels      map[string]string
	Annotations map[string]string
}

// AlertManager simulates alert evaluation
type AlertManager struct {
	registry *MockMetricsRegistry
	rules    []AlertRule
	alerts   []FiredAlert
	mu       sync.RWMutex
}

// FiredAlert represents a triggered alert
type FiredAlert struct {
	Rule      AlertRule
	Value     float64
	Timestamp time.Time
}

// NewAlertManager creates new alert manager
func NewAlertManager(registry *MockMetricsRegistry) *AlertManager {
	return &AlertManager{
		registry: registry,
		rules:    make([]AlertRule, 0),
		alerts:   make([]FiredAlert, 0),
	}
}

// AddRule adds an alert rule
func (a *AlertManager) AddRule(rule AlertRule) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules = append(a.rules, rule)
}

// EvaluateRules evaluates all rules against current metrics
func (a *AlertManager) EvaluateRules() []FiredAlert {
	a.mu.Lock()
	defer a.mu.Unlock()

	var fired []FiredAlert
	for _, rule := range a.rules {
		// Simplified evaluation based on rule expression patterns
		if a.evaluateRule(rule) {
			alert := FiredAlert{
				Rule:      rule,
				Timestamp: time.Now(),
			}
			fired = append(fired, alert)
			a.alerts = append(a.alerts, alert)
		}
	}
	return fired
}

func (a *AlertManager) evaluateRule(rule AlertRule) bool {
	// Simplified expression evaluation for testing
	switch {
	case strings.Contains(rule.Expression, "kb17_errors_total"):
		// Check error rate threshold
		errors := a.registry.GetCounter("kb17_errors_total", map[string]string{
			"operation":  "enrollment",
			"error_type": "validation",
		})
		return errors > 10

	case strings.Contains(rule.Expression, "kb17_kafka_consumer_lag"):
		// Check consumer lag threshold
		lag := a.registry.GetGauge("kb17_kafka_consumer_lag", map[string]string{
			"topic":     "clinical.events",
			"partition": "0",
		})
		return lag > 1000

	case strings.Contains(rule.Expression, "kb17_api_request_duration"):
		// Check latency threshold
		histogram := a.registry.GetHistogram("kb17_api_request_duration_ms", map[string]string{
			"endpoint": "/v1/enrollments",
			"method":   "POST",
		})
		if histogram != nil && histogram.Count > 0 {
			avgLatency := histogram.Sum / float64(histogram.Count)
			return avgLatency > 500
		}
		return false

	case strings.Contains(rule.Expression, "kb17_population_critical"):
		// Check critical population threshold
		critical := a.registry.GetGauge("kb17_population_size", map[string]string{
			"registry":  string(models.RegistryDiabetes),
			"risk_tier": string(models.RiskTierCritical),
		})
		return critical > 100
	}
	return false
}

// GetFiredAlerts returns all fired alerts
func (a *AlertManager) GetFiredAlerts() []FiredAlert {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.alerts
}

// TestMetrics_AlertOnHighErrorRate tests error rate alerting
func TestMetrics_AlertOnHighErrorRate(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)
	alertManager := NewAlertManager(registry)

	// Configure alert rule
	alertManager.AddRule(AlertRule{
		Name:       "KB17HighErrorRate",
		Expression: "rate(kb17_errors_total[5m]) > 10",
		For:        5 * time.Minute,
		Severity:   "critical",
		Labels:     map[string]string{"team": "population-registry"},
		Annotations: map[string]string{
			"summary":     "High error rate in KB-17",
			"description": "Error rate exceeded threshold",
		},
	})

	// Generate errors below threshold
	for i := 0; i < 5; i++ {
		collector.RecordError("enrollment", "validation")
	}
	fired := alertManager.EvaluateRules()
	assert.Len(t, fired, 0, "Should not fire below threshold")

	// Generate errors above threshold
	for i := 0; i < 10; i++ {
		collector.RecordError("enrollment", "validation")
	}
	fired = alertManager.EvaluateRules()
	assert.Len(t, fired, 1, "Should fire above threshold")
	assert.Equal(t, "KB17HighErrorRate", fired[0].Rule.Name)
}

// TestMetrics_AlertOnConsumerLag tests Kafka lag alerting
func TestMetrics_AlertOnConsumerLag(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)
	alertManager := NewAlertManager(registry)

	alertManager.AddRule(AlertRule{
		Name:       "KB17KafkaConsumerLag",
		Expression: "kb17_kafka_consumer_lag > 1000",
		For:        2 * time.Minute,
		Severity:   "warning",
	})

	// Normal lag
	collector.RecordKafkaConsumerLag("clinical.events", 0, 500)
	fired := alertManager.EvaluateRules()
	assert.Len(t, fired, 0)

	// High lag
	collector.RecordKafkaConsumerLag("clinical.events", 0, 2000)
	fired = alertManager.EvaluateRules()
	assert.Len(t, fired, 1)
	assert.Equal(t, "KB17KafkaConsumerLag", fired[0].Rule.Name)
}

// TestMetrics_AlertOnHighLatency tests API latency alerting
func TestMetrics_AlertOnHighLatency(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)
	alertManager := NewAlertManager(registry)

	alertManager.AddRule(AlertRule{
		Name:       "KB17HighAPILatency",
		Expression: "histogram_quantile(0.99, kb17_api_request_duration_ms) > 500",
		For:        5 * time.Minute,
		Severity:   "warning",
	})

	// Normal latency
	for i := 0; i < 10; i++ {
		collector.RecordAPILatency("/v1/enrollments", "POST", 50)
	}
	fired := alertManager.EvaluateRules()
	assert.Len(t, fired, 0)

	// High latency
	for i := 0; i < 20; i++ {
		collector.RecordAPILatency("/v1/enrollments", "POST", 1000)
	}
	fired = alertManager.EvaluateRules()
	assert.Len(t, fired, 1)
	assert.Equal(t, "KB17HighAPILatency", fired[0].Rule.Name)
}

// TestMetrics_AlertOnCriticalPopulation tests critical tier alerting
func TestMetrics_AlertOnCriticalPopulation(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)
	alertManager := NewAlertManager(registry)

	alertManager.AddRule(AlertRule{
		Name:       "KB17CriticalPopulationHigh",
		Expression: "kb17_population_critical > 100",
		For:        0,
		Severity:   "critical",
		Annotations: map[string]string{
			"summary": "Critical risk population exceeds threshold",
		},
	})

	// Normal critical population
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierCritical, 50)
	fired := alertManager.EvaluateRules()
	assert.Len(t, fired, 0)

	// High critical population
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierCritical, 150)
	fired = alertManager.EvaluateRules()
	assert.Len(t, fired, 1)
	assert.Equal(t, "critical", fired[0].Rule.Severity)
}

// =============================================================================
// CACHE METRICS TESTS
// =============================================================================

// TestMetrics_CacheHitRatio tests cache hit/miss tracking
func TestMetrics_CacheHitRatio(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// 80 hits, 20 misses = 80% hit rate
	for i := 0; i < 80; i++ {
		collector.RecordCacheHit("enrollments")
	}
	for i := 0; i < 20; i++ {
		collector.RecordCacheMiss("enrollments")
	}

	hits := registry.GetCounter("kb17_cache_hits_total", map[string]string{"cache": "enrollments"})
	misses := registry.GetCounter("kb17_cache_misses_total", map[string]string{"cache": "enrollments"})

	hitRate := hits / (hits + misses)
	assert.Equal(t, 0.8, hitRate, "Cache hit rate should be 80%")
}

// =============================================================================
// CONCURRENT METRICS TESTS
// =============================================================================

// TestMetrics_ConcurrentUpdates tests thread-safe metric updates
func TestMetrics_ConcurrentUpdates(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	const goroutines = 100
	const updatesPerGoroutine = 100

	var wg sync.WaitGroup
	var enrollmentCount int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)
				atomic.AddInt32(&enrollmentCount, 1)
			}
		}()
	}

	wg.Wait()

	// Verify all updates recorded
	metricCount := registry.GetCounter("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryDiabetes),
		"source":   string(models.EnrollmentSourceDiagnosis),
	})
	assert.Equal(t, float64(goroutines*updatesPerGoroutine), metricCount)
}

// =============================================================================
// BUSINESS METRICS TESTS (Clinical Quality Indicators)
// =============================================================================

// TestMetrics_ClinicalQualityIndicators tests quality metric exposure
func TestMetrics_ClinicalQualityIndicators(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Setup realistic population data
	// Diabetes registry: 1750 patients across risk tiers
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierLow, 1000)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierModerate, 500)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierHigh, 200)
	collector.SetPopulationSize(models.RegistryDiabetes, models.RiskTierCritical, 50)

	// Calculate quality indicators from metrics
	low := registry.GetGauge("kb17_population_size", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"risk_tier": string(models.RiskTierLow),
	})
	moderate := registry.GetGauge("kb17_population_size", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"risk_tier": string(models.RiskTierModerate),
	})
	high := registry.GetGauge("kb17_population_size", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"risk_tier": string(models.RiskTierHigh),
	})
	critical := registry.GetGauge("kb17_population_size", map[string]string{
		"registry":  string(models.RegistryDiabetes),
		"risk_tier": string(models.RiskTierCritical),
	})

	total := low + moderate + high + critical
	assert.Equal(t, float64(1750), total)

	// High-risk percentage (HIGH + CRITICAL) / TOTAL
	highRiskPercent := ((high + critical) / total) * 100
	assert.InDelta(t, 14.3, highRiskPercent, 0.5, "High-risk percentage should be ~14%")

	// Critical percentage
	criticalPercent := (critical / total) * 100
	assert.InDelta(t, 2.9, criticalPercent, 0.5, "Critical percentage should be ~3%")
}

// TestMetrics_EnrollmentVelocity tests enrollment rate tracking
func TestMetrics_EnrollmentVelocity(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Simulate enrollments over time
	startTime := time.Now()
	for i := 0; i < 100; i++ {
		collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)
	}
	duration := time.Since(startTime)

	enrollments := registry.GetCounter("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryDiabetes),
		"source":   string(models.EnrollmentSourceDiagnosis),
	})

	// Calculate velocity (enrollments per second)
	velocity := enrollments / duration.Seconds()
	assert.True(t, velocity > 0, "Enrollment velocity should be positive")
	t.Logf("Enrollment velocity: %.2f enrollments/second", velocity)
}

// =============================================================================
// HEALTH CHECK METRICS
// =============================================================================

// TestMetrics_HealthCheckMetrics tests health status exposure
func TestMetrics_HealthCheckMetrics(t *testing.T) {
	registry := NewMockMetricsRegistry()

	// Simulate health check metrics
	registry.SetGauge("kb17_health_status", 1, map[string]string{"component": "database"})
	registry.SetGauge("kb17_health_status", 1, map[string]string{"component": "redis"})
	registry.SetGauge("kb17_health_status", 1, map[string]string{"component": "kafka"})
	registry.SetGauge("kb17_health_status", 0, map[string]string{"component": "neo4j"}) // Unhealthy

	// Verify individual component health
	dbHealth := registry.GetGauge("kb17_health_status", map[string]string{"component": "database"})
	assert.Equal(t, float64(1), dbHealth, "Database should be healthy")

	neo4jHealth := registry.GetGauge("kb17_health_status", map[string]string{"component": "neo4j"})
	assert.Equal(t, float64(0), neo4jHealth, "Neo4j should be unhealthy")
}

// =============================================================================
// METRICS LABEL VALIDATION
// =============================================================================

// TestMetrics_LabelCardinality tests label cardinality limits
func TestMetrics_LabelCardinality(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Ensure we don't have high cardinality labels
	// Only use bounded enums for labels
	validRegistries := []models.RegistryCode{
		models.RegistryDiabetes,
		models.RegistryHypertension,
		models.RegistryHeartFailure,
		models.RegistryCKD,
		models.RegistryCOPD,
		models.RegistryPregnancy,
		models.RegistryOpioidUse,
		models.RegistryAnticoagulation,
	}

	for _, reg := range validRegistries {
		collector.RecordEnrollment(reg, models.EnrollmentSourceDiagnosis)
	}

	// Verify only 8 unique registry metrics (bounded cardinality)
	output := registry.GetAllMetrics()
	count := strings.Count(output, "kb17_enrollments_total")
	assert.LessOrEqual(t, count, 16, "Should have bounded metric cardinality")
}

// TestMetrics_TimestampFreshness tests metric timestamp updates
func TestMetrics_TimestampFreshness(t *testing.T) {
	registry := NewMockMetricsRegistry()
	collector := NewPopulationMetricsCollector(registry)

	// Record metric
	collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)

	// Wait and record again
	time.Sleep(10 * time.Millisecond)
	collector.RecordEnrollment(models.RegistryDiabetes, models.EnrollmentSourceDiagnosis)

	// Timestamp should be recent
	registry.mu.RLock()
	metric := registry.counters[metricKey("kb17_enrollments_total", map[string]string{
		"registry": string(models.RegistryDiabetes),
		"source":   string(models.EnrollmentSourceDiagnosis),
	})]
	registry.mu.RUnlock()

	require.NotNil(t, metric)
	assert.WithinDuration(t, time.Now(), metric.Timestamp, time.Second,
		"Metric timestamp should be fresh")
}
