package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestPAI_RajeshKumar_HighAcuity(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		PatientID:               "rajesh-kumar-001",
		MHRICompositeSlope:      floatPtr(-1.4),
		GlucoseDomainSlope:      floatPtr(-1.8),
		CardioDomainSlope:       floatPtr(-1.0),
		ConcordantDeterioration: true,
		DomainsDeterioriating:   3,
		CurrentEGFR:             floatPtr(42),
		CurrentHbA1c:            floatPtr(8.2),
		CurrentSBP:              floatPtr(170),
		EngagementComposite:     floatPtr(0.35),
		EngagementStatus:        "DECLINING",
		DaysSinceLastBPReading:  3,
		AvgReadingsPerWeek:      5,
		CurrentReadingsPerWeek:  2,
		MeasurementFreqDrop:     0.60,
		CKMStage:                "2",
		Age:                     58,
		MedicationCount:         4,
		DaysSinceLastClinician:  35,
		DaysSinceLastCardAck:    14,
		UnacknowledgedCardCount: 3,
		HasUnacknowledgedCards:  true,
	}

	result := ComputePAI(input, cfg)

	if result.Score < 65 {
		t.Errorf("expected Score >= 65, got %.2f", result.Score)
	}
	if result.Tier != "HIGH" {
		t.Errorf("expected Tier HIGH, got %s", result.Tier)
	}
	if result.VelocityScore < 60 {
		t.Errorf("expected VelocityScore >= 60, got %.2f", result.VelocityScore)
	}
	if result.ProximityScore < 30 {
		t.Errorf("expected ProximityScore >= 30, got %.2f", result.ProximityScore)
	}
	if result.BehavioralScore < 40 {
		t.Errorf("expected BehavioralScore >= 40, got %.2f", result.BehavioralScore)
	}
	if result.PrimaryReason == "" {
		t.Error("expected PrimaryReason not empty")
	}
	if result.SuggestedAction == "" {
		t.Error("expected SuggestedAction not empty")
	}
	if result.PatientID != "rajesh-kumar-001" {
		t.Errorf("expected PatientID rajesh-kumar-001, got %s", result.PatientID)
	}
}

func TestPAI_StableWellManaged_LowAcuity(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		PatientID:              "stable-patient-001",
		MHRICompositeSlope:     floatPtr(0.1),
		CurrentEGFR:            floatPtr(72),
		CurrentHbA1c:           floatPtr(6.8),
		CurrentSBP:             floatPtr(128),
		EngagementComposite:    floatPtr(0.85),
		EngagementStatus:       "ACTIVE",
		AvgReadingsPerWeek:     7,
		CurrentReadingsPerWeek: 6,
		CKMStage:               "2",
		DaysSinceLastClinician: 12,
		DaysSinceLastCardAck:   3,
	}

	result := ComputePAI(input, cfg)

	if result.Score >= 25 {
		t.Errorf("expected Score < 25, got %.2f", result.Score)
	}
	if result.Tier != "LOW" && result.Tier != "MINIMAL" {
		t.Errorf("expected Tier LOW or MINIMAL, got %s", result.Tier)
	}
}

func TestPAI_AcuteOnChronic_CriticalAcuity(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		PatientID:              "acute-chronic-001",
		MHRICompositeSlope:     floatPtr(-2.8),
		SecondDerivative:       paiStringPtr("ACCELERATING_DECLINE"),
		CurrentEGFR:            floatPtr(28),
		CurrentSBP:             floatPtr(165),
		CurrentWeight:          floatPtr(92.5),
		PreviousWeight72h:      floatPtr(90.0),
		CKMStage:               "4c",
		HFType:                 "HFrEF",
		NYHAClass:              "III",
		IsPostDischarge30d:     true,
		DaysSinceDischarge:     paiIntPtr(10),
		MedicationCount:        8,
		Age:                    72,
		DaysSinceLastClinician:  45,
		EngagementComposite:    floatPtr(0.25),
		EngagementStatus:       "DECLINING",
		MeasurementFreqDrop:    0.70,
		HasUnacknowledgedCards: true,
		UnacknowledgedCardCount: 4,
	}

	result := ComputePAI(input, cfg)

	if result.Score < 85 {
		t.Errorf("expected Score >= 85, got %.2f", result.Score)
	}
	if result.Tier != "CRITICAL" {
		t.Errorf("expected Tier CRITICAL, got %s", result.Tier)
	}
	if result.EscalationTier != "SAFETY" {
		t.Errorf("expected EscalationTier SAFETY, got %s", result.EscalationTier)
	}
}

func TestPAI_DominantDimension_Identified(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		PatientID:               "dominant-velocity-001",
		MHRICompositeSlope:      floatPtr(-2.5),
		ConcordantDeterioration: true,
		DomainsDeterioriating:   3,
		CurrentEGFR:             floatPtr(65),
		EngagementComposite:     floatPtr(0.80),
		CKMStage:                "2",
		DaysSinceLastClinician:  7,
	}

	result := ComputePAI(input, cfg)

	if result.DominantDimension != "VELOCITY" {
		t.Errorf("expected DominantDimension VELOCITY, got %s", result.DominantDimension)
	}
	if result.DominantContribution < 50 {
		t.Errorf("expected DominantContribution >= 50%%, got %.2f%%", result.DominantContribution)
	}
}

func TestPAI_NoData_MinimalAcuity(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		PatientID: "no-data-001",
		CKMStage:  "0",
	}

	result := ComputePAI(input, cfg)

	if result.Score >= 15 {
		t.Errorf("expected Score < 15, got %.2f", result.Score)
	}
	if result.DataFreshness != "STALE" {
		t.Errorf("expected DataFreshness STALE, got %s", result.DataFreshness)
	}
}

func TestPAI_SignificantChange_Detected(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		PatientID:               "change-detect-001",
		MHRICompositeSlope:      floatPtr(-1.5),
		ConcordantDeterioration: true,
		DomainsDeterioriating:   3,
		CurrentEGFR:             floatPtr(40),
		CurrentHbA1c:            floatPtr(8.5),
		CurrentSBP:              floatPtr(165),
		EngagementComposite:     floatPtr(0.45),
		EngagementStatus:        "DECLINING",
		MeasurementFreqDrop:     0.55,
		CKMStage:                "3",
		DaysSinceLastClinician:  40,
		HasUnacknowledgedCards:  true,
		UnacknowledgedCardCount: 2,
	}

	current := ComputePAI(input, cfg)

	// Simulate a previous score of 30 (tier LOW) — use the actual
	// PAIEventTrigger.ProcessResult path instead of manual wiring.
	previous := models.PAIScore{
		PatientID: "change-detect-001",
		Score:     30.0,
		Tier:      string(models.TierLow),
	}

	trigger := NewPAIEventTrigger(15, cfg.SignificantDelta)
	event := trigger.ProcessResult(current, previous)

	if event == nil {
		t.Fatal("expected significant change event, got nil")
	}
	if event.NewScore != current.Score {
		t.Errorf("expected NewScore %.2f, got %.2f", current.Score, event.NewScore)
	}
	if event.PreviousScore != 30.0 {
		t.Errorf("expected PreviousScore 30.0, got %.2f", event.PreviousScore)
	}
	if current.Score-30.0 < 10 {
		t.Errorf("expected score delta >= 10, got %.2f", current.Score-30.0)
	}
}
