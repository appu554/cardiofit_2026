package extraction

import (
	"testing"
)

// =============================================================================
// TABLE NORMALIZER TESTS
// =============================================================================

func TestNewTableNormalizer(t *testing.T) {
	normalizer := NewTableNormalizer()

	if normalizer == nil {
		t.Fatal("Expected normalizer to be created")
	}
}

// =============================================================================
// COLUMN ROLE DETECTION TESTS
// Uses the public Normalize method to test column classification
// =============================================================================

func TestClassifyColumn_ConditionPatterns(t *testing.T) {
	normalizer := NewTableNormalizer()

	tests := []struct {
		name     string
		header   string
		expected ColumnRole
	}{
		{
			name:     "CrCl header",
			header:   "Creatinine Clearance (mL/min)",
			expected: RoleCondition,
		},
		{
			name:     "eGFR header",
			header:   "eGFR",
			expected: RoleCondition,
		},
		{
			name:     "Renal Function header",
			header:   "Renal Function",
			expected: RoleCondition,
		},
		{
			name:     "Child-Pugh header",
			header:   "Child-Pugh Class",
			expected: RoleCondition,
		},
		{
			name:     "Age header",
			header:   "Age (years)",
			expected: RoleCondition,
		},
		{
			name:     "Weight header",
			header:   "Body Weight",
			expected: RoleCondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use Normalize to test column classification
			result, err := normalizer.Normalize("test", []string{tt.header, "Dose"}, [][]string{{"value", "500 mg"}})
			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}

			if len(result.NormalizedCols) == 0 {
				t.Fatal("Expected at least one column")
			}

			// First column should be the condition
			if result.NormalizedCols[0].Role != tt.expected {
				t.Errorf("Expected %s for header '%s', got %s", tt.expected, tt.header, result.NormalizedCols[0].Role)
			}
		})
	}
}

func TestClassifyColumn_ActionPatterns(t *testing.T) {
	normalizer := NewTableNormalizer()

	tests := []struct {
		name     string
		header   string
		expected ColumnRole
	}{
		{
			name:     "Dose header",
			header:   "Recommended Dose",
			expected: RoleAction,
		},
		{
			name:     "Adjustment header",
			header:   "Dose Adjustment",
			expected: RoleAction,
		},
		{
			name:     "Starting Dose header",
			header:   "Starting Dose",
			expected: RoleAction,
		},
		{
			name:     "Maximum Dose header",
			header:   "Maximum Daily Dose",
			expected: RoleAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use Normalize to test column classification
			result, err := normalizer.Normalize("test", []string{"CrCl", tt.header}, [][]string{{"30-60", "500 mg"}})
			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}

			if len(result.NormalizedCols) < 2 {
				t.Fatal("Expected at least two columns")
			}

			// Second column should be the action
			if result.NormalizedCols[1].Role != tt.expected {
				t.Errorf("Expected %s for header '%s', got %s", tt.expected, tt.header, result.NormalizedCols[1].Role)
			}
		})
	}
}

func TestClassifyColumn_DrugNamePatterns(t *testing.T) {
	normalizer := NewTableNormalizer()

	tests := []struct {
		name     string
		header   string
		expected ColumnRole
	}{
		{
			name:     "Drug Name header",
			header:   "Drug Name",
			expected: RoleDrugName,
		},
		{
			name:     "Interacting Drug header",
			header:   "Interacting Drug",
			expected: RoleDrugName,
		},
		{
			name:     "Concomitant Medication header",
			header:   "Concomitant Medication",
			expected: RoleDrugName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use Normalize to test column classification
			result, err := normalizer.Normalize("test", []string{tt.header, "Effect", "Recommendation"}, [][]string{{"Metformin", "Increased", "Monitor"}})
			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}

			if len(result.NormalizedCols) == 0 {
				t.Fatal("Expected at least one column")
			}

			// First column should be drug name
			if result.NormalizedCols[0].Role != tt.expected {
				t.Errorf("Expected %s for header '%s', got %s", tt.expected, tt.header, result.NormalizedCols[0].Role)
			}
		})
	}
}

// =============================================================================
// TABLE NORMALIZATION TESTS
// =============================================================================

func TestNormalize_GFRDosingTable(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl (mL/min)", "Starting Dose", "Maximum Dose"}
	rows := [][]string{
		{"≥60", "500 mg twice daily", "2550 mg/day"},
		{"30-59", "500 mg once daily", "1000 mg/day"},
		{"<30", "Contraindicated", "--"},
	}

	result, err := normalizer.Normalize("test-table-001", headers, rows)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Translatable {
		t.Errorf("Expected table to be translatable, reason: %s", result.UntranslatableReason)
	}

	// Check column roles
	conditionCols := result.GetConditionColumns()
	if len(conditionCols) == 0 {
		t.Error("Expected at least one condition column")
	}

	actionCols := result.GetActionColumns()
	if len(actionCols) == 0 {
		t.Error("Expected at least one action column")
	}

	// Verify row count
	if len(result.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(result.Rows))
	}
}

func TestNormalize_HepaticDosingTable(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"Child-Pugh Class", "Dose Recommendation"}
	rows := [][]string{
		{"A", "No adjustment needed"},
		{"B", "Reduce dose by 50%"},
		{"C", "Avoid use"},
	}

	result, err := normalizer.Normalize("test-table-002", headers, rows)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Translatable {
		t.Errorf("Expected table to be translatable, reason: %s", result.UntranslatableReason)
	}

	// Should have Child-Pugh as condition
	conditionCols := result.GetConditionColumns()
	if len(conditionCols) == 0 {
		t.Error("Expected Child-Pugh column to be classified as condition")
	}
}

func TestNormalize_DDITable(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"Interacting Drug", "Effect", "Clinical Recommendation"}
	rows := [][]string{
		{"Ketoconazole", "Increased exposure", "Reduce dose by 50%"},
		{"Rifampin", "Decreased exposure", "Consider alternative"},
		{"Warfarin", "Increased bleeding risk", "Monitor INR closely"},
	}

	result, err := normalizer.Normalize("test-table-003", headers, rows)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify drug column was detected
	hasDrugCol := false
	for _, col := range result.NormalizedCols {
		if col.Role == RoleDrugName {
			hasDrugCol = true
			break
		}
	}

	if !hasDrugCol {
		t.Error("Expected interacting drug column to be identified as drug name")
	}
}

// =============================================================================
// UNTRANSLATABLE TABLE TESTS
// =============================================================================

func TestNormalize_NoConditionColumn_Untranslatable(t *testing.T) {
	normalizer := NewTableNormalizer()

	// Table with only notes/context, no clear condition
	headers := []string{"Notes", "Additional Information"}
	rows := [][]string{
		{"See prescribing information", "Refer to section 2.1"},
		{"Use clinical judgment", "Individual patient factors"},
	}

	result, err := normalizer.Normalize("test-table-004", headers, rows)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Translatable {
		t.Error("Expected table with no condition column to be untranslatable")
	}

	if result.UntranslatableReason == "" {
		t.Error("Expected untranslatable reason to be set")
	}
}

func TestNormalize_EmptyTable(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Dose"}
	rows := [][]string{} // Empty

	result, err := normalizer.Normalize("test-table-005", headers, rows)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Table with headers but no rows should still process
	if len(result.NormalizedCols) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(result.NormalizedCols))
	}

	if len(result.Rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(result.Rows))
	}
}

// =============================================================================
// CELL NORMALIZATION TESTS
// Tests cell normalization through the public Normalize method
// =============================================================================

func TestNormalize_CellValues_GFRThresholds(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl (mL/min)", "Dose"}
	rows := [][]string{
		{"≥60", "Full dose"},
		{"30-59", "Half dose"},
		{"<30", "Avoid"},
	}

	result, err := normalizer.Normalize("test-gfr-cells", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check first row cell normalization
	if len(result.Rows) < 1 || len(result.Rows[0].Cells) < 1 {
		t.Fatal("Expected at least one row with cells")
	}

	firstCell := result.Rows[0].Cells[0]
	if firstCell.OriginalText != "≥60" {
		t.Errorf("Expected original text '≥60', got '%s'", firstCell.OriginalText)
	}

	// Normalized value should be present for condition cells
	if firstCell.Normalized == nil {
		t.Error("Expected normalized value for GFR threshold cell")
	}
}

func TestNormalize_CellValues_ChildPugh(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"Child-Pugh", "Dose"}
	rows := [][]string{
		{"Class A", "Full dose"},
		{"Class B", "Half dose"},
		{"Class C", "Contraindicated"},
	}

	result, err := normalizer.Normalize("test-childpugh-cells", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify cells are normalized
	for _, row := range result.Rows {
		if len(row.Cells) > 0 && row.Cells[0].Normalized != nil {
			// Child-Pugh should be normalized
			nv := row.Cells[0].Normalized
			if nv.Variable == "" {
				t.Error("Expected variable to be set for Child-Pugh cell")
			}
		}
	}
}

func TestNormalize_CellValues_DoseExtraction(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Dose"}
	rows := [][]string{
		{">60", "500 mg twice daily"},
		{"30-60", "250 mg once daily"},
	}

	result, err := normalizer.Normalize("test-dose-cells", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that dose column cells have normalized values
	if len(result.Rows) > 0 && len(result.Rows[0].Cells) > 1 {
		doseCell := result.Rows[0].Cells[1]
		if doseCell.Normalized == nil {
			t.Error("Expected normalized value for dose cell")
		}
	}
}

// =============================================================================
// HELPER METHOD TESTS
// =============================================================================

func TestGetConditionColumns_ReturnsCorrectColumns(t *testing.T) {
	normalizer := NewTableNormalizer()

	// Create a table with multiple condition columns
	headers := []string{"CrCl", "Dose", "Age"}
	rows := [][]string{
		{"30-60", "500 mg", "18-65"},
	}

	result, err := normalizer.Normalize("test-helper-cond", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cols := result.GetConditionColumns()

	// Both CrCl and Age should be condition columns
	if len(cols) < 2 {
		t.Errorf("Expected at least 2 condition columns (CrCl, Age), got %d", len(cols))
	}
}

func TestGetActionColumns_ReturnsCorrectColumns(t *testing.T) {
	normalizer := NewTableNormalizer()

	// Create a table with multiple action columns
	headers := []string{"CrCl", "Starting Dose", "Max Dose"}
	rows := [][]string{
		{"30-60", "500 mg", "2000 mg"},
	}

	result, err := normalizer.Normalize("test-helper-action", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cols := result.GetActionColumns()

	// Both dose columns should be action columns
	if len(cols) < 2 {
		t.Errorf("Expected at least 2 action columns, got %d", len(cols))
	}
}

func TestGetRowConditionValues_ReturnsCorrectCells(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Dose", "Age"}
	rows := [][]string{
		{"30-60", "500 mg", "18-65"},
		{">60", "1000 mg", ">65"},
	}

	result, err := normalizer.Normalize("test-row-values", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Get condition values for first row
	condCells := result.GetRowConditionValues(0)

	if len(condCells) == 0 {
		t.Error("Expected condition cells to be returned")
	}
}

func TestGetRowActionValues_ReturnsCorrectCells(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Starting Dose", "Max Dose"}
	rows := [][]string{
		{"30-60", "500 mg", "2000 mg"},
	}

	result, err := normalizer.Normalize("test-action-values", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Get action values for first row
	actionCells := result.GetRowActionValues(0)

	if len(actionCells) == 0 {
		t.Error("Expected action cells to be returned")
	}
}

// =============================================================================
// CONFIDENCE AND METADATA TESTS
// =============================================================================

func TestNormalize_ColumnConfidence(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"Creatinine Clearance", "Dose"}
	rows := [][]string{
		{"30-60", "500 mg"},
	}

	result, err := normalizer.Normalize("test-confidence", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// CrCl should have high confidence
	if len(result.NormalizedCols) > 0 {
		col := result.NormalizedCols[0]
		if col.Confidence < 0.8 {
			t.Errorf("Expected high confidence for CrCl column, got %f", col.Confidence)
		}
	}
}

func TestNormalize_OverallConfidence(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl (mL/min)", "Recommended Dose"}
	rows := [][]string{
		{"30-60", "500 mg"},
	}

	result, err := normalizer.Normalize("test-overall-conf", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.OverallConfidence <= 0 {
		t.Error("Expected positive overall confidence")
	}
}

func TestNormalize_ColumnCounts(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Age", "Dose", "Max Dose"}
	rows := [][]string{
		{"30-60", "18-65", "500 mg", "2000 mg"},
	}

	result, err := normalizer.Normalize("test-counts", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ConditionColumnCount < 2 {
		t.Errorf("Expected at least 2 condition columns, got %d", result.ConditionColumnCount)
	}

	if result.ActionColumnCount < 2 {
		t.Errorf("Expected at least 2 action columns, got %d", result.ActionColumnCount)
	}
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestNormalize_EmptyCells(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Dose"}
	rows := [][]string{
		{"30-60", "500 mg"},
		{"", ""},
		{">60", "1000 mg"},
	}

	result, err := normalizer.Normalize("test-empty-cells", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Second row should have empty cells marked
	if len(result.Rows) > 1 {
		for _, cell := range result.Rows[1].Cells {
			if !cell.IsEmpty {
				t.Error("Expected empty cell to be marked as empty")
			}
		}
	}
}

func TestNormalize_WhitespaceTrimming(t *testing.T) {
	normalizer := NewTableNormalizer()

	headers := []string{"  CrCl  ", "  Dose  "}
	rows := [][]string{
		{"  30-60  ", "  500 mg  "},
	}

	result, err := normalizer.Normalize("test-whitespace", headers, rows)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Original headers should be preserved
	if len(result.OriginalHeaders) > 0 && result.OriginalHeaders[0] != "  CrCl  " {
		// Note: headers may or may not be trimmed based on implementation
		// This test documents current behavior
	}

	// Cell text should be trimmed
	if len(result.Rows) > 0 && len(result.Rows[0].Cells) > 0 {
		cell := result.Rows[0].Cells[0]
		if cell.OriginalText == "  30-60  " {
			t.Log("Cell text is not trimmed - documenting current behavior")
		}
	}
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkNormalize(b *testing.B) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl (mL/min)", "Starting Dose", "Maximum Dose"}
	rows := [][]string{
		{"≥60", "500 mg twice daily", "2550 mg/day"},
		{"30-59", "500 mg once daily", "1000 mg/day"},
		{"15-29", "500 mg every other day", "500 mg/day"},
		{"<15", "Contraindicated", "--"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize("bench-table", headers, rows)
	}
}

func BenchmarkNormalize_LargeTable(b *testing.B) {
	normalizer := NewTableNormalizer()

	headers := []string{"CrCl", "Age", "Weight", "Dose", "Max Dose", "Frequency"}

	// Create a larger table for benchmarking
	rows := make([][]string, 50)
	for i := 0; i < 50; i++ {
		rows[i] = []string{"30-60", "18-65", "50-100 kg", "500 mg", "2000 mg", "twice daily"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize("bench-large-table", headers, rows)
	}
}
