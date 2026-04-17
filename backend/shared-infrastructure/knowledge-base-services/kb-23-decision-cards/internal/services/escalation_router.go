package services

import (
	"fmt"
	"sync"
	"time"

	"kb-23-decision-cards/internal/models"
)

// RoutingResult is the output of the escalation router.
type RoutingResult struct {
	Tier              models.EscalationTier
	Reason            string
	Suppressed        bool
	SuppressionReason string
}

// EscalationRouterInput carries everything the router needs.
type EscalationRouterInput struct {
	// Card-based routing
	CardDifferentialID string // e.g. "RENAL_CONTRAINDICATION"
	MCUGate            string // "HALT", "MODIFY", "SAFE"

	// PAI-based routing
	PAITier  string  // "CRITICAL", "HIGH", etc.
	PAIScore float64
	EGFR     *float64

	// Context
	PatientID string
}

// EscalationProtocolConfig holds the YAML-driven routing tables.
type EscalationProtocolConfig struct {
	CardTypeRouting map[string]string // DifferentialID -> tier
	PAIRouting      map[string]string // PAI tier -> escalation tier
	SustainedElevation struct {
		Enabled        bool
		MinConsecutive int
		ExemptTiers    []string
	}
	DeduplicationWindowHours int
}

// EscalationRouter implements the 5-stage escalation routing pipeline.
type EscalationRouter struct {
	config         *EscalationProtocolConfig
	sustainedCount map[string]int       // patientID -> consecutive above-threshold count
	recentRouted   map[string]time.Time // "patientID:cardType" -> last routed time
	mu             sync.Mutex
}

// NewEscalationRouter creates a new router with the given config.
func NewEscalationRouter(config *EscalationProtocolConfig) *EscalationRouter {
	return &EscalationRouter{
		config:         config,
		sustainedCount: make(map[string]int),
		recentRouted:   make(map[string]time.Time),
	}
}

// DefaultEscalationProtocolConfig returns the standard config matching the YAML spec.
func DefaultEscalationProtocolConfig() *EscalationProtocolConfig {
	return &EscalationProtocolConfig{
		CardTypeRouting: map[string]string{
			"RENAL_CONTRAINDICATION":      "SAFETY",
			"RENAL_DOSE_REDUCE":           "URGENT",
			"CKM_4C_MANDATORY_MEDICATION": "IMMEDIATE",
			"THERAPEUTIC_INERTIA":         "URGENT",
			"DUAL_DOMAIN_INERTIA":         "IMMEDIATE",
			"MASKED_HYPERTENSION":         "URGENT",
			"ADHERENCE_GAP":               "ROUTINE",
			"DEPRESCRIBING_REVIEW":        "ROUTINE",
			"PHENOTYPE_TRANSITION":        "ROUTINE",
			"PHENOTYPE_FLAP_WARNING":      "ROUTINE",
			"MONITORING_LAPSED":           "URGENT",
		},
		PAIRouting: map[string]string{
			"CRITICAL": "IMMEDIATE",
			"HIGH":     "URGENT",
			"MODERATE": "ROUTINE",
			"LOW":      "INFORMATIONAL",
			"MINIMAL":  "INFORMATIONAL",
		},
		SustainedElevation: struct {
			Enabled        bool
			MinConsecutive int
			ExemptTiers    []string
		}{
			Enabled:        true,
			MinConsecutive: 2,
			ExemptTiers:    []string{"SAFETY"},
		},
		DeduplicationWindowHours: 24,
	}
}

// RouteCard runs the 5-stage escalation routing pipeline.
func (r *EscalationRouter) RouteCard(input EscalationRouterInput) *RoutingResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := &RoutingResult{}
	var reasons []string

	// ── Stage 1: Card type routing ──────────────────────────────────────
	tier := models.TierInformational
	if input.CardDifferentialID != "" {
		if mapped, ok := r.config.CardTypeRouting[input.CardDifferentialID]; ok {
			tier = models.EscalationTier(mapped)
			reasons = append(reasons, fmt.Sprintf("card_type=%s->%s", input.CardDifferentialID, mapped))
		}
	}

	// ── Stage 2: PAI-based routing (take higher tier) ───────────────────
	if input.PAITier != "" {
		if mapped, ok := r.config.PAIRouting[input.PAITier]; ok {
			paiTier := models.EscalationTier(mapped)
			if tierPriority(paiTier) > tierPriority(tier) {
				tier = paiTier
				reasons = append(reasons, fmt.Sprintf("pai_tier=%s->%s", input.PAITier, mapped))
			}
		}
	}

	// ── Stage 3: Amplification ──────────────────────────────────────────
	if input.MCUGate == "HALT" {
		tier = models.TierSafety
		reasons = append(reasons, "amplified: MCU HALT gate")
	}
	if input.PAITier == "CRITICAL" && input.EGFR != nil && *input.EGFR < 30 {
		tier = models.TierSafety
		reasons = append(reasons, fmt.Sprintf("amplified: PAI CRITICAL + eGFR=%.0f<30", *input.EGFR))
	}

	result.Tier = tier
	result.Reason = joinReasons(reasons)

	// ── Stage 4: Sustained-elevation gate ───────────────────────────────
	// Only applies to PAI-triggered routing (sustained PAI elevation check).
	if r.config.SustainedElevation.Enabled && input.PAITier != "" && !isTierExempt(string(tier), r.config.SustainedElevation.ExemptTiers) {
		if tierPriority(tier) <= tierPriority(models.TierRoutine) {
			// Tier is ROUTINE or below — reset counter
			r.sustainedCount[input.PatientID] = 0
		} else {
			r.sustainedCount[input.PatientID]++
			if r.sustainedCount[input.PatientID] < r.config.SustainedElevation.MinConsecutive {
				result.Suppressed = true
				result.SuppressionReason = "awaiting sustained confirmation"
				return result
			}
		}
	}

	// ── Stage 5: Deduplication ──────────────────────────────────────────
	dedupKey := fmt.Sprintf("%s:%s", input.PatientID, dedupCardType(input))
	window := time.Duration(r.config.DeduplicationWindowHours) * time.Hour
	if lastTime, ok := r.recentRouted[dedupKey]; ok {
		if time.Since(lastTime) < window {
			result.Suppressed = true
			result.SuppressionReason = "deduplicated"
			return result
		}
	}
	r.recentRouted[dedupKey] = time.Now()

	return result
}

// tierPriority maps an EscalationTier to a numeric priority for comparison.
func tierPriority(tier models.EscalationTier) int {
	switch tier {
	case models.TierSafety:
		return 5
	case models.TierImmediate:
		return 4
	case models.TierUrgent:
		return 3
	case models.TierRoutine:
		return 2
	case models.TierInformational:
		return 1
	default:
		return 0
	}
}

// isTierExempt checks if the tier string is in the exempt list.
func isTierExempt(tier string, exemptTiers []string) bool {
	for _, e := range exemptTiers {
		if e == tier {
			return true
		}
	}
	return false
}

// dedupCardType returns the deduplication key component for the card.
func dedupCardType(input EscalationRouterInput) string {
	if input.CardDifferentialID != "" {
		return input.CardDifferentialID
	}
	if input.PAITier != "" {
		return "PAI:" + input.PAITier
	}
	return "UNKNOWN"
}

// joinReasons concatenates routing reasons into a single string.
func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "default"
	}
	result := reasons[0]
	for i := 1; i < len(reasons); i++ {
		result += "; " + reasons[i]
	}
	return result
}
