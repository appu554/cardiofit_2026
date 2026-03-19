package services

import (
	"testing"
	"time"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestNudgeEngine_SelectTechnique_RespectsPhase(t *testing.T) {
	ne := NewNudgeEngine(nil, nil, nil, nil, nil, 3, 4)

	// In RECOVERY phase, T-11 should be selected overwhelmingly
	records := ne.bayesian.BuildDefaultRecords("patient-1")
	ptrs := make([]*models.TechniqueEffectiveness, len(records))
	for i := range records {
		ptrs[i] = &records[i]
	}

	phase := &models.PatientMotivationPhase{
		PatientID: "patient-1",
		Phase:     models.PhaseRecovery,
		CycleDay:  30,
	}

	t11Wins := 0
	for i := 0; i < 100; i++ {
		tech := ne.selectTechniqueForPhase(ptrs, phase)
		if tech.Technique == models.TechRecoveryProtocol {
			t11Wins++
		}
	}
	if t11Wins < 80 {
		t.Errorf("T-11 should dominate in RECOVERY phase, got %d/100 wins", t11Wins)
	}
}

func TestNudgeEngine_FatigueCheck_BlocksRecentTechnique(t *testing.T) {
	ne := NewNudgeEngine(nil, nil, nil, nil, nil, 3, 4)

	recent := time.Now().Add(-2 * time.Hour) // 2 hours ago
	rec := &models.TechniqueEffectiveness{
		Technique:     models.TechMicroCommitment,
		LastDelivered: &recent,
	}

	if !ne.isFatigued(rec, 4*time.Hour) {
		t.Error("technique delivered 2h ago should be fatigued with 4h cooldown")
	}
}

func TestNudgeEngine_FatigueCheck_AllowsOldTechnique(t *testing.T) {
	ne := NewNudgeEngine(nil, nil, nil, nil, nil, 3, 4)

	old := time.Now().Add(-6 * time.Hour) // 6 hours ago
	rec := &models.TechniqueEffectiveness{
		Technique:     models.TechMicroCommitment,
		LastDelivered: &old,
	}

	if ne.isFatigued(rec, 4*time.Hour) {
		t.Error("technique delivered 6h ago should not be fatigued with 4h cooldown")
	}
}

func TestNudgeEngine_DailyLimit(t *testing.T) {
	ne := NewNudgeEngine(nil, nil, nil, nil, nil, 3, 4)
	if ne.maxNudgesPerDay != 3 {
		t.Errorf("max nudges per day: got %d, want 3", ne.maxNudgesPerDay)
	}
}
