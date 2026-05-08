package permissions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// VisibilityClass encodes who can see what, per Self-Visibility Guidelines §2.1.
type VisibilityClass int

const (
	POA VisibilityClass = iota // Pharmacist-Only-Always
	PDP                        // Pharmacist-Default-Private
	PFA                        // Pharmacist-First-Then-Aggregated
	WO                         // Workflow-Operational
	AD                         // Audit-Defensible
)

const (
	ViewTypePharmacist = "pharmacist"
	ViewTypeEmployer   = "pharmacy_employer"
	ViewTypeRACH       = "rach"
	ViewTypeChain      = "chain"
	ViewTypeRegulator  = "regulator"
)

// AggregationGate guards PFA-class aggregation per Self-Visibility Guidelines §2.3.
type AggregationGate struct {
	MinObservations  int           `json:"min_observations"`
	TimeWindow       time.Duration `json:"time_window"`
	DelayWindow      time.Duration `json:"delay_window"`
	ContractualBasis string        `json:"contractual_basis"`
	ExplicitNotice   bool          `json:"explicit_notice"`
}

// Satisfied returns true when all PFA gating conditions are met:
//   - observationCount >= MinObservations
//   - periodStart is within TimeWindow + DelayWindow looking back from asOf
//     (the combined window accounts for the mandatory pharmacist-first delay)
//   - asOf is at least DelayWindow after periodStart (pharmacist sees first)
func (ag AggregationGate) Satisfied(observationCount int, asOf time.Time, periodStart time.Time) bool {
	if observationCount < ag.MinObservations {
		return false
	}
	// Reject if the period started too far in the past: beyond TimeWindow + DelayWindow.
	// The DelayWindow is added because aggregation can only occur after the delay elapses,
	// so the effective lookback is TimeWindow measured from when aggregation first became possible.
	cutoff := asOf.Add(-(ag.TimeWindow + ag.DelayWindow))
	if periodStart.Before(cutoff) {
		return false
	}
	if asOf.Before(periodStart.Add(ag.DelayWindow)) {
		return false
	}
	return true
}

// Scope defines what a ViewPermission grants access to.
type Scope struct {
	ViewType      string          `json:"view_type"`
	ResourceTypes []string        `json:"resource_types"`
	Class         VisibilityClass `json:"class"`
	FacilityIDs   []uuid.UUID     `json:"facility_ids,omitempty"`
	// Gate is required for PFA class and must be nil for all other classes.
	Gate *AggregationGate `json:"gate,omitempty"`
}

// Validate enforces the PFA/non-PFA gate invariant.
func (s Scope) Validate() error {
	if s.Class == PFA && s.Gate == nil {
		return errors.New("PFA scope must have a non-nil AggregationGate")
	}
	if s.Class != PFA && s.Gate != nil {
		return errors.New("non-PFA scope must have nil AggregationGate")
	}
	return nil
}

type ViewPermission struct {
	ID                    uuid.UUID  `json:"id"`
	SubjectID             uuid.UUID  `json:"subject_id"`
	ViewerRoleID          uuid.UUID  `json:"viewer_role_id"`
	Scope                 Scope      `json:"scope"`
	GrantedAt             time.Time  `json:"granted_at"`
	GrantedByID           uuid.UUID  `json:"granted_by_id"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
	ContestationRecordRef *uuid.UUID `json:"contestation_record_ref,omitempty"`
}

// Allows reports whether p permits a read of resourceType,
// given subjectID as the identity of the data subject being requested.
//
// Semantics by VisibilityClass:
//
//	POA — only the subject themselves: returns true iff ViewerRoleID == subjectID
//	PDP — subject always; non-subject viewer requires a DataAggregationConsent (enforced in Middleware)
//	PFA — subject always; aggregator path requires AggregationGate.Satisfied() (enforced in Middleware)
//	WO  — any holder of this ViewPermission may read workflow-operational resources
//	AD  — any holder of this ViewPermission with an AD-grant may read audit-defensible resources
func (p ViewPermission) Allows(resourceType string, subjectID uuid.UUID) bool {
	if p.ExpiresAt != nil && time.Now().UTC().After(*p.ExpiresAt) {
		return false
	}
	// POA and PDP: only the subject themselves at the ViewPermission level.
	// Non-subject PDP/PFA access is gated additionally in the middleware.
	if p.Scope.Class == POA || p.Scope.Class == PDP {
		if p.ViewerRoleID != subjectID {
			return false
		}
	}
	for _, rt := range p.Scope.ResourceTypes {
		if rt == resourceType {
			return true
		}
	}
	return false
}
