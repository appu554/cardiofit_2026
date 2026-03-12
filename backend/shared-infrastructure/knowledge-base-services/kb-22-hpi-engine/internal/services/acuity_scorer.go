package services

import (
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// AcuityScorer implements G7: a parallel scoring track that classifies the
// temporal acuity of the presenting complaint as ACUTE, SUBACUTE, or CHRONIC.
//
// This classification runs alongside the Bayesian differential engine and
// modifies the effective LRs for questions that have acuity-dependent
// discriminatory power. For example, a question about "sudden onset" has
// high LR+ for ACS in ACUTE presentations but not in CHRONIC ones.
//
// Architecture:
//   - Questions tagged with acuity_tag (ONSET, DURATION, PROGRESSION, PATTERN)
//     contribute to the acuity score.
//   - After 2+ acuity-tagged questions are answered, the scorer classifies
//     the presentation acuity.
//   - The classification is used by BayesianEngine to apply acuity-dependent
//     LR scaling (deferred to Phase 2 — for now, the classification is
//     recorded in the session for KB-23 Decision Card rendering).
//
// Scoring rules (clinician-authored, not ML):
//   - ONSET=YES (sudden onset): +2 acute points
//   - ONSET=NO: +1 chronic point
//   - DURATION answer: mapped via onset-to-category (< 24h=acute, 24h-14d=subacute, >14d=chronic)
//   - PROGRESSION=YES (getting worse): +1 acute point
//   - PATTERN=YES (intermittent/recurring): +1 chronic point
type AcuityScorer struct {
	log *zap.Logger
}

// AcuityState holds the running acuity scoring state during a session.
type AcuityState struct {
	AcutePoints    int                    `json:"acute_points"`
	SubacutePoints int                    `json:"subacute_points"`
	ChronicPoints  int                    `json:"chronic_points"`
	TagsAnswered   int                    `json:"tags_answered"`
	Category       models.AcuityCategory  `json:"category"`
	Confident      bool                   `json:"confident"` // true after 2+ acuity-tagged answers
}

// NewAcuityScorer creates a new AcuityScorer.
func NewAcuityScorer(log *zap.Logger) *AcuityScorer {
	return &AcuityScorer{log: log}
}

// NewAcuityState creates an initial (unclassified) acuity state.
func NewAcuityState() *AcuityState {
	return &AcuityState{
		Category: models.AcuityUnknown,
	}
}

// Update processes an answer to an acuity-tagged question and updates the
// running acuity score. Returns true if the classification changed.
//
// Tag semantics:
//   ONSET:       YES → acute (+2), NO → chronic (+1)
//   DURATION:    YES → acute (+1), NO → chronic (+1) [simplified; real mapping deferred]
//   PROGRESSION: YES → acute (+1), NO → subacute (+1)
//   PATTERN:     YES → chronic (+1), NO → acute (+1)
func (s *AcuityScorer) Update(state *AcuityState, acuityTag string, answer string) bool {
	if acuityTag == "" {
		return false
	}

	oldCategory := state.Category
	state.TagsAnswered++

	switch acuityTag {
	case "ONSET":
		if answer == string(models.AnswerYes) {
			// Sudden onset → strong acute signal
			state.AcutePoints += 2
		} else if answer == string(models.AnswerNo) {
			state.ChronicPoints++
		}
		// PATA_NAHI: no contribution

	case "DURATION":
		if answer == string(models.AnswerYes) {
			// "Has this been going on for less than 24 hours?"
			state.AcutePoints++
		} else if answer == string(models.AnswerNo) {
			state.ChronicPoints++
		}

	case "PROGRESSION":
		if answer == string(models.AnswerYes) {
			// Getting worse → acute direction
			state.AcutePoints++
		} else if answer == string(models.AnswerNo) {
			state.SubacutePoints++
		}

	case "PATTERN":
		if answer == string(models.AnswerYes) {
			// Intermittent/recurring → chronic direction
			state.ChronicPoints++
		} else if answer == string(models.AnswerNo) {
			state.AcutePoints++
		}

	default:
		s.log.Warn("G7: unknown acuity tag",
			zap.String("acuity_tag", acuityTag),
		)
		return false
	}

	// Classify after 2+ tags answered
	if state.TagsAnswered >= 2 {
		state.Confident = true
		state.Category = s.classify(state)
	}

	changed := state.Category != oldCategory
	if changed {
		s.log.Info("G7: acuity classification updated",
			zap.String("category", string(state.Category)),
			zap.Int("acute", state.AcutePoints),
			zap.Int("subacute", state.SubacutePoints),
			zap.Int("chronic", state.ChronicPoints),
			zap.Int("tags_answered", state.TagsAnswered),
		)
	}

	return changed
}

// classify determines the acuity category based on accumulated points.
// Uses simple majority vote with tie-breaking: acute > subacute > chronic
// (clinical safety: in doubt, assume acute).
func (s *AcuityScorer) classify(state *AcuityState) models.AcuityCategory {
	a := state.AcutePoints
	sub := state.SubacutePoints
	c := state.ChronicPoints

	if a >= sub && a >= c {
		return models.AcuityAcute
	}
	if sub >= a && sub >= c {
		return models.AcuitySubacute
	}
	return models.AcuityChronic
}

// GetCategory returns the current acuity classification.
func (s *AcuityScorer) GetCategory(state *AcuityState) models.AcuityCategory {
	return state.Category
}

// IsConfident returns true when enough acuity-tagged questions have been
// answered to make a reliable classification (>= 2 tags).
func (s *AcuityScorer) IsConfident(state *AcuityState) bool {
	return state.Confident
}

// ComputeLRScale returns a scaling factor for likelihood ratios based on the
// temporal acuity classification (G7 Phase 2: time-decay LR scaling).
//
// Rationale: questions about "sudden onset" or "worsening over weeks" have
// different discriminatory power depending on the acuity classification.
// A question whose LR was calibrated for ACUTE presentations loses power
// when the presentation is actually CHRONIC, and vice versa.
//
// Scaling factors (clinician-authored):
//
//	ACUTE presentations:
//	  - acuity_tag=ONSET questions: 1.0 (full weight, as calibrated)
//	  - acuity_tag=PATTERN questions: 0.6 (less discriminatory in acute)
//	  - acuity_tag=DURATION questions: 0.8
//	  - acuity_tag=PROGRESSION questions: 0.9
//
//	CHRONIC presentations:
//	  - acuity_tag=ONSET questions: 0.5 (onset less relevant when chronic)
//	  - acuity_tag=PATTERN questions: 1.0 (full weight)
//	  - acuity_tag=DURATION questions: 0.9
//	  - acuity_tag=PROGRESSION questions: 0.7
//
//	SUBACUTE: returns 1.0 for all tags (no scaling — intermediate category)
//	UNKNOWN or not confident: returns 1.0 (no modification before classification)
func (s *AcuityScorer) ComputeLRScale(state *AcuityState, acuityTag string) float64 {
	if !state.Confident || state.Category == models.AcuityUnknown {
		return 1.0
	}

	switch state.Category {
	case models.AcuityAcute:
		switch acuityTag {
		case "ONSET":
			return 1.0
		case "DURATION":
			return 0.8
		case "PROGRESSION":
			return 0.9
		case "PATTERN":
			return 0.6
		}
	case models.AcuityChronic:
		switch acuityTag {
		case "ONSET":
			return 0.5
		case "DURATION":
			return 0.9
		case "PROGRESSION":
			return 0.7
		case "PATTERN":
			return 1.0
		}
	case models.AcuitySubacute:
		return 1.0
	}

	return 1.0
}
