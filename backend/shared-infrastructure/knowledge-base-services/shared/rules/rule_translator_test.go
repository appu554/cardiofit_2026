package rules

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/datasources/dailymed"
)

// =============================================================================
// RULE TRANSLATOR TESTS
// =============================================================================

func TestNewRuleTranslator(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	if translator == nil {
		t.Fatal("Expected translator to be created")
	}
}

func TestNewRuleTranslator_WithDependencies(t *testing.T) {
	mockRegistry := &MockFingerprintRegistry{}
	mockQueue := &MockUntranslatableQueue{}

	translator := NewRuleTranslator(mockRegistry, mockQueue)

	if translator == nil {
		t.Fatal("Expected translator to be created")
	}
}

// =============================================================================
// TRANSLATION TESTS
// =============================================================================

func TestTranslateExtractedTables_GFRDosingTable(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-001",
			TableType: dailymed.TableTypeGFRDosing,
			Headers:   []string{"CrCl (mL/min)", "Dose"},
			Rows: [][]string{
				{"≥60", "500 mg twice daily"},
				{"30-59", "500 mg once daily"},
				{"<30", "Contraindicated"},
			},
		},
	}

	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
		DocumentID:       "metformin-001",
		SectionCode:      "34068-7",
		ExtractionMethod: "TABLE_PARSE",
	}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Stats.TablesProcessed != 1 {
		t.Errorf("Expected 1 table processed, got %d", result.Stats.TablesProcessed)
	}

	if result.Stats.RulesGenerated == 0 {
		t.Error("Expected at least one rule to be generated")
	}

	// Verify all rules have correct domain
	for _, rule := range result.Rules {
		if rule.Domain != "KB-1" {
			t.Errorf("Expected domain KB-1, got %s", rule.Domain)
		}
	}
}

func TestTranslateExtractedTables_HepaticDosingTable(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-002",
			TableType: dailymed.TableTypeHepaticDosing,
			Headers:   []string{"Child-Pugh Class", "Dose Adjustment"},
			Rows: [][]string{
				{"A", "No adjustment"},
				{"B", "Reduce by 50%"},
				{"C", "Avoid use"},
			},
		},
	}

	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Stats.TablesTranslated != 1 {
		t.Errorf("Expected 1 table translated, got %d", result.Stats.TablesTranslated)
	}
}

func TestTranslateExtractedTables_DDITable(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-003",
			TableType: dailymed.TableTypeDDI,
			Headers:   []string{"Interacting Drug", "Recommendation"},
			Rows: [][]string{
				{"Ketoconazole", "Reduce dose by 50%"},
				{"Rifampin", "Avoid combination"},
			},
		},
	}

	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-5")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify rules have correct type
	for _, rule := range result.Rules {
		if rule.RuleType != RuleTypeInteraction {
			t.Errorf("Expected RuleType INTERACTION for DDI table, got %s", rule.RuleType)
		}
	}
}

// =============================================================================
// TABLE SKIPPING TESTS
// =============================================================================

func TestTranslateExtractedTables_SkipsUnknownTables(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-unknown",
			TableType: dailymed.TableTypeUnknown,
			Headers:   []string{"Some", "Random", "Headers"},
			Rows: [][]string{
				{"Value1", "Value2", "Value3"},
			},
		},
	}

	provenance := Provenance{SourceDocumentID: uuid.New()}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Stats.TablesSkipped != 1 {
		t.Errorf("Expected 1 table skipped, got %d", result.Stats.TablesSkipped)
	}

	if len(result.Rules) != 0 {
		t.Errorf("Expected 0 rules for unknown table, got %d", len(result.Rules))
	}
}

func TestTranslateExtractedTables_SkipsAdverseEventsTables(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-ae",
			TableType: dailymed.TableTypeAdverseEvents,
			Headers:   []string{"Adverse Event", "Incidence"},
			Rows: [][]string{
				{"Nausea", "10%"},
				{"Headache", "5%"},
			},
		},
	}

	provenance := Provenance{SourceDocumentID: uuid.New()}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Stats.TablesSkipped != 1 {
		t.Errorf("Expected 1 table skipped, got %d", result.Stats.TablesSkipped)
	}
}

func TestTranslateExtractedTables_SkipsPKTables(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-pk",
			TableType: dailymed.TableTypePK,
			Headers:   []string{"Parameter", "Value"},
			Rows: [][]string{
				{"Cmax", "100 ng/mL"},
				{"AUC", "500 ng·h/mL"},
			},
		},
	}

	provenance := Provenance{SourceDocumentID: uuid.New()}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Stats.TablesSkipped != 1 {
		t.Errorf("Expected 1 table skipped, got %d", result.Stats.TablesSkipped)
	}
}

// =============================================================================
// UNTRANSLATABLE HANDLING TESTS
// =============================================================================

func TestTranslateExtractedTables_HandlesUntranslatable(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	// Table with no recognizable structure
	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-untranslatable",
			TableType: dailymed.TableTypeDosing,
			Headers:   []string{"Notes", "See Also"},
			Rows: [][]string{
				{"Refer to prescribing information", "Section 2.1"},
			},
		},
	}

	provenance := Provenance{SourceDocumentID: uuid.New()}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Stats.TablesUntranslatable != 1 {
		t.Errorf("Expected 1 untranslatable table, got %d", result.Stats.TablesUntranslatable)
	}

	if len(result.UntranslatableTables) != 1 {
		t.Errorf("Expected 1 untranslatable entry, got %d", len(result.UntranslatableTables))
	}
}

// =============================================================================
// FINGERPRINT DEDUPLICATION TESTS
// =============================================================================

func TestTranslateExtractedTables_DeduplicatesWithFingerprint(t *testing.T) {
	mockRegistry := &MockFingerprintRegistry{
		existingHashes: map[string]bool{},
	}
	translator := NewRuleTranslator(mockRegistry, nil)

	// Same table twice - should deduplicate
	table := &dailymed.ExtractedTable{
		TableID:   "table-dup",
		TableType: dailymed.TableTypeGFRDosing,
		Headers:   []string{"CrCl", "Dose"},
		Rows: [][]string{
			{"<30", "Contraindicated"},
		},
	}

	tables := []*dailymed.ExtractedTable{table, table}

	provenance := Provenance{SourceDocumentID: uuid.New()}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// First rule should be registered, second should be deduplicated
	if result.Stats.DuplicatesSkipped == 0 {
		t.Error("Expected at least one duplicate to be skipped")
	}
}

// =============================================================================
// TABLE TYPE MAPPING TESTS
// =============================================================================

func TestMapTableTypeToRuleType(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	tests := []struct {
		tableType dailymed.TableType
		expected  RuleType
	}{
		{dailymed.TableTypeGFRDosing, RuleTypeDosing},
		{dailymed.TableTypeHepaticDosing, RuleTypeDosing},
		{dailymed.TableTypeDosing, RuleTypeDosing},
		{dailymed.TableTypeDDI, RuleTypeInteraction},
		{dailymed.TableTypeContraindications, RuleTypeContraindication},
	}

	for _, tt := range tests {
		t.Run(string(tt.tableType), func(t *testing.T) {
			result := translator.mapTableTypeToRuleType(tt.tableType)
			if result != tt.expected {
				t.Errorf("Expected %s for table type %s, got %s", tt.expected, tt.tableType, result)
			}
		})
	}
}

// =============================================================================
// BATCH PROCESSING TESTS
// =============================================================================

func TestTranslateBatch(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	requests := []BatchTranslationRequest{
		{
			Tables: []*dailymed.ExtractedTable{
				{
					TableID:   "batch-1",
					TableType: dailymed.TableTypeGFRDosing,
					Headers:   []string{"CrCl", "Dose"},
					Rows:      [][]string{{"<30", "Contraindicated"}},
				},
			},
			Provenance: Provenance{SourceDocumentID: uuid.New()},
			Domain:     "KB-1",
		},
		{
			Tables: []*dailymed.ExtractedTable{
				{
					TableID:   "batch-2",
					TableType: dailymed.TableTypeDDI,
					Headers:   []string{"Drug", "Recommendation"},
					Rows:      [][]string{{"Warfarin", "Monitor INR"}},
				},
			},
			Provenance: Provenance{SourceDocumentID: uuid.New()},
			Domain:     "KB-5",
		},
	}

	ctx := context.Background()
	results, err := translator.TranslateBatch(ctx, requests)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// =============================================================================
// STATISTICS AGGREGATION TESTS
// =============================================================================

func TestAggregateStats(t *testing.T) {
	results := []*TranslationResult{
		{
			Stats: TranslationStats{
				TablesProcessed:   5,
				TablesTranslated:  4,
				RulesGenerated:    10,
				DuplicatesSkipped: 2,
				AverageConfidence: 0.9,
				ProcessingTimeMs:  100,
			},
		},
		{
			Stats: TranslationStats{
				TablesProcessed:   3,
				TablesTranslated:  2,
				RulesGenerated:    5,
				DuplicatesSkipped: 1,
				AverageConfidence: 0.85,
				ProcessingTimeMs:  50,
			},
		},
	}

	agg := AggregateStats(results)

	if agg.TablesProcessed != 8 {
		t.Errorf("Expected 8 tables processed, got %d", agg.TablesProcessed)
	}

	if agg.TablesTranslated != 6 {
		t.Errorf("Expected 6 tables translated, got %d", agg.TablesTranslated)
	}

	if agg.RulesGenerated != 15 {
		t.Errorf("Expected 15 rules generated, got %d", agg.RulesGenerated)
	}

	if agg.DuplicatesSkipped != 3 {
		t.Errorf("Expected 3 duplicates skipped, got %d", agg.DuplicatesSkipped)
	}

	if agg.ProcessingTimeMs != 150 {
		t.Errorf("Expected 150ms total processing time, got %d", agg.ProcessingTimeMs)
	}
}

// =============================================================================
// PROVENANCE TESTS
// =============================================================================

func TestTranslateExtractedTables_PreservesProvenance(t *testing.T) {
	translator := NewRuleTranslatorSimple()

	docID := uuid.New()
	sectionID := uuid.New()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "table-prov",
			TableType: dailymed.TableTypeGFRDosing,
			Headers:   []string{"CrCl", "Dose"},
			Rows:      [][]string{{"<30", "Contraindicated"}},
		},
	}

	provenance := Provenance{
		SourceDocumentID: docID,
		SourceSectionID:  &sectionID,
		SourceType:       "FDA_SPL",
		DocumentID:       "test-doc",
		SectionCode:      "34068-7",
		ExtractionMethod: "TABLE_PARSE",
	}

	ctx := context.Background()
	result, err := translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Rules) == 0 {
		t.Fatal("Expected at least one rule")
	}

	rule := result.Rules[0]

	if rule.Provenance.SourceDocumentID != docID {
		t.Error("Expected provenance to preserve SourceDocumentID")
	}

	if rule.Provenance.SourceSectionID == nil || *rule.Provenance.SourceSectionID != sectionID {
		t.Error("Expected provenance to preserve SourceSectionID")
	}

	if rule.Provenance.SourceType != "FDA_SPL" {
		t.Errorf("Expected SourceType FDA_SPL, got %s", rule.Provenance.SourceType)
	}

	if rule.Provenance.TableID != "table-prov" {
		t.Errorf("Expected TableID table-prov, got %s", rule.Provenance.TableID)
	}
}

// =============================================================================
// MOCK IMPLEMENTATIONS
// =============================================================================

type MockFingerprintRegistry struct {
	existingHashes map[string]bool
	registered     []*DraftRule
}

func (m *MockFingerprintRegistry) Exists(ctx context.Context, hash string) (bool, error) {
	return m.existingHashes[hash], nil
}

func (m *MockFingerprintRegistry) Register(ctx context.Context, rule *DraftRule) error {
	m.existingHashes[rule.SemanticFingerprint.Hash] = true
	m.registered = append(m.registered, rule)
	return nil
}

func (m *MockFingerprintRegistry) GetRuleByFingerprint(ctx context.Context, hash string) (*uuid.UUID, error) {
	return nil, nil
}

type MockUntranslatableQueue struct {
	entries []*UntranslatableEntry
}

func (m *MockUntranslatableQueue) Enqueue(ctx context.Context, entry *UntranslatableEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkTranslateExtractedTables(b *testing.B) {
	translator := NewRuleTranslatorSimple()

	tables := []*dailymed.ExtractedTable{
		{
			TableID:   "bench-table",
			TableType: dailymed.TableTypeGFRDosing,
			Headers:   []string{"CrCl (mL/min)", "Starting Dose", "Maximum Dose"},
			Rows: [][]string{
				{"≥60", "500 mg twice daily", "2550 mg/day"},
				{"30-59", "500 mg once daily", "1000 mg/day"},
				{"15-29", "500 mg every other day", "500 mg/day"},
				{"<15", "Contraindicated", "--"},
			},
		},
	}

	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
		ExtractedAt:      time.Now(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		translator.TranslateExtractedTables(ctx, tables, provenance, "KB-1")
	}
}
