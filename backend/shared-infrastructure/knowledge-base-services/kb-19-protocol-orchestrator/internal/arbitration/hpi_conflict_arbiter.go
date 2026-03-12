// Package arbitration implements the core protocol arbitration engine for KB-19.
//
// hpi_conflict_arbiter.go implements BAY-7: Conflict Arbiter for Multi-Node HPI Sessions.
//
// When a patient presents with multiple complaints (e.g., dizziness + chest pain),
// KB-22 runs independent Bayesian sessions per node. The Conflict Arbiter is a
// post-processing layer that reconciles results AFTER both nodes complete.
//
// Four scenarios:
//   - BOOST:          Same diagnosis in top-3 of both nodes → over-determined
//   - FLAG:           Node A supports dx X, Node B opposes same dx X → conflict
//   - REPORT:         Independent top diagnoses (most common) → report both
//   - RED_FLAG_WINS:  One node escalated → escalation always wins
package arbitration

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// HPIConflictAction enumerates the arbiter's output actions.
type HPIConflictAction string

const (
	// ActionBoost: same diagnosis over-determined by independent evidence from multiple nodes.
	ActionBoost HPIConflictAction = "BOOST"
	// ActionFlag: contradictory evidence across nodes — clinician must resolve.
	ActionFlag HPIConflictAction = "FLAG"
	// ActionReport: independent top diagnoses — report both (most common case).
	ActionReport HPIConflictAction = "REPORT"
	// ActionRedFlagWins: one node has safety escalation — escalation always wins.
	ActionRedFlagWins HPIConflictAction = "RED_FLAG_WINS"
)

// HPINodeResult represents the completed output of a single KB-22 HPI node session.
type HPINodeResult struct {
	SessionID   uuid.UUID              `json:"session_id"`
	NodeID      string                 `json:"node_id"`
	PatientID   uuid.UUID              `json:"patient_id"`
	TopDx       []RankedDifferential   `json:"top_differentials"`
	SafetyFlags []string               `json:"safety_flags,omitempty"`
	Escalated   bool                   `json:"escalated"`
	Posterior   map[string]float64     `json:"posteriors"`
	CompletedAt time.Time              `json:"completed_at"`
}

// RankedDifferential is a diagnosis with its posterior probability.
type RankedDifferential struct {
	DxID      string  `json:"dx_id"`
	Label     string  `json:"label"`
	Posterior float64 `json:"posterior"`
}

// ArbiterResult is the output of the Conflict Arbiter for a multi-node encounter.
type ArbiterResult struct {
	PatientID      uuid.UUID               `json:"patient_id"`
	Action         HPIConflictAction        `json:"action"`
	NodeResults    []HPINodeResult          `json:"node_results"`
	SharedDxIDs    []string                 `json:"shared_dx_ids,omitempty"`
	ConflictDxIDs  []string                 `json:"conflict_dx_ids,omitempty"`
	EscalationNode string                   `json:"escalation_node,omitempty"`
	Narrative      string                   `json:"narrative"`
	MergedOutput   []MergedDifferential     `json:"merged_output"`
	ArbitratedAt   time.Time               `json:"arbitrated_at"`
}

// MergedDifferential is a single diagnosis entry in the merged multi-node output.
type MergedDifferential struct {
	DxID          string  `json:"dx_id"`
	Label         string  `json:"label"`
	MaxPosterior  float64 `json:"max_posterior"`
	SourceNodes   []string `json:"source_nodes"`
	BoostApplied  bool    `json:"boost_applied"`
	Flagged       bool    `json:"flagged"`
}

// HPIConflictArbiter reconciles HPI outputs from multiple concurrent node sessions.
// It runs as post-processing in KB-19 AFTER all node sessions complete.
//
// Conservative defaults (BAY-7 spec):
//   - REPORT (informational) is the default, not BOOST
//   - BOOST requires 500+ multi-node sessions before enabling (configurable)
//   - RED_FLAG_WINS is unconditional and cannot be overridden
type HPIConflictArbiter struct {
	log              *logrus.Entry
	boostEnabled     bool // disabled by default until 500+ multi-node sessions
	boostMinSessions int  // threshold to enable BOOST (default 500)
	observedSessions int  // track how many multi-node sessions observed
}

// NewHPIConflictArbiter creates an arbiter with conservative defaults.
func NewHPIConflictArbiter(log *logrus.Entry) *HPIConflictArbiter {
	return &HPIConflictArbiter{
		log:              log.WithField("component", "hpi-conflict-arbiter"),
		boostEnabled:     false,
		boostMinSessions: 500,
		observedSessions: 0,
	}
}

// SetBoostEnabled overrides the automatic BOOST gating for testing.
func (a *HPIConflictArbiter) SetBoostEnabled(enabled bool) {
	a.boostEnabled = enabled
}

// Arbitrate reconciles results from two or more completed HPI node sessions.
//
// Decision logic (§11.7 of upgrade plan):
//  1. RED_FLAG_WINS: if ANY node escalated, escalation takes priority unconditionally.
//  2. BOOST: if same dx_id appears in top-3 of multiple nodes with independent evidence.
//  3. FLAG: if Node A supports dx X (top-3) while Node B's posterior for same dx is < 0.05.
//  4. REPORT: independent top diagnoses — most common case for multi-complaint patients.
func (a *HPIConflictArbiter) Arbitrate(results []HPINodeResult) *ArbiterResult {
	if len(results) < 2 {
		a.log.Warn("BAY-7: Arbitrate called with fewer than 2 node results")
		return nil
	}

	a.observedSessions++
	patientID := results[0].PatientID

	out := &ArbiterResult{
		PatientID:    patientID,
		NodeResults:  results,
		ArbitratedAt: time.Now(),
	}

	// Rule 1: RED_FLAG_WINS — unconditional
	for _, r := range results {
		if r.Escalated || len(r.SafetyFlags) > 0 {
			out.Action = ActionRedFlagWins
			out.EscalationNode = r.NodeID
			out.Narrative = fmt.Sprintf(
				"BAY-7 RED_FLAG_WINS: Node %s triggered safety escalation. "+
					"Escalation takes priority regardless of other node outputs. "+
					"Other node results are supplementary.",
				r.NodeID,
			)
			out.MergedOutput = a.mergeWithEscalation(results, r.NodeID)
			a.log.WithFields(logrus.Fields{
				"patient_id":      patientID,
				"escalation_node": r.NodeID,
				"safety_flags":    r.SafetyFlags,
			}).Info("BAY-7: RED_FLAG_WINS — safety escalation from node")
			return out
		}
	}

	// Build top-3 sets per node for overlap detection
	top3ByNode := make(map[string]map[string]float64)
	for _, r := range results {
		dxSet := make(map[string]float64)
		limit := 3
		if len(r.TopDx) < limit {
			limit = len(r.TopDx)
		}
		for i := 0; i < limit; i++ {
			dxSet[r.TopDx[i].DxID] = r.TopDx[i].Posterior
		}
		top3ByNode[r.NodeID] = dxSet
	}

	// Rule 2: Detect shared diagnoses (BOOST candidates)
	sharedDxIDs := a.findSharedDiagnoses(top3ByNode)

	// Rule 3: Detect contradictory evidence (FLAG candidates)
	conflictDxIDs := a.findConflicts(results, top3ByNode)

	// Determine action
	switch {
	case len(conflictDxIDs) > 0:
		out.Action = ActionFlag
		out.ConflictDxIDs = conflictDxIDs
		out.Narrative = fmt.Sprintf(
			"BAY-7 FLAG: Conflicting evidence for %v across nodes. "+
				"Recommend clinical evaluation to resolve. Do NOT auto-resolve — GP's decision.",
			conflictDxIDs,
		)
		a.log.WithFields(logrus.Fields{
			"patient_id":     patientID,
			"conflict_dx_ids": conflictDxIDs,
		}).Info("BAY-7: FLAG — contradictory evidence across nodes")

	case len(sharedDxIDs) > 0 && a.isBoostEnabled():
		out.Action = ActionBoost
		out.SharedDxIDs = sharedDxIDs
		out.Narrative = fmt.Sprintf(
			"BAY-7 BOOST: Diagnosis %v over-determined by independent evidence "+
				"from multiple nodes. Report once with combined evidence. Increased confidence.",
			sharedDxIDs,
		)
		a.log.WithFields(logrus.Fields{
			"patient_id":    patientID,
			"shared_dx_ids": sharedDxIDs,
		}).Info("BAY-7: BOOST — shared diagnosis over-determined")

	case len(sharedDxIDs) > 0 && !a.isBoostEnabled():
		// Shared diagnoses found but BOOST not yet enabled — report as informational
		out.Action = ActionReport
		out.SharedDxIDs = sharedDxIDs
		out.Narrative = fmt.Sprintf(
			"BAY-7 REPORT: Shared diagnosis %v found in multiple nodes. "+
				"BOOST not yet enabled (requires %d multi-node sessions, observed %d). "+
				"Reporting both node outputs independently.",
			sharedDxIDs, a.boostMinSessions, a.observedSessions,
		)
		a.log.WithFields(logrus.Fields{
			"patient_id":    patientID,
			"shared_dx_ids": sharedDxIDs,
			"boost_pending": true,
		}).Info("BAY-7: REPORT — shared diagnosis but BOOST not enabled")

	default:
		out.Action = ActionReport
		out.Narrative = "BAY-7 REPORT: Independent top diagnoses across nodes. " +
			"Normal case for multi-complaint patients. Reporting both independently."
		a.log.WithField("patient_id", patientID).Info("BAY-7: REPORT — independent diagnoses")
	}

	out.MergedOutput = a.mergeResults(results, sharedDxIDs, conflictDxIDs, a.isBoostEnabled())
	return out
}

// isBoostEnabled returns true if BOOST actions are allowed.
func (a *HPIConflictArbiter) isBoostEnabled() bool {
	return a.boostEnabled || a.observedSessions >= a.boostMinSessions
}

// findSharedDiagnoses returns dx_ids that appear in top-3 of 2+ nodes.
func (a *HPIConflictArbiter) findSharedDiagnoses(top3ByNode map[string]map[string]float64) []string {
	dxCount := make(map[string]int)
	for _, dxSet := range top3ByNode {
		for dxID := range dxSet {
			dxCount[dxID]++
		}
	}
	var shared []string
	for dxID, count := range dxCount {
		if count >= 2 {
			shared = append(shared, dxID)
		}
	}
	sort.Strings(shared)
	return shared
}

// findConflicts detects contradictory evidence: Node A has dx in top-3,
// Node B's posterior for same dx is < 0.05.
func (a *HPIConflictArbiter) findConflicts(results []HPINodeResult, top3ByNode map[string]map[string]float64) []string {
	conflictSet := make(map[string]bool)
	for i, rA := range results {
		for j, rB := range results {
			if i == j {
				continue
			}
			for dxID := range top3ByNode[rA.NodeID] {
				posteriorInB, exists := rB.Posterior[dxID]
				if exists && posteriorInB < 0.05 {
					conflictSet[dxID] = true
					a.log.WithFields(logrus.Fields{
						"dx_id":        dxID,
						"node_supports": rA.NodeID,
						"node_opposes":  rB.NodeID,
						"posterior_in_b": posteriorInB,
					}).Debug("BAY-7: contradiction detected")
				}
			}
		}
	}
	var conflicts []string
	for dxID := range conflictSet {
		conflicts = append(conflicts, dxID)
	}
	sort.Strings(conflicts)
	return conflicts
}

// mergeResults produces the unified differential list from all nodes.
func (a *HPIConflictArbiter) mergeResults(
	results []HPINodeResult,
	sharedDxIDs, conflictDxIDs []string,
	boostEnabled bool,
) []MergedDifferential {
	sharedSet := toSet(sharedDxIDs)
	conflictSet := toSet(conflictDxIDs)

	dxMap := make(map[string]*MergedDifferential)
	for _, r := range results {
		for _, dx := range r.TopDx {
			existing, ok := dxMap[dx.DxID]
			if !ok {
				dxMap[dx.DxID] = &MergedDifferential{
					DxID:         dx.DxID,
					Label:        dx.Label,
					MaxPosterior: dx.Posterior,
					SourceNodes:  []string{r.NodeID},
					BoostApplied: sharedSet[dx.DxID] && boostEnabled,
					Flagged:      conflictSet[dx.DxID],
				}
			} else {
				if dx.Posterior > existing.MaxPosterior {
					existing.MaxPosterior = dx.Posterior
				}
				existing.SourceNodes = append(existing.SourceNodes, r.NodeID)
				if sharedSet[dx.DxID] && boostEnabled {
					existing.BoostApplied = true
				}
				if conflictSet[dx.DxID] {
					existing.Flagged = true
				}
			}
		}
	}

	merged := make([]MergedDifferential, 0, len(dxMap))
	for _, md := range dxMap {
		merged = append(merged, *md)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].MaxPosterior > merged[j].MaxPosterior
	})
	return merged
}

// mergeWithEscalation merges results prioritising the escalated node.
func (a *HPIConflictArbiter) mergeWithEscalation(results []HPINodeResult, escalationNode string) []MergedDifferential {
	var merged []MergedDifferential
	// Escalation node's results come first
	for _, r := range results {
		if r.NodeID == escalationNode {
			for _, dx := range r.TopDx {
				merged = append(merged, MergedDifferential{
					DxID:         dx.DxID,
					Label:        dx.Label,
					MaxPosterior: dx.Posterior,
					SourceNodes:  []string{r.NodeID},
				})
			}
		}
	}
	// Supplementary nodes follow
	for _, r := range results {
		if r.NodeID != escalationNode {
			for _, dx := range r.TopDx {
				merged = append(merged, MergedDifferential{
					DxID:         dx.DxID,
					Label:        dx.Label,
					MaxPosterior: dx.Posterior,
					SourceNodes:  []string{r.NodeID},
				})
			}
		}
	}
	return merged
}

func toSet(ids []string) map[string]bool {
	s := make(map[string]bool, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}
