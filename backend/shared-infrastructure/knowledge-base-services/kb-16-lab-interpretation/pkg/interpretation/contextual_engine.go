// Package interpretation provides context-aware clinical interpretation of lab results
// Phase 3b.6: Enhanced engine with conditional reference range selection
package interpretation

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// ContextualEngine performs context-aware clinical interpretation using conditional reference ranges
// This enhances the base Engine by selecting reference ranges based on patient context
// (pregnancy/trimester, CKD stage, age, neonatal status)
type ContextualEngine struct {
	*Engine                                    // Embed base engine for standard functionality
	rangeSelector       *reference.RangeSelector
	bilirubinInterpreter *reference.BilirubinInterpreter
	db                  *gorm.DB
	log                 *logrus.Entry
}

// NewContextualEngine creates a new context-aware interpretation engine
func NewContextualEngine(
	refDB *reference.Database,
	resultStore *store.ResultStore,
	db *gorm.DB,
	log *logrus.Entry,
) *ContextualEngine {
	baseEngine := NewEngine(refDB, resultStore, log)

	return &ContextualEngine{
		Engine:               baseEngine,
		rangeSelector:        reference.NewRangeSelector(db),
		bilirubinInterpreter: reference.NewBilirubinInterpreter(db),
		db:                   db,
		log:                  log.WithField("component", "contextual_engine"),
	}
}

// InterpretWithContext analyzes a lab result using context-aware conditional reference ranges
// This is the primary method for Phase 3b.6 functionality
func (e *ContextualEngine) InterpretWithContext(ctx context.Context, result *types.LabResult, patientCtx *types.PatientContext) (*ContextualInterpretedResult, error) {
	if result.ValueNumeric == nil {
		// Delegate non-numeric results to base engine
		baseResult, err := e.Engine.Interpret(ctx, result, patientCtx)
		if err != nil {
			return nil, err
		}
		return &ContextualInterpretedResult{
			InterpretedResult:    *baseResult,
			ContextApplied:       "Standard",
			RangeSourceAuthority: "CLSI",
			RangeSourceRef:       "CLSI C28-A3c",
		}, nil
	}

	// Special handling for neonatal bilirubin
	if e.isNeonatalBilirubinTest(result.Code) && patientCtx != nil && patientCtx.IsNeonate {
		return e.interpretNeonatalBilirubin(ctx, result, patientCtx)
	}

	// Use conditional range selection
	return e.interpretWithConditionalRange(ctx, result, patientCtx)
}

// interpretWithConditionalRange uses the RangeSelector to find the most specific matching range
func (e *ContextualEngine) interpretWithConditionalRange(ctx context.Context, result *types.LabResult, patientCtx *types.PatientContext) (*ContextualInterpretedResult, error) {
	// Convert PatientContext to enhanced version with all clinical fields populated
	var enhancedCtx *types.PatientContext
	if patientCtx != nil {
		enhancedCtx = e.toEnhancedPatientContext(patientCtx)
	} else {
		// Create minimal context for adult if not provided
		enhancedCtx = &types.PatientContext{
			PatientID: "unknown",
			Age:       40,
			Sex:       "",
		}
	}

	// Select the most specific matching range
	selectedRange, err := e.rangeSelector.SelectRange(ctx, result.Code, enhancedCtx)
	if err != nil {
		// Fallback to base engine if no conditional range found
		e.log.WithError(err).WithField("code", result.Code).Debug("No conditional range found, using base engine")
		baseResult, baseErr := e.Engine.Interpret(ctx, result, patientCtx)
		if baseErr != nil {
			return nil, baseErr
		}
		return &ContextualInterpretedResult{
			InterpretedResult:    *baseResult,
			ContextApplied:       "Standard (fallback)",
			RangeSourceAuthority: "CLSI",
			RangeSourceRef:       "CLSI C28-A3c",
		}, nil
	}

	// Interpret using the selected range
	rangeInterpretation := selectedRange.Interpret(*result.ValueNumeric, result.Unit)

	// Generate enhanced clinical comment
	comment := e.generateEnhancedComment(result, rangeInterpretation, selectedRange, enhancedCtx)

	// Generate recommendations
	recommendations := e.generateContextualRecommendations(result, rangeInterpretation, selectedRange, enhancedCtx)

	// Perform delta check using base engine
	deltaResult := e.Engine.performDeltaCheck(ctx, result)

	// Get deviation percent (default to 0 if nil)
	deviationPercent := float64(0)
	if rangeInterpretation.DeviationPercent != nil {
		deviationPercent = *rangeInterpretation.DeviationPercent
	}

	return &ContextualInterpretedResult{
		InterpretedResult: types.InterpretedResult{
			Result: *result,
			Interpretation: types.Interpretation{
				Flag:               types.InterpretationFlag(rangeInterpretation.Flag),
				Severity:           e.determineSeverityFromFlag(rangeInterpretation.Flag),
				IsCritical:         e.isCriticalFlag(rangeInterpretation.Flag),
				IsPanic:            e.isPanicFlag(rangeInterpretation.Flag),
				RequiresAction:     e.requiresAction(rangeInterpretation.Flag),
				DeviationPercent:   deviationPercent,
				DeviationDirection: rangeInterpretation.DeviationDirection,
				DeltaCheck:         deltaResult,
				ClinicalComment:    comment,
				Recommendations:    recommendations,
			},
		},
		ContextApplied:       rangeInterpretation.ContextApplied,
		SpecificityScore:     rangeInterpretation.SpecificityScore,
		RangeSourceAuthority: selectedRange.Authority,
		RangeSourceRef:       selectedRange.AuthorityRef,
		RangeID:              rangeInterpretation.RangeID.String(),
		SelectedLowNormal:    rangeInterpretation.LowNormal,
		SelectedHighNormal:   rangeInterpretation.HighNormal,
	}, nil
}

// interpretNeonatalBilirubin uses the Bhutani nomogram for neonatal bilirubin
func (e *ContextualEngine) interpretNeonatalBilirubin(ctx context.Context, result *types.LabResult, patientCtx *types.PatientContext) (*ContextualInterpretedResult, error) {
	enhancedCtx := e.toEnhancedPatientContext(patientCtx)

	// Ensure neonatal parameters are set
	if enhancedCtx.HoursOfLife <= 0 || enhancedCtx.GestationalAgeAtBirth <= 0 {
		return nil, fmt.Errorf("neonatal bilirubin interpretation requires hours_of_life and gestational_age_at_birth")
	}

	// Use bilirubin interpreter
	biliInterpretation, err := e.bilirubinInterpreter.InterpretBilirubin(ctx, *result.ValueNumeric, enhancedCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to interpret neonatal bilirubin: %w", err)
	}

	// Determine flag from bilirubin interpretation
	flag := e.bilirubinToFlag(biliInterpretation)

	// Generate recommendation
	zone := reference.BilirubinZone(biliInterpretation.Zone)
	recommendation := e.bilirubinInterpreter.GetZoneRecommendation(zone, biliInterpretation)

	// Check if specialty consult needed
	needsConsult, consultReason := e.bilirubinInterpreter.NeedsSpecialtyConsult(biliInterpretation)

	var recommendations []types.Recommendation
	recommendations = append(recommendations, types.Recommendation{
		Type:        "action",
		Priority:    e.bilirubinPriority(biliInterpretation),
		Description: recommendation,
	})
	if needsConsult {
		recommendations = append(recommendations, types.Recommendation{
			Type:        "consultation",
			Priority:    "HIGH",
			Description: consultReason,
		})
	}

	// Build comment
	comment := fmt.Sprintf(
		"Neonatal bilirubin %.1f mg/dL at %d hours of life (GA %d weeks, %s risk). Phototherapy threshold: %.1f mg/dL. Zone: %s.",
		biliInterpretation.Value,
		biliInterpretation.HoursOfLife,
		biliInterpretation.GestationalAge,
		biliInterpretation.RiskCategory,
		biliInterpretation.PhotoThreshold,
		biliInterpretation.Zone,
	)

	return &ContextualInterpretedResult{
		InterpretedResult: types.InterpretedResult{
			Result: *result,
			Interpretation: types.Interpretation{
				Flag:               flag,
				Severity:           e.determineSeverityFromFlag(reference.InterpretationFlag(flag)),
				IsCritical:         biliInterpretation.NeedsPhototherapy,
				IsPanic:            biliInterpretation.NeedsExchange,
				RequiresAction:     biliInterpretation.NeedsPhototherapy || biliInterpretation.NeedsExchange,
				ClinicalComment:    comment,
				Recommendations:    recommendations,
			},
		},
		ContextApplied:       fmt.Sprintf("Neonatal %s risk, %dh of life", biliInterpretation.RiskCategory, biliInterpretation.HoursOfLife),
		SpecificityScore:     10, // Highly specific
		RangeSourceAuthority: biliInterpretation.Authority,
		RangeSourceRef:       biliInterpretation.AuthorityRef,
		BilirubinInterpretation: biliInterpretation,
	}, nil
}

// toEnhancedPatientContext converts standard PatientContext to enhanced version with all clinical fields populated
func (e *ContextualEngine) toEnhancedPatientContext(patientCtx *types.PatientContext) *types.PatientContext {
	if patientCtx == nil {
		return &types.PatientContext{
			PatientID: "unknown",
			Age:       40,
			Sex:       "",
		}
	}

	// Create a copy with all fields populated
	enhanced := &types.PatientContext{
		PatientID:             patientCtx.PatientID,
		Age:                   patientCtx.Age,
		AgeInDays:             patientCtx.AgeInDays,
		Sex:                   patientCtx.Sex,
		Conditions:            patientCtx.Conditions,
		Medications:           patientCtx.Medications,
		Phenotypes:            patientCtx.Phenotypes,
		IsPregnant:            patientCtx.IsPregnant,
		Trimester:             patientCtx.Trimester,
		GestationalWeek:       patientCtx.GestationalWeek,
		IsPostpartum:          patientCtx.IsPostpartum,
		IsLactating:           patientCtx.IsLactating,
		IsNeonate:             patientCtx.IsNeonate,
		GestationalAgeAtBirth: patientCtx.GestationalAgeAtBirth,
		HoursOfLife:           patientCtx.HoursOfLife,
		NeonatalRiskCategory:  patientCtx.NeonatalRiskCategory,
		CKDStage:              patientCtx.CKDStage,
		EGFR:                  patientCtx.EGFR,
		IsOnDialysis:          patientCtx.IsOnDialysis,
		ChildPughClass:        patientCtx.ChildPughClass,
	}

	// Infer pregnancy status from conditions if not explicitly set
	if !enhanced.IsPregnant && containsCondition(patientCtx.Conditions, "pregnant", "pregnancy") {
		enhanced.IsPregnant = true
		// Try to determine trimester from conditions
		if containsCondition(patientCtx.Conditions, "first trimester", "trimester 1") {
			enhanced.Trimester = 1
			enhanced.GestationalWeek = 10 // Approximate
		} else if containsCondition(patientCtx.Conditions, "second trimester", "trimester 2") {
			enhanced.Trimester = 2
			enhanced.GestationalWeek = 20
		} else if containsCondition(patientCtx.Conditions, "third trimester", "trimester 3") {
			enhanced.Trimester = 3
			enhanced.GestationalWeek = 32
		} else {
			enhanced.Trimester = 2 // Default to T2 if unknown
			enhanced.GestationalWeek = 20
		}
	}

	// Infer CKD status from conditions if not explicitly set
	if enhanced.CKDStage == 0 && containsCondition(patientCtx.Conditions, "ckd", "chronic kidney disease", "renal failure", "esrd") {
		// Try to determine stage
		if containsCondition(patientCtx.Conditions, "stage 5", "esrd", "end stage") {
			enhanced.CKDStage = 5
		} else if containsCondition(patientCtx.Conditions, "stage 4") {
			enhanced.CKDStage = 4
		} else if containsCondition(patientCtx.Conditions, "stage 3") {
			enhanced.CKDStage = 3
		} else {
			enhanced.CKDStage = 3 // Default to stage 3 if unknown
		}
	}

	// Infer dialysis status from conditions
	if !enhanced.IsOnDialysis && containsCondition(patientCtx.Conditions, "dialysis", "hemodialysis", "peritoneal dialysis") {
		enhanced.IsOnDialysis = true
		enhanced.CKDStage = 5
	}

	// Determine neonatal status if appropriate
	if patientCtx.Age < 1 && patientCtx.GestationalAgeAtBirth > 0 {
		enhanced.IsNeonate = true
		// Determine risk category based on gestational age at birth (AAP 2022)
		switch {
		case patientCtx.GestationalAgeAtBirth >= 38:
			enhanced.NeonatalRiskCategory = "LOW"
		case patientCtx.GestationalAgeAtBirth >= 35:
			enhanced.NeonatalRiskCategory = "MEDIUM"
		default:
			enhanced.NeonatalRiskCategory = "HIGH"
		}
	}

	return enhanced
}

// buildInterpretation creates the interpretation structure from range interpretation
func (e *ContextualEngine) buildInterpretation(result *types.LabResult, rangeInterp *reference.RangeInterpretation, patientCtx *types.PatientContext) *types.Interpretation {
	// Get deviation percent (default to 0 if nil)
	deviationPercent := float64(0)
	if rangeInterp.DeviationPercent != nil {
		deviationPercent = *rangeInterp.DeviationPercent
	}

	return &types.Interpretation{
		Flag:               types.InterpretationFlag(rangeInterp.Flag),
		Severity:           e.determineSeverityFromFlag(rangeInterp.Flag),
		IsCritical:         e.isCriticalFlag(rangeInterp.Flag),
		IsPanic:            e.isPanicFlag(rangeInterp.Flag),
		RequiresAction:     e.requiresAction(rangeInterp.Flag),
		DeviationPercent:   deviationPercent,
		DeviationDirection: rangeInterp.DeviationDirection,
	}
}

// generateEnhancedComment creates context-aware clinical comments
func (e *ContextualEngine) generateEnhancedComment(result *types.LabResult, rangeInterp *reference.RangeInterpretation, selectedRange *reference.ConditionalReferenceRange, patientCtx *types.PatientContext) string {
	var parts []string

	// Add range context note
	if rangeInterp.InterpretationNote != "" {
		parts = append(parts, rangeInterp.InterpretationNote)
	}

	// Add context description
	contextDesc := rangeInterp.ContextApplied
	if contextDesc != "" && contextDesc != "Standard" {
		parts = append(parts, fmt.Sprintf("Range adjusted for %s (Authority: %s).", contextDesc, selectedRange.Authority))
	}

	// Add specific guidance based on flag
	if rangeInterp.Flag == reference.FlagCriticalHigh || rangeInterp.Flag == reference.FlagCriticalLow {
		parts = append(parts, "Critical value - notify provider immediately.")
	}
	if rangeInterp.Flag == reference.FlagPanicHigh || rangeInterp.Flag == reference.FlagPanicLow {
		parts = append(parts, "PANIC VALUE - immediate clinical action required.")
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Result within %s reference range.", contextDesc)
	}

	return strings.Join(parts, " ")
}

// generateContextualRecommendations creates context-aware recommendations
func (e *ContextualEngine) generateContextualRecommendations(result *types.LabResult, rangeInterp *reference.RangeInterpretation, selectedRange *reference.ConditionalReferenceRange, patientCtx *types.PatientContext) []types.Recommendation {
	var recommendations []types.Recommendation

	// Add clinical action if present
	if selectedRange.ClinicalAction != "" && rangeInterp.Flag != reference.FlagNormal {
		recommendations = append(recommendations, types.Recommendation{
			Type:        "clinical_action",
			Priority:    e.priorityFromFlag(rangeInterp.Flag),
			Description: selectedRange.ClinicalAction,
		})
	}

	// Add standard recommendations based on flag
	switch rangeInterp.Flag {
	case reference.FlagPanicLow, reference.FlagPanicHigh:
		recommendations = append(recommendations, types.Recommendation{
			Type:        "urgent",
			Priority:    "CRITICAL",
			Description: "Panic value - notify provider immediately and document notification.",
		})
	case reference.FlagCriticalLow, reference.FlagCriticalHigh:
		recommendations = append(recommendations, types.Recommendation{
			Type:        "notify",
			Priority:    "HIGH",
			Description: "Critical value - provider notification required within 1 hour.",
		})
	case reference.FlagLow, reference.FlagHigh:
		recommendations = append(recommendations, types.Recommendation{
			Type:        "follow_up",
			Priority:    "MEDIUM",
			Description: "Abnormal result - consider clinical correlation and repeat testing if indicated.",
		})
	}

	return recommendations
}

// Helper methods
func (e *ContextualEngine) isNeonatalBilirubinTest(code string) bool {
	return code == "1975-2" || code == "58941-6" // Total bilirubin LOINC codes
}

func (e *ContextualEngine) determineSeverityFromFlag(flag reference.InterpretationFlag) types.Severity {
	switch flag {
	case reference.FlagPanicLow, reference.FlagPanicHigh:
		return types.SeverityCritical
	case reference.FlagCriticalLow, reference.FlagCriticalHigh:
		return types.SeverityHigh
	case reference.FlagLow, reference.FlagHigh:
		return types.SeverityMedium
	default:
		return types.SeverityLow
	}
}

func (e *ContextualEngine) isCriticalFlag(flag reference.InterpretationFlag) bool {
	return flag == reference.FlagCriticalLow || flag == reference.FlagCriticalHigh ||
		flag == reference.FlagPanicLow || flag == reference.FlagPanicHigh
}

func (e *ContextualEngine) isPanicFlag(flag reference.InterpretationFlag) bool {
	return flag == reference.FlagPanicLow || flag == reference.FlagPanicHigh
}

func (e *ContextualEngine) requiresAction(flag reference.InterpretationFlag) bool {
	return flag != reference.FlagNormal
}

func (e *ContextualEngine) priorityFromFlag(flag reference.InterpretationFlag) string {
	switch flag {
	case reference.FlagPanicLow, reference.FlagPanicHigh:
		return "CRITICAL"
	case reference.FlagCriticalLow, reference.FlagCriticalHigh:
		return "HIGH"
	case reference.FlagLow, reference.FlagHigh:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func (e *ContextualEngine) bilirubinToFlag(interp *reference.BilirubinInterpretation) types.InterpretationFlag {
	switch {
	case interp.NeedsExchange:
		return types.FlagPanicHigh
	case interp.NeedsPhototherapy:
		return types.FlagCriticalHigh
	case interp.Zone == "HIGH_RISK":
		return types.FlagHigh
	case interp.Zone == "HIGH_INTERMEDIATE":
		return types.FlagHigh
	default:
		return types.FlagNormal
	}
}

func (e *ContextualEngine) bilirubinPriority(interp *reference.BilirubinInterpretation) string {
	switch {
	case interp.NeedsExchange:
		return "CRITICAL"
	case interp.NeedsPhototherapy:
		return "HIGH"
	case interp.Zone == "HIGH_RISK" || interp.Zone == "HIGH_INTERMEDIATE":
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// ContextualInterpretedResult extends InterpretedResult with context metadata
type ContextualInterpretedResult struct {
	types.InterpretedResult

	// Context Information
	ContextApplied       string `json:"context_applied"`        // e.g., "Pregnancy T3", "CKD Stage 4"
	SpecificityScore     int    `json:"specificity_score"`      // Higher = more specific range selected
	RangeSourceAuthority string `json:"range_source_authority"` // e.g., "ACOG", "KDIGO", "AAP"
	RangeSourceRef       string `json:"range_source_ref"`       // Specific guideline reference
	RangeID              string `json:"range_id,omitempty"`     // UUID of selected range

	// Selected range values (for transparency)
	SelectedLowNormal  *float64 `json:"selected_low_normal,omitempty"`
	SelectedHighNormal *float64 `json:"selected_high_normal,omitempty"`

	// Neonatal bilirubin specific (if applicable)
	BilirubinInterpretation *reference.BilirubinInterpretation `json:"bilirubin_interpretation,omitempty"`
}
