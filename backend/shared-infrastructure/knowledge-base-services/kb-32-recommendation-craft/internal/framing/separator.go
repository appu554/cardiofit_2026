// Package framing implements Stage 5 of the six-stage rendering pipeline:
// frame-vs-content separation for audit-defensible multi-audience delivery.
//
// VisibilityClass: AD — frame-vs-content separation per Guidelines §8
// (audit-defensibility against "did the system tell different audiences
// different things?")
//
// The architectural contract is that ClinicalContent is audience-invariant:
// identical clinical content must produce the same ContentHash regardless of
// which audience it is later framed for. FramingAdaptation is the
// audience-variable layer; many framings can attach to one ClinicalContent
// without affecting its hash.
//
// ContentHash sorts EvidenceAnchors alphabetically before hashing so that
// anchor-list ordering imposed by upstream stages never changes the hash value.
// This is a non-negotiable audit-defensibility commitment per Guidelines §8.
package framing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

// validAudiences is the canonical set of accepted audience codes.
var validAudiences = map[string]struct{}{
	"gp":         {},
	"pharmacist": {},
	"rach_staff": {},
	"regulator":  {},
}

// validUrgencies is the canonical set of accepted urgency values.
var validUrgencies = map[string]struct{}{
	"red":   {},
	"amber": {},
	"green": {},
}

// ErrInvalidContent is returned by ClinicalContent.Validate when the struct
// contains an empty or invalid field value.
var ErrInvalidContent = errors.New("framing: invalid clinical content")

// ClinicalContent is the audience-invariant payload. The same ContentHash is
// guaranteed regardless of how this content is later framed for different
// audiences. This is the information the regulator audit checks for consistency.
type ClinicalContent struct {
	// RuleID is the stable identifier for the clinical rule that fired.
	// Matches the RuleID field used in the generator and reasoning packages.
	RuleID string

	// Type is the recommendation type (e.g. "STOP", "MONITOR", "DOSE_CHANGE").
	// Matches the Type field used in the generator package.
	Type string

	// EvidenceAnchors is the list of evidence source IDs supporting this content.
	// Populated from Anchor.SourceID values selected by the evidence package.
	// Sorted alphabetically before hashing; ordering in this slice does NOT
	// affect the ContentHash.
	EvidenceAnchors []string

	// Urgency is the clinical urgency tier: "red", "amber", or "green".
	Urgency string
}

// FramingAdaptation is the audience-variable layer. Multiple framings may
// reference the same ClinicalContent without changing its hash. Each framing
// tailors message tone and structure for a specific clinical audience.
type FramingAdaptation struct {
	// Audience is the target audience code. Must be one of the values accepted
	// by IsValidAudience: "gp", "pharmacist", "rach_staff", "regulator".
	Audience string

	// OpeningLine is the audience-specific opening statement.
	OpeningLine string

	// ClosingCall is the audience-specific call to action.
	ClosingCall string
}

// IsValidAudience reports whether s is one of the four recognised audience codes.
// Valid values: "gp", "pharmacist", "rach_staff", "regulator".
// The check is case-sensitive.
func IsValidAudience(s string) bool {
	_, ok := validAudiences[s]
	return ok
}

// Validate returns ErrInvalidContent when the ClinicalContent contains an empty
// RuleID, an empty Type, or an Urgency value that is not "red", "amber", or
// "green". This guards against caller bugs sending malformed content through the
// pipeline before it is hashed and stored.
func (c ClinicalContent) Validate() error {
	if c.RuleID == "" {
		return errors.New("framing: invalid clinical content: RuleID must not be empty")
	}
	if c.Type == "" {
		return errors.New("framing: invalid clinical content: Type must not be empty")
	}
	if _, ok := validUrgencies[c.Urgency]; !ok {
		return errors.New("framing: invalid clinical content: Urgency must be one of red/amber/green, got: " + c.Urgency)
	}
	return nil
}

// ContentHash returns a deterministic SHA-256 hex string for c.
// EvidenceAnchors are sorted alphabetically before serialisation so that
// anchor-list ordering does not affect the hash value — the same clinical
// content must produce the same hash regardless of how upstream stages ordered
// the anchor slice.
//
// The returned string is always 64 lower-case hexadecimal characters.
// Marshalling errors are treated as programmer errors (unreachable for this
// struct shape); any such error panics rather than silently producing a
// wrong hash.
func ContentHash(c ClinicalContent) string {
	// Sort a copy of EvidenceAnchors so the caller's slice is not mutated.
	sorted := make([]string, len(c.EvidenceAnchors))
	copy(sorted, c.EvidenceAnchors)
	sort.Strings(sorted)
	c.EvidenceAnchors = sorted

	b, err := json.Marshal(c)
	if err != nil {
		// json.Marshal on a simple struct with only string/[]string fields cannot
		// return an error under normal conditions. Panic signals a programming error.
		panic("framing: ContentHash: unexpected json.Marshal error: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// IsContentInvariantAcross verifies that all entries in contents produce the
// same ContentHash. It is intended for regulator audit assertions: given a
// recommendation that was delivered to multiple audiences (framings), confirm
// that every paired ClinicalContent is identical in hash.
//
// Returns true when contents is empty or contains only one entry (vacuously
// true), and true when all entries hash identically.
// Returns false as soon as any entry's hash differs from the first.
func IsContentInvariantAcross(framings []FramingAdaptation, contents []ClinicalContent) bool {
	if len(contents) <= 1 {
		return true
	}
	reference := ContentHash(contents[0])
	for _, c := range contents[1:] {
		if ContentHash(c) != reference {
			return false
		}
	}
	return true
}
