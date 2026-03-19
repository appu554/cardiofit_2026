package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestSelectDeliveryTime_ReturnsSlot(t *testing.T) {
	bandit := NewTimingBandit(nil, nil)
	profiles := make([]*models.PatientTimingProfile, 0)
	for _, slot := range models.AllTimingSlots() {
		profiles = append(profiles, &models.PatientTimingProfile{
			PatientID: "patient-t-1",
			Slot:      slot,
			Alpha:     1.0,
			Beta:      1.0,
		})
	}
	slot := bandit.SelectDeliveryTime(profiles)
	found := false
	for _, s := range models.AllTimingSlots() {
		if s == slot {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SelectDeliveryTime returned invalid slot: %s", slot)
	}
}

func TestSelectDeliveryTime_FavorsHighAlpha(t *testing.T) {
	bandit := NewTimingBandit(nil, nil)
	profiles := []*models.PatientTimingProfile{
		{PatientID: "patient-t-2", Slot: models.Slot7AM, Alpha: 1.0, Beta: 10.0},
		{PatientID: "patient-t-2", Slot: models.Slot9AM, Alpha: 50.0, Beta: 1.0}, // strong winner
		{PatientID: "patient-t-2", Slot: models.Slot7PM, Alpha: 1.0, Beta: 10.0},
	}
	counts := map[models.TimingSlot]int{}
	for i := 0; i < 100; i++ {
		slot := bandit.SelectDeliveryTime(profiles)
		counts[slot]++
	}
	if counts[models.Slot9AM] <= 80 {
		t.Errorf("Expected Slot9AM to win >80 of 100 trials, got %d", counts[models.Slot9AM])
	}
}

func TestObserveTimingReward_UpdatesAlpha(t *testing.T) {
	bandit := NewTimingBandit(nil, nil)
	profile := &models.PatientTimingProfile{
		PatientID: "patient-t-3",
		Slot:      models.Slot7PM,
		Alpha:     1.0,
		Beta:      1.0,
	}
	bandit.ObserveReward(profile, true)
	if profile.Alpha != 2.0 {
		t.Errorf("Alpha: got %f, want 2.0", profile.Alpha)
	}
	if profile.Beta != 1.0 {
		t.Errorf("Beta: got %f, want 1.0", profile.Beta)
	}
	if profile.Deliveries != 1 {
		t.Errorf("Deliveries: got %d, want 1", profile.Deliveries)
	}
	if profile.Responses != 1 {
		t.Errorf("Responses: got %d, want 1", profile.Responses)
	}
}

func TestObserveTimingReward_UpdatesBeta(t *testing.T) {
	bandit := NewTimingBandit(nil, nil)
	profile := &models.PatientTimingProfile{
		PatientID: "patient-t-4",
		Slot:      models.Slot12PM,
		Alpha:     1.0,
		Beta:      1.0,
	}
	bandit.ObserveReward(profile, false)
	if profile.Alpha != 1.0 {
		t.Errorf("Alpha: got %f, want 1.0", profile.Alpha)
	}
	if profile.Beta != 2.0 {
		t.Errorf("Beta: got %f, want 2.0", profile.Beta)
	}
	if profile.Deliveries != 1 {
		t.Errorf("Deliveries: got %d, want 1", profile.Deliveries)
	}
	if profile.Responses != 0 {
		t.Errorf("Responses: got %d, want 0", profile.Responses)
	}
}

func TestEnsureTimingProfiles_Creates7Slots(t *testing.T) {
	bandit := NewTimingBandit(nil, nil)
	profiles := bandit.BuildDefaultProfiles("patient-t-5")
	if len(profiles) != 7 {
		t.Errorf("BuildDefaultProfiles: got %d profiles, want 7", len(profiles))
	}
}
