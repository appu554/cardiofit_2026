// Package extraction provides table normalization for clinical rule extraction.
//
// Phase 3b.5.3: Table Normalizer
// Key Principle: Detect column roles (condition vs action) and standardize table structure
// before rule extraction. This enables algorithmic IF/THEN rule generation.
//
// Column Roles:
// - CONDITION: The IF part (GFR, Child-Pugh, age, weight)
// - ACTION: The THEN part (dose, recommendation, adjustment)
// - DRUG_NAME: Drug identifier column
// - UNKNOWN: Columns needing manual classification
package extraction

import (
	"strings"
)

// =============================================================================
// TABLE NORMALIZER
// =============================================================================

// TableNormalizer detects column roles and standardizes table structure
type TableNormalizer struct {
	unitNormalizer    *UnitNormalizer
	conditionPatterns []columnPattern
	actionPatterns    []columnPattern
	drugPatterns      []columnPattern
}

// columnPattern contains keywords for detecting column roles
type columnPattern struct {
	keywords []string
	role     ColumnRole
	weight   float64 // Higher weight = more confident match
}

// ColumnRole identifies the semantic role of a table column
type ColumnRole string

const (
	RoleCondition ColumnRole = "CONDITION" // IF part: GFR, Child-Pugh, age
	RoleAction    ColumnRole = "ACTION"    // THEN part: dose, recommendation
	RoleDrugName  ColumnRole = "DRUG_NAME" // Drug identifier
	RoleMetadata  ColumnRole = "METADATA"  // Notes, references, footnotes
	RoleUnknown   ColumnRole = "UNKNOWN"   // Needs manual classification
)

// NewTableNormalizer creates a normalizer with clinical column patterns
func NewTableNormalizer() *TableNormalizer {
	return &TableNormalizer{
		unitNormalizer: NewUnitNormalizer(),
		conditionPatterns: []columnPattern{
			// Renal function patterns (highest weight - most specific)
			{keywords: []string{"crcl", "creatinine clearance"}, role: RoleCondition, weight: 1.0},
			{keywords: []string{"egfr", "estimated gfr"}, role: RoleCondition, weight: 1.0},
			{keywords: []string{"gfr", "glomerular filtration"}, role: RoleCondition, weight: 0.95},
			{keywords: []string{"renal function", "renal impairment", "kidney function"}, role: RoleCondition, weight: 0.9},
			{keywords: []string{"ml/min", "ml per min"}, role: RoleCondition, weight: 0.85},

			// Hepatic function patterns
			{keywords: []string{"child-pugh", "child pugh", "childpugh"}, role: RoleCondition, weight: 1.0},
			{keywords: []string{"hepatic impairment", "hepatic function"}, role: RoleCondition, weight: 0.95},
			{keywords: []string{"liver function", "liver impairment"}, role: RoleCondition, weight: 0.9},
			{keywords: []string{"cirrhosis"}, role: RoleCondition, weight: 0.85},

			// Patient demographics patterns
			{keywords: []string{"age", "patient age"}, role: RoleCondition, weight: 0.85},
			{keywords: []string{"pediatric", "geriatric", "elderly"}, role: RoleCondition, weight: 0.8},
			{keywords: []string{"weight", "body weight", "bsa"}, role: RoleCondition, weight: 0.85},

			// Lab values
			{keywords: []string{"potassium", "k+"}, role: RoleCondition, weight: 0.8},
			{keywords: []string{"platelet", "plt"}, role: RoleCondition, weight: 0.8},
			{keywords: []string{"hemoglobin", "hgb"}, role: RoleCondition, weight: 0.8},

			// Generic condition indicators
			{keywords: []string{"population", "patient group", "category"}, role: RoleCondition, weight: 0.7},
			{keywords: []string{"indication", "condition"}, role: RoleCondition, weight: 0.6},
		},
		actionPatterns: []columnPattern{
			// Dose patterns (highest weight)
			{keywords: []string{"dose", "dosage", "dosing"}, role: RoleAction, weight: 1.0},
			{keywords: []string{"recommended dose", "starting dose"}, role: RoleAction, weight: 1.0},
			{keywords: []string{"maximum dose", "max dose", "maximum daily"}, role: RoleAction, weight: 0.95},
			{keywords: []string{"maintenance dose"}, role: RoleAction, weight: 0.95},
			{keywords: []string{"initial dose"}, role: RoleAction, weight: 0.95},

			// Adjustment patterns
			{keywords: []string{"adjustment", "dose adjustment"}, role: RoleAction, weight: 0.95},
			{keywords: []string{"modification", "dose modification"}, role: RoleAction, weight: 0.9},
			{keywords: []string{"reduction", "dose reduction"}, role: RoleAction, weight: 0.9},

			// Recommendation patterns
			{keywords: []string{"recommendation", "recommended"}, role: RoleAction, weight: 0.9},
			{keywords: []string{"clinical comment", "comments"}, role: RoleAction, weight: 0.8},
			{keywords: []string{"action", "clinical action"}, role: RoleAction, weight: 0.85},

			// Contraindication/warning patterns
			{keywords: []string{"contraindicated", "avoid"}, role: RoleAction, weight: 0.95},
			{keywords: []string{"do not use", "not recommended"}, role: RoleAction, weight: 0.95},
			{keywords: []string{"use with caution", "caution"}, role: RoleAction, weight: 0.85},
			{keywords: []string{"precaution"}, role: RoleAction, weight: 0.8},

			// Frequency patterns
			{keywords: []string{"frequency", "interval"}, role: RoleAction, weight: 0.85},
			{keywords: []string{"administration", "route"}, role: RoleAction, weight: 0.75},
		},
		drugPatterns: []columnPattern{
			{keywords: []string{"drug", "medication", "medicine"}, role: RoleDrugName, weight: 0.9},
			{keywords: []string{"interacting drug", "concomitant drug"}, role: RoleDrugName, weight: 0.95},
			{keywords: []string{"drug name", "generic name"}, role: RoleDrugName, weight: 1.0},
			{keywords: []string{"inhibitor", "inducer"}, role: RoleDrugName, weight: 0.8},
		},
	}
}

// =============================================================================
// NORMALIZED TABLE STRUCTURES
// =============================================================================

// NormalizedTable represents a table with classified columns
type NormalizedTable struct {
	ID                   string             `json:"id"`
	OriginalHeaders      []string           `json:"original_headers"`
	NormalizedCols       []NormalizedColumn `json:"normalized_columns"`
	Rows                 []NormalizedRow    `json:"rows"`
	Translatable         bool               `json:"translatable"`
	UntranslatableReason string             `json:"untranslatable_reason,omitempty"`
	ConditionColumnCount int                `json:"condition_column_count"`
	ActionColumnCount    int                `json:"action_column_count"`
	OverallConfidence    float64            `json:"overall_confidence"`
}

// NormalizedColumn contains column metadata
type NormalizedColumn struct {
	Index          int        `json:"index"`
	OriginalHeader string     `json:"original_header"`
	NormalizedName string     `json:"normalized_name"`
	Role           ColumnRole `json:"role"`
	Unit           string     `json:"unit,omitempty"`
	Confidence     float64    `json:"confidence"` // How confident in role assignment
	MatchedPattern string     `json:"matched_pattern,omitempty"`
}

// NormalizedRow contains normalized cell values
type NormalizedRow struct {
	Index int              `json:"index"`
	Cells []NormalizedCell `json:"cells"`
}

// NormalizedCell contains a normalized value
type NormalizedCell struct {
	ColumnIndex  int              `json:"column_index"`
	OriginalText string           `json:"original_text"`
	Normalized   *NormalizedValue `json:"normalized,omitempty"`
	IsEmpty      bool             `json:"is_empty"`
}

// =============================================================================
// TABLE NORMALIZATION
// =============================================================================

// Normalize processes a raw table into a normalized structure
func (n *TableNormalizer) Normalize(tableID string, headers []string, rows [][]string) (*NormalizedTable, error) {
	result := &NormalizedTable{
		ID:              tableID,
		OriginalHeaders: headers,
		Translatable:    true,
	}

	// Step 1: Classify each column
	var totalConfidence float64
	for i, header := range headers {
		col := n.classifyColumn(i, header)
		result.NormalizedCols = append(result.NormalizedCols, col)
		totalConfidence += col.Confidence

		switch col.Role {
		case RoleCondition:
			result.ConditionColumnCount++
		case RoleAction:
			result.ActionColumnCount++
		}
	}

	// Calculate overall confidence
	if len(headers) > 0 {
		result.OverallConfidence = totalConfidence / float64(len(headers))
	}

	// Step 2: Check if table is translatable
	if result.ConditionColumnCount == 0 {
		result.Translatable = false
		result.UntranslatableReason = "NO_CONDITION_COLUMN: Cannot identify IF part (no renal/hepatic/demographic columns found)"
	} else if result.ActionColumnCount == 0 {
		result.Translatable = false
		result.UntranslatableReason = "NO_ACTION_COLUMN: Cannot identify THEN part (no dose/recommendation columns found)"
	}

	// Step 3: Normalize row values
	for rowIdx, row := range rows {
		normalizedRow := NormalizedRow{Index: rowIdx}

		for colIdx, cell := range row {
			normalizedCell := NormalizedCell{
				ColumnIndex:  colIdx,
				OriginalText: strings.TrimSpace(cell),
				IsEmpty:      strings.TrimSpace(cell) == "",
			}

			// Apply normalization based on column role
			if colIdx < len(result.NormalizedCols) && !normalizedCell.IsEmpty {
				col := result.NormalizedCols[colIdx]
				normalizedCell.Normalized = n.normalizeCell(cell, col)
			}

			normalizedRow.Cells = append(normalizedRow.Cells, normalizedCell)
		}

		result.Rows = append(result.Rows, normalizedRow)
	}

	return result, nil
}

// =============================================================================
// COLUMN CLASSIFICATION
// =============================================================================

// classifyColumn determines the semantic role of a column
func (n *TableNormalizer) classifyColumn(index int, header string) NormalizedColumn {
	headerLower := strings.ToLower(strings.TrimSpace(header))

	col := NormalizedColumn{
		Index:          index,
		OriginalHeader: header,
		Role:           RoleUnknown,
		Confidence:     0.0,
	}

	// Check condition patterns first (priority for clinical conditions)
	for _, pattern := range n.conditionPatterns {
		for _, keyword := range pattern.keywords {
			if strings.Contains(headerLower, keyword) {
				if pattern.weight > col.Confidence {
					col.Role = RoleCondition
					col.NormalizedName = n.unitNormalizer.NormalizeVariable(header)
					col.Confidence = pattern.weight
					col.MatchedPattern = keyword
				}
			}
		}
	}

	// Check action patterns
	for _, pattern := range n.actionPatterns {
		for _, keyword := range pattern.keywords {
			if strings.Contains(headerLower, keyword) {
				if pattern.weight > col.Confidence {
					col.Role = RoleAction
					col.NormalizedName = "action." + sanitizeColumnName(headerLower)
					col.Confidence = pattern.weight
					col.MatchedPattern = keyword
				}
			}
		}
	}

	// Check drug name patterns
	for _, pattern := range n.drugPatterns {
		for _, keyword := range pattern.keywords {
			if strings.Contains(headerLower, keyword) {
				if pattern.weight > col.Confidence {
					col.Role = RoleDrugName
					col.NormalizedName = "drug_name"
					col.Confidence = pattern.weight
					col.MatchedPattern = keyword
				}
			}
		}
	}

	// If still unknown, set a low confidence normalized name
	if col.Role == RoleUnknown {
		col.NormalizedName = sanitizeColumnName(headerLower)
		col.Confidence = 0.3
	}

	// Try to extract unit from header
	col.Unit = n.extractUnitFromHeader(header)

	return col
}

// extractUnitFromHeader attempts to extract a unit from column header
func (n *TableNormalizer) extractUnitFromHeader(header string) string {
	headerLower := strings.ToLower(header)

	// Common patterns for units in headers
	unitPatterns := []struct {
		pattern string
		unit    string
	}{
		{"ml/min/1.73", "mL/min/1.73m²"},
		{"ml/min", "mL/min"},
		{"(mg)", "mg"},
		{"(mcg)", "mcg"},
		{"(%)", "percent"},
		{"(ml)", "mL"},
		{"mg/kg", "mg/kg"},
		{"years", "years"},
		{"kg", "kg"},
	}

	for _, up := range unitPatterns {
		if strings.Contains(headerLower, up.pattern) {
			return up.unit
		}
	}

	return ""
}

// =============================================================================
// CELL NORMALIZATION
// =============================================================================

// normalizeCell applies normalization based on column role
func (n *TableNormalizer) normalizeCell(cell string, col NormalizedColumn) *NormalizedValue {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return nil
	}

	switch col.Role {
	case RoleCondition:
		// Try to parse as GFR threshold
		if nv, err := n.unitNormalizer.ParseGFRThreshold(cell); err == nil {
			return nv
		}

		// Try to extract GFR range (e.g., "30-60")
		if min, max, err := n.unitNormalizer.ParseGFRRange(cell); err == nil {
			return &NormalizedValue{
				Variable:     col.NormalizedName,
				MinValue:     &min,
				MaxValue:     &max,
				Unit:         col.Unit,
				OriginalText: cell,
				Confidence:   0.85,
			}
		}

		// Try Child-Pugh normalization
		if childPugh, conf := n.unitNormalizer.NormalizeChildPugh(cell); childPugh != "" {
			return &NormalizedValue{
				Variable:     "hepatic.child_pugh",
				StringValue:  &childPugh,
				OriginalText: cell,
				Confidence:   conf,
			}
		}

		// Try renal category
		if category, conf := n.unitNormalizer.ParseRenalCategory(cell); category != "" {
			catStr := string(category)
			return &NormalizedValue{
				Variable:     "renal_function.category",
				StringValue:  &catStr,
				OriginalText: cell,
				Confidence:   conf,
			}
		}

		// Fallback: keep as string
		return &NormalizedValue{
			Variable:     col.NormalizedName,
			StringValue:  &cell,
			OriginalText: cell,
			Confidence:   0.5,
		}

	case RoleAction:
		// Try to parse dose
		if nv, err := n.unitNormalizer.ParseDose(cell); err == nil {
			nv.Variable = col.NormalizedName
			return nv
		}

		// Try to extract percentage
		if pct, err := n.unitNormalizer.ParsePercentage(cell); err == nil {
			return &NormalizedValue{
				Variable:     col.NormalizedName,
				NumericValue: pct,
				Unit:         "percent",
				OriginalText: cell,
				Confidence:   0.85,
			}
		}

		// Try to extract frequency
		if freq := n.unitNormalizer.ParseFrequency(cell); freq != "" {
			return &NormalizedValue{
				Variable:     col.NormalizedName + ".frequency",
				StringValue:  &freq,
				OriginalText: cell,
				Confidence:   0.8,
			}
		}

		// Keep as string for recommendations
		return &NormalizedValue{
			Variable:     col.NormalizedName,
			StringValue:  &cell,
			OriginalText: cell,
			Confidence:   0.7,
		}

	case RoleDrugName:
		return &NormalizedValue{
			Variable:     "drug_name",
			StringValue:  &cell,
			OriginalText: cell,
			Confidence:   0.9,
		}

	default:
		return &NormalizedValue{
			Variable:     col.NormalizedName,
			StringValue:  &cell,
			OriginalText: cell,
			Confidence:   0.5,
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// sanitizeColumnName converts a header to a safe column name
func sanitizeColumnName(header string) string {
	// Replace spaces and special chars with underscores
	result := strings.ToLower(header)
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "(", "")
	result = strings.ReplaceAll(result, ")", "")

	// Remove consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	return strings.Trim(result, "_")
}

// =============================================================================
// TABLE ANALYSIS
// =============================================================================

// GetConditionColumns returns all condition columns
func (t *NormalizedTable) GetConditionColumns() []NormalizedColumn {
	var cols []NormalizedColumn
	for _, col := range t.NormalizedCols {
		if col.Role == RoleCondition {
			cols = append(cols, col)
		}
	}
	return cols
}

// GetActionColumns returns all action columns
func (t *NormalizedTable) GetActionColumns() []NormalizedColumn {
	var cols []NormalizedColumn
	for _, col := range t.NormalizedCols {
		if col.Role == RoleAction {
			cols = append(cols, col)
		}
	}
	return cols
}

// GetRowConditionValues returns condition cell values for a row
func (t *NormalizedTable) GetRowConditionValues(rowIndex int) []*NormalizedCell {
	if rowIndex >= len(t.Rows) {
		return nil
	}

	var cells []*NormalizedCell
	row := t.Rows[rowIndex]

	for _, col := range t.NormalizedCols {
		if col.Role == RoleCondition && col.Index < len(row.Cells) {
			cells = append(cells, &row.Cells[col.Index])
		}
	}
	return cells
}

// GetRowActionValues returns action cell values for a row
func (t *NormalizedTable) GetRowActionValues(rowIndex int) []*NormalizedCell {
	if rowIndex >= len(t.Rows) {
		return nil
	}

	var cells []*NormalizedCell
	row := t.Rows[rowIndex]

	for _, col := range t.NormalizedCols {
		if col.Role == RoleAction && col.Index < len(row.Cells) {
			cells = append(cells, &row.Cells[col.Index])
		}
	}
	return cells
}
