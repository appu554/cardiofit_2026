package services

import (
	"testing"
	"time"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestStreakUpdate_IncreasesStreak(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	streak := &models.PatientStreak{
		PatientID:     "patient-g-1",
		Behavior:      "WALK_AFTER_LUNCH",
		CurrentStreak: 5,
		LongestStreak: 10,
		LastActiveDay: time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour),
	}
	engine.UpdateStreak(streak, time.Now().UTC())
	if streak.CurrentStreak != 6 {
		t.Errorf("expected CurrentStreak=6, got %d", streak.CurrentStreak)
	}
	if streak.LongestStreak != 10 {
		t.Errorf("expected LongestStreak=10 (not beaten yet), got %d", streak.LongestStreak)
	}
}

func TestStreakUpdate_BreaksOnGap(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	streak := &models.PatientStreak{
		PatientID:     "patient-g-2",
		Behavior:      "WALK_AFTER_LUNCH",
		CurrentStreak: 5,
		LongestStreak: 5,
		LastActiveDay: time.Now().UTC().AddDate(0, 0, -3).Truncate(24 * time.Hour), // 3-day gap
	}
	engine.UpdateStreak(streak, time.Now().UTC())
	if streak.CurrentStreak != 1 {
		t.Errorf("expected CurrentStreak=1 (reset), got %d", streak.CurrentStreak)
	}
	if streak.LongestStreak != 5 {
		t.Errorf("expected LongestStreak=5 (preserved), got %d", streak.LongestStreak)
	}
}

func TestStreakUpdate_PausedDoesNotBreak(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	pausedAt := time.Now().UTC().AddDate(0, 0, -3)
	streak := &models.PatientStreak{
		PatientID:     "patient-g-3",
		Behavior:      "MEDICATION_TAKEN",
		CurrentStreak: 7,
		LongestStreak: 7,
		LastActiveDay: time.Now().UTC().AddDate(0, 0, -4).Truncate(24 * time.Hour),
		Paused:        true,
		PausedAt:      &pausedAt,
		PauseReason:   "ILLNESS",
	}
	engine.UpdateStreak(streak, time.Now().UTC())
	if streak.CurrentStreak != 8 {
		t.Errorf("expected CurrentStreak=8 (continues from pause), got %d", streak.CurrentStreak)
	}
	if streak.Paused {
		t.Error("expected Paused=false (unpaused after event), got true")
	}
}

func TestStreakUpdate_NewLongest(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	streak := &models.PatientStreak{
		PatientID:     "patient-g-4",
		Behavior:      "WALK_AFTER_LUNCH",
		CurrentStreak: 10,
		LongestStreak: 10,
		LastActiveDay: time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour),
	}
	engine.UpdateStreak(streak, time.Now().UTC())
	if streak.CurrentStreak != 11 {
		t.Errorf("expected CurrentStreak=11, got %d", streak.CurrentStreak)
	}
	if streak.LongestStreak != 11 {
		t.Errorf("expected LongestStreak=11 (new record), got %d", streak.LongestStreak)
	}
}

func TestShouldActivateGamification_RewardResponsive(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	result := engine.ShouldActivate(models.PhenotypeRewardResponsive, 0.05)
	if !result {
		t.Error("expected ShouldActivate=true for REWARD_RESPONSIVE phenotype, got false")
	}
}

func TestShouldActivateGamification_HighT06Posterior(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	result := engine.ShouldActivate(models.PhenotypeRoutineBuilder, 0.20)
	if !result {
		t.Error("expected ShouldActivate=true for T-06 posterior > 0.15, got false")
	}
}

func TestShouldActivateGamification_LowT06_NonReward(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	result := engine.ShouldActivate(models.PhenotypeSupportDependent, 0.05)
	if result {
		t.Error("expected ShouldActivate=false for SUPPORT_DEPENDENT with low T-06 posterior, got true")
	}
}

func TestDetectMilestone_FirstWeek(t *testing.T) {
	engine := NewGamificationEngine(nil, nil)
	milestones := engine.DetectMilestones("patient-g-5", 7, 0.65, nil)
	found := false
	for _, m := range milestones {
		if m.MilestoneType == "FIRST_WEEK_COMPLETE" {
			found = true
		}
	}
	if !found {
		t.Error("expected FIRST_WEEK_COMPLETE milestone to be detected at cycleDay=7, but it was not")
	}
}
