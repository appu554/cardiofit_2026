package services

import (
	"fmt"
	"time"

	"kb-patient-profile/internal/models"
)

// StabilityConfig holds tunable parameters for the phenotype stability engine.
type StabilityConfig struct {
	DwellMinWeeks          int
	DwellExtendedWeeks     int
	FlapLookbackDays       int
	FlapMinOscillations    int
	HighMembershipProb     float64
	ModerateMembershipProb float64
	CGMStartGraceWeeks     int
	CGMStopGraceWeeks      int
	ConservatismRank       map[string]int // cluster label -> rank (1 = most conservative)
}

// StabilityInput is everything the engine needs for one evaluation.
type StabilityInput struct {
	PatientID         string
	RawClusterLabel   string
	MembershipProb    float64
	SeparabilityRatio float64
	IsNoise           bool
	RunDate           time.Time

	// Current patient state (nil for first assignment).
	CurrentState *models.PatientClusterState

	// Clinical context.
	OverrideEvents []models.OverrideEvent
	DomainDriver   string     // dominant MHRI domain driver
	CGMStartDate   *time.Time // nil if no CGM change
	CGMStopDate    *time.Time

	// Config (loaded from phenotype_stability.yaml).
	Config StabilityConfig
}

// StabilityEngine evaluates whether a raw HDBSCAN cluster assignment should
// be accepted as the patient's new stable phenotype, or held pending further
// confirmation (dwell gate, flap dampening, data-modality grace, etc.).
type StabilityEngine struct{}

// NewStabilityEngine returns a ready-to-use StabilityEngine.
func NewStabilityEngine() *StabilityEngine {
	return &StabilityEngine{}
}

// Evaluate runs the decision cascade for a single patient evaluation cycle.
//
// Decision cascade (evaluated in order):
//  1. First assignment   → ACCEPT / INITIAL
//  2. Same cluster       → ACCEPT / no transition
//  3. Noise label        → HOLD_DWELL / keep previous
//  4. CGM grace period   → HOLD_DWELL / keep previous
//  5. Flap check         → HOLD_FLAP / conservative cluster (unless override)
//  6. Override check     → ACCEPT / OVERRIDE
//  7. Dwell gate         → HOLD_DWELL / keep previous
//  8. Accept transition  → ACCEPT / GENUINE
func (e *StabilityEngine) Evaluate(input StabilityInput) models.StabilityDecision {
	// 1. First assignment — no prior state.
	if input.CurrentState == nil || input.CurrentState.CurrentStableCluster == "" {
		return models.StabilityDecision{
			PatientID:          input.PatientID,
			RawClusterLabel:    input.RawClusterLabel,
			StableClusterLabel: input.RawClusterLabel,
			Decision:           models.DecisionAccept,
			Reason:             "first assignment",
			Confidence:         input.MembershipProb,
			TransitionType:     models.TransitionTypeInitial,
			DomainDriver:       input.DomainDriver,
		}
	}

	stableCluster := input.CurrentState.CurrentStableCluster

	// 2. Same cluster — no transition needed.
	if input.RawClusterLabel == stableCluster {
		return models.StabilityDecision{
			PatientID:          input.PatientID,
			RawClusterLabel:    input.RawClusterLabel,
			StableClusterLabel: stableCluster,
			Decision:           models.DecisionAccept,
			Reason:             "same cluster",
			Confidence:         input.MembershipProb,
			DomainDriver:       input.DomainDriver,
		}
	}

	// 3. Noise label — hold previous stable cluster.
	if input.IsNoise {
		return models.StabilityDecision{
			PatientID:          input.PatientID,
			RawClusterLabel:    input.RawClusterLabel,
			StableClusterLabel: stableCluster,
			Decision:           models.DecisionHoldDwell,
			Reason:             "noise label held",
			Confidence:         input.CurrentState.Confidence,
			DomainDriver:       input.DomainDriver,
		}
	}

	// 4. Data modality grace period (CGM start/stop).
	if e.inCGMGracePeriod(input) {
		return models.StabilityDecision{
			PatientID:          input.PatientID,
			RawClusterLabel:    input.RawClusterLabel,
			StableClusterLabel: stableCluster,
			Decision:           models.DecisionHoldDwell,
			Reason:             "CGM data modality grace period",
			Confidence:         input.CurrentState.Confidence,
			DomainDriver:       input.DomainDriver,
		}
	}

	// 5. Flap check — but overrides beat flap holds.
	if input.CurrentState.IsFlapping && e.rawInFlapPair(input) {
		if !e.hasOverride(input) {
			conservative := e.moreConservativeCluster(input.CurrentState.FlapPair, input.Config)
			return models.StabilityDecision{
				PatientID:          input.PatientID,
				RawClusterLabel:    input.RawClusterLabel,
				StableClusterLabel: conservative,
				Decision:           models.DecisionHoldFlap,
				Reason:             fmt.Sprintf("flap dampened (%d oscillations)", input.CurrentState.FlapCount),
				Confidence:         input.MembershipProb,
				DomainDriver:       input.DomainDriver,
			}
		}
	}

	// 6. Override check — bypass dwell.
	if e.hasOverride(input) {
		evt := e.mostRecentOverride(input.OverrideEvents)
		return models.StabilityDecision{
			PatientID:          input.PatientID,
			RawClusterLabel:    input.RawClusterLabel,
			StableClusterLabel: input.RawClusterLabel,
			Decision:           models.DecisionAccept,
			Reason:             fmt.Sprintf("override: %s", evt.EventType),
			Confidence:         input.MembershipProb,
			TransitionType:     models.TransitionTypeOverride,
			TriggerEvent:       evt.EventType,
			DomainDriver:       input.DomainDriver,
		}
	}

	// 7. Dwell gate — has the pending assignment been sustained long enough?
	requiredDwellDays := e.requiredDwellDays(stableCluster, input.Config)
	pendingDays := e.pendingDwellDays(input)
	if pendingDays < requiredDwellDays {
		return models.StabilityDecision{
			PatientID:          input.PatientID,
			RawClusterLabel:    input.RawClusterLabel,
			StableClusterLabel: stableCluster,
			Decision:           models.DecisionHoldDwell,
			Reason:             fmt.Sprintf("dwell %d/%d days", pendingDays, requiredDwellDays),
			Confidence:         input.MembershipProb,
			DomainDriver:       input.DomainDriver,
		}
	}

	// 8. Accept the transition.
	return models.StabilityDecision{
		PatientID:          input.PatientID,
		RawClusterLabel:    input.RawClusterLabel,
		StableClusterLabel: input.RawClusterLabel,
		Decision:           models.DecisionAccept,
		Reason:             "dwell gate passed",
		Confidence:         input.MembershipProb,
		TransitionType:     models.TransitionTypeGenuine,
		DomainDriver:       input.DomainDriver,
	}
}

// --- internal helpers ---

// inCGMGracePeriod returns true if a CGM modality change occurred within the
// configured grace window.
func (e *StabilityEngine) inCGMGracePeriod(input StabilityInput) bool {
	if input.CGMStartDate != nil {
		graceEnd := input.CGMStartDate.AddDate(0, 0, input.Config.CGMStartGraceWeeks*7)
		if input.RunDate.Before(graceEnd) {
			return true
		}
	}
	if input.CGMStopDate != nil {
		graceEnd := input.CGMStopDate.AddDate(0, 0, input.Config.CGMStopGraceWeeks*7)
		if input.RunDate.Before(graceEnd) {
			return true
		}
	}
	return false
}

// rawInFlapPair returns true if the raw cluster label is one of the two
// labels in the patient's detected flap pair.
func (e *StabilityEngine) rawInFlapPair(input StabilityInput) bool {
	for _, label := range input.CurrentState.FlapPair {
		if input.RawClusterLabel == label {
			return true
		}
	}
	return false
}

// hasOverride returns true if any override events are present.
func (e *StabilityEngine) hasOverride(input StabilityInput) bool {
	return len(input.OverrideEvents) > 0
}

// moreConservativeCluster returns the cluster from the pair that has the
// LOWER conservatism rank number (i.e., is more conservative).
// Unknown clusters default to rank 99 (least conservative) so that a
// known-safe cluster is always preferred over an unmapped one.
func (e *StabilityEngine) moreConservativeCluster(pair []string, cfg StabilityConfig) string {
	if len(pair) < 2 {
		if len(pair) == 1 {
			return pair[0]
		}
		return ""
	}
	rankA := e.clusterRank(pair[0], cfg)
	rankB := e.clusterRank(pair[1], cfg)
	if rankA <= rankB {
		return pair[0]
	}
	return pair[1]
}

// clusterRank returns the conservatism rank for a cluster, defaulting to 99
// (least conservative) if the label is not in the config map.
func (e *StabilityEngine) clusterRank(label string, cfg StabilityConfig) int {
	if rank, ok := cfg.ConservatismRank[label]; ok {
		return rank
	}
	return 99
}

// mostRecentOverride returns the override event with the latest EventDate.
func (e *StabilityEngine) mostRecentOverride(events []models.OverrideEvent) models.OverrideEvent {
	best := events[0]
	for _, evt := range events[1:] {
		if evt.EventDate.After(best.EventDate) {
			best = evt
		}
	}
	return best
}

// requiredDwellDays returns the number of days a pending assignment must be
// sustained before it is accepted. Conservative clusters (rank <= 2) use the
// extended dwell window.
func (e *StabilityEngine) requiredDwellDays(currentStable string, cfg StabilityConfig) int {
	rank := e.clusterRank(currentStable, cfg)
	if rank <= 2 {
		return cfg.DwellExtendedWeeks * 7
	}
	return cfg.DwellMinWeeks * 7
}

// pendingDwellDays calculates how many days the raw cluster has been pending.
// If there is no pending state recorded, it falls back to the current state's
// DwellDays (which represents time since last stable assignment).
func (e *StabilityEngine) pendingDwellDays(input StabilityInput) int {
	if input.CurrentState.PendingSince != nil {
		days := int(input.RunDate.Sub(*input.CurrentState.PendingSince).Hours() / 24)
		if days < 0 {
			return 0 // guard against PendingSince in the future (data error)
		}
		return days
	}
	return input.CurrentState.DwellDays
}
