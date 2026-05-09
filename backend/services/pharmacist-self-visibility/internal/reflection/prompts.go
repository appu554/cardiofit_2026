package reflection

import (
	"context"
	"errors"
	"hash/fnv"

	"github.com/google/uuid"
)

// ErrEmptyLibrary is returned by Selector.Select when the prompt library is empty.
var ErrEmptyLibrary = errors.New("reflection: empty prompt library")

// Prompt is a single reflective writing prompt from the curated library.
// Prompts are open-ended, past-tense/present-tense, and pattern-revealing
// without judgment — per Self-Visibility Guidelines §5.1.
type Prompt struct {
	ID   uuid.UUID
	Body string
	Tags []string // e.g. ["restraint", "deprescribing", "general"]
}

// HasTag reports whether the prompt carries the given tag.
func (p Prompt) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// DefaultPromptLibrary returns the curated v1 prompts from
// Self-Visibility Guidelines §5.1.
func DefaultPromptLibrary() []Prompt {
	return []Prompt{
		{
			ID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Body: "This month you authored N recommendations. Which one are you proudest of, and why?",
			Tags: []string{"general", "achievement"},
		},
		{
			ID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Body: "You overrode the restraint signal on M antipsychotic recommendations this quarter. What clinical reasoning supported that?",
			Tags: []string{"restraint", "antipsychotic"},
		},
		{
			ID:   uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Body: "Your context-assembly time has changed recently. What's making that possible?",
			Tags: []string{"context_time", "trajectory"},
		},
		{
			ID:   uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			Body: "You've been working with this GP for 6 months. What have you learned about what lands well?",
			Tags: []string{"gp_relationship", "communication"},
		},
	}
}

// Signals provides activity-signal facts about a pharmacist's clinical work.
//
// VisibilityClass constraint: this interface deliberately does NOT include any
// method that accesses reflective entries (Entry, Store, or equivalent).
// Prompt selection consults substrate facts about clinical work only — never
// about the pharmacist's reflective writing.  Reflective entries are
// Pharmacist-Only-Always (POA) and must remain isolated from all algorithmic
// selection.  See Self-Visibility Guidelines §6.4.
//
// If you need to add a new signal, it must be derivable from clinical-work
// substrate (overrides, recommendation distributions, timing data) — NOT from
// what the pharmacist has written in their reflective journal.
type Signals interface {
	// RestraintOverridesIn returns the number of restraint-signal overrides
	// the pharmacist issued in the last `days` days.
	RestraintOverridesIn(ctx context.Context, pharmacistID uuid.UUID, days int) (int, error)

	// RecommendationTypeMix returns a count map of recommendation types issued
	// in the last `days` days (e.g. {"deprescribing": 4, "dose_adjust": 7}).
	RecommendationTypeMix(ctx context.Context, pharmacistID uuid.UUID, days int) (map[string]int, error)
}

// Selector selects a reflective prompt for a given pharmacist and calendar month.
//
// VisibilityClass: POA-isolated — does NOT read reflective entries.
//
// Selection logic:
//  1. If the restraint-override signal is strong (≥3 in 90 days), the first
//     restraint-tagged prompt in the library is returned, regardless of month.
//  2. Otherwise, a deterministic FNV-1a hash of (pharmacistID, year, month) is
//     used to index into the library, producing stable monthly rotation that
//     varies naturally across pharmacists.
type Selector struct {
	library []Prompt
	signals Signals
}

// NewSelector constructs a Selector with the given prompt library and signal provider.
func NewSelector(lib []Prompt, sig Signals) *Selector {
	return &Selector{library: lib, signals: sig}
}

// Select returns a prompt for the given pharmacist and calendar period.
// It returns ErrEmptyLibrary if the library has no prompts.
//
// The method consults clinical-work signals only — it does NOT read reflective
// entries (POA isolation, Guidelines §6.4).
func (s *Selector) Select(ctx context.Context, pharmacistID uuid.UUID, year, month int) (Prompt, error) {
	if len(s.library) == 0 {
		return Prompt{}, ErrEmptyLibrary
	}

	// Signal path: strong restraint-override activity overrides rotation.
	overrides, _ := s.signals.RestraintOverridesIn(ctx, pharmacistID, 90)
	if overrides >= 3 {
		for _, p := range s.library {
			if p.HasTag("restraint") {
				return p, nil
			}
		}
	}

	// Default rotation: deterministic FNV-1a hash of (pharmacistID bytes, year, month).
	h := fnv.New32a()
	_, _ = h.Write(pharmacistID[:])
	var ymBuf [3]byte
	ymBuf[0] = byte(year >> 8)
	ymBuf[1] = byte(year)
	ymBuf[2] = byte(month)
	_, _ = h.Write(ymBuf[:])

	idx := int(h.Sum32()) % len(s.library)
	if idx < 0 {
		idx = -idx
	}
	return s.library[idx], nil
}
