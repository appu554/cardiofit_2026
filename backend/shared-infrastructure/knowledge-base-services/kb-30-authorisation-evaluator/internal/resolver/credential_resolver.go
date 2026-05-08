// Package resolver provides production implementations of the
// evaluator.ConditionResolver interface. CredentialResolver replaces the
// AlwaysPassResolver test stub by querying the credentials and
// prescribing_agreements tables (kb-30 migration 002) to answer
// authorisation questions.
package resolver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

// CredentialResolver answers kb-30's evaluator.ConditionResolver interface
// by querying the credentials and prescribing_agreements tables. Each
// rule's Check string is dispatched to a specific query; unknown checks
// safely deny with a Detail string so a misconfigured rule never silently
// passes.
//
// Check strings dispatched (matching the example rules in examples/):
//   - "Credential.kind='apc_training' AND Credential.valid_at_action_time"
//   - "Credential.kind='ahpra_pharmacist_registration' AND Credential.valid_at_action_time"
//   - "Credential.endorsement_valid_at_action_time"
//   - "PrescribingAgreement.exists_for_person_AND_resident_AND_medication_class"
//   - "PrescribingAgreement.scope_includes(medication_class)"
//   - "MentorshipStatus IN ['active', 'complete']"
//   - "Action.scope_matches(PrescribingAgreement.scope)"
//
// This is the production replacement for evaluator.AlwaysPassResolver.
type CredentialResolver struct {
	db *sql.DB
}

// NewCredentialResolver constructs a resolver wired to db.
func NewCredentialResolver(db *sql.DB) *CredentialResolver {
	return &CredentialResolver{db: db}
}

// Resolve dispatches on c.Check and returns a ConditionResult. Per the
// evaluator.ConditionResolver contract, Passed=false means the condition
// was evaluated and failed; an error return means the resolver itself
// could not evaluate.
func (r *CredentialResolver) Resolve(ctx context.Context,
	q evaluator.Query, c dsl.Condition) (evaluator.ConditionResult, error) {
	res := evaluator.ConditionResult{
		Condition: c.Condition,
		Check:     c.Check,
	}

	switch c.Check {

	// --- Credential checks (ACOP example) ---

	case "Credential.kind='apc_training' AND Credential.valid_at_action_time":
		ok, err := r.hasValidCredential(ctx, q.ActorRef, "apc_training", q.ActionDate)
		res.Passed = ok
		if !ok {
			res.Detail = "APC training credential not active at action time"
		} else {
			res.Detail = "APC training credential active"
		}
		return res, err

	case "Credential.kind='ahpra_pharmacist_registration' AND Credential.valid_at_action_time":
		ok, err := r.hasValidCredential(ctx, q.ActorRef, "ahpra_pharmacist_registration", q.ActionDate)
		res.Passed = ok
		if !ok {
			res.Detail = "AHPRA pharmacist registration not active at action time"
		} else {
			res.Detail = "AHPRA pharmacist registration active"
		}
		return res, err

	// --- DRNP example checks ---

	case "Credential.endorsement_valid_at_action_time":
		ok, err := r.hasValidCredential(ctx, q.ActorRef, "NMBA_DRNP_endorsement", q.ActionDate)
		res.Passed = ok
		if !ok {
			res.Detail = "NMBA designated-RN-prescriber endorsement not active at action time"
		} else {
			res.Detail = "DRNP endorsement active"
		}
		return res, err

	case "PrescribingAgreement.exists_for_person_AND_resident_AND_medication_class":
		ok, err := r.hasActivePrescribingAgreement(ctx, q.ActorRef,
			q.ResidentRef, q.MedicationClass, q.ActionDate)
		res.Passed = ok
		if !ok {
			res.Detail = "no active prescribing agreement covers this prescriber + resident + medication class"
		} else {
			res.Detail = "active prescribing agreement covers this resident and medication class"
		}
		return res, err

	case "PrescribingAgreement.scope_includes(medication_class)":
		ok, err := r.agreementScopeIncludesClass(ctx, q.ActorRef, q.MedicationClass, q.ActionDate)
		res.Passed = ok
		if !ok {
			res.Detail = fmt.Sprintf("no active agreement scope includes medication class %q", q.MedicationClass)
		} else {
			res.Detail = "agreement scope includes medication class"
		}
		return res, err

	case "MentorshipStatus IN ['active', 'complete']":
		ok, err := r.mentorshipActiveOrComplete(ctx, q.ActorRef)
		res.Passed = ok
		if !ok {
			res.Detail = "DRNP mentorship not in 'in_progress' or 'complete' state (or breached)"
		} else {
			res.Detail = "DRNP mentorship in progress or complete"
		}
		return res, err

	case "Action.scope_matches(PrescribingAgreement.scope)":
		// Treat scope match as "the prescriber has at least one active
		// agreement whose medication_classes include the action's class".
		// More elaborate scope semantics (formulary, dose limits) would
		// require additional substrate beyond migration 002.
		ok, err := r.agreementScopeIncludesClass(ctx, q.ActorRef, q.MedicationClass, q.ActionDate)
		res.Passed = ok
		if !ok {
			res.Detail = "action scope outside any active prescribing agreement"
		} else {
			res.Detail = "action scope matches an active prescribing agreement"
		}
		return res, err

	default:
		// SAFE DEFAULT: unknown checks deny. A misconfigured rule must
		// never silently pass.
		res.Passed = false
		res.Detail = fmt.Sprintf("unknown check %q (no resolver mapping); defaulting to deny", c.Check)
		return res, nil
	}
}

// hasValidCredential reports whether person holds an unrevoked, in-date
// credential of the given type at actionDate. A zero actionDate falls
// back to CURRENT_DATE (Postgres) so callers without a populated
// ActionDate still get a sensible answer.
func (r *CredentialResolver) hasValidCredential(ctx context.Context,
	personID uuid.UUID, credType string, actionDate interface{}) (bool, error) {
	const q = `
SELECT 1 FROM credentials
WHERE person_id = $1 AND type = $2
  AND revoked_at IS NULL
  AND valid_from <= COALESCE($3::date, CURRENT_DATE)
  AND (valid_to IS NULL OR valid_to >= COALESCE($3::date, CURRENT_DATE))
LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, personID, credType, nullableDate(actionDate)).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// hasActivePrescribingAgreement reports whether prescriber holds a current
// prescribing agreement covering the given medication class for the
// (optionally named) resident, with mentorship not in a 'breached' state.
func (r *CredentialResolver) hasActivePrescribingAgreement(ctx context.Context,
	prescriberID, residentID uuid.UUID, medicationClass string, actionDate interface{}) (bool, error) {
	const q = `
SELECT 1 FROM prescribing_agreements
WHERE prescriber_id = $1
  AND $3 = ANY(medication_classes)
  AND mentorship_status IN ('in_progress','complete')
  AND revoked_at IS NULL
  AND valid_from <= COALESCE($4::date, CURRENT_DATE)
  AND (valid_to IS NULL OR valid_to >= COALESCE($4::date, CURRENT_DATE))
  AND (resident_scope = 'all' OR $2 = ANY(named_residents))
LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, prescriberID, residentID, medicationClass,
		nullableDate(actionDate)).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// agreementScopeIncludesClass reports whether prescriber holds at least
// one active agreement whose medication_classes array includes the given
// class. Distinct from hasActivePrescribingAgreement in that it ignores
// the resident scope (used by scope_includes / scope_matches checks).
func (r *CredentialResolver) agreementScopeIncludesClass(ctx context.Context,
	prescriberID uuid.UUID, medicationClass string, actionDate interface{}) (bool, error) {
	const q = `
SELECT 1 FROM prescribing_agreements
WHERE prescriber_id = $1
  AND $2 = ANY(medication_classes)
  AND mentorship_status IN ('in_progress','complete')
  AND revoked_at IS NULL
  AND valid_from <= COALESCE($3::date, CURRENT_DATE)
  AND (valid_to IS NULL OR valid_to >= COALESCE($3::date, CURRENT_DATE))
LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, prescriberID, medicationClass,
		nullableDate(actionDate)).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// mentorshipActiveOrComplete reports whether prescriber has at least one
// current agreement with mentorship_status of 'in_progress' or 'complete'
// (i.e. not 'breached'). Migration 002 uses 'in_progress' rather than
// 'active'; the rule's literal "MentorshipStatus IN ['active','complete']"
// is treated as an aliased equivalent.
func (r *CredentialResolver) mentorshipActiveOrComplete(ctx context.Context,
	prescriberID uuid.UUID) (bool, error) {
	const q = `
SELECT 1 FROM prescribing_agreements
WHERE prescriber_id = $1
  AND mentorship_status IN ('in_progress','complete')
  AND revoked_at IS NULL
LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, q, prescriberID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// nullableDate converts a time.Time-or-nil into something pq accepts as
// a nullable date parameter. A zero-value time.Time is treated as nil so
// the COALESCE in queries falls back to CURRENT_DATE.
func nullableDate(v interface{}) interface{} {
	type zeroable interface{ IsZero() bool }
	if v == nil {
		return nil
	}
	if z, ok := v.(zeroable); ok && z.IsZero() {
		return nil
	}
	return v
}

// Compile-time check that CredentialResolver satisfies the
// evaluator.ConditionResolver interface.
var _ evaluator.ConditionResolver = (*CredentialResolver)(nil)
