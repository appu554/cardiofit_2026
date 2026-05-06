// Package validation — scoring instrument validators (Wave 2.6 of the Layer 2
// substrate plan; Layer 2 doc §2.4 / §2.6). One validator per score type:
// CFS / AKPS are clinician-entered; DBI / ACB are computed by
// shared/v2_substrate/scoring from the resident's active MedicineUse list.
package validation

import (
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// dbiComponentSumTolerance is the float-comparison tolerance used by
// ValidateDBIScore when checking that Score == AnticholinergicComponent +
// SedativeComponent. Picked well above typical IEEE-754 rounding for the
// 0.0/0.5/1.0 weights the calculator produces but tight enough that any
// real arithmetic mistake (component miscount) trips the validator.
const dbiComponentSumTolerance = 1e-9

// ValidateCFSScore reports any structural problem with c. Per Layer 2 doc
// §2.4 / §2.6 (Wave 2.6 of the Layer 2 substrate plan).
//
// Required fields:
//   - ResidentRef       (uuid.Nil rejected)
//   - AssessedAt        (zero time rejected)
//   - AssessorRoleRef   (uuid.Nil rejected — clinician-entered)
//   - InstrumentVersion (empty string rejected — pin to a specific revision
//     so historic scores can be re-interpreted if the scale changes)
//   - Score             (must be in [1, 9] inclusive — the Rockwood scale)
func ValidateCFSScore(c models.CFSScore) error {
	if c.ResidentRef == uuid.Nil {
		return errors.New("resident_ref is required")
	}
	if c.AssessedAt.IsZero() {
		return errors.New("assessed_at is required")
	}
	if c.AssessorRoleRef == uuid.Nil {
		return errors.New("assessor_role_ref is required")
	}
	if c.InstrumentVersion == "" {
		return errors.New("instrument_version is required")
	}
	if c.Score < 1 || c.Score > 9 {
		return fmt.Errorf("score must be in [1,9]; got %d", c.Score)
	}
	return nil
}

// ValidateAKPSScore reports any structural problem with a. Per Layer 2 doc
// §2.4 / §2.6.
//
// Required fields:
//   - ResidentRef       (uuid.Nil rejected)
//   - AssessedAt        (zero time rejected)
//   - AssessorRoleRef   (uuid.Nil rejected — clinician-entered)
//   - InstrumentVersion (empty string rejected)
//   - Score             (must be in [0, 100] inclusive AND a multiple of 10)
func ValidateAKPSScore(a models.AKPSScore) error {
	if a.ResidentRef == uuid.Nil {
		return errors.New("resident_ref is required")
	}
	if a.AssessedAt.IsZero() {
		return errors.New("assessed_at is required")
	}
	if a.AssessorRoleRef == uuid.Nil {
		return errors.New("assessor_role_ref is required")
	}
	if a.InstrumentVersion == "" {
		return errors.New("instrument_version is required")
	}
	if a.Score < 0 || a.Score > 100 {
		return fmt.Errorf("score must be in [0,100]; got %d", a.Score)
	}
	if a.Score%10 != 0 {
		return fmt.Errorf("score must be a multiple of 10; got %d", a.Score)
	}
	return nil
}

// ValidateDBIScore reports any structural problem with d. Per Layer 2 doc
// §2.6.
//
// Required fields and invariants:
//   - ResidentRef            (uuid.Nil rejected)
//   - ComputedAt             (zero time rejected)
//   - Score >= 0
//   - AnticholinergicComponent >= 0
//   - SedativeComponent       >= 0
//   - Sanity: Score == AnticholinergicComponent + SedativeComponent
//     (within dbiComponentSumTolerance — floats). A drift trips the
//     validator because either the calculator or the persistence layer has
//     desynchronised the components from the total.
//
// DBI is computed; there is no AssessorRoleRef field. Empty
// ComputationInputs is permitted (residents with no active medications
// have a legitimate zero score).
func ValidateDBIScore(d models.DBIScore) error {
	if d.ResidentRef == uuid.Nil {
		return errors.New("resident_ref is required")
	}
	if d.ComputedAt.IsZero() {
		return errors.New("computed_at is required")
	}
	if d.Score < 0 {
		return fmt.Errorf("score must be >= 0; got %v", d.Score)
	}
	if d.AnticholinergicComponent < 0 {
		return fmt.Errorf("anticholinergic_component must be >= 0; got %v", d.AnticholinergicComponent)
	}
	if d.SedativeComponent < 0 {
		return fmt.Errorf("sedative_component must be >= 0; got %v", d.SedativeComponent)
	}
	sum := d.AnticholinergicComponent + d.SedativeComponent
	if math.Abs(d.Score-sum) > dbiComponentSumTolerance {
		return fmt.Errorf("score (%v) must equal anticholinergic_component + sedative_component (%v)", d.Score, sum)
	}
	return nil
}

// ValidateACBScore reports any structural problem with a. Per Layer 2 doc
// §2.6.
//
// Required fields:
//   - ResidentRef       (uuid.Nil rejected)
//   - ComputedAt        (zero time rejected)
//   - Score >= 0
//
// ACB is computed; there is no AssessorRoleRef. Empty ComputationInputs
// is permitted (residents with no active or no anticholinergic medications
// have a legitimate zero score).
func ValidateACBScore(a models.ACBScore) error {
	if a.ResidentRef == uuid.Nil {
		return errors.New("resident_ref is required")
	}
	if a.ComputedAt.IsZero() {
		return errors.New("computed_at is required")
	}
	if a.Score < 0 {
		return fmt.Errorf("score must be >= 0; got %d", a.Score)
	}
	return nil
}
