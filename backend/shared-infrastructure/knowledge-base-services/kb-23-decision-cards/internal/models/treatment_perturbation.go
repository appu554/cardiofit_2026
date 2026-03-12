package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// StringArray implements driver.Valuer and sql.Scanner for PostgreSQL text[].
// ---------------------------------------------------------------------------

type StringArray []string

// Value serialises the string slice into PostgreSQL array literal format.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	escaped := make([]string, len(a))
	for i, s := range a {
		// Escape backslashes and double-quotes for PG array syntax.
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		escaped[i] = `"` + s + `"`
	}
	return "{" + strings.Join(escaped, ",") + "}", nil
}

// Scan deserialises a PostgreSQL text[] value into a Go string slice.
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	var raw string
	switch v := value.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("StringArray.Scan: unsupported type %T", value)
	}

	// Strip surrounding braces.
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")

	if raw == "" {
		*a = StringArray{}
		return nil
	}

	// Simple CSV parse handling quoted elements.
	var result []string
	var current strings.Builder
	inQuote := false
	escaped := false

	for _, ch := range raw {
		switch {
		case escaped:
			current.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '"':
			inQuote = !inQuote
		case ch == ',' && !inQuote:
			result = append(result, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}
	result = append(result, current.String())

	*a = result
	return nil
}

// ---------------------------------------------------------------------------
// TreatmentPerturbation tracks medication changes and their expected effect
// windows so downstream systems can attenuate observation confidence.
// ---------------------------------------------------------------------------

type TreatmentPerturbation struct {
	PerturbationID      uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"perturbation_id"`
	PatientID           uuid.UUID        `gorm:"type:uuid;index;not null" json:"patient_id"`
	InterventionType    InterventionType `gorm:"type:varchar(20);not null" json:"intervention_type"`
	DoseDelta           float64          `json:"dose_delta"`
	BaselineDose        float64          `json:"baseline_dose"`
	EffectWindowStart   time.Time        `gorm:"index" json:"effect_window_start"`
	EffectWindowEnd     time.Time        `gorm:"index" json:"effect_window_end"`
	AffectedObservables StringArray      `gorm:"type:text[]" json:"affected_observables"`
	StabilityFactor     float64          `json:"stability_factor"`

	// ── HTN co-management extensions (Wave 2) ──
	// These fields enable Channel B and C to distinguish expected drug effects
	// from pathological changes. Without them, a post-ACEi creatinine rise
	// or post-SGLT2i SBP dip would trigger false alarms.
	ExpectedDirection    string  `gorm:"type:varchar(10)" json:"expected_direction,omitempty"`     // UP | DOWN
	ExpectedMagnitudeMin float64 `json:"expected_magnitude_min,omitempty"`                         // lower bound of expected change
	ExpectedMagnitudeMax float64 `json:"expected_magnitude_max,omitempty"`                         // upper bound of expected change
	CausalNote           string  `gorm:"type:varchar(500)" json:"causal_note,omitempty"`           // human-readable for CTL Panel 3

	CreatedAt time.Time `json:"created_at"`
}

// TableName sets the PostgreSQL table name.
func (TreatmentPerturbation) TableName() string { return "treatment_perturbations" }

// BeforeCreate generates a UUID primary key if not already set.
func (p *TreatmentPerturbation) BeforeCreate(tx *gorm.DB) error {
	if p.PerturbationID == uuid.Nil {
		p.PerturbationID = uuid.New()
	}
	return nil
}
