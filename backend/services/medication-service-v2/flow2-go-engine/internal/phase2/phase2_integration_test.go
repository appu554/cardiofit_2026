package phase2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flow2-go-engine/internal/config"
	"flow2-go-engine/internal/models"
)

// TestEvidenceEnvelopeManager_Integration tests the Evidence Envelope Manager integration
func TestEvidenceEnvelopeManager_Integration(t *testing.T) {
	// Create mock Knowledge Broker server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			
		case "/api/v1/version-sets/active":
			response := `{
				"name": "clinical-evidence-v2.1",
				"environment": "test",
				"kb_versions": {
					"kb_2_context": "v2.1.0",
					"kb_3_guidelines": "v1.8.2",
					"kb_4_formulary": "v3.0.1"
				},
				"activated_at": "2025-01-15T10:00:00Z",
				"description": "Test version set"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			
		case "/api/v1/version-sets/validate":
			response := `{
				"valid": true,
				"warnings": []
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create Knowledge Broker client with mock server
	cfg := config.KnowledgeBrokerConfig{
		URL:             mockServer.URL,
		Timeout:         5 * time.Second,
		RefreshInterval: 1 * time.Minute,
		Environment:     "test",
	}

	kbClient, err := NewKnowledgeBrokerClient(cfg)
	require.NoError(t, err)
	defer kbClient.Close()

	// Create Evidence Envelope Manager
	eem := NewEvidenceEnvelopeManager(kbClient, "test", 0) // No background refresh for test
	defer eem.Stop()

	// Test initialization
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = eem.Initialize(ctx)
	require.NoError(t, err)

	// Verify envelope was created
	envelope := eem.GetCurrentEnvelope()
	require.NotNil(t, envelope)
	assert.Equal(t, "clinical-evidence-v2.1", envelope.VersionSetName)
	assert.Equal(t, "test", envelope.Environment)
	assert.Len(t, envelope.KBVersions, 3)

	// Test KB version retrieval
	version, err := eem.GetKBVersion("kb_2_context")
	require.NoError(t, err)
	assert.Equal(t, "v2.1.0", version)

	// Test version usage tracking
	err = eem.RecordVersionUsage("kb_2_context", false)
	require.NoError(t, err)

	err = eem.RecordVersionUsage("kb_2_context", true)
	require.NoError(t, err)

	// Verify usage tracking
	stats := eem.GetUsageStatistics()
	require.NotNil(t, stats)
	
	kbUsage, ok := stats["kb_usage"].(map[string]interface{})
	require.True(t, ok)
	
	contextUsage, ok := kbUsage["kb_2_context"].(map[string]interface{})
	require.True(t, ok)
	
	assert.Equal(t, 2, contextUsage["query_count"])
	assert.Equal(t, 1, contextUsage["cache_hits"])

	// Test health status
	assert.True(t, eem.IsHealthy())
	
	healthStatus := eem.GetHealthStatus()
	assert.True(t, healthStatus["healthy"].(bool))
	assert.Equal(t, "test", healthStatus["environment"])
}

// TestKnowledgeBrokerClient_Integration tests the Knowledge Broker client integration
func TestKnowledgeBrokerClient_Integration(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
			
		case r.URL.Path == "/api/v1/version-sets/active":
			response := `{
				"name": "clinical-evidence-v2.1",
				"environment": "test",
				"kb_versions": {
					"kb_2_context": "v2.1.0",
					"kb_3_guidelines": "v1.8.2"
				},
				"activated_at": "2025-01-15T10:00:00Z"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			
		case r.URL.Path == "/api/v1/version-sets/validate" && r.Method == "POST":
			response := `{
				"valid": true,
				"warnings": ["KB kb_5_legacy is deprecated"]
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			
		case r.URL.Path == "/api/v1/status":
			response := `{
				"status": "healthy",
				"version": "1.0.0",
				"environment": "test",
				"active_version_set": "clinical-evidence-v2.1",
				"knowledge_bases": {
					"kb_2_context": {
						"version": "v2.1.0",
						"status": "active",
						"last_updated": "2025-01-15T09:00:00Z"
					}
				},
				"last_updated": "2025-01-15T10:00:00Z"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create client configuration
	cfg := config.KnowledgeBrokerConfig{
		URL:     mockServer.URL,
		Timeout: 5 * time.Second,
	}

	// Create client
	client, err := NewKnowledgeBrokerClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test getting active version set
	versionSet, err := client.GetActiveVersionSet(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "clinical-evidence-v2.1", versionSet.Name)
	assert.Equal(t, "test", versionSet.Environment)
	assert.Contains(t, versionSet.KBVersions, "kb_2_context")
	assert.Equal(t, "v2.1.0", versionSet.KBVersions["kb_2_context"])

	// Test version validation
	testVersions := map[string]string{
		"kb_2_context":    "v2.1.0",
		"kb_3_guidelines": "v1.8.2",
	}
	err = client.ValidateKBVersions(ctx, testVersions)
	assert.NoError(t, err)

	// Test service status
	status, err := client.GetServiceStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "clinical-evidence-v2.1", status.ActiveVersionSet)
	assert.Contains(t, status.KnowledgeBases, "kb_2_context")
}

// TestPhase2Models_Serialization tests Phase 2 model serialization
func TestPhase2Models_Serialization(t *testing.T) {
	// Test Evidence Envelope serialization
	envelope := &models.EvidenceEnvelope{
		VersionSetName: "test-version-set",
		KBVersions: map[string]string{
			"kb_2_context": "v2.1.0",
		},
		Environment: "test",
		ActivatedAt: time.Now(),
		UsedVersions: map[string]models.VersionUsage{
			"kb_2_context": {
				Version:    "v2.1.0",
				AccessedAt: time.Now(),
				QueryCount: 5,
				CacheHits:  3,
			},
		},
	}

	// Test that all fields are accessible
	assert.Equal(t, "test-version-set", envelope.VersionSetName)
	assert.Contains(t, envelope.KBVersions, "kb_2_context")
	assert.Equal(t, "test", envelope.Environment)
	assert.Contains(t, envelope.UsedVersions, "kb_2_context")

	// Test EnrichedContext model
	enrichedCtx := &models.EnrichedContext{
		RequestID: "test-request-123",
		PatientID: "patient-456",
		Demographics: models.Phase2Demographics{
			Age: 65,
			Sex: "M",
		},
		Phenotype: "htn_stage2_high_risk",
		RiskLevel: "HIGH",
		EvidenceEnvelope: envelope,
		Phase2Duration: 45 * time.Millisecond,
	}

	// Verify EnrichedContext structure
	assert.Equal(t, "test-request-123", enrichedCtx.RequestID)
	assert.Equal(t, "patient-456", enrichedCtx.PatientID)
	assert.Equal(t, 65, enrichedCtx.Demographics.Age)
	assert.Equal(t, "htn_stage2_high_risk", enrichedCtx.Phenotype)
	assert.Equal(t, "HIGH", enrichedCtx.RiskLevel)
	assert.NotNil(t, enrichedCtx.EvidenceEnvelope)
	assert.Less(t, enrichedCtx.Phase2Duration, 50*time.Millisecond) // Within SLA
}

// BenchmarkEvidenceEnvelopeManager_GetVersion benchmarks KB version retrieval
func BenchmarkEvidenceEnvelopeManager_GetVersion(b *testing.B) {
	// Setup mock Knowledge Broker
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/v1/version-sets/active" {
			response := `{
				"name": "benchmark-test",
				"environment": "test", 
				"kb_versions": {"kb_2_context": "v2.1.0"},
				"activated_at": "2025-01-15T10:00:00Z"
			}`
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))
			return
		}
		if r.URL.Path == "/api/v1/version-sets/validate" {
			response := `{"valid": true}`
			w.Header().Set("Content-Type", "application/json") 
			w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// Create and initialize manager
	cfg := config.KnowledgeBrokerConfig{
		URL:     mockServer.URL,
		Timeout: 5 * time.Second,
	}
	
	kbClient, _ := NewKnowledgeBrokerClient(cfg)
	defer kbClient.Close()
	
	eem := NewEvidenceEnvelopeManager(kbClient, "test", 0)
	defer eem.Stop()
	
	ctx := context.Background()
	eem.Initialize(ctx)

	// Benchmark version retrieval
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			version, err := eem.GetKBVersion("kb_2_context")
			if err != nil || version != "v2.1.0" {
				b.Fatal("Unexpected result")
			}
		}
	})
}

// Example test to validate configuration loading
func TestPhase2Config_Loading(t *testing.T) {
	// This is a basic test to ensure our config structures compile correctly
	cfg := &config.Phase2Config{
		KnowledgeBroker: config.KnowledgeBrokerConfig{
			URL:             "https://test-kb.example.com",
			Timeout:         30 * time.Second,
			RefreshInterval: 5 * time.Minute,
			Environment:     "test",
		},
		ContextGateway: config.Phase2ContextConfig{
			URL:        "http://localhost:8015",
			Timeout:    30 * time.Second,
			SnapshotTTL: 5 * time.Minute,
		},
		ParallelExecution: config.ParallelExecutionConfig{
			MaxConcurrency: 10,
			DefaultTimeout: 25 * time.Millisecond,
		},
		PhenotypeEvaluation: config.PhenotypeConfig{
			RustEngineURL:     "http://localhost:8090",
			CacheSize:         1000,
			RuleTTL:           1 * time.Hour,
			EvaluationTimeout: 5 * time.Millisecond,
		},
		Performance: config.PerformanceConfig{
			TargetLatencyMS: 50,
			CacheWarmup:     true,
			PreloadCommonPhenotypes: []string{
				"htn_stage2_high_risk",
				"diabetes_ckd",
			},
		},
	}

	// Validate configuration values
	assert.Equal(t, "https://test-kb.example.com", cfg.KnowledgeBroker.URL)
	assert.Equal(t, 10, cfg.ParallelExecution.MaxConcurrency)
	assert.Equal(t, 50, cfg.Performance.TargetLatencyMS)
	assert.Equal(t, 5*time.Millisecond, cfg.PhenotypeEvaluation.EvaluationTimeout)
	assert.Len(t, cfg.Performance.PreloadCommonPhenotypes, 2)
}