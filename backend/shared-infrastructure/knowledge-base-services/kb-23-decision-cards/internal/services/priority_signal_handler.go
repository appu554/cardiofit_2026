package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

// PrioritySignalHandler processes priority signals and creates decision cards.
type PrioritySignalHandler struct {
	db          *database.Database
	gateCache   *MCUGateCache
	kb19        *KB19Publisher
	hypoHandler *HypoglycaemiaHandler
	metrics     *metrics.Collector
	log         *zap.Logger
}

// NewPrioritySignalHandler creates a new handler for priority signals.
func NewPrioritySignalHandler(
	db *database.Database,
	gateCache *MCUGateCache,
	kb19 *KB19Publisher,
	hypoHandler *HypoglycaemiaHandler,
	m *metrics.Collector,
	log *zap.Logger,
) *PrioritySignalHandler {
	return &PrioritySignalHandler{
		db:          db,
		gateCache:   gateCache,
		kb19:        kb19,
		hypoHandler: hypoHandler,
		metrics:     m,
		log:         log,
	}
}

// Handle dispatches a priority signal to the appropriate handler.
func (h *PrioritySignalHandler) Handle(ctx context.Context, action PriorityRouteAction, patientID string, rawMsg json.RawMessage) error {
	var env priorityEnvelope
	if err := json.Unmarshal(rawMsg, &env); err != nil {
		return fmt.Errorf("unmarshal priority envelope: %w", err)
	}

	switch action {
	case RouteHypo:
		return h.handleHypo(ctx, env)
	case RouteOrthostatic:
		return h.handleOrthostatic(ctx, env)
	case RoutePotassium:
		return h.handlePotassium(ctx, env)
	case RouteAdverseEvent:
		return h.handleAdverseEvent(ctx, env)
	case RouteHospitalisation:
		return h.handleHospitalisation(ctx, env)
	default:
		return nil
	}
}

// handleHypo delegates to the existing HypoglycaemiaHandler.
func (h *PrioritySignalHandler) handleHypo(ctx context.Context, env priorityEnvelope) error {
	var payload struct {
		GlucoseMmolL     float64 `json:"glucose_mmol_l"`
		Source           string  `json:"source"`
		DurationMinutes  int     `json:"duration_minutes"`
		PredictedAtHours float64 `json:"predicted_at_hours"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal hypo payload: %w", err)
	}

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	req := &models.SafetyAlertRequest{
		PatientID:        patientID,
		Source:           payload.Source,
		GlucoseMmolL:     payload.GlucoseMmolL,
		DurationMinutes:  payload.DurationMinutes,
		PredictedAtHours: payload.PredictedAtHours,
		Timestamp:        time.Now(),
	}

	_, err = h.hypoHandler.HandleAlert(ctx, req)
	if err != nil {
		return fmt.Errorf("hypo handler: %w", err)
	}

	h.log.Info("priority hypo signal handled via existing handler",
		zap.String("patient_id", env.PatientID),
	)
	return nil
}

// orthostaticPayload is the expected payload for orthostatic signals.
type orthostaticPayload struct {
	SBPDrop     float64 `json:"sbp_drop"`
	DBPDrop     float64 `json:"dbp_drop"`
	StandingSBP float64 `json:"standing_sbp"`
}

// handleOrthostatic generates an ORTHOSTATIC_ALERT card when SBP drop >20 mmHg.
func (h *PrioritySignalHandler) handleOrthostatic(ctx context.Context, env priorityEnvelope) error {
	var payload orthostaticPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal orthostatic payload: %w", err)
	}

	if payload.SBPDrop <= 20 {
		h.log.Debug("orthostatic SBP drop below threshold, skipping",
			zap.Float64("sbp_drop", payload.SBPDrop),
		)
		return nil
	}

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinicianSummary := fmt.Sprintf(
		"ORTHOSTATIC HYPOTENSION: SBP drop %.0f mmHg, DBP drop %.0f mmHg, standing SBP %.0f mmHg. Review medications causing orthostatic drop.",
		payload.SBPDrop, payload.DBPDrop, payload.StandingSBP,
	)
	notes := clinicianSummary

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               "PRIORITY_ORTHOSTATIC_ALERT",
		NodeID:                   "CROSS_NODE",
		PrimaryDifferentialID:    "ORTHOSTATIC_HYPOTENSION",
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GateModify,
		MCUGateRationale:         clinicianSummary,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyUrgent,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         "A significant blood pressure drop was detected when standing. Your medications are being reviewed.",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save orthostatic card: %w", err)
	}

	if err := h.gateCache.WriteGate(card); err != nil {
		h.log.Error("gate cache write failed for orthostatic alert", zap.Error(err))
	}

	go h.kb19.PublishGateChanged(card)

	h.log.Warn("priority orthostatic alert card generated",
		zap.String("card_id", card.CardID.String()),
		zap.String("patient_id", env.PatientID),
		zap.Float64("sbp_drop", payload.SBPDrop),
	)
	return nil
}

// potassiumPayload is the expected payload for potassium signals.
type potassiumPayload struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// handlePotassium generates a HIGH_POTASSIUM card when K+ >5.5 mmol/L.
func (h *PrioritySignalHandler) handlePotassium(ctx context.Context, env priorityEnvelope) error {
	var payload potassiumPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal potassium payload: %w", err)
	}

	if payload.Value <= 5.5 {
		h.log.Debug("potassium value below threshold, skipping",
			zap.Float64("value", payload.Value),
		)
		return nil
	}

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinicianSummary := fmt.Sprintf(
		"HIGH POTASSIUM: K+ %.1f %s. On finerenone — dose hold required per safety protocol.",
		payload.Value, payload.Unit,
	)
	notes := clinicianSummary

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               "PRIORITY_HIGH_POTASSIUM",
		NodeID:                   "CROSS_NODE",
		PrimaryDifferentialID:    "HYPERKALAEMIA",
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GatePause,
		MCUGateRationale:         clinicianSummary,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyImmediate,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         "Your potassium level is elevated. Your medication dosing has been paused pending review.",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Add MEDICATION_HOLD recommendation for finerenone
	rec := models.CardRecommendation{
		RecommendationID:       uuid.New(),
		CardID:                 card.CardID,
		RecType:                models.RecMedicationHold,
		Urgency:                models.UrgencyImmediate,
		Target:                 "finerenone",
		ActionTextEn:           "Hold finerenone until potassium is confirmed <5.0 mmol/L on repeat testing.",
		RationaleEn:            fmt.Sprintf("K+ %.1f %s exceeds safety threshold of 5.5 mmol/L for MRA therapy.", payload.Value, payload.Unit),
		ConfidenceTierRequired: models.TierFirm,
		SortOrder:              1,
		CreatedAt:              time.Now(),
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save potassium card: %w", err)
	}
	if err := h.db.DB.Create(&rec).Error; err != nil {
		return fmt.Errorf("save potassium recommendation: %w", err)
	}

	if err := h.gateCache.WriteGate(card); err != nil {
		h.log.Error("gate cache write failed for potassium alert", zap.Error(err))
	}

	go h.kb19.PublishGateChanged(card)

	h.log.Warn("priority high potassium card generated",
		zap.String("card_id", card.CardID.String()),
		zap.String("patient_id", env.PatientID),
		zap.Float64("k_value", payload.Value),
	)
	return nil
}

// adverseEventPayload is the expected payload for adverse event signals.
type adverseEventPayload struct {
	EventType   string `json:"event_type"`
	Severity    string `json:"severity"`
	DrugClass   string `json:"drug_class"`
	Description string `json:"description"`
}

// handleAdverseEvent generates an ADR_REVIEW card. HARD_BLOCK or SEVERE triggers HALT + PHYSICIAN_ALERT.
func (h *PrioritySignalHandler) handleAdverseEvent(ctx context.Context, env priorityEnvelope) error {
	var payload adverseEventPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal adverse event payload: %w", err)
	}

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	gate := models.GatePause
	safetyTier := models.SafetyUrgent
	isHardBlock := payload.Severity == "SEVERE" || strings.Contains(payload.EventType, "HARD_BLOCK")
	if isHardBlock {
		gate = models.GateHalt
		safetyTier = models.SafetyImmediate
	}

	clinicianSummary := fmt.Sprintf(
		"ADVERSE DRUG REACTION: %s — severity %s, drug class %s. %s",
		payload.EventType, payload.Severity, payload.DrugClass, payload.Description,
	)
	notes := clinicianSummary

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               "PRIORITY_ADR_REVIEW",
		NodeID:                   "CROSS_NODE",
		PrimaryDifferentialID:    "ADVERSE_DRUG_REACTION",
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  gate,
		MCUGateRationale:         clinicianSummary,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityModerate,
		SafetyTier:               safetyTier,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		PendingReaffirmation:     isHardBlock,
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         "A possible adverse drug reaction has been reported. Your care team has been notified for review.",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Add MEDICATION_REVIEW recommendation
	rec := models.CardRecommendation{
		RecommendationID:       uuid.New(),
		CardID:                 card.CardID,
		RecType:                models.RecMedicationReview,
		Urgency:                models.Urgency(safetyTier),
		Target:                 payload.DrugClass,
		ActionTextEn:           fmt.Sprintf("Review %s therapy due to adverse event: %s.", payload.DrugClass, payload.EventType),
		RationaleEn:            fmt.Sprintf("Patient-reported adverse event (%s) requires clinician validation.", payload.Description),
		ConfidenceTierRequired: models.TierProbable,
		SortOrder:              1,
		CreatedAt:              time.Now(),
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save ADR card: %w", err)
	}
	if err := h.db.DB.Create(&rec).Error; err != nil {
		return fmt.Errorf("save ADR recommendation: %w", err)
	}

	if err := h.gateCache.WriteGate(card); err != nil {
		h.log.Error("gate cache write failed for ADR alert", zap.Error(err))
	}

	go h.kb19.PublishGateChanged(card)

	h.log.Warn("priority ADR review card generated",
		zap.String("card_id", card.CardID.String()),
		zap.String("patient_id", env.PatientID),
		zap.String("severity", payload.Severity),
		zap.Bool("hard_block", isHardBlock),
	)
	return nil
}

// hospitalisationPayload is the expected payload for hospitalisation signals.
type hospitalisationPayload struct {
	Reason        string `json:"reason"`
	AdmissionDate string `json:"admission_date"`
}

// handleHospitalisation generates a CLINICAL_REVIEW card and pauses all active protocol cards.
func (h *PrioritySignalHandler) handleHospitalisation(ctx context.Context, env priorityEnvelope) error {
	var payload hospitalisationPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal hospitalisation payload: %w", err)
	}

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinicianSummary := fmt.Sprintf(
		"HOSPITALISATION: reason=%s, admission=%s. All automated titration suspended.",
		payload.Reason, payload.AdmissionDate,
	)
	notes := clinicianSummary

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               "PRIORITY_CLINICAL_REVIEW",
		NodeID:                   "CROSS_NODE",
		PrimaryDifferentialID:    "HOSPITALISATION",
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GateHalt,
		MCUGateRationale:         clinicianSummary,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyImmediate,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		PendingReaffirmation:     true,
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         "Hospital admission recorded. All automated medication adjustments have been paused.",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Pause all active protocol cards for this patient
	now := time.Now()
	result := h.db.DB.Model(&models.DecisionCard{}).
		Where("patient_id = ? AND status = ?", patientID, models.StatusActive).
		Updates(map[string]interface{}{
			"status":       models.StatusSuperseded,
			"superseded_at": now,
			"superseded_by": card.CardID,
			"updated_at":    now,
		})
	if result.Error != nil {
		h.log.Error("failed to pause active cards on hospitalisation",
			zap.String("patient_id", env.PatientID),
			zap.Error(result.Error),
		)
	} else {
		h.log.Info("paused active protocol cards on hospitalisation",
			zap.String("patient_id", env.PatientID),
			zap.Int64("cards_paused", result.RowsAffected),
		)
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save hospitalisation card: %w", err)
	}

	if err := h.gateCache.WriteGate(card); err != nil {
		h.log.Error("gate cache write failed for hospitalisation alert", zap.Error(err))
	}

	go h.kb19.PublishGateChanged(card)

	h.log.Warn("priority hospitalisation card generated",
		zap.String("card_id", card.CardID.String()),
		zap.String("patient_id", env.PatientID),
		zap.String("reason", payload.Reason),
	)
	return nil
}
