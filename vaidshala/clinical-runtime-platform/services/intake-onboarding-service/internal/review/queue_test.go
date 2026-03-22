package review

import (
	"testing"
)

func TestClassifyRisk_High_HardStop(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 1,
		SoftFlagCount: 0,
		Age:           50,
		MedCount:      2,
		EGFRValue:     90,
	})
	if got != RiskHigh {
		t.Errorf("expected HIGH, got %s", got)
	}
}

func TestClassifyRisk_High_ManySoftFlags(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 3,
		Age:           50,
		MedCount:      2,
		EGFRValue:     90,
	})
	if got != RiskHigh {
		t.Errorf("expected HIGH, got %s", got)
	}
}

func TestClassifyRisk_High_LowEGFR(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 0,
		Age:           50,
		MedCount:      2,
		EGFRValue:     25,
	})
	if got != RiskHigh {
		t.Errorf("expected HIGH, got %s", got)
	}
}

func TestClassifyRisk_Medium_SoftFlag(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 1,
		Age:           50,
		MedCount:      2,
		EGFRValue:     90,
	})
	if got != RiskMedium {
		t.Errorf("expected MEDIUM, got %s", got)
	}
}

func TestClassifyRisk_Medium_Polypharmacy(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 0,
		Age:           50,
		MedCount:      6,
		EGFRValue:     90,
	})
	if got != RiskMedium {
		t.Errorf("expected MEDIUM, got %s", got)
	}
}

func TestClassifyRisk_Medium_Elderly(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 0,
		Age:           80,
		MedCount:      2,
		EGFRValue:     90,
	})
	if got != RiskMedium {
		t.Errorf("expected MEDIUM, got %s", got)
	}
}

func TestClassifyRisk_Low(t *testing.T) {
	got := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 0,
		Age:           40,
		MedCount:      1,
		EGFRValue:     95,
	})
	if got != RiskLow {
		t.Errorf("expected LOW, got %s", got)
	}
}

func TestReviewStatus_Constants(t *testing.T) {
	statuses := []ReviewStatus{
		StatusPending,
		StatusApproved,
		StatusClarification,
		StatusEscalated,
	}
	if len(statuses) != 4 {
		t.Fatalf("expected 4 statuses, got %d", len(statuses))
	}

	expected := map[ReviewStatus]bool{
		"PENDING":       true,
		"APPROVED":      true,
		"CLARIFICATION": true,
		"ESCALATED":     true,
	}
	for _, s := range statuses {
		if !expected[s] {
			t.Errorf("unexpected status value: %s", s)
		}
	}
}
