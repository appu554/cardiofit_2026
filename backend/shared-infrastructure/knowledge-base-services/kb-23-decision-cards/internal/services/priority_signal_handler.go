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
	db                   *database.Database
	gateCache            *MCUGateCache
	kb19                 *KB19Publisher
	hypoHandler          *HypoglycaemiaHandler
	mandatoryMedChecker  *MandatoryMedChecker // Phase 6 P6-6
	kb20Client           *KB20Client          // Phase 6 P6-6 + P6-2
	renalDoseGate        *RenalDoseGate       // Phase 6 P6-2
	metrics              *metrics.Collector
	log                  *zap.Logger
}

// NewPrioritySignalHandler creates a new handler for priority signals.
// mandatoryMedChecker, kb20Client, and renalDoseGate are optional — pass
// nil to disable the corresponding Phase 6 handlers (e.g., in unit tests
// for other routes).
func NewPrioritySignalHandler(
	db *database.Database,
	gateCache *MCUGateCache,
	kb19 *KB19Publisher,
	hypoHandler *HypoglycaemiaHandler,
	mandatoryMedChecker *MandatoryMedChecker,
	kb20Client *KB20Client,
	renalDoseGate *RenalDoseGate,
	m *metrics.Collector,
	log *zap.Logger,
) *PrioritySignalHandler {
	return &PrioritySignalHandler{
		db:                  db,
		gateCache:           gateCache,
		kb19:                kb19,
		hypoHandler:         hypoHandler,
		mandatoryMedChecker: mandatoryMedChecker,
		kb20Client:          kb20Client,
		renalDoseGate:       renalDoseGate,
		metrics:             m,
		log:                 log,
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
	case RouteCKMTransition:
		return h.handleCKMTransition(ctx, env)
	case RouteRenalGate:
		return h.handleRenalGate(ctx, env)
	default:
		return nil
	}
}

// handleRenalGate processes a new derived eGFR lab event and runs the
// reactive renal dose gate against the patient's active medications.
// Phase 6 P6-2: closes the loop where a new creatinine result is the
// only thing standing between a metformin patient and an undetected
// contraindication when their eGFR drops below 30.
//
// Scope note — like P6-6, this handler currently logs detected gating
// gaps rather than persisting them as DecisionCards. The DecisionCard
// model is template-driven (TemplateID, NodeID, ClinicianSummary,
// SafetyCheckSummary) and building real cards needs new
// RENAL_CONTRAINDICATION + RENAL_DOSE_REDUCE YAML templates plus the
// full card-builder pipeline (~1d follow-up). The Decision 9 abstraction
// proof — KB-20 publishes EGFR_LAB → KB-23 routes → handler runs gate
// → contraindications detected — ships fully. Card persistence is the
// missing last mile alongside the Prometheus metrics
// renal_contraindication_detected_total{drug_class} and
// renal_reactive_latency_seconds.
func (h *PrioritySignalHandler) handleRenalGate(ctx context.Context, env priorityEnvelope) error {
	var payload struct {
		LabType    string  `json:"lab_type"`
		Value      float64 `json:"value"`
		Unit       string  `json:"unit"`
		MeasuredAt string  `json:"measured_at"`
		Source     string  `json:"source"`
		IsDerived  bool    `json:"is_derived"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal eGFR lab payload: %w", err)
	}

	if payload.LabType != "EGFR" {
		// Defensive: the router already filters on signal_type=EGFR_LAB,
		// but keep this check so a payload with an unexpected lab_type
		// can't accidentally trigger gating.
		h.log.Debug("renal gate handler received non-EGFR payload",
			zap.String("lab_type", payload.LabType))
		return nil
	}

	if h.renalDoseGate == nil || h.kb20Client == nil {
		h.log.Warn("renal gate event received but RenalDoseGate or KB20Client not wired",
			zap.String("patient_id", env.PatientID))
		return nil
	}

	patientCtx, err := h.kb20Client.FetchSummaryContext(ctx, env.PatientID)
	if err != nil {
		return fmt.Errorf("fetch KB-20 patient context for renal gate: %w", err)
	}
	if patientCtx == nil {
		h.log.Warn("KB-20 returned nil patient context for renal gate",
			zap.String("patient_id", env.PatientID))
		return nil
	}

	// Construct ActiveMedication slice from the string medication list
	// in PatientContext. Drug names + doses aren't carried here, so the
	// gate evaluates only by drug class — sufficient for contraindication
	// detection (the formulary keys on drug class). The dose-reduce path
	// will need richer medication data in a Phase 6 follow-up.
	meds := make([]ActiveMedication, 0, len(patientCtx.Medications))
	for _, drugClass := range patientCtx.Medications {
		meds = append(meds, ActiveMedication{DrugClass: drugClass})
	}

	measuredAt := time.Now().UTC()
	if payload.MeasuredAt != "" {
		if t, parseErr := time.Parse(time.RFC3339, payload.MeasuredAt); parseErr == nil {
			measuredAt = t
		}
	}

	rs := models.RenalStatus{
		EGFR:           payload.Value,
		EGFRMeasuredAt: measuredAt,
	}
	report := h.renalDoseGate.EvaluatePatient(env.PatientID, rs, meds)

	if !report.HasContraindicated && !report.HasDoseReduce {
		h.log.Info("renal gate: no contraindications or dose-reduce gaps",
			zap.String("patient_id", env.PatientID),
			zap.Float64("egfr", payload.Value),
			zap.Int("med_count", len(meds)))
		return nil
	}

	contraindicated := make([]string, 0)
	doseReduce := make([]string, 0)
	for _, r := range report.MedicationResults {
		switch r.Verdict {
		case models.VerdictContraindicated:
			contraindicated = append(contraindicated, r.DrugClass)
		case models.VerdictDoseReduce:
			doseReduce = append(doseReduce, r.DrugClass)
		}
	}

	h.log.Info("renal gate: gating gaps detected",
		zap.String("patient_id", env.PatientID),
		zap.Float64("egfr", payload.Value),
		zap.String("urgency", report.OverallUrgency),
		zap.Strings("contraindicated", contraindicated),
		zap.Strings("dose_reduce", doseReduce),
		zap.String("note", "card persistence pending Phase 6 follow-up — needs RENAL_CONTRAINDICATION + RENAL_DOSE_REDUCE YAML templates + card builder wiring"),
	)
	return nil
}

// handleCKMTransition processes a CKM_STAGE_TRANSITION event published by
// KB-20's outbox relay. Phase 6 P6-6 (Decision 9): on a transition where
// to_stage="4c", invoke MandatoryMedChecker for GDMT gap detection. Other
// transitions (3a→3b, 4a→4b, etc.) are visible on the topic for downstream
// consumers (dashboards, audit) but trigger no KB-23 action — only 4c
// needs the mandatory-med check.
//
// Scope note — this handler currently logs detected gaps rather than
// persisting them as DecisionCard rows. The KB-23 DecisionCard model is
// template-driven (TemplateID, NodeID, ClinicianSummary, PatientSummaryEn,
// SafetyCheckSummary, etc.) and building real cards requires a new
// "CKM_4C_MANDATORY_MEDICATION" YAML template plus the full card-builder
// pipeline (~1d follow-up). The Decision 9 abstraction proof — KB-20
// publishes → KB-23 routes → handler filters for 4c → MandatoryMedChecker
// invoked → gaps detected — ships fully. Card persistence is the missing
// last mile and lands as a Phase 6 follow-up alongside the metrics.
func (h *PrioritySignalHandler) handleCKMTransition(ctx context.Context, env priorityEnvelope) error {
	var payload struct {
		FromStage string `json:"from_stage"`
		ToStage   string `json:"to_stage"`
		HFType    string `json:"hf_type"`
		Rationale string `json:"staging_rationale"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal CKM transition payload: %w", err)
	}

	if payload.ToStage != "4c" {
		// Non-4c transitions are routed but produce no KB-23 action.
		// Logged at debug level so the routing remains observable
		// without spamming logs at the default level.
		h.log.Debug("CKM stage transition routed but to_stage != 4c (no-op)",
			zap.String("patient_id", env.PatientID),
			zap.String("from_stage", payload.FromStage),
			zap.String("to_stage", payload.ToStage))
		return nil
	}

	if h.mandatoryMedChecker == nil || h.kb20Client == nil {
		// Defensive default: if the dependencies aren't wired (e.g.
		// in a test harness), log and skip without erroring. Production
		// must wire both via NewPrioritySignalHandler.
		h.log.Warn("CKM 4c transition received but MandatoryMedChecker or KB20Client not wired",
			zap.String("patient_id", env.PatientID))
		return nil
	}

	patientCtx, err := h.kb20Client.FetchSummaryContext(ctx, env.PatientID)
	if err != nil {
		return fmt.Errorf("fetch KB-20 patient context for CKM 4c: %w", err)
	}
	if patientCtx == nil {
		h.log.Warn("KB-20 returned nil patient context for CKM 4c",
			zap.String("patient_id", env.PatientID))
		return nil
	}

	gaps := h.mandatoryMedChecker.CheckMandatory("4c", payload.HFType, patientCtx.Medications)
	if len(gaps) == 0 {
		h.log.Info("CKM 4c transition: no mandatory medication gaps",
			zap.String("patient_id", env.PatientID),
			zap.String("from_stage", payload.FromStage))
		return nil
	}

	missing := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		missing = append(missing, gap.MissingClass)
	}

	h.log.Info("CKM 4c transition: mandatory medication gaps detected",
		zap.String("patient_id", env.PatientID),
		zap.String("from_stage", payload.FromStage),
		zap.String("hf_type", payload.HFType),
		zap.Int("gaps_found", len(gaps)),
		zap.Strings("missing_classes", missing),
		zap.String("note", "card persistence pending Phase 6 follow-up — needs CKM_4C_MANDATORY_MEDICATION YAML template + card builder wiring"),
	)
	return nil
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
		Urgency:                safetyTierToUrgency(safetyTier),
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

// safetyTierToUrgency maps a SafetyTier to the corresponding Urgency value.
// This avoids an unsafe direct type cast between two independent string enums.
func safetyTierToUrgency(tier models.SafetyTier) models.Urgency {
	switch tier {
	case models.SafetyImmediate:
		return models.UrgencyImmediate
	case models.SafetyUrgent:
		return models.UrgencyUrgent
	case models.SafetyRoutine:
		return models.UrgencyRoutine
	default:
		return models.UrgencyUrgent
	}
}
