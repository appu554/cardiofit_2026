// Package routing provides authority-based routing for clinical fact extraction.
// This file contains unit tests for the AuthorityRouter.
package routing

import (
	"context"
	"testing"
	"time"

	"github.com/cardiofit/shared/datasources"
)

// =============================================================================
// MOCK AUTHORITY CLIENT
// =============================================================================

// MockAuthorityClient implements AuthorityClient for testing
type MockAuthorityClient struct {
	name            string
	authorityLevel  AuthorityLevel
	llmPolicy       LLMPolicy
	supportedFacts  []FactType
	facts           []AuthorityFact
	syncCalled      bool
	syncDeltaCalled bool
}

// Local type aliases to avoid import cycles
type AuthorityLevel = datasources.AuthorityLevel
type LLMPolicy = datasources.LLMPolicy
type FactType = datasources.FactType
type AuthorityFact = datasources.AuthorityFact
type AuthorityMetadata = datasources.AuthorityMetadata
type SyncResult = datasources.SyncResult

// DataSource interface methods
func (m *MockAuthorityClient) Name() string                            { return m.name }
func (m *MockAuthorityClient) HealthCheck(ctx context.Context) error   { return nil }
func (m *MockAuthorityClient) Close() error                            { return nil }

// AuthorityClient interface methods
func (m *MockAuthorityClient) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	return m.facts, nil
}

func (m *MockAuthorityClient) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	return m.facts, nil
}

func (m *MockAuthorityClient) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	for _, f := range m.facts {
		if f.FactType == factType {
			return &f, nil
		}
	}
	return nil, nil
}

func (m *MockAuthorityClient) Sync(ctx context.Context) (*SyncResult, error) {
	m.syncCalled = true
	return &SyncResult{TotalFacts: 100, NewFacts: 10, UpdatedFacts: 5, Success: true}, nil
}

func (m *MockAuthorityClient) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	m.syncDeltaCalled = true
	return &SyncResult{TotalFacts: 10, NewFacts: 5, UpdatedFacts: 2, Success: true}, nil
}

func (m *MockAuthorityClient) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:           m.name,
		AuthorityLevel: m.authorityLevel,
		Description:    "Mock authority for testing",
	}
}

func (m *MockAuthorityClient) SupportedFactTypes() []FactType {
	return m.supportedFacts
}

func (m *MockAuthorityClient) LLMPolicy() LLMPolicy {
	return m.llmPolicy
}

// Helper to create mock client
func newMockClient(name string, level AuthorityLevel, policy LLMPolicy, facts []FactType) *MockAuthorityClient {
	return &MockAuthorityClient{
		name:           name,
		authorityLevel: level,
		llmPolicy:      policy,
		supportedFacts: facts,
		facts:          []AuthorityFact{},
	}
}

// =============================================================================
// INITIALIZATION TESTS
// =============================================================================

func TestNewAuthorityRouter(t *testing.T) {
	router := NewAuthorityRouter()

	if router == nil {
		t.Fatal("NewAuthorityRouter returned nil")
	}

	// Verify default LOINC routes are initialized
	routes := router.GetLOINCRoutes()
	if len(routes) == 0 {
		t.Error("LOINC routes should be initialized by default")
	}

	// Verify specific LOINC routes exist
	expectedLOINCCodes := []string{
		LOINCNursingMothers,       // 34080-2
		LOINCGeriatricUse,         // 34082-8
		LOINCClinicalPharmacology, // 34090-1
		LOINCWarningsPrecautions,  // 43685-7
		LOINCDosageAdministration, // 34068-7
		LOINCDrugInteractions,     // 34073-7
	}

	for _, loinc := range expectedLOINCCodes {
		if _, exists := routes[loinc]; !exists {
			t.Errorf("Expected LOINC route for %s not found", loinc)
		}
	}
}

// =============================================================================
// AUTHORITY REGISTRATION TESTS
// =============================================================================

func TestRegisterAuthority(t *testing.T) {
	router := NewAuthorityRouter()
	mockClient := newMockClient("TestAuthority", datasources.AuthorityDefinitive, datasources.LLMNever, nil)

	// First registration should succeed
	err := router.RegisterAuthority("TestAuthority", mockClient)
	if err != nil {
		t.Errorf("First registration should succeed: %v", err)
	}

	// Duplicate registration should fail
	err = router.RegisterAuthority("TestAuthority", mockClient)
	if err == nil {
		t.Error("Duplicate registration should return error")
	}
}

func TestGetAuthority(t *testing.T) {
	router := NewAuthorityRouter()
	mockClient := newMockClient("TestAuth", datasources.AuthorityDefinitive, datasources.LLMNever, nil)

	// Register authority
	router.RegisterAuthority("TestAuth", mockClient)

	// Should retrieve registered authority
	client, exists := router.GetAuthority("TestAuth")
	if !exists {
		t.Error("Registered authority should exist")
	}
	if client == nil {
		t.Error("Retrieved client should not be nil")
	}

	// Non-existent authority should return false
	_, exists = router.GetAuthority("NonExistent")
	if exists {
		t.Error("Non-existent authority should not exist")
	}
}

func TestListAuthorities(t *testing.T) {
	router := NewAuthorityRouter()

	// Initially empty
	if len(router.ListAuthorities()) != 0 {
		t.Error("Initially should have no authorities")
	}

	// Register some authorities
	router.RegisterAuthority("Auth1", newMockClient("Auth1", datasources.AuthorityDefinitive, datasources.LLMNever, nil))
	router.RegisterAuthority("Auth2", newMockClient("Auth2", datasources.AuthorityPrimary, datasources.LLMGapFillOnly, nil))

	authorities := router.ListAuthorities()
	if len(authorities) != 2 {
		t.Errorf("Expected 2 authorities, got %d", len(authorities))
	}
}

// =============================================================================
// LOINC ROUTING TESTS
// =============================================================================

func TestRouteByLOINC_NursingMothers(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 34080-2 (NURSING MOTHERS) should route to LactMed
	decision := router.RouteByLOINC(LOINCNursingMothers, "")

	if decision.PrimaryAuthority != AuthorityLactMed {
		t.Errorf("Expected LactMed, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMNever {
		t.Errorf("Expected LLMNever policy, got %s", decision.LLMPolicy)
	}
	if decision.FactType != datasources.FactTypeLactationSafety {
		t.Errorf("Expected FactTypeLactationSafety, got %s", decision.FactType)
	}
	if decision.LOINCCode != LOINCNursingMothers {
		t.Errorf("Expected LOINC %s, got %s", LOINCNursingMothers, decision.LOINCCode)
	}
}

func TestRouteByLOINC_GeriatricUse(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 34082-8 (GERIATRIC USE) should route to OHDSI
	decision := router.RouteByLOINC(LOINCGeriatricUse, "")

	if decision.PrimaryAuthority != AuthorityOHDSI {
		t.Errorf("Expected OHDSI, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMNever {
		t.Errorf("Expected LLMNever policy, got %s", decision.LLMPolicy)
	}
}

func TestRouteByLOINC_ClinicalPharmacology(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 34090-1 (CLINICAL PHARMACOLOGY) should route to DrugBank
	decision := router.RouteByLOINC(LOINCClinicalPharmacology, "")

	if decision.PrimaryAuthority != AuthorityDrugBank {
		t.Errorf("Expected DrugBank, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMGapFillOnly {
		t.Errorf("Expected LLMGapFillOnly policy, got %s", decision.LLMPolicy)
	}
}

func TestRouteByLOINC_UnknownCode(t *testing.T) {
	router := NewAuthorityRouter()

	// Unknown LOINC code should fall back to LLM consensus
	decision := router.RouteByLOINC("99999-9", "")

	if decision.PrimaryAuthority != "" {
		t.Errorf("Expected no primary authority, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMWithConsensus {
		t.Errorf("Expected LLMWithConsensus fallback policy, got %s", decision.LLMPolicy)
	}
	if decision.ExtractionMethod != "LLM_CONSENSUS" {
		t.Errorf("Expected LLM_CONSENSUS extraction method, got %s", decision.ExtractionMethod)
	}
}

// =============================================================================
// CONTENT-TRIGGERED ROUTING TESTS
// =============================================================================

func TestRouteByLOINC_WarningsWithQTContent(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 43685-7 (WARNINGS) with QT content should route to CredibleMeds
	content := "This drug may cause QT prolongation and torsade de pointes"
	decision := router.RouteByLOINC(LOINCWarningsPrecautions, content)

	if decision.PrimaryAuthority != AuthorityCredibleMeds {
		t.Errorf("Expected CredibleMeds for QT content, got %s", decision.PrimaryAuthority)
	}
	if decision.FactType != datasources.FactTypeQTProlongation {
		t.Errorf("Expected FactTypeQTProlongation, got %s", decision.FactType)
	}
	if decision.LLMPolicy != datasources.LLMNever {
		t.Errorf("Expected LLMNever policy for QT risk, got %s", decision.LLMPolicy)
	}
}

func TestRouteByLOINC_WarningsWithHepatotoxicityContent(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 43685-7 (WARNINGS) with hepatotoxicity content should route to LiverTox
	content := "Hepatotoxicity has been reported. Monitor ALT and AST levels."
	decision := router.RouteByLOINC(LOINCWarningsPrecautions, content)

	if decision.PrimaryAuthority != AuthorityLiverTox {
		t.Errorf("Expected LiverTox for hepatotoxicity content, got %s", decision.PrimaryAuthority)
	}
	if decision.FactType != datasources.FactTypeHepatotoxicity {
		t.Errorf("Expected FactTypeHepatotoxicity, got %s", decision.FactType)
	}
}

func TestRouteByLOINC_DosageWithPGxContent(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 34068-7 (DOSAGE) with pharmacogenomics content should route to CPIC
	content := "CYP2D6 poor metabolizers may require dose adjustment"
	decision := router.RouteByLOINC(LOINCDosageAdministration, content)

	if decision.PrimaryAuthority != AuthorityCPIC {
		t.Errorf("Expected CPIC for PGx content, got %s", decision.PrimaryAuthority)
	}
	if decision.FactType != datasources.FactTypePharmacogenomics {
		t.Errorf("Expected FactTypePharmacogenomics, got %s", decision.FactType)
	}
	if decision.LLMPolicy != datasources.LLMNever {
		t.Errorf("Expected LLMNever policy for CPIC, got %s", decision.LLMPolicy)
	}
}

func TestRouteByLOINC_DosageWithRenalContent(t *testing.T) {
	router := NewAuthorityRouter()

	// LOINC 34068-7 (DOSAGE) with renal dosing content should route to FDA SPL
	content := "For patients with CrCl < 30 mL/min, reduce dose by 50%"
	decision := router.RouteByLOINC(LOINCDosageAdministration, content)

	if decision.PrimaryAuthority != AuthorityFDASPL {
		t.Errorf("Expected FDA_SPL for renal dosing content, got %s", decision.PrimaryAuthority)
	}
	if decision.FactType != datasources.FactTypeRenalDosing {
		t.Errorf("Expected FactTypeRenalDosing, got %s", decision.FactType)
	}
	if decision.LLMPolicy != datasources.LLMWithConsensus {
		t.Errorf("Expected LLMWithConsensus for SPL table extraction, got %s", decision.LLMPolicy)
	}
}

// =============================================================================
// FACT TYPE ROUTING TESTS
// =============================================================================

func TestRouteByFactType_LactationSafety(t *testing.T) {
	router := NewAuthorityRouter()

	decision := router.RouteByFactType(datasources.FactTypeLactationSafety)

	if decision.PrimaryAuthority != AuthorityLactMed {
		t.Errorf("Expected LactMed for lactation safety, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMNever {
		t.Errorf("Expected LLMNever policy, got %s", decision.LLMPolicy)
	}
	if decision.ExtractionMethod != "AUTHORITY_LOOKUP" {
		t.Errorf("Expected AUTHORITY_LOOKUP, got %s", decision.ExtractionMethod)
	}
}

func TestRouteByFactType_QTProlongation(t *testing.T) {
	router := NewAuthorityRouter()

	decision := router.RouteByFactType(datasources.FactTypeQTProlongation)

	if decision.PrimaryAuthority != AuthorityCredibleMeds {
		t.Errorf("Expected CredibleMeds for QT prolongation, got %s", decision.PrimaryAuthority)
	}
}

func TestRouteByFactType_Hepatotoxicity(t *testing.T) {
	router := NewAuthorityRouter()

	decision := router.RouteByFactType(datasources.FactTypeHepatotoxicity)

	if decision.PrimaryAuthority != AuthorityLiverTox {
		t.Errorf("Expected LiverTox for hepatotoxicity, got %s", decision.PrimaryAuthority)
	}
}

func TestRouteByFactType_Pharmacogenomics(t *testing.T) {
	router := NewAuthorityRouter()

	decision := router.RouteByFactType(datasources.FactTypePharmacogenomics)

	if decision.PrimaryAuthority != AuthorityCPIC {
		t.Errorf("Expected CPIC for pharmacogenomics, got %s", decision.PrimaryAuthority)
	}
}

func TestRouteByFactType_GeriatricPIM(t *testing.T) {
	router := NewAuthorityRouter()

	decision := router.RouteByFactType(datasources.FactTypeGeriatricPIM)

	if decision.PrimaryAuthority != AuthorityOHDSI {
		t.Errorf("Expected OHDSI for geriatric PIM, got %s", decision.PrimaryAuthority)
	}
}

func TestRouteByFactType_PKParameters(t *testing.T) {
	router := NewAuthorityRouter()

	decision := router.RouteByFactType(datasources.FactTypePKParameters)

	if decision.PrimaryAuthority != AuthorityDrugBank {
		t.Errorf("Expected DrugBank for PK parameters, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMGapFillOnly {
		t.Errorf("Expected LLMGapFillOnly for DrugBank, got %s", decision.LLMPolicy)
	}
}

func TestRouteByFactType_UnknownType(t *testing.T) {
	router := NewAuthorityRouter()

	// Unknown fact type should fall back to LLM consensus
	decision := router.RouteByFactType(datasources.FactType("UnknownFactType"))

	if decision.PrimaryAuthority != "" {
		t.Errorf("Expected no primary authority for unknown fact type, got %s", decision.PrimaryAuthority)
	}
	if decision.LLMPolicy != datasources.LLMWithConsensus {
		t.Errorf("Expected LLMWithConsensus fallback, got %s", decision.LLMPolicy)
	}
}

// =============================================================================
// CONTENT-BASED ROUTING TESTS
// =============================================================================

func TestRouteByContent_QTProlongation(t *testing.T) {
	router := NewAuthorityRouter()

	content := "Warning: QT prolongation and torsades de pointes have been reported."
	decisions := router.RouteByContent(content)

	if len(decisions) == 0 {
		t.Fatal("Expected routing decisions for QT content")
	}

	found := false
	for _, d := range decisions {
		if d.PrimaryAuthority == AuthorityCredibleMeds && d.FactType == datasources.FactTypeQTProlongation {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected CredibleMeds route for QT prolongation content")
	}
}

func TestRouteByContent_Hepatotoxicity(t *testing.T) {
	router := NewAuthorityRouter()

	content := "Hepatotoxicity and drug-induced liver injury (DILI) have been reported."
	decisions := router.RouteByContent(content)

	found := false
	for _, d := range decisions {
		if d.PrimaryAuthority == AuthorityLiverTox && d.FactType == datasources.FactTypeHepatotoxicity {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected LiverTox route for hepatotoxicity content")
	}
}

func TestRouteByContent_Breastfeeding(t *testing.T) {
	router := NewAuthorityRouter()

	content := "Use with caution during breastfeeding. Present in breast milk with RID of 2%."
	decisions := router.RouteByContent(content)

	found := false
	for _, d := range decisions {
		if d.PrimaryAuthority == AuthorityLactMed && d.FactType == datasources.FactTypeLactationSafety {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected LactMed route for breastfeeding content")
	}
}

func TestRouteByContent_Pharmacogenomics(t *testing.T) {
	router := NewAuthorityRouter()

	content := "CYP2D6 poor metabolizers may experience increased drug exposure."
	decisions := router.RouteByContent(content)

	found := false
	for _, d := range decisions {
		if d.PrimaryAuthority == AuthorityCPIC && d.FactType == datasources.FactTypePharmacogenomics {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected CPIC route for pharmacogenomics content")
	}
}

func TestRouteByContent_Geriatric(t *testing.T) {
	router := NewAuthorityRouter()

	content := "Elderly patients should be monitored. Listed in Beers criteria."
	decisions := router.RouteByContent(content)

	found := false
	for _, d := range decisions {
		if d.PrimaryAuthority == AuthorityOHDSI && d.FactType == datasources.FactTypeGeriatricPIM {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected OHDSI route for geriatric/Beers criteria content")
	}
}

func TestRouteByContent_NoMatch(t *testing.T) {
	router := NewAuthorityRouter()

	content := "Take this medication with food for better absorption."
	decisions := router.RouteByContent(content)

	if len(decisions) != 0 {
		t.Errorf("Expected no routing decisions for generic content, got %d", len(decisions))
	}
}

func TestRouteByContent_MultipleMatches(t *testing.T) {
	router := NewAuthorityRouter()

	// Content that triggers multiple authorities
	content := "This drug may cause QT prolongation and hepatotoxicity. Monitor ECG and liver function."
	decisions := router.RouteByContent(content)

	if len(decisions) < 2 {
		t.Errorf("Expected at least 2 routing decisions, got %d", len(decisions))
	}

	hasQT := false
	hasHepato := false
	for _, d := range decisions {
		if d.PrimaryAuthority == AuthorityCredibleMeds {
			hasQT = true
		}
		if d.PrimaryAuthority == AuthorityLiverTox {
			hasHepato = true
		}
	}
	if !hasQT {
		t.Error("Expected CredibleMeds route for QT content")
	}
	if !hasHepato {
		t.Error("Expected LiverTox route for hepatotoxicity content")
	}
}

// =============================================================================
// FACT RETRIEVAL TESTS
// =============================================================================

func TestGetFacts_WithRegisteredAuthority(t *testing.T) {
	router := NewAuthorityRouter()

	// Create mock with test facts
	mockClient := newMockClient(AuthorityLactMed, datasources.AuthorityDefinitive, datasources.LLMNever,
		[]FactType{datasources.FactTypeLactationSafety})
	mockClient.facts = []AuthorityFact{
		{
			FactType:      datasources.FactTypeLactationSafety,
			RxCUI:     "12345",
			Content:     "Compatible with breastfeeding",
			AuthoritySource:    "LactMed",
			EvidenceLevel: "Level 1",
		},
	}
	router.RegisterAuthority(AuthorityLactMed, mockClient)

	// Get facts for lactation safety
	ctx := context.Background()
	facts, err := router.GetFacts(ctx, "12345", datasources.FactTypeLactationSafety)

	if err != nil {
		t.Fatalf("GetFacts returned error: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("Expected 1 fact, got %d", len(facts))
	}
	if facts[0].Content != "Compatible with breastfeeding" {
		t.Errorf("Unexpected fact content: %v", facts[0].Content)
	}
}

func TestGetFacts_UnregisteredAuthority(t *testing.T) {
	router := NewAuthorityRouter()

	// Don't register the authority - should fail
	ctx := context.Background()
	_, err := router.GetFacts(ctx, "12345", datasources.FactTypeLactationSafety)

	if err == nil {
		t.Error("Expected error for unregistered authority")
	}
}

func TestGetFacts_NoRouteForFactType(t *testing.T) {
	router := NewAuthorityRouter()

	ctx := context.Background()
	_, err := router.GetFacts(ctx, "12345", datasources.FactType("UnknownType"))

	if err == nil {
		t.Error("Expected error for unknown fact type")
	}
}

func TestGetFactsForSection_WithMatchingAuthority(t *testing.T) {
	router := NewAuthorityRouter()

	// Register LactMed mock
	mockClient := newMockClient(AuthorityLactMed, datasources.AuthorityDefinitive, datasources.LLMNever,
		[]FactType{datasources.FactTypeLactationSafety})
	mockClient.facts = []AuthorityFact{
		{
			FactType:   datasources.FactTypeLactationSafety,
			RxCUI:  "54321",
			Content:  "Monitor infant for sedation",
			AuthoritySource: "LactMed",
		},
	}
	router.RegisterAuthority(AuthorityLactMed, mockClient)

	ctx := context.Background()
	facts, err := router.GetFactsForSection(ctx, "54321", LOINCNursingMothers, "")

	if err != nil {
		t.Fatalf("GetFactsForSection returned error: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("Expected 1 fact, got %d", len(facts))
	}
}

func TestGetFactsForSection_NoAuthorityRoute(t *testing.T) {
	router := NewAuthorityRouter()

	ctx := context.Background()
	facts, err := router.GetFactsForSection(ctx, "12345", "99999-9", "Generic content")

	if err != nil {
		t.Fatalf("Unexpected error for no route: %v", err)
	}
	if facts != nil {
		t.Error("Expected nil facts when no authority route exists")
	}
}

// =============================================================================
// INTROSPECTION TESTS
// =============================================================================

func TestGetLOINCRoutes_ReturnsCopy(t *testing.T) {
	router := NewAuthorityRouter()

	routes1 := router.GetLOINCRoutes()
	routes2 := router.GetLOINCRoutes()

	// Modify routes1, should not affect routes2
	delete(routes1, LOINCNursingMothers)

	if _, exists := routes2[LOINCNursingMothers]; !exists {
		t.Error("GetLOINCRoutes should return a copy, not the original map")
	}
}

func TestGetRoutingDecisionSummary_KnownLOINC(t *testing.T) {
	router := NewAuthorityRouter()

	summary := router.GetRoutingDecisionSummary(LOINCNursingMothers)

	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if !contains(summary, "LactMed") {
		t.Errorf("Summary should mention LactMed: %s", summary)
	}
	if !contains(summary, LOINCNursingMothers) {
		t.Errorf("Summary should mention LOINC code: %s", summary)
	}
}

func TestGetRoutingDecisionSummary_UnknownLOINC(t *testing.T) {
	router := NewAuthorityRouter()

	summary := router.GetRoutingDecisionSummary("99999-9")

	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if !contains(summary, "No authority route") {
		t.Errorf("Summary should indicate no route: %s", summary)
	}
}

// =============================================================================
// CONCURRENCY TESTS
// =============================================================================

func TestConcurrentAccess(t *testing.T) {
	router := NewAuthorityRouter()

	// Register some authorities
	for i := 0; i < 5; i++ {
		name := string(rune('A' + i))
		router.RegisterAuthority(name, newMockClient(name, datasources.AuthorityDefinitive, datasources.LLMNever, nil))
	}

	// Concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				router.RouteByLOINC(LOINCNursingMothers, "test content")
				router.RouteByFactType(datasources.FactTypeLactationSafety)
				router.ListAuthorities()
				router.GetLOINCRoutes()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// LLM POLICY TESTS
// =============================================================================

func TestLLMPolicyForAuthorities(t *testing.T) {
	router := NewAuthorityRouter()

	tests := []struct {
		loincCode      string
		expectedPolicy datasources.LLMPolicy
		description    string
	}{
		{LOINCNursingMothers, datasources.LLMNever, "LactMed should be LLMNever"},
		{LOINCGeriatricUse, datasources.LLMNever, "OHDSI should be LLMNever"},
		{LOINCClinicalPharmacology, datasources.LLMGapFillOnly, "DrugBank should be LLMGapFillOnly"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			decision := router.RouteByLOINC(tt.loincCode, "")
			if decision.LLMPolicy != tt.expectedPolicy {
				t.Errorf("Expected %s, got %s", tt.expectedPolicy, decision.LLMPolicy)
			}
		})
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
