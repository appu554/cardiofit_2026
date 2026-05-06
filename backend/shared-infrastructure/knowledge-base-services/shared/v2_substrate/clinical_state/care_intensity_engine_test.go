package clinical_state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestCareIntensityEngine_OnTransition_ActiveToPalliative_ProducesThreeCascades(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	eng := NewCareIntensityEngine(WithCareIntensityClock(fixedClock(now)))

	residentRef := uuid.New()
	roleRef := uuid.New()
	ev, cascades := eng.OnTransition(
		models.CareIntensityTagActiveTreatment,
		models.CareIntensityTagPalliative,
		residentRef,
		roleRef,
	)

	if len(cascades) != 3 {
		t.Fatalf("expected 3 cascades for active→palliative; got %d: %+v", len(cascades), cascades)
	}
	want := map[string]bool{
		CareIntensityCascadeReviewPreventiveMedications: false,
		CareIntensityCascadeRevisitMonitoringPlan:       false,
		CareIntensityCascadeConsentRefreshNeeded:        false,
	}
	for _, c := range cascades {
		if _, ok := want[c.Kind]; !ok {
			t.Errorf("unexpected cascade kind %q", c.Kind)
			continue
		}
		want[c.Kind] = true
		if c.Reason == "" {
			t.Errorf("cascade %q has empty Reason", c.Kind)
		}
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("expected cascade %q to be produced", k)
		}
	}

	// Event invariants.
	if ev.EventType != models.EventTypeCareIntensityTransition {
		t.Errorf("expected event_type=care_intensity_transition; got %s", ev.EventType)
	}
	if ev.ResidentID != residentRef {
		t.Errorf("ResidentID drift")
	}
	if ev.ReportedByRef != roleRef {
		t.Errorf("ReportedByRef drift")
	}
	if !ev.OccurredAt.Equal(now) {
		t.Errorf("OccurredAt drift: got %v want %v", ev.OccurredAt, now)
	}
	if ev.Severity != models.EventSeverityModerate {
		t.Errorf("expected severity=moderate for active→palliative; got %s", ev.Severity)
	}
	if len(ev.DescriptionStructured) == 0 {
		t.Fatalf("expected DescriptionStructured non-empty")
	}
	var got careIntensityTransitionDescription
	if err := json.Unmarshal(ev.DescriptionStructured, &got); err != nil {
		t.Fatalf("unmarshal description: %v", err)
	}
	if got.From != models.CareIntensityTagActiveTreatment || got.To != models.CareIntensityTagPalliative {
		t.Errorf("description from/to drift: %+v", got)
	}
	if len(got.Cascades) != 3 {
		t.Errorf("description cascades count drift: %d", len(got.Cascades))
	}
}

func TestCareIntensityEngine_OnTransition_ActiveToRehab_ProducesNoCascades(t *testing.T) {
	eng := NewCareIntensityEngine()
	ev, cascades := eng.OnTransition(
		models.CareIntensityTagActiveTreatment,
		models.CareIntensityTagRehabilitation,
		uuid.New(),
		uuid.New(),
	)
	if len(cascades) != 0 {
		t.Errorf("expected 0 cascades for active→rehabilitation; got %+v", cascades)
	}
	if ev.Severity != models.EventSeverityMinor {
		t.Errorf("expected severity=minor for active→rehabilitation; got %s", ev.Severity)
	}
	if ev.EventType != models.EventTypeCareIntensityTransition {
		t.Errorf("expected event_type=care_intensity_transition; got %s", ev.EventType)
	}
}

func TestCareIntensityEngine_OnTransition_PalliativeToComfortFocused_ProducesOneCascade(t *testing.T) {
	eng := NewCareIntensityEngine()
	ev, cascades := eng.OnTransition(
		models.CareIntensityTagPalliative,
		models.CareIntensityTagComfortFocused,
		uuid.New(),
		uuid.New(),
	)
	if len(cascades) != 1 {
		t.Fatalf("expected 1 cascade for palliative→comfort_focused (specific rule); got %d: %+v",
			len(cascades), cascades)
	}
	if cascades[0].Kind != CareIntensityCascadeRevisitMonitoringPlan {
		t.Errorf("expected revisit_monitoring_plan; got %s", cascades[0].Kind)
	}
	// Severity is moderate because target is comfort_focused (the engine
	// keys severity on the target tag, not the source).
	if ev.Severity != models.EventSeverityModerate {
		t.Errorf("expected severity=moderate for target=comfort_focused; got %s", ev.Severity)
	}
}

func TestCareIntensityEngine_OnTransition_EmptyFromToPalliative_MatchesGenericRule(t *testing.T) {
	eng := NewCareIntensityEngine()
	_, cascades := eng.OnTransition(
		"", // first-ever tagging
		models.CareIntensityTagPalliative,
		uuid.New(),
		uuid.New(),
	)
	if len(cascades) != 3 {
		t.Errorf("expected 3 cascades for empty→palliative (generic rule); got %d", len(cascades))
	}
}

func TestCareIntensityEngine_OnTransition_AnyToComfortFocused_ProducesTwoCascades(t *testing.T) {
	eng := NewCareIntensityEngine()
	_, cascades := eng.OnTransition(
		models.CareIntensityTagActiveTreatment,
		models.CareIntensityTagComfortFocused,
		uuid.New(),
		uuid.New(),
	)
	if len(cascades) != 2 {
		t.Errorf("expected 2 cascades for active→comfort_focused; got %d: %+v", len(cascades), cascades)
	}
}

func TestCareIntensityEngine_OnTransition_RehabToActive_ProducesNoCascades(t *testing.T) {
	eng := NewCareIntensityEngine()
	_, cascades := eng.OnTransition(
		models.CareIntensityTagRehabilitation,
		models.CareIntensityTagActiveTreatment,
		uuid.New(),
		uuid.New(),
	)
	if len(cascades) != 0 {
		t.Errorf("expected 0 cascades for rehabilitation→active_treatment; got %+v", cascades)
	}
}
