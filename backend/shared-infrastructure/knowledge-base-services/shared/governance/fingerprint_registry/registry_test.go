package fingerprint_registry_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	. "github.com/cardiofit/shared/governance/fingerprint_registry"
	"github.com/cardiofit/shared/rules"
)

// =============================================================================
// IN-MEMORY REGISTRY TESTS
// =============================================================================

func TestNewInMemoryRegistry(t *testing.T) {
	registry := NewInMemoryRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}
}

func TestInMemoryRegistry_Exists_NotFound(t *testing.T) {
	registry := NewInMemoryRegistry()

	ctx := context.Background()
	exists, err := registry.Exists(ctx, "nonexistent-hash")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exists {
		t.Error("Expected hash to not exist")
	}
}

func TestInMemoryRegistry_Register_ThenExists(t *testing.T) {
	registry := NewInMemoryRegistry()

	rule := createTestDraftRule()

	ctx := context.Background()
	err := registry.Register(ctx, rule)

	if err != nil {
		t.Fatalf("Unexpected error on register: %v", err)
	}

	exists, err := registry.Exists(ctx, rule.SemanticFingerprint.Hash)

	if err != nil {
		t.Fatalf("Unexpected error on exists: %v", err)
	}

	if !exists {
		t.Error("Expected hash to exist after registration")
	}
}

func TestInMemoryRegistry_Register_SameFingerprint(t *testing.T) {
	registry := NewInMemoryRegistry()

	rule1 := createTestDraftRule()
	rule2 := createTestDraftRuleWithSameFingerprint(rule1)

	ctx := context.Background()

	// Register first rule
	err := registry.Register(ctx, rule1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Register same rule again (different provenance, same fingerprint)
	err = registry.Register(ctx, rule2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify hash still exists
	exists, err := registry.Exists(ctx, rule1.SemanticFingerprint.Hash)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !exists {
		t.Error("Expected hash to still exist after second registration")
	}
}

func TestInMemoryRegistry_GetRuleByFingerprint(t *testing.T) {
	registry := NewInMemoryRegistry()

	rule := createTestDraftRule()

	ctx := context.Background()
	err := registry.Register(ctx, rule)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ruleID, err := registry.GetRuleByFingerprint(ctx, rule.SemanticFingerprint.Hash)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if ruleID == nil {
		t.Fatal("Expected rule ID to be returned")
	}

	if *ruleID != rule.RuleID {
		t.Errorf("Expected rule ID %s, got %s", rule.RuleID, *ruleID)
	}
}

func TestInMemoryRegistry_GetRuleByFingerprint_NotFound(t *testing.T) {
	registry := NewInMemoryRegistry()

	ctx := context.Background()
	ruleID, err := registry.GetRuleByFingerprint(ctx, "nonexistent-hash")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if ruleID != nil {
		t.Error("Expected nil rule ID for nonexistent hash")
	}
}

// =============================================================================
// ENTRY TESTS
// =============================================================================

func TestEntry_Fields(t *testing.T) {
	entry := Entry{
		Hash:        "abc123def456",
		RuleID:      uuid.New(),
		Domain:      "KB-1",
		RuleType:    "DOSING",
		Version:     1,
		CreatedAt:   time.Now(),
		SourceCount: 3,
	}

	if entry.Hash == "" {
		t.Error("Expected hash to be set")
	}

	if entry.Domain != "KB-1" {
		t.Errorf("Expected domain KB-1, got %s", entry.Domain)
	}

	if entry.SourceCount != 3 {
		t.Errorf("Expected source count 3, got %d", entry.SourceCount)
	}
}

// =============================================================================
// STATS TESTS
// =============================================================================

func TestStats_Fields(t *testing.T) {
	stats := Stats{
		TotalFingerprints: 100,
		UniqueRules:       80,
		TotalSources:      150,
		DuplicationRate:   0.33,
		ByDomain: map[string]int64{
			"KB-1": 50,
			"KB-4": 30,
			"KB-5": 20,
		},
		ByRuleType: map[string]int64{
			"DOSING":          60,
			"CONTRAINDICATION": 25,
			"INTERACTION":      15,
		},
		CacheHitRate: 0.85,
		LastUpdated:  time.Now(),
	}

	if stats.TotalFingerprints != 100 {
		t.Errorf("Expected 100 total fingerprints, got %d", stats.TotalFingerprints)
	}

	if stats.DuplicationRate != 0.33 {
		t.Errorf("Expected 0.33 duplication rate, got %f", stats.DuplicationRate)
	}

	if len(stats.ByDomain) != 3 {
		t.Errorf("Expected 3 domains, got %d", len(stats.ByDomain))
	}
}

// =============================================================================
// CONCURRENT ACCESS TESTS
// =============================================================================

func TestInMemoryRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewInMemoryRegistry()
	ctx := context.Background()

	// Run concurrent registrations and lookups
	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func() {
			rule := createTestDraftRule()
			registry.Register(ctx, rule)
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		go func() {
			registry.Exists(ctx, "some-hash")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createTestDraftRule() *rules.DraftRule {
	condition := rules.Condition{
		Variable: "renal_function.crcl",
		Operator: rules.OpLessThan,
		Value:    ptrFloat64(30),
		Unit:     "ml/min",
	}

	action := rules.Action{
		Effect: rules.EffectContraindicated,
	}

	provenance := rules.Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
		DocumentID:       "test-doc-001",
		SectionCode:      "34068-7",
		ExtractionMethod: "TABLE_PARSE",
	}

	return rules.NewDraftRule("KB-1", rules.RuleTypeDosing, condition, action, provenance)
}

func createTestDraftRuleWithSameFingerprint(original *rules.DraftRule) *rules.DraftRule {
	// Same condition and action = same fingerprint
	condition := original.Condition
	action := original.Action

	// Different provenance
	provenance := rules.Provenance{
		SourceDocumentID: uuid.New(), // Different document
		SourceType:       "CPIC",      // Different source type
		DocumentID:       "cpic-guideline-001",
		SectionCode:      "recommendations",
		ExtractionMethod: "AUTHORITY_LOOKUP",
	}

	return rules.NewDraftRule(original.Domain, original.RuleType, condition, action, provenance)
}

func ptrFloat64(f float64) *float64 {
	return &f
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkInMemoryRegistry_Register(b *testing.B) {
	registry := NewInMemoryRegistry()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rule := createTestDraftRule()
		registry.Register(ctx, rule)
	}
}

func BenchmarkInMemoryRegistry_Exists(b *testing.B) {
	registry := NewInMemoryRegistry()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		rule := createTestDraftRule()
		registry.Register(ctx, rule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Exists(ctx, "some-hash-"+string(rune(i%100)))
	}
}
