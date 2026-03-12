package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-22-hpi-engine/internal/models"
)

// NodeLoader parses and validates P1-P26 YAML node definitions at startup.
// Implements hot-reload without restart via Reload().
type NodeLoader struct {
	nodesDir string
	log      *zap.Logger

	mu    sync.RWMutex
	nodes map[string]*models.NodeDefinition // node_id -> definition
}

func NewNodeLoader(nodesDir string, log *zap.Logger) *NodeLoader {
	return &NodeLoader{
		nodesDir: nodesDir,
		log:      log,
		nodes:    make(map[string]*models.NodeDefinition),
	}
}

// NewNodeLoaderFromMap creates a NodeLoader pre-populated with the given nodes.
// Intended for unit tests that need a NodeLoader without reading YAML from disk.
func NewNodeLoaderFromMap(nodes map[string]*models.NodeDefinition) *NodeLoader {
	return &NodeLoader{
		log:   zap.NewNop(),
		nodes: nodes,
	}
}

// Load parses all YAML files in nodesDir and validates them.
// Service exits if any node fails validation (spec Section 6.2).
func (nl *NodeLoader) Load() error {
	entries, err := os.ReadDir(nl.nodesDir)
	if err != nil {
		if os.IsNotExist(err) {
			nl.log.Warn("nodes directory does not exist, starting with empty node set",
				zap.String("dir", nl.nodesDir))
			return nil
		}
		return fmt.Errorf("read nodes dir: %w", err)
	}

	loaded := make(map[string]*models.NodeDefinition)
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}
		if entry.Name() == "cross_node_triggers.yaml" {
			continue // Handled separately by CrossNodeSafety
		}

		path := filepath.Join(nl.nodesDir, entry.Name())
		node, err := nl.parseNode(path)
		if err != nil {
			return fmt.Errorf("node %s: %w", entry.Name(), err)
		}

		if err := nl.validate(node, entry.Name()); err != nil {
			return fmt.Errorf("node %s validation: %w", entry.Name(), err)
		}

		// R-05: auto-inject minimum_inclusion_guard on safety trigger component questions
		nl.injectSafetyGuards(node)

		loaded[node.NodeID] = node
		nl.log.Info("loaded node",
			zap.String("node_id", node.NodeID),
			zap.String("version", node.Version),
			zap.Int("questions", len(node.Questions)),
			zap.Int("differentials", len(node.Differentials)),
			zap.Int("safety_triggers", len(node.SafetyTriggers)),
			zap.Int("context_modifiers", len(node.ContextModifiers)),
		)
	}

	nl.mu.Lock()
	nl.nodes = loaded
	nl.mu.Unlock()

	nl.log.Info("node loading complete", zap.Int("total_nodes", len(loaded)))
	return nil
}

// Reload re-reads all nodes from disk. Used by POST /internal/nodes/reload.
func (nl *NodeLoader) Reload() error {
	return nl.Load()
}

// Get returns a node by ID. Returns nil if not found.
func (nl *NodeLoader) Get(nodeID string) *models.NodeDefinition {
	nl.mu.RLock()
	defer nl.mu.RUnlock()
	return nl.nodes[nodeID]
}

// List returns all loaded node IDs.
func (nl *NodeLoader) List() []string {
	nl.mu.RLock()
	defer nl.mu.RUnlock()
	ids := make([]string, 0, len(nl.nodes))
	for id := range nl.nodes {
		ids = append(ids, id)
	}
	return ids
}

// All returns all loaded nodes.
func (nl *NodeLoader) All() map[string]*models.NodeDefinition {
	nl.mu.RLock()
	defer nl.mu.RUnlock()
	result := make(map[string]*models.NodeDefinition, len(nl.nodes))
	for k, v := range nl.nodes {
		result[k] = v
	}
	return result
}

func (nl *NodeLoader) parseNode(path string) (*models.NodeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var node models.NodeDefinition
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	// Default convergence_logic if not specified
	if node.ConvergenceLogic == "" {
		node.ConvergenceLogic = "BOTH"
	}

	return &node, nil
}

// validate enforces all Section 6.2 rules at startup.
func (nl *NodeLoader) validate(node *models.NodeDefinition, filename string) error {
	// node_id must match filename stem
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	if !strings.EqualFold(node.NodeID, strings.ToUpper(stem)) &&
		!strings.EqualFold(strings.ToLower(node.NodeID), strings.ReplaceAll(stem, "_", "")) {
		// Flexible matching: p01_chest_pain -> P01_CHEST_PAIN
		nl.log.Warn("node_id does not exactly match filename stem",
			zap.String("node_id", node.NodeID), zap.String("filename", filename))
	}

	// convergence_threshold must be in (0.50, 1.00) exclusive
	if node.ConvergenceThreshold <= 0.50 || node.ConvergenceThreshold >= 1.00 {
		return fmt.Errorf("convergence_threshold %.2f must be in (0.50, 1.00)", node.ConvergenceThreshold)
	}

	// convergence_logic must be valid
	switch node.ConvergenceLogic {
	case "BOTH", "EITHER", "POSTERIOR_ONLY":
		// valid
	default:
		return fmt.Errorf("invalid convergence_logic: %s (must be BOTH|EITHER|POSTERIOR_ONLY)", node.ConvergenceLogic)
	}

	// Build differential ID set
	diffIDs := make(map[string]bool, len(node.Differentials))
	for _, d := range node.Differentials {
		diffIDs[d.ID] = true
		// R-07: warn if population_reference is missing
		if d.PopulationReference == "" {
			nl.log.Warn("differential missing population_reference",
				zap.String("node_id", node.NodeID), zap.String("differential", d.ID))
		}
	}

	// Build question ID set
	questionIDs := make(map[string]bool, len(node.Questions))
	mandatoryCount := 0
	for _, q := range node.Questions {
		questionIDs[q.ID] = true
		if q.Mandatory {
			mandatoryCount++
		}
	}

	// max_questions > count of mandatory questions
	if node.MaxQuestions <= mandatoryCount {
		return fmt.Errorf("max_questions (%d) must be > mandatory question count (%d)",
			node.MaxQuestions, mandatoryCount)
	}

	// Validate question LR references
	for _, q := range node.Questions {
		// G10: validate CATEGORICAL questions
		if q.AnswerType == models.AnswerTypeCategorical {
			// Legacy CATEGORICAL nodes may use the options: schema (not yet migrated
			// to answer_options + lr_categorical). Skip strict validation for those.
			if len(q.AnswerOptions) == 0 && len(q.LRCategorical) == 0 {
				nl.log.Debug("G10: categorical question uses legacy options schema, skipping strict validation",
					zap.String("question_id", q.ID))
			} else if len(q.AnswerOptions) < 2 {
				return fmt.Errorf("categorical question %s must have >= 2 answer_options", q.ID)
			} else if len(q.LRCategorical) == 0 {
				return fmt.Errorf("categorical question %s must have lr_categorical defined", q.ID)
			} else {
				// Every answer option must have an lr_categorical entry
				for _, opt := range q.AnswerOptions {
					lrMap, ok := q.LRCategorical[opt]
					if !ok {
						return fmt.Errorf("categorical question %s lr_categorical missing entry for answer option %s", q.ID, opt)
					}
					// Every LR entry must reference declared differentials
					for diffID := range lrMap {
						if !diffIDs[diffID] {
							return fmt.Errorf("categorical question %s lr_categorical[%s] references undeclared differential %s", q.ID, opt, diffID)
						}
					}
				}
				// No extra keys in lr_categorical beyond declared options
				for key := range q.LRCategorical {
					found := false
					for _, opt := range q.AnswerOptions {
						if key == opt {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("categorical question %s lr_categorical has undeclared answer option %s", q.ID, key)
					}
				}
				// LRPositive/LRNegative should not be set on categorical questions
				if len(q.LRPositive) > 0 || len(q.LRNegative) > 0 {
					nl.log.Warn("G10: categorical question has lr_positive/lr_negative (ignored)",
						zap.String("question_id", q.ID))
				}
			}
		} else {
			// BINARY (default): validate lr_positive/lr_negative as before
			// All differential IDs in lr_positive must be declared
			for diffID := range q.LRPositive {
				if !diffIDs[diffID] {
					return fmt.Errorf("question %s lr_positive references undeclared differential %s", q.ID, diffID)
				}
			}
			for diffID := range q.LRNegative {
				if !diffIDs[diffID] {
					return fmt.Errorf("question %s lr_negative references undeclared differential %s", q.ID, diffID)
				}
			}
		}

		// G6: validate stratum-conditional LR overrides
		if len(q.LRPositiveByStratum) > 0 || len(q.LRNegativeByStratum) > 0 {
			if q.AnswerType == models.AnswerTypeCategorical {
				return fmt.Errorf("question %s: lr_positive_by_stratum/lr_negative_by_stratum not supported for CATEGORICAL questions", q.ID)
			}
			for stratum, sMap := range q.LRPositiveByStratum {
				if len(node.StrataSupported) > 0 {
					found := false
					for _, s := range node.StrataSupported {
						if s == stratum {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("question %s lr_positive_by_stratum references undeclared stratum %s", q.ID, stratum)
					}
				}
				for diffID := range sMap {
					if !diffIDs[diffID] {
						return fmt.Errorf("question %s lr_positive_by_stratum[%s] references undeclared differential %s", q.ID, stratum, diffID)
					}
				}
			}
			for stratum, sMap := range q.LRNegativeByStratum {
				if len(node.StrataSupported) > 0 {
					found := false
					for _, s := range node.StrataSupported {
						if s == stratum {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("question %s lr_negative_by_stratum references undeclared stratum %s", q.ID, stratum)
					}
				}
				for diffID := range sMap {
					if !diffIDs[diffID] {
						return fmt.Errorf("question %s lr_negative_by_stratum[%s] references undeclared differential %s", q.ID, stratum, diffID)
					}
				}
			}
		}

		// Mandatory questions must have LR data (binary: lr_positive+lr_negative, categorical: lr_categorical)
		if q.Mandatory {
			if q.AnswerType == models.AnswerTypeCategorical {
				if len(q.LRCategorical) == 0 {
					return fmt.Errorf("mandatory categorical question %s must have lr_categorical defined", q.ID)
				}
			} else {
				if len(q.LRPositive) == 0 {
					return fmt.Errorf("mandatory question %s must have lr_positive defined", q.ID)
				}
				if len(q.LRNegative) == 0 {
					return fmt.Errorf("mandatory question %s must have lr_negative defined", q.ID)
				}
			}
		}

		// R-07: warn if lr_source is missing
		if q.LRSource == "" {
			nl.log.Warn("question missing lr_source",
				zap.String("node_id", node.NodeID), zap.String("question", q.ID))
		}
	}

	// Validate context modifiers (G14 YAML CMs, G5 effect types)
	cmIDs := make(map[string]bool, len(node.ContextModifiers))
	for _, cm := range node.ContextModifiers {
		if cm.ID == "" {
			return fmt.Errorf("context modifier missing id")
		}
		if cmIDs[cm.ID] {
			return fmt.Errorf("duplicate context modifier id: %s", cm.ID)
		}
		cmIDs[cm.ID] = true

		// G5: Validation varies by effect type
		switch cm.EffectType {
		case models.CMEffectHardBlock:
			// HARD_BLOCK requires blocked_treatment; adjustments optional (diagnostic shift secondary)
			if cm.BlockedTreatment == "" {
				return fmt.Errorf("context modifier %s (HARD_BLOCK) missing blocked_treatment", cm.ID)
			}
			// Adjustments are optional for HARD_BLOCK (treatment blocking is primary purpose)
			for adjDiffID := range cm.Adjustments {
				if !diffIDs[adjDiffID] {
					return fmt.Errorf("context modifier %s references undeclared differential %s", cm.ID, adjDiffID)
				}
			}

		case models.CMEffectOverride:
			// OVERRIDE requires override_targets with valid posterior values
			if len(cm.OverrideTargets) == 0 {
				return fmt.Errorf("context modifier %s (OVERRIDE) has empty override_targets", cm.ID)
			}
			for diffID, minPost := range cm.OverrideTargets {
				if !diffIDs[diffID] {
					return fmt.Errorf("context modifier %s override_targets references undeclared differential %s", cm.ID, diffID)
				}
				if minPost <= 0 || minPost >= 1.0 {
					return fmt.Errorf("context modifier %s override_targets for %s has min_posterior %.3f outside (0, 1.0)", cm.ID, diffID, minPost)
				}
			}

		default:
			// Standard INCREASE_PRIOR / DECREASE_PRIOR: adjustments required
			if len(cm.Adjustments) == 0 {
				return fmt.Errorf("context modifier %s has empty adjustments", cm.ID)
			}
			for adjDiffID, mag := range cm.Adjustments {
				if !diffIDs[adjDiffID] {
					return fmt.Errorf("context modifier %s references undeclared differential %s", cm.ID, adjDiffID)
				}
				if mag <= 0 || mag >= 0.50 {
					return fmt.Errorf("context modifier %s adjustment for %s has magnitude %.3f outside (0, 0.50)", cm.ID, adjDiffID, mag)
				}
			}
		}
	}

	// Validate safety floors (G1)
	for diffID, floor := range node.SafetyFloors {
		if !diffIDs[diffID] {
			return fmt.Errorf("safety_floors references undeclared differential %s", diffID)
		}
		if floor <= 0 || floor >= 1.0 {
			return fmt.Errorf("safety_floors[%s] = %.4f must be in (0, 1.0)", diffID, floor)
		}
	}
	for stratum, floors := range node.SafetyFloorsByStratum {
		for diffID, floor := range floors {
			if !diffIDs[diffID] {
				return fmt.Errorf("safety_floors_by_stratum[%s] references undeclared differential %s", stratum, diffID)
			}
			if floor <= 0 || floor >= 1.0 {
				return fmt.Errorf("safety_floors_by_stratum[%s][%s] = %.4f must be in (0, 1.0)", stratum, diffID, floor)
			}
		}
	}
	// A03: warn if node uses strata but only has simple safety_floors (no per-stratum)
	if len(node.StrataSupported) > 0 && len(node.SafetyFloors) > 0 && len(node.SafetyFloorsByStratum) == 0 {
		nl.log.Warn("A03: node uses strata but safety_floors are not stratum-specific",
			zap.String("node_id", node.NodeID),
			zap.Strings("strata", node.StrataSupported),
		)
	}

	// Validate sex modifiers (G2)
	smIDs := make(map[string]bool, len(node.SexModifiers))
	for _, sm := range node.SexModifiers {
		if sm.ID == "" {
			return fmt.Errorf("sex modifier missing id")
		}
		if smIDs[sm.ID] {
			return fmt.Errorf("duplicate sex modifier id: %s", sm.ID)
		}
		smIDs[sm.ID] = true
		if sm.Condition == "" {
			return fmt.Errorf("sex modifier %s missing condition", sm.ID)
		}
		if len(sm.Adjustments) == 0 {
			return fmt.Errorf("sex modifier %s has empty adjustments", sm.ID)
		}
		for adjDiffID := range sm.Adjustments {
			if !diffIDs[adjDiffID] {
				return fmt.Errorf("sex modifier %s references undeclared differential %s", sm.ID, adjDiffID)
			}
		}
	}

	// ═══════════════════════════════════════════════════════════════
	// A01: Stratum-vs-Modifier compliance checks
	// Enforces the 4-question decision framework at load time.
	// All A01 checks are warnings (non-blocking) — they flag potential
	// authoring issues for Canon Framework review.
	// ═══════════════════════════════════════════════════════════════
	nl.validateA01Compliance(node, diffIDs)

	// Validate safety trigger question references
	for _, t := range node.SafetyTriggers {
		if t.Type == "COMPOSITE_SCORE" {
			// G12/R-06: validate COMPOSITE_SCORE trigger
			if len(t.Weights) == 0 {
				return fmt.Errorf("COMPOSITE_SCORE trigger %s must have weights defined", t.ID)
			}
			if t.Threshold <= 0 {
				return fmt.Errorf("COMPOSITE_SCORE trigger %s must have positive threshold, got %.4f", t.ID, t.Threshold)
			}
			// Weights keys are "QUESTION_ID=ANSWER_VALUE" pairs
			for key := range t.Weights {
				parts := strings.SplitN(key, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("COMPOSITE_SCORE trigger %s weight key %q must be QUESTION_ID=ANSWER format", t.ID, key)
				}
				qid := strings.TrimSpace(parts[0])
				if !questionIDs[qid] {
					return fmt.Errorf("COMPOSITE_SCORE trigger %s weight references undeclared question %s", t.ID, qid)
				}
			}
		} else {
			// BOOLEAN (default): extract question IDs from condition expression
			referencedQIDs := extractQuestionIDsFromCondition(t.Condition)
			for _, qid := range referencedQIDs {
				if !questionIDs[qid] {
					return fmt.Errorf("safety trigger %s references undeclared question %s", t.ID, qid)
				}
			}
		}
	}

	// G13: validate node transitions
	transitionIDs := make(map[string]bool, len(node.Transitions))
	for _, t := range node.Transitions {
		if t.ID == "" {
			return fmt.Errorf("node transition missing id")
		}
		if transitionIDs[t.ID] {
			return fmt.Errorf("duplicate node transition id: %s", t.ID)
		}
		transitionIDs[t.ID] = true

		if t.TargetNode == "" {
			return fmt.Errorf("node transition %s missing target_node", t.ID)
		}
		if t.TriggerCondition == "" {
			return fmt.Errorf("node transition %s missing trigger_condition", t.ID)
		}

		switch t.Mode {
		case models.TransitionConcurrent, models.TransitionHandoff, models.TransitionFlag:
			// valid
		default:
			return fmt.Errorf("node transition %s has invalid mode %q (must be CONCURRENT, HANDOFF, or FLAG)", t.ID, t.Mode)
		}
	}

	// G17: validate contradiction pairs
	contradictionIDs := make(map[string]bool, len(node.ContradictionPairs))
	for _, cp := range node.ContradictionPairs {
		if cp.ID == "" {
			return fmt.Errorf("contradiction pair missing id")
		}
		if contradictionIDs[cp.ID] {
			return fmt.Errorf("duplicate contradiction pair id: %s", cp.ID)
		}
		contradictionIDs[cp.ID] = true

		if cp.QuestionA == "" || cp.QuestionB == "" {
			return fmt.Errorf("contradiction pair %s must have both question_a and question_b", cp.ID)
		}
		if !questionIDs[cp.QuestionA] {
			return fmt.Errorf("contradiction pair %s references undeclared question_a %s", cp.ID, cp.QuestionA)
		}
		if !questionIDs[cp.QuestionB] {
			return fmt.Errorf("contradiction pair %s references undeclared question_b %s", cp.ID, cp.QuestionB)
		}
		if cp.QuestionA == cp.QuestionB {
			return fmt.Errorf("contradiction pair %s question_a and question_b must be different", cp.ID)
		}
	}

	return nil
}

// injectSafetyGuards auto-sets minimum_inclusion_guard on questions that are
// components of safety triggers (R-05).
func (nl *NodeLoader) injectSafetyGuards(node *models.NodeDefinition) {
	safetyQuestionIDs := make(map[string]bool)
	for _, t := range node.SafetyTriggers {
		for _, qid := range extractQuestionIDsFromCondition(t.Condition) {
			safetyQuestionIDs[qid] = true
		}
	}

	for i := range node.Questions {
		if safetyQuestionIDs[node.Questions[i].ID] {
			node.Questions[i].MinimumInclusionGuard = true
		}
	}
}

// validateA01Compliance enforces the A01 Stratum-vs-Modifier Decision Framework.
// All checks are warnings — they flag potential authoring issues but do not block loading.
//
// Checks:
//
//	A01-Q1: CM targeting >= 3 differentials may be better as a stratum
//	A01-Q4: Multi-stratum node has differentials without population_reference (evidence gap)
//	A01-PRIOR: When other_bucket_enabled, priors should sum to ~(1.0 - other_bucket_prior)
//	A01-STRATA-CM: CM adjustments target a conditional differential that may be excluded
//	A01-COVERAGE: Strata declared in strata_supported but no differential provides priors for it
func (nl *NodeLoader) validateA01Compliance(node *models.NodeDefinition, diffIDs map[string]bool) {
	// A01-Q1: Warn if any CM targets >= 3 differentials (may warrant stratum instead)
	for _, cm := range node.ContextModifiers {
		if len(cm.Adjustments) >= 3 {
			nl.log.Warn("A01-Q1: context modifier targets >= 3 differentials; consider if this should be a stratum instead",
				zap.String("node_id", node.NodeID),
				zap.String("cm_id", cm.ID),
				zap.Int("target_count", len(cm.Adjustments)),
			)
		}
	}

	// A01-Q4: If node has multiple strata, all differentials SHOULD have population_reference
	if len(node.StrataSupported) > 1 {
		for _, d := range node.Differentials {
			if d.PopulationReference == "" {
				nl.log.Warn("A01-Q4: multi-stratum node has differential without population_reference (evidence gap)",
					zap.String("node_id", node.NodeID),
					zap.String("differential", d.ID),
					zap.Strings("strata", node.StrataSupported),
				)
			}
		}
	}

	// A01-PRIOR: When other_bucket_enabled, verify priors sum to approximately
	// (1.0 - other_bucket_prior) for each declared stratum.
	if node.OtherBucketEnabled && len(node.StrataSupported) > 0 {
		expectedSum := 1.0 - node.OtherBucketPrior
		if node.OtherBucketPrior <= 0 {
			expectedSum = 0.85 // default
		}
		for _, stratum := range node.StrataSupported {
			sum := 0.0
			count := 0
			for _, d := range node.Differentials {
				if p, ok := d.Priors[stratum]; ok {
					sum += p
					count++
				}
			}
			if count > 0 {
				delta := sum - expectedSum
				if delta < 0 {
					delta = -delta
				}
				if delta > 0.02 {
					nl.log.Warn("A01-PRIOR: priors do not sum to expected value for stratum (Other bucket mismatch?)",
						zap.String("node_id", node.NodeID),
						zap.String("stratum", stratum),
						zap.Float64("prior_sum", sum),
						zap.Float64("expected", expectedSum),
						zap.Float64("other_bucket_prior", node.OtherBucketPrior),
					)
				}
			}
		}
	}

	// A01-STRATA-CM: Warn if a CM targets a medication-conditional differential
	// (which may be excluded at runtime, making the CM adjustment a no-op).
	conditionalDiffs := make(map[string]bool)
	for _, d := range node.Differentials {
		if d.ActivationCondition != "" {
			conditionalDiffs[d.ID] = true
		}
	}
	if len(conditionalDiffs) > 0 {
		for _, cm := range node.ContextModifiers {
			for adjDiffID := range cm.Adjustments {
				if conditionalDiffs[adjDiffID] {
					nl.log.Warn("A01-STRATA-CM: CM targets a medication-conditional differential (may be excluded at runtime)",
						zap.String("node_id", node.NodeID),
						zap.String("cm_id", cm.ID),
						zap.String("conditional_diff", adjDiffID),
					)
				}
			}
		}
	}

	// A01-COVERAGE: Warn if strata_supported declares a stratum that no differential provides priors for.
	for _, stratum := range node.StrataSupported {
		found := false
		for _, d := range node.Differentials {
			if _, ok := d.Priors[stratum]; ok {
				found = true
				break
			}
		}
		if !found {
			nl.log.Warn("A01-COVERAGE: declared stratum has no differential priors",
				zap.String("node_id", node.NodeID),
				zap.String("stratum", stratum),
			)
		}
	}
}

// extractQuestionIDsFromCondition parses question IDs from boolean expressions
// like 'Q001=YES AND Q003=YES'.
func extractQuestionIDsFromCondition(condition string) []string {
	var ids []string
	parts := strings.Fields(condition)
	for _, part := range parts {
		// Look for patterns like Q001=YES, Q003=NO
		if idx := strings.IndexByte(part, '='); idx > 0 {
			qid := part[:idx]
			if len(qid) > 0 && (qid[0] == 'Q' || qid[0] == 'q') {
				ids = append(ids, qid)
			}
		}
	}
	return ids
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}
