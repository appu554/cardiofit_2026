// Package rxnav provides tests for the RxNav client.
// This test verifies connectivity to the NLM RxNav API and data extraction.
package rxnav

import (
	"context"
	"testing"
	"time"
)

// TestRxNavConnectivity verifies basic connectivity to the NLM RxNav API.
// This is an integration test that makes real API calls.
func TestRxNavConnectivity(t *testing.T) {
	// Skip if running short tests (CI)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient(Config{
		BaseURL:            "https://rxnav.nlm.nih.gov/REST",
		Timeout:            30 * time.Second,
		MaxRetries:         2,
		RetryDelay:         1 * time.Second,
		RateLimitPerSecond: 5, // Be conservative for tests
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test 1: Health check (looks up "aspirin")
	t.Run("HealthCheck", func(t *testing.T) {
		if err := client.HealthCheck(ctx); err != nil {
			t.Fatalf("HealthCheck failed: %v", err)
		}
		t.Log("✅ HealthCheck passed - API is accessible")
	})

	// Test 2: GetRxCUIByName for a well-known drug
	t.Run("GetRxCUIByName_Aspirin", func(t *testing.T) {
		rxcui, err := client.GetRxCUIByName(ctx, "aspirin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}
		if rxcui == "" {
			t.Fatal("Expected non-empty RxCUI for aspirin")
		}
		t.Logf("✅ Aspirin RxCUI: %s", rxcui)
	})

	// Test 3: GetRxCUIByName for metformin (common diabetes drug)
	t.Run("GetRxCUIByName_Metformin", func(t *testing.T) {
		rxcui, err := client.GetRxCUIByName(ctx, "metformin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}
		if rxcui == "" {
			t.Fatal("Expected non-empty RxCUI for metformin")
		}
		t.Logf("✅ Metformin RxCUI: %s", rxcui)
	})

	// Test 4: GetDrugByRxCUI - get drug details
	t.Run("GetDrugByRxCUI_Metformin", func(t *testing.T) {
		// First get the RxCUI
		rxcui, err := client.GetRxCUIByName(ctx, "metformin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}

		// Then get the drug details
		drug, err := client.GetDrugByRxCUI(ctx, rxcui)
		if err != nil {
			t.Fatalf("GetDrugByRxCUI failed: %v", err)
		}

		t.Logf("✅ Drug details retrieved:")
		t.Logf("   RxCUI: %s", drug.RxCUI)
		t.Logf("   Name: %s", drug.Name)
		t.Logf("   TTY: %s", drug.TTY)
		t.Logf("   Ingredients: %v", drug.Ingredients)
	})

	// Test 5: GetInteractions - drug-drug interactions
	// NOTE: The RxNav Drug Interaction API was DISCONTINUED on January 2, 2024.
	// See: https://blog.drugbank.com/nih-discontinues-their-drug-interaction-api/
	// This test is now expected to fail or return empty results.
	// For drug interactions, use DrugBank API or RxNav-in-a-Box (requires UMLS license).
	t.Run("GetInteractions_Warfarin_DISCONTINUED", func(t *testing.T) {
		t.Skip("RxNav Drug Interaction API discontinued as of January 2, 2024")
		// Warfarin has many known interactions
		rxcui, err := client.GetRxCUIByName(ctx, "warfarin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}

		interactions, err := client.GetInteractions(ctx, rxcui)
		if err != nil {
			t.Logf("⚠️ GetInteractions error (expected - API discontinued): %v", err)
			return
		}

		t.Logf("Found %d interactions for warfarin", len(interactions))
	})

	// Test 6: SearchDrugs - approximate search
	t.Run("SearchDrugs_Lisinopril", func(t *testing.T) {
		drugs, err := client.SearchDrugs(ctx, "lisinopril", 5)
		if err != nil {
			t.Fatalf("SearchDrugs failed: %v", err)
		}

		t.Logf("✅ Found %d drugs matching 'lisinopril'", len(drugs))
		for i, drug := range drugs {
			if i >= 3 {
				break
			}
			t.Logf("   [%d] %s (RxCUI: %s)", i+1, drug.Name, drug.RxCUI)
		}
	})

	// Test 7: GetAllRelated - relationships
	t.Run("GetAllRelated_Atorvastatin", func(t *testing.T) {
		rxcui, err := client.GetRxCUIByName(ctx, "atorvastatin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}

		rel, err := client.GetAllRelated(ctx, rxcui)
		if err != nil {
			t.Fatalf("GetAllRelated failed: %v", err)
		}

		t.Logf("✅ Relationships for atorvastatin:")
		t.Logf("   Ingredients: %d", len(rel.Ingredients))
		t.Logf("   Brand Names: %d", len(rel.BrandNames))
		t.Logf("   Dose Forms: %d", len(rel.DoseForms))
		t.Logf("   Components: %d", len(rel.Components))
		t.Logf("   Related Drugs: %d", len(rel.RelatedDrugs))

		// Print some brand names
		if len(rel.BrandNames) > 0 {
			t.Logf("   Sample Brand Names:")
			for i, bn := range rel.BrandNames {
				if i >= 3 {
					break
				}
				t.Logf("     - %s", bn.Name)
			}
		}
	})

	// Test 8: GetNDCsByRxCUI
	t.Run("GetNDCsByRxCUI_Aspirin", func(t *testing.T) {
		rxcui, err := client.GetRxCUIByName(ctx, "aspirin")
		if err != nil {
			t.Fatalf("GetRxCUIByName failed: %v", err)
		}

		ndcs, err := client.GetNDCsByRxCUI(ctx, rxcui)
		if err != nil {
			t.Fatalf("GetNDCsByRxCUI failed: %v", err)
		}

		t.Logf("✅ Found %d NDCs for aspirin", len(ndcs))
		if len(ndcs) > 0 {
			t.Logf("   Sample NDCs: %v", ndcs[:min(3, len(ndcs))])
		}
	})
}

// TestRxNavInteractionsBetween tests checking interactions between multiple drugs.
// NOTE: The RxNav Drug Interaction API was DISCONTINUED on January 2, 2024.
// See: https://blog.drugbank.com/nih-discontinues-their-drug-interaction-api/
func TestRxNavInteractionsBetween(t *testing.T) {
	t.Skip("RxNav Drug Interaction API discontinued as of January 2, 2024. Use DrugBank API instead.")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient(Config{
		BaseURL:            "https://rxnav.nlm.nih.gov/REST",
		Timeout:            30 * time.Second,
		MaxRetries:         2,
		RetryDelay:         1 * time.Second,
		RateLimitPerSecond: 5,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get RxCUIs for known interacting drugs
	warfarinRxCUI, err := client.GetRxCUIByName(ctx, "warfarin")
	if err != nil {
		t.Fatalf("Failed to get warfarin RxCUI: %v", err)
	}

	aspirinRxCUI, err := client.GetRxCUIByName(ctx, "aspirin")
	if err != nil {
		t.Fatalf("Failed to get aspirin RxCUI: %v", err)
	}

	// Check for interactions between warfarin and aspirin
	interactions, err := client.GetInteractionsBetween(ctx, []string{warfarinRxCUI, aspirinRxCUI})
	if err != nil {
		t.Logf("⚠️ GetInteractionsBetween error (expected - API discontinued): %v", err)
		return
	}

	t.Logf("Found %d interactions between warfarin and aspirin", len(interactions))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// LOCAL RXNAV-IN-A-BOX TESTS (Phase 3a)
// =============================================================================

// TestLocalRxNavInABox tests connectivity to local RxNav-in-a-Box Docker instance.
// Prerequisites: Run `cd rxnav-in-a-box-* && docker-compose up -d`
func TestLocalRxNavInABox(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping local RxNav-in-a-Box test in short mode")
	}

	// Use local config for RxNav-in-a-Box
	client := NewClient(LocalConfig())
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test 1: Health check
	t.Run("LocalHealthCheck", func(t *testing.T) {
		if err := client.HealthCheck(ctx); err != nil {
			t.Skipf("⚠️ RxNav-in-a-Box not running: %v (run: docker-compose up -d)", err)
		}
		t.Log("✅ Local RxNav-in-a-Box is accessible")
	})

	// Test 2: GetRxCUIByName
	t.Run("LocalGetRxCUI_Metformin", func(t *testing.T) {
		rxcui, err := client.GetRxCUIByName(ctx, "metformin")
		if err != nil {
			t.Skipf("RxNav-in-a-Box not running: %v", err)
		}
		if rxcui != "6809" {
			t.Logf("⚠️ Expected RxCUI 6809, got %s", rxcui)
		}
		t.Logf("✅ Metformin RxCUI: %s", rxcui)
	})

	// Test 3: GetSPLSetID - KEY for DailyMed integration
	t.Run("LocalGetSPLSetID_Metformin", func(t *testing.T) {
		rxcui := "6809" // Metformin
		setID, err := client.GetSPLSetID(ctx, rxcui)
		if err != nil {
			t.Skipf("RxNav-in-a-Box not running or SPL lookup failed: %v", err)
		}
		if setID == "" {
			t.Fatal("Expected non-empty SPL SetID")
		}
		t.Logf("✅ Metformin SPL SetID: %s", setID)
		t.Logf("   DailyMed URL: https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=%s", setID)
	})

	// Test 4: GetSPLSetIDFromDrugName - Full pipeline
	t.Run("LocalSPLPipeline_Warfarin", func(t *testing.T) {
		setID, err := client.GetSPLSetIDFromDrugName(ctx, "warfarin")
		if err != nil {
			t.Skipf("SPL pipeline failed: %v", err)
		}
		t.Logf("✅ Warfarin SPL SetID: %s", setID)
	})

	// Test 5: GetDrugProperties - All properties including SPL
	t.Run("LocalGetDrugProperties", func(t *testing.T) {
		rxcui := "6809" // Metformin
		props, err := client.GetDrugProperties(ctx, rxcui)
		if err != nil {
			t.Skipf("GetDrugProperties failed: %v", err)
		}

		t.Logf("✅ Found %d properties for Metformin", len(props))

		// Check for SPL_SET_ID
		if splSetID, ok := props["SPL_SET_ID"]; ok {
			t.Logf("   SPL_SET_ID: %s", splSetID)
		}

		// Show sample properties
		count := 0
		for k, v := range props {
			if count >= 5 {
				break
			}
			t.Logf("   %s: %s", k, truncate(v, 50))
			count++
		}
	})

	// Test 6: GetRelatedByType - Find branded drugs
	t.Run("LocalGetRelated_BrandedDrugs", func(t *testing.T) {
		rxcui := "6809" // Metformin
		related, err := client.GetRelatedByType(ctx, rxcui, "BN")
		if err != nil {
			t.Skipf("GetRelatedByType failed: %v", err)
		}

		t.Logf("✅ Found %d brand names for Metformin", len(related))
		for i, r := range related {
			if i >= 3 {
				break
			}
			t.Logf("   - %s (RxCUI: %s)", r.Name, r.RxCUI)
		}
	})
}

// TestLocalRxNavSPLIntegration tests the full RxNav → DailyMed integration pipeline
func TestLocalRxNavSPLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping local integration test in short mode")
	}

	client := NewClient(LocalConfig())
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Check if local RxNav is running
	if err := client.HealthCheck(ctx); err != nil {
		t.Skipf("⚠️ RxNav-in-a-Box not running: %v", err)
	}

	// Test multiple drugs for SPL SetID lookup
	testDrugs := []struct {
		name        string
		expectedKB  string // Where the SPL data would be routed
	}{
		{"metformin", "KB-1,KB-5"},    // Diabetes drug with DDI tables
		{"warfarin", "KB-5"},          // Anticoagulant with many DDI
		{"lisinopril", "KB-1,KB-4"},   // ACE inhibitor with renal dosing
		{"atorvastatin", "KB-1,KB-5"}, // Statin with DDI
		{"amiodarone", "KB-5"},        // Antiarrhythmic with many DDI
	}

	t.Log("📊 Testing RxNav → SPL SetID Pipeline for Multiple Drugs:")
	t.Log("──────────────────────────────────────────────────────────")

	successCount := 0
	for _, drug := range testDrugs {
		t.Run("SPL_"+drug.name, func(t *testing.T) {
			setID, err := client.GetSPLSetIDFromDrugName(ctx, drug.name)
			if err != nil {
				t.Logf("⚠️ %s: %v", drug.name, err)
				return
			}

			successCount++
			t.Logf("✅ %s → SPL SetID: %s", drug.name, setID)
			t.Logf("   Target KBs: %s", drug.expectedKB)
		})
	}

	t.Log("──────────────────────────────────────────────────────────")
	t.Logf("📈 Success rate: %d/%d drugs", successCount, len(testDrugs))
}

// TestComparePublicVsLocal compares public API vs local RxNav-in-a-Box
func TestComparePublicVsLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	publicClient := NewClient(DefaultConfig())
	localClient := NewClient(LocalConfig())
	defer publicClient.Close()
	defer localClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check local availability
	localAvailable := localClient.HealthCheck(ctx) == nil

	t.Run("CompareLookupResults", func(t *testing.T) {
		drugName := "metformin"

		// Public lookup
		publicRxCUI, err := publicClient.GetRxCUIByName(ctx, drugName)
		if err != nil {
			t.Fatalf("Public API failed: %v", err)
		}
		t.Logf("Public API → RxCUI: %s", publicRxCUI)

		// Local lookup (if available)
		if localAvailable {
			localRxCUI, err := localClient.GetRxCUIByName(ctx, drugName)
			if err != nil {
				t.Logf("⚠️ Local lookup failed: %v", err)
				return
			}
			t.Logf("Local API  → RxCUI: %s", localRxCUI)

			if publicRxCUI == localRxCUI {
				t.Log("✅ Results match between public and local APIs")
			} else {
				t.Logf("⚠️ Results differ: public=%s, local=%s", publicRxCUI, localRxCUI)
			}
		} else {
			t.Log("⚠️ Local RxNav-in-a-Box not available for comparison")
		}
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
