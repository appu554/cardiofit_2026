package services

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"kb-patient-profile/internal/models"
)

// ── Test Helpers ────────────────────────────────────────────────────────────

func baseProfile() models.PatientProfile {
	return models.PatientProfile{
		PatientID: "test-patient-001",
		Age:       55,
		Sex:       "M",
		DMType:    "T2DM",
		HTNStatus: "CONFIRMED",
		CKDStage:  "",
		CKDStatus: "NONE",
	}
}

func f64(v float64) *float64 { return &v }

// ── FBG Target Tests ────────────────────────────────────────────────────────

func TestFBGTarget_Standard(t *testing.T) {
	p := baseProfile()
	p.DMDurationYears = 8

	target, reason := computeFBGTarget(p)
	assert.Equal(t, DefaultFBGTarget, target)
	assert.Contains(t, reason, "standard")
}

func TestFBGTarget_Elderly(t *testing.T) {
	p := baseProfile()
	p.Age = 78

	target, reason := computeFBGTarget(p)
	assert.Equal(t, 130.0, target)
	assert.Contains(t, reason, "age ≥75")
}

func TestFBGTarget_LongDurationDM(t *testing.T) {
	p := baseProfile()
	p.DMDurationYears = 25

	target, reason := computeFBGTarget(p)
	assert.Equal(t, 130.0, target)
	assert.Contains(t, reason, "DM duration >20y")
}

func TestFBGTarget_CKDG4(t *testing.T) {
	p := baseProfile()
	p.CKDStage = models.CKDG4

	target, reason := computeFBGTarget(p)
	assert.Equal(t, 130.0, target)
	assert.Contains(t, reason, "CKD G4/G5")
}

func TestFBGTarget_HeartFailure(t *testing.T) {
	p := baseProfile()
	p.Comorbidities = pq.StringArray{"HFrEF"}

	target, reason := computeFBGTarget(p)
	assert.Equal(t, 130.0, target)
	assert.Contains(t, reason, "heart failure")
}

func TestFBGTarget_YoungShortDuration(t *testing.T) {
	p := baseProfile()
	p.Age = 35
	p.DMDurationYears = 2

	target, reason := computeFBGTarget(p)
	assert.Equal(t, 100.0, target)
	assert.Contains(t, reason, "tightened")
}

// ── HbA1c Target Tests ──────────────────────────────────────────────────────

func TestHbA1cTarget_Standard(t *testing.T) {
	p := baseProfile()
	p.DMDurationYears = 8

	target, reason := computeHbA1cTarget(p)
	assert.Equal(t, DefaultHbA1cTarget, target)
	assert.Contains(t, reason, "standard")
}

func TestHbA1cTarget_Elderly(t *testing.T) {
	p := baseProfile()
	p.Age = 80

	target, _ := computeHbA1cTarget(p)
	assert.Equal(t, 8.0, target)
}

func TestHbA1cTarget_LongDurationDM(t *testing.T) {
	p := baseProfile()
	p.DMDurationYears = 18

	target, _ := computeHbA1cTarget(p)
	assert.Equal(t, 8.0, target)
}

func TestHbA1cTarget_YoungNoCVD(t *testing.T) {
	p := baseProfile()
	p.Age = 40
	p.DMDurationYears = 3
	p.HasClinicalCVD = false

	target, reason := computeHbA1cTarget(p)
	assert.Equal(t, 6.5, target)
	assert.Contains(t, reason, "tightened")
}

func TestHbA1cTarget_YoungWithCVD(t *testing.T) {
	p := baseProfile()
	p.Age = 40
	p.DMDurationYears = 3
	p.HasClinicalCVD = true

	target, _ := computeHbA1cTarget(p)
	assert.Equal(t, DefaultHbA1cTarget, target, "CVD should prevent tightening")
}

// ── SBP Target Tests ────────────────────────────────────────────────────────

func TestSBPTarget_Standard(t *testing.T) {
	p := baseProfile()

	target, _ := computeSBPTarget(p, nil, nil, nil)
	assert.Equal(t, DefaultSBPTarget, target)
}

func TestSBPTarget_ElderlyNoProteinuria(t *testing.T) {
	p := baseProfile()
	p.Age = 82

	target, reason := computeSBPTarget(p, nil, nil, nil)
	assert.Equal(t, 140.0, target)
	assert.Contains(t, reason, "age ≥80")
}

func TestSBPTarget_Proteinuria(t *testing.T) {
	p := baseProfile()
	uacr := 45.0

	target, reason := computeSBPTarget(p, nil, &uacr, nil)
	assert.Equal(t, 120.0, target)
	assert.Contains(t, reason, "proteinuria")
}

func TestSBPTarget_ElderlyWithProteinuria(t *testing.T) {
	// Proteinuria should override elderly relaxation
	p := baseProfile()
	p.Age = 85
	uacr := 60.0

	target, _ := computeSBPTarget(p, nil, &uacr, nil)
	// Age ≥80 check comes first, but UACR > 30 should not match since age check fails first
	// Actually looking at the code: age ≥80 AND uacr ≤ 30 → 140. Here uacr > 30 so it falls through.
	assert.Equal(t, 120.0, target, "proteinuria should override age relaxation")
}

func TestSBPTarget_PREVENTOverride(t *testing.T) {
	p := baseProfile()
	preventTarget := 120.0

	target, reason := computeSBPTarget(p, nil, nil, &preventTarget)
	assert.Equal(t, 120.0, target)
	assert.Contains(t, reason, "PREVENT")
}

func TestSBPTarget_AdvancedCKD(t *testing.T) {
	p := baseProfile()
	egfr := 38.0

	target, _ := computeSBPTarget(p, &egfr, nil, nil)
	assert.Equal(t, 120.0, target)
}

// ── eGFR Threshold Tests ────────────────────────────────────────────────────

func TestEGFRThreshold_NoCKD(t *testing.T) {
	p := baseProfile()

	threshold, _ := computeEGFRThreshold(p, nil)
	assert.Equal(t, DefaultEGFRThreshold, threshold)
}

func TestEGFRThreshold_G3a(t *testing.T) {
	p := baseProfile()
	p.CKDStage = models.CKDG3a

	threshold, reason := computeEGFRThreshold(p, nil)
	assert.Equal(t, 45.0, threshold)
	assert.Contains(t, reason, "G3a")
}

func TestEGFRThreshold_G3b(t *testing.T) {
	p := baseProfile()
	p.CKDStage = models.CKDG3b

	threshold, reason := computeEGFRThreshold(p, nil)
	assert.Equal(t, 30.0, threshold)
	assert.Contains(t, reason, "G3b")
}

func TestEGFRThreshold_G4(t *testing.T) {
	p := baseProfile()
	p.CKDStage = models.CKDG4

	threshold, _ := computeEGFRThreshold(p, nil)
	assert.Equal(t, 15.0, threshold)
}

func TestEGFRThreshold_G5(t *testing.T) {
	p := baseProfile()
	p.CKDStage = models.CKDG5

	threshold, _ := computeEGFRThreshold(p, nil)
	assert.Equal(t, 0.0, threshold)
}

// ── SBP Kidney Threshold Tests ──────────────────────────────────────────────

func TestSBPKidneyThreshold_Default(t *testing.T) {
	p := baseProfile()

	threshold := computeSBPKidneyThreshold(p, nil)
	assert.Equal(t, DefaultSBPKidneyThreshold, threshold)
}

func TestSBPKidneyThreshold_Proteinuria(t *testing.T) {
	p := baseProfile()
	uacr := 50.0

	threshold := computeSBPKidneyThreshold(p, &uacr)
	assert.Equal(t, 130.0, threshold)
}

func TestSBPKidneyThreshold_Elderly(t *testing.T) {
	p := baseProfile()
	p.Age = 82

	threshold := computeSBPKidneyThreshold(p, nil)
	assert.Equal(t, 150.0, threshold)
}

// ── Integration: ComputePersonalizedTargets ─────────────────────────────────

func TestComputePersonalizedTargets_StandardPatient(t *testing.T) {
	p := baseProfile()
	p.DMDurationYears = 8
	egfr := 72.0

	targets := ComputePersonalizedTargets(p, &egfr, nil, nil)

	assert.Equal(t, p.PatientID, targets.PatientID)
	assert.Equal(t, DefaultFBGTarget, targets.FBGTarget)
	assert.Equal(t, DefaultHbA1cTarget, targets.HbA1cTarget)
	assert.Equal(t, DefaultSBPTarget, targets.SBPTarget)
	assert.Equal(t, DefaultSBPKidneyThreshold, targets.SBPKidneyThreshold)
	assert.Equal(t, DefaultEGFRThreshold, targets.EGFRThreshold)
	assert.NotZero(t, targets.ComputedAt)
}

func TestComputePersonalizedTargets_ElderlyWithCKD(t *testing.T) {
	p := baseProfile()
	p.Age = 78
	p.CKDStage = models.CKDG3b
	p.DMDurationYears = 20
	egfr := 35.0
	uacr := 80.0

	targets := ComputePersonalizedTargets(p, &egfr, &uacr, nil)

	assert.Equal(t, 130.0, targets.FBGTarget, "relaxed for age ≥75")
	assert.Equal(t, 8.0, targets.HbA1cTarget, "relaxed for age ≥75")
	assert.Equal(t, 120.0, targets.SBPTarget, "tightened for proteinuria")
	assert.Equal(t, 130.0, targets.SBPKidneyThreshold, "tightened for proteinuria")
	assert.Equal(t, 30.0, targets.EGFRThreshold, "G3b → alert on G4 progression")
}

func TestComputePersonalizedTargets_YoungNewlyDiagnosed(t *testing.T) {
	p := baseProfile()
	p.Age = 35
	p.DMDurationYears = 2
	p.HasClinicalCVD = false
	egfr := 95.0

	targets := ComputePersonalizedTargets(p, &egfr, nil, nil)

	assert.Equal(t, 100.0, targets.FBGTarget, "tightened for young short-duration DM")
	assert.Equal(t, 6.5, targets.HbA1cTarget, "tightened for young short-duration no CVD")
	assert.Equal(t, DefaultSBPTarget, targets.SBPTarget)
	assert.Equal(t, DefaultEGFRThreshold, targets.EGFRThreshold)
}

// ── Helper Tests ────────────────────────────────────────────────────────────

func TestContainsAny(t *testing.T) {
	assert.True(t, containsAny([]string{"HTN", "HFrEF", "T2DM"}, "HFrEF"))
	assert.True(t, containsAny([]string{"HF"}, "HF", "HEART_FAILURE"))
	assert.False(t, containsAny([]string{"HTN"}, "HF", "HEART_FAILURE"))
	assert.False(t, containsAny(nil, "HF"))
	assert.False(t, containsAny([]string{}, "HF"))
}
