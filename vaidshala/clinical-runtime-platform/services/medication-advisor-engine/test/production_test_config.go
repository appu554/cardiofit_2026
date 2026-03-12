// Package test provides production-grade test configuration.
// ALL tests MUST use real KB services - no mocks, no fallbacks.
package test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// ============================================================================
// Production Test Configuration
// ============================================================================

// ProductionTestConfig holds the configuration for production-grade testing.
// NO MOCKS, NO FALLBACKS - all KB services must be available.
type ProductionTestConfig struct {
	// KB Service URLs (required - will fail if not set or unavailable)
	KB1URL string // Drug Rules service - http://localhost:8081
	KB2URL string // Clinical Context - http://localhost:8082
	KB3URL string // Guidelines - http://localhost:8083
	KB4URL string // Patient Safety - http://localhost:8088
	KB5URL string // Drug Interactions - http://localhost:8095
	KB6URL string // Formulary - http://localhost:8087 (HTTP), 8086 is gRPC
	KB7URL string // Terminology - http://localhost:8092

	// Test behavior
	SkipIfKBUnavailable bool   // If true, skip tests instead of failing
	Environment         string // production, staging, development
}

// DefaultProductionConfig returns production-grade test configuration.
// Reads from environment variables, falls back to localhost defaults.
//
// Updated Port Mapping (per Docker deployment):
//   KB-1: 8081 (Drug Rules)
//   KB-2: 8082 (Clinical Context)
//   KB-3: 8083 (Guidelines)
//   KB-4: 8088 (Patient Safety)
//   KB-5: 8095 (Drug Interactions)
//   KB-6: 8087 HTTP, 8086 gRPC (Formulary)
//   KB-7: 8092 (Terminology)
func DefaultProductionConfig() ProductionTestConfig {
	return ProductionTestConfig{
		KB1URL:              getEnvWithDefault("KB1_URL", "http://localhost:8081"),
		KB2URL:              getEnvWithDefault("KB2_URL", "http://localhost:8082"),
		KB3URL:              getEnvWithDefault("KB3_URL", "http://localhost:8083"),
		KB4URL:              getEnvWithDefault("KB4_URL", "http://localhost:8088"),
		KB5URL:              getEnvWithDefault("KB5_URL", "http://localhost:8095"),
		KB6URL:              getEnvWithDefault("KB6_URL", "http://localhost:8087"),
		KB7URL:              getEnvWithDefault("KB7_URL", "http://localhost:8092"),
		SkipIfKBUnavailable: os.Getenv("SKIP_IF_KB_UNAVAILABLE") == "true",
		Environment:         getEnvWithDefault("TEST_ENVIRONMENT", "production"),
	}
}

func getEnvWithDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ============================================================================
// KB Health Check Validation
// ============================================================================

// KBHealthStatus represents the health status of a KB service
type KBHealthStatus struct {
	ServiceName string
	URL         string
	Healthy     bool
	Error       error
	ResponseMs  int64
}

// ValidateKBServices checks that ALL required KB services are healthy.
// This MUST pass before any production-grade tests can run.
func ValidateKBServices(t *testing.T, config ProductionTestConfig) {
	t.Helper()

	services := []struct {
		name string
		url  string
	}{
		{"KB-1 Drug Rules", config.KB1URL},
		{"KB-3 Guidelines", config.KB3URL},
		{"KB-4 Patient Safety", config.KB4URL},
		{"KB-7 Terminology", config.KB7URL},
	}

	var unhealthyServices []string
	var healthStatuses []KBHealthStatus

	for _, svc := range services {
		status := checkKBHealth(svc.name, svc.url)
		healthStatuses = append(healthStatuses, status)
		if !status.Healthy {
			unhealthyServices = append(unhealthyServices, fmt.Sprintf("%s (%s): %v", svc.name, svc.url, status.Error))
		}
	}

	// Log all health statuses
	t.Log("=== KB SERVICE HEALTH CHECK ===")
	for _, status := range healthStatuses {
		if status.Healthy {
			t.Logf("✅ %s: healthy (%dms)", status.ServiceName, status.ResponseMs)
		} else {
			t.Logf("❌ %s: UNHEALTHY - %v", status.ServiceName, status.Error)
		}
	}
	t.Log("================================")

	// Fail or skip based on configuration
	if len(unhealthyServices) > 0 {
		if config.SkipIfKBUnavailable {
			t.Skipf("Skipping test: KB services unavailable:\n%v", unhealthyServices)
		} else {
			t.Fatalf("PRODUCTION TEST FAILURE: Required KB services are unavailable:\n%v\n"+
				"Set SKIP_IF_KB_UNAVAILABLE=true to skip instead of fail", unhealthyServices)
		}
	}

	t.Log("✅ All KB services healthy - proceeding with production tests")
}

// checkKBHealth performs a health check on a KB service
func checkKBHealth(name, baseURL string) KBHealthStatus {
	healthURL := baseURL + "/health"
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return KBHealthStatus{
			ServiceName: name,
			URL:         baseURL,
			Healthy:     false,
			Error:       fmt.Errorf("failed to create request: %w", err),
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return KBHealthStatus{
			ServiceName: name,
			URL:         baseURL,
			Healthy:     false,
			Error:       fmt.Errorf("health check failed: %w", err),
			ResponseMs:  elapsed,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return KBHealthStatus{
			ServiceName: name,
			URL:         baseURL,
			Healthy:     false,
			Error:       fmt.Errorf("unhealthy status: %d", resp.StatusCode),
			ResponseMs:  elapsed,
		}
	}

	return KBHealthStatus{
		ServiceName: name,
		URL:         baseURL,
		Healthy:     true,
		ResponseMs:  elapsed,
	}
}

// ============================================================================
// Production Test Helpers
// ============================================================================

// RequireProductionKB ensures KB services are available before test.
// Use this at the start of any test that requires real KB data.
func RequireProductionKB(t *testing.T) ProductionTestConfig {
	t.Helper()
	config := DefaultProductionConfig()
	ValidateKBServices(t, config)
	return config
}

// SkipIfNotProduction skips the test if not running in production test mode
func SkipIfNotProduction(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_ENVIRONMENT") != "production" {
		t.Skip("Skipping: TEST_ENVIRONMENT != production")
	}
}

// MustHaveKB1 ensures KB-1 Drug Rules service is available
func MustHaveKB1(t *testing.T) string {
	t.Helper()
	config := DefaultProductionConfig()
	status := checkKBHealth("KB-1 Drug Rules", config.KB1URL)
	if !status.Healthy {
		if config.SkipIfKBUnavailable {
			t.Skipf("KB-1 unavailable: %v", status.Error)
		}
		t.Fatalf("KB-1 Drug Rules REQUIRED but unavailable: %v", status.Error)
	}
	return config.KB1URL
}

// MustHaveKB4 ensures KB-4 Patient Safety service is available
func MustHaveKB4(t *testing.T) string {
	t.Helper()
	config := DefaultProductionConfig()
	status := checkKBHealth("KB-4 Patient Safety", config.KB4URL)
	if !status.Healthy {
		if config.SkipIfKBUnavailable {
			t.Skipf("KB-4 unavailable: %v", status.Error)
		}
		t.Fatalf("KB-4 Patient Safety REQUIRED but unavailable: %v", status.Error)
	}
	return config.KB4URL
}

// MustHaveKB7 ensures KB-7 Terminology service is available
func MustHaveKB7(t *testing.T) string {
	t.Helper()
	config := DefaultProductionConfig()
	status := checkKBHealth("KB-7 Terminology", config.KB7URL)
	if !status.Healthy {
		if config.SkipIfKBUnavailable {
			t.Skipf("KB-7 unavailable: %v", status.Error)
		}
		t.Fatalf("KB-7 Terminology REQUIRED but unavailable: %v", status.Error)
	}
	return config.KB7URL
}
