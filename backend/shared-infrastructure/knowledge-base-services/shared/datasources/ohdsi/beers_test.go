package ohdsi

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

	if client.Name() != "OHDSI_Beers_STOPP" {
		t.Errorf("Expected name 'OHDSI_Beers_STOPP', got '%s'", client.Name())
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BeersVersion != "2023" {
		t.Errorf("Expected BeersVersion '2023', got '%s'", config.BeersVersion)
	}

	if !config.IncludeSTOPP {
		t.Error("IncludeSTOPP should be true by default")
	}

	if config.AthenaURL == "" {
		t.Error("AthenaURL should have a default value")
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

func TestHealthCheck_Loaded(t *testing.T) {
	client := setupTestClient(t)

	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("HealthCheck should succeed when data is loaded: %v", err)
	}
}

func TestClose(t *testing.T) {
	client := setupTestClient(t)

	err := client.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}

	if client.loaded {
		t.Error("loaded should be false after Close")
	}

	if len(client.pimEntries) != 0 {
		t.Error("pimEntries should be empty after Close")
	}
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE TESTS
// =============================================================================

func TestAuthority(t *testing.T) {
	client := NewClient(DefaultConfig())

	meta := client.Authority()

	if meta.Name != "OHDSI_Beers_STOPP" {
		t.Errorf("Expected name 'OHDSI_Beers_STOPP', got '%s'", meta.Name)
	}

	if meta.Level != AuthorityDefinitive {
		t.Errorf("Expected level DEFINITIVE, got '%s'", meta.Level)
	}

	if meta.LLMPolicy != LLMNever {
		t.Errorf("Expected LLM policy NEVER, got '%s'", meta.LLMPolicy)
	}
}

func TestLLMPolicy(t *testing.T) {
	client := NewClient(DefaultConfig())

	policy := client.LLMPolicy()
	if policy != LLMNever {
		t.Errorf("Expected NEVER (Beers is definitive authority), got '%s'", policy)
	}
}

func TestSupportedFactTypes(t *testing.T) {
	client := NewClient(DefaultConfig())

	factTypes := client.SupportedFactTypes()

	if len(factTypes) == 0 {
		t.Error("SupportedFactTypes should return at least one fact type")
	}

	hasGeriatricPIM := false
	for _, ft := range factTypes {
		if ft == FactTypeGeriatricPIM {
			hasGeriatricPIM = true
			break
		}
	}

	if !hasGeriatricPIM {
		t.Error("Should support GERIATRIC_PIM fact type")
	}
}

// =============================================================================
// EMBEDDED DATA LOADING TESTS
// =============================================================================

func TestLoadEmbeddedBeers(t *testing.T) {
	client := NewClient(DefaultConfig())

	ctx := context.Background()
	err := client.loadEmbeddedBeers(ctx)

	if err != nil {
		t.Fatalf("loadEmbeddedBeers failed: %v", err)
	}

	if !client.loaded {
		t.Error("loaded should be true after loadEmbeddedBeers")
	}

	if len(client.pimEntries) == 0 {
		t.Error("Should have embedded PIM entries")
	}

	// Verify we have major categories
	hasAnticholinergics := false
	hasCNS := false
	hasPain := false

	for _, entry := range client.pimEntries {
		switch entry.Category {
		case "ANTICHOLINERGICS":
			hasAnticholinergics = true
		case "CNS":
			hasCNS = true
		case "PAIN":
			hasPain = true
		}
	}

	if !hasAnticholinergics {
		t.Error("Should have anticholinergic entries")
	}
	if !hasCNS {
		t.Error("Should have CNS entries")
	}
	if !hasPain {
		t.Error("Should have pain medication entries")
	}
}

// =============================================================================
// FACT RETRIEVAL TESTS
// =============================================================================

func TestGetFacts_ByRxCUI(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	// RxCUI 596 is alprazolam (benzodiazepine)
	facts, err := client.GetFacts(ctx, "596")

	if err != nil {
		t.Fatalf("GetFacts failed: %v", err)
	}

	if len(facts) == 0 {
		t.Fatal("Expected at least one fact for benzodiazepine")
	}

	fact := facts[0]
	if fact.FactType != FactTypeGeriatricPIM {
		t.Errorf("Expected fact type GERIATRIC_PIM, got %s", fact.FactType)
	}

	if fact.ExtractionMethod != "AUTHORITY_LOOKUP" {
		t.Errorf("Expected extraction method 'AUTHORITY_LOOKUP', got '%s'", fact.ExtractionMethod)
	}

	if fact.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", fact.Confidence)
	}

	if fact.RiskLevel != "HIGH" {
		t.Errorf("Benzodiazepines should have HIGH risk level, got '%s'", fact.RiskLevel)
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
	facts, err := client.GetFactsByName(ctx, "Benzodiazepines")

	if err != nil {
		t.Fatalf("GetFactsByName failed: %v", err)
	}

	if len(facts) == 0 {
		t.Error("Expected facts for Benzodiazepines")
	}
}

func TestGetFactsByName_CaseInsensitive(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()

	facts1, _ := client.GetFactsByName(ctx, "benzodiazepines")
	facts2, _ := client.GetFactsByName(ctx, "BENZODIAZEPINES")
	facts3, _ := client.GetFactsByName(ctx, "Benzodiazepines")

	if len(facts1) != len(facts2) || len(facts2) != len(facts3) {
		t.Error("GetFactsByName should be case-insensitive")
	}
}

func TestGetFactByType(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	fact, err := client.GetFactByType(ctx, "596", FactTypeGeriatricPIM)

	if err != nil {
		t.Fatalf("GetFactByType failed: %v", err)
	}

	if fact == nil {
		t.Fatal("Expected geriatric PIM fact")
	}

	if fact.FactType != FactTypeGeriatricPIM {
		t.Errorf("Expected fact type GERIATRIC_PIM, got %s", fact.FactType)
	}
}

func TestGetFactByType_WrongType(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	// FactType that OHDSI doesn't support
	fact, err := client.GetFactByType(ctx, "596", FactType("WRONG_TYPE"))

	if err != nil {
		t.Fatalf("GetFactByType should not error: %v", err)
	}

	if fact != nil {
		t.Error("Should return nil for unsupported fact type")
	}
}

// =============================================================================
// QUERY METHOD TESTS
// =============================================================================

func TestIsPIM(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()

	// Alprazolam (benzodiazepine) should be a PIM
	isPIM, err := client.IsPIM(ctx, "596")
	if err != nil {
		t.Fatalf("IsPIM failed: %v", err)
	}
	if !isPIM {
		t.Error("Alprazolam should be identified as a PIM")
	}

	// Non-existent drug should not be a PIM
	isPIM, err = client.IsPIM(ctx, "NONEXISTENT")
	if err != nil {
		t.Fatalf("IsPIM failed: %v", err)
	}
	if isPIM {
		t.Error("Non-existent drug should not be a PIM")
	}
}

func TestGetPIMDetails(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	entries, err := client.GetPIMDetails(ctx, "596")

	if err != nil {
		t.Fatalf("GetPIMDetails failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected PIM details for alprazolam")
	}

	entry := entries[0]
	if entry.Source != CriteriaBeers {
		t.Errorf("Expected source BEERS, got %s", entry.Source)
	}

	if entry.Recommendation != RecommendAvoid {
		t.Errorf("Expected recommendation AVOID for benzodiazepines, got %s", entry.Recommendation)
	}

	if entry.AgeThreshold != 65 {
		t.Errorf("Expected age threshold 65, got %d", entry.AgeThreshold)
	}
}

func TestGetPIMsByCategory(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	entries, err := client.GetPIMsByCategory(ctx, BeersCNS)

	if err != nil {
		t.Fatalf("GetPIMsByCategory failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected CNS category entries")
	}

	// Verify all entries are CNS category
	for _, entry := range entries {
		if entry.Category != "CNS" {
			t.Errorf("Expected CNS category, got %s", entry.Category)
		}
	}
}

func TestGetAlternatives(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	alternatives, err := client.GetAlternatives(ctx, "596")

	if err != nil {
		t.Fatalf("GetAlternatives failed: %v", err)
	}

	if len(alternatives) == 0 {
		t.Error("Expected alternative drugs for benzodiazepines")
	}

	// Check for expected alternatives
	hasSSRI := false
	for _, alt := range alternatives {
		if strings.Contains(strings.ToUpper(alt), "SSRI") {
			hasSSRI = true
			break
		}
	}

	if !hasSSRI {
		t.Log("Alternative drugs:", alternatives)
		// SSRIs should be an alternative for benzodiazepines
	}
}

func TestGetAllPIMEntries(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	entries := client.GetAllPIMEntries(ctx)

	if len(entries) == 0 {
		t.Error("Expected PIM entries")
	}

	// Verify it returns a copy (modifying shouldn't affect original)
	originalCount := len(client.pimEntries)
	entries = append(entries, &PIMEntry{ID: "test"})

	if len(client.pimEntries) != originalCount {
		t.Error("GetAllPIMEntries should return a copy")
	}
}

// =============================================================================
// SYNC TESTS
// =============================================================================

func TestSync(t *testing.T) {
	config := DefaultConfig()
	config.DataPath = "/nonexistent/path" // Force use of embedded data
	client := NewClient(config)

	ctx := context.Background()
	result, err := client.Sync(ctx)

	if result == nil {
		t.Fatal("Sync should return a result")
	}

	if err != nil {
		t.Logf("Sync returned error (may be expected): %v", err)
	}

	if result.Authority != "OHDSI_Beers_STOPP" {
		t.Errorf("Expected authority 'OHDSI_Beers_STOPP', got '%s'", result.Authority)
	}

	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	// Should have loaded embedded data
	if result.TotalRecords == 0 && result.Success {
		t.Error("Should have records after successful sync")
	}
}

func TestSyncDelta(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	result, err := client.SyncDelta(ctx, since)

	if result == nil {
		t.Fatal("SyncDelta should return a result")
	}

	if err != nil {
		t.Logf("SyncDelta returned error (may be expected): %v", err)
	}
}

// =============================================================================
// PIM TO FACT CONVERSION TESTS
// =============================================================================

func TestPIMToFact_AvoidRecommendation(t *testing.T) {
	client := NewClient(DefaultConfig())

	entry := &PIMEntry{
		ID:             "test-001",
		Source:         CriteriaBeers,
		DrugName:       "TestDrug",
		Category:       "CNS",
		Recommendation: RecommendAvoid,
		RxCUIs:         []string{"12345"},
		Rationale:      "Test rationale",
		EvidenceQuality: EvidenceHigh,
		Strength:       StrengthStrong,
	}

	fact := client.pimToFact(entry)

	if fact.RiskLevel != "HIGH" {
		t.Errorf("AVOID recommendation should map to HIGH risk, got %s", fact.RiskLevel)
	}

	if fact.ActionRequired != "AVOID" {
		t.Errorf("Expected action AVOID, got %s", fact.ActionRequired)
	}

	if fact.AuthoritySource != "BEERS" {
		t.Errorf("Expected authority BEERS, got %s", fact.AuthoritySource)
	}
}

func TestPIMToFact_CautionRecommendation(t *testing.T) {
	client := NewClient(DefaultConfig())

	entry := &PIMEntry{
		ID:             "test-002",
		Source:         CriteriaBeers,
		DrugName:       "TestDrug2",
		Category:       "CARDIOVASCULAR",
		Recommendation: RecommendUseCaution,
		RxCUIs:         []string{"67890"},
	}

	fact := client.pimToFact(entry)

	if fact.RiskLevel != "MODERATE" {
		t.Errorf("USE_CAUTION recommendation should map to MODERATE risk, got %s", fact.RiskLevel)
	}

	if fact.ActionRequired != "CAUTION" {
		t.Errorf("Expected action CAUTION, got %s", fact.ActionRequired)
	}
}

func TestPIMToFact_FactValueStructure(t *testing.T) {
	client := NewClient(DefaultConfig())

	entry := &PIMEntry{
		ID:              "test-003",
		Source:          CriteriaBeers,
		DrugName:        "TestDrug3",
		DrugClass:       "Test Class",
		Category:        "PAIN",
		Recommendation:  RecommendAvoid,
		Rationale:       "Test rationale",
		AgeThreshold:    65,
		MaxDose:         "100mg",
		MaxDuration:     "7 days",
		AlternativeDrugs: []string{"Drug A", "Drug B"},
		Exceptions:      []string{"Exception 1"},
	}

	fact := client.pimToFact(entry)

	factValue, ok := fact.FactValue.(map[string]interface{})
	if !ok {
		t.Fatal("FactValue should be a map")
	}

	if factValue["category"] != "PAIN" {
		t.Errorf("Expected category PAIN, got %v", factValue["category"])
	}

	if factValue["drug_class"] != "Test Class" {
		t.Errorf("Expected drug_class 'Test Class', got %v", factValue["drug_class"])
	}

	if factValue["age_threshold"] != 65 {
		t.Errorf("Expected age_threshold 65, got %v", factValue["age_threshold"])
	}

	if factValue["max_dose"] != "100mg" {
		t.Errorf("Expected max_dose '100mg', got %v", factValue["max_dose"])
	}

	if factValue["max_duration"] != "7 days" {
		t.Errorf("Expected max_duration '7 days', got %v", factValue["max_duration"])
	}
}

// =============================================================================
// INDEXING TESTS
// =============================================================================

func TestIndexEntry(t *testing.T) {
	client := NewClient(DefaultConfig())

	entry := &PIMEntry{
		ID:        "test-index-001",
		Source:    CriteriaBeers,
		DrugName:  "TestIndexDrug",
		Category:  "CNS",
		RxCUIs:    []string{"111", "222"},
		ATCCodes:  []string{"N05BA"},
	}

	client.indexEntry(entry)

	// Check pimEntries
	if len(client.pimEntries) != 1 {
		t.Errorf("Expected 1 PIM entry, got %d", len(client.pimEntries))
	}

	// Check byRxCUI index
	for _, rxcui := range entry.RxCUIs {
		if _, exists := client.byRxCUI[rxcui]; !exists {
			t.Errorf("Entry should be indexed by RxCUI %s", rxcui)
		}
	}

	// Check byName index
	if _, exists := client.byDrugName["testindexdrug"]; !exists {
		t.Error("Entry should be indexed by lowercase drug name")
	}

	// Check byCategory index
	if _, exists := client.byCategory[BeersCNS]; !exists {
		t.Error("Entry should be indexed by category")
	}

	// Check byATCCode index
	if _, exists := client.byATCCode["N05BA"]; !exists {
		t.Error("Entry should be indexed by ATC code")
	}
}

// =============================================================================
// LOAD FROM JSON TESTS
// =============================================================================

func TestLoadFromJSON(t *testing.T) {
	client := NewClient(DefaultConfig())

	testJSON := `[
		{
			"id": "json-001",
			"source": "BEERS",
			"drug_name": "JSONTestDrug",
			"category": "CNS",
			"rxcuis": ["999"],
			"recommendation": "AVOID",
			"rationale": "Test from JSON",
			"age_threshold": 65
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

	if len(client.pimEntries) != 1 {
		t.Errorf("Expected 1 PIM entry, got %d", len(client.pimEntries))
	}

	// Verify indexed correctly
	if _, exists := client.byRxCUI["999"]; !exists {
		t.Error("Entry should be indexed by RxCUI")
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func setupTestClient(t *testing.T) *Client {
	t.Helper()

	client := NewClient(DefaultConfig())

	ctx := context.Background()
	if err := client.loadEmbeddedBeers(ctx); err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}

	return client
}
