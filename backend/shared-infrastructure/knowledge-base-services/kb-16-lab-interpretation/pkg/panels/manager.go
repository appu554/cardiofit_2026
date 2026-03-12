// Package panels provides lab panel assembly and pattern detection
package panels

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/pkg/integration"
	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// Manager handles panel operations
type Manager struct {
	resultStore *store.ResultStore
	refDB       *reference.Database
	kb8Client   *integration.KB8Client
	log         *logrus.Entry
}

// NewManager creates a new panel manager
func NewManager(resultStore *store.ResultStore, refDB *reference.Database, kb8Client *integration.KB8Client, log *logrus.Entry) *Manager {
	return &Manager{
		resultStore: resultStore,
		refDB:       refDB,
		kb8Client:   kb8Client,
		log:         log.WithField("component", "panel_manager"),
	}
}

// PanelDefinition defines a lab panel
type PanelDefinition struct {
	Type             types.PanelType
	Name             string
	Components       []string // LOINC codes
	CalculatedValues []string
	Patterns         []string
}

// Panel definitions
var PanelDefinitions = map[types.PanelType]PanelDefinition{
	types.PanelBMP: {
		Type: types.PanelBMP,
		Name: "Basic Metabolic Panel",
		Components: []string{
			"2951-2",  // Sodium
			"2823-3",  // Potassium
			"2075-0",  // Chloride
			"1963-8",  // CO2
			"3094-0",  // BUN
			"2160-0",  // Creatinine
			"2345-7",  // Glucose
			"17861-6", // Calcium
		},
		CalculatedValues: []string{"anion_gap"},
		Patterns:         []string{"aki", "electrolyte_disorder", "acidosis", "alkalosis"},
	},
	types.PanelCMP: {
		Type: types.PanelCMP,
		Name: "Comprehensive Metabolic Panel",
		Components: []string{
			// BMP components
			"2951-2", "2823-3", "2075-0", "1963-8", "3094-0", "2160-0", "2345-7", "17861-6",
			// Additional CMP components
			"1920-8", // AST
			"1742-6", // ALT
			"6768-6", // ALP
			"1975-2", // Bilirubin Total
			"1751-7", // Albumin
			"2885-2", // Total Protein
		},
		CalculatedValues: []string{"anion_gap", "globulin"},
		Patterns:         []string{"aki", "hepatocellular_injury", "cholestatic_injury", "mixed_liver"},
	},
	types.PanelCBC: {
		Type: types.PanelCBC,
		Name: "Complete Blood Count",
		Components: []string{
			"6690-2", // WBC
			"789-8",  // RBC
			"718-7",  // Hemoglobin
			"4544-3", // Hematocrit
			"777-3",  // Platelets
			"787-2",  // MCV
			"785-6",  // MCH
			"786-4",  // MCHC
		},
		Patterns: []string{"pancytopenia", "anemia_microcytic", "anemia_macrocytic", "anemia_normocytic", "thrombocytopenia", "leukopenia", "leukocytosis"},
	},
	types.PanelLFT: {
		Type: types.PanelLFT,
		Name: "Liver Function Tests",
		Components: []string{
			"1920-8", // AST
			"1742-6", // ALT
			"6768-6", // ALP
			"1975-2", // Bilirubin Total
			"1751-7", // Albumin
		},
		CalculatedValues: []string{"r_ratio"},
		Patterns:         []string{"hepatocellular", "cholestatic", "mixed"},
	},
	types.PanelLipid: {
		Type: types.PanelLipid,
		Name: "Lipid Panel",
		Components: []string{
			"2093-3",  // Total Cholesterol
			"2571-8",  // Triglycerides
			"2085-9",  // HDL
			"13457-7", // LDL (calculated)
		},
		CalculatedValues: []string{"non_hdl", "tc_hdl_ratio"},
		Patterns:         []string{"hyperlipidemia", "low_hdl"},
	},
	types.PanelRenal: {
		Type: types.PanelRenal,
		Name: "Renal Function Panel",
		Components: []string{
			"2160-0",  // Creatinine
			"3094-0",  // BUN
			"33914-3", // eGFR
			"14959-1", // Microalbumin/Creatinine ratio
		},
		CalculatedValues: []string{"egfr", "bun_cr_ratio"},
		Patterns:         []string{"aki_stage_1", "aki_stage_2", "aki_stage_3", "ckd_staging"},
	},
}

// GetPanelDefinition returns a panel definition by type
func (m *Manager) GetPanelDefinition(panelType types.PanelType) (*PanelDefinition, error) {
	def, exists := PanelDefinitions[panelType]
	if !exists {
		return nil, fmt.Errorf("unknown panel type: %s", panelType)
	}
	return &def, nil
}

// ListPanelDefinitions returns all available panel definitions
func (m *Manager) ListPanelDefinitions() []PanelDefinition {
	defs := make([]PanelDefinition, 0, len(PanelDefinitions))
	for _, def := range PanelDefinitions {
		defs = append(defs, def)
	}
	return defs
}

// AssemblePanel assembles a panel from patient results
func (m *Manager) AssemblePanel(ctx context.Context, patientID string, panelType types.PanelType, lookbackDays int) (*types.AssembledPanel, error) {
	def, err := m.GetPanelDefinition(panelType)
	if err != nil {
		return nil, err
	}

	if lookbackDays <= 0 {
		lookbackDays = 7 // Default to 7 days
	}

	// Get recent results
	results, err := m.resultStore.GetRecentByPatient(ctx, patientID, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient results: %w", err)
	}

	// Map results by code (most recent for each)
	resultMap := make(map[string]*types.LabResult)
	for i := range results {
		r := &results[i]
		existing, exists := resultMap[r.Code]
		if !exists || r.CollectedAt.After(existing.CollectedAt) {
			resultMap[r.Code] = r
		}
	}

	// Assemble panel components
	components := make([]types.PanelComponent, 0, len(def.Components))
	for _, code := range def.Components {
		testDef := m.refDB.GetTest(code)
		component := types.PanelComponent{
			Code:     code,
			Name:     code,
			Required: true,
		}
		if testDef != nil {
			component.Name = testDef.Name
		}

		if result, exists := resultMap[code]; exists {
			component.Result = result
			component.Available = true
		}
		components = append(components, component)
	}

	// Calculate completeness
	available := 0
	for _, c := range components {
		if c.Available {
			available++
		}
	}
	completeness := float64(available) / float64(len(def.Components)) * 100

	panel := &types.AssembledPanel{
		Type:              panelType,
		Name:              def.Name,
		PatientID:         patientID,
		Components:        components,
		Completeness:      completeness,
		AssembledAt:       time.Now(),
		CalculatedValues:  make(map[string]float64),
		DetectedPatterns:  []types.DetectedPattern{},
	}

	// Calculate derived values (uses KB-8 for calculations)
	m.calculateDerivedValues(ctx, panel, resultMap)

	// Detect patterns
	panel.DetectedPatterns = m.DetectPatterns(panel, resultMap)

	return panel, nil
}

// DetectAvailablePanels detects which panels can be assembled from patient data
func (m *Manager) DetectAvailablePanels(ctx context.Context, patientID string, lookbackDays int) ([]types.AvailablePanel, error) {
	if lookbackDays <= 0 {
		lookbackDays = 7
	}

	results, err := m.resultStore.GetRecentByPatient(ctx, patientID, lookbackDays)
	if err != nil {
		return nil, err
	}

	// Get available codes
	availableCodes := make(map[string]bool)
	for _, r := range results {
		availableCodes[r.Code] = true
	}

	available := make([]types.AvailablePanel, 0)
	for panelType, def := range PanelDefinitions {
		matchedCount := 0
		for _, code := range def.Components {
			if availableCodes[code] {
				matchedCount++
			}
		}

		completeness := float64(matchedCount) / float64(len(def.Components)) * 100

		if completeness >= 50 { // At least 50% complete
			available = append(available, types.AvailablePanel{
				Type:         panelType,
				Name:         def.Name,
				Completeness: completeness,
				Available:    matchedCount,
				Total:        len(def.Components),
			})
		}
	}

	return available, nil
}

// DetectPatterns detects clinical patterns in a panel
func (m *Manager) DetectPatterns(panel *types.AssembledPanel, resultMap map[string]*types.LabResult) []types.DetectedPattern {
	var patterns []types.DetectedPattern

	switch panel.Type {
	case types.PanelBMP, types.PanelCMP:
		patterns = append(patterns, m.detectBMPPatterns(panel, resultMap)...)
	case types.PanelCBC:
		patterns = append(patterns, m.detectCBCPatterns(resultMap)...)
	case types.PanelLFT:
		patterns = append(patterns, m.detectLFTPatterns(resultMap)...)
	case types.PanelLipid:
		patterns = append(patterns, m.detectLipidPatterns(resultMap)...)
	case types.PanelRenal:
		patterns = append(patterns, m.detectRenalPatterns(resultMap)...)
	}

	return patterns
}

// calculateDerivedValues calculates derived values for a panel using KB-8
func (m *Manager) calculateDerivedValues(ctx context.Context, panel *types.AssembledPanel, resultMap map[string]*types.LabResult) {
	switch panel.Type {
	case types.PanelBMP, types.PanelCMP:
		// Anion Gap - calculated via KB-8
		na := m.getValue(resultMap, "2951-2")
		cl := m.getValue(resultMap, "2075-0")
		co2 := m.getValue(resultMap, "1963-8")
		if na != nil && cl != nil && co2 != nil && m.kb8Client != nil {
			result, err := m.kb8Client.CalculateAnionGap(ctx, integration.CalculateAnionGapRequest{
				PatientID:   panel.PatientID,
				Sodium:      *na,
				Chloride:    *cl,
				Bicarbonate: *co2,
			})
			if err != nil {
				m.log.WithError(err).Warn("KB-8 anion gap calculation failed")
			} else {
				panel.CalculatedValues["anion_gap"] = result.Value
				// Store additional KB-8 metadata if needed
				if result.CorrectedValue != nil {
					panel.CalculatedValues["anion_gap_corrected"] = *result.CorrectedValue
				}
			}
		}

		// BUN/Cr ratio - simple ratio, calculated locally as it's not a KB-8 calculator
		bun := m.getValue(resultMap, "3094-0")
		cr := m.getValue(resultMap, "2160-0")
		if bun != nil && cr != nil && *cr > 0 {
			panel.CalculatedValues["bun_cr_ratio"] = *bun / *cr
		}

	case types.PanelLFT:
		// R-ratio for liver injury classification
		// R = (ALT / ALT_ULN) / (ALP / ALP_ULN)
		// Note: This is a simple ratio not requiring KB-8
		alt := m.getValue(resultMap, "1742-6")
		alp := m.getValue(resultMap, "6768-6")
		if alt != nil && alp != nil {
			altULN := 40.0
			alpULN := 120.0
			rRatio := (*alt / altULN) / (*alp / alpULN)
			panel.CalculatedValues["r_ratio"] = rRatio
		}

	case types.PanelLipid:
		// Non-HDL = TC - HDL (simple subtraction, not KB-8)
		tc := m.getValue(resultMap, "2093-3")
		hdl := m.getValue(resultMap, "2085-9")
		if tc != nil && hdl != nil {
			panel.CalculatedValues["non_hdl"] = *tc - *hdl
		}

		// TC/HDL ratio (simple ratio, not KB-8)
		if tc != nil && hdl != nil && *hdl > 0 {
			panel.CalculatedValues["tc_hdl_ratio"] = *tc / *hdl
		}

	case types.PanelRenal:
		// eGFR - calculated via KB-8
		cr := m.getValue(resultMap, "2160-0")
		if cr != nil && m.kb8Client != nil {
			result, err := m.kb8Client.GetEGFRForPatient(ctx, panel.PatientID)
			if err != nil {
				m.log.WithError(err).Warn("KB-8 eGFR calculation failed")
			} else {
				panel.CalculatedValues["egfr"] = result.Value
				// Store CKD stage for pattern detection
				if result.CKDStage != "" {
					panel.CalculatedValues["ckd_stage_numeric"] = m.ckdStageToNumeric(result.CKDStage)
				}
			}
		}

		// BUN/Cr ratio (simple ratio, not KB-8)
		bun := m.getValue(resultMap, "3094-0")
		crForRatio := m.getValue(resultMap, "2160-0")
		if bun != nil && crForRatio != nil && *crForRatio > 0 {
			panel.CalculatedValues["bun_cr_ratio"] = *bun / *crForRatio
		}
	}
}

// ckdStageToNumeric converts CKD stage string to numeric for pattern detection
func (m *Manager) ckdStageToNumeric(stage string) float64 {
	switch stage {
	case "G1":
		return 1
	case "G2":
		return 2
	case "G3a":
		return 3.1
	case "G3b":
		return 3.2
	case "G4":
		return 4
	case "G5":
		return 5
	default:
		return 0
	}
}

// detectBMPPatterns detects patterns in BMP/CMP panels
func (m *Manager) detectBMPPatterns(panel *types.AssembledPanel, resultMap map[string]*types.LabResult) []types.DetectedPattern {
	var patterns []types.DetectedPattern

	// High anion gap metabolic acidosis
	if ag, exists := panel.CalculatedValues["anion_gap"]; exists {
		if ag > 16 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "high_anion_gap_acidosis",
				Name:        "High Anion Gap Metabolic Acidosis",
				Confidence:  0.85,
				Description: fmt.Sprintf("Anion gap elevated at %.1f (normal 8-12)", ag),
				Severity:    types.SeverityHigh,
			})
		}
	}

	// AKI detection based on creatinine
	cr := m.getValue(resultMap, "2160-0")
	if cr != nil && *cr > 1.5 {
		severity := types.SeverityMedium
		stage := "Stage 1"
		if *cr > 3.0 {
			severity = types.SeverityCritical
			stage = "Stage 3"
		} else if *cr > 2.0 {
			severity = types.SeverityHigh
			stage = "Stage 2"
		}
		patterns = append(patterns, types.DetectedPattern{
			Code:        "aki",
			Name:        fmt.Sprintf("Acute Kidney Injury - %s", stage),
			Confidence:  0.75,
			Description: fmt.Sprintf("Creatinine elevated at %.2f mg/dL", *cr),
			Severity:    severity,
		})
	}

	// Electrolyte disorders
	k := m.getValue(resultMap, "2823-3")
	if k != nil {
		if *k < 3.5 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "hypokalemia",
				Name:        "Hypokalemia",
				Confidence:  0.9,
				Description: fmt.Sprintf("Potassium low at %.1f mEq/L", *k),
				Severity:    types.SeverityMedium,
			})
		} else if *k > 5.0 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "hyperkalemia",
				Name:        "Hyperkalemia",
				Confidence:  0.9,
				Description: fmt.Sprintf("Potassium elevated at %.1f mEq/L", *k),
				Severity:    types.SeverityHigh,
			})
		}
	}

	na := m.getValue(resultMap, "2951-2")
	if na != nil {
		if *na < 135 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "hyponatremia",
				Name:        "Hyponatremia",
				Confidence:  0.9,
				Description: fmt.Sprintf("Sodium low at %.0f mEq/L", *na),
				Severity:    types.SeverityMedium,
			})
		} else if *na > 145 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "hypernatremia",
				Name:        "Hypernatremia",
				Confidence:  0.9,
				Description: fmt.Sprintf("Sodium elevated at %.0f mEq/L", *na),
				Severity:    types.SeverityMedium,
			})
		}
	}

	return patterns
}

// detectCBCPatterns detects patterns in CBC panel
func (m *Manager) detectCBCPatterns(resultMap map[string]*types.LabResult) []types.DetectedPattern {
	var patterns []types.DetectedPattern

	hgb := m.getValue(resultMap, "718-7")
	mcv := m.getValue(resultMap, "787-2")
	plt := m.getValue(resultMap, "777-3")
	wbc := m.getValue(resultMap, "6690-2")

	// Anemia classification
	if hgb != nil && *hgb < 12 {
		anemiaType := "Normocytic"
		if mcv != nil {
			if *mcv < 80 {
				anemiaType = "Microcytic"
			} else if *mcv > 100 {
				anemiaType = "Macrocytic"
			}
		}
		severity := types.SeverityLow
		if *hgb < 8 {
			severity = types.SeverityHigh
		} else if *hgb < 10 {
			severity = types.SeverityMedium
		}
		patterns = append(patterns, types.DetectedPattern{
			Code:        fmt.Sprintf("anemia_%s", anemiaType),
			Name:        fmt.Sprintf("%s Anemia", anemiaType),
			Confidence:  0.85,
			Description: fmt.Sprintf("Hemoglobin %.1f g/dL with MCV indicating %s pattern", *hgb, anemiaType),
			Severity:    severity,
		})
	}

	// Thrombocytopenia
	if plt != nil && *plt < 150000 {
		severity := types.SeverityLow
		if *plt < 50000 {
			severity = types.SeverityCritical
		} else if *plt < 100000 {
			severity = types.SeverityHigh
		}
		patterns = append(patterns, types.DetectedPattern{
			Code:        "thrombocytopenia",
			Name:        "Thrombocytopenia",
			Confidence:  0.9,
			Description: fmt.Sprintf("Platelet count low at %.0f K/uL", *plt/1000),
			Severity:    severity,
		})
	}

	// Leukocytosis/Leukopenia
	if wbc != nil {
		if *wbc < 4.0 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "leukopenia",
				Name:        "Leukopenia",
				Confidence:  0.85,
				Description: fmt.Sprintf("WBC low at %.1f K/uL", *wbc),
				Severity:    types.SeverityMedium,
			})
		} else if *wbc > 11.0 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "leukocytosis",
				Name:        "Leukocytosis",
				Confidence:  0.85,
				Description: fmt.Sprintf("WBC elevated at %.1f K/uL", *wbc),
				Severity:    types.SeverityMedium,
			})
		}
	}

	// Pancytopenia
	if hgb != nil && plt != nil && wbc != nil {
		if *hgb < 10 && *plt < 100000 && *wbc < 4.0 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "pancytopenia",
				Name:        "Pancytopenia",
				Confidence:  0.9,
				Description: "Low counts across all cell lines - requires investigation",
				Severity:    types.SeverityHigh,
			})
		}
	}

	return patterns
}

// detectLFTPatterns detects patterns in liver function tests
func (m *Manager) detectLFTPatterns(resultMap map[string]*types.LabResult) []types.DetectedPattern {
	var patterns []types.DetectedPattern

	alt := m.getValue(resultMap, "1742-6")
	ast := m.getValue(resultMap, "1920-8")
	alp := m.getValue(resultMap, "6768-6")
	bili := m.getValue(resultMap, "1975-2")

	// Calculate R-ratio if possible
	if alt != nil && alp != nil {
		altULN := 40.0
		alpULN := 120.0
		rRatio := (*alt / altULN) / (*alp / alpULN)

		if rRatio > 5 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "hepatocellular_injury",
				Name:        "Hepatocellular Pattern",
				Confidence:  0.85,
				Description: fmt.Sprintf("R-ratio %.1f indicates hepatocellular injury pattern", rRatio),
				Severity:    types.SeverityHigh,
			})
		} else if rRatio < 2 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "cholestatic_injury",
				Name:        "Cholestatic Pattern",
				Confidence:  0.85,
				Description: fmt.Sprintf("R-ratio %.1f indicates cholestatic injury pattern", rRatio),
				Severity:    types.SeverityHigh,
			})
		} else {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "mixed_liver_injury",
				Name:        "Mixed Liver Injury Pattern",
				Confidence:  0.75,
				Description: fmt.Sprintf("R-ratio %.1f indicates mixed injury pattern", rRatio),
				Severity:    types.SeverityMedium,
			})
		}
	}

	// AST:ALT ratio (alcoholic vs non-alcoholic)
	if ast != nil && alt != nil && *alt > 0 {
		astAltRatio := *ast / *alt
		if astAltRatio > 2 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "ast_alt_ratio_elevated",
				Name:        "Elevated AST:ALT Ratio",
				Confidence:  0.7,
				Description: fmt.Sprintf("AST:ALT ratio %.1f may suggest alcoholic liver disease", astAltRatio),
				Severity:    types.SeverityMedium,
			})
		}
	}

	// Hyperbilirubinemia
	if bili != nil && *bili > 1.2 {
		severity := types.SeverityLow
		if *bili > 5 {
			severity = types.SeverityHigh
		} else if *bili > 2.5 {
			severity = types.SeverityMedium
		}
		patterns = append(patterns, types.DetectedPattern{
			Code:        "hyperbilirubinemia",
			Name:        "Hyperbilirubinemia",
			Confidence:  0.9,
			Description: fmt.Sprintf("Total bilirubin elevated at %.1f mg/dL", *bili),
			Severity:    severity,
		})
	}

	return patterns
}

// detectLipidPatterns detects patterns in lipid panel
func (m *Manager) detectLipidPatterns(resultMap map[string]*types.LabResult) []types.DetectedPattern {
	var patterns []types.DetectedPattern

	tc := m.getValue(resultMap, "2093-3")
	ldl := m.getValue(resultMap, "13457-7")
	hdl := m.getValue(resultMap, "2085-9")
	tg := m.getValue(resultMap, "2571-8")

	// Hypercholesterolemia
	if tc != nil && *tc > 200 {
		severity := types.SeverityLow
		if *tc > 300 {
			severity = types.SeverityHigh
		} else if *tc > 240 {
			severity = types.SeverityMedium
		}
		patterns = append(patterns, types.DetectedPattern{
			Code:        "hypercholesterolemia",
			Name:        "Hypercholesterolemia",
			Confidence:  0.9,
			Description: fmt.Sprintf("Total cholesterol elevated at %.0f mg/dL", *tc),
			Severity:    severity,
		})
	}

	// High LDL
	if ldl != nil && *ldl > 130 {
		patterns = append(patterns, types.DetectedPattern{
			Code:        "elevated_ldl",
			Name:        "Elevated LDL Cholesterol",
			Confidence:  0.9,
			Description: fmt.Sprintf("LDL cholesterol %.0f mg/dL above optimal (<100)", *ldl),
			Severity:    types.SeverityMedium,
		})
	}

	// Low HDL
	if hdl != nil && *hdl < 40 {
		patterns = append(patterns, types.DetectedPattern{
			Code:        "low_hdl",
			Name:        "Low HDL Cholesterol",
			Confidence:  0.9,
			Description: fmt.Sprintf("HDL cholesterol low at %.0f mg/dL", *hdl),
			Severity:    types.SeverityMedium,
		})
	}

	// Hypertriglyceridemia
	if tg != nil && *tg > 150 {
		severity := types.SeverityLow
		if *tg > 500 {
			severity = types.SeverityHigh
		} else if *tg > 200 {
			severity = types.SeverityMedium
		}
		patterns = append(patterns, types.DetectedPattern{
			Code:        "hypertriglyceridemia",
			Name:        "Hypertriglyceridemia",
			Confidence:  0.9,
			Description: fmt.Sprintf("Triglycerides elevated at %.0f mg/dL", *tg),
			Severity:    severity,
		})
	}

	return patterns
}

// detectRenalPatterns detects patterns in renal function panel
func (m *Manager) detectRenalPatterns(resultMap map[string]*types.LabResult) []types.DetectedPattern {
	var patterns []types.DetectedPattern

	cr := m.getValue(resultMap, "2160-0")
	egfr := m.getValue(resultMap, "33914-3")
	bun := m.getValue(resultMap, "3094-0")

	// CKD staging based on eGFR
	if egfr != nil {
		var ckdStage string
		var severity types.Severity

		switch {
		case *egfr >= 90:
			// Normal
		case *egfr >= 60:
			ckdStage = "Stage 2"
			severity = types.SeverityLow
		case *egfr >= 45:
			ckdStage = "Stage 3a"
			severity = types.SeverityMedium
		case *egfr >= 30:
			ckdStage = "Stage 3b"
			severity = types.SeverityMedium
		case *egfr >= 15:
			ckdStage = "Stage 4"
			severity = types.SeverityHigh
		default:
			ckdStage = "Stage 5 (ESRD)"
			severity = types.SeverityCritical
		}

		if ckdStage != "" {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "ckd",
				Name:        fmt.Sprintf("Chronic Kidney Disease - %s", ckdStage),
				Confidence:  0.85,
				Description: fmt.Sprintf("eGFR %.0f mL/min/1.73m² indicates %s", *egfr, ckdStage),
				Severity:    severity,
			})
		}
	}

	// BUN/Cr ratio for prerenal vs intrinsic
	if bun != nil && cr != nil && *cr > 0 {
		bunCrRatio := *bun / *cr
		if bunCrRatio > 20 {
			patterns = append(patterns, types.DetectedPattern{
				Code:        "prerenal_azotemia",
				Name:        "Prerenal Azotemia Pattern",
				Confidence:  0.7,
				Description: fmt.Sprintf("BUN/Cr ratio %.1f suggests prerenal cause", bunCrRatio),
				Severity:    types.SeverityMedium,
			})
		}
	}

	return patterns
}

// getValue safely extracts a numeric value from results map
func (m *Manager) getValue(resultMap map[string]*types.LabResult, code string) *float64 {
	if result, exists := resultMap[code]; exists && result.ValueNumeric != nil {
		return result.ValueNumeric
	}
	return nil
}

// NOTE: All clinical calculations (eGFR, Anion Gap) are performed by KB-8 Calculator Service.
// This ensures CQL-based calculation with proper provenance tracking for SaMD compliance.
// No local calculation fallbacks are used - KB-8 is the single source of truth.
