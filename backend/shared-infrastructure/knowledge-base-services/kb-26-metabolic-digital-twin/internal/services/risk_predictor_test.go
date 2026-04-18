package services

import (
	"math"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func riskFloat(v float64) *float64 { return &v }

// ── Test 1: Declining trajectory → high contribution ────────────────────────

func TestRisk_DecliningTrajectory_HighContribution(t *testing.T) {
	slope := -1.2
	input := models.PredictedRiskInput{
		PatientID:         "p-traj",
		CompositeSlope30d: &slope,
	}
	risk := ComputePredictedRisk(input)

	// Expected contribution: min(abs(-1.2)/2.0 * 25, 25) = min(0.6*25, 25) = 15
	var trajFactor *models.RiskFactor
	for i := range risk.PrimaryDrivers {
		if risk.PrimaryDrivers[i].FactorName == "declining_trajectory" {
			trajFactor = &risk.PrimaryDrivers[i]
			break
		}
	}
	if trajFactor == nil {
		t.Fatal("expected declining_trajectory factor in primary drivers")
	}

	expectedContrib := math.Min(math.Abs(slope)/2.0*25.0, 25.0) // 15.0
	if math.Abs(trajFactor.Contribution-expectedContrib) > 0.01 {
		t.Fatalf("trajectory contribution = %.2f; want ~%.2f", trajFactor.Contribution, expectedContrib)
	}
	if trajFactor.Interpretation != "Clinical trajectory has been declining" {
		t.Fatalf("unexpected interpretation: %s", trajFactor.Interpretation)
	}
}

// ── Test 2: Rising PAI trend → contribution ─────────────────────────────────

func TestRisk_RisingPAITrend_Contribution(t *testing.T) {
	trend := 0.8
	input := models.PredictedRiskInput{
		PatientID:   "p-pai",
		PAITrend30d: &trend,
	}
	risk := ComputePredictedRisk(input)

	var paiFactor *models.RiskFactor
	for i := range risk.PrimaryDrivers {
		if risk.PrimaryDrivers[i].FactorName == "rising_pai_trend" {
			paiFactor = &risk.PrimaryDrivers[i]
			break
		}
	}
	if paiFactor == nil {
		t.Fatal("expected rising_pai_trend factor in primary drivers")
	}

	// Expected: min(0.8/1.0 * 20, 20) = 16
	expectedContrib := math.Min(trend/1.0*20.0, 20.0) // 16.0
	if math.Abs(paiFactor.Contribution-expectedContrib) > 0.01 {
		t.Fatalf("PAI trend contribution = %.2f; want ~%.2f", paiFactor.Contribution, expectedContrib)
	}
}

// ── Test 3: Declining engagement → modifiable driver ────────────────────────

func TestRisk_DecliningEngagement_ModifiableDriver(t *testing.T) {
	engTrend := -0.40
	input := models.PredictedRiskInput{
		PatientID:           "p-engage",
		EngagementTrend30d:  &engTrend,
		MeasurementFreqDrop: 0.40,
	}
	risk := ComputePredictedRisk(input)

	var engFactor *models.RiskFactor
	for i := range risk.PrimaryDrivers {
		if risk.PrimaryDrivers[i].FactorName == "declining_engagement" {
			engFactor = &risk.PrimaryDrivers[i]
			break
		}
	}
	if engFactor == nil {
		t.Fatal("expected declining_engagement factor in primary drivers")
	}

	// Both signals are 0.40 — drop = max(0.40, 0.40) = 0.40
	// Contribution = min(0.40/0.50 * 20, 20) = min(16, 20) = 16
	if engFactor.Contribution < 16.0 || engFactor.Contribution > 20.0 {
		t.Fatalf("engagement contribution = %.2f; want 16-20", engFactor.Contribution)
	}
	if !engFactor.Modifiable {
		t.Fatal("engagement factor must be Modifiable=true")
	}
	if engFactor.RecommendedAction == "" {
		t.Fatal("engagement factor must have RecommendedAction set")
	}
}

// ── Test 4: Post-discharge contribution ─────────────────────────────────────

func TestRisk_PostDischarge_Contribution(t *testing.T) {
	input := models.PredictedRiskInput{
		PatientID:          "p-discharge",
		IsPostDischarge:    true,
		DaysSinceDischarge: 7,
	}
	risk := ComputePredictedRisk(input)

	var dischargeFactor *models.RiskFactor
	for i := range risk.PrimaryDrivers {
		if risk.PrimaryDrivers[i].FactorName == "post_discharge_window" {
			dischargeFactor = &risk.PrimaryDrivers[i]
			break
		}
	}
	if dischargeFactor == nil {
		t.Fatal("expected post_discharge_window factor in primary drivers")
	}

	// Expected: 15 * (1 - 7/30) = 15 * 0.7667 ≈ 11.5
	expected := 15.0 * (1.0 - 7.0/30.0)
	if math.Abs(dischargeFactor.Contribution-expected) > 0.5 {
		t.Fatalf("post-discharge contribution = %.2f; want ~%.2f", dischargeFactor.Contribution, expected)
	}
}

// ── Test 5: Stable patient → low risk ───────────────────────────────────────

func TestRisk_StablePatient_LowRisk(t *testing.T) {
	slope := -0.1    // above -0.5 threshold — no trigger
	paiTrend := 0.1  // below 0.3 threshold — no trigger
	engTrend := -0.05 // above -0.20 threshold — no trigger
	input := models.PredictedRiskInput{
		PatientID:             "p-stable",
		CompositeSlope30d:     &slope,
		PAITrend30d:           &paiTrend,
		EngagementTrend30d:    &engTrend,
		MeasurementFreqDrop:   0.05,
		IsPostDischarge:       false,
		MedicationChanges30d:  1,
		PolypharmacyCount:     4,
		ActiveConfounderScore: 0.1,
	}
	risk := ComputePredictedRisk(input)

	if risk.RiskScore >= 15 {
		t.Fatalf("stable patient risk score = %.2f; want < 15", risk.RiskScore)
	}
	if risk.RiskTier != string(models.RiskTierLow) {
		t.Fatalf("stable patient tier = %s; want LOW", risk.RiskTier)
	}
}

// ── Test 6: Compound risk → HIGH tier ───────────────────────────────────────

func TestRisk_CompoundRisk_HighTier(t *testing.T) {
	slope := -1.5
	paiTrend := 0.9
	engTrend := -0.45
	input := models.PredictedRiskInput{
		PatientID:             "p-compound",
		CompositeSlope30d:     &slope,
		PAITrend30d:           &paiTrend,
		EngagementTrend30d:    &engTrend,
		MeasurementFreqDrop:   0.35,
		IsPostDischarge:       true,
		DaysSinceDischarge:    5,
		MedicationChanges30d:  0,
		PolypharmacyCount:     3,
		ActiveConfounderScore: 0.1,
	}
	risk := ComputePredictedRisk(input)

	if risk.RiskScore < 50 {
		t.Fatalf("compound risk score = %.2f; want >= 50", risk.RiskScore)
	}
	if risk.RiskTier != string(models.RiskTierHigh) {
		t.Fatalf("compound risk tier = %s; want HIGH", risk.RiskTier)
	}
	if len(risk.PrimaryDrivers) < 3 {
		t.Fatalf("compound risk primary drivers = %d; want >= 3", len(risk.PrimaryDrivers))
	}
}
