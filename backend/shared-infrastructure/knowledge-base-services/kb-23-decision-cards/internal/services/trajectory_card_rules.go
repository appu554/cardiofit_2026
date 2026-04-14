package services

import (
	"fmt"
	"strings"
	"time"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// TrajectoryCard represents a decision card generated from domain trajectory analysis.
type TrajectoryCard struct {
	CardType  string   `json:"card_type"`
	Urgency   string   `json:"urgency"`
	Title     string   `json:"title"`
	Rationale string   `json:"rationale"`
	Actions   []string `json:"actions"`
	Domain    string   `json:"domain,omitempty"`
}

// EvaluateTrajectoryCards generates decision cards from decomposed domain trajectories.
// Card priority order:
//  1. CONCORDANT_DETERIORATION (IMMEDIATE if >=3 domains, URGENT if 2)
//  2. DOMAIN_DIVERGENCE (URGENT)
//  3. BEHAVIORAL_LEADING_INDICATOR (URGENT)
//  4. DOMAIN_RAPID_DECLINE (URGENT, only if not already covered by concordant)
//  5. DOMAIN_CATEGORY_CROSSING (ROUTINE)
func EvaluateTrajectoryCards(traj *dtModels.DecomposedTrajectory) []TrajectoryCard {
	if traj == nil {
		return nil
	}

	var cards []TrajectoryCard

	// 1. Concordant deterioration — highest priority.
	if traj.ConcordantDeterioration {
		var decliningDomains []string
		// Iterate AllMHRIDomains for deterministic ordering.
		// Use Trend field (not raw slope) to stay consistent with KB-26's
		// DomainsDeteriorating count — avoids title/rationale mismatch.
		for _, domain := range dtModels.AllMHRIDomains {
			ds := traj.DomainSlopes[domain]
			if ds.Trend == dtModels.TrendDeclining || ds.Trend == dtModels.TrendRapidDeclining {
				decliningDomains = append(decliningDomains, string(domain))
			}
		}

		urgency := "URGENT"
		if traj.DomainsDeteriorating >= 3 {
			urgency = "IMMEDIATE"
		}

		cards = append(cards, TrajectoryCard{
			CardType: "CONCORDANT_DETERIORATION",
			Urgency:  urgency,
			Title:    fmt.Sprintf("Multi-Domain Deterioration — %d Domains Declining", traj.DomainsDeteriorating),
			Rationale: fmt.Sprintf("Simultaneous decline across %s. Concordant multi-domain "+
				"deterioration indicates systemic worsening — risk is multiplicative, not additive "+
				"(AHA CKM Framework). Composite MHRI slope: %.2f/day.",
				strings.Join(decliningDomains, ", "), traj.CompositeSlope),
			Actions: []string{
				"Comprehensive multi-domain medication review",
				"Identify root cause: medication non-adherence, intercurrent illness, lifestyle change",
				"Consider dual-benefit agents (SGLT2i for glucose + BP + renal)",
				"Schedule urgent clinical review within 1 week",
			},
		})
	}

	// 2. Domain divergence.
	for _, div := range traj.Divergences {
		cards = append(cards, TrajectoryCard{
			CardType: "DOMAIN_DIVERGENCE",
			Urgency:  "URGENT",
			Title: fmt.Sprintf("Discordant Trajectory — %s Improving, %s Declining",
				div.ImprovingDomain, div.DecliningDomain),
			Rationale: fmt.Sprintf("%s (slope +%.2f/day) while %s (slope %.2f/day). %s",
				div.ImprovingDomain, div.ImprovingSlope,
				div.DecliningDomain, div.DecliningSlope,
				div.ClinicalConcern),
			Actions: []string{
				div.PossibleMechanism,
				"Review whether improvement in one domain is masking deterioration in another",
				"Consider medication with cross-domain benefit",
			},
			Domain: string(div.DecliningDomain),
		})
	}

	// 3. Behavioral leading indicator.
	for _, lead := range traj.LeadingIndicators {
		laggingNames := make([]string, len(lead.LaggingDomains))
		for i, d := range lead.LaggingDomains {
			laggingNames[i] = string(d)
		}

		cards = append(cards, TrajectoryCard{
			CardType: "BEHAVIORAL_LEADING_INDICATOR",
			Urgency:  "URGENT",
			Title:    "Engagement Collapse Preceding Clinical Deterioration",
			Rationale: fmt.Sprintf("Behavioral domain declining before %s. %s "+
				"Clinical evidence shows behavioral disengagement predicts clinical "+
				"deterioration by 2-4 weeks. Intervene now to prevent further clinical decline.",
				strings.Join(laggingNames, " and "), lead.Interpretation),
			Actions: []string{
				"Clinical outreach — phone call preferred over digital notification",
				"Assess barriers to engagement: health anxiety, cost, side effects, access",
				"Do NOT default to app engagement nudge — this is a clinical signal, not a UX problem",
			},
		})
	}

	// 4. Single domain rapid decline (only if not already covered by concordant).
	if !traj.ConcordantDeterioration {
		// Iterate AllMHRIDomains for deterministic order.
		for _, domain := range dtModels.AllMHRIDomains {
			ds := traj.DomainSlopes[domain]
			if ds.Trend == dtModels.TrendRapidDeclining && ds.Confidence != dtModels.ConfidenceLow {
				cards = append(cards, TrajectoryCard{
					CardType: "DOMAIN_RAPID_DECLINE",
					Urgency:  "URGENT",
					Title:    fmt.Sprintf("%s Domain Rapid Decline", domain),
					Rationale: fmt.Sprintf("%s domain declining at %.2f/day (R²=%.2f, %s confidence). "+
						"Score dropped from %.0f to %.0f over the observation window.",
						domain, ds.SlopePerDay, ds.R2, ds.Confidence, ds.StartScore, ds.EndScore),
					Actions: []string{
						fmt.Sprintf("Review %s domain clinical data and recent changes", domain),
						"Investigate cause of rapid decline",
					},
					Domain: string(domain),
				})
			}
		}
	}

	// 5. Domain category crossing.
	// Only surface worsening crossings; improvements do not warrant a decision card.
	for _, crossing := range traj.DomainCrossings {
		if crossing.Direction == dtModels.DirectionWorsened {
			cards = append(cards, TrajectoryCard{
				CardType: "DOMAIN_CATEGORY_CROSSING",
				Urgency:  "ROUTINE",
				Title: fmt.Sprintf("%s Domain: %s → %s",
					crossing.Domain, crossing.PrevCategory, crossing.CurrCategory),
				Rationale: fmt.Sprintf("%s domain crossed from %s to %s status. "+
					"This threshold crossing may indicate need for therapy adjustment.",
					crossing.Domain, crossing.PrevCategory, crossing.CurrCategory),
				Actions: []string{
					fmt.Sprintf("Review %s domain for therapy adjustment in light of category change", crossing.Domain),
				},
				Domain: string(crossing.Domain),
			})
		}
	}

	return cards
}

// EvaluateTrajectoryCardsWithSeasonalContext is the season-aware variant of
// EvaluateTrajectoryCards. Single-domain cards (DOMAIN_RAPID_DECLINE,
// DOMAIN_DIVERGENCE, DOMAIN_CATEGORY_CROSSING) can be downgraded or suppressed
// if the seasonal context marks their domain as affected at `now`.
// CONCORDANT_DETERIORATION and BEHAVIORAL_LEADING_INDICATOR are never
// seasonally suppressed — multi-domain risk and engagement collapse are
// clinically significant regardless of season.
func EvaluateTrajectoryCardsWithSeasonalContext(
	traj *dtModels.DecomposedTrajectory,
	seasonalCtx *SeasonalContext,
	now time.Time,
) []TrajectoryCard {
	if traj == nil {
		return nil
	}

	cards := EvaluateTrajectoryCards(traj)
	if seasonalCtx == nil {
		return cards
	}

	filtered := make([]TrajectoryCard, 0, len(cards))
	for _, card := range cards {
		// CONCORDANT and BEHAVIORAL_LEADING_INDICATOR always pass through.
		if card.CardType == "CONCORDANT_DETERIORATION" || card.CardType == "BEHAVIORAL_LEADING_INDICATOR" {
			filtered = append(filtered, card)
			continue
		}

		// Single-domain cards: check seasonal context.
		if card.Domain == "" {
			filtered = append(filtered, card)
			continue
		}

		domain := dtModels.MHRIDomain(card.Domain)
		suppress, downgrade, rationale := seasonalCtx.ShouldSuppress(domain, now)
		if suppress {
			continue // do not emit
		}
		if downgrade {
			card.Urgency = downgradeUrgency(card.Urgency)
			card.Rationale += " (seasonal context: " + rationale + ")"
		}
		filtered = append(filtered, card)
	}

	return filtered
}

// downgradeUrgency returns the next less-urgent urgency level.
func downgradeUrgency(urgency string) string {
	switch urgency {
	case "IMMEDIATE":
		return "URGENT"
	case "URGENT":
		return "ROUTINE"
	default:
		return urgency // ROUTINE stays ROUTINE
	}
}
