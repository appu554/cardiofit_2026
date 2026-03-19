package services

import "testing"

func TestGetCheckinCadence_PRPStabilization(t *testing.T) {
	c := GetCheckinCadence("M3-PRP", "STABILIZATION")
	if len(c.FixedDays) != 5 {
		t.Errorf("expected 5 fixed check-in days, got %d", len(c.FixedDays))
	}
	if c.FixedDays[0] != 1 || c.FixedDays[4] != 14 {
		t.Error("expected first=1, last=14")
	}
}

func TestGetCheckinCadence_UnknownPhase_DefaultWeekly(t *testing.T) {
	c := GetCheckinCadence("M3-PRP", "UNKNOWN_PHASE")
	if c.IntervalDays != 7 {
		t.Errorf("expected default 7-day interval, got %d", c.IntervalDays)
	}
}

func TestCheckinCadence_NextCheckinDay_FixedDays(t *testing.T) {
	c := GetCheckinCadence("M3-PRP", "STABILIZATION")
	// At day 5, next should be day 7
	next := c.NextCheckinDay(5)
	if next != 7 {
		t.Errorf("expected next check-in at day 7, got %d", next)
	}
}

func TestCheckinCadence_NextCheckinDay_PastAllFixed(t *testing.T) {
	c := GetCheckinCadence("M3-PRP", "STABILIZATION")
	next := c.NextCheckinDay(14)
	if next != -1 {
		t.Errorf("expected -1 (past all fixed days), got %d", next)
	}
}

func TestCheckinCadence_NextCheckinDay_Interval(t *testing.T) {
	c := GetCheckinCadence("M3-VFRP", "METABOLIC_STABILIZATION")
	next := c.NextCheckinDay(5)
	if next != 8 {
		t.Errorf("expected next check-in at day 8 (interval 4), got %d", next)
	}
}

func TestCheckinCadence_IsLabDay(t *testing.T) {
	c := GetCheckinCadence("M3-PRP", "RESTORATION")
	if !c.IsLabDay(14) {
		t.Error("expected day 14 to be a lab day")
	}
	if c.IsLabDay(7) {
		t.Error("expected day 7 to NOT be a lab day")
	}
}

func TestCheckinCadence_VFRPSustainedReduction_HasFullPanelLabs(t *testing.T) {
	c := GetCheckinCadence("M3-VFRP", "SUSTAINED_REDUCTION")
	if len(c.LabTypes) < 4 {
		t.Errorf("expected at least 4 lab types for VFRP Phase 3, got %d", len(c.LabTypes))
	}
}
