// Package arbitration implements the core protocol arbitration engine for KB-19.
package arbitration

import (
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/models"
)

// ConflictDetector identifies conflicts between applicable protocols.
type ConflictDetector struct {
	log *logrus.Entry
}

// NewConflictDetector creates a new ConflictDetector.
func NewConflictDetector(log *logrus.Entry) *ConflictDetector {
	return &ConflictDetector{
		log: log.WithField("component", "conflict-detector"),
	}
}

// DetectConflicts identifies conflicts between the given protocol evaluations.
func (cd *ConflictDetector) DetectConflicts(evaluations []models.ProtocolEvaluation) []models.ConflictResolution {
	var conflicts []models.ConflictResolution

	// Only consider applicable protocols
	applicable := make([]models.ProtocolEvaluation, 0)
	for _, eval := range evaluations {
		if eval.IsApplicable && !eval.Contraindicated {
			applicable = append(applicable, eval)
		}
	}

	// Check each pair of applicable protocols for conflicts
	for i := 0; i < len(applicable); i++ {
		for j := i + 1; j < len(applicable); j++ {
			if conflict := cd.checkConflict(applicable[i], applicable[j]); conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	cd.log.WithField("conflicts_found", len(conflicts)).Debug("Conflict detection complete")

	return conflicts
}

// checkConflict checks if two protocols conflict.
func (cd *ConflictDetector) checkConflict(evalA, evalB models.ProtocolEvaluation) *models.ConflictResolution {
	// Look up in predefined conflict matrix
	conflictEntry := models.FindConflict(evalA.ProtocolID, evalB.ProtocolID)
	if conflictEntry == nil {
		return nil
	}

	cd.log.WithFields(logrus.Fields{
		"protocol_a":    evalA.ProtocolID,
		"protocol_b":    evalB.ProtocolID,
		"conflict_type": conflictEntry.ConflictType,
	}).Debug("Conflict detected")

	// Determine winner based on resolution rule
	winner := conflictEntry.Resolution.Winner
	loser := evalA.ProtocolID
	if winner == evalA.ProtocolID {
		loser = evalB.ProtocolID
	}

	// Determine loser outcome
	loserOutcome := models.DecisionDelay
	switch conflictEntry.Resolution.LoserOutcome {
	case "DELAY":
		loserOutcome = models.DecisionDelay
	case "AVOID":
		loserOutcome = models.DecisionAvoid
	case "MODIFY":
		loserOutcome = models.DecisionConsider
	}

	return &models.ConflictResolution{
		ProtocolA:      evalA.ProtocolID,
		ProtocolB:      evalB.ProtocolID,
		ConflictType:   conflictEntry.ConflictType,
		Winner:         winner,
		Loser:          loser,
		ResolutionRule: conflictEntry.Resolution.Condition,
		Explanation:    conflictEntry.ClinicalRationale,
		LoserOutcome:   loserOutcome,
		Confidence:     conflictEntry.Resolution.Confidence,
	}
}
