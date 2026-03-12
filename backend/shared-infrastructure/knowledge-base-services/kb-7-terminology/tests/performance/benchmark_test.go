package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kb-7-terminology/tests/fixtures"
)

// BenchmarkConfig holds benchmark configuration
type BenchmarkConfig struct {
	BaseURL string
	Client  *http.Client
}

// NewBenchmarkConfig creates a new benchmark configuration
func NewBenchmarkConfig() *BenchmarkConfig {
	baseURL := os.Getenv("KB7_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8087"
	}

	return &BenchmarkConfig{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// BenchmarkHealthEndpoint benchmarks the health check endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Warm up
	resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkValueSetList benchmarks listing value sets
func BenchmarkValueSetList(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Warm up and verify service
	resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=10")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=10")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkValueSetListAll benchmarks listing all 49 value sets
func BenchmarkValueSetListAll(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Warm up
	resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=100")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=100")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkSubsumptionCheck benchmarks subsumption checking
func BenchmarkSubsumptionCheck(b *testing.B) {
	cfg := NewBenchmarkConfig()

	request := map[string]interface{}{
		"subCode":   "64572001",  // Disease
		"superCode": "404684003", // Clinical finding
		"system":    "http://snomed.info/sct",
	}
	reqJSON, _ := json.Marshal(request)

	// Warm up
	resp, err := cfg.Client.Post(
		cfg.BaseURL+"/v1/subsumption/check",
		"application/json",
		bytes.NewBuffer(reqJSON))
	if err != nil {
		b.Skipf("Subsumption not available: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		b.Skipf("Subsumption returned status %d", resp.StatusCode)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := cfg.Client.Post(
			cfg.BaseURL+"/v1/subsumption/check",
			"application/json",
			bytes.NewBuffer(reqJSON))
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkValueSetContains benchmarks value set membership check
func BenchmarkValueSetContains(b *testing.B) {
	cfg := NewBenchmarkConfig()

	url := cfg.BaseURL + "/v1/rules/valuesets/AdministrativeGender/contains?code=male&system=http://hl7.org/fhir/administrative-gender"

	// Warm up
	resp, err := cfg.Client.Get(url)
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		b.Skipf("Contains endpoint returned status %d", resp.StatusCode)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := cfg.Client.Get(url)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkBuiltinCount benchmarks the builtin count endpoint
func BenchmarkBuiltinCount(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Warm up
	resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/valuesets/builtin/count")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/valuesets/builtin/count")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkConcurrentHealth benchmarks concurrent health checks
func BenchmarkConcurrentHealth(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Verify service is available
	resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 10 * time.Second}
		for pb.Next() {
			resp, err := client.Get(cfg.BaseURL + "/health")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkConcurrentValueSetList benchmarks concurrent value set listing
func BenchmarkConcurrentValueSetList(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Verify service is available
	resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=10")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 10 * time.Second}
		for pb.Next() {
			resp, err := client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=10")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}

// TestLoadTest performs a basic load test
func TestLoadTest(t *testing.T) {
	if os.Getenv("RUN_LOAD_TEST") != "true" {
		t.Skip("Load test skipped. Set RUN_LOAD_TEST=true to run")
	}

	cfg := NewBenchmarkConfig()
	loadCfg := fixtures.DefaultLoadTestConfig()

	// Verify service is available
	resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
	if err != nil {
		t.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	var (
		successCount int64
		errorCount   int64
		totalLatency int64
	)

	duration, _ := time.ParseDuration(loadCfg.Duration)
	vus := loadCfg.VUs

	t.Logf("Starting load test: %d VUs for %v", vus, duration)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// Start virtual users
	for i := 0; i < vus; i++ {
		wg.Add(1)
		go func(vuID int) {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}

			for {
				select {
				case <-stopChan:
					return
				default:
					start := time.Now()
					resp, err := client.Get(cfg.BaseURL + "/health")
					latency := time.Since(start)

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							atomic.AddInt64(&successCount, 1)
							atomic.AddInt64(&totalLatency, int64(latency))
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					}

					// Small delay to prevent CPU saturation
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	// Calculate statistics
	totalRequests := successCount + errorCount
	successRate := float64(successCount) / float64(totalRequests) * 100
	avgLatency := time.Duration(0)
	if successCount > 0 {
		avgLatency = time.Duration(totalLatency / successCount)
	}
	rps := float64(totalRequests) / duration.Seconds()

	t.Logf("Load Test Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d (%.2f%%)", successCount, successRate)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Average Latency: %v", avgLatency)
	t.Logf("  Requests/Second: %.2f", rps)

	// Assertions
	if successRate < 99.0 {
		t.Errorf("Success rate %.2f%% is below 99%% threshold", successRate)
	}
	if avgLatency > time.Duration(loadCfg.Thresholds.HTTPReqDuration95thPercentile)*time.Millisecond {
		t.Errorf("Average latency %v exceeds threshold", avgLatency)
	}
}

// TestStressTest performs a stress test with increasing load
func TestStressTest(t *testing.T) {
	if os.Getenv("RUN_STRESS_TEST") != "true" {
		t.Skip("Stress test skipped. Set RUN_STRESS_TEST=true to run")
	}

	cfg := NewBenchmarkConfig()
	stressCfg := fixtures.DefaultStressTestConfig()

	// Verify service is available
	resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
	if err != nil {
		t.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	t.Logf("Starting stress test with %d stages", len(stressCfg.Stages))

	for i, stage := range stressCfg.Stages {
		duration, _ := time.ParseDuration(stage.Duration)
		target := stage.Target

		t.Logf("Stage %d: %d VUs for %v", i+1, target, duration)

		var (
			successCount int64
			errorCount   int64
		)

		var wg sync.WaitGroup
		stopChan := make(chan struct{})

		// Start virtual users for this stage
		for vu := 0; vu < target; vu++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := &http.Client{Timeout: 10 * time.Second}

				for {
					select {
					case <-stopChan:
						return
					default:
						resp, err := client.Get(cfg.BaseURL + "/health")
						if err != nil {
							atomic.AddInt64(&errorCount, 1)
						} else {
							resp.Body.Close()
							if resp.StatusCode == http.StatusOK {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}
						}
						time.Sleep(50 * time.Millisecond)
					}
				}
			}()
		}

		time.Sleep(duration)
		close(stopChan)
		wg.Wait()

		totalRequests := successCount + errorCount
		successRate := float64(successCount) / float64(totalRequests) * 100

		t.Logf("  Stage %d Results: %d requests, %.2f%% success", i+1, totalRequests, successRate)

		// Check if we've reached breaking point
		if successRate < 90.0 {
			t.Logf("  Breaking point reached at %d VUs", target)
			break
		}
	}
}

// TestLatencyPercentiles measures latency percentiles
func TestLatencyPercentiles(t *testing.T) {
	if os.Getenv("RUN_PERF_TEST") != "true" {
		t.Skip("Performance test skipped. Set RUN_PERF_TEST=true to run")
	}

	cfg := NewBenchmarkConfig()

	// Verify service is available
	resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
	if err != nil {
		t.Skipf("Service not available: %v", err)
	}
	resp.Body.Close()

	const numRequests = 100
	latencies := make([]time.Duration, numRequests)

	for i := 0; i < numRequests; i++ {
		start := time.Now()
		resp, err := cfg.Client.Get(cfg.BaseURL + "/health")
		latencies[i] = time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	// Sort latencies
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[j] < latencies[i] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	p50 := latencies[numRequests*50/100]
	p90 := latencies[numRequests*90/100]
	p95 := latencies[numRequests*95/100]
	p99 := latencies[numRequests*99/100]

	t.Logf("Latency Percentiles (%d requests):", numRequests)
	t.Logf("  p50: %v", p50)
	t.Logf("  p90: %v", p90)
	t.Logf("  p95: %v", p95)
	t.Logf("  p99: %v", p99)

	// Assertions
	if p95 > 500*time.Millisecond {
		t.Errorf("p95 latency %v exceeds 500ms threshold", p95)
	}
}

// BenchmarkAllValueSets benchmarks fetching each value set individually
func BenchmarkAllValueSets(b *testing.B) {
	cfg := NewBenchmarkConfig()

	// Get list of value sets
	resp, err := cfg.Client.Get(cfg.BaseURL + "/v1/rules/valuesets?limit=100")
	if err != nil {
		b.Skipf("Service not available: %v", err)
	}

	var listResult struct {
		ValueSets []struct {
			ID string `json:"id"`
		} `json:"value_sets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		resp.Body.Close()
		b.Skipf("Cannot parse value sets: %v", err)
	}
	resp.Body.Close()

	if len(listResult.ValueSets) == 0 {
		b.Skip("No value sets found")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Fetch a value set (round-robin through all)
		vs := listResult.ValueSets[i%len(listResult.ValueSets)]
		url := fmt.Sprintf("%s/v1/rules/valuesets/%s", cfg.BaseURL, vs.ID)
		resp, err := cfg.Client.Get(url)
		if err != nil {
			b.Error(err)
			continue
		}
		resp.Body.Close()
	}
}
