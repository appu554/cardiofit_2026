// Package clients provides HTTP clients for KB services.
//
// ClientFactory creates pre-configured KB clients for connecting to
// the Knowledge Base microservices running in Docker.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// These clients are used by KnowledgeSnapshotBuilder to pre-compute
// KB answers at snapshot build time. Engines NEVER call KB services
// directly at execution time.
//
// Default Docker Ports:
// - KB-1 Drug Rules: 8081
// - KB-4 Patient Safety: 8088
// - KB-5 Drug Interactions: 8095/8096
// - KB-6 Formulary: 8087 (HTTP), 8086 (gRPC)
// - KB-7 Terminology: 8092
// - KB-8 Calculator: 8097
package clients

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

// KBClientConfig holds configuration for all KB service clients.
type KBClientConfig struct {
	// Base URLs for each KB service
	KB1BaseURL string // Drug Rules (default: http://localhost:8081)
	KB4BaseURL string // Patient Safety (default: http://localhost:8088)
	KB5BaseURL string // Drug Interactions (default: http://localhost:8095)
	KB6BaseURL string // Formulary (default: http://localhost:8087 HTTP, 8086 gRPC)
	KB7BaseURL string // Terminology (default: http://localhost:8092)
	KB8BaseURL string // Calculator (default: http://localhost:8097)

	// HTTP client settings
	Timeout time.Duration // Default: 30s
}

// DefaultKBClientConfig returns default configuration for local Docker setup.
func DefaultKBClientConfig() KBClientConfig {
	return KBClientConfig{
		KB1BaseURL: "http://localhost:8081",
		KB4BaseURL: "http://localhost:8088",
		KB5BaseURL: "http://localhost:8095",
		KB6BaseURL: "http://localhost:8087",
		KB7BaseURL: "http://localhost:8092",
		KB8BaseURL: "http://localhost:8097",
		Timeout:    30 * time.Second,
	}
}

// DockerKBClientConfig returns configuration for Docker Compose networking.
// Use this when running inside Docker Compose with service names.
func DockerKBClientConfig() KBClientConfig {
	return KBClientConfig{
		KB1BaseURL: "http://kb1-drug-rules:8081",
		KB4BaseURL: "http://kb4-patient-safety:8088",
		KB5BaseURL: "http://kb5-drug-interactions:8095",
		KB6BaseURL: "http://kb6-formulary:8087",
		KB7BaseURL: "http://kb7-terminology:8092",
		KB8BaseURL: "http://kb8-calculator:8097",
		Timeout:    30 * time.Second,
	}
}

// KBClients holds all KB client instances.
type KBClients struct {
	KB1 *KB1HTTPClient     // Drug Rules
	KB4 *KB4HTTPClient     // Patient Safety
	KB5 *KB5HTTPClient     // Drug Interactions
	KB6 *KB6HTTPClient     // Formulary
	KB7 *KB7FHIRHTTPClient // Terminology (FHIR)
	KB8 *KB8HTTPClient     // Calculator
}

// NewKBClients creates all KB clients from configuration.
func NewKBClients(config KBClientConfig) *KBClients {
	// Create shared HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &KBClients{
		KB1: NewKB1HTTPClientWithHTTP(config.KB1BaseURL, httpClient),
		KB4: NewKB4HTTPClientWithHTTP(config.KB4BaseURL, httpClient),
		KB5: NewKB5HTTPClientWithHTTP(config.KB5BaseURL, httpClient),
		KB6: NewKB6HTTPClientWithHTTP(config.KB6BaseURL, httpClient),
		KB7: NewKB7FHIRClientWithHTTP(config.KB7BaseURL, httpClient),
		KB8: NewKB8HTTPClientWithHTTP(config.KB8BaseURL, httpClient),
	}
}

// NewDefaultKBClients creates KB clients with default local Docker configuration.
func NewDefaultKBClients() *KBClients {
	return NewKBClients(DefaultKBClientConfig())
}

// ============================================================================
// Environment-based Configuration
// ============================================================================

// KBClientConfigFromEnv creates configuration from environment variables.
// Falls back to default values if environment variables are not set.
//
// Environment Variables:
// - KB1_URL: Drug Rules service URL
// - KB4_URL: Patient Safety service URL
// - KB5_URL: Drug Interactions service URL
// - KB6_URL: Formulary service URL
// - KB7_URL: Terminology service URL
// - KB8_URL: Calculator service URL
// - KB_TIMEOUT: HTTP timeout in seconds (default: 30)
func KBClientConfigFromEnv() KBClientConfig {
	config := DefaultKBClientConfig()

	// Override with environment variables if set
	if url := os.Getenv("KB1_URL"); url != "" {
		config.KB1BaseURL = url
	}
	if url := os.Getenv("KB4_URL"); url != "" {
		config.KB4BaseURL = url
	}
	if url := os.Getenv("KB5_URL"); url != "" {
		config.KB5BaseURL = url
	}
	if url := os.Getenv("KB6_URL"); url != "" {
		config.KB6BaseURL = url
	}
	if url := os.Getenv("KB7_URL"); url != "" {
		config.KB7BaseURL = url
	}
	if url := os.Getenv("KB8_URL"); url != "" {
		config.KB8BaseURL = url
	}
	if timeoutStr := os.Getenv("KB_TIMEOUT"); timeoutStr != "" {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil {
			config.Timeout = time.Duration(seconds) * time.Second
		}
	}

	return config
}

// ============================================================================
// Health Check Utilities
// ============================================================================

// KBHealthStatus represents the health status of all KB services.
type KBHealthStatus struct {
	KB1Healthy bool
	KB4Healthy bool
	KB5Healthy bool
	KB6Healthy bool
	KB7Healthy bool
	KB8Healthy bool
	AllHealthy bool
}

// CheckHealth verifies connectivity to all KB services.
// Returns health status without blocking - failures are non-fatal.
func (c *KBClients) CheckHealth() *KBHealthStatus {
	status := &KBHealthStatus{
		KB1Healthy: c.KB1 != nil,
		KB4Healthy: c.KB4 != nil,
		KB5Healthy: c.KB5 != nil,
		KB6Healthy: c.KB6 != nil,
		KB7Healthy: c.KB7 != nil,
		KB8Healthy: c.KB8 != nil,
	}

	status.AllHealthy = status.KB1Healthy && status.KB4Healthy &&
		status.KB5Healthy && status.KB6Healthy &&
		status.KB7Healthy && status.KB8Healthy

	return status
}
