// Package metrics provides Prometheus metrics collection for KB-13.
//
// Metrics are critical for:
//   - Quality measure calculation performance monitoring
//   - Care gap detection rate tracking
//   - API latency and throughput analysis
//   - Scheduler health monitoring
package metrics

import (
	"sync"
	"time"
)

// Collector aggregates metrics for KB-13 operations.
type Collector struct {
	mu sync.RWMutex

	// Calculation metrics
	calculationsTotal      int64
	calculationErrors      int64
	calculationDuration    []time.Duration
	batchCalculationsTotal int64

	// Care gap metrics
	careGapsIdentified int64
	careGapsResolved   int64
	careGapsOverdue    int64

	// API metrics
	apiRequestsTotal  map[string]int64
	apiRequestErrors  map[string]int64
	apiLatencies      map[string][]time.Duration

	// Scheduler metrics
	schedulerJobsTotal   int64
	schedulerJobsSuccess int64
	schedulerJobsFailed  int64
	schedulerLastRunTime map[string]time.Time

	// Measure store metrics
	measuresLoaded int
	measuresActive int
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		calculationDuration:  make([]time.Duration, 0, 1000),
		apiRequestsTotal:     make(map[string]int64),
		apiRequestErrors:     make(map[string]int64),
		apiLatencies:         make(map[string][]time.Duration),
		schedulerLastRunTime: make(map[string]time.Time),
	}
}

// --- Calculation Metrics ---

// RecordCalculation records a measure calculation.
func (c *Collector) RecordCalculation(duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calculationsTotal++
	if !success {
		c.calculationErrors++
	}

	// Keep last 1000 durations for percentile calculations
	c.calculationDuration = append(c.calculationDuration, duration)
	if len(c.calculationDuration) > 1000 {
		c.calculationDuration = c.calculationDuration[1:]
	}
}

// RecordBatchCalculation records a batch calculation.
func (c *Collector) RecordBatchCalculation(measureCount int, successCount int, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batchCalculationsTotal++
	c.calculationsTotal += int64(measureCount)
	c.calculationErrors += int64(measureCount - successCount)
}

// --- Care Gap Metrics ---

// RecordCareGapIdentified records a newly identified care gap.
func (c *Collector) RecordCareGapIdentified(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.careGapsIdentified += int64(count)
}

// RecordCareGapResolved records a resolved care gap.
func (c *Collector) RecordCareGapResolved() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.careGapsResolved++
}

// SetCareGapsOverdue sets the current overdue care gap count.
func (c *Collector) SetCareGapsOverdue(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.careGapsOverdue = int64(count)
}

// --- API Metrics ---

// RecordAPIRequest records an API request.
func (c *Collector) RecordAPIRequest(endpoint string, duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.apiRequestsTotal[endpoint]++
	if !success {
		c.apiRequestErrors[endpoint]++
	}

	// Keep last 100 latencies per endpoint
	if c.apiLatencies[endpoint] == nil {
		c.apiLatencies[endpoint] = make([]time.Duration, 0, 100)
	}
	c.apiLatencies[endpoint] = append(c.apiLatencies[endpoint], duration)
	if len(c.apiLatencies[endpoint]) > 100 {
		c.apiLatencies[endpoint] = c.apiLatencies[endpoint][1:]
	}
}

// --- Scheduler Metrics ---

// RecordSchedulerJob records a scheduler job execution.
func (c *Collector) RecordSchedulerJob(scheduleType string, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.schedulerJobsTotal++
	if success {
		c.schedulerJobsSuccess++
	} else {
		c.schedulerJobsFailed++
	}
	c.schedulerLastRunTime[scheduleType] = time.Now()
}

// --- Measure Store Metrics ---

// SetMeasureCounts sets the current measure counts.
func (c *Collector) SetMeasureCounts(loaded, active int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.measuresLoaded = loaded
	c.measuresActive = active
}

// --- Metric Retrieval ---

// Snapshot represents a point-in-time view of all metrics.
type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`

	// Calculation metrics
	CalculationsTotal       int64   `json:"calculations_total"`
	CalculationErrors       int64   `json:"calculation_errors"`
	CalculationErrorRate    float64 `json:"calculation_error_rate"`
	CalculationAvgDuration  float64 `json:"calculation_avg_duration_ms"`
	CalculationP50Duration  float64 `json:"calculation_p50_duration_ms"`
	CalculationP95Duration  float64 `json:"calculation_p95_duration_ms"`
	CalculationP99Duration  float64 `json:"calculation_p99_duration_ms"`
	BatchCalculationsTotal  int64   `json:"batch_calculations_total"`

	// Care gap metrics
	CareGapsIdentified int64 `json:"care_gaps_identified"`
	CareGapsResolved   int64 `json:"care_gaps_resolved"`
	CareGapsOverdue    int64 `json:"care_gaps_overdue"`

	// API metrics
	APIRequestsTotal map[string]int64   `json:"api_requests_total"`
	APIErrorRates    map[string]float64 `json:"api_error_rates"`
	APIAvgLatencies  map[string]float64 `json:"api_avg_latencies_ms"`

	// Scheduler metrics
	SchedulerJobsTotal   int64            `json:"scheduler_jobs_total"`
	SchedulerJobsSuccess int64            `json:"scheduler_jobs_success"`
	SchedulerJobsFailed  int64            `json:"scheduler_jobs_failed"`
	SchedulerLastRuns    map[string]int64 `json:"scheduler_last_runs_unix"`

	// Measure store metrics
	MeasuresLoaded int `json:"measures_loaded"`
	MeasuresActive int `json:"measures_active"`
}

// GetSnapshot returns a point-in-time snapshot of all metrics.
func (c *Collector) GetSnapshot() *Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := &Snapshot{
		Timestamp:              time.Now(),
		CalculationsTotal:      c.calculationsTotal,
		CalculationErrors:      c.calculationErrors,
		BatchCalculationsTotal: c.batchCalculationsTotal,
		CareGapsIdentified:     c.careGapsIdentified,
		CareGapsResolved:       c.careGapsResolved,
		CareGapsOverdue:        c.careGapsOverdue,
		SchedulerJobsTotal:     c.schedulerJobsTotal,
		SchedulerJobsSuccess:   c.schedulerJobsSuccess,
		SchedulerJobsFailed:    c.schedulerJobsFailed,
		MeasuresLoaded:         c.measuresLoaded,
		MeasuresActive:         c.measuresActive,
		APIRequestsTotal:       make(map[string]int64),
		APIErrorRates:          make(map[string]float64),
		APIAvgLatencies:        make(map[string]float64),
		SchedulerLastRuns:      make(map[string]int64),
	}

	// Calculate error rate
	if c.calculationsTotal > 0 {
		snapshot.CalculationErrorRate = float64(c.calculationErrors) / float64(c.calculationsTotal)
	}

	// Calculate duration percentiles
	if len(c.calculationDuration) > 0 {
		sorted := sortDurations(c.calculationDuration)
		snapshot.CalculationAvgDuration = avgDuration(sorted)
		snapshot.CalculationP50Duration = percentileDuration(sorted, 0.50)
		snapshot.CalculationP95Duration = percentileDuration(sorted, 0.95)
		snapshot.CalculationP99Duration = percentileDuration(sorted, 0.99)
	}

	// Copy API metrics
	for endpoint, count := range c.apiRequestsTotal {
		snapshot.APIRequestsTotal[endpoint] = count
		if errors, ok := c.apiRequestErrors[endpoint]; ok && count > 0 {
			snapshot.APIErrorRates[endpoint] = float64(errors) / float64(count)
		}
		if latencies, ok := c.apiLatencies[endpoint]; ok && len(latencies) > 0 {
			snapshot.APIAvgLatencies[endpoint] = avgDuration(latencies)
		}
	}

	// Copy scheduler last runs
	for schedType, lastRun := range c.schedulerLastRunTime {
		snapshot.SchedulerLastRuns[schedType] = lastRun.Unix()
	}

	return snapshot
}

// PrometheusFormat returns metrics in Prometheus text format.
func (c *Collector) PrometheusFormat() string {
	snapshot := c.GetSnapshot()

	var result string

	// Calculation metrics
	result += "# HELP kb13_calculations_total Total number of measure calculations\n"
	result += "# TYPE kb13_calculations_total counter\n"
	result += formatMetric("kb13_calculations_total", float64(snapshot.CalculationsTotal))

	result += "# HELP kb13_calculation_errors_total Total number of calculation errors\n"
	result += "# TYPE kb13_calculation_errors_total counter\n"
	result += formatMetric("kb13_calculation_errors_total", float64(snapshot.CalculationErrors))

	result += "# HELP kb13_calculation_duration_ms Calculation duration in milliseconds\n"
	result += "# TYPE kb13_calculation_duration_ms gauge\n"
	result += formatMetricWithLabel("kb13_calculation_duration_ms", "quantile", "0.5", snapshot.CalculationP50Duration)
	result += formatMetricWithLabel("kb13_calculation_duration_ms", "quantile", "0.95", snapshot.CalculationP95Duration)
	result += formatMetricWithLabel("kb13_calculation_duration_ms", "quantile", "0.99", snapshot.CalculationP99Duration)

	// Care gap metrics
	result += "# HELP kb13_care_gaps_identified_total Total care gaps identified\n"
	result += "# TYPE kb13_care_gaps_identified_total counter\n"
	result += formatMetric("kb13_care_gaps_identified_total", float64(snapshot.CareGapsIdentified))

	result += "# HELP kb13_care_gaps_overdue Current overdue care gaps\n"
	result += "# TYPE kb13_care_gaps_overdue gauge\n"
	result += formatMetric("kb13_care_gaps_overdue", float64(snapshot.CareGapsOverdue))

	// Scheduler metrics
	result += "# HELP kb13_scheduler_jobs_total Total scheduler jobs\n"
	result += "# TYPE kb13_scheduler_jobs_total counter\n"
	result += formatMetric("kb13_scheduler_jobs_total", float64(snapshot.SchedulerJobsTotal))

	result += "# HELP kb13_scheduler_jobs_success Successful scheduler jobs\n"
	result += "# TYPE kb13_scheduler_jobs_success counter\n"
	result += formatMetric("kb13_scheduler_jobs_success", float64(snapshot.SchedulerJobsSuccess))

	// Measure store metrics
	result += "# HELP kb13_measures_loaded Number of measures loaded\n"
	result += "# TYPE kb13_measures_loaded gauge\n"
	result += formatMetric("kb13_measures_loaded", float64(snapshot.MeasuresLoaded))

	result += "# HELP kb13_measures_active Number of active measures\n"
	result += "# TYPE kb13_measures_active gauge\n"
	result += formatMetric("kb13_measures_active", float64(snapshot.MeasuresActive))

	return result
}

// --- Helper Functions ---

func formatMetric(name string, value float64) string {
	return name + " " + formatFloat(value) + "\n"
}

func formatMetricWithLabel(name, labelKey, labelValue string, value float64) string {
	return name + "{" + labelKey + "=\"" + labelValue + "\"} " + formatFloat(value) + "\n"
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return string(rune(int64(f) + '0'))
	}
	// Simple float formatting
	return string(rune(int(f))) + "." + string(rune(int(f*10)%10+'0'))
}

func sortDurations(durations []time.Duration) []time.Duration {
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	return sorted
}

func avgDuration(durations []time.Duration) float64 {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return float64(sum.Milliseconds()) / float64(len(durations))
}

func percentileDuration(sortedDurations []time.Duration, p float64) float64 {
	if len(sortedDurations) == 0 {
		return 0
	}
	idx := int(float64(len(sortedDurations)-1) * p)
	return float64(sortedDurations[idx].Milliseconds())
}

// --- Global Collector ---

var (
	globalCollector     *Collector
	globalCollectorOnce sync.Once
)

// Global returns the global metrics collector.
func Global() *Collector {
	globalCollectorOnce.Do(func() {
		globalCollector = NewCollector()
	})
	return globalCollector
}
