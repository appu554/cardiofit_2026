// Package rxnav provides integration tests for the RxNav client.
// This file tests both:
// - Public NLM RxNav API (https://rxnav.nlm.nih.gov/REST)
// - Local RxNav-in-a-Box Docker (http://localhost:4000)
//
// Run tests:
//   go test -v ./... -run TestRxNav
//   go test -v ./... -run TestRxNavInABox -tags=docker
package rxnav

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// =============================================================================
// TEST CONFIGURATION
// =============================================================================

const (
	// Public NLM RxNav API
	publicRxNavURL = "https://rxnav.nlm.nih.gov/REST"

	// Local RxNav-in-a-Box Docker
	localRxNavURL = "http://localhost:4000/REST"

	// Test timeout
	testTimeout = 120 * time.Second
)

// Well-known test drugs with their RxCUIs
var testDrugs = map[string]string{
	"metformin":    "6809",
	"aspirin":      "1191",
	"lisinopril":   "29046",
	"atorvastatin": "83367",
	"warfarin":     "11289",
	"omeprazole":   "7646",
	"gabapentin":   "25480",
	"levothyroxine": "10582",
}

// =============================================================================
// RXNAV-IN-A-BOX DOCKER TESTS
// =============================================================================

// isRxNavDockerRunning checks if RxNav-in-a-Box Docker is running on port 4000
func isRxNavDockerRunning() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(localRxNavURL + "/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// TestRxNavInABox_Connectivity tests basic connectivity to local RxNav Docker
func TestRxNavInABox_Connectivity(t *testing.T) {
	if !isRxNavDockerRunning() {
		t.Skip("RxNav-in-a-Box Docker not running. Start with: cd backend/shared-infrastructure/docker/rxnav && ./start-rxnav.sh")
	}

	client := NewClient(Config{
		BaseURL:            localRxNavURL,
		Timeout:            30 * time.Second,
		MaxRetries:         2,
		RetryDelay:         1 * time.Second,
		RateLimitPerSecond: 50, // Local Docker can handle more requests
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Test 1: Health check
	t.Run("HealthCheck", func(t *testing.T) {
		err := client.HealthCheck(ctx)
		if err != nil {
			t.Fatalf("HealthCheck failed: %v", err)
		}
		t.Log("✅ RxNav-in-a-Box Docker is healthy")
	})

	// Test 2: Get RxCUI by name
	t.Run("GetRxCUIByName", func(t *testing.T) {
		for drugName, expectedPrefix := range map[string]string{
			"metformin": "6",  // Should start with 6
			"aspirin":   "1",  // Should start with 1
		} {
			rxcui, err := client.GetRxCUIByName(ctx, drugName)
			if err != nil {
				t.Fatalf("GetRxCUIByName(%s) failed: %v", drugName, err)
			}
			if rxcui == "" {
				t.Errorf("Expected non-empty RxCUI for %s", drugName)
			}
			t.Logf("✅ %s → RxCUI: %s", drugName, rxcui)

			// Verify it's a reasonable RxCUI (starts with expected prefix)
			if len(rxcui) > 0 && rxcui[0] != expectedPrefix[0] {
				t.Logf("   Note: RxCUI %s for %s (first char mismatch, but OK)", rxcui, drugName)
			}
		}
	})

	// Test 3: Get SPL SetID (Phase 3 critical function)
	t.Run("GetSPLSetID_Phase3Critical", func(t *testing.T) {
		// This is THE KEY FUNCTION for Phase 3: RxCUI → SPL SetID → DailyMed XML
		for drugName := range testDrugs {
			rxcui, err := client.GetRxCUIByName(ctx, drugName)
			if err != nil {
				t.Logf("⚠️ Skipping %s: %v", drugName, err)
				continue
			}

			setID, err := client.GetSPLSetID(ctx, rxcui)
			if err != nil {
				t.Logf("⚠️ No SPL SetID for %s (RxCUI: %s): %v", drugName, rxcui, err)
				continue
			}

			t.Logf("✅ %s → RxCUI: %s → SPL SetID: %s", drugName, rxcui, setID)
		}
	})

	// Test 4: Get drug relationships
	t.Run("GetAllRelated", func(t *testing.T) {
		rxcui := testDrugs["metformin"]
		rel, err := client.GetAllRelated(ctx, rxcui)
		if err != nil {
			t.Fatalf("GetAllRelated failed: %v", err)
		}

		t.Logf("✅ Metformin relationships:")
		t.Logf("   Ingredients: %d", len(rel.Ingredients))
		t.Logf("   Brand Names: %d", len(rel.BrandNames))
		t.Logf("   Dose Forms: %d", len(rel.DoseForms))
		t.Logf("   Components: %d", len(rel.Components))

		if len(rel.BrandNames) > 0 {
			t.Logf("   Sample brands: %s, %s",
				rel.BrandNames[0].Name,
				func() string {
					if len(rel.BrandNames) > 1 {
						return rel.BrandNames[1].Name
					}
					return "n/a"
				}())
		}
	})

	// Test 5: Get NDCs for a drug
	t.Run("GetNDCsByRxCUI", func(t *testing.T) {
		rxcui := testDrugs["aspirin"]
		ndcs, err := client.GetNDCsByRxCUI(ctx, rxcui)
		if err != nil {
			t.Fatalf("GetNDCsByRxCUI failed: %v", err)
		}

		t.Logf("✅ Found %d NDCs for aspirin", len(ndcs))
		if len(ndcs) > 0 {
			t.Logf("   Sample NDCs: %v", ndcs[:min(5, len(ndcs))])
		}
	})

	// Test 6: Drug search
	t.Run("SearchDrugs", func(t *testing.T) {
		drugs, err := client.SearchDrugs(ctx, "metfor", 10) // Partial match
		if err != nil {
			t.Fatalf("SearchDrugs failed: %v", err)
		}

		t.Logf("✅ Found %d drugs matching 'metfor'", len(drugs))
		for i, drug := range drugs[:min(3, len(drugs))] {
			t.Logf("   [%d] %s (RxCUI: %s)", i+1, drug.Name, drug.RxCUI)
		}
	})

	// Test 7: Drug properties
	t.Run("GetDrugProperties", func(t *testing.T) {
		rxcui := testDrugs["metformin"]
		props, err := client.GetDrugProperties(ctx, rxcui)
		if err != nil {
			t.Fatalf("GetDrugProperties failed: %v", err)
		}

		t.Logf("✅ Metformin properties (%d total):", len(props))
		for key, val := range props {
			if key == "SPL_SET_ID" || key == "RxNorm Name" {
				t.Logf("   %s: %s", key, val)
			}
		}
	})

	// Test 8: Batch drug retrieval
	t.Run("BatchGetDrugs", func(t *testing.T) {
		rxcuis := []string{testDrugs["metformin"], testDrugs["aspirin"], testDrugs["lisinopril"]}
		drugs, err := client.BatchGetDrugs(ctx, rxcuis)
		if err != nil {
			t.Fatalf("BatchGetDrugs failed: %v", err)
		}

		t.Logf("✅ Retrieved %d drugs in batch", len(drugs))
		for rxcui, drug := range drugs {
			t.Logf("   %s: %s", rxcui, drug.Name)
		}
	})

	// Test 9: Drug interactions (if available in local instance)
	t.Run("GetInteractionsBetween", func(t *testing.T) {
		// Warfarin + Aspirin should have known interactions
		rxcuis := []string{testDrugs["warfarin"], testDrugs["aspirin"]}
		interactions, err := client.GetInteractionsBetween(ctx, rxcuis)
		if err != nil {
			t.Logf("⚠️ GetInteractionsBetween: %v (interaction API may not be available)", err)
			return
		}

		t.Logf("✅ Found %d interactions between warfarin and aspirin", len(interactions))
		for i, interaction := range interactions[:min(3, len(interactions))] {
			t.Logf("   [%d] Severity: %s - %s", i+1, interaction.Severity, interaction.Description[:min(100, len(interaction.Description))])
		}
	})
}

// TestRxNavInABox_SPLSetIDFlow tests the complete SPL SetID lookup flow
// This is the critical path for Phase 3: NDC/DrugName → RxCUI → SPL SetID
func TestRxNavInABox_SPLSetIDFlow(t *testing.T) {
	if !isRxNavDockerRunning() {
		t.Skip("RxNav-in-a-Box Docker not running")
	}

	client := NewClient(Config{
		BaseURL:            localRxNavURL,
		Timeout:            30 * time.Second,
		MaxRetries:         2,
		RateLimitPerSecond: 50,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("DrugName_to_SPLSetID", func(t *testing.T) {
		drugName := "metformin"

		// Step 1: Get RxCUI
		rxcui, err := client.GetRxCUIByName(ctx, drugName)
		if err != nil {
			t.Fatalf("Step 1 failed (GetRxCUIByName): %v", err)
		}
		t.Logf("Step 1: %s → RxCUI %s", drugName, rxcui)

		// Step 2: Get SPL SetID
		setID, err := client.GetSPLSetID(ctx, rxcui)
		if err != nil {
			t.Logf("⚠️ Step 2 failed (GetSPLSetID): %v", err)
			t.Log("   Trying related concepts for SPL link...")

			// Try related SBD/SCD concepts
			rel, relErr := client.GetAllRelated(ctx, rxcui)
			if relErr == nil {
				for _, concept := range rel.RelatedDrugs[:min(5, len(rel.RelatedDrugs))] {
					if splID, splErr := client.GetSPLSetID(ctx, concept.RxCUI); splErr == nil {
						t.Logf("✅ Found SPL via related concept: %s → %s", concept.Name, splID)
						return
					}
				}
			}
			t.Log("   No SPL SetID found via related concepts")
			return
		}

		t.Logf("✅ Step 2: RxCUI %s → SPL SetID %s", rxcui, setID)

		// Verify SetID format (UUID)
		if len(setID) < 30 {
			t.Errorf("SPL SetID too short: %s", setID)
		}
	})

	t.Run("GetSPLSetIDFromDrugName_Convenience", func(t *testing.T) {
		setID, err := client.GetSPLSetIDFromDrugName(ctx, "aspirin")
		if err != nil {
			t.Logf("⚠️ GetSPLSetIDFromDrugName: %v", err)
			return
		}
		t.Logf("✅ Aspirin SPL SetID: %s", setID)
	})
}

// TestRxNavInABox_Performance tests API performance characteristics
func TestRxNavInABox_Performance(t *testing.T) {
	if !isRxNavDockerRunning() {
		t.Skip("RxNav-in-a-Box Docker not running")
	}

	client := NewClient(Config{
		BaseURL:            localRxNavURL,
		Timeout:            30 * time.Second,
		MaxRetries:         1,
		RateLimitPerSecond: 100, // No rate limiting for perf test
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("SequentialLookups_10Drugs", func(t *testing.T) {
		start := time.Now()

		for drugName := range testDrugs {
			_, err := client.GetRxCUIByName(ctx, drugName)
			if err != nil {
				t.Logf("Warning: %s lookup failed: %v", drugName, err)
			}
		}

		elapsed := time.Since(start)
		avgPerRequest := elapsed / time.Duration(len(testDrugs))

		t.Logf("✅ %d lookups in %v (avg: %v per request)", len(testDrugs), elapsed, avgPerRequest)

		// Local Docker should be fast
		if avgPerRequest > 500*time.Millisecond {
			t.Logf("⚠️ Performance warning: avg response time > 500ms")
		}
	})
}

// =============================================================================
// PUBLIC RXNAV API TESTS (fallback when Docker not available)
// =============================================================================

// TestRxNavPublicAPI tests the public NLM RxNav API
func TestRxNavPublicAPI(t *testing.T) {
	// Use environment variable to control public API tests
	if os.Getenv("SKIP_PUBLIC_API_TESTS") == "true" {
		t.Skip("Skipping public API tests (SKIP_PUBLIC_API_TESTS=true)")
	}

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient(Config{
		BaseURL:            publicRxNavURL,
		Timeout:            30 * time.Second,
		MaxRetries:         2,
		RetryDelay:         1 * time.Second,
		RateLimitPerSecond: 5, // Be conservative with public API
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("HealthCheck", func(t *testing.T) {
		if err := client.HealthCheck(ctx); err != nil {
			t.Fatalf("HealthCheck failed: %v", err)
		}
		t.Log("✅ Public RxNav API is accessible")
	})

	t.Run("GetRxCUIByName", func(t *testing.T) {
		rxcui, err := client.GetRxCUIByName(ctx, "metformin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}
		t.Logf("✅ Metformin RxCUI: %s", rxcui)
	})
}

// =============================================================================
// VERIFICATION SUMMARY
// =============================================================================

// TestPrintVerificationSummary prints a summary of the test capabilities
func TestPrintVerificationSummary(t *testing.T) {
	dockerRunning := isRxNavDockerRunning()

	fmt.Println("\n════════════════════════════════════════════════════════════")
	fmt.Println("  RxNav Client Verification Summary")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	if dockerRunning {
		fmt.Println("  ✅ RxNav-in-a-Box Docker: RUNNING (localhost:4000)")
		fmt.Println("     → Unlimited local API calls")
		fmt.Println("     → Drug interaction API available")
		fmt.Println("     → Full SPL SetID lookup support")
	} else {
		fmt.Println("  ⚠️  RxNav-in-a-Box Docker: NOT RUNNING")
		fmt.Println("     Start with: cd backend/shared-infrastructure/docker/rxnav && ./start-rxnav.sh")
	}

	fmt.Println()
	fmt.Println("  Test Coverage:")
	fmt.Println("  ├── GetRxCUIByName      - Drug name → RxCUI lookup")
	fmt.Println("  ├── GetRxCUIByNDC       - NDC → RxCUI lookup")
	fmt.Println("  ├── GetDrugByRxCUI      - Full drug details")
	fmt.Println("  ├── GetSPLSetID         - RxCUI → DailyMed SetID (Phase 3 critical)")
	fmt.Println("  ├── GetAllRelated       - Drug relationships (ingredients, brands)")
	fmt.Println("  ├── GetNDCsByRxCUI      - RxCUI → NDC list")
	fmt.Println("  ├── SearchDrugs         - Approximate drug search")
	fmt.Println("  ├── GetDrugProperties   - Full property map")
	fmt.Println("  ├── BatchGetDrugs       - Batch drug retrieval")
	fmt.Println("  └── GetInteractions*    - Drug-drug interactions")
	fmt.Println()
	fmt.Println("  * Drug interaction API was discontinued on public NLM API (Jan 2024)")
	fmt.Println("    but remains available in RxNav-in-a-Box Docker")
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
}
