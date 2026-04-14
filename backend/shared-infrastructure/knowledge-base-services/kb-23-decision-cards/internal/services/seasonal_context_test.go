package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

func TestSeasonalContext_EmptyOnMissingFile(t *testing.T) {
	tmp := t.TempDir()
	ctx, err := NewSeasonalContext("india", filepath.Join(tmp, "missing.yaml"))
	if err != nil {
		t.Fatalf("expected nil error on missing file, got %v", err)
	}
	if len(ctx.windows) != 0 {
		t.Errorf("expected no windows, got %d", len(ctx.windows))
	}
}

func TestSeasonalContext_DiwaliWindow_DowngradesGlucose(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "calendar.yaml")
	yamlContent := `
windows:
  - name: diwali
    start: "2026-11-04"
    end: "2026-11-14"
    affected_domains: [GLUCOSE, BODY_COMP]
    mode: DOWNGRADE_URGENCY
    rationale: "festival eating"
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, err := NewSeasonalContext("india", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Inside Diwali window
	tsDiwali := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	suppress, downgrade, rationale := ctx.ShouldSuppress(dtModels.DomainGlucose, tsDiwali)
	if suppress {
		t.Error("expected no suppress for DOWNGRADE_URGENCY mode")
	}
	if !downgrade {
		t.Error("expected downgrade=true during Diwali for GLUCOSE")
	}
	if rationale == "" {
		t.Error("expected non-empty rationale")
	}

	// CARDIO is not affected
	suppress, downgrade, _ = ctx.ShouldSuppress(dtModels.DomainCardio, tsDiwali)
	if suppress || downgrade {
		t.Error("expected no suppression for CARDIO during Diwali")
	}

	// Outside Diwali window
	tsAfter := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	suppress, downgrade, _ = ctx.ShouldSuppress(dtModels.DomainGlucose, tsAfter)
	if suppress || downgrade {
		t.Error("expected no suppression outside Diwali")
	}
}

func TestSeasonalContext_DOYWindow_HeatSeason(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "calendar.yaml")
	yamlContent := `
windows:
  - name: summer_heat
    start_doy: 121
    end_doy: 181
    affected_domains: [CARDIO]
    mode: DOWNGRADE_URGENCY
    rationale: "extreme heat"
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, err := NewSeasonalContext("india", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// May 15 = doy 135 (within window)
	tsMay := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	_, downgrade, _ := ctx.ShouldSuppress(dtModels.DomainCardio, tsMay)
	if !downgrade {
		t.Errorf("expected CARDIO downgrade on May 15 (doy %d)", tsMay.YearDay())
	}

	// April 1 = doy 91 (before window)
	tsApr := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	_, downgrade, _ = ctx.ShouldSuppress(dtModels.DomainCardio, tsApr)
	if downgrade {
		t.Error("expected no CARDIO downgrade on April 1")
	}
}

func TestSeasonalContext_SuppressOverridesDowngrade(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "calendar.yaml")
	yamlContent := `
windows:
  - name: window1
    start: "2026-06-01"
    end: "2026-06-30"
    affected_domains: [GLUCOSE]
    mode: DOWNGRADE_URGENCY
    rationale: "downgrade"
  - name: window2
    start: "2026-06-15"
    end: "2026-06-20"
    affected_domains: [GLUCOSE]
    mode: SUPPRESS
    rationale: "full suppress"
`
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, err := NewSeasonalContext("test", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// June 17 — both windows active
	ts := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	suppress, _, _ := ctx.ShouldSuppress(dtModels.DomainGlucose, ts)
	if !suppress {
		t.Error("expected SUPPRESS to win when both modes active")
	}
}
