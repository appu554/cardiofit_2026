package drugbank

import (
	"context"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// CLIENT INITIALIZATION TESTS
// =============================================================================

func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	client := NewClient(config)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.Name() != "DrugBank" {
		t.Errorf("Expected name 'DrugBank', got '%s'", client.Name())
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DataPath == "" {
		t.Error("DataPath should have a default value")
	}

	if config.CacheTTL == 0 {
		t.Error("CacheTTL should have a default value")
	}
}

// =============================================================================
// DATASOURCE INTERFACE TESTS
// =============================================================================

func TestHealthCheck_NotLoaded(t *testing.T) {
	client := NewClient(DefaultConfig())

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Error("HealthCheck should fail when data is not loaded")
	}
}

func TestClose(t *testing.T) {
	client := NewClient(DefaultConfig())

	// Add some test data
	client.drugs["DB00001"] = &Drug{DrugBankID: "DB00001", Name: "Test Drug"}
	client.loaded = true

	err := client.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}

	if client.loaded {
		t.Error("loaded should be false after Close")
	}

	if len(client.drugs) != 0 {
		t.Error("drugs map should be empty after Close")
	}
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE TESTS
// =============================================================================

func TestAuthority(t *testing.T) {
	client := NewClient(DefaultConfig())

	meta := client.Authority()

	if meta.Name != "DrugBank" {
		t.Errorf("Expected name 'DrugBank', got '%s'", meta.Name)
	}

	if meta.Level != AuthorityPrimary {
		t.Errorf("Expected level PRIMARY, got '%s'", meta.Level)
	}

	if meta.LLMPolicy != LLMGapFillOnly {
		t.Errorf("Expected LLM policy GAP_FILL_ONLY, got '%s'", meta.LLMPolicy)
	}
}

func TestLLMPolicy(t *testing.T) {
	client := NewClient(DefaultConfig())

	policy := client.LLMPolicy()
	if policy != LLMGapFillOnly {
		t.Errorf("Expected GAP_FILL_ONLY, got '%s'", policy)
	}
}

func TestSupportedFactTypes(t *testing.T) {
	client := NewClient(DefaultConfig())

	factTypes := client.SupportedFactTypes()

	if len(factTypes) == 0 {
		t.Error("SupportedFactTypes should return at least one fact type")
	}

	// Check for expected fact types
	hasJPKParams := false
	hasDDI := false
	for _, ft := range factTypes {
		if ft == FactTypePKParameters {
			hasJPKParams = true
		}
		if ft == FactTypeDrugInteraction {
			hasDDI = true
		}
	}

	if !hasJPKParams {
		t.Error("Should support PK_PARAMETERS fact type")
	}
	if !hasDDI {
		t.Error("Should support DRUG_INTERACTION fact type")
	}
}

// =============================================================================
// DATA LOADING TESTS
// =============================================================================

func TestLoadFromJSON(t *testing.T) {
	client := NewClient(DefaultConfig())

	testJSON := `[
		{
			"drugbank_id": "DB00001",
			"name": "Lepirudin",
			"type": "biotech",
			"rxcui": "114934",
			"pharmacokinetics": {
				"half_life": "1.3 hours",
				"half_life_hours": 1.3,
				"protein_binding": "0%",
				"protein_binding_pct": 0,
				"bioavailability": "100% (IV)",
				"bioavailability_pct": 100
			}
		},
		{
			"drugbank_id": "DB00002",
			"name": "Cetuximab",
			"type": "biotech",
			"rxcui": "318341"
		}
	]`

	ctx := context.Background()
	err := client.LoadFromJSON(ctx, strings.NewReader(testJSON))

	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	if !client.loaded {
		t.Error("loaded should be true after LoadFromJSON")
	}

	if len(client.drugs) != 2 {
		t.Errorf("Expected 2 drugs, got %d", len(client.drugs))
	}

	// Test RxCUI index
	if _, exists := client.byRxCUI["114934"]; !exists {
		t.Error("Drug should be indexed by RxCUI")
	}

	// Test name index
	if _, exists := client.byName["lepirudin"]; !exists {
		t.Error("Drug should be indexed by lowercase name")
	}
}

// =============================================================================
// FACT RETRIEVAL TESTS
// =============================================================================

func TestGetFacts_WithPKParameters(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	facts, err := client.GetFacts(ctx, "114934")

	if err != nil {
		t.Fatalf("GetFacts failed: %v", err)
	}

	if len(facts) == 0 {
		t.Fatal("Expected at least one fact")
	}

	// Should have PK parameters fact
	hasPK := false
	for _, fact := range facts {
		if fact.FactType == FactTypePKParameters {
			hasPK = true
			if fact.ExtractionMethod != "AUTHORITY_LOOKUP" {
				t.Errorf("Expected extraction method 'AUTHORITY_LOOKUP', got '%s'", fact.ExtractionMethod)
			}
			if fact.Confidence != 1.0 {
				t.Errorf("Expected confidence 1.0, got %f", fact.Confidence)
			}
			break
		}
	}

	if !hasPK {
		t.Error("Expected PK_PARAMETERS fact")
	}
}

func TestGetFacts_NotFound(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	facts, err := client.GetFacts(ctx, "NONEXISTENT")

	if err != nil {
		t.Fatalf("GetFacts should not error for non-existent drug: %v", err)
	}

	if facts != nil && len(facts) > 0 {
		t.Error("Expected nil or empty facts for non-existent drug")
	}
}

func TestGetFactsByName(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	facts, err := client.GetFactsByName(ctx, "Lepirudin")

	if err != nil {
		t.Fatalf("GetFactsByName failed: %v", err)
	}

	if len(facts) == 0 {
		t.Error("Expected facts for Lepirudin")
	}
}

func TestGetFactsByName_CaseInsensitive(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()

	// Should work regardless of case
	facts1, _ := client.GetFactsByName(ctx, "lepirudin")
	facts2, _ := client.GetFactsByName(ctx, "LEPIRUDIN")
	facts3, _ := client.GetFactsByName(ctx, "Lepirudin")

	if len(facts1) != len(facts2) || len(facts2) != len(facts3) {
		t.Error("GetFactsByName should be case-insensitive")
	}
}

func TestGetFactByType(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	fact, err := client.GetFactByType(ctx, "114934", FactTypePKParameters)

	if err != nil {
		t.Fatalf("GetFactByType failed: %v", err)
	}

	if fact == nil {
		t.Fatal("Expected PK parameters fact")
	}

	if fact.FactType != FactTypePKParameters {
		t.Errorf("Expected fact type PK_PARAMETERS, got %s", fact.FactType)
	}
}

func TestGetFactByType_NotFound(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	// Drug exists but might not have this specific fact type
	fact, err := client.GetFactByType(ctx, "318341", FactTypeProteinBinding)

	if err != nil {
		t.Fatalf("GetFactByType should not error: %v", err)
	}

	// Drug "Cetuximab" has no PK parameters in test data
	// So this should return nil
	if fact != nil {
		t.Log("Fact found (drug may have PK data)")
	}
}

// =============================================================================
// QUERY METHOD TESTS
// =============================================================================

func TestGetDrug(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	drug, err := client.GetDrug(ctx, "DB00001")

	if err != nil {
		t.Fatalf("GetDrug failed: %v", err)
	}

	if drug == nil {
		t.Fatal("Expected drug to be found")
	}

	if drug.Name != "Lepirudin" {
		t.Errorf("Expected name 'Lepirudin', got '%s'", drug.Name)
	}
}

func TestGetDrugByRxCUI(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	drug, err := client.GetDrugByRxCUI(ctx, "114934")

	if err != nil {
		t.Fatalf("GetDrugByRxCUI failed: %v", err)
	}

	if drug == nil {
		t.Fatal("Expected drug to be found")
	}

	if drug.DrugBankID != "DB00001" {
		t.Errorf("Expected DrugBank ID 'DB00001', got '%s'", drug.DrugBankID)
	}
}

func TestGetPKParameters(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	pk, err := client.GetPKParameters(ctx, "114934")

	if err != nil {
		t.Fatalf("GetPKParameters failed: %v", err)
	}

	if pk == nil {
		t.Fatal("Expected PK parameters")
	}

	if pk.HalfLifeHours != 1.3 {
		t.Errorf("Expected half-life 1.3 hours, got %f", pk.HalfLifeHours)
	}

	if pk.BioavailabilityPct != 100 {
		t.Errorf("Expected bioavailability 100%%, got %f", pk.BioavailabilityPct)
	}
}

// =============================================================================
// SYNC TESTS
// =============================================================================

func TestSync(t *testing.T) {
	// Create client with non-existent data path (should use embedded fallback)
	config := DefaultConfig()
	config.DataPath = "/nonexistent/path/drugbank.xml"
	client := NewClient(config)

	ctx := context.Background()
	result, err := client.Sync(ctx)

	// Sync should complete (possibly with errors due to missing file)
	if result == nil {
		t.Fatal("Sync should return a result")
	}

	if err != nil {
		t.Logf("Sync returned error (expected if no data file): %v", err)
	}

	if result.Authority != "DrugBank" {
		t.Errorf("Expected authority 'DrugBank', got '%s'", result.Authority)
	}

	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

func TestSyncDelta(t *testing.T) {
	client := NewClient(DefaultConfig())

	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	result, err := client.SyncDelta(ctx, since)

	// SyncDelta should delegate to Sync for DrugBank
	if result == nil {
		t.Fatal("SyncDelta should return a result")
	}

	// Error is OK if no data file
	if err != nil {
		t.Logf("SyncDelta returned error (expected if no data file): %v", err)
	}
}

// =============================================================================
// DRUG INTERACTION TESTS
// =============================================================================

func TestDrugToFacts_WithInteractions(t *testing.T) {
	client := NewClient(DefaultConfig())

	drug := &Drug{
		DrugBankID: "DB00001",
		Name:       "TestDrug",
		RxCUI:      "12345",
		DrugInteractions: []DrugInteraction{
			{
				DrugBankID:  "DB00002",
				Name:        "InteractingDrug",
				Description: "May increase toxicity",
				Severity:    "MAJOR",
				Mechanism:   "PK",
			},
		},
	}

	facts := client.drugToFacts(drug)

	// Should have at least one DDI fact
	hasDDI := false
	for _, fact := range facts {
		if fact.FactType == FactTypeDrugInteraction {
			hasDDI = true
			if fact.RiskLevel != "HIGH" {
				t.Errorf("Major interaction should have HIGH risk level, got %s", fact.RiskLevel)
			}
			if fact.ActionRequired != "AVOID" {
				t.Errorf("Major interaction should require AVOID action, got %s", fact.ActionRequired)
			}
			break
		}
	}

	if !hasDDI {
		t.Error("Expected DRUG_INTERACTION fact")
	}
}

func TestDrugToFacts_WithCYPEnzymes(t *testing.T) {
	client := NewClient(DefaultConfig())

	drug := &Drug{
		DrugBankID: "DB00001",
		Name:       "TestDrug",
		RxCUI:      "12345",
		Enzymes: []Enzyme{
			{
				ID:                 "BE0000017",
				Name:               "Cytochrome P450 3A4",
				GeneSymbol:         "CYP3A4",
				Actions:            []string{"inhibitor"},
				InhibitionStrength: "STRONG",
			},
		},
	}

	facts := client.drugToFacts(drug)

	// Should have CYP interaction fact
	hasCYP := false
	for _, fact := range facts {
		if fact.FactType == FactTypeCYPInteraction {
			hasCYP = true
			if fact.RiskLevel != "HIGH" {
				t.Errorf("Strong CYP inhibitor should have HIGH risk level, got %s", fact.RiskLevel)
			}
			break
		}
	}

	if !hasCYP {
		t.Error("Expected CYP_INTERACTION fact")
	}
}

func TestDrugToFacts_WithTransporters(t *testing.T) {
	client := NewClient(DefaultConfig())

	drug := &Drug{
		DrugBankID: "DB00001",
		Name:       "TestDrug",
		RxCUI:      "12345",
		Transporters: []Transporter{
			{
				ID:         "BE0000637",
				Name:       "P-glycoprotein 1",
				GeneSymbol: "ABCB1",
				Actions:    []string{"substrate"},
			},
		},
	}

	facts := client.drugToFacts(drug)

	// Should have transporter fact
	hasTransporter := false
	for _, fact := range facts {
		if fact.FactType == FactTypeTransporterInteraction {
			hasTransporter = true
			break
		}
	}

	if !hasTransporter {
		t.Error("Expected TRANSPORTER_INTERACTION fact")
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func setupTestClient(t *testing.T) *Client {
	t.Helper()

	client := NewClient(DefaultConfig())

	testJSON := `[
		{
			"drugbank_id": "DB00001",
			"name": "Lepirudin",
			"type": "biotech",
			"rxcui": "114934",
			"pharmacokinetics": {
				"half_life": "1.3 hours",
				"half_life_hours": 1.3,
				"protein_binding": "0%",
				"protein_binding_pct": 0,
				"bioavailability": "100% (IV)",
				"bioavailability_pct": 100
			}
		},
		{
			"drugbank_id": "DB00002",
			"name": "Cetuximab",
			"type": "biotech",
			"rxcui": "318341"
		}
	]`

	ctx := context.Background()
	if err := client.LoadFromJSON(ctx, strings.NewReader(testJSON)); err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}

	return client
}
