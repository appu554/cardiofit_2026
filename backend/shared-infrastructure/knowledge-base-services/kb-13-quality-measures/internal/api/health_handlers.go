// Package api provides HTTP handlers for KB-13 Quality Measures Engine.
package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"kb-13-quality-measures/internal/config"
)

// startTime records when the server started for uptime calculation.
var startTime = time.Now()

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Uptime      string            `json:"uptime"`
	Checks      map[string]string `json:"checks,omitempty"`
}

// ReadinessResponse represents the readiness check response.
type ReadinessResponse struct {
	Ready       bool              `json:"ready"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Measures    int               `json:"measures_loaded"`
	Benchmarks  int               `json:"benchmarks_loaded"`
	Checks      map[string]string `json:"checks"`
}

// HealthCheck handles the /health endpoint.
// This is a simple liveness probe that always returns OK if the server is running.
func (s *Server) HealthCheck(c *gin.Context) {
	uptime := time.Since(startTime).Round(time.Second)

	response := HealthResponse{
		Status:      "healthy",
		Service:     config.ServiceName,
		Version:     config.Version,
		Environment: s.config.Server.Environment,
		Uptime:      uptime.String(),
	}

	c.JSON(http.StatusOK, response)
}

// ReadinessCheck handles the /ready endpoint.
// This checks if the service is ready to accept traffic.
func (s *Server) ReadinessCheck(c *gin.Context) {
	checks := make(map[string]string)
	ready := true

	// Check 1: Measure store loaded
	measureCount := s.store.Count()
	if measureCount > 0 {
		checks["measures"] = "ok"
	} else {
		checks["measures"] = "no_measures_loaded"
		// Not a hard failure - service can run without pre-loaded measures
	}

	// Check 2: Benchmark store
	benchmarkCount := s.store.BenchmarkCount()
	if benchmarkCount > 0 {
		checks["benchmarks"] = "ok"
	} else {
		checks["benchmarks"] = "no_benchmarks_loaded"
		// Not a hard failure
	}

	// Check 3: Memory usage (warn if too high)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	allocMB := memStats.Alloc / 1024 / 1024
	if allocMB < 500 {
		checks["memory"] = "ok"
	} else if allocMB < 1000 {
		checks["memory"] = "warning_high_usage"
	} else {
		checks["memory"] = "critical_high_usage"
		// Still allow traffic but log warning
	}

	// TODO: Add database connectivity check when PostgreSQL is integrated
	// TODO: Add Redis connectivity check when caching is implemented
	// TODO: Add Vaidshala CQL Engine connectivity check

	response := ReadinessResponse{
		Ready:      ready,
		Service:    config.ServiceName,
		Version:    config.Version,
		Measures:   measureCount,
		Benchmarks: benchmarkCount,
		Checks:     checks,
	}

	if ready {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// MetricsHandler handles the /metrics endpoint for Prometheus scraping.
func (s *Server) MetricsHandler(c *gin.Context) {
	// Placeholder for Prometheus metrics
	// In Phase 4, this will be replaced with proper prometheus/client_golang integration

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Return basic metrics in Prometheus text format
	metrics := `# HELP kb13_up KB-13 service up status
# TYPE kb13_up gauge
kb13_up 1

# HELP kb13_measures_loaded Number of measures loaded in store
# TYPE kb13_measures_loaded gauge
kb13_measures_loaded ` + itoa(s.store.Count()) + `

# HELP kb13_benchmarks_loaded Number of benchmarks loaded in store
# TYPE kb13_benchmarks_loaded gauge
kb13_benchmarks_loaded ` + itoa(s.store.BenchmarkCount()) + `

# HELP kb13_memory_alloc_bytes Current memory allocation in bytes
# TYPE kb13_memory_alloc_bytes gauge
kb13_memory_alloc_bytes ` + uitoa(memStats.Alloc) + `

# HELP kb13_goroutines Number of goroutines
# TYPE kb13_goroutines gauge
kb13_goroutines ` + itoa(runtime.NumGoroutine()) + `
`

	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(metrics))
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var negative bool
	if i < 0 {
		negative = true
		i = -i
	}

	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte(i%10) + '0'
		i /= 10
	}

	if negative {
		pos--
		b[pos] = '-'
	}

	return string(b[pos:])
}

// uitoa converts uint64 to string without importing strconv.
func uitoa(i uint64) string {
	if i == 0 {
		return "0"
	}

	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte(i%10) + '0'
		i /= 10
	}

	return string(b[pos:])
}
