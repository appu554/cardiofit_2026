package aggregation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// RestraintSignalCard is the per-card rendering unit for the S2
// restraint-signals panel per v1.0 Part 7.1. Phase 1 commitment: every
// restraint signal is surfaced as ADVISORY ONLY — the platform does NOT
// auto-suppress paired recommendations (Part 7.3).
//
// TransitionCriteriaSatisfied is INFORMATIONAL ONLY per v1.0 Part 7.3:
// the renderer MUST NOT use this field to filter signals out of the
// panel. The field exists so the panel can surface platform-level
// transition status (e.g., "pilot evidence accumulated: 6 weeks"), not
// so individual signals are auto-suppressed.
type RestraintSignalCard struct {
	SignalID                    uuid.UUID
	Type                        string
	Severity                    int
	PairedRecommendationID      uuid.UUID
	TriggeredAt                 time.Time
	PharmacistAcknowledged      bool
	AcknowledgedDecision        string
	BypassReasoning             string
	TransitionCriteriaSatisfied bool // INFORMATIONAL ONLY — see type doc.
	SubstrateRefs               []SubstrateRef
}

// RestraintAcknowledgmentRequest is the inbound payload for the
// acknowledgment workflow per v1.0 Part 7.2 + Part 7.4. BypassReasoning
// is MANDATORY when Decision = invoke_safety_critical_bypass; optional
// otherwise.
type RestraintAcknowledgmentRequest struct {
	SignalID        uuid.UUID
	PharmacistID    uuid.UUID
	Decision        string
	BypassReasoning string
}

// Validate enforces the Part 7.4 safety-critical bypass contract:
// reasoning is mandatory for bypass. Returns nil when the request is
// internally consistent.
func (r RestraintAcknowledgmentRequest) Validate() error {
	if r.SignalID == uuid.Nil {
		return fmt.Errorf("restraint acknowledgment: signal_id must not be nil")
	}
	if r.PharmacistID == uuid.Nil {
		return fmt.Errorf("restraint acknowledgment: pharmacist_id must not be nil")
	}
	switch r.Decision {
	case substrate_types.RestraintDecisionAcknowledgeAdvisory:
		// reasoning optional
	case substrate_types.RestraintDecisionSafetyCriticalBypass:
		if strings.TrimSpace(r.BypassReasoning) == "" {
			return fmt.Errorf("restraint acknowledgment: bypass_reasoning is mandatory for safety_critical_bypass (v1.0 Part 7.4)")
		}
	default:
		return fmt.Errorf("restraint acknowledgment: unknown decision %q", r.Decision)
	}
	return nil
}

// BuildRestraintSignalCards composes the per-resident restraint-signals
// panel per v1.0 Part 7. The aggregator does NOT filter by transition
// criteria — Phase 1 commitment per v1.0 Part 7.3 is informational only.
//
// Empty-state: returns a non-nil empty slice when no signals are active.
func BuildRestraintSignalCards(
	ctx context.Context,
	client SubstrateClient,
	residentID uuid.UUID,
) ([]RestraintSignalCard, error) {
	if client == nil {
		return nil, fmt.Errorf("BuildRestraintSignalCards: nil SubstrateClient")
	}

	sigs, err := client.ActiveRestraintSignals(ctx, residentID)
	if err != nil {
		return nil, fmt.Errorf("ActiveRestraintSignals: %w", err)
	}

	out := make([]RestraintSignalCard, 0, len(sigs))
	for _, s := range sigs {
		out = append(out, RestraintSignalCard{
			SignalID:               s.SignalID,
			Type:                   s.Type,
			Severity:               s.Severity,
			PairedRecommendationID: s.PairedRecommendationID,
			TriggeredAt:            s.TriggeredAt,
			// Phase 1: TransitionCriteriaSatisfied always false here;
			// platform-level status is computed elsewhere (kb-29 work).
			// Field is informational only — renderer MUST NOT auto-suppress.
			TransitionCriteriaSatisfied: false,
			SubstrateRefs: []SubstrateRef{{
				Source:      s.SubstrateSource,
				ID:          s.SubstrateID,
				Description: fmt.Sprintf("restraint signal %s (severity %d)", s.Type, s.Severity),
			}},
		})
	}
	return out, nil
}

// PairRestraintSignalsWithRecommendations attaches each restraint
// signal's pointer to the matching PendingRecommendationCard's
// PairedRestraintSignal field per v1.0 Part 6.4. Signals with
// PairedRecommendationID == uuid.Nil are panel-level (not card-level)
// and are not attached here.
//
// This function is idempotent: calling it twice with the same inputs
// produces the same result. The cards slice is returned (in place
// modification of the underlying array) for fluent chaining.
func PairRestraintSignalsWithRecommendations(
	signals []RestraintSignalCard,
	cards []PendingRecommendationCard,
) []PendingRecommendationCard {
	if len(signals) == 0 || len(cards) == 0 {
		return cards
	}
	signalByRec := make(map[uuid.UUID]*substrate_types.RestraintSignal, len(signals))
	for i := range signals {
		s := signals[i]
		if s.PairedRecommendationID == uuid.Nil {
			continue
		}
		signalByRec[s.PairedRecommendationID] = &substrate_types.RestraintSignal{
			SignalID:               s.SignalID,
			Type:                   s.Type,
			Severity:               s.Severity,
			PairedRecommendationID: s.PairedRecommendationID,
			TriggeredAt:            s.TriggeredAt,
		}
	}
	for i := range cards {
		if sig, ok := signalByRec[cards[i].RecommendationID]; ok {
			cards[i].PairedRestraintSignal = sig
		}
	}
	return cards
}
