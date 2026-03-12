package services

import (
	"sync"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/metrics"
)

// sharedCollector ensures metrics.NewCollector() is called exactly once per test
// binary, avoiding Prometheus duplicate registration panics.
var (
	sharedCollector     *metrics.Collector
	sharedCollectorOnce sync.Once
)

func testMetrics() *metrics.Collector {
	sharedCollectorOnce.Do(func() {
		sharedCollector = metrics.NewCollector()
	})
	return sharedCollector
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}
