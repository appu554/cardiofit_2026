// Package generator implements Stage 3 of the six-stage rendering pipeline:
// draft recommendation packet generation.
//
// VisibilityClass: PDP — pharmacist's drafted recommendation packet
//
// Generate takes a ClinicalSnapshot (from Stage 1) and a slice of
// ApplicableRules (from Stage 2) and produces a draft Packet whose sections
// conform to the v3 §7 template enforced by the template package.
//
// Only the first ApplicableRule is used to drive the packet; a future
// orderer stage (Task 8) reorders the input slice so that the highest-priority
// rule appears first.
//
// Sections produced at this stage:
//   - issue:            rule-fire description
//   - clinical_context: snapshot summary (eGFR, DBI, CFS, care intensity, fall/admission flags)
//   - urgency:          rule urgency tier
//   - rationale, evidence, proposed_plan, monitoring: template.NA placeholders
//     (filled by Tasks 7, 10, and 12)
package generator

import (
	"errors"
	"fmt"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/cardiofit/kb32/internal/template"
	"github.com/google/uuid"
)

// ErrNoApplicableRules is returned by Generate when the rules slice is empty
// or nil. Without at least one fired rule there is no basis for a packet.
var ErrNoApplicableRules = errors.New("generator: no applicable rules to generate packet from")

// validPacketTypes is the canonical set of recommendation type values.
var validPacketTypes = map[string]struct{}{
	"STOP":        {},
	"MONITOR":     {},
	"DOSE_CHANGE": {},
	"ADD":         {},
}

// IsValidPacketType reports whether s is one of the four recognised
// recommendation type values. The check is case-sensitive.
func IsValidPacketType(s string) bool {
	_, ok := validPacketTypes[s]
	return ok
}

// Packet is a draft recommendation packet produced by Stage 3.
// It targets the v3 §7 template structure; all 7 sections are present and
// template-valid at construction time.
type Packet struct {
	// RecommendationID is a unique identifier for this draft packet.
	RecommendationID uuid.UUID

	// AuthorID is the pharmacist (or system) principal generating this packet.
	AuthorID uuid.UUID

	// Type is the recommendation type derived from the applied rule (e.g. "STOP").
	Type string

	// Sections maps the 7 canonical section keys to their content.
	// Use template.Enforce to validate. Downstream stages fill the NA placeholders.
	Sections map[string]string

	// AppliedRule is the first ApplicableRule from the reasoning chain that
	// drove this packet. Task 8's orderer ensures this is the highest-priority rule.
	AppliedRule reasoning.ApplicableRule

	// SnapshotRef is the ResidentID from the ClinicalSnapshot, providing a
	// stable reference back to the clinical state that informed this packet.
	SnapshotRef uuid.UUID
}

// Generate produces a draft Packet from the first ApplicableRule in rules,
// populated with clinical context from snap.
//
// Returns ErrNoApplicableRules if rules is nil or empty.
// The returned Packet always passes template.Enforce at the point of return.
func Generate(snap kb32ctx.ClinicalSnapshot, rules []reasoning.ApplicableRule, authorID uuid.UUID) (*Packet, error) {
	if len(rules) == 0 {
		return nil, ErrNoApplicableRules
	}

	// First rule wins; Task 8's orderer reorders before this stage is called.
	first := rules[0]

	sections := map[string]string{
		template.SectionIssue:        buildIssueSection(first),
		template.SectionClinicalCtx:  buildClinicalContextSection(snap),
		template.SectionRationale:    template.NA,
		template.SectionEvidence:     template.NA,
		template.SectionProposedPlan: template.NA,
		template.SectionMonitoring:   template.NA,
		template.SectionUrgency:      buildUrgencySection(first),
	}

	// Defensive: verify template compliance at construction time.
	// This should never fail given the map above, but an explicit check guards
	// against future section additions diverging from requiredSections.
	if err := template.Enforce(sections); err != nil {
		return nil, fmt.Errorf("generator: internal template violation: %w", err)
	}

	return &Packet{
		RecommendationID: uuid.New(),
		AuthorID:         authorID,
		Type:             first.Type,
		Sections:         sections,
		AppliedRule:      first,
		SnapshotRef:      snap.ResidentID,
	}, nil
}

// ---------------------------------------------------------------------------
// Section builders
// ---------------------------------------------------------------------------

func buildIssueSection(rule reasoning.ApplicableRule) string {
	return fmt.Sprintf("Rule %s fired: recommendation type %s at urgency %s.",
		rule.RuleID, rule.Type, rule.Urgency)
}

func buildClinicalContextSection(snap kb32ctx.ClinicalSnapshot) string {
	fall := "no"
	if snap.RecentFall72h {
		fall = "yes"
	}
	admission := "no"
	if snap.RecentAdmission72h {
		admission = "yes"
	}
	return fmt.Sprintf(
		"eGFR: %.1f mL/min/1.73m2; DBI: %.2f; CFS: %d; CareIntensity: %s; RecentFall72h: %s; RecentAdmission72h: %s.",
		snap.EGFR, snap.DBI, snap.CFS, snap.CareIntensity, fall, admission,
	)
}

func buildUrgencySection(rule reasoning.ApplicableRule) string {
	return fmt.Sprintf("Urgency: %s (derived from rule %s).", rule.Urgency, rule.RuleID)
}
