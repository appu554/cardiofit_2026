// visibility_class.go enforces S2 v1.0 Part 13.3 visibility-class
// boundaries on audit-trail reads. The pharmacist self-visibility module
// defines four classes used here:
//
//   - PDP (Pharmacist-Default-Private): the pharmacist's own clinical
//     workspace activity. PDP rows are readable ONLY by the owning
//     pharmacist. Cross-pharmacist reads violate v1.0 Part 13.3 +
//     Addendum Part 5.2 (algorithmic management protections).
//
//   - PEV (Pharmacist-Employer-View): expressly restricted. Aggregated
//     pharmacist-work patterns are NOT shared with employer for
//     performance monitoring (v1.0 Part 13.4 + Addendum Part 5.2).
//
//   - AD (Audit-Defensible): records readable under formal regulatory /
//     ethics-committee data-sharing agreement only.
//
//   - PDF (Pharmacist-Default-Family): family-visible records (used by
//     the family communication panel, not directly by this enforcement
//     module — included here so the enum is complete).
//
// Enforcement model:
//
//   - EnforcePDPRead is the per-row gate: the pharmacist may read their
//     own PDP rows; any cross-pharmacist read returns ErrCrossPharmacistRead.
//
//   - EnforcePDPAggregateRead is the aggregate gate: even
//     clinical-informatics and ethics-steering-committee roles get
//     anonymised aggregate access only — never per-pharmacist patterns.
//     Any other requester role returns ErrSurveillanceAttempt.
package audit

import (
	"errors"

	"github.com/google/uuid"
)

// VisibilityClass is the pharmacist self-visibility module visibility
// classification applied to every S2 audit row.
type VisibilityClass string

const (
	// VisibilityPDP — Pharmacist-Default-Private. The pharmacist's own
	// S2 workspace activity. Owner-only read access.
	VisibilityPDP VisibilityClass = "PDP"
	// VisibilityPEV — Pharmacist-Employer-View. Restricted; performance
	// surveillance is forbidden by Addendum Part 5.2.
	VisibilityPEV VisibilityClass = "PEV"
	// VisibilityAD — Audit-Defensible. Regulator / ethics-committee
	// access under formal data-sharing agreement only.
	VisibilityAD VisibilityClass = "AD"
	// VisibilityPDF — Pharmacist-Default-Family. Family-visible records.
	VisibilityPDF VisibilityClass = "PDF"
)

// Sentinel errors returned by the enforcement functions. Callers in
// API layer (Task 8) translate these to 403 responses with appropriate
// audit-log entries.
var (
	// ErrCrossPharmacistRead indicates a pharmacist attempted to read
	// another pharmacist's PDP-class audit rows. This is the v1.0 Part
	// 13.3 + Addendum Part 5.2 boundary — pharmacists do not see each
	// other's S2 patterns.
	ErrCrossPharmacistRead = errors.New("audit: cross-pharmacist read of PDP row forbidden (v1.0 Part 13.3 + Addendum Part 5.2)")

	// ErrSurveillanceAttempt indicates an aggregate-read requester role
	// is one that is not authorised for aggregate access at all — most
	// frequently an employer / manager role attempting the read PEV
	// classification expressly prohibits.
	ErrSurveillanceAttempt = errors.New("audit: surveillance attempt rejected (Addendum Part 5.2 forbids employer / cross-pharmacist surveillance)")

	// ErrUnauthorizedAggregate indicates the requester role is in
	// principle authorised for aggregate access but is missing some
	// other condition (e.g., purpose-of-use missing, aggregation level
	// invalid). Reserved for richer policy enforcement at Task 8.
	ErrUnauthorizedAggregate = errors.New("audit: aggregate access request missing required policy fields")
)

// Roles authorised for anonymised-aggregate access. Even these roles
// get NO per-pharmacist visibility — Addendum Part 5.2's restrictions
// apply at every level of aggregation up to and including these roles.
const (
	// RoleClinicalInformatics is the clinical-informatics role
	// authorised to review activation-criteria calibration patterns at
	// aggregate, anonymised level per Addendum Part 5.3.
	RoleClinicalInformatics = "clinical_informatics"
	// RoleEthicsSteeringCommittee is the ethics-steering-committee role
	// authorised to review pattern detections under formal review per
	// Addendum Part 5.5 (Phase 4 gating).
	RoleEthicsSteeringCommittee = "ethics_steering_committee"
)

// AggregateAccessRequest is the audit-trail-bound context every
// aggregate-read attempt carries. The struct is captured to the audit
// substrate so every aggregate read leaves a trail per v1.0 Part 13.5
// (audit trail surfacing) and Addendum Part 4.6.
type AggregateAccessRequest struct {
	// RequesterID identifies the human or service issuing the request.
	RequesterID uuid.UUID
	// RequesterRole names the authorisation role. Only the two
	// constants above are accepted by EnforcePDPAggregateRead.
	RequesterRole string
	// Purpose is the free-form purpose-of-use string captured for the
	// audit log.
	Purpose string
	// AggregationLevel names the level of aggregation requested (e.g.,
	// "platform_wide", "facility", "month"). Per-pharmacist values are
	// rejected upstream and never reach this struct.
	AggregationLevel string
}

// EnforcePDPRead is the per-row PDP read gate. It returns nil when
// requesterID matches ownerID and ErrCrossPharmacistRead otherwise.
//
// This is the load-bearing single-row check applied to every read of a
// PDP-class audit row. v1.0 Part 13.3 + Addendum Part 5.2: a pharmacist
// may not read another pharmacist's S2 patterns.
func EnforcePDPRead(requesterID uuid.UUID, ownerID uuid.UUID) error {
	if requesterID == uuid.Nil || ownerID == uuid.Nil {
		return ErrCrossPharmacistRead
	}
	if requesterID != ownerID {
		return ErrCrossPharmacistRead
	}
	return nil
}

// EnforcePDPAggregateRead is the aggregate-read role gate. It accepts
// only the two roles authorised for anonymised-aggregate access:
// clinical_informatics and ethics_steering_committee. Any other
// requester role returns ErrSurveillanceAttempt.
//
// CRITICAL: even authorised roles get anonymised aggregate-level
// access ONLY. Per-pharmacist patterns are NEVER returned to any
// requester regardless of role — this is enforced by the absence of any
// per-pharmacist read API in the audit package (see also the
// no-surveillance-reader structural test in tests/structural).
func EnforcePDPAggregateRead(requesterRole string) error {
	switch requesterRole {
	case RoleClinicalInformatics, RoleEthicsSteeringCommittee:
		return nil
	default:
		return ErrSurveillanceAttempt
	}
}
