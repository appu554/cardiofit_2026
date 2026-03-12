package metrics

import (
	"time"
	"github.com/shopspring/decimal"
)

// PerformanceTracker tracks service performance metrics
type PerformanceTracker struct {
	requestCount    int64
	totalDuration   time.Duration
	lastReset      time.Time
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		lastReset: time.Now(),
	}
}

// RecordRequest records a request with its duration
func (pt *PerformanceTracker) RecordRequest(duration time.Duration) {
	pt.requestCount++
	pt.totalDuration += duration
}

// GetAverageLatency returns the average request latency
func (pt *PerformanceTracker) GetAverageLatency() time.Duration {
	if pt.requestCount == 0 {
		return 0
	}
	return pt.totalDuration / time.Duration(pt.requestCount)
}

// GetRequestCount returns total request count
func (pt *PerformanceTracker) GetRequestCount() int64 {
	return pt.requestCount
}

// GetThroughput returns requests per second
func (pt *PerformanceTracker) GetThroughput() decimal.Decimal {
	if pt.requestCount == 0 {
		return decimal.Zero
	}
	
	duration := time.Since(pt.lastReset).Seconds()
	if duration == 0 {
		return decimal.Zero
	}
	
	return decimal.NewFromInt(pt.requestCount).Div(decimal.NewFromFloat(duration))
}

// Reset resets all performance counters
func (pt *PerformanceTracker) Reset() {
	pt.requestCount = 0
	pt.totalDuration = 0
	pt.lastReset = time.Now()
}