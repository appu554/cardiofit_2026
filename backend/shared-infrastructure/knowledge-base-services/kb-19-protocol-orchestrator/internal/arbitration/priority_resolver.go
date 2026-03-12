// Package arbitration implements the core protocol arbitration engine for KB-19.
package arbitration

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/models"
)

// PriorityResolver resolves protocol conflicts based on priority hierarchy.
// Priority order (highest to lowest):
//   1. Emergency (PriorityClass = 1) - Life-preserving/resuscitation
//   2. Acute (PriorityClass = 2) - Organ-failure stabilization
//   3. Morbidity (PriorityClass = 3) - Immediate morbidity prevention
//   4. Chronic (PriorityClass = 4) - Long-term optimization
type PriorityResolver struct {
	log *logrus.Entry
}

// NewPriorityResolver creates a new PriorityResolver.
func NewPriorityResolver(log *logrus.Entry) *PriorityResolver {
	return &PriorityResolver{
		log: log.WithField("component", "priority-resolver"),
	}
}

// Resolve resolves conflicts and produces ArbitratedDecisions.
func (pr *PriorityResolver) Resolve(evaluations []models.ProtocolEvaluation, conflicts []models.ConflictResolution) []models.ArbitratedDecision {
	var decisions []models.ArbitratedDecision

	// Sort evaluations by priority class
	sortedEvals := make([]models.ProtocolEvaluation, len(evaluations))
	copy(sortedEvals, evaluations)
	sort.Slice(sortedEvals, func(i, j int) bool {
		return sortedEvals[i].PriorityClass < sortedEvals[j].PriorityClass
	})

	// Build a map of losers from conflicts
	losers := make(map[string]models.ConflictResolution)
	for _, conflict := range conflicts {
		losers[conflict.Loser] = conflict
	}

	// Process each evaluation
	for _, eval := range sortedEvals {
		if !eval.IsApplicable {
			continue
		}

		// Check if this protocol lost a conflict
		if conflict, isLoser := losers[eval.ProtocolID]; isLoser {
			// Create a decision for the losing protocol
			decision := pr.createLoserDecision(eval, conflict)
			decisions = append(decisions, decision)
		} else if !eval.Contraindicated {
			// Create decisions for winning/non-conflicting protocols
			protocolDecisions := pr.createProtocolDecisions(eval)
			decisions = append(decisions, protocolDecisions...)
		}
	}

	pr.log.WithField("decisions_created", len(decisions)).Debug("Priority resolution complete")

	return decisions
}

// createLoserDecision creates a decision for a protocol that lost a conflict.
func (pr *PriorityResolver) createLoserDecision(eval models.ProtocolEvaluation, conflict models.ConflictResolution) models.ArbitratedDecision {
	decision := models.ArbitratedDecision{
		ID:                uuid.New(),
		DecisionType:      conflict.LoserOutcome,
		Target:            eval.ProtocolName,
		Rationale:         fmt.Sprintf("Protocol deferred due to conflict with %s", conflict.Winner),
		Evidence:          *models.NewEvidenceEnvelope(),
		SourceProtocol:    eval.ProtocolID, // Must be protocol ID for KB-3 pathway binding
		SourceProtocolID:  eval.ProtocolID,
		ArbitrationReason: conflict.Explanation,
		ConflictedWith:    conflict.Winner,
		ConflictType:      string(conflict.ConflictType),
		Urgency:           models.UrgencyScheduled,
	}

	// Add inference step for the conflict resolution
	decision.Evidence.AddInferenceStep(
		models.StepConflictResolution,
		"KB-19 Arbitration Engine",
		conflict.ResolutionRule,
		fmt.Sprintf("%s wins over %s", conflict.Winner, conflict.Loser),
		map[string]interface{}{
			"conflict_type": conflict.ConflictType,
			"winner":        conflict.Winner,
			"loser":         conflict.Loser,
		},
		conflict.Confidence,
	)

	return decision
}

// createProtocolDecisions creates decisions from a protocol's recommended actions.
func (pr *PriorityResolver) createProtocolDecisions(eval models.ProtocolEvaluation) []models.ArbitratedDecision {
	var decisions []models.ArbitratedDecision

	if len(eval.RecommendedActions) == 0 {
		// Create a single "protocol applies" decision if no specific actions
		decision := models.ArbitratedDecision{
			ID:               uuid.New(),
			DecisionType:     models.DecisionDo,
			Target:           eval.ProtocolName,
			Rationale:        eval.ApplicabilityReason,
			Evidence:         *models.NewEvidenceEnvelope(),
			SourceProtocol:   eval.ProtocolID, // Must be protocol ID for KB-3 pathway binding
			SourceProtocolID: eval.ProtocolID,
			Urgency:          pr.determineUrgency(eval.PriorityClass),
		}

		// Add inference step for protocol matching
		decision.Evidence.AddInferenceStep(
			models.StepProtocolMatch,
			"KB-19 Arbitration Engine",
			"Protocol trigger criteria evaluation",
			fmt.Sprintf("Protocol %s is applicable", eval.ProtocolID),
			map[string]interface{}{
				"cql_facts_used":    eval.CQLFactsUsed,
				"calculators_used":  eval.CalculatorsUsed,
				"priority_class":    eval.PriorityClass,
			},
			eval.Confidence,
		)

		decisions = append(decisions, decision)
	} else {
		// Create a decision for each recommended action
		for _, action := range eval.RecommendedActions {
			decision := models.ArbitratedDecision{
				ID:               uuid.New(),
				DecisionType:     models.DecisionDo,
				Target:           action.Target,
				TargetRxNorm:     action.RxNormCode,
				TargetSNOMED:     action.SNOMEDCode,
				Rationale:        action.Description,
				Evidence:         *models.NewEvidenceEnvelope(),
				SourceProtocol:   eval.ProtocolID, // Must be protocol ID for KB-3 pathway binding
				SourceProtocolID: eval.ProtocolID,
				Urgency:          action.Urgency,
			}

			decision.Evidence.AddInferenceStep(
				models.StepProtocolMatch,
				"KB-19 Arbitration Engine",
				fmt.Sprintf("Action from protocol %s", eval.ProtocolID),
				action.Description,
				map[string]interface{}{
					"action_type": action.ActionType,
					"target":      action.Target,
				},
				eval.Confidence,
			)

			decisions = append(decisions, decision)
		}
	}

	return decisions
}

// determineUrgency determines urgency based on priority class.
func (pr *PriorityResolver) determineUrgency(priorityClass models.PriorityClass) models.ActionUrgency {
	switch priorityClass {
	case models.PriorityEmergency:
		return models.UrgencySTAT
	case models.PriorityAcute:
		return models.UrgencyUrgent
	case models.PriorityMorbidity:
		return models.UrgencyRoutine
	case models.PriorityChronic:
		return models.UrgencyScheduled
	default:
		return models.UrgencyRoutine
	}
}
