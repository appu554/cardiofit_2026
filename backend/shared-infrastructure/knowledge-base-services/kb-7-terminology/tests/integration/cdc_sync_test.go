// Package integration provides end-to-end integration tests for KB-7 Terminology Service.
//
// CDC Sync Test: Validates the complete CDC pipeline from GraphDB to Neo4j to Go API.
//
// Architecture being tested:
//   GraphDB (Source) → Kafka (CDC) → Neo4j (Read Replica) → Go API (Service)
//
// Run with:
//   go test -v -tags=integration ./tests/integration/...
//
// Skip in short mode:
//   go test -short ./tests/integration/...
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestConfig holds integration test configuration from environment
type TestConfig struct {
	GraphDBURL  string
	GraphDBRepo string
	GoAPIURL    string
	Neo4jURL    string
	MaxWaitSec  int
}

// loadTestConfig loads configuration from environment variables
func loadTestConfig() *TestConfig {
	return &TestConfig{
		GraphDBURL:  getEnvOrDefault("GRAPHDB_URL", "http://localhost:7200"),
		GraphDBRepo: getEnvOrDefault("GRAPHDB_REPO", "kb7-terminology"),
		GoAPIURL:    getEnvOrDefault("GO_API_URL", "http://localhost:8087"),
		Neo4jURL:    getEnvOrDefault("NEO4J_URL", "bolt://localhost:7688"),
		MaxWaitSec:  10,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ConceptResponse represents the API response for concept lookup
type ConceptResponse struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
	Active  bool   `json:"active"`
	Backend string `json:"backend"`
}

// HealthResponse represents the API health response
type HealthResponse struct {
	Status   string                 `json:"status"`
	Database map[string]interface{} `json:"database"`
	Redis    map[string]interface{} `json:"redis"`
	GraphDB  map[string]interface{} `json:"graphdb"`
}

// TestEndToEndCDCSync validates the complete CDC pipeline.
// It inserts a "canary" concept into GraphDB and verifies it appears in the Go API.
//
// This test requires:
//   - GraphDB running with kb7-terminology repository
//   - Kafka CDC pipeline configured
//   - Neo4j read replica running
//   - KB-7 Go API service running
func TestEndToEndCDCSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := loadTestConfig()
	ctx := context.Background()

	// Generate unique canary to avoid conflicts
	canaryCode := fmt.Sprintf("TEST_%d", time.Now().UnixNano())
	canaryLabel := fmt.Sprintf("End_To_End_Verification_Node_%d", time.Now().Unix())

	t.Logf("🧪 CDC Sync Test Configuration:")
	t.Logf("   GraphDB: %s (repo: %s)", config.GraphDBURL, config.GraphDBRepo)
	t.Logf("   Go API: %s", config.GoAPIURL)
	t.Logf("   Canary: %s (%s)", canaryCode, canaryLabel)

	// Step 0: Verify Go API is healthy
	t.Run("PreflightHealthCheck", func(t *testing.T) {
		healthy, err := checkGoAPIHealth(config.GoAPIURL)
		if err != nil {
			t.Fatalf("Go API health check failed: %v", err)
		}
		if !healthy {
			t.Fatal("Go API is not healthy")
		}
		t.Log("✅ Go API is healthy")
	})

	// Step 1: Insert canary into GraphDB
	t.Run("InsertCanaryToGraphDB", func(t *testing.T) {
		err := insertCanaryToGraphDB(ctx, config, canaryCode, canaryLabel)
		if err != nil {
			t.Fatalf("Failed to insert canary into GraphDB: %v", err)
		}
		t.Log("✅ Canary inserted into GraphDB")
	})

	// Step 2: Wait for CDC propagation and verify in Go API
	t.Run("VerifyCDCPropagation", func(t *testing.T) {
		found, latency, err := pollForCanary(ctx, config, canaryCode, canaryLabel)
		if err != nil {
			t.Fatalf("Error during CDC verification: %v", err)
		}
		if !found {
			t.Fatalf("❌ CDC Sync failed: Canary not found in Go API after %d seconds", config.MaxWaitSec)
		}
		t.Logf("✅ Canary found in Go API (latency: %v)", latency)

		// Performance assertion
		if latency > 5*time.Second {
			t.Logf("⚠️ CDC latency is high (%v), consider tuning Kafka consumer", latency)
		}
	})

	// Step 3: Cleanup
	t.Run("CleanupCanary", func(t *testing.T) {
		err := deleteCanaryFromGraphDB(ctx, config, canaryCode, canaryLabel)
		if err != nil {
			t.Logf("⚠️ Warning: Failed to cleanup canary: %v", err)
		} else {
			t.Log("✅ Canary cleaned up from GraphDB")
		}
	})
}

// TestNeo4jBridgeConceptLookup tests direct concept lookup through Neo4j bridge
func TestNeo4jBridgeConceptLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := loadTestConfig()

	// Test with known SNOMED concept: Diabetes mellitus type 2
	testCases := []struct {
		name   string
		system string
		code   string
	}{
		{"Type2Diabetes", "SNOMED", "44054006"},
		{"DiabetesMellitus", "SNOMED", "73211009"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/v1/concepts/%s/%s", config.GoAPIURL, tc.system, tc.code)

			start := time.Now()
			resp, err := http.Get(url)
			latency := time.Since(start)

			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected 200, got %d", resp.StatusCode)
			}

			var concept ConceptResponse
			if err := json.NewDecoder(resp.Body).Decode(&concept); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Verify response
			if concept.Code != tc.code {
				t.Errorf("Expected code %s, got %s", tc.code, concept.Code)
			}
			if concept.Display == "" || concept.Display == "?" {
				t.Errorf("Display name is empty or '?': %s", concept.Display)
			}

			// Performance assertion: <50ms target
			if latency > 50*time.Millisecond {
				t.Logf("⚠️ Concept lookup latency (%v) exceeds 50ms target", latency)
			}

			t.Logf("✅ %s: %s (%s) [%v]", tc.name, concept.Display, concept.Backend, latency)
		})
	}
}

// TestSubsumptionViaNeoBridge tests subsumption through Neo4j ELK hierarchy
func TestSubsumptionViaNeoBridge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := loadTestConfig()

	// Test: Type 2 Diabetes IS-A Diabetes Mellitus
	testCases := []struct {
		name           string
		codeA          string
		codeB          string
		system         string
		expectSubsumes bool
	}{
		{"Type2DM_IsA_DM", "44054006", "73211009", "SNOMED", true},
		{"DM_NotA_Type2DM", "73211009", "44054006", "SNOMED", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/v1/subsumption/test", config.GoAPIURL)

			payload := map[string]string{
				"code_a": tc.codeA,
				"code_b": tc.codeB,
				"system": tc.system,
			}
			body, _ := json.Marshal(payload)

			start := time.Now()
			resp, err := http.Post(url, "application/json", bytes.NewReader(body))
			latency := time.Since(start)

			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			subsumes, ok := result["subsumes"].(bool)
			if !ok {
				t.Fatalf("'subsumes' field not found or not boolean")
			}

			if subsumes != tc.expectSubsumes {
				t.Errorf("Expected subsumes=%v, got %v", tc.expectSubsumes, subsumes)
			}

			backend := result["backend"]
			if backend != "neo4j" {
				t.Logf("⚠️ Backend is %v (expected neo4j)", backend)
			}

			// Performance assertion: <100ms target
			if latency > 100*time.Millisecond {
				t.Logf("⚠️ Subsumption latency (%v) exceeds 100ms target", latency)
			}

			t.Logf("✅ %s: subsumes=%v, backend=%v [%v]", tc.name, subsumes, backend, latency)
		})
	}
}

// Helper functions

func checkGoAPIHealth(apiURL string) (bool, error) {
	resp, err := http.Get(apiURL + "/health")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return false, err
	}

	return health.Status == "healthy", nil
}

func insertCanaryToGraphDB(ctx context.Context, config *TestConfig, code, label string) error {
	query := fmt.Sprintf(`
PREFIX snomed: <http://snomed.info/id/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
INSERT DATA {
    snomed:%s a owl:Class ;
        rdfs:label "%s" ;
        skos:prefLabel "%s" .
}`, code, label, label)

	url := fmt.Sprintf("%s/repositories/%s/statements", config.GraphDBURL, config.GraphDBRepo)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(query))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphDB insert returned %d", resp.StatusCode)
	}

	return nil
}

func deleteCanaryFromGraphDB(ctx context.Context, config *TestConfig, code, label string) error {
	query := fmt.Sprintf(`
PREFIX snomed: <http://snomed.info/id/>
DELETE WHERE { snomed:%s ?p ?o }`, code)

	url := fmt.Sprintf("%s/repositories/%s/statements", config.GraphDBURL, config.GraphDBRepo)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(query))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func pollForCanary(ctx context.Context, config *TestConfig, code, label string) (bool, time.Duration, error) {
	url := fmt.Sprintf("%s/v1/concepts/SNOMED/%s", config.GoAPIURL, code)

	start := time.Now()
	deadline := time.Now().Add(time.Duration(config.MaxWaitSec) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false, 0, ctx.Err()
		default:
		}

		resp, err := http.Get(url)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var concept ConceptResponse
			if err := json.NewDecoder(resp.Body).Decode(&concept); err == nil {
				if concept.Display != "" && concept.Display != "?" {
					resp.Body.Close()
					return true, time.Since(start), nil
				}
			}
		}
		resp.Body.Close()

		time.Sleep(500 * time.Millisecond)
	}

	return false, time.Since(start), nil
}
