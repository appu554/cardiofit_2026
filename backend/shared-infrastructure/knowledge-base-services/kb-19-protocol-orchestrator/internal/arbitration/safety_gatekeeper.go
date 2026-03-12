// Package arbitration implements the core protocol arbitration engine for KB-19.
package arbitration

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/models"
)

// SafetyGatekeeper applies global safety checks to decisions.
// Safety gates can block, warn, or modify decisions based on patient state.
// This integrates with ICU Intelligence, pregnancy safety, and other safety systems.
type SafetyGatekeeper struct {
	log *logrus.Entry
}

// NewSafetyGatekeeper creates a new SafetyGatekeeper.
func NewSafetyGatekeeper(log *logrus.Entry) *SafetyGatekeeper {
	return &SafetyGatekeeper{
		log: log.WithField("component", "safety-gatekeeper"),
	}
}

// Apply applies safety checks to all decisions.
func (sg *SafetyGatekeeper) Apply(decisions []models.ArbitratedDecision, patientCtx *models.PatientContext) ([]models.ArbitratedDecision, []models.SafetyGate) {
	var gates []models.SafetyGate

	// Apply ICU safety checks if patient is in ICU
	if patientCtx.IsICU() {
		icuGate := sg.applyICUSafety(decisions, patientCtx)
		gates = append(gates, icuGate)
	}

	// Apply pregnancy safety checks
	if patientCtx.PregnancyStatus != nil && patientCtx.PregnancyStatus.IsPregnant {
		pregnancyGate := sg.applyPregnancySafety(decisions, patientCtx)
		gates = append(gates, pregnancyGate)
	}

	// Apply renal safety checks
	if patientCtx.GetCQLFlag("HasAKI") || patientCtx.GetCalculatorScore("eGFR") < 30 {
		renalGate := sg.applyRenalSafety(decisions, patientCtx)
		gates = append(gates, renalGate)
	}

	// Apply bleeding risk checks
	if patientCtx.ICUStateSummary != nil && patientCtx.ICUStateSummary.BleedingRisk == "HIGH" {
		bleedingGate := sg.applyBleedingSafety(decisions, patientCtx)
		gates = append(gates, bleedingGate)
	}

	// Apply critical vital signs safety
	if patientCtx.HasCriticalVitals() {
		vitalGate := sg.applyCriticalVitalsSafety(decisions, patientCtx)
		gates = append(gates, vitalGate)
	}

	sg.log.WithField("gates_applied", len(gates)).Debug("Safety gatekeeper checks complete")

	return decisions, gates
}

// applyICUSafety applies ICU-specific safety rules.
func (sg *SafetyGatekeeper) applyICUSafety(decisions []models.ArbitratedDecision, patientCtx *models.PatientContext) models.SafetyGate {
	gate := models.SafetyGate{
		Name:              "ICU Safety Engine",
		Source:            "ICU_INTELLIGENCE",
		Triggered:         false,
		Result:            "PASS",
		AffectedDecisions: make([]uuid.UUID, 0),
	}

	icuState := patientCtx.ICUStateSummary

	for i := range decisions {
		// Check for severe shock state
		if icuState.ShockState == "UNCOMPENSATED" && decisions[i].DecisionType == models.DecisionDo {
			// Block anything that could worsen hemodynamics
			if isHemodynamicRisk(decisions[i].Target) {
				decisions[i].DecisionType = models.DecisionAvoid
				decisions[i].AddSafetyFlag(
					models.FlagICUHardBlock,
					"HARD_BLOCK",
					"Contraindicated in uncompensated shock",
					"ICU_INTELLIGENCE",
				)
				gate.Triggered = true
				gate.Result = "BLOCK"
				gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
			}
		}

		// Check for severe AKI
		if icuState.AKIStage >= 2 {
			if isNephrotoxic(decisions[i].Target) {
				decisions[i].AddSafetyFlag(
					models.FlagRenal,
					"WARNING",
					"Nephrotoxic in setting of AKI stage "+string(rune('0'+icuState.AKIStage)),
					"ICU_INTELLIGENCE",
				)
				gate.Triggered = true
				if gate.Result != "BLOCK" {
					gate.Result = "WARN"
				}
				gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
			}
		}

		// Check for severe coagulopathy
		if icuState.DICScore >= 5 || icuState.PlateletsLow {
			if isAnticoagulant(decisions[i].Target) {
				decisions[i].DecisionType = models.DecisionAvoid
				decisions[i].AddSafetyFlag(
					models.FlagBleeding,
					"HARD_BLOCK",
					"Contraindicated due to severe coagulopathy",
					"ICU_INTELLIGENCE",
				)
				gate.Triggered = true
				gate.Result = "BLOCK"
				gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
			}
		}
	}

	gate.Details = "ICU safety rules applied for hemodynamic, renal, and coagulation state"

	return gate
}

// applyPregnancySafety applies pregnancy-specific safety rules.
func (sg *SafetyGatekeeper) applyPregnancySafety(decisions []models.ArbitratedDecision, patientCtx *models.PatientContext) models.SafetyGate {
	gate := models.SafetyGate{
		Name:              "Pregnancy Safety",
		Source:            "PREGNANCY_CHECKER",
		Triggered:         false,
		Result:            "PASS",
		AffectedDecisions: make([]uuid.UUID, 0),
	}

	teratogenicDrugs := map[string]bool{
		"warfarin":     true,
		"methotrexate": true,
		"isotretinoin": true,
		"valproate":    true,
		"lithium":      true,
	}

	// Category X drugs - absolute contraindication
	categoryXDrugs := map[string]bool{
		"atorvastatin":  true,
		"simvastatin":   true,
		"rosuvastatin":  true,
		"isotretinoin":  true,
		"methotrexate":  true,
		"misoprostol":   true,
		"finasteride":   true,
	}

	// ACE inhibitors - contraindicated
	aceInhibitors := map[string]bool{
		"lisinopril":  true,
		"enalapril":   true,
		"ramipril":    true,
		"captopril":   true,
		"benazepril":  true,
		"fosinopril":  true,
		"quinapril":   true,
		"trandolapril": true,
	}

	for i := range decisions {
		target := decisions[i].Target

		if categoryXDrugs[target] || teratogenicDrugs[target] || aceInhibitors[target] {
			decisions[i].DecisionType = models.DecisionAvoid
			decisions[i].AddSafetyFlag(
				models.FlagPregnancy,
				"HARD_BLOCK",
				"Teratogenic - contraindicated in pregnancy",
				"PREGNANCY_CHECKER",
			)
			gate.Triggered = true
			gate.Result = "BLOCK"
			gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
		}
	}

	gate.Details = "Pregnancy safety check for teratogenic medications"

	return gate
}

// applyRenalSafety applies renal-specific safety rules.
func (sg *SafetyGatekeeper) applyRenalSafety(decisions []models.ArbitratedDecision, patientCtx *models.PatientContext) models.SafetyGate {
	gate := models.SafetyGate{
		Name:              "Renal Safety",
		Source:            "RENAL_DOSING",
		Triggered:         false,
		Result:            "PASS",
		AffectedDecisions: make([]uuid.UUID, 0),
	}

	nephrotoxicDrugs := map[string]bool{
		"gentamicin":    true,
		"tobramycin":    true,
		"amikacin":      true,
		"vancomycin":    true,
		"amphotericin":  true,
		"ibuprofen":     true,
		"ketorolac":     true,
		"naproxen":      true,
		"indomethacin":  true,
		"diclofenac":    true,
	}

	for i := range decisions {
		if nephrotoxicDrugs[decisions[i].Target] {
			decisions[i].AddSafetyFlag(
				models.FlagRenal,
				"CAUTION",
				"Nephrotoxic medication - requires dose adjustment or alternative",
				"RENAL_DOSING",
			)
			decisions[i].AddMonitoring(models.MonitoringItem{
				Parameter:    "Creatinine",
				Frequency:    "daily",
				AlertIfAbove: ptrFloat64(2.0),
			})
			gate.Triggered = true
			if gate.Result == "PASS" {
				gate.Result = "WARN"
			}
			gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
		}
	}

	gate.Details = "Renal safety check for nephrotoxic medications"

	return gate
}

// applyBleedingSafety applies bleeding risk safety rules.
func (sg *SafetyGatekeeper) applyBleedingSafety(decisions []models.ArbitratedDecision, patientCtx *models.PatientContext) models.SafetyGate {
	gate := models.SafetyGate{
		Name:              "Bleeding Risk",
		Source:            "BLEEDING_RISK",
		Triggered:         false,
		Result:            "PASS",
		AffectedDecisions: make([]uuid.UUID, 0),
	}

	anticoagulants := map[string]bool{
		"heparin":       true,
		"enoxaparin":    true,
		"warfarin":      true,
		"apixaban":      true,
		"rivaroxaban":   true,
		"dabigatran":    true,
		"edoxaban":      true,
	}

	antiplatelets := map[string]bool{
		"aspirin":       true,
		"clopidogrel":   true,
		"prasugrel":     true,
		"ticagrelor":    true,
	}

	for i := range decisions {
		if anticoagulants[decisions[i].Target] || antiplatelets[decisions[i].Target] {
			decisions[i].AddSafetyFlag(
				models.FlagBleeding,
				"WARNING",
				"High bleeding risk - consider benefit/risk ratio",
				"BLEEDING_RISK",
			)
			gate.Triggered = true
			if gate.Result == "PASS" {
				gate.Result = "WARN"
			}
			gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
		}
	}

	gate.Details = "Bleeding risk assessment for anticoagulants and antiplatelets"

	return gate
}

// applyCriticalVitalsSafety applies safety rules for critical vital signs.
func (sg *SafetyGatekeeper) applyCriticalVitalsSafety(decisions []models.ArbitratedDecision, patientCtx *models.PatientContext) models.SafetyGate {
	gate := models.SafetyGate{
		Name:              "Critical Vitals",
		Source:            "VITAL_SIGNS",
		Triggered:         true,
		Result:            "WARN",
		Details:           "Patient has critical vital signs - exercise caution with all interventions",
		AffectedDecisions: make([]uuid.UUID, 0),
	}

	// Add urgency escalation for all decisions
	for i := range decisions {
		if decisions[i].Urgency == models.UrgencyRoutine || decisions[i].Urgency == models.UrgencyScheduled {
			decisions[i].Urgency = models.UrgencyUrgent
		}
		gate.AffectedDecisions = append(gate.AffectedDecisions, decisions[i].ID)
	}

	return gate
}

// Helper functions

func isHemodynamicRisk(target string) bool {
	hemodynamicRiskDrugs := map[string]bool{
		"nitroprusside": true,
		"nitroglycerin": true,
		"hydralazine":   true,
		"propofol":      true,
	}
	return hemodynamicRiskDrugs[target]
}

func isNephrotoxic(target string) bool {
	nephrotoxicDrugs := map[string]bool{
		"gentamicin":   true,
		"tobramycin":   true,
		"vancomycin":   true,
		"amphotericin": true,
		"contrast":     true,
	}
	return nephrotoxicDrugs[target]
}

func isAnticoagulant(target string) bool {
	anticoagulants := map[string]bool{
		"heparin":     true,
		"enoxaparin":  true,
		"warfarin":    true,
		"apixaban":    true,
		"rivaroxaban": true,
		"dabigatran":  true,
	}
	return anticoagulants[target]
}

func ptrFloat64(f float64) *float64 {
	return &f
}
