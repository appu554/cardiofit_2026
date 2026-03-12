package extraction

import (
	"testing"
)

// =============================================================================
// CONDITION ACTION GENERATOR TESTS
// =============================================================================

func TestNewConditionActionGenerator(t *testing.T) {
	generator := NewConditionActionGenerator()

	if generator == nil {
		t.Fatal("Expected generator to be created")
	}
}

// =============================================================================
// RULE GENERATION FROM TABLES
// =============================================================================

func TestGenerateFromTable_GFRDosingTable(t *testing.T) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:              "test-gfr-001",
		OriginalHeaders: []string{"CrCl (mL/min)", "Dose Recommendation"},
		NormalizedCols: []NormalizedColumn{
			{Index: 0, OriginalHeader: "CrCl (mL/min)", Role: RoleCondition, Confidence: 0.9},
			{Index: 1, OriginalHeader: "Dose Recommendation", Role: RoleAction, Confidence: 0.85},
		},
		Rows: []NormalizedRow{
			{
				Index: 0,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: ">=60", Normalized: &NormalizedValue{NumericValue: ptrFloat(60), Unit: "ml/min"}},
					{ColumnIndex: 1, OriginalText: "500 mg twice daily", Normalized: &NormalizedValue{StringValue: ptrString("500 mg twice daily")}},
				},
			},
			{
				Index: 1,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "30-59", Normalized: &NormalizedValue{MinValue: ptrFloat(30), MaxValue: ptrFloat(59), Unit: "ml/min"}},
					{ColumnIndex: 1, OriginalText: "500 mg once daily", Normalized: &NormalizedValue{StringValue: ptrString("500 mg once daily")}},
				},
			},
			{
				Index: 2,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "<30", Normalized: &NormalizedValue{NumericValue: ptrFloat(30), Unit: "ml/min"}},
					{ColumnIndex: 1, OriginalText: "Contraindicated", Normalized: &NormalizedValue{StringValue: ptrString("contraindicated")}},
				},
			},
		},
		Translatable:         true,
		ConditionColumnCount: 1,
		ActionColumnCount:    1,
		OverallConfidence:    0.875,
	}

	result, err := generator.GenerateFromTable(table)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.TotalRows != 3 {
		t.Errorf("Expected 3 total rows, got %d", result.TotalRows)
	}

	if len(result.Rules) == 0 {
		t.Fatal("Expected at least one rule to be generated")
	}

	// Check last row produces contraindication rule
	foundContraindication := false
	for _, rule := range result.Rules {
		if rule.Action.Effect == "CONTRAINDICATED" {
			foundContraindication = true
			break
		}
	}

	if !foundContraindication {
		t.Error("Expected to find CONTRAINDICATED rule for CrCl < 30")
	}
}

func TestGenerateFromTable_HepaticDosingTable(t *testing.T) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:              "test-hepatic-001",
		OriginalHeaders: []string{"Child-Pugh Class", "Dose Adjustment"},
		NormalizedCols: []NormalizedColumn{
			{Index: 0, OriginalHeader: "Child-Pugh Class", Role: RoleCondition, Confidence: 0.9},
			{Index: 1, OriginalHeader: "Dose Adjustment", Role: RoleAction, Confidence: 0.85},
		},
		Rows: []NormalizedRow{
			{
				Index: 0,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "A", Normalized: &NormalizedValue{StringValue: ptrString("A")}},
					{ColumnIndex: 1, OriginalText: "No adjustment", Normalized: &NormalizedValue{StringValue: ptrString("no adjustment")}},
				},
			},
			{
				Index: 1,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "B", Normalized: &NormalizedValue{StringValue: ptrString("B")}},
					{ColumnIndex: 1, OriginalText: "Reduce by 50%", Normalized: &NormalizedValue{StringValue: ptrString("reduce by 50%")}},
				},
			},
			{
				Index: 2,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "C", Normalized: &NormalizedValue{StringValue: ptrString("C")}},
					{ColumnIndex: 1, OriginalText: "Avoid use", Normalized: &NormalizedValue{StringValue: ptrString("avoid use")}},
				},
			},
		},
		Translatable:         true,
		ConditionColumnCount: 1,
		ActionColumnCount:    1,
		OverallConfidence:    0.875,
	}

	result, err := generator.GenerateFromTable(table)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Rules) != 3 {
		t.Errorf("Expected 3 rules (one per Child-Pugh class), got %d", len(result.Rules))
	}
}

func TestGenerateFromTable_UntranslatableTable(t *testing.T) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:                   "test-untrans-001",
		OriginalHeaders:      []string{"Column A", "Column B"},
		Translatable:         false,
		UntranslatableReason: "No condition or action columns identified",
	}

	_, err := generator.GenerateFromTable(table)

	if err == nil {
		t.Error("Expected error for untranslatable table")
	}
}

func TestGenerateFromTable_NoConditionColumns(t *testing.T) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:              "test-nocond-001",
		OriginalHeaders: []string{"Drug Name", "Notes"},
		NormalizedCols: []NormalizedColumn{
			{Index: 0, OriginalHeader: "Drug Name", Role: RoleDrugName, Confidence: 0.9},
			{Index: 1, OriginalHeader: "Notes", Role: RoleMetadata, Confidence: 0.5},
		},
		Rows: []NormalizedRow{
			{
				Index: 0,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "Metformin"},
					{ColumnIndex: 1, OriginalText: "Take with food"},
				},
			},
		},
		Translatable:         true,
		ConditionColumnCount: 0,
		ActionColumnCount:    0,
		OverallConfidence:    0.7,
	}

	_, err := generator.GenerateFromTable(table)

	if err == nil {
		t.Error("Expected error for table with no condition columns")
	}
}

// =============================================================================
// SKIPPED ROW TESTS
// =============================================================================

func TestGenerateFromTable_SkipsEmptyRows(t *testing.T) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:              "test-skip-001",
		OriginalHeaders: []string{"CrCl", "Dose"},
		NormalizedCols: []NormalizedColumn{
			{Index: 0, OriginalHeader: "CrCl", Role: RoleCondition},
			{Index: 1, OriginalHeader: "Dose", Role: RoleAction},
		},
		Rows: []NormalizedRow{
			{
				Index: 0,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: ">= 60", Normalized: &NormalizedValue{NumericValue: ptrFloat(60)}},
					{ColumnIndex: 1, OriginalText: "500 mg", Normalized: &NormalizedValue{StringValue: ptrString("500 mg")}},
				},
			},
			{
				Index: 1,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "", IsEmpty: true}, // Empty condition
					{ColumnIndex: 1, OriginalText: "", IsEmpty: true}, // Empty action
				},
			},
			{
				Index: 2,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "< 30", Normalized: &NormalizedValue{NumericValue: ptrFloat(30)}},
					{ColumnIndex: 1, OriginalText: "Contraindicated", Normalized: &NormalizedValue{StringValue: ptrString("contraindicated")}},
				},
			},
		},
		Translatable:         true,
		ConditionColumnCount: 1,
		ActionColumnCount:    1,
		OverallConfidence:    0.85,
	}

	result, err := generator.GenerateFromTable(table)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.TotalRows != 3 {
		t.Errorf("Expected 3 total rows, got %d", result.TotalRows)
	}

	if result.SkippedCount != 1 {
		t.Errorf("Expected 1 skipped row, got %d", result.SkippedCount)
	}

	if len(result.Rules) != 2 {
		t.Errorf("Expected 2 rules (skipping empty row), got %d", len(result.Rules))
	}
}

// =============================================================================
// CONFIDENCE CALCULATION TESTS
// =============================================================================

func TestGeneratedRule_Confidence(t *testing.T) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:              "test-conf-001",
		OriginalHeaders: []string{"CrCl", "Dose"},
		NormalizedCols: []NormalizedColumn{
			{Index: 0, OriginalHeader: "CrCl", Role: RoleCondition, Confidence: 0.95},
			{Index: 1, OriginalHeader: "Dose", Role: RoleAction, Confidence: 0.90},
		},
		Rows: []NormalizedRow{
			{
				Index: 0,
				Cells: []NormalizedCell{
					{ColumnIndex: 0, OriginalText: "< 30", Normalized: &NormalizedValue{NumericValue: ptrFloat(30), Confidence: 0.85}},
					{ColumnIndex: 1, OriginalText: "Contraindicated", Normalized: &NormalizedValue{StringValue: ptrString("contraindicated"), Confidence: 0.90}},
				},
			},
		},
		Translatable:         true,
		ConditionColumnCount: 1,
		ActionColumnCount:    1,
		OverallConfidence:    0.925,
	}

	result, err := generator.GenerateFromTable(table)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Rules) == 0 {
		t.Fatal("Expected at least one rule")
	}

	// Confidence should be based on column scores
	rule := result.Rules[0]
	if rule.Confidence < 0.5 || rule.Confidence > 1.0 {
		t.Errorf("Expected confidence between 0.5 and 1.0, got %f", rule.Confidence)
	}
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkGenerateFromTable(b *testing.B) {
	generator := NewConditionActionGenerator()

	table := &NormalizedTable{
		ID:              "bench-001",
		OriginalHeaders: []string{"CrCl", "Starting Dose", "Max Dose"},
		NormalizedCols: []NormalizedColumn{
			{Index: 0, OriginalHeader: "CrCl", Role: RoleCondition, Confidence: 0.9},
			{Index: 1, OriginalHeader: "Starting Dose", Role: RoleAction, Confidence: 0.85},
			{Index: 2, OriginalHeader: "Max Dose", Role: RoleAction, Confidence: 0.85},
		},
		Rows: []NormalizedRow{
			{Index: 0, Cells: []NormalizedCell{
				{ColumnIndex: 0, OriginalText: ">=60", Normalized: &NormalizedValue{NumericValue: ptrFloat(60)}},
				{ColumnIndex: 1, OriginalText: "500 mg", Normalized: &NormalizedValue{StringValue: ptrString("500 mg")}},
				{ColumnIndex: 2, OriginalText: "2000 mg/day", Normalized: &NormalizedValue{StringValue: ptrString("2000 mg/day")}},
			}},
			{Index: 1, Cells: []NormalizedCell{
				{ColumnIndex: 0, OriginalText: "30-59", Normalized: &NormalizedValue{MinValue: ptrFloat(30), MaxValue: ptrFloat(59)}},
				{ColumnIndex: 1, OriginalText: "250 mg", Normalized: &NormalizedValue{StringValue: ptrString("250 mg")}},
				{ColumnIndex: 2, OriginalText: "1000 mg/day", Normalized: &NormalizedValue{StringValue: ptrString("1000 mg/day")}},
			}},
			{Index: 2, Cells: []NormalizedCell{
				{ColumnIndex: 0, OriginalText: "<30", Normalized: &NormalizedValue{NumericValue: ptrFloat(30)}},
				{ColumnIndex: 1, OriginalText: "Contraindicated", Normalized: &NormalizedValue{StringValue: ptrString("contraindicated")}},
				{ColumnIndex: 2, OriginalText: "--", Normalized: &NormalizedValue{StringValue: ptrString("--")}},
			}},
		},
		Translatable:         true,
		ConditionColumnCount: 1,
		ActionColumnCount:    2,
		OverallConfidence:    0.87,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateFromTable(table)
	}
}

// Helper functions
func ptrFloat(f float64) *float64 {
	return &f
}

func ptrString(s string) *string {
	return &s
}
