package aggregation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// GoCFreshness thresholds per v1.0 Part 9.1:
//
//	soft (6 months): "last documented >6 months ago" — informational.
//	strong (12 months): "no recent GoC documentation" — stronger
//	                    language because the document is stale enough
//	                    that the pharmacist should not assume it
//	                    reflects current family/clinical intent.
//
// The two-tier policy mirrors v1.0 Part 9.1 ("Flag if >12 months since
// last review") with an added intermediate 6-month soft flag to
// support Phase 1 pilot reviewer signal collection.
const (
	GoCFreshnessSoftThreshold   = 6 * 30 * 24 * time.Hour  // ~6 months
	GoCFreshnessStrongThreshold = 12 * 30 * 24 * time.Hour // ~12 months

	GoCFreshnessReasonSoft   = "last documented >6 months ago"
	GoCFreshnessReasonStrong = "no recent GoC documentation"
)

// GoalsOfCarePanel is the per-resident Panel G rendering unit per
// v1.0 Part 9.1.
type GoalsOfCarePanel struct {
	Current         *substrate_types.GoalsOfCareEntry
	History         []substrate_types.GoalsOfCareEntry
	FreshnessFlag   bool
	FreshnessReason string
	SubstrateRefs   []SubstrateRef
}

// CareIntensityPanel is the per-resident Panel I rendering unit per
// v1.0 Part 9.2. SparseDataFlag follows the Substrate Feasibility
// Analysis HIGHER-RISK degradation pattern: when <2 history entries
// the renderer should display "care intensity state machine
// implementation in progress" guidance.
type CareIntensityPanel struct {
	Current        *substrate_types.CareIntensityEntry
	History        []substrate_types.CareIntensityEntry
	SparseDataFlag bool
	SubstrateRefs  []SubstrateRef
}

// GoalsConflict surfaces a pending-recommendation vs GoC misalignment
// per v1.0 Part 9.4. Mirrors the logic in
// kb-32/internal/appropriateness/substrate_scorer.go
// scoreGoalsOfCareAlignment (Phase 2-completion Task 2).
type GoalsConflict struct {
	RecommendationID   uuid.UUID
	ConflictReason     string
	CurrentGoCState    string
	RecommendationType string
	SubstrateRefs      []SubstrateRef
}

// BuildGoalsOfCarePanel composes Panel G. Returns a non-nil panel with
// zero-value Current when no GoC documentation exists; History is
// always a non-nil slice.
func BuildGoalsOfCarePanel(
	ctx context.Context,
	client SubstrateClient,
	residentID uuid.UUID,
) (GoalsOfCarePanel, error) {
	if client == nil {
		return GoalsOfCarePanel{}, fmt.Errorf("BuildGoalsOfCarePanel: nil SubstrateClient")
	}
	current, err := client.CurrentGoalsOfCare(ctx, residentID)
	if err != nil {
		return GoalsOfCarePanel{}, fmt.Errorf("CurrentGoalsOfCare: %w", err)
	}
	history, err := client.GoalsOfCareHistory(ctx, residentID)
	if err != nil {
		return GoalsOfCarePanel{}, fmt.Errorf("GoalsOfCareHistory: %w", err)
	}
	if history == nil {
		history = []substrate_types.GoalsOfCareEntry{}
	}
	panel := GoalsOfCarePanel{
		Current:       current,
		History:       history,
		SubstrateRefs: []SubstrateRef{},
	}
	if current != nil {
		age := time.Since(current.EffectiveFrom)
		switch {
		case age >= GoCFreshnessStrongThreshold:
			panel.FreshnessFlag = true
			panel.FreshnessReason = GoCFreshnessReasonStrong
		case age >= GoCFreshnessSoftThreshold:
			panel.FreshnessFlag = true
			panel.FreshnessReason = GoCFreshnessReasonSoft
		}
		panel.SubstrateRefs = append(panel.SubstrateRefs, SubstrateRef{
			Source: "kb-20-goc",
			ID:     current.SubstrateID,
			Description: fmt.Sprintf(
				"goals-of-care state %s effective %s",
				current.State,
				current.EffectiveFrom.Format("2006-01-02"),
			),
		})
	}
	return panel, nil
}

// BuildCareIntensityPanel composes Panel I. SparseDataFlag is set when
// fewer than 2 history entries are on record — the Substrate
// Feasibility Analysis HIGHER-RISK degradation pattern per v1.0 Part
// 9.2.
func BuildCareIntensityPanel(
	ctx context.Context,
	client SubstrateClient,
	residentID uuid.UUID,
) (CareIntensityPanel, error) {
	if client == nil {
		return CareIntensityPanel{}, fmt.Errorf("BuildCareIntensityPanel: nil SubstrateClient")
	}
	current, err := client.CurrentCareIntensity(ctx, residentID)
	if err != nil {
		return CareIntensityPanel{}, fmt.Errorf("CurrentCareIntensity: %w", err)
	}
	history, err := client.CareIntensityHistory(ctx, residentID)
	if err != nil {
		return CareIntensityPanel{}, fmt.Errorf("CareIntensityHistory: %w", err)
	}
	if history == nil {
		history = []substrate_types.CareIntensityEntry{}
	}
	panel := CareIntensityPanel{
		Current:        current,
		History:        history,
		SparseDataFlag: len(history) < 2,
		SubstrateRefs:  []SubstrateRef{},
	}
	if current != nil {
		panel.SubstrateRefs = append(panel.SubstrateRefs, SubstrateRef{
			Source: "kb-20-care-intensity",
			ID:     current.SubstrateID,
			Description: fmt.Sprintf(
				"care intensity %s effective %s",
				current.Tag,
				current.EffectiveDate.Format("2006-01-02"),
			),
		})
	}
	return panel, nil
}

// normalizeGoCState maps any of the s2 mirror, Wave 2.4 canonical, and
// legacy short forms to a normalized lowercase string for equality.
// Equality groups:
//
//	active_treatment ≡ active
//	comfort_focused  ≡ comfort
//	palliative       (no legacy difference)
//	end_of_life      (legacy passthrough; not in Wave 2.4)
//	rehabilitation   (no legacy difference)
func normalizeGoCState(s string) string {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "active", "active_treatment":
		return "active_treatment"
	case "comfort", "comfort_focused":
		return "comfort_focused"
	}
	return v
}

// DetectGoalsConflicts surfaces pending-recommendation vs GoC
// misalignments per v1.0 Part 9.4. Empty (non-nil) slice when no
// conflicts. Logic mirrors
// SubstrateBackedScorer.scoreGoalsOfCareAlignment (kb-32 Phase
// 2-completion Task 2):
//
//	ADD on palliative / comfort_focused / end_of_life → conflict
//	(canonical anti-pattern; blocking-tier reason)
//
//	STOP psychotropic on active_treatment → informational potential
//	conflict (not blocking; surfaces deprescribing-pressure ambiguity)
//
// Restraint considered? Rule-ID-prefix heuristic for psychotropic per
// kb-32 isPsychotropicRule (PSYCH- / ANTIPSY- / BZD-).
func DetectGoalsConflicts(
	cards []PendingRecommendationCard,
	goc *substrate_types.GoalsOfCareEntry,
) []GoalsConflict {
	out := []GoalsConflict{}
	if goc == nil {
		return out
	}
	state := normalizeGoCState(goc.State)
	gocRef := SubstrateRef{
		Source:      "kb-20-goc",
		ID:          goc.SubstrateID,
		Description: fmt.Sprintf("current goals-of-care state %s", goc.State),
	}
	for _, c := range cards {
		recRef := SubstrateRef{
			Source:      "kb-32",
			ID:          c.RecommendationID,
			Description: fmt.Sprintf("%s recommendation", c.Type),
		}
		switch c.Type {
		case "ADD":
			if state == "palliative" || state == "comfort_focused" || state == "end_of_life" {
				out = append(out, GoalsConflict{
					RecommendationID: c.RecommendationID,
					ConflictReason: fmt.Sprintf(
						"ADD recommendation conflicts with %s goals-of-care (kb-32 canonical anti-pattern: ADD aggressive intervention on comfort/palliative posture)",
						goc.State,
					),
					CurrentGoCState:    goc.State,
					RecommendationType: c.Type,
					SubstrateRefs:      []SubstrateRef{gocRef, recRef},
				})
			}
		case "STOP":
			if state == "active_treatment" && isPsychotropicRuleID(c) {
				out = append(out, GoalsConflict{
					RecommendationID: c.RecommendationID,
					ConflictReason: fmt.Sprintf(
						"STOP psychotropic on %s goals-of-care — informational: deprescribing remains appropriate at any intensity but family/clinical conversation may be warranted",
						goc.State,
					),
					CurrentGoCState:    goc.State,
					RecommendationType: c.Type,
					SubstrateRefs:      []SubstrateRef{gocRef, recRef},
				})
			}
		}
	}
	return out
}

// isPsychotropicRuleID applies the same prefix heuristic kb-32 uses
// internally (isPsychotropicRule in substrate_scorer.go). The
// PendingRecommendationCard does not surface the RuleID directly, so
// we read it from the SubstrateRefs description prefix
// "%s recommendation (rule %s)" — fragile but acceptable for
// informational-only conflict surfacing. A cleaner adapter would
// thread RuleID onto the card; deferred to a follow-up.
func isPsychotropicRuleID(c PendingRecommendationCard) bool {
	for _, ref := range c.SubstrateRefs {
		if ref.Source != "kb-32" {
			continue
		}
		// Description format: "STOP recommendation (rule rule-STOP)" —
		// match the "rule <id>" segment case-insensitively.
		upper := strings.ToUpper(ref.Description)
		if strings.Contains(upper, "RULE PSYCH-") ||
			strings.Contains(upper, "RULE ANTIPSY-") ||
			strings.Contains(upper, "RULE BZD-") ||
			strings.Contains(upper, "RULE STOP_PSYCH_") ||
			strings.Contains(upper, "RULE STOP_BENZO_") {
			return true
		}
	}
	return false
}
