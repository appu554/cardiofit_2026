// Package template enforces the seven-section structure of a v3 §7 recommendation packet.
//
// VisibilityClass: PDP — clinical content per v3 §7
//
// A valid recommendation packet must contain all seven canonical sections.
// Each section must be either a non-empty, non-whitespace-only string, or the
// explicit not-applicable marker "NA". Empty strings are rejected — callers must
// use "NA" to signal that a section is intentionally absent for a given context.
package template

import (
	"errors"
	"fmt"
	"strings"
)

// NA is the explicit not-applicable marker accepted in place of real content.
// Callers MUST use this constant (not the bare string) so that usage sites are
// discoverable and the semantic meaning is unambiguous.
const NA = "NA"

// Canonical section names per v3 §7 line 369.
const (
	SectionIssue          = "issue"
	SectionClinicalCtx    = "clinical_context"
	SectionRationale      = "rationale"
	SectionEvidence       = "evidence"
	SectionProposedPlan   = "proposed_plan"
	SectionMonitoring     = "monitoring"
	SectionUrgency        = "urgency"
)

// requiredSections is the ordered slice of the 7 canonical section keys.
var requiredSections = []string{
	SectionIssue,
	SectionClinicalCtx,
	SectionRationale,
	SectionEvidence,
	SectionProposedPlan,
	SectionMonitoring,
	SectionUrgency,
}

// validSectionSet provides O(1) membership testing for IsValidSection.
var validSectionSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(requiredSections))
	for _, s := range requiredSections {
		m[s] = struct{}{}
	}
	return m
}()

// Sentinel errors returned by Enforce.
var (
	// ErrMissingSection is returned when a required section key is absent from the packet.
	ErrMissingSection = errors.New("template: missing required section")

	// ErrInvalidSection is returned when a section is present but its value is
	// empty or contains only whitespace. Use NA for intentionally absent content.
	ErrInvalidSection = errors.New("template: invalid section value (use NA for not-applicable)")
)

// RequiredSections returns the ordered list of the 7 canonical section names.
// Callers such as the generator (Task 6) should use this function rather than
// redefining the list to avoid divergence.
func RequiredSections() []string {
	out := make([]string, len(requiredSections))
	copy(out, requiredSections)
	return out
}

// IsValidSection reports whether s is one of the 7 canonical section names.
func IsValidSection(s string) bool {
	_, ok := validSectionSet[s]
	return ok
}

// Enforce validates that packet contains all 7 required sections.
//
// Rules:
//   - A section key that is absent from the map → ErrMissingSection.
//   - A section key whose value is an empty string or whitespace-only → ErrInvalidSection.
//   - The literal "NA" is accepted as an explicit not-applicable declaration.
//   - Any other non-empty, non-whitespace string is accepted as content.
func Enforce(packet map[string]string) error {
	for _, sec := range requiredSections {
		val, present := packet[sec]
		if !present {
			return fmt.Errorf("%w: %q", ErrMissingSection, sec)
		}
		if strings.TrimSpace(val) == "" {
			return fmt.Errorf("%w: %q is empty or whitespace (set to %q if not applicable)", ErrInvalidSection, sec, NA)
		}
	}
	return nil
}
