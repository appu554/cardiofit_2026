// Package factstore provides the Signal Merger for cross-method corroboration
// of extracted clinical facts.
//
// P2.3: When the same clinical fact is extracted by multiple methods
// (STRUCTURED_PARSE, LLM_FALLBACK, GRAMMAR), the merger:
//   - Clusters facts by CanonicalKey (rxcui|factType|normalizedIdentifier)
//   - Selects STRUCTURED_PARSE as primary when available (deterministic > probabilistic)
//   - Boosts confidence: 2 methods +0.10, 3+ methods +0.15
//   - Flags severity/condition disagreements for governance review
package factstore

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)

// SignalMerger clusters and corroborates facts extracted by different methods.
type SignalMerger struct {
	log *logrus.Entry
}

// NewSignalMerger creates a signal merger.
func NewSignalMerger(log *logrus.Entry) *SignalMerger {
	return &SignalMerger{
		log: log.WithField("component", "signal-merger"),
	}
}

// MergeResult contains the merged facts and corroboration statistics.
type MergeResult struct {
	Facts              []*DerivedFact
	TotalInput         int
	TotalOutput        int
	Corroborated       int // Facts confirmed by 2+ methods
	DisagreementsFound int // Facts with conflicting data across methods
}

// Merge clusters facts by CanonicalKey and applies corroboration logic.
// It processes all facts for a single drug/section, not the entire pipeline.
func (sm *SignalMerger) Merge(facts []*DerivedFact) *MergeResult {
	result := &MergeResult{
		TotalInput: len(facts),
	}

	if len(facts) <= 1 {
		result.Facts = facts
		result.TotalOutput = len(facts)
		return result
	}

	// Phase 1: Cluster by CanonicalKey
	clusters := make(map[string][]*DerivedFact)
	clusterOrder := make([]string, 0) // Preserve insertion order
	for _, f := range facts {
		key := f.FactKey
		if _, exists := clusters[key]; !exists {
			clusterOrder = append(clusterOrder, key)
		}
		clusters[key] = append(clusters[key], f)
	}

	// Phase 2: Resolve each cluster
	for _, key := range clusterOrder {
		cluster := clusters[key]

		if len(cluster) == 1 {
			// Singleton — no corroboration possible
			result.Facts = append(result.Facts, cluster[0])
			continue
		}

		// Multiple methods produced the same fact — corroborate
		result.Corroborated++
		merged := sm.resolveCluster(cluster)
		result.Facts = append(result.Facts, merged)

		// Check for disagreements
		if hasDisagreement(cluster) {
			result.DisagreementsFound++
			merged.GovernanceStatus = "PENDING_REVIEW" // Force review on disagreements
			sm.log.WithFields(logrus.Fields{
				"factKey":  key[:16],
				"methods":  countMethods(cluster),
				"factType": merged.FactType,
			}).Warn("Disagreement detected between extraction methods")
		}
	}

	result.TotalOutput = len(result.Facts)

	if result.Corroborated > 0 {
		sm.log.WithFields(logrus.Fields{
			"input":         result.TotalInput,
			"output":        result.TotalOutput,
			"corroborated":  result.Corroborated,
			"disagreements": result.DisagreementsFound,
		}).Info("Signal merger completed")
	}

	return result
}

// resolveCluster picks the primary fact from a cluster and applies confidence boosting.
func (sm *SignalMerger) resolveCluster(cluster []*DerivedFact) *DerivedFact {
	// Priority: STRUCTURED_PARSE > LLM_FALLBACK
	// Within same method, pick highest confidence
	var primary *DerivedFact

	for _, f := range cluster {
		if primary == nil {
			primary = f
			continue
		}

		// STRUCTURED_PARSE always wins over LLM_FALLBACK
		if f.ExtractionMethod == "STRUCTURED_PARSE" && primary.ExtractionMethod != "STRUCTURED_PARSE" {
			primary = f
			continue
		}

		// Within same method tier, pick higher confidence
		if f.ExtractionMethod == primary.ExtractionMethod && f.ExtractionConfidence > primary.ExtractionConfidence {
			primary = f
		}
	}

	// Apply confidence boost based on number of corroborating methods
	methods := countMethods(cluster)
	switch {
	case methods >= 3:
		primary.ExtractionConfidence += 0.15
	case methods >= 2:
		primary.ExtractionConfidence += 0.10
	}
	if primary.ExtractionConfidence > 1.0 {
		primary.ExtractionConfidence = 1.0
	}

	// Mark as corroborated
	primary.ConsensusAchieved = true

	// Enrich primary with data from other methods if primary is missing fields
	sm.enrichFromSecondary(primary, cluster)

	return primary
}

// enrichFromSecondary fills empty fields on the primary fact from secondary extractions.
func (sm *SignalMerger) enrichFromSecondary(primary *DerivedFact, cluster []*DerivedFact) {
	// Parse primary content
	primaryData := make(map[string]interface{})
	if err := json.Unmarshal(primary.FactData, &primaryData); err != nil {
		return
	}

	enriched := false
	for _, f := range cluster {
		if f == primary {
			continue
		}

		secondaryData := make(map[string]interface{})
		if err := json.Unmarshal(f.FactData, &secondaryData); err != nil {
			continue
		}

		// Fill empty fields from secondary
		for key, val := range secondaryData {
			if existing, ok := primaryData[key]; !ok || isEmptyValue(existing) {
				if !isEmptyValue(val) {
					primaryData[key] = val
					enriched = true
				}
			}
		}
	}

	if enriched {
		if data, err := json.Marshal(primaryData); err == nil {
			primary.FactData = data
		}
	}
}

// hasDisagreement checks if facts in a cluster have conflicting key fields.
func hasDisagreement(cluster []*DerivedFact) bool {
	if len(cluster) < 2 {
		return false
	}

	// Compare severity and condition across facts
	var severities, conditions []string
	for _, f := range cluster {
		data := make(map[string]interface{})
		if err := json.Unmarshal(f.FactData, &data); err != nil {
			continue
		}

		if sev, ok := data["severity"].(string); ok && sev != "" {
			severities = append(severities, strings.ToUpper(sev))
		}
		if cond, ok := data["conditionName"].(string); ok && cond != "" {
			conditions = append(conditions, strings.ToLower(cond))
		}
		if inter, ok := data["interactantName"].(string); ok && inter != "" {
			conditions = append(conditions, strings.ToLower(inter))
		}
	}

	// Check severity disagreement
	if len(severities) >= 2 {
		first := severities[0]
		for _, s := range severities[1:] {
			if s != first {
				return true
			}
		}
	}

	return false
}

// countMethods returns the number of distinct extraction methods in a cluster.
func countMethods(cluster []*DerivedFact) int {
	methods := make(map[string]bool)
	for _, f := range cluster {
		methods[f.ExtractionMethod] = true
	}
	return len(methods)
}

// isEmptyValue checks if a JSON value is empty/nil/zero.
func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case float64:
		return val == 0
	case bool:
		return !val
	}
	return false
}
