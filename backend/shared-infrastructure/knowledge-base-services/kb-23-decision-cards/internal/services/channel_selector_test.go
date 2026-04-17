package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

func TestSelector_Safety_AllChannels(t *testing.T) {
	sel := SelectChannels(models.TierSafety, nil, time.Now())

	if !sel.Simultaneous {
		t.Fatal("expected Simultaneous=true for SAFETY tier")
	}
	want := []string{"push", "sms", "whatsapp"}
	if len(sel.PrimaryChannels) != len(want) {
		t.Fatalf("expected %d primary channels, got %d", len(want), len(sel.PrimaryChannels))
	}
	for i, ch := range want {
		if sel.PrimaryChannels[i] != ch {
			t.Errorf("PrimaryChannels[%d]: want %q, got %q", i, ch, sel.PrimaryChannels[i])
		}
	}
	if sel.Suppressed {
		t.Error("SAFETY should not be suppressed")
	}
}

func TestSelector_Immediate_PrimaryWithFallback(t *testing.T) {
	sel := SelectChannels(models.TierImmediate, nil, time.Date(2026, 4, 17, 14, 0, 0, 0, time.UTC))

	if len(sel.PrimaryChannels) != 1 || sel.PrimaryChannels[0] != "push" {
		t.Fatalf("expected primary=[push], got %v", sel.PrimaryChannels)
	}
	if len(sel.FallbackChannels) != 1 || sel.FallbackChannels[0] != "sms" {
		t.Fatalf("expected fallback=[sms], got %v", sel.FallbackChannels)
	}
	if sel.FallbackAfter != 2*time.Hour {
		t.Errorf("expected FallbackAfter=2h, got %v", sel.FallbackAfter)
	}
	if sel.Simultaneous {
		t.Error("IMMEDIATE should not be simultaneous")
	}
}

func TestSelector_Routine_InAppOnly(t *testing.T) {
	sel := SelectChannels(models.TierRoutine, nil, time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC))

	if len(sel.PrimaryChannels) != 1 || sel.PrimaryChannels[0] != "in_app" {
		t.Fatalf("expected primary=[in_app], got %v", sel.PrimaryChannels)
	}
	if len(sel.FallbackChannels) != 0 {
		t.Errorf("expected no fallback channels, got %v", sel.FallbackChannels)
	}
}

func TestSelector_QuietHours_SuppressUrgent(t *testing.T) {
	now := time.Date(2026, 4, 17, 23, 0, 0, 0, time.UTC) // 23:00 — inside default quiet hours 22:00-06:00

	sel := SelectChannels(models.TierUrgent, nil, now)

	if !sel.Suppressed {
		t.Fatal("expected URGENT to be suppressed during quiet hours")
	}
}

func TestSelector_QuietHours_BypassSafety(t *testing.T) {
	now := time.Date(2026, 4, 17, 23, 0, 0, 0, time.UTC) // 23:00 — inside quiet hours

	sel := SelectChannels(models.TierSafety, nil, now)

	if sel.Suppressed {
		t.Fatal("SAFETY must BYPASS quiet hours — should not be suppressed")
	}
	if !sel.Simultaneous {
		t.Error("SAFETY should remain simultaneous even during quiet hours")
	}
	if len(sel.PrimaryChannels) != 3 {
		t.Errorf("expected 3 primary channels for SAFETY, got %d", len(sel.PrimaryChannels))
	}
}

func TestSelector_ClinicianPreference_Override(t *testing.T) {
	prefs := &models.ClinicianPreferences{
		PreferredChannels: `["whatsapp"]`,
		QuietHoursStart:   "22:00",
		QuietHoursEnd:     "06:00",
	}
	now := time.Date(2026, 4, 17, 14, 0, 0, 0, time.UTC) // outside quiet hours

	sel := SelectChannels(models.TierImmediate, prefs, now)

	if len(sel.PrimaryChannels) != 1 || sel.PrimaryChannels[0] != "whatsapp" {
		t.Fatalf("expected clinician pref override primary=[whatsapp], got %v", sel.PrimaryChannels)
	}
}
