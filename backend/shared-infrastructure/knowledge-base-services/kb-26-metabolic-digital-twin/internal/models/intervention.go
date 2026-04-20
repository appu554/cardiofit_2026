package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// InterventionCategory is the coarse taxonomy bucket an intervention falls into.
// Sprint 1 vocabulary covers HCF CHF and Aged Care AU. New categories require a
// market-config + clinical-lead review (spec §9 and §13 item 6).
type InterventionCategory string

const (
	CategoryFollowUp           InterventionCategory = "FOLLOW_UP"
	CategorySpecialistReferral InterventionCategory = "SPECIALIST_REFERRAL"
	CategoryMedicationReview   InterventionCategory = "MEDICATION_REVIEW"
	CategoryDeviceEnrolment    InterventionCategory = "DEVICE_ENROLMENT"
	CategoryAlliedHealth       InterventionCategory = "ALLIED_HEALTH"
	CategoryCarePlanning       InterventionCategory = "CARE_PLANNING"
)

var validInterventionCategories = map[InterventionCategory]struct{}{
	CategoryFollowUp:           {},
	CategorySpecialistReferral: {},
	CategoryMedicationReview:   {},
	CategoryDeviceEnrolment:    {},
	CategoryAlliedHealth:       {},
	CategoryCarePlanning:       {},
}

// EligibilityCriterion is a single feature-predicate test applied against the patient's
// consolidated pre-alert record. Multiple criteria compose as logical AND.
type EligibilityCriterion struct {
	FeatureKey string   `json:"feature_key"`
	Operator   string   `json:"operator"` // "gte", "lte", "eq", "in"
	Threshold  float64  `json:"threshold,omitempty"`
	Set        []string `json:"set,omitempty"`
}

// Contraindication is a hard disqualifier. If any matches the patient, the intervention
// is not CATE-scored and not recommended.
type Contraindication struct {
	FeatureKey string   `json:"feature_key"`
	Operator   string   `json:"operator"`
	Threshold  float64  `json:"threshold,omitempty"`
	Set        []string `json:"set,omitempty"`
	Reason     string   `json:"reason"`
}

// InterventionDefinition is a versioned, cohort-scoped intervention that the CATE engine
// can score. Persisted to DB on service start from market-config YAML.
type InterventionDefinition struct {
	ID                    string        `gorm:"primaryKey;size:80" json:"id"`
	CohortID              string        `gorm:"size:60;index;not null" json:"cohort_id"`
	Category              string        `gorm:"size:40;not null" json:"category"`
	Name                  string        `gorm:"size:200;not null" json:"name"`
	ClinicianLanguage     string        `gorm:"size:300" json:"clinician_language"`
	CoolDownHours         int           `json:"cool_down_hours"`
	ResourceCost          float64       `json:"resource_cost"`
	FeatureSignature      pq.StringArray `gorm:"type:text[]" json:"feature_signature"`
	EligibilityJSON       string        `gorm:"type:text" json:"-"`
	ContraindicationsJSON string        `gorm:"type:text" json:"-"`
	Version               string        `gorm:"size:20;not null;default:'1.0.0'" json:"version"`
	SourceYAMLPath        string        `gorm:"size:300" json:"source_yaml_path"`
	LoadedAt              time.Time     `gorm:"autoCreateTime" json:"loaded_at"`
	LedgerEntryID         *uuid.UUID    `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`
}

func (InterventionDefinition) TableName() string { return "intervention_definitions" }

// Validate checks that required fields are present and values are in their enums.
// Called on YAML load and on every CATE request.
func (d InterventionDefinition) Validate() error {
	if d.ID == "" {
		return errors.New("intervention ID required")
	}
	if d.CohortID == "" {
		return errors.New("cohort ID required")
	}
	if _, ok := validInterventionCategories[InterventionCategory(d.Category)]; !ok {
		return errors.New("unknown intervention category: " + d.Category)
	}
	if d.Name == "" {
		return errors.New("intervention name required")
	}
	return nil
}

// MarshalEligibility converts eligibility criteria to JSON for storage.
func (d *InterventionDefinition) MarshalEligibility(criteria []EligibilityCriterion) error {
	if len(criteria) == 0 {
		d.EligibilityJSON = "[]"
		return nil
	}
	b, err := json.Marshal(criteria)
	if err != nil {
		return err
	}
	d.EligibilityJSON = string(b)
	return nil
}

// UnmarshalEligibility converts stored JSON back to criteria.
func (d InterventionDefinition) UnmarshalEligibility() ([]EligibilityCriterion, error) {
	if d.EligibilityJSON == "" || d.EligibilityJSON == "[]" {
		return []EligibilityCriterion{}, nil
	}
	var criteria []EligibilityCriterion
	if err := json.Unmarshal([]byte(d.EligibilityJSON), &criteria); err != nil {
		return nil, err
	}
	return criteria, nil
}

// MarshalContraindications converts contraindications to JSON for storage.
func (d *InterventionDefinition) MarshalContraindications(contras []Contraindication) error {
	if len(contras) == 0 {
		d.ContraindicationsJSON = "[]"
		return nil
	}
	b, err := json.Marshal(contras)
	if err != nil {
		return err
	}
	d.ContraindicationsJSON = string(b)
	return nil
}

// UnmarshalContraindications converts stored JSON back to contraindications.
func (d InterventionDefinition) UnmarshalContraindications() ([]Contraindication, error) {
	if d.ContraindicationsJSON == "" || d.ContraindicationsJSON == "[]" {
		return []Contraindication{}, nil
	}
	var contras []Contraindication
	if err := json.Unmarshal([]byte(d.ContraindicationsJSON), &contras); err != nil {
		return nil, err
	}
	return contras, nil
}
