// Package engines provides clinical decision engines that consume
// the frozen ClinicalExecutionContext contract.
//
// ENGINE CONTRACT:
// 1. Engines receive ClinicalExecutionContext - they NEVER call KBs directly
// 2. Engines are STATELESS - all data comes from context
// 3. Engines return EngineResult - recommendations, alerts, measures
// 4. Engines are DETERMINISTIC - same context = same result
//
// This is the reference implementation for the Medication Advisor engine.
package engines

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"vaidshala/clinical-runtime-platform/contracts"
)

// MedicationEngine provides medication-related clinical decision support.
// It uses ONLY the ClinicalExecutionContext - no external KB calls.
type MedicationEngine struct {
	// config for engine behavior
	config MedicationEngineConfig
}

// MedicationEngineConfig configures the engine.
type MedicationEngineConfig struct {
	// CheckInteractions enable DDI checking
	CheckInteractions bool

	// CheckContraindications enable drug-condition checking
	CheckContraindications bool

	// CheckDosing enable dose adjustment recommendations
	CheckDosing bool

	// CheckFormulary enable formulary status recommendations
	CheckFormulary bool

	// Region for regional rules
	Region string
}

// DefaultMedicationEngineConfig returns sensible defaults.
func DefaultMedicationEngineConfig() MedicationEngineConfig {
	return MedicationEngineConfig{
		CheckInteractions:      true,
		CheckContraindications: true,
		CheckDosing:            true,
		CheckFormulary:         true,
		Region:                 "AU",
	}
}

// NewMedicationEngine creates a new medication engine.
func NewMedicationEngine(config MedicationEngineConfig) *MedicationEngine {
	return &MedicationEngine{config: config}
}

// Name returns the engine identifier.
func (e *MedicationEngine) Name() string {
	return "medication-advisor"
}

// Evaluate processes the context and returns medication recommendations.
//
// CRITICAL: This engine makes NO external calls.
// All data comes from ClinicalExecutionContext.
func (e *MedicationEngine) Evaluate(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
) (*contracts.EngineResult, error) {

	startTime := time.Now()

	result := &contracts.EngineResult{
		EngineName:      e.Name(),
		Success:         true,
		Recommendations: make([]contracts.Recommendation, 0),
		Alerts:          make([]contracts.Alert, 0),
		EvidenceLinks:   make([]string, 0),
	}

	// ========================================================================
	// CHECK 1: Drug-Drug Interactions (from Interactions snapshot - KB-5)
	// ========================================================================
	if e.config.CheckInteractions {
		e.checkInteractions(execCtx, result)
	}

	// ========================================================================
	// CHECK 2: Drug-Condition Contraindications (from Safety snapshot - KB-4)
	// ========================================================================
	if e.config.CheckContraindications {
		e.checkContraindications(execCtx, result)
	}

	// ========================================================================
	// CHECK 3: Dose Adjustments (from Dosing snapshot - KB-1)
	// ========================================================================
	if e.config.CheckDosing {
		e.checkDosing(execCtx, result)
	}

	// ========================================================================
	// CHECK 4: Formulary Status (from Formulary snapshot - KB-6)
	// ========================================================================
	if e.config.CheckFormulary {
		e.checkFormulary(execCtx, result)
	}

	// ========================================================================
	// CHECK 5: Safety Alerts (from Safety snapshot - KB-4)
	// ========================================================================
	e.checkSafetyAlerts(execCtx, result)

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// checkInteractions processes DDIs from the Interactions snapshot (KB-5).
func (e *MedicationEngine) checkInteractions(
	execCtx *contracts.ClinicalExecutionContext,
	result *contracts.EngineResult,
) {
	// Get current DDIs from Interactions snapshot (per CTO/CMO spec)
	interactions := execCtx.Knowledge.Interactions.CurrentDDIs

	for _, ddi := range interactions {
		severity := mapSeverityToAlertSeverity(ddi.Severity)

		// Create alert for severe/critical interactions
		if ddi.Severity == "severe" || ddi.Severity == "critical" {
			result.Alerts = append(result.Alerts, contracts.Alert{
				ID:          fmt.Sprintf("DDI-%s", uuid.New().String()[:8]),
				Severity:    severity,
				Category:    "medication-safety",
				Title:       fmt.Sprintf("Drug Interaction: %s + %s", ddi.Drug1.Display, ddi.Drug2.Display),
				Description: ddi.Description,
				CreatedAt:   time.Now(),
			})
		}

		// Create recommendation for all interactions
		result.Recommendations = append(result.Recommendations, contracts.Recommendation{
			ID:          fmt.Sprintf("REC-DDI-%s", uuid.New().String()[:8]),
			Type:        "medication-review",
			Title:       fmt.Sprintf("Review interaction: %s + %s", ddi.Drug1.Display, ddi.Drug2.Display),
			Description: ddi.Description,
			Priority:    mapSeverityToPriority(ddi.Severity),
			Source:      "medication-advisor/ddi-checker",
			Actions: []contracts.SuggestedAction{
				{
					Type:        "review",
					Description: ddi.Recommendation,
				},
			},
		})
	}

	// Also check if there are critical interactions flagged
	if execCtx.Knowledge.Interactions.HasCriticalInteraction {
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          fmt.Sprintf("DDI-CRIT-%s", uuid.New().String()[:8]),
			Severity:    "critical",
			Category:    "medication-safety",
			Title:       "Critical Drug Interaction Present",
			Description: fmt.Sprintf("Patient has critical drug interaction(s). Maximum severity: %s", execCtx.Knowledge.Interactions.SeverityMax),
			CreatedAt:   time.Now(),
		})
	}
}

// checkContraindications processes drug-condition contraindications from Safety snapshot (KB-4).
func (e *MedicationEngine) checkContraindications(
	execCtx *contracts.ClinicalExecutionContext,
	result *contracts.EngineResult,
) {
	// Get contraindications from Safety snapshot (per CTO/CMO spec)
	contraindications := execCtx.Knowledge.Safety.Contraindications

	for _, ci := range contraindications {
		// All contraindications generate alerts
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          fmt.Sprintf("CI-%s", uuid.New().String()[:8]),
			Severity:    mapSeverityToAlertSeverity(ci.Severity),
			Category:    "medication-safety",
			Title:       fmt.Sprintf("Contraindicated: %s with %s", ci.Medication.Display, ci.Condition.Display),
			Description: ci.Description,
			CreatedAt:   time.Now(),
		})

		// Recommendation to review/discontinue
		result.Recommendations = append(result.Recommendations, contracts.Recommendation{
			ID:          fmt.Sprintf("REC-CI-%s", uuid.New().String()[:8]),
			Type:        "medication-discontinuation",
			Title:       fmt.Sprintf("Consider discontinuing %s", ci.Medication.Display),
			Description: fmt.Sprintf("Contraindicated due to patient condition: %s. %s", ci.Condition.Display, ci.Recommendation),
			Priority:    "high",
			Source:      "medication-advisor/contraindication-checker",
		})
	}
}

// checkDosing processes dose adjustment recommendations from Dosing snapshot (KB-1).
func (e *MedicationEngine) checkDosing(
	execCtx *contracts.ClinicalExecutionContext,
	result *contracts.EngineResult,
) {
	// Check renal dose adjustments (per CTO/CMO spec)
	for medKey, adj := range execCtx.Knowledge.Dosing.RenalAdjustments {
		result.Recommendations = append(result.Recommendations, contracts.Recommendation{
			ID:          fmt.Sprintf("REC-RENAL-%s", uuid.New().String()[:8]),
			Type:        "dose-adjustment",
			Title:       fmt.Sprintf("Renal dose adjustment needed: %s", adj.Medication.Display),
			Description: adj.Guidance,
			Priority:    "medium",
			Source:      fmt.Sprintf("medication-advisor/renal-dosing (eGFR threshold: %.0f)", adj.ThresholdEGFR),
			Actions: []contracts.SuggestedAction{
				{
					Type:        "modify-order",
					Description: fmt.Sprintf("Adjust dose by %.0f%% for medication %s", adj.AdjustmentPercent, medKey),
				},
			},
		})
	}

	// Check hepatic dose adjustments
	for medKey, adj := range execCtx.Knowledge.Dosing.HepaticAdjustments {
		result.Recommendations = append(result.Recommendations, contracts.Recommendation{
			ID:          fmt.Sprintf("REC-HEPATIC-%s", uuid.New().String()[:8]),
			Type:        "dose-adjustment",
			Title:       fmt.Sprintf("Hepatic dose adjustment needed: %s", adj.Medication.Display),
			Description: adj.Guidance,
			Priority:    "medium",
			Source:      "medication-advisor/hepatic-dosing",
			Actions: []contracts.SuggestedAction{
				{
					Type:        "modify-order",
					Description: fmt.Sprintf("Adjust dose by %.0f%% for medication %s", adj.AdjustmentPercent, medKey),
				},
			},
		})
	}

	// Check age-based adjustments (pediatric/geriatric)
	for medKey, adj := range execCtx.Knowledge.Dosing.AgeBasedAdjustments {
		result.Recommendations = append(result.Recommendations, contracts.Recommendation{
			ID:          fmt.Sprintf("REC-AGE-%s", uuid.New().String()[:8]),
			Type:        "dose-adjustment",
			Title:       fmt.Sprintf("Age-based dose adjustment needed: %s", adj.Medication.Display),
			Description: adj.Guidance,
			Priority:    "medium",
			Source:      fmt.Sprintf("medication-advisor/age-dosing (%s)", adj.Reason),
			Actions: []contracts.SuggestedAction{
				{
					Type:        "modify-order",
					Description: fmt.Sprintf("Adjust dose for medication %s", medKey),
				},
			},
		})
	}

	// Add alert if renal or hepatic adjustment needed (from Safety snapshot flags)
	if execCtx.Knowledge.Safety.RenalDoseAdjustmentNeeded {
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          fmt.Sprintf("RENAL-ADJ-%s", uuid.New().String()[:8]),
			Severity:    "moderate",
			Category:    "dosing",
			Title:       "Renal Dose Adjustments Required",
			Description: "Patient has reduced kidney function. Review all renally-cleared medications for dose adjustment.",
			CreatedAt:   time.Now(),
		})
	}

	if execCtx.Knowledge.Safety.HepaticDoseAdjustmentNeeded {
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          fmt.Sprintf("HEPATIC-ADJ-%s", uuid.New().String()[:8]),
			Severity:    "moderate",
			Category:    "dosing",
			Title:       "Hepatic Dose Adjustments Required",
			Description: "Patient has reduced liver function. Review all hepatically-metabolized medications for dose adjustment.",
			CreatedAt:   time.Now(),
		})
	}
}

// checkFormulary processes formulary status from Formulary snapshot (KB-6).
func (e *MedicationEngine) checkFormulary(
	execCtx *contracts.ClinicalExecutionContext,
	result *contracts.EngineResult,
) {
	// Get formulary data (per CTO/CMO spec)
	formulary := execCtx.Knowledge.Formulary.MedicationStatus
	alternatives := execCtx.Knowledge.Formulary.GenericAlternatives
	priorAuth := execCtx.Knowledge.Formulary.PriorAuthRequired

	// Check each patient medication
	for _, med := range execCtx.Patient.ActiveMedications {
		key := fmt.Sprintf("%s|%s", med.Code.System, med.Code.Code)
		entry, exists := formulary[key]

		if !exists {
			continue
		}

		// Non-preferred medications
		if entry.Status == "non-preferred" || entry.Status == "excluded" {
			alts := alternatives[key]
			altNames := make([]string, 0, len(alts))
			for _, alt := range alts {
				altNames = append(altNames, alt.Display)
			}

			result.Recommendations = append(result.Recommendations, contracts.Recommendation{
				ID:          fmt.Sprintf("REC-FORM-%s", uuid.New().String()[:8]),
				Type:        "formulary-substitution",
				Title:       fmt.Sprintf("Consider formulary alternative for %s", med.Code.Display),
				Description: fmt.Sprintf("Current status: %s on %s. Alternatives: %v", entry.Status, entry.FormularyName, altNames),
				Priority:    "low",
				Source:      "medication-advisor/formulary-checker",
			})
		}
	}

	// Check prior authorization requirements
	if len(priorAuth) > 0 {
		for _, med := range priorAuth {
			result.Alerts = append(result.Alerts, contracts.Alert{
				ID:          fmt.Sprintf("PA-%s", uuid.New().String()[:8]),
				Severity:    "info",
				Category:    "administrative",
				Title:       fmt.Sprintf("Prior Authorization Required: %s", med.Display),
				Description: fmt.Sprintf("Medication %s requires prior authorization before dispensing.", med.Display),
				CreatedAt:   time.Now(),
			})
		}
	}

	// Check regional availability (NLEM for India, PBS for Australia)
	region := execCtx.Runtime.Region
	if region == "IN" {
		for _, med := range execCtx.Patient.ActiveMedications {
			key := fmt.Sprintf("%s|%s", med.Code.System, med.Code.Code)
			if onNLEM, exists := execCtx.Knowledge.Formulary.NLEMAvailability[key]; exists && !onNLEM {
				result.Recommendations = append(result.Recommendations, contracts.Recommendation{
					ID:          fmt.Sprintf("REC-NLEM-%s", uuid.New().String()[:8]),
					Type:        "formulary-substitution",
					Title:       fmt.Sprintf("Non-NLEM medication: %s", med.Code.Display),
					Description: "This medication is not on the National List of Essential Medicines (NLEM). Consider NLEM alternatives if available.",
					Priority:    "low",
					Source:      "medication-advisor/nlem-checker",
				})
			}
		}
	}

	if region == "AU" {
		for _, med := range execCtx.Patient.ActiveMedications {
			key := fmt.Sprintf("%s|%s", med.Code.System, med.Code.Code)
			if onPBS, exists := execCtx.Knowledge.Formulary.PBSAvailability[key]; exists && !onPBS {
				result.Recommendations = append(result.Recommendations, contracts.Recommendation{
					ID:          fmt.Sprintf("REC-PBS-%s", uuid.New().String()[:8]),
					Type:        "formulary-substitution",
					Title:       fmt.Sprintf("Non-PBS medication: %s", med.Code.Display),
					Description: "This medication is not on the Pharmaceutical Benefits Scheme (PBS). Patient may face higher out-of-pocket costs.",
					Priority:    "low",
					Source:      "medication-advisor/pbs-checker",
				})
			}
		}
	}
}

// checkSafetyAlerts processes pre-computed safety alerts from Safety snapshot (KB-4).
func (e *MedicationEngine) checkSafetyAlerts(
	execCtx *contracts.ClinicalExecutionContext,
	result *contracts.EngineResult,
) {
	// Copy safety alerts from snapshot
	for _, alert := range execCtx.Knowledge.Safety.SafetyAlerts {
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          alert.AlertID,
			Severity:    alert.Severity,
			Category:    "medication-safety",
			Title:       alert.Title,
			Description: alert.Description,
			CreatedAt:   alert.CreatedAt,
		})
	}

	// Check pregnancy status (per CTO/CMO spec)
	if execCtx.Knowledge.Safety.PregnancyStatus != nil && execCtx.Knowledge.Safety.PregnancyStatus.IsPregnant {
		pregnancy := execCtx.Knowledge.Safety.PregnancyStatus
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          fmt.Sprintf("PREG-%s", uuid.New().String()[:8]),
			Severity:    "high",
			Category:    "medication-safety",
			Title:       "Pregnancy Alert - Review All Medications",
			Description: fmt.Sprintf("Patient is pregnant (Trimester %d, ~%d weeks). Review all medications for pregnancy safety categories.", pregnancy.Trimester, pregnancy.EstimatedWeeks),
			CreatedAt:   time.Now(),
		})
	}

	// Check lactation status
	if execCtx.Knowledge.Safety.PregnancyStatus != nil && execCtx.Knowledge.Safety.PregnancyStatus.LactationStatus {
		result.Alerts = append(result.Alerts, contracts.Alert{
			ID:          fmt.Sprintf("LACT-%s", uuid.New().String()[:8]),
			Severity:    "moderate",
			Category:    "medication-safety",
			Title:       "Lactation Alert - Review Medications",
			Description: "Patient is breastfeeding. Review all medications for lactation safety.",
			CreatedAt:   time.Now(),
		})
	}

	// Check high-risk allergies
	for _, allergy := range execCtx.Knowledge.Safety.ActiveAllergies {
		if allergy.Criticality == "high" {
			result.Alerts = append(result.Alerts, contracts.Alert{
				ID:          fmt.Sprintf("ALLERGY-%s", uuid.New().String()[:8]),
				Severity:    "high",
				Category:    "allergy",
				Title:       fmt.Sprintf("High-Risk Allergy: %s", allergy.Allergen.Display),
				Description: fmt.Sprintf("Patient has high-criticality allergy to %s (%s). Reactions: %v", allergy.Allergen.Display, allergy.Category, allergy.Reactions),
				CreatedAt:   time.Now(),
			})
		}
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func mapSeverityToAlertSeverity(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "severe", "high":
		return "high"
	case "moderate", "medium":
		return "moderate"
	default:
		return "low"
	}
}

func mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical", "severe":
		return "high"
	case "moderate":
		return "medium"
	default:
		return "low"
	}
}
