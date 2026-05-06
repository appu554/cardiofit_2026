// Package models defines the v2 substrate entity types used across all
// Vaidshala KBs. Each entity is a clean Go struct optimized for Vaidshala's
// aged-care-medication-stewardship domain. FHIR translation happens at
// boundaries via the sibling fhir/ package.
package models

// CareIntensity classifies the resident's overall care plan posture. It
// shapes every recommendation downstream (palliative residents are not
// candidates for primary-prevention deprescribing, etc.).
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	CareIntensityPalliative     = "palliative"
	CareIntensityComfort        = "comfort"
	CareIntensityActive         = "active"
	CareIntensityRehabilitation = "rehabilitation"
)

// IsValidCareIntensity reports whether s is one of the recognized
// CareIntensity values.
func IsValidCareIntensity(s string) bool {
	switch s {
	case CareIntensityPalliative, CareIntensityComfort,
		CareIntensityActive, CareIntensityRehabilitation:
		return true
	}
	return false
}

// RoleKind enumerates the v2 actor types. The set mirrors the
// regulatory_scope_rules.role values authored in Phase 1C-γ — the
// Authorisation evaluator joins on these strings, so changes here MUST
// be coordinated with kb-22 ScopeRules data.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	RoleRN                  = "RN"
	RoleEN                  = "EN"
	RoleNP                  = "NP"
	RoleDRNP                = "DRNP"
	RoleGP                  = "GP"
	RolePharmacist          = "pharmacist"
	RoleACOP                = "ACOP"
	RolePCW                 = "PCW"
	RoleSDM                 = "SDM"
	RoleFamily              = "family"
	RoleATSIHP              = "ATSIHP"
	RoleMedicalPractitioner = "medical_practitioner"
	RoleDentist             = "dentist"
)

// IsValidRoleKind reports whether s is one of the recognized RoleKind values.
func IsValidRoleKind(s string) bool {
	switch s {
	case RoleRN, RoleEN, RoleNP, RoleDRNP, RoleGP, RolePharmacist,
		RoleACOP, RolePCW, RoleSDM, RoleFamily, RoleATSIHP,
		RoleMedicalPractitioner, RoleDentist:
		return true
	}
	return false
}

// ResidentStatus enumerates residency lifecycle.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	ResidentStatusActive      = "active"
	ResidentStatusDeceased    = "deceased"
	ResidentStatusTransferred = "transferred"
	ResidentStatusDischarged  = "discharged"
)

// IsValidResidentStatus reports whether s is one of the recognized
// ResidentStatus values.
func IsValidResidentStatus(s string) bool {
	switch s {
	case ResidentStatusActive, ResidentStatusDeceased,
		ResidentStatusTransferred, ResidentStatusDischarged:
		return true
	}
	return false
}

// MedicineUseStatus enumerates the lifecycle of a v2 MedicineUse row. Values
// are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	MedicineUseStatusActive    = "active"
	MedicineUseStatusPaused    = "paused"
	MedicineUseStatusCeased    = "ceased"
	MedicineUseStatusCompleted = "completed"
)

// IsValidMedicineUseStatus reports whether s is one of the recognized
// MedicineUseStatus values.
func IsValidMedicineUseStatus(s string) bool {
	switch s {
	case MedicineUseStatusActive, MedicineUseStatusPaused,
		MedicineUseStatusCeased, MedicineUseStatusCompleted:
		return true
	}
	return false
}

// Intent categories — describes WHY a medicine is used. Values are stored
// as plain strings to round-trip cleanly through FHIR code elements.
const (
	IntentTherapeutic   = "therapeutic"   // treating an active condition
	IntentPreventive    = "preventive"    // primary or secondary prevention
	IntentSymptomatic   = "symptomatic"   // PRN / symptom relief
	IntentTrial         = "trial"         // therapeutic trial period
	IntentDeprescribing = "deprescribing" // tapering / withdrawal
	// IntentUnspecified is a migration/legacy sentinel value used by the
	// medicine_uses_v2 view to backfill rows that never declared an intent.
	// It is NOT a substantive clinical claim. v2 writers MUST populate
	// Intent.Category with a real category (therapeutic, preventive,
	// symptomatic, trial, or deprescribing) — "unspecified" is accepted by
	// IsValidIntentCategory only so legacy reads round-trip cleanly through
	// validation; new writes that emit "unspecified" indicate a bug in the
	// caller, not a valid clinical state.
	IntentUnspecified = "unspecified" // legacy/migration default; v2 writers should not use this
)

// IsValidIntentCategory reports whether s is one of the recognized
// Intent.Category values.
func IsValidIntentCategory(s string) bool {
	switch s {
	case IntentTherapeutic, IntentPreventive, IntentSymptomatic,
		IntentTrial, IntentDeprescribing, IntentUnspecified:
		return true
	}
	return false
}

// Target kinds — discriminator for Target.Spec JSONB shape. Each kind has
// a documented spec struct in target_schemas.go.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	TargetKindBPThreshold       = "BP_threshold"       // antihypertensives
	TargetKindCompletionDate    = "completion_date"    // antibiotics, deprescribing
	TargetKindSymptomResolution = "symptom_resolution" // symptomatic
	TargetKindHbA1cBand         = "HbA1c_band"         // diabetes
	TargetKindOpen              = "open"               // chronic, no specific target
)

// IsValidTargetKind reports whether s is one of the recognized Target.Kind
// values. Adding a new kind requires also adding the spec struct in
// target_schemas.go and a delegated validator in target_validator.go.
func IsValidTargetKind(s string) bool {
	switch s {
	case TargetKindBPThreshold, TargetKindCompletionDate,
		TargetKindSymptomResolution, TargetKindHbA1cBand, TargetKindOpen:
		return true
	}
	return false
}

// StopTrigger enumerates the structured reasons a MedicineUse stop can be
// initiated. Stored in StopCriteria.Triggers []string.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	StopTriggerAdverseEvent   = "adverse_event"
	StopTriggerTargetAchieved = "target_achieved"
	StopTriggerReviewDue      = "review_due"
	StopTriggerPatientRequest = "patient_request"
	StopTriggerCarerRequest   = "carer_request"
	StopTriggerCompletion     = "completion"  // course completed (antibiotics, etc.)
	StopTriggerInteraction    = "interaction" // contraindicated by new medicine
)

// IsValidStopTrigger reports whether s is one of the recognized StopTrigger
// values.
func IsValidStopTrigger(s string) bool {
	switch s {
	case StopTriggerAdverseEvent, StopTriggerTargetAchieved,
		StopTriggerReviewDue, StopTriggerPatientRequest,
		StopTriggerCarerRequest, StopTriggerCompletion, StopTriggerInteraction:
		return true
	}
	return false
}

// ObservationKind discriminates the row kind in the observations table.
// vital — BP, HR, temp, SpO2; lab — eGFR, HbA1c (also surfaced from lab_entries);
// behavioural — BPSD events, agitation; mobility — mobility scores, falls;
// weight — weight, BMI.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	ObservationKindVital       = "vital"
	ObservationKindLab         = "lab"
	ObservationKindBehavioural = "behavioural"
	ObservationKindMobility    = "mobility"
	ObservationKindWeight      = "weight"
)

// IsValidObservationKind reports whether s is one of the recognized
// ObservationKind values. AU spelling ("behavioural") is canonical;
// US "behavioral" is intentionally rejected.
func IsValidObservationKind(s string) bool {
	switch s {
	case ObservationKindVital, ObservationKindLab,
		ObservationKindBehavioural, ObservationKindMobility, ObservationKindWeight:
		return true
	}
	return false
}

// DeltaFlag enumerates the directional flag emitted by the delta-on-write
// service. Threshold semantics live in shared/v2_substrate/delta/compute.go.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
	DeltaFlagWithinBaseline   = "within_baseline"
	DeltaFlagElevated         = "elevated"
	DeltaFlagSeverelyElevated = "severely_elevated"
	DeltaFlagLow              = "low"
	DeltaFlagSeverelyLow      = "severely_low"
	DeltaFlagNoBaseline       = "no_baseline"
)

// IsValidDeltaFlag reports whether s is one of the recognized DeltaFlag values.
func IsValidDeltaFlag(s string) bool {
	switch s {
	case DeltaFlagWithinBaseline, DeltaFlagElevated, DeltaFlagSeverelyElevated,
		DeltaFlagLow, DeltaFlagSeverelyLow, DeltaFlagNoBaseline:
		return true
	}
	return false
}
