package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestActiveConcernJSONRoundTrip(t *testing.T) {
	startedBy := uuid.New()
	owner := uuid.New()
	resolved := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	in := ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          ActiveConcernPostFall72h,
		StartedAt:            time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		StartedByEventRef:    &startedBy,
		ExpectedResolutionAt: time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC),
		OwnerRoleRef:         &owner,
		ResolutionStatus:     ResolutionStatusResolvedStopCriteria,
		ResolvedAt:           &resolved,
		Notes:                "Vitals stable post-fall",
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ActiveConcern
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ConcernType != in.ConcernType {
		t.Errorf("ConcernType drift: got %s want %s", out.ConcernType, in.ConcernType)
	}
	if out.StartedByEventRef == nil || *out.StartedByEventRef != startedBy {
		t.Errorf("StartedByEventRef drift")
	}
	if out.OwnerRoleRef == nil || *out.OwnerRoleRef != owner {
		t.Errorf("OwnerRoleRef drift")
	}
	if out.ResolvedAt == nil || !out.ResolvedAt.Equal(resolved) {
		t.Errorf("ResolvedAt drift: got %v want %v", out.ResolvedAt, resolved)
	}
}

func TestActiveConcernOmitsEmptyOptionalFields(t *testing.T) {
	in := ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          ActiveConcernAcuteInfectionActive,
		StartedAt:            time.Now().UTC(),
		ExpectedResolutionAt: time.Now().UTC().Add(72 * time.Hour),
		ResolutionStatus:     ResolutionStatusOpen,
	}
	b, _ := json.Marshal(in)
	s := string(b)
	for _, k := range []string{
		`"started_by_event_ref"`, `"owner_role_ref"`,
		`"related_monitoring_plan_ref"`, `"resolved_at"`,
		`"resolution_evidence_trace_ref"`, `"notes"`,
	} {
		if strings.Contains(s, k) {
			t.Errorf("expected %s to be omitted, got: %s", k, s)
		}
	}
}

func TestIsValidActiveConcernType(t *testing.T) {
	for _, ct := range []string{
		ActiveConcernPostFall72h,
		ActiveConcernPostFall24h,
		ActiveConcernPostHospitalDischarge72h,
		ActiveConcernAntibioticCourseActive,
		ActiveConcernNewPsychotropicTitration,
		ActiveConcernAcuteInfectionActive,
		ActiveConcernEndOfLifeRecognition,
		ActiveConcernPostDeprescribingMonitoring,
		ActiveConcernPreEventWarning,
		ActiveConcernAwaitingConsentReview,
		ActiveConcernAwaitingSpecialistInput,
	} {
		if !IsValidActiveConcernType(ct) {
			t.Errorf("expected %q to be valid", ct)
		}
	}
	for _, ct := range []string{"", "unknown_concern", "fall"} {
		if IsValidActiveConcernType(ct) {
			t.Errorf("expected %q to be invalid", ct)
		}
	}
}

func TestIsValidResolutionStatus(t *testing.T) {
	for _, s := range []string{
		ResolutionStatusOpen, ResolutionStatusResolvedStopCriteria,
		ResolutionStatusEscalated, ResolutionStatusExpiredUnresolved,
	} {
		if !IsValidResolutionStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if IsValidResolutionStatus("closed") {
		t.Errorf("expected 'closed' to be invalid")
	}
	if IsValidResolutionStatus("") {
		t.Errorf("expected empty to be invalid")
	}
}

func TestIsTerminalResolutionStatus(t *testing.T) {
	if IsTerminalResolutionStatus(ResolutionStatusOpen) {
		t.Errorf("open is not terminal")
	}
	for _, s := range []string{
		ResolutionStatusResolvedStopCriteria,
		ResolutionStatusEscalated,
		ResolutionStatusExpiredUnresolved,
	} {
		if !IsTerminalResolutionStatus(s) {
			t.Errorf("%s is terminal", s)
		}
	}
}

func TestIsValidResolutionTransition(t *testing.T) {
	// Legal: open → any other terminal status
	for _, to := range []string{
		ResolutionStatusResolvedStopCriteria,
		ResolutionStatusEscalated,
		ResolutionStatusExpiredUnresolved,
	} {
		if !IsValidResolutionTransition(ResolutionStatusOpen, to) {
			t.Errorf("expected open→%s to be valid", to)
		}
	}
	// Illegal: terminal → anything
	for _, from := range []string{
		ResolutionStatusResolvedStopCriteria,
		ResolutionStatusEscalated,
		ResolutionStatusExpiredUnresolved,
	} {
		if IsValidResolutionTransition(from, ResolutionStatusOpen) {
			t.Errorf("expected %s→open to be invalid (terminal source)", from)
		}
		if IsValidResolutionTransition(from, ResolutionStatusResolvedStopCriteria) {
			t.Errorf("expected %s→resolved_stop_criteria to be invalid (terminal source)", from)
		}
	}
	// Illegal: self-transition
	if IsValidResolutionTransition(ResolutionStatusOpen, ResolutionStatusOpen) {
		t.Errorf("expected open→open to be invalid (no-op)")
	}
	// Illegal: unknown values
	if IsValidResolutionTransition("bogus", ResolutionStatusOpen) {
		t.Errorf("expected unknown→open to be invalid")
	}
	if IsValidResolutionTransition(ResolutionStatusOpen, "bogus") {
		t.Errorf("expected open→unknown to be invalid")
	}
}

func TestConcernExpiredUnresolvedIsSystemEvent(t *testing.T) {
	if !IsValidEventType(EventTypeConcernExpiredUnresolved) {
		t.Errorf("expected concern_expired_unresolved to be a valid event type")
	}
	if !IsSystemEventType(EventTypeConcernExpiredUnresolved) {
		t.Errorf("expected concern_expired_unresolved to route to System bucket (FHIR Communication)")
	}
}
