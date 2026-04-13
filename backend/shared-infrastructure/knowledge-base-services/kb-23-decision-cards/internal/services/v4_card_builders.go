package services

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"kb-23-decision-cards/internal/models"
)

// urgencyToMCUGate maps the V4 local card urgency strings to the MCU gate
// values that control V-MCU insulin titration.
//
// Mapping rationale (most → least restrictive):
//   IMMEDIATE → GateHalt   — multi-domain or masked HTN compound risk requires
//                            immediate clinical halt of automated titration.
//   URGENT    → GatePause  — single-domain or moderate phenotype warrants a
//                            pause pending clinician review.
//   ROUTINE   → GateModify — low-urgency phenotypes allow titration to
//                            continue with modified parameters.
//   (default) → GateSafe   — unknown urgency should not restrict the gate.
func urgencyToMCUGate(urgency string) models.MCUGate {
	switch strings.ToUpper(urgency) {
	case "IMMEDIATE":
		return models.GateHalt
	case "URGENT":
		return models.GatePause
	case "ROUTINE":
		return models.GateModify
	default:
		return models.GateSafe
	}
}

// urgencyToSafetyTier maps urgency strings to SafetyTier, mirroring the same
// strictness ordering used in urgencyToMCUGate.
func urgencyToSafetyTier(urgency string) models.SafetyTier {
	switch strings.ToUpper(urgency) {
	case "IMMEDIATE":
		return models.SafetyImmediate
	case "URGENT":
		return models.SafetyUrgent
	default:
		return models.SafetyRoutine
	}
}

// BuildTrajectoryDecisionCard converts a TrajectoryCard produced by
// EvaluateTrajectoryCards into a persistent DecisionCard row.
//
// NodeID is not derivable from the local trajectory card (it normally
// identifies a KB-22 Bayesian node). We use the CardType string as a
// stable synthetic node ID so the field satisfies the not-null constraint
// without fabricating clinical meaning.
func BuildTrajectoryDecisionCard(card TrajectoryCard, patientID string) *models.DecisionCard {
	pid, err := uuid.Parse(patientID)
	if err != nil {
		// Fallback: use nil UUID; BeforeCreate will still assign a card ID.
		pid = uuid.Nil
	}

	gate := urgencyToMCUGate(card.Urgency)
	tier := urgencyToSafetyTier(card.Urgency)

	// TemplateID follows the same dc-<type>-v1 convention used by masked HTN cards.
	templateID := "dc-trajectory-" + strings.ToLower(strings.ReplaceAll(card.CardType, "_", "-")) + "-v1"
	// NodeID: use CardType as a synthetic node identifier (see function doc).
	nodeID := card.CardType

	return &models.DecisionCard{
		PatientID:                pid,
		TemplateID:               templateID,
		NodeID:                   nodeID,
		DiagnosticConfidenceTier: models.TierProbable,    // trajectory analysis is model-derived, not Bayesian posterior
		MCUGate:                  gate,
		MCUGateRationale:         card.Rationale,
		SafetyTier:               tier,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         card.Title + " — " + card.Rationale,
		PatientSummaryEn:         card.Title,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
}

// BuildMaskedHTNDecisionCard converts a MaskedHTNCard produced by
// EvaluateMaskedHTNCards into a persistent DecisionCard row.
//
// NodeID: same synthetic approach as BuildTrajectoryDecisionCard — uses
// CardType so the not-null DB constraint is satisfied without inventing a
// clinical node reference.
func BuildMaskedHTNDecisionCard(card MaskedHTNCard, patientID string) *models.DecisionCard {
	pid, err := uuid.Parse(patientID)
	if err != nil {
		pid = uuid.Nil
	}

	gate := urgencyToMCUGate(card.Urgency)
	tier := urgencyToSafetyTier(card.Urgency)

	templateID := "dc-masked-htn-" + strings.ToLower(strings.ReplaceAll(card.CardType, "_", "-")) + "-v1"
	nodeID := card.CardType

	return &models.DecisionCard{
		PatientID:                pid,
		TemplateID:               templateID,
		NodeID:                   nodeID,
		DiagnosticConfidenceTier: card.ConfidenceTier,
		MCUGate:                  gate,
		MCUGateRationale:         card.Rationale,
		SafetyTier:               tier,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         card.Title + " — " + card.Rationale,
		PatientSummaryEn:         card.Title,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
}
