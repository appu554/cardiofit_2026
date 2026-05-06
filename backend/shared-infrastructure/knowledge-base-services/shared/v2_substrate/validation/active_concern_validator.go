package validation

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateActiveConcern reports any structural problem with c. Per Layer 2
// doc §2.3.
//
// Required fields:
//   - ResidentID  (uuid.Nil rejected)
//   - ConcernType (must be one of models.ActiveConcern* constants)
//   - StartedAt   (zero time rejected)
//   - ExpectedResolutionAt (zero time rejected; must be > StartedAt)
//   - ResolutionStatus (must be one of models.ResolutionStatus* values)
//
// Conditional rules:
//   - If ResolutionStatus is terminal (resolved_stop_criteria, escalated,
//     expired_unresolved), ResolvedAt must be set and >= StartedAt.
//   - If ResolutionStatus is open, ResolvedAt must be nil.
func ValidateActiveConcern(c models.ActiveConcern) error {
	if c.ResidentID == uuid.Nil {
		return errors.New("resident_id is required")
	}
	if !models.IsValidActiveConcernType(c.ConcernType) {
		return fmt.Errorf("invalid concern_type %q", c.ConcernType)
	}
	if c.StartedAt.IsZero() {
		return errors.New("started_at is required")
	}
	if c.ExpectedResolutionAt.IsZero() {
		return errors.New("expected_resolution_at is required")
	}
	if !c.ExpectedResolutionAt.After(c.StartedAt) {
		return errors.New("expected_resolution_at must be after started_at")
	}
	if !models.IsValidResolutionStatus(c.ResolutionStatus) {
		return fmt.Errorf("invalid resolution_status %q", c.ResolutionStatus)
	}

	if c.ResolutionStatus == models.ResolutionStatusOpen {
		if c.ResolvedAt != nil {
			return errors.New("open concern must not have resolved_at set")
		}
	} else {
		if c.ResolvedAt == nil {
			return fmt.Errorf("%s concern requires resolved_at", c.ResolutionStatus)
		}
		if c.ResolvedAt.Before(c.StartedAt) {
			return errors.New("resolved_at must be >= started_at")
		}
	}
	return nil
}

// ValidateActiveConcernResolutionTransition reports whether moving an
// existing concern from `fromStatus` to `toStatus` is permitted by the
// state machine. Used by PATCH /active-concerns/:id handlers.
func ValidateActiveConcernResolutionTransition(fromStatus, toStatus string) error {
	if !models.IsValidResolutionStatus(fromStatus) {
		return fmt.Errorf("invalid current resolution_status %q", fromStatus)
	}
	if !models.IsValidResolutionStatus(toStatus) {
		return fmt.Errorf("invalid target resolution_status %q", toStatus)
	}
	if !models.IsValidResolutionTransition(fromStatus, toStatus) {
		return fmt.Errorf("illegal status transition %s → %s", fromStatus, toStatus)
	}
	return nil
}
