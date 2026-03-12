// Package governance provides Tier-7 governance event emission for KB-16
// Integration component connects governance to the interpretation engine.
package governance

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// GOVERNANCE-AWARE INTERPRETATION
// =============================================================================

// InterpretationObserver observes interpretation results and emits governance events
type InterpretationObserver struct {
	publisher *Publisher
	log       *logrus.Entry
	config    ObserverConfig
}

// ObserverConfig configures governance observation behavior
type ObserverConfig struct {
	// Enable event types
	CriticalLabEvents    bool
	PanicLabEvents       bool
	DeltaCheckEvents     bool
	PatternDetection     bool
	TrendingEvents       bool
	CareGapEvents        bool

	// SLA configuration (in minutes)
	PanicAckSLAMin       int
	CriticalAckSLAMin    int
	HighAckSLAMin        int
	PanicReviewSLAMin    int
	CriticalReviewSLAMin int
	HighReviewSLAMin     int

	// Escalation paths
	PanicEscalation      string
	CriticalEscalation   string
	HighEscalation       string
}

// DefaultObserverConfig returns production-safe defaults
func DefaultObserverConfig() ObserverConfig {
	return ObserverConfig{
		CriticalLabEvents:    true,
		PanicLabEvents:       true,
		DeltaCheckEvents:     true,
		PatternDetection:     true,
		TrendingEvents:       true,
		CareGapEvents:        true,

		PanicAckSLAMin:       15,
		CriticalAckSLAMin:    30,
		HighAckSLAMin:        60,
		PanicReviewSLAMin:    30,
		CriticalReviewSLAMin: 60,
		HighReviewSLAMin:     120,

		PanicEscalation:      "rapid_response",
		CriticalEscalation:   "attending_physician",
		HighEscalation:       "care_team",
	}
}

// NewInterpretationObserver creates a new governance observer
func NewInterpretationObserver(publisher *Publisher, log *logrus.Entry) *InterpretationObserver {
	return &InterpretationObserver{
		publisher: publisher,
		log:       log.WithField("component", "governance-observer"),
		config:    DefaultObserverConfig(),
	}
}

// NewInterpretationObserverWithConfig creates an observer with custom config
func NewInterpretationObserverWithConfig(publisher *Publisher, log *logrus.Entry, config ObserverConfig) *InterpretationObserver {
	return &InterpretationObserver{
		publisher: publisher,
		log:       log.WithField("component", "governance-observer"),
		config:    config,
	}
}

// =============================================================================
// OBSERVATION METHODS
// =============================================================================

// OnInterpretation observes an interpretation result and emits governance events
func (o *InterpretationObserver) OnInterpretation(ctx context.Context, result *types.InterpretedResult, provenance *EventProvenance) error {
	if result == nil {
		return nil
	}

	interp := &result.Interpretation

	// Check for panic values
	if interp.IsPanic && o.config.PanicLabEvents {
		if err := o.emitPanicEvent(ctx, result, provenance); err != nil {
			o.log.WithError(err).Error("Failed to emit panic event")
			// Don't return - try to emit other events
		}
	}

	// Check for critical values
	if interp.IsCritical && o.config.CriticalLabEvents {
		if err := o.emitCriticalEvent(ctx, result, provenance); err != nil {
			o.log.WithError(err).Error("Failed to emit critical event")
		}
	}

	// Check for significant delta
	if interp.DeltaCheck != nil && interp.DeltaCheck.IsSignificant && o.config.DeltaCheckEvents {
		if err := o.emitDeltaEvent(ctx, result, provenance); err != nil {
			o.log.WithError(err).Error("Failed to emit delta event")
		}
	}

	return nil
}

// OnBatchInterpretation observes batch interpretations
func (o *InterpretationObserver) OnBatchInterpretation(ctx context.Context, results []types.InterpretedResult, provenance *EventProvenance) error {
	for _, result := range results {
		if err := o.OnInterpretation(ctx, &result, provenance); err != nil {
			o.log.WithError(err).WithField("result_id", result.Result.ID).Warn("Failed to observe interpretation")
		}
	}
	return nil
}

// OnTrendDetected observes trend analysis results
func (o *InterpretationObserver) OnTrendDetected(ctx context.Context, patientID, labCode, labName string, trend *types.TrendAnalysis, provenance *EventProvenance) error {
	if trend == nil || !o.config.TrendingEvents {
		return nil
	}

	// Only emit for worsening or volatile trends
	if trend.Trajectory != types.TrajectoryWorsening && trend.Trajectory != types.TrajectoryVolatile {
		return nil
	}

	event := o.createTrendEvent(patientID, labCode, labName, trend, provenance)
	return o.publisher.Publish(ctx, event)
}

// OnCareGapIdentified observes care gap detection
func (o *InterpretationObserver) OnCareGapIdentified(ctx context.Context, patientID, gapCode, gapName string, daysOverdue int, provenance *EventProvenance) error {
	if !o.config.CareGapEvents {
		return nil
	}

	event := CareGapEvent(patientID, gapCode, gapName, daysOverdue)
	if provenance != nil {
		event.Provenance = *provenance
	}

	return o.publisher.Publish(ctx, event)
}

// OnPatternDetected observes clinical pattern detection
func (o *InterpretationObserver) OnPatternDetected(ctx context.Context, patientID string, pattern *types.DetectedPattern, provenance *EventProvenance) error {
	if pattern == nil || !o.config.PatternDetection {
		return nil
	}

	severity := o.patternSeverityToGovernance(string(pattern.Severity))
	event := ClinicalPatternEvent(patientID, pattern.Code, pattern.Name, pattern.Confidence, severity)
	if provenance != nil {
		event.Provenance = *provenance
	}

	// Critical patterns need immediate publishing
	if severity == SeverityCritical {
		return o.publisher.PublishCritical(ctx, event)
	}
	return o.publisher.Publish(ctx, event)
}

// =============================================================================
// EVENT EMISSION HELPERS
// =============================================================================

func (o *InterpretationObserver) emitPanicEvent(ctx context.Context, result *types.InterpretedResult, provenance *EventProvenance) error {
	lab := &result.Result
	interp := &result.Interpretation

	flag := "PANIC"
	if interp.Flag == types.FlagPanicLow {
		flag = "PANIC_LOW"
	} else if interp.Flag == types.FlagPanicHigh {
		flag = "PANIC_HIGH"
	}

	event := PanicLabEvent(
		lab.PatientID,
		lab.ID.String(),
		lab.Code,
		lab.Name,
		*lab.ValueNumeric,
		lab.Unit,
		flag,
	)

	// Override with configured SLAs
	event.AcknowledgmentSLAMin = o.config.PanicAckSLAMin
	event.ReviewSLAMin = o.config.PanicReviewSLAMin
	event.EscalationPath = o.config.PanicEscalation

	// Add provenance if available
	if provenance != nil {
		event.Provenance = *provenance
	}

	// Add clinical recommendations to payload
	if len(interp.Recommendations) > 0 {
		event.Payload["recommendations"] = interp.Recommendations
	}
	event.Payload["clinical_comment"] = interp.ClinicalComment

	o.log.WithFields(map[string]interface{}{
		"patient_id": lab.PatientID,
		"lab_code":   lab.Code,
		"value":      *lab.ValueNumeric,
		"flag":       flag,
	}).Warn("PANIC value detected - emitting governance event")

	return o.publisher.PublishCritical(ctx, event)
}

func (o *InterpretationObserver) emitCriticalEvent(ctx context.Context, result *types.InterpretedResult, provenance *EventProvenance) error {
	lab := &result.Result
	interp := &result.Interpretation

	flag := "CRITICAL"
	if interp.Flag == types.FlagCriticalLow {
		flag = "CRITICAL_LOW"
	} else if interp.Flag == types.FlagCriticalHigh {
		flag = "CRITICAL_HIGH"
	}

	event := CriticalLabEvent(
		lab.PatientID,
		lab.ID.String(),
		lab.Code,
		lab.Name,
		*lab.ValueNumeric,
		lab.Unit,
		flag,
	)

	// Override with configured SLAs
	event.AcknowledgmentSLAMin = o.config.CriticalAckSLAMin
	event.ReviewSLAMin = o.config.CriticalReviewSLAMin
	event.EscalationPath = o.config.CriticalEscalation

	if provenance != nil {
		event.Provenance = *provenance
	}

	if len(interp.Recommendations) > 0 {
		event.Payload["recommendations"] = interp.Recommendations
	}
	event.Payload["clinical_comment"] = interp.ClinicalComment

	o.log.WithFields(map[string]interface{}{
		"patient_id": lab.PatientID,
		"lab_code":   lab.Code,
		"value":      *lab.ValueNumeric,
		"flag":       flag,
	}).Warn("CRITICAL value detected - emitting governance event")

	return o.publisher.Publish(ctx, event)
}

func (o *InterpretationObserver) emitDeltaEvent(ctx context.Context, result *types.InterpretedResult, provenance *EventProvenance) error {
	lab := &result.Result
	delta := result.Interpretation.DeltaCheck

	event := SignificantDeltaEvent(
		lab.PatientID,
		lab.ID.String(),
		lab.Code,
		lab.Name,
		*lab.ValueNumeric,
		delta.PreviousValue,
		delta.PercentChange,
		lab.Unit,
	)

	if provenance != nil {
		event.Provenance = *provenance
	}

	// Add context
	event.Payload["window_hours"] = delta.WindowHours
	event.Payload["previous_time"] = delta.PreviousTime.Format(time.RFC3339)

	o.log.WithFields(map[string]interface{}{
		"patient_id":     lab.PatientID,
		"lab_code":       lab.Code,
		"current_value":  *lab.ValueNumeric,
		"previous_value": delta.PreviousValue,
		"percent_change": delta.PercentChange,
	}).Info("Significant delta detected - emitting governance event")

	return o.publisher.Publish(ctx, event)
}

func (o *InterpretationObserver) createTrendEvent(patientID, labCode, labName string, trend *types.TrendAnalysis, provenance *EventProvenance) *GovernanceEvent {
	eventType := EventWorseningTrend
	if trend.Trajectory == types.TrajectoryVolatile {
		eventType = EventVolatileTrend
	}

	event := NewGovernanceEvent(eventType, patientID)
	event.Severity = SeverityMedium
	event.Priority = 3
	event.Title = fmt.Sprintf("Trend Alert: %s", labName)
	event.Description = fmt.Sprintf("%s shows %s trajectory with rate of change %.3f per day",
		labName, trend.Trajectory, trend.RateOfChange)
	event.RequiresReview = true
	event.ReviewSLAMin = o.config.HighReviewSLAMin

	event.Payload = map[string]interface{}{
		"lab_code":       labCode,
		"lab_name":       labName,
		"trajectory":     string(trend.Trajectory),
		"rate_of_change": trend.RateOfChange,
	}

	// Add window data
	if len(trend.Windows) > 0 {
		windowData := make(map[string]interface{})
		for name, window := range trend.Windows {
			windowData[name] = map[string]interface{}{
				"slope":            window.Slope,
				"r_squared":        window.RSquared,
				"data_points_count": len(window.DataPoints),
				"days":             window.Days,
			}
		}
		event.Payload["windows"] = windowData
	}

	if provenance != nil {
		event.Provenance = *provenance
	}

	return event
}

func (o *InterpretationObserver) patternSeverityToGovernance(severity string) Severity {
	switch severity {
	case "critical", "CRITICAL":
		return SeverityCritical
	case "high", "HIGH":
		return SeverityHigh
	case "medium", "MEDIUM":
		return SeverityMedium
	case "low", "LOW":
		return SeverityLow
	default:
		return SeverityInfo
	}
}

// =============================================================================
// PROVENANCE BUILDER
// =============================================================================

// ProvenanceBuilder helps construct provenance records
type ProvenanceBuilder struct {
	provenance EventProvenance
}

// NewProvenanceBuilder creates a new provenance builder
func NewProvenanceBuilder(interpretationVersion string) *ProvenanceBuilder {
	return &ProvenanceBuilder{
		provenance: EventProvenance{
			InterpretationVersion: interpretationVersion,
			Timestamp:             time.Now().UTC(),
		},
	}
}

// AddKB8Calculation records a KB-8 calculation
func (b *ProvenanceBuilder) AddKB8Calculation(calculator string, input, output map[string]interface{}, formula, version string) *ProvenanceBuilder {
	b.provenance.KB8Calculations = append(b.provenance.KB8Calculations, KB8Calculation{
		Calculator: calculator,
		Input:      input,
		Output:     output,
		Formula:    formula,
		Version:    version,
		Timestamp:  time.Now().UTC(),
	})
	return b
}

// AddReferenceRange records a reference range used
func (b *ProvenanceBuilder) AddReferenceRange(code, source, version string, low, high, critLow, critHigh *float64, ageAdj, sexAdj bool) *ProvenanceBuilder {
	b.provenance.ReferenceRanges = append(b.provenance.ReferenceRanges, ReferenceUsed{
		Code:         code,
		Source:       source,
		Version:      version,
		Low:          low,
		High:         high,
		CriticalLow:  critLow,
		CriticalHigh: critHigh,
		AgeAdjusted:  ageAdj,
		SexAdjusted:  sexAdj,
	})
	return b
}

// AddRule records a rule that was applied
func (b *ProvenanceBuilder) AddRule(ruleID, ruleName, version string, triggered bool, confidence float64) *ProvenanceBuilder {
	b.provenance.RulesApplied = append(b.provenance.RulesApplied, RuleApplied{
		RuleID:     ruleID,
		RuleName:   ruleName,
		Version:    version,
		Triggered:  triggered,
		Confidence: confidence,
	})
	return b
}

// Build returns the completed provenance record
func (b *ProvenanceBuilder) Build() *EventProvenance {
	return &b.provenance
}
