package services

import (
	"go.uber.org/zap"
)

const (
	// BehavioralGapThreshold is the adherence level below which
	// a BEHAVIORAL_GAP alert is triggered. This is distinct from
	// adherenceThreshold (0.70) in adherence_service.go which is used
	// for CM weight scaling. The 0.40 threshold represents clinically
	// significant non-adherence requiring behavioural intervention
	// and V-MCU titration suppression (gain factor → 0.0).
	BehavioralGapThreshold = 0.40
)

// BehavioralGapResult holds the outcome of a gap assessment.
type BehavioralGapResult struct {
	GapDetected bool    `json:"gap_detected"`
	Adherence   float64 `json:"adherence"`
	Threshold   float64 `json:"threshold"`
	DrugClass   string  `json:"drug_class,omitempty"`
}

// BehavioralGapDetector checks whether a patient's adherence has dropped
// below the clinically significant threshold.
type BehavioralGapDetector struct {
	logger *zap.Logger
}

func NewBehavioralGapDetector(logger *zap.Logger) *BehavioralGapDetector {
	return &BehavioralGapDetector{logger: logger}
}

// Assess evaluates whether the given adherence score triggers a behavioral gap.
func (d *BehavioralGapDetector) Assess(adherence float64, drugClass string) BehavioralGapResult {
	result := BehavioralGapResult{
		GapDetected: adherence < BehavioralGapThreshold,
		Adherence:   adherence,
		Threshold:   BehavioralGapThreshold,
		DrugClass:   drugClass,
	}
	if result.GapDetected {
		d.logger.Debug("behavioral gap detected",
			zap.String("drug_class", drugClass),
			zap.Float64("adherence", adherence))
	}
	return result
}
