package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestDetectMonitoringLapse_ActiveThenStopped verifies the canonical
// Patient 18 scenario: a patient with 10 readings in the prior 28
// days and no reading in the last 14 days → lapsed.
func TestDetectMonitoringLapse_ActiveThenStopped(t *testing.T) {
	lapsed := DetectMonitoringLapse(10, 20, 7, 14)
	if !lapsed {
		t.Error("expected lapsed=true for 10 readings in prior window + 20 days since last reading")
	}
}

// TestDetectMonitoringLapse_NeverMonitored verifies that a patient
// who never monitored (0 readings in prior window) does NOT produce
// a lapse — you can't lapse from a state you were never in.
func TestDetectMonitoringLapse_NeverMonitored(t *testing.T) {
	lapsed := DetectMonitoringLapse(0, 30, 7, 14)
	if lapsed {
		t.Error("expected lapsed=false for patient who never monitored")
	}
}

// TestDetectMonitoringLapse_StillActive verifies that a patient
// who is still actively monitoring (latest reading 2 days ago) does
// NOT produce a lapse.
func TestDetectMonitoringLapse_StillActive(t *testing.T) {
	lapsed := DetectMonitoringLapse(12, 2, 7, 14)
	if lapsed {
		t.Error("expected lapsed=false for patient still actively monitoring")
	}
}

// TestDetectMonitoringLapse_InfrequentMonitor verifies that a patient
// with only 3 readings in the prior window (below the 7-reading
// minimum) is NOT flagged — they weren't actively monitoring enough
// to constitute a lapse.
func TestDetectMonitoringLapse_InfrequentMonitor(t *testing.T) {
	lapsed := DetectMonitoringLapse(3, 20, 7, 14)
	if lapsed {
		t.Error("expected lapsed=false for infrequent monitor (3 readings < 7 minimum)")
	}
}

// TestDetectMonitoringLapse_ExactlyAtThreshold verifies boundary
// behavior: exactly 7 readings in prior window + exactly 14 days
// since last reading → lapsed (>=7 AND >=14 both satisfied).
func TestDetectMonitoringLapse_ExactlyAtThreshold(t *testing.T) {
	lapsed := DetectMonitoringLapse(7, 14, 7, 14)
	if !lapsed {
		t.Error("expected lapsed=true at exact threshold boundaries")
	}
}

// TestMonitoringEngagementBatch_ShouldRun_OnlyWednesday04UTC verifies
// the batch's cadence gate. Phase 9 P9-B.
func TestMonitoringEngagementBatch_ShouldRun_OnlyWednesday04UTC(t *testing.T) {
	batch := NewMonitoringEngagementBatch(nil, nil, nil, nil, nil, nil, zap.NewNop())

	cases := []struct {
		name     string
		when     time.Time
		expected bool
	}{
		{"Wed 04:00 fires", time.Date(2026, 4, 15, 4, 0, 0, 0, time.UTC), true},
		{"Wed 03:00 skips", time.Date(2026, 4, 15, 3, 0, 0, 0, time.UTC), false},
		{"Wed 05:00 skips", time.Date(2026, 4, 15, 5, 0, 0, 0, time.UTC), false},
		{"Sun 04:00 skips", time.Date(2026, 4, 19, 4, 0, 0, 0, time.UTC), false},
		{"Mon 04:00 skips", time.Date(2026, 4, 13, 4, 0, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := batch.ShouldRun(context.Background(), tc.when)
			if got != tc.expected {
				t.Errorf("ShouldRun(%v) = %v, want %v", tc.when, got, tc.expected)
			}
		})
	}
}

// TestMonitoringEngagementBatch_Name verifies the batch name constant.
func TestMonitoringEngagementBatch_Name(t *testing.T) {
	batch := NewMonitoringEngagementBatch(nil, nil, nil, nil, nil, nil, zap.NewNop())
	if batch.Name() != "monitoring_engagement_weekly" {
		t.Errorf("Name() = %q, want monitoring_engagement_weekly", batch.Name())
	}
}

// TestMonitoringLapsedTemplate_LoadsFromDisk verifies the YAML
// template parses via TemplateLoader. Phase 9 P9-B.
func TestMonitoringLapsedTemplate_LoadsFromDisk(t *testing.T) {
	templatesDir, err := filepath.Abs("../../templates")
	if err != nil {
		t.Fatalf("resolve templates dir: %v", err)
	}
	loader := NewTemplateLoader(templatesDir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("TemplateLoader.Load: %v", err)
	}

	tmpl, ok := loader.Get("dc-monitoring-lapsed-v1")
	if !ok {
		t.Fatal("template dc-monitoring-lapsed-v1 not loaded")
	}
	if tmpl.DifferentialID != "MONITORING_LAPSED" {
		t.Errorf("differential_id = %q, want MONITORING_LAPSED", tmpl.DifferentialID)
	}
	if len(tmpl.Fragments) == 0 {
		t.Error("expected non-empty fragments")
	}
}
