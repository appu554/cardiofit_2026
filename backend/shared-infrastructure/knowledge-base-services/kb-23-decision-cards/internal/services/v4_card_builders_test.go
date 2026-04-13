package services

import (
	"testing"

	"kb-23-decision-cards/internal/models"
)

// TestBuildTrajectoryDecisionCard_Immediate verifies that IMMEDIATE urgency
// maps to the strictest MCU gate (GateHalt) and SafetyImmediate tier.
func TestBuildTrajectoryDecisionCard_Immediate(t *testing.T) {
	card := TrajectoryCard{
		CardType:  "CONCORDANT_DETERIORATION",
		Urgency:   "IMMEDIATE",
		Title:     "Multi-Domain Deterioration",
		Rationale: "Three domains declining simultaneously.",
		Actions:   []string{"Review all medications"},
	}
	patientID := "00000000-0000-0000-0000-000000000001"

	dc := BuildTrajectoryDecisionCard(card, patientID)

	if dc.MCUGate != models.GateHalt {
		t.Errorf("expected MCUGate %q for IMMEDIATE urgency, got %q", models.GateHalt, dc.MCUGate)
	}
	if dc.SafetyTier != models.SafetyImmediate {
		t.Errorf("expected SafetyTier %q for IMMEDIATE urgency, got %q", models.SafetyImmediate, dc.SafetyTier)
	}
	if dc.Status != models.StatusActive {
		t.Errorf("expected Status %q, got %q", models.StatusActive, dc.Status)
	}
	if dc.CardSource != models.SourceClinicalSignal {
		t.Errorf("expected CardSource %q, got %q", models.SourceClinicalSignal, dc.CardSource)
	}
}

// TestBuildTrajectoryDecisionCard_Urgent verifies that URGENT urgency maps to
// GatePause and SafetyUrgent tier.
func TestBuildTrajectoryDecisionCard_Urgent(t *testing.T) {
	card := TrajectoryCard{
		CardType:  "DOMAIN_DIVERGENCE",
		Urgency:   "URGENT",
		Title:     "Discordant Trajectory",
		Rationale: "Cardio improving while glucose declines.",
		Actions:   []string{"Review medication cross-domain benefit"},
	}
	patientID := "00000000-0000-0000-0000-000000000002"

	dc := BuildTrajectoryDecisionCard(card, patientID)

	if dc.MCUGate != models.GatePause {
		t.Errorf("expected MCUGate %q for URGENT urgency, got %q", models.GatePause, dc.MCUGate)
	}
	if dc.SafetyTier != models.SafetyUrgent {
		t.Errorf("expected SafetyTier %q for URGENT urgency, got %q", models.SafetyUrgent, dc.SafetyTier)
	}
}

// TestBuildMaskedHTNDecisionCard_Basic verifies that PatientID, Status, and
// CardSource are set correctly for a basic masked HTN card.
func TestBuildMaskedHTNDecisionCard_Basic(t *testing.T) {
	card := MaskedHTNCard{
		CardType:       "MASKED_HYPERTENSION",
		Urgency:        "URGENT",
		Title:          "Masked Hypertension Detected",
		Rationale:      "Home BP elevated, clinic normal.",
		Actions:        []string{"Do not rely on clinic BP alone"},
		ConfidenceTier: models.TierFirm,
	}
	patientID := "00000000-0000-0000-0000-000000000003"

	dc := BuildMaskedHTNDecisionCard(card, patientID)

	if dc.Status != models.StatusActive {
		t.Errorf("expected Status %q, got %q", models.StatusActive, dc.Status)
	}
	if dc.CardSource != models.SourceClinicalSignal {
		t.Errorf("expected CardSource %q, got %q", models.SourceClinicalSignal, dc.CardSource)
	}
	if dc.DiagnosticConfidenceTier != models.TierFirm {
		t.Errorf("expected ConfidenceTier %q, got %q", models.TierFirm, dc.DiagnosticConfidenceTier)
	}
	if dc.PatientID.String() != patientID {
		t.Errorf("expected PatientID %q, got %q", patientID, dc.PatientID.String())
	}
}

// TestBuildTrajectoryDecisionCard_PreservesTitleAndRationale verifies that the
// Title and Rationale from the local TrajectoryCard flow into ClinicianSummary
// and PatientSummaryEn on the persistent DecisionCard.
func TestBuildTrajectoryDecisionCard_PreservesTitleAndRationale(t *testing.T) {
	title := "Engagement Collapse Preceding Clinical Deterioration"
	rationale := "Behavioral domain declining before Cardio and Glucose."

	card := TrajectoryCard{
		CardType:  "BEHAVIORAL_LEADING_INDICATOR",
		Urgency:   "URGENT",
		Title:     title,
		Rationale: rationale,
		Actions:   []string{"Clinical outreach"},
	}
	patientID := "00000000-0000-0000-0000-000000000004"

	dc := BuildTrajectoryDecisionCard(card, patientID)

	if dc.PatientSummaryEn != title {
		t.Errorf("expected PatientSummaryEn %q, got %q", title, dc.PatientSummaryEn)
	}
	expectedSummary := title + " — " + rationale
	if dc.ClinicianSummary != expectedSummary {
		t.Errorf("expected ClinicianSummary %q, got %q", expectedSummary, dc.ClinicianSummary)
	}
	if dc.MCUGateRationale != rationale {
		t.Errorf("expected MCUGateRationale %q, got %q", rationale, dc.MCUGateRationale)
	}
}
