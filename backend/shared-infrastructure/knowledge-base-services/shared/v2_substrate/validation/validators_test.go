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
