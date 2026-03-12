// Package arbitration implements the core protocol arbitration engine for KB-19.
package arbitration

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/models"
)

// NarrativeGenerator creates human-readable explanations of recommendations.
type NarrativeGenerator struct {
	log *logrus.Entry
}

// NewNarrativeGenerator creates a new NarrativeGenerator.
func NewNarrativeGenerator(log *logrus.Entry) *NarrativeGenerator {
	return &NarrativeGenerator{
		log: log.WithField("component", "narrative-generator"),
	}
}

// Generate creates a human-readable narrative from a recommendation bundle.
func (ng *NarrativeGenerator) Generate(bundle *models.RecommendationBundle) string {
	var sections []string

	// Header
	sections = append(sections, ng.generateHeader(bundle))

	// Executive Summary
	sections = append(sections, ng.generateExecutiveSummary(bundle))

	// Key Recommendations
	if len(bundle.Decisions) > 0 {
		sections = append(sections, ng.generateRecommendations(bundle))
	}

	// Conflicts Resolved
	if len(bundle.ConflictsResolved) > 0 {
		sections = append(sections, ng.generateConflictNarrative(bundle))
	}

	// Safety Considerations
	if len(bundle.SafetyGatesApplied) > 0 {
		sections = append(sections, ng.generateSafetyNarrative(bundle))
	}

	// Alerts
	if len(bundle.Alerts) > 0 {
		sections = append(sections, ng.generateAlerts(bundle))
	}

	// Footer
	sections = append(sections, ng.generateFooter(bundle))

	return strings.Join(sections, "\n\n")
}

// generateHeader creates the narrative header.
func (ng *NarrativeGenerator) generateHeader(bundle *models.RecommendationBundle) string {
	return fmt.Sprintf(`═══════════════════════════════════════════════════════════════════════════════
                    CLINICAL DECISION SUPPORT SUMMARY
═══════════════════════════════════════════════════════════════════════════════
Generated: %s
Patient ID: %s
Encounter ID: %s
Bundle ID: %s`,
		bundle.Timestamp.Format("2006-01-02 15:04:05 MST"),
		bundle.PatientID,
		bundle.EncounterID,
		bundle.ID)
}

// generateExecutiveSummary creates the executive summary section.
func (ng *NarrativeGenerator) generateExecutiveSummary(bundle *models.RecommendationBundle) string {
	summary := bundle.ExecutiveSummary

	urgencySymbol := ng.getUrgencySymbol(summary.HighestUrgency)

	return fmt.Sprintf(`EXECUTIVE SUMMARY
─────────────────────────────────────────────────────────────────────────────────
Protocols Evaluated: %d | Applicable: %d | Conflicts Resolved: %d | Safety Blocks: %d

Highest Urgency: %s %s

Decisions:
  ✅ DO:       %d actions
  ⏳ DELAY:    %d actions
  ⛔ AVOID:    %d actions
  🤔 CONSIDER: %d actions`,
		summary.ProtocolsEvaluated,
		summary.ProtocolsApplicable,
		summary.ConflictsDetected,
		summary.SafetyBlocks,
		urgencySymbol,
		summary.HighestUrgency,
		summary.DecisionsByType[models.DecisionDo],
		summary.DecisionsByType[models.DecisionDelay],
		summary.DecisionsByType[models.DecisionAvoid],
		summary.DecisionsByType[models.DecisionConsider])
}

// generateRecommendations creates the recommendations section.
func (ng *NarrativeGenerator) generateRecommendations(bundle *models.RecommendationBundle) string {
	var lines []string
	lines = append(lines, `KEY RECOMMENDATIONS
─────────────────────────────────────────────────────────────────────────────────`)

	// Group by decision type
	doDecisions := bundle.GetDODecisions()
	avoidDecisions := bundle.GetAVOIDDecisions()

	if len(doDecisions) > 0 {
		lines = append(lines, "\n✅ RECOMMENDED ACTIONS:")
		for i, d := range doDecisions {
			if i >= 5 { // Limit to top 5
				lines = append(lines, fmt.Sprintf("   ... and %d more", len(doDecisions)-5))
				break
			}
			urgencySymbol := ng.getUrgencySymbol(d.Urgency)
			classSymbol := ng.getClassSymbol(d.Evidence.RecommendationClass)
			lines = append(lines, fmt.Sprintf("   %s %s %s: %s",
				urgencySymbol,
				classSymbol,
				d.Target,
				d.Rationale))
		}
	}

	if len(avoidDecisions) > 0 {
		lines = append(lines, "\n⛔ ACTIONS TO AVOID:")
		for i, d := range avoidDecisions {
			if i >= 5 {
				lines = append(lines, fmt.Sprintf("   ... and %d more", len(avoidDecisions)-5))
				break
			}
			lines = append(lines, fmt.Sprintf("   ⛔ %s: %s", d.Target, d.Rationale))
			if len(d.SafetyFlags) > 0 {
				lines = append(lines, fmt.Sprintf("      Reason: %s", d.SafetyFlags[0].Reason))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// generateConflictNarrative creates the conflict resolution section.
func (ng *NarrativeGenerator) generateConflictNarrative(bundle *models.RecommendationBundle) string {
	var lines []string
	lines = append(lines, `PROTOCOL CONFLICTS RESOLVED
─────────────────────────────────────────────────────────────────────────────────`)

	for _, conflict := range bundle.ConflictsResolved {
		lines = append(lines, fmt.Sprintf(`
⚖️  %s vs %s
    Type: %s
    Winner: %s
    Reason: %s
    Confidence: %.0f%%`,
			conflict.ProtocolA,
			conflict.ProtocolB,
			conflict.ConflictType,
			conflict.Winner,
			conflict.Explanation,
			conflict.Confidence*100))
	}

	return strings.Join(lines, "\n")
}

// generateSafetyNarrative creates the safety considerations section.
func (ng *NarrativeGenerator) generateSafetyNarrative(bundle *models.RecommendationBundle) string {
	var lines []string
	lines = append(lines, `SAFETY CONSIDERATIONS
─────────────────────────────────────────────────────────────────────────────────`)

	for _, gate := range bundle.SafetyGatesApplied {
		if !gate.Triggered {
			continue
		}

		symbol := "ℹ️"
		if gate.Result == "BLOCK" {
			symbol = "🛑"
		} else if gate.Result == "WARN" {
			symbol = "⚠️"
		}

		lines = append(lines, fmt.Sprintf("\n%s %s [%s]", symbol, gate.Name, gate.Result))
		lines = append(lines, fmt.Sprintf("   %s", gate.Details))
		if len(gate.AffectedDecisions) > 0 {
			lines = append(lines, fmt.Sprintf("   Affected decisions: %d", len(gate.AffectedDecisions)))
		}
	}

	return strings.Join(lines, "\n")
}

// generateAlerts creates the alerts section.
func (ng *NarrativeGenerator) generateAlerts(bundle *models.RecommendationBundle) string {
	var lines []string
	lines = append(lines, `CLINICAL ALERTS
─────────────────────────────────────────────────────────────────────────────────`)

	for _, alert := range bundle.Alerts {
		symbol := "ℹ️"
		switch alert.Severity {
		case "CRITICAL":
			symbol = "🚨"
		case "HIGH":
			symbol = "⚠️"
		case "MEDIUM":
			symbol = "📢"
		}

		ackRequired := ""
		if alert.RequiresAck {
			ackRequired = " [ACKNOWLEDGMENT REQUIRED]"
		}

		lines = append(lines, fmt.Sprintf("\n%s %s%s", symbol, alert.Message, ackRequired))
	}

	return strings.Join(lines, "\n")
}

// generateFooter creates the narrative footer.
func (ng *NarrativeGenerator) generateFooter(bundle *models.RecommendationBundle) string {
	return fmt.Sprintf(`═══════════════════════════════════════════════════════════════════════════════
Processing Time: %dms | Service Versions: KB-19 v1.0.0
NOTE: This is clinical decision support. Final decisions rest with the treating clinician.
═══════════════════════════════════════════════════════════════════════════════`,
		bundle.ProcessingMetrics.TotalDurationMs)
}

// getUrgencySymbol returns an emoji for the urgency level.
func (ng *NarrativeGenerator) getUrgencySymbol(urgency models.ActionUrgency) string {
	switch urgency {
	case models.UrgencySTAT:
		return "🔴"
	case models.UrgencyUrgent:
		return "🟠"
	case models.UrgencyRoutine:
		return "🟡"
	case models.UrgencyScheduled:
		return "🟢"
	default:
		return "⚪"
	}
}

// getClassSymbol returns a symbol for the recommendation class.
func (ng *NarrativeGenerator) getClassSymbol(class models.RecommendationClass) string {
	switch class {
	case models.ClassI:
		return "[Class I]"
	case models.ClassIIa:
		return "[Class IIa]"
	case models.ClassIIb:
		return "[Class IIb]"
	case models.ClassIII:
		return "[Class III]"
	default:
		return ""
	}
}
