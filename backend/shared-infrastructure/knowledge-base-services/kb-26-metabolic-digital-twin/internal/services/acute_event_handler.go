package services

import (
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
)

// PAIUpdater is the interface for feeding acute deviations into the PAI engine.
// Implemented by a wrapper around ComputePAI in the server layer.
// Nil-safe: when unset, acute events don't update PAI (tests, standalone mode).
type PAIUpdater interface {
	UpdateFromAcuteEvent(patientID string, deviationPercent float64, severity string, vitalType string)
}

// AcuteEventHandler orchestrates the full acute-on-chronic detection pipeline:
// baseline computation → deviation scoring → compound pattern detection →
// event creation → resolution tracking → PAI velocity/proximity update.
type AcuteEventHandler struct {
	config     *AcuteDetectionConfig
	repo       *AcuteRepository // nil-safe for tests
	paiUpdater PAIUpdater       // nil-safe: when unset, PAI is not updated
	log        *zap.Logger
}

// SetPAIUpdater injects the PAI integration callback. Called from server wiring.
func (h *AcuteEventHandler) SetPAIUpdater(u PAIUpdater) {
	h.paiUpdater = u
}

// NewAcuteEventHandler constructs a handler. repo may be nil for unit tests.
func NewAcuteEventHandler(config *AcuteDetectionConfig, repo *AcuteRepository, log *zap.Logger) *AcuteEventHandler {
	if config == nil {
		config = DefaultAcuteDetectionConfig()
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &AcuteEventHandler{
		config: config,
		repo:   repo,
		log:    log,
	}
}

// HandleNewReading runs the full acute detection pipeline for a single vital
// sign reading. It returns the detected acute event (nil if sub-threshold)
// and a list of event IDs that were resolved by this reading.
func (h *AcuteEventHandler) HandleNewReading(
	patientID string,
	vitalType string,
	value float64,
	timestamp time.Time,
	readings []float64,
	readingTimestamps []time.Time,
	context DeviationContext,
	recentDeviations []models.DeviationResult,
	compoundContext CompoundContext,
	activeEvents []models.AcuteEvent,
) (*models.AcuteEvent, []string) {

	// Step 1: Compute baseline from readings.
	baseline := ComputeBaseline(readings, readingTimestamps, 7)
	baseline.PatientID = patientID
	baseline.VitalSignType = vitalType

	// Persist baseline if repo is available.
	if h.repo != nil {
		if err := h.repo.SaveBaseline(&baseline); err != nil {
			h.log.Error("failed to save baseline", zap.Error(err))
		}
	}

	// Step 2: Compute deviation from baseline.
	deviation := ComputeDeviation(value, baseline, vitalType, h.config, context)

	// Step 9 (early): Check resolution of active events regardless of deviation.
	resolvedIDs := h.checkResolution(patientID, vitalType, value, baseline, activeEvents)

	// Step 3: If no clinical significance → return nil event + any resolved IDs.
	if deviation.ClinicalSignificance == "" {
		return nil, resolvedIDs
	}

	// Step 4: Check compound patterns by combining current deviation with recent.
	allDeviations := append(recentDeviations, deviation)
	compoundMatches := DetectCompoundPatterns(allDeviations, compoundContext)

	var event *models.AcuteEvent

	// Step 5-6: Create event — compound or single-vital.
	if len(compoundMatches) > 0 {
		best := compoundMatches[0]
		event = h.buildCompoundEvent(patientID, vitalType, value, timestamp, deviation, best)
	} else {
		event = h.buildSingleEvent(patientID, vitalType, value, timestamp, deviation)
	}

	// Step 7: Map severity to escalation tier.
	event.EscalationTier = mapEscalationTier(event.Severity)

	// Step 8: Set suggested action from config/event type.
	event.SuggestedAction = suggestedActionForEvent(event.EventType, event.Severity)

	// Persist event if repo is available.
	if h.repo != nil {
		if err := h.repo.SaveEvent(event); err != nil {
			h.log.Error("failed to save acute event", zap.Error(err))
		}
	}

	// Update PAI velocity + proximity with acute deviation data.
	// This feeds the acute event into the PAI's real-time urgency
	// computation so the escalation engine sees the updated PAI tier.
	if h.paiUpdater != nil {
		h.paiUpdater.UpdateFromAcuteEvent(
			patientID,
			deviation.DeviationPercent,
			event.Severity,
			vitalType,
		)
	}

	return event, resolvedIDs
}

// checkResolution examines active events and resolves any where the new reading
// is within 10% of baseline for the same vital type.
func (h *AcuteEventHandler) checkResolution(
	patientID string,
	vitalType string,
	value float64,
	baseline models.PatientBaselineSnapshot,
	activeEvents []models.AcuteEvent,
) []string {
	var resolvedIDs []string

	if baseline.BaselineMedian == 0 {
		return resolvedIDs
	}

	deviationPct := math.Abs(value-baseline.BaselineMedian) / baseline.BaselineMedian * 100.0

	if deviationPct > 10.0 {
		return resolvedIDs
	}

	// Reading is within 10% of baseline — resolve matching active events.
	for _, ev := range activeEvents {
		if ev.VitalSignType == vitalType && ev.ResolvedAt == nil {
			resolvedIDs = append(resolvedIDs, ev.ID.String())

			if h.repo != nil {
				if err := h.repo.MarkResolved(ev.ID, "READING_RECOVERY"); err != nil {
					h.log.Error("failed to mark event resolved",
						zap.String("event_id", ev.ID.String()),
						zap.Error(err))
				}
			}
		}
	}

	return resolvedIDs
}

// buildSingleEvent creates an AcuteEvent from a single vital-sign deviation.
func (h *AcuteEventHandler) buildSingleEvent(
	patientID string,
	vitalType string,
	value float64,
	timestamp time.Time,
	deviation models.DeviationResult,
) *models.AcuteEvent {
	return &models.AcuteEvent{
		ID:                 uuid.New(),
		PatientID:          patientID,
		DetectedAt:         timestamp,
		EventType:          mapEventType(vitalType),
		Severity:           deviation.ClinicalSignificance,
		VitalSignType:      vitalType,
		CurrentValue:       value,
		BaselineMedian:     deviation.BaselineMedian,
		DeviationPercent:   deviation.DeviationPercent,
		DeviationAbsolute:  deviation.DeviationAbsolute,
		Direction:          deviation.Direction,
		GapAmplified:       deviation.GapAmplified,
		ConfounderDampened: deviation.ConfounderDampened,
	}
}

// buildCompoundEvent creates an AcuteEvent from a compound pattern match.
func (h *AcuteEventHandler) buildCompoundEvent(
	patientID string,
	vitalType string,
	value float64,
	timestamp time.Time,
	deviation models.DeviationResult,
	compound models.CompoundPatternMatch,
) *models.AcuteEvent {
	return &models.AcuteEvent{
		ID:                 uuid.New(),
		PatientID:          patientID,
		DetectedAt:         timestamp,
		EventType:          mapCompoundEventType(compound.PatternName),
		Severity:           compound.CompoundSeverity,
		VitalSignType:      vitalType,
		CurrentValue:       value,
		BaselineMedian:     deviation.BaselineMedian,
		DeviationPercent:   deviation.DeviationPercent,
		DeviationAbsolute:  deviation.DeviationAbsolute,
		Direction:          deviation.Direction,
		CompoundPattern:    compound.PatternName,
		GapAmplified:       deviation.GapAmplified,
		ConfounderDampened: deviation.ConfounderDampened,
	}
}

// mapEventType translates a vital sign type to its acute event type.
func mapEventType(vitalType string) string {
	switch vitalType {
	case "EGFR":
		return string(models.AcuteKidneyInjury)
	case "SBP":
		return string(models.AcuteHypertensiveEmergency)
	case "WEIGHT":
		return string(models.AcuteFluidOverload)
	case "GLUCOSE":
		return string(models.AcuteSevereHyperglycaemia)
	case "POTASSIUM":
		return string(models.AcuteMedicationCrisis)
	default:
		return string(models.AcuteMeasurementGapDeviation)
	}
}

// mapCompoundEventType translates a compound pattern name to its acute event type.
func mapCompoundEventType(patternName string) string {
	switch patternName {
	case "CARDIORENAL_SYNDROME":
		return string(models.AcuteCompoundCardiorenal)
	case "INFECTION_CASCADE":
		return string(models.AcuteCompoundInfection)
	case "MEDICATION_CRISIS":
		return string(models.AcuteMedicationCrisis)
	case "FLUID_OVERLOAD_TRIAD":
		return string(models.AcuteFluidOverload)
	default:
		return patternName
	}
}

// mapEscalationTier translates severity to escalation tier.
func mapEscalationTier(severity string) string {
	switch severity {
	case "CRITICAL":
		return "SAFETY"
	case "HIGH":
		return "IMMEDIATE"
	case "MODERATE":
		return "URGENT"
	default:
		return "ROUTINE"
	}
}

// suggestedActionForEvent returns a clinical action recommendation based on event type and severity.
func suggestedActionForEvent(eventType, severity string) string {
	switch eventType {
	case string(models.AcuteKidneyInjury):
		if severity == "CRITICAL" {
			return "Immediate nephrology consult. Hold nephrotoxic agents. Check urine output and fluid status."
		}
		return "Urgent renal function review. Reassess ACEi/ARB dosing and hydration status."
	case string(models.AcuteHypertensiveEmergency):
		if severity == "CRITICAL" {
			return "Hypertensive emergency protocol. IV antihypertensive therapy. Monitor end-organ damage."
		}
		return "Urgent BP management review. Assess medication adherence and titrate therapy."
	case string(models.AcuteFluidOverload):
		return "Assess volume status. Consider diuretic adjustment and sodium restriction."
	case string(models.AcuteCompoundCardiorenal):
		return "Urgent nephrology/cardiology review. Assess volume status and cardiac output."
	case string(models.AcuteCompoundInfection):
		return "Screen for infection source. Consider blood cultures and empiric therapy."
	case string(models.AcuteSevereHyperglycaemia):
		return "Urgent glycaemic management. Assess for DKA/HHS. Review insulin regimen."
	case string(models.AcuteMedicationCrisis):
		return "Review recent medication changes. Consider dose reduction or discontinuation."
	default:
		return "Clinical review recommended. Assess for acute-on-chronic deterioration."
	}
}
