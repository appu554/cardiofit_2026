package validation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestValidateResidentRequiresGivenAndFamilyName(t *testing.T) {
	r := models.Resident{ID: uuid.New(), Status: models.ResidentStatusActive, CareIntensity: models.CareIntensityActive, FacilityID: uuid.New(), DOB: time.Now()}
	if err := ValidateResident(r); err == nil {
		t.Errorf("expected error for missing given_name + family_name; got nil")
	}
	r.GivenName = "X"
	r.FamilyName = "Y"
	if err := ValidateResident(r); err != nil {
		t.Errorf("expected pass for valid Resident; got %v", err)
	}
}

func TestValidateResidentChecksCareIntensity(t *testing.T) {
	r := models.Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y", DOB: time.Now(), FacilityID: uuid.New(), Status: models.ResidentStatusActive, CareIntensity: "wrong"}
	if err := ValidateResident(r); err == nil {
		t.Errorf("expected error for invalid care_intensity; got nil")
	}
}

func TestValidateResidentChecksIHIWhenPresent(t *testing.T) {
	r := models.Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y", DOB: time.Now(), FacilityID: uuid.New(), Status: models.ResidentStatusActive, CareIntensity: models.CareIntensityActive, IHI: "abc"}
	if err := ValidateResident(r); err == nil {
		t.Errorf("expected error for non-numeric IHI; got nil")
	}
	r.IHI = "8003608000000570" // 16 digits
	if err := ValidateResident(r); err != nil {
		t.Errorf("expected pass for valid 16-digit IHI; got %v", err)
	}
}

func TestValidatePersonRequiresGivenAndFamilyName(t *testing.T) {
	p := models.Person{ID: uuid.New()}
	if err := ValidatePerson(p); err == nil {
		t.Errorf("expected error for missing names; got nil")
	}
}

func TestValidatePersonChecksHPIIWhenPresent(t *testing.T) {
	p := models.Person{ID: uuid.New(), GivenName: "X", FamilyName: "Y", HPII: "abc"}
	if err := ValidatePerson(p); err == nil {
		t.Errorf("expected error for non-numeric HPII; got nil")
	}
	p.HPII = "8003614900000000" // 16 digits
	if err := ValidatePerson(p); err != nil {
		t.Errorf("expected pass for valid 16-digit HPII; got %v", err)
	}
}

func TestValidateRoleChecksKind(t *testing.T) {
	r := models.Role{ID: uuid.New(), PersonID: uuid.New(), Kind: "nurse", ValidFrom: time.Now()}
	if err := ValidateRole(r); err == nil {
		t.Errorf("expected error for invalid Kind=nurse; got nil")
	}
	r.Kind = models.RoleRN
	if err := ValidateRole(r); err != nil {
		t.Errorf("expected pass for Kind=RN; got %v", err)
	}
}

func TestValidateRoleChecksValidityWindow(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-24 * time.Hour)
	r := models.Role{ID: uuid.New(), PersonID: uuid.New(), Kind: models.RoleRN, ValidFrom: now, ValidTo: &earlier}
	if err := ValidateRole(r); err == nil {
		t.Errorf("expected error when ValidTo < ValidFrom; got nil")
	}
}

func TestValidateMedicineUseRequiresFields(t *testing.T) {
	base := models.MedicineUse{
		ID: uuid.New(), ResidentID: uuid.New(),
		DisplayName:  "X",
		Intent:       models.Intent{Category: models.IntentTherapeutic, Indication: "y"},
		Target:       models.Target{Kind: models.TargetKindOpen, Spec: json.RawMessage(`{}`)},
		StopCriteria: models.StopCriteria{Triggers: []string{}},
		StartedAt:    time.Now(), Status: models.MedicineUseStatusActive,
	}
	if err := ValidateMedicineUse(base); err != nil {
		t.Errorf("expected pass for valid base; got %v", err)
	}

	bad := base
	bad.DisplayName = ""
	if err := ValidateMedicineUse(bad); err == nil {
		t.Errorf("expected error for missing display_name")
	}

	bad = base
	bad.Status = "wrong"
	if err := ValidateMedicineUse(bad); err == nil {
		t.Errorf("expected error for invalid status")
	}

	bad = base
	bad.Intent.Category = "wrong"
	if err := ValidateMedicineUse(bad); err == nil {
		t.Errorf("expected error for invalid intent.category")
	}

	bad = base
	bad.StopCriteria.Triggers = []string{"unknown_trigger"}
	if err := ValidateMedicineUse(bad); err == nil {
		t.Errorf("expected error for invalid stop trigger")
	}
}

func TestValidateMedicineUseEndedAtAfterStartedAt(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-24 * time.Hour)
	in := models.MedicineUse{
		ID: uuid.New(), ResidentID: uuid.New(),
		DisplayName:  "X",
		Intent:       models.Intent{Category: models.IntentTherapeutic, Indication: "y"},
		Target:       models.Target{Kind: models.TargetKindOpen, Spec: json.RawMessage(`{}`)},
		StopCriteria: models.StopCriteria{Triggers: []string{}},
		StartedAt:    now,
		EndedAt:      &earlier,
		Status:       models.MedicineUseStatusActive,
	}
	if err := ValidateMedicineUse(in); err == nil {
		t.Errorf("expected error when ended_at < started_at")
	}
}

func TestValidateTargetBPThresholdSpec(t *testing.T) {
	valid, _ := json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90})
	if err := ValidateTarget(models.Target{Kind: models.TargetKindBPThreshold, Spec: valid}); err != nil {
		t.Errorf("expected pass: %v", err)
	}
	bad, _ := json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 80, DiastolicMax: 90})
	if err := ValidateTarget(models.Target{Kind: models.TargetKindBPThreshold, Spec: bad}); err == nil {
		t.Errorf("expected error when systolic_max < diastolic_max")
	}
	bad, _ = json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 500, DiastolicMax: 90})
	if err := ValidateTarget(models.Target{Kind: models.TargetKindBPThreshold, Spec: bad}); err == nil {
		t.Errorf("expected error when systolic_max > 300")
	}
}

func TestValidateTargetCompletionDateSpec(t *testing.T) {
	valid, _ := json.Marshal(models.TargetCompletionDateSpec{
		EndDate: time.Now().Add(7 * 24 * time.Hour), DurationDays: 7,
	})
	if err := ValidateTarget(models.Target{Kind: models.TargetKindCompletionDate, Spec: valid}); err != nil {
		t.Errorf("expected pass: %v", err)
	}
	bad := json.RawMessage(`{"duration_days": 7}`)
	if err := ValidateTarget(models.Target{Kind: models.TargetKindCompletionDate, Spec: bad}); err == nil {
		t.Errorf("expected error for missing end_date")
	}
}

func TestValidateTargetHbA1cBandSpec(t *testing.T) {
	valid, _ := json.Marshal(models.TargetHbA1cBandSpec{Min: 6.5, Max: 8.0})
	if err := ValidateTarget(models.Target{Kind: models.TargetKindHbA1cBand, Spec: valid}); err != nil {
		t.Errorf("expected pass: %v", err)
	}
	bad, _ := json.Marshal(models.TargetHbA1cBandSpec{Min: 8.0, Max: 6.5})
	if err := ValidateTarget(models.Target{Kind: models.TargetKindHbA1cBand, Spec: bad}); err == nil {
		t.Errorf("expected error when min >= max")
	}
}

func TestValidateTargetUnknownKind(t *testing.T) {
	if err := ValidateTarget(models.Target{Kind: "LDL_target", Spec: json.RawMessage(`{}`)}); err == nil {
		t.Errorf("expected error for unrecognized target kind")
	}
}

func TestValidateObservationRequiresValueOrText(t *testing.T) {
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindVital,
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err == nil {
		t.Errorf("expected error when both Value and ValueText empty; got nil")
	}
}

func TestValidateObservationAcceptsValueOnly(t *testing.T) {
	v := 120.0
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindVital,
		Value:      &v,
		Unit:       "mmHg",
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err != nil {
		t.Errorf("expected pass for valid vital observation; got %v", err)
	}
}

func TestValidateObservationAcceptsValueTextOnly(t *testing.T) {
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindBehavioural,
		ValueText:  "agitation episode",
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err != nil {
		t.Errorf("expected pass for behavioural with ValueText only; got %v", err)
	}
}

func TestValidateObservationRejectsInvalidKind(t *testing.T) {
	v := 1.0
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       "behavioral", // US spelling
		Value:      &v,
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err == nil {
		t.Errorf("expected error for invalid kind; got nil")
	}
}

func TestValidateObservationRejectsZeroResidentID(t *testing.T) {
	v := 1.0
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.Nil,
		Kind:       models.ObservationKindLab,
		Value:      &v,
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err == nil {
		t.Errorf("expected error for zero resident_id; got nil")
	}
}

func TestValidateObservationRejectsZeroObservedAt(t *testing.T) {
	v := 1.0
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindLab,
		Value:      &v,
		ObservedAt: time.Time{},
	}
	if err := ValidateObservation(o); err == nil {
		t.Errorf("expected error for zero observed_at; got nil")
	}
}

func TestValidateObservationVitalRange(t *testing.T) {
	bad := 999.0
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindVital,
		LOINCCode:  "8480-6", // systolic BP
		Value:      &bad,
		Unit:       "mmHg",
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err == nil {
		t.Errorf("expected error for BP=999; got nil")
	}
	good := 130.0
	o.Value = &good
	if err := ValidateObservation(o); err != nil {
		t.Errorf("expected pass for BP=130; got %v", err)
	}
}

func TestValidateObservationWeightPositive(t *testing.T) {
	bad := 0.0
	o := models.Observation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		Kind:       models.ObservationKindWeight,
		Value:      &bad,
		Unit:       "kg",
		ObservedAt: time.Now(),
	}
	if err := ValidateObservation(o); err == nil {
		t.Errorf("expected error for weight=0; got nil")
	}
}

// ---------------------------------------------------------------------------
// Event validator
// ---------------------------------------------------------------------------

func validBaseEvent(eventType string) models.Event {
	return models.Event{
		ID:            uuid.New(),
		EventType:     eventType,
		OccurredAt:    time.Now().UTC(),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
}

func TestValidateEventUniversal_PassesWithMinimumFields(t *testing.T) {
	e := validBaseEvent(models.EventTypeGPVisit) // no per-type rules
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass for GP_visit minimum; got %v", err)
	}
}

func TestValidateEventUniversal_RejectsZeroResidentID(t *testing.T) {
	e := validBaseEvent(models.EventTypeGPVisit)
	e.ResidentID = uuid.Nil
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for zero resident_id")
	}
}

func TestValidateEventUniversal_RejectsInvalidEventType(t *testing.T) {
	e := validBaseEvent(models.EventTypeGPVisit)
	e.EventType = "not_a_real_event"
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for invalid event_type")
	}
}

func TestValidateEventUniversal_RejectsZeroOccurredAt(t *testing.T) {
	e := validBaseEvent(models.EventTypeGPVisit)
	e.OccurredAt = time.Time{}
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for zero occurred_at")
	}
}

func TestValidateEventUniversal_RejectsZeroReportedByRef(t *testing.T) {
	e := validBaseEvent(models.EventTypeGPVisit)
	e.ReportedByRef = uuid.Nil
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for zero reported_by_ref")
	}
}

func TestValidateEventUniversal_RejectsInvalidSeverity(t *testing.T) {
	e := validBaseEvent(models.EventTypeGPVisit)
	e.Severity = "critical"
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for invalid severity")
	}
}

func TestValidateEventUniversal_RejectsInvalidStateMachine(t *testing.T) {
	e := validBaseEvent(models.EventTypeRecommendationDecided)
	e.TriggeredStateChanges = []models.TriggeredStateChange{
		{StateMachine: "Bogus", StateChange: json.RawMessage(`{}`)},
	}
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for invalid state_machine")
	}
}

func TestValidateEventFall_RequiresSeverity(t *testing.T) {
	e := validBaseEvent(models.EventTypeFall)
	// no severity
	e.WitnessedByRefs = []uuid.UUID{uuid.New()}
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for fall without severity")
	}
	e.Severity = models.EventSeverityModerate
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass once severity + witnessed_by_refs set; got %v", err)
	}
}

func TestValidateEventFall_RequiresWitnessedOrDescription(t *testing.T) {
	e := validBaseEvent(models.EventTypeFall)
	e.Severity = models.EventSeverityMinor
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error for fall with no witnessed_by_refs and no description_structured")
	}
	// either alone passes
	e.DescriptionStructured = json.RawMessage(`{"location":"bathroom"}`)
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass with description_structured only; got %v", err)
	}
	e.DescriptionStructured = nil
	e.WitnessedByRefs = []uuid.UUID{uuid.New()}
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass with witnessed_by_refs only; got %v", err)
	}
}

func TestValidateEventMedicationError_RequiresSeverityAndRelatedMed(t *testing.T) {
	e := validBaseEvent(models.EventTypeMedicationError)
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error: missing severity + related_medication_uses")
	}
	e.Severity = models.EventSeverityMajor
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error: missing related_medication_uses")
	}
	e.RelatedMedicationUses = []uuid.UUID{uuid.New()}
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

func TestValidateEventAdverseDrugEvent_RequiresRelatedMed(t *testing.T) {
	e := validBaseEvent(models.EventTypeAdverseDrugEvent)
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error: missing related_medication_uses")
	}
	e.RelatedMedicationUses = []uuid.UUID{uuid.New()}
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass once related_medication_uses set; got %v", err)
	}
}

func TestValidateEventDeath_UniversalOnly(t *testing.T) {
	e := validBaseEvent(models.EventTypeDeath)
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass for death with universal fields; got %v", err)
	}
}

func TestValidateEventHospitalAdmission_RequiresSeverityAndDescription(t *testing.T) {
	e := validBaseEvent(models.EventTypeHospitalAdmission)
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error: missing severity + description")
	}
	e.Severity = models.EventSeverityMajor
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error: missing description_structured")
	}
	e.DescriptionStructured = json.RawMessage(`{"facility":"acme"}`)
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

func TestValidateEventHospitalDischarge_RequiresSeverityAndDescription(t *testing.T) {
	e := validBaseEvent(models.EventTypeHospitalDischarge)
	e.Severity = models.EventSeverityMinor
	if err := ValidateEvent(e); err == nil {
		t.Errorf("expected error: missing description_structured")
	}
	e.DescriptionStructured = json.RawMessage(`{"summary":"stable"}`)
	if err := ValidateEvent(e); err != nil {
		t.Errorf("expected pass; got %v", err)
	}
}

func TestValidateEventSystemEvent_UniversalOnly(t *testing.T) {
	for _, tp := range []string{
		models.EventTypeRuleFire,
		models.EventTypeRecommendationSubmitted,
		models.EventTypeMonitoringPlanActivated,
		models.EventTypeConsentGrantedOrWithdrawn,
		models.EventTypeCredentialVerifiedOrExpired,
	} {
		e := validBaseEvent(tp)
		if err := ValidateEvent(e); err != nil {
			t.Errorf("%s: expected pass with universal fields; got %v", tp, err)
		}
	}
}

// ----------------------------------------------------------------------------
// EvidenceTraceNode validation
// ----------------------------------------------------------------------------

func validBaseEvidenceTraceNode() models.EvidenceTraceNode {
	now := time.Now().UTC()
	return models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: "draft -> submitted",
		RecordedAt:      now,
		OccurredAt:      now,
	}
}

func TestValidateEvidenceTraceNode_Universal(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	if err := ValidateEvidenceTraceNode(n); err != nil {
		t.Errorf("expected pass with universal fields; got %v", err)
	}
}

func TestValidateEvidenceTraceNode_RejectsInvalidStateMachine(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.StateMachine = "Other"
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for invalid state_machine")
	}
}

func TestValidateEvidenceTraceNode_RequiresStateChangeType(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.StateChangeType = ""
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for missing state_change_type")
	}
}

func TestValidateEvidenceTraceNode_RequiresRecordedAt(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.RecordedAt = time.Time{}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for zero recorded_at")
	}
}

func TestValidateEvidenceTraceNode_RequiresOccurredAt(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.OccurredAt = time.Time{}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for zero occurred_at")
	}
}

func TestValidateEvidenceTraceNode_InputsRequireType(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.Inputs = []models.TraceInput{
		{InputType: "", InputRef: uuid.New(), RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
	}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for empty input_type")
	}
}

func TestValidateEvidenceTraceNode_InputsRequireRef(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.Inputs = []models.TraceInput{
		{InputType: models.TraceInputTypeObservation, InputRef: uuid.Nil, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
	}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for zero input_ref")
	}
}

func TestValidateEvidenceTraceNode_InputsValidateRoleInDecision(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.Inputs = []models.TraceInput{
		{InputType: models.TraceInputTypeObservation, InputRef: uuid.New(), RoleInDecision: "primary"},
	}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for invalid role_in_decision (must be primary_evidence)")
	}
}

func TestValidateEvidenceTraceNode_OutputsRequireFields(t *testing.T) {
	n := validBaseEvidenceTraceNode()
	n.Outputs = []models.TraceOutput{{OutputType: "", OutputRef: uuid.New()}}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for empty output_type")
	}
	n.Outputs = []models.TraceOutput{{OutputType: "Recommendation", OutputRef: uuid.Nil}}
	if err := ValidateEvidenceTraceNode(n); err == nil {
		t.Error("expected error for zero output_ref")
	}
}

func TestValidateEvidenceTraceNode_AllowsNoResidentRef(t *testing.T) {
	// System-only node (rule_fire on global config, credential check) has no resident.
	n := validBaseEvidenceTraceNode()
	n.StateMachine = models.EvidenceTraceStateMachineAuthorisation
	n.ResidentRef = nil
	if err := ValidateEvidenceTraceNode(n); err != nil {
		t.Errorf("expected pass for system-only node; got %v", err)
	}
}

func TestValidateEvidenceTraceNode_AcceptsAllStateMachines(t *testing.T) {
	for _, sm := range []string{
		models.EvidenceTraceStateMachineAuthorisation,
		models.EvidenceTraceStateMachineRecommendation,
		models.EvidenceTraceStateMachineMonitoring,
		models.EvidenceTraceStateMachineClinicalState,
		models.EvidenceTraceStateMachineConsent,
	} {
		n := validBaseEvidenceTraceNode()
		n.StateMachine = sm
		if err := ValidateEvidenceTraceNode(n); err != nil {
			t.Errorf("%s: expected pass; got %v", sm, err)
		}
	}
}

// ============================================================================
// Wave 2.3: ActiveConcern validators
// ============================================================================

func validBaseActiveConcern() models.ActiveConcern {
	started := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	return models.ActiveConcern{
		ID:                   uuid.New(),
		ResidentID:           uuid.New(),
		ConcernType:          models.ActiveConcernPostFall72h,
		StartedAt:            started,
		ExpectedResolutionAt: started.Add(72 * time.Hour),
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
}

func TestValidateActiveConcernRequiresResidentID(t *testing.T) {
	c := validBaseActiveConcern()
	c.ResidentID = uuid.Nil
	if err := ValidateActiveConcern(c); err == nil {
		t.Errorf("expected error for missing resident_id")
	}
}

func TestValidateActiveConcernRejectsInvalidType(t *testing.T) {
	c := validBaseActiveConcern()
	c.ConcernType = "made_up"
	if err := ValidateActiveConcern(c); err == nil {
		t.Errorf("expected error for invalid concern_type")
	}
}

func TestValidateActiveConcernRequiresExpectedResolutionAfterStarted(t *testing.T) {
	c := validBaseActiveConcern()
	c.ExpectedResolutionAt = c.StartedAt
	if err := ValidateActiveConcern(c); err == nil {
		t.Errorf("expected error when expected_resolution_at == started_at")
	}
	c.ExpectedResolutionAt = c.StartedAt.Add(-time.Hour)
	if err := ValidateActiveConcern(c); err == nil {
		t.Errorf("expected error when expected_resolution_at < started_at")
	}
}

func TestValidateActiveConcernRejectsInvalidStatus(t *testing.T) {
	c := validBaseActiveConcern()
	c.ResolutionStatus = "settled"
	if err := ValidateActiveConcern(c); err == nil {
		t.Errorf("expected error for invalid resolution_status")
	}
}

func TestValidateActiveConcernOpenForbidsResolvedAt(t *testing.T) {
	c := validBaseActiveConcern()
	now := c.StartedAt.Add(time.Hour)
	c.ResolvedAt = &now
	if err := ValidateActiveConcern(c); err == nil {
		t.Errorf("expected error: open concern must not have resolved_at")
	}
}

func TestValidateActiveConcernTerminalRequiresResolvedAt(t *testing.T) {
	for _, term := range []string{
		models.ResolutionStatusResolvedStopCriteria,
		models.ResolutionStatusEscalated,
		models.ResolutionStatusExpiredUnresolved,
	} {
		c := validBaseActiveConcern()
		c.ResolutionStatus = term
		c.ResolvedAt = nil
		if err := ValidateActiveConcern(c); err == nil {
			t.Errorf("%s: expected error when resolved_at is missing", term)
		}
		// Set resolved_at before started_at — also rejected.
		earlier := c.StartedAt.Add(-time.Minute)
		c.ResolvedAt = &earlier
		if err := ValidateActiveConcern(c); err == nil {
			t.Errorf("%s: expected error when resolved_at < started_at", term)
		}
		// Valid case.
		later := c.StartedAt.Add(time.Hour)
		c.ResolvedAt = &later
		if err := ValidateActiveConcern(c); err != nil {
			t.Errorf("%s: expected pass with resolved_at >= started_at; got %v", term, err)
		}
	}
}

func TestValidateActiveConcernResolutionTransition(t *testing.T) {
	// Legal
	if err := ValidateActiveConcernResolutionTransition(
		models.ResolutionStatusOpen, models.ResolutionStatusResolvedStopCriteria,
	); err != nil {
		t.Errorf("expected open→resolved_stop_criteria to pass; got %v", err)
	}
	if err := ValidateActiveConcernResolutionTransition(
		models.ResolutionStatusOpen, models.ResolutionStatusEscalated,
	); err != nil {
		t.Errorf("expected open→escalated to pass; got %v", err)
	}
	if err := ValidateActiveConcernResolutionTransition(
		models.ResolutionStatusOpen, models.ResolutionStatusExpiredUnresolved,
	); err != nil {
		t.Errorf("expected open→expired_unresolved to pass; got %v", err)
	}
	// Illegal: terminal source
	if err := ValidateActiveConcernResolutionTransition(
		models.ResolutionStatusExpiredUnresolved, models.ResolutionStatusOpen,
	); err == nil {
		t.Errorf("expected expired_unresolved→open to fail")
	}
	if err := ValidateActiveConcernResolutionTransition(
		models.ResolutionStatusResolvedStopCriteria, models.ResolutionStatusEscalated,
	); err == nil {
		t.Errorf("expected resolved_stop_criteria→escalated to fail")
	}
	// Illegal: invalid current/target
	if err := ValidateActiveConcernResolutionTransition("bogus", models.ResolutionStatusOpen); err == nil {
		t.Errorf("expected invalid source to fail")
	}
	if err := ValidateActiveConcernResolutionTransition(models.ResolutionStatusOpen, "bogus"); err == nil {
		t.Errorf("expected invalid target to fail")
	}
}
