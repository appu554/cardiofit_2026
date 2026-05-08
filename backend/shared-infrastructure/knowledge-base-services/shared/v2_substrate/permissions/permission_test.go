package permissions

import (
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
	if err := s.Validate(); err == nil {
		t.Error("PFA scope with nil Gate must fail Validate()")
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
	if err := s.Validate(); err == nil {
		t.Error("non-PFA scope with non-nil Gate must fail Validate()")
	}
}

func TestAggregationGate_Satisfied(t *testing.T) {
	gate := AggregationGate{
		MinObservations: 30,
		TimeWindow:      90 * 24 * time.Hour,
		DelayWindow:     30 * 24 * time.Hour,
	}
	now := time.Now().UTC()
	periodStart := now.Add(-100 * 24 * time.Hour) // 100 days ago — within 90d window

	// Happy path: 35 obs, period within window, delay elapsed
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
