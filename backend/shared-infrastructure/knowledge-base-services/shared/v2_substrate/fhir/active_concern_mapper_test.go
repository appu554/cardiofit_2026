package fhir

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestActiveConcernToFHIRCondition_Open_RoundTrip(t *testing.T) {
	startedBy := uuid.New()
	owner := uuid.New()
	monPlan := uuid.New()
	in := models.ActiveConcern{
		ID:                       uuid.New(),
		ResidentID:               uuid.New(),
		ConcernType:              models.ActiveConcernPostFall72h,
		StartedAt:                time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		StartedByEventRef:        &startedBy,
		ExpectedResolutionAt:     time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC),
		OwnerRoleRef:             &owner,
		RelatedMonitoringPlanRef: &monPlan,
		ResolutionStatus:         models.ResolutionStatusOpen,
		Notes:                    "Vitals q4h × 72h",
	}
	cond, err := ActiveConcernToFHIRCondition(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	if cond["resourceType"] != "Condition" {
		t.Errorf("resourceType: got %v want Condition", cond["resourceType"])
	}
	out, err := FHIRConditionToActiveConcern(reMarshal(t, cond))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.ConcernType != in.ConcernType {
		t.Errorf("ConcernType drift: got %s want %s", out.ConcernType, in.ConcernType)
	}
	if out.ResolutionStatus != in.ResolutionStatus {
		t.Errorf("ResolutionStatus drift: got %s want %s", out.ResolutionStatus, in.ResolutionStatus)
	}
	if !out.StartedAt.Equal(in.StartedAt) {
		t.Errorf("StartedAt drift: got %v want %v", out.StartedAt, in.StartedAt)
	}
	if !out.ExpectedResolutionAt.Equal(in.ExpectedResolutionAt) {
		t.Errorf("ExpectedResolutionAt drift: got %v want %v", out.ExpectedResolutionAt, in.ExpectedResolutionAt)
	}
	if out.StartedByEventRef == nil || *out.StartedByEventRef != startedBy {
		t.Errorf("StartedByEventRef drift")
	}
	if out.OwnerRoleRef == nil || *out.OwnerRoleRef != owner {
		t.Errorf("OwnerRoleRef drift")
	}
	if out.RelatedMonitoringPlanRef == nil || *out.RelatedMonitoringPlanRef != monPlan {
		t.Errorf("RelatedMonitoringPlanRef drift")
	}
	if out.Notes != in.Notes {
		t.Errorf("Notes drift: got %q want %q", out.Notes, in.Notes)
	}
}

func TestActiveConcernToFHIRCondition_ResolvedRoundTrip(t *testing.T) {
	resolved := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	traceRef := uuid.New()
	in := models.ActiveConcern{
		ID:                         uuid.New(),
		ResidentID:                 uuid.New(),
		ConcernType:                models.ActiveConcernAntibioticCourseActive,
		StartedAt:                  time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
		ExpectedResolutionAt:       time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
		ResolutionStatus:           models.ResolutionStatusResolvedStopCriteria,
		ResolvedAt:                 &resolved,
		ResolutionEvidenceTraceRef: &traceRef,
	}
	cond, err := ActiveConcernToFHIRCondition(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	// Verify clinicalStatus.coding[0].code == "resolved"
	if cs, ok := cond["clinicalStatus"].(map[string]interface{}); ok {
		if codings, ok := cs["coding"].([]map[string]interface{}); ok && len(codings) > 0 {
			if codings[0]["code"] != "resolved" {
				t.Errorf("clinicalStatus.code: got %v want resolved", codings[0]["code"])
			}
		}
	}
	if cond["abatementDateTime"] != resolved.Format(time.RFC3339) {
		t.Errorf("abatementDateTime drift")
	}

	out, err := FHIRConditionToActiveConcern(reMarshal(t, cond))
	if err != nil {
		t.Fatalf("ingress: %v", err)
	}
	if out.ResolutionStatus != models.ResolutionStatusResolvedStopCriteria {
		t.Errorf("ResolutionStatus drift: got %s", out.ResolutionStatus)
	}
	if out.ResolvedAt == nil || !out.ResolvedAt.Equal(resolved) {
		t.Errorf("ResolvedAt drift: got %v want %v", out.ResolvedAt, resolved)
	}
	if out.ResolutionEvidenceTraceRef == nil || *out.ResolutionEvidenceTraceRef != traceRef {
		t.Errorf("ResolutionEvidenceTraceRef drift")
	}
}

func TestActiveConcernToFHIRCondition_RejectsInvalid(t *testing.T) {
	bad := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          "made_up",
		StartedAt:            time.Now().UTC(),
		ExpectedResolutionAt: time.Now().UTC().Add(time.Hour),
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
	if _, err := ActiveConcernToFHIRCondition(bad); err == nil {
		t.Errorf("expected egress validation error")
	}
}

func TestFHIRConditionToActiveConcern_WrongResourceType(t *testing.T) {
	if _, err := FHIRConditionToActiveConcern(map[string]interface{}{"resourceType": "Patient"}); err == nil {
		t.Errorf("expected error for resourceType=Patient")
	}
}

func TestActiveConcernToFHIRCondition_WireFormatHasExtensions(t *testing.T) {
	in := models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          models.ActiveConcernAcuteInfectionActive,
		StartedAt:            time.Now().UTC(),
		ExpectedResolutionAt: time.Now().UTC().Add(72 * time.Hour),
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
	cond, err := ActiveConcernToFHIRCondition(in)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	b, _ := json.Marshal(cond)
	s := string(b)
	for _, must := range []string{
		ExtActiveConcernType,
		ExtActiveConcernResolutionStatus,
		ExtActiveConcernExpectedResolutionAt,
		`"resourceType":"Condition"`,
	} {
		if !strings.Contains(s, must) {
			t.Errorf("wire format missing %s; got: %s", must, s)
		}
	}
}
