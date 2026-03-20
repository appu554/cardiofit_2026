package services

import (
	"testing"
	"time"

	"kb-22-hpi-engine/internal/models"
)

func TestStalenessRules_GlucoseStale(t *testing.T) {
	rules := DefaultStalenessRules()
	glucoseTS := time.Now().Add(-25 * time.Hour) // 25 hours ago
	result := rules.Check("fbg", glucoseTS)
	if result != models.DataPartial {
		t.Errorf("expected PARTIAL for 25h-old glucose, got %s", result)
	}
}

func TestStalenessRules_GlucoseFresh(t *testing.T) {
	rules := DefaultStalenessRules()
	glucoseTS := time.Now().Add(-12 * time.Hour)
	result := rules.Check("fbg", glucoseTS)
	if result != models.DataSufficient {
		t.Errorf("expected SUFFICIENT for 12h-old glucose, got %s", result)
	}
}

func TestStalenessRules_BPStale(t *testing.T) {
	rules := DefaultStalenessRules()
	bpTS := time.Now().Add(-8 * 24 * time.Hour) // 8 days ago
	result := rules.Check("sbp_home_mean", bpTS)
	if result != models.DataPartial {
		t.Errorf("expected PARTIAL for 8d-old BP, got %s", result)
	}
}

func TestStalenessRules_LabFresh(t *testing.T) {
	rules := DefaultStalenessRules()
	labTS := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	result := rules.Check("egfr", labTS)
	if result != models.DataSufficient {
		t.Errorf("expected SUFFICIENT for 30d-old lab, got %s", result)
	}
}

func TestStalenessRules_LabStale(t *testing.T) {
	rules := DefaultStalenessRules()
	labTS := time.Now().Add(-91 * 24 * time.Hour) // 91 days ago
	result := rules.Check("egfr", labTS)
	if result != models.DataPartial {
		t.Errorf("expected PARTIAL for 91d-old lab, got %s", result)
	}
}

func TestStalenessRules_UnknownFieldUsesDefault(t *testing.T) {
	rules := DefaultStalenessRules()
	ts := time.Now().Add(-31 * 24 * time.Hour) // 31 days
	result := rules.Check("unknown_field", ts)
	if result != models.DataPartial {
		t.Errorf("expected PARTIAL for 31d-old unknown field (default 30d), got %s", result)
	}
}
