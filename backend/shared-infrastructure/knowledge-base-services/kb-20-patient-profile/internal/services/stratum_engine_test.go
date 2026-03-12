package services

import (
	"testing"

	"github.com/lib/pq"
	"go.uber.org/zap"

	"kb-patient-profile/internal/models"
)

// newTestStratumEngine creates a minimal StratumEngine for unit-testing
// determineStratum — no DB, cache, or metrics required.
func newTestStratumEngine() *StratumEngine {
	return &StratumEngine{
		logger: zap.NewNop(),
	}
}

func TestDetermineStratum_DMHTNCKDHF(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN", "HF"},
	}
	got := se.determineStratum(profile, true, 45.0)
	if got != models.StratumDMHTNCKDHF {
		t.Errorf("expected %s, got %s", models.StratumDMHTNCKDHF, got)
	}
}

func TestDetermineStratum_DMHTNCKDHF_HFrEF(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN", "HFrEF"},
	}
	got := se.determineStratum(profile, true, 35.0)
	if got != models.StratumDMHTNCKDHF {
		t.Errorf("expected %s for HFrEF variant, got %s", models.StratumDMHTNCKDHF, got)
	}
}

func TestDetermineStratum_DMHTNCKDHF_HFpEF(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"UNCONTROLLED_HTN", "HFpEF"},
	}
	got := se.determineStratum(profile, true, 50.0)
	if got != models.StratumDMHTNCKDHF {
		t.Errorf("expected %s for HFpEF variant, got %s", models.StratumDMHTNCKDHF, got)
	}
}

func TestDetermineStratum_DMHTNCKDHF_HFmrEF(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T1DM",
		Comorbidities: pq.StringArray{"HTN", "HFmrEF"},
	}
	got := se.determineStratum(profile, true, 28.0)
	if got != models.StratumDMHTNCKDHF {
		t.Errorf("expected %s for HFmrEF variant, got %s", models.StratumDMHTNCKDHF, got)
	}
}

func TestDetermineStratum_DMHTNCKDHF_HeartFailureString(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN", "HEART_FAILURE"},
	}
	got := se.determineStratum(profile, true, 40.0)
	if got != models.StratumDMHTNCKDHF {
		t.Errorf("expected %s for HEART_FAILURE string, got %s", models.StratumDMHTNCKDHF, got)
	}
}

func TestDetermineStratum_DMHTNCKD_WithoutHF(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN"},
	}
	got := se.determineStratum(profile, true, 45.0)
	if got != models.StratumDMHTNCKD {
		t.Errorf("expected %s (no HF), got %s", models.StratumDMHTNCKD, got)
	}
}

func TestDetermineStratum_HFWithoutCKD_FallsToDMHTN(t *testing.T) {
	// HF present but eGFR >= 60 (no CKD) → DM_HTN, not DM_HTN_CKD_HF
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN", "HF"},
	}
	got := se.determineStratum(profile, true, 75.0)
	if got != models.StratumDMHTN {
		t.Errorf("expected %s (eGFR >= 60, no CKD), got %s", models.StratumDMHTN, got)
	}
}

func TestDetermineStratum_HFWithoutDM_FallsToHTNOnly(t *testing.T) {
	// HF + HTN but no DM → HTN_ONLY (HF alone doesn't create a stratum)
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "NONE",
		Comorbidities: pq.StringArray{"HTN", "HF"},
	}
	got := se.determineStratum(profile, true, 45.0)
	if got != models.StratumHTNOnly {
		t.Errorf("expected %s (no DM), got %s", models.StratumHTNOnly, got)
	}
}

func TestDetermineStratum_DMHTN(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN"},
	}
	got := se.determineStratum(profile, false, 0)
	if got != models.StratumDMHTN {
		t.Errorf("expected %s, got %s", models.StratumDMHTN, got)
	}
}

func TestDetermineStratum_DMOnly(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{},
	}
	got := se.determineStratum(profile, false, 0)
	if got != models.StratumDMOnly {
		t.Errorf("expected %s, got %s", models.StratumDMOnly, got)
	}
}

func TestDetermineStratum_HTNOnly(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "NONE",
		Comorbidities: pq.StringArray{"HTN"},
	}
	got := se.determineStratum(profile, false, 0)
	if got != models.StratumHTNOnly {
		t.Errorf("expected %s, got %s", models.StratumHTNOnly, got)
	}
}

func TestDetermineStratum_NoConditions_NONE(t *testing.T) {
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "NONE",
		Comorbidities: pq.StringArray{},
	}
	got := se.determineStratum(profile, false, 0)
	if got != "NONE" {
		t.Errorf("expected NONE, got %s", got)
	}
}

func TestDetermineStratum_CKDBoundary_eGFR60_NoCKD(t *testing.T) {
	// eGFR exactly 60 is NOT CKD (CKD is < 60)
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN"},
	}
	got := se.determineStratum(profile, true, 60.0)
	if got != models.StratumDMHTN {
		t.Errorf("expected %s (eGFR=60, not CKD), got %s", models.StratumDMHTN, got)
	}
}

func TestDetermineStratum_CKDBoundary_eGFR59_IsCKD(t *testing.T) {
	// eGFR 59 triggers CKD
	se := newTestStratumEngine()
	profile := models.PatientProfile{
		DMType:        "T2DM",
		Comorbidities: pq.StringArray{"HTN"},
	}
	got := se.determineStratum(profile, true, 59.0)
	if got != models.StratumDMHTNCKD {
		t.Errorf("expected %s (eGFR=59, is CKD), got %s", models.StratumDMHTNCKD, got)
	}
}
