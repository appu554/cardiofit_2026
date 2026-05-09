package permissions

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestViewPermissionAllows_POA_SubjectOnly(t *testing.T) {
	subject := uuid.New()
	other := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: subject, // pharmacist viewing own POA data
		Scope: Scope{
			Class:         POA,
			ResourceTypes: []string{"reflective_writing"},
		},
		GrantedAt: time.Now().UTC(),
	}
	if !p.Allows("reflective_writing", subject) {
		t.Errorf("POA: expected allow when viewerRoleID == subjectID")
	}
	// Reassign viewer to a different role; POA must deny
	p.ViewerRoleID = other
	if p.Allows("reflective_writing", subject) {
		t.Errorf("POA: expected deny when viewerRoleID != subjectID")
	}
}

func TestViewPermissionAllows_PDP_RequiresConsent(t *testing.T) {
	subject := uuid.New()
	employer := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: employer,
		Scope: Scope{
			Class:         PDP,
			ResourceTypes: []string{"rir_summary"},
		},
		GrantedAt: time.Now().UTC(),
	}
	// Non-subject viewer without consent path — Allows() enforces subject==viewer for PDP
	if p.Allows("rir_summary", subject) {
		t.Errorf("PDP: expected deny when viewer != subject (consent check is in middleware)")
	}
	// Subject viewing their own PDP data is always allowed
	p.ViewerRoleID = subject
	if !p.Allows("rir_summary", subject) {
		t.Errorf("PDP: expected allow when viewer == subject")
	}
}

func TestViewPermissionAllows_WO_WorkflowParticipants(t *testing.T) {
	subject := uuid.New()
	viewer := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: viewer,
		Scope: Scope{
			Class:         WO,
			ResourceTypes: []string{"active_recommendations", "monitoring_obligations"},
		},
		GrantedAt: time.Now().UTC(),
	}
	if !p.Allows("active_recommendations", subject) {
		t.Errorf("WO: expected allow for workflow participant")
	}
	if p.Allows("reflective_writing", subject) {
		t.Errorf("WO: expected deny for resource not in scope")
	}
}

func TestScope_Validate_PFA_RequiresGate(t *testing.T) {
	s := Scope{Class: PFA, ResourceTypes: []string{"rir_summary"}}
	if err := s.Validate(); !errors.Is(err, ErrPFARequiresGate) {
		t.Errorf("PFA scope with nil Gate must return ErrPFARequiresGate, got %v", err)
	}
	gate := &AggregationGate{
		MinObservations:  30,
		TimeWindow:       90 * 24 * time.Hour,
		DelayWindow:      30 * 24 * time.Hour,
		ContractualBasis: "enterprise tier deployment §4.2",
		ExplicitNotice:   true,
	}
	s.Gate = gate
	if err := s.Validate(); err != nil {
		t.Errorf("PFA scope with valid Gate must pass Validate(): %v", err)
	}
}

func TestScope_Validate_NonPFA_GateMustBeNil(t *testing.T) {
	gate := &AggregationGate{MinObservations: 30}
	s := Scope{Class: POA, ResourceTypes: []string{"reflective_writing"}, Gate: gate}
	if err := s.Validate(); !errors.Is(err, ErrNonPFAGateMustBeNil) {
		t.Errorf("non-PFA scope with non-nil Gate must return ErrNonPFAGateMustBeNil, got %v", err)
	}
}

func TestScope_Validate_RejectsUnsetClass(t *testing.T) {
	s := Scope{ResourceTypes: []string{"x"}} // Class is zero value VisibilityClassUnset
	if err := s.Validate(); !errors.Is(err, ErrMissingVisibilityClass) {
		t.Errorf("expected ErrMissingVisibilityClass, got %v", err)
	}
}

func TestScope_Validate_RejectsEmptyResourceTypes(t *testing.T) {
	s := Scope{Class: POA}
	if err := s.Validate(); !errors.Is(err, ErrEmptyResourceTypes) {
		t.Errorf("expected ErrEmptyResourceTypes, got %v", err)
	}
}

func TestAggregationGate_Satisfied(t *testing.T) {
	gate := AggregationGate{
		MinObservations: 30,
		TimeWindow:      90 * 24 * time.Hour,
		DelayWindow:     30 * 24 * time.Hour,
	}
	now := time.Now().UTC()
	// 100 days ago — within combined 120d (TimeWindow + DelayWindow) lookback
	periodStart := now.Add(-100 * 24 * time.Hour)

	// Happy path: 35 obs, 100d period within 120d combined window, asOf is 70d after periodStart so DelayWindow elapsed.
	if !gate.Satisfied(35, now, periodStart) {
		t.Error("gate: expected satisfied with 35 obs, 100d period")
	}
	// Insufficient observations
	if gate.Satisfied(10, now, periodStart) {
		t.Error("gate: expected unsatisfied with only 10 obs")
	}
	// Outside time window (period started 200d ago)
	if gate.Satisfied(35, now, now.Add(-200*24*time.Hour)) {
		t.Error("gate: expected unsatisfied when period outside time window")
	}
	// Before delay window (asOf is only 10d after periodStart — less than 30d delay)
	if gate.Satisfied(35, periodStart.Add(10*24*time.Hour), periodStart) {
		t.Error("gate: expected unsatisfied before delay window elapses")
	}
}

func TestVisibilityClass_String(t *testing.T) {
	cases := []struct {
		c    VisibilityClass
		want string
	}{
		{VisibilityClassUnset, "unset"},
		{POA, "POA"},
		{PDP, "PDP"},
		{PFA, "PFA"},
		{WO, "WO"},
		{AD, "AD"},
		{VisibilityClass(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("VisibilityClass(%d).String() = %q, want %q", int(tc.c), got, tc.want)
		}
	}
}

func TestVisibilityClass_Valid(t *testing.T) {
	valid := []VisibilityClass{POA, PDP, PFA, WO, AD}
	for _, c := range valid {
		if !c.Valid() {
			t.Errorf("expected %v to be Valid()", c)
		}
	}
	invalid := []VisibilityClass{VisibilityClassUnset, VisibilityClass(99)}
	for _, c := range invalid {
		if c.Valid() {
			t.Errorf("expected %v to be invalid", c)
		}
	}
}
