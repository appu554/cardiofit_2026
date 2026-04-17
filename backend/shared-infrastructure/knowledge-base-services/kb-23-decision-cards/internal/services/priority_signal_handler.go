package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	texttemplate "text/template"
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
	templateLoader       *TemplateLoader      // Phase 7 P7-A
	fhirNotifier         FHIRCardNotifier     // Phase 10 Gap 9: FHIR outbound
	metrics              *metrics.Collector
	log                  *zap.Logger
}

// SetFHIRNotifier injects the FHIR notification client. Phase 10 Gap 9.
func (h *PrioritySignalHandler) SetFHIRNotifier(n FHIRCardNotifier) {
	h.fhirNotifier = n
}

// NewPrioritySignalHandler creates a new handler for priority signals.
// mandatoryMedChecker, kb20Client, renalDoseGate, and templateLoader are
// optional — pass nil to disable the corresponding handlers (e.g., in
// unit tests for other routes, or in bootstrap paths where a dependency
// is not yet wired).
func NewPrioritySignalHandler(
	db *database.Database,
	gateCache *MCUGateCache,
	kb19 *KB19Publisher,
	hypoHandler *HypoglycaemiaHandler,
	mandatoryMedChecker *MandatoryMedChecker,
	kb20Client *KB20Client,
	renalDoseGate *RenalDoseGate,
	templateLoader *TemplateLoader,
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
		templateLoader:      templateLoader,
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

// Phase 7 P7-A / P7-B: template IDs for priority-signal cards. Each
// constant must match the `template_id` key in its YAML file so the
// TemplateLoader can resolve it at card-build time.
const (
	renalContraindicationTemplateID = "dc-renal-contraindication-v1"
	renalDoseReduceTemplateID       = "dc-renal-dose-reduce-v1"
	ckm4cMandatoryMedTemplateID     = "dc-ckm-4c-mandatory-medication-v1"
)

// handleRenalGate processes a new derived eGFR lab event and runs the
// reactive renal dose gate against the patient's active medications.
// Phase 6 P6-2 closed the detection loop (KB-20 publishes EGFR_LAB →
// KB-23 routes → handler runs gate). Phase 7 P7-A closes the card
// persistence loop: detected gaps are now written as DecisionCards
// via the renal YAML templates, gate cache is updated, and the
// KB-19 gate-changed event is emitted. The end-to-end latency from
// signal receipt to persisted card is recorded in the
// kb23_renal_reactive_latency_seconds histogram.
func (h *PrioritySignalHandler) handleRenalGate(ctx context.Context, env priorityEnvelope) error {
	receivedAt := time.Now()

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
	// detection (the formulary keys on drug class).
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

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id for renal card: %w", err)
	}

	// Persist one card per detection category — contraindication first
	// (HALT, SafetyImmediate) then dose-reduce (MODIFY, SafetyUrgent).
	// The two cards are persisted independently so a template-loader
	// failure on one doesn't suppress the other.
	if len(contraindicated) > 0 {
		if err := h.persistRenalCard(
			patientID,
			renalContraindicationTemplateID,
			"RENAL_CONTRAINDICATION",
			models.GateHalt,
			models.SafetyImmediate,
			payload.Value,
			contraindicated,
		); err != nil {
			h.log.Error("failed to persist renal contraindication card",
				zap.String("patient_id", env.PatientID),
				zap.Error(err))
		} else if h.metrics != nil {
			for _, drugClass := range contraindicated {
				h.metrics.RenalContraindicationDetected.WithLabelValues(drugClass).Inc()
			}
		}
	}

	if len(doseReduce) > 0 {
		if err := h.persistRenalCard(
			patientID,
			renalDoseReduceTemplateID,
			"RENAL_DOSE_REDUCE",
			models.GateModify,
			models.SafetyUrgent,
			payload.Value,
			doseReduce,
		); err != nil {
			h.log.Error("failed to persist renal dose-reduce card",
				zap.String("patient_id", env.PatientID),
				zap.Error(err))
		} else if h.metrics != nil {
			for _, drugClass := range doseReduce {
				h.metrics.RenalDoseReduceDetected.WithLabelValues(drugClass).Inc()
			}
		}
	}

	if h.metrics != nil {
		h.metrics.RenalReactiveLatency.Observe(time.Since(receivedAt).Seconds())
	}

	h.log.Info("renal gate: cards persisted",
		zap.String("patient_id", env.PatientID),
		zap.Float64("egfr", payload.Value),
		zap.String("urgency", report.OverallUrgency),
		zap.Strings("contraindicated", contraindicated),
		zap.Strings("dose_reduce", doseReduce),
	)
	return nil
}

// persistRenalCard looks up the given template via the TemplateLoader,
// renders the clinician + patient summaries, assembles a DecisionCard,
// persists it, writes the gate cache, and emits the KB-19 gate-changed
// event. Returns an error if any of those steps fail; the caller
// records metrics only on success.
//
// When templateLoader, db, gateCache, or kb19 are nil (test harness or
// bootstrap mode where KB-23 runs without the full priority stack), the
// handler is defensive: it logs a warning and returns nil so a missing
// dependency can't panic a production pipeline.
func (h *PrioritySignalHandler) persistRenalCard(
	patientID uuid.UUID,
	templateID string,
	differentialID string,
	gate models.MCUGate,
	safetyTier models.SafetyTier,
	egfr float64,
	drugClasses []string,
) error {
	if h.templateLoader == nil || h.db == nil {
		h.log.Warn("renal card persistence skipped: templateLoader or db not wired",
			zap.String("template_id", templateID),
			zap.Int("drug_class_count", len(drugClasses)))
		return nil
	}

	tmpl, ok := h.templateLoader.Get(templateID)
	if !ok {
		return fmt.Errorf("template %s not loaded (check templates/renal/*.yaml)", templateID)
	}

	clinicianSummary, patientSummaryEn, patientSummaryHi := renderRenalSummaries(tmpl, egfr, drugClasses)
	notes := clinicianSummary

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    differentialID,
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  gate,
		MCUGateRationale:         clinicianSummary,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               safetyTier,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         patientSummaryEn,
		PatientSummaryHi:         patientSummaryHi,
		PendingReaffirmation:     gate == models.GateHalt,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save renal card: %w", err)
	}

	if h.gateCache != nil {
		if err := h.gateCache.WriteGate(card); err != nil {
			h.log.Error("gate cache write failed for renal card",
				zap.String("card_id", card.CardID.String()),
				zap.Error(err))
		}
	}
	if h.kb19 != nil {
		go h.kb19.PublishGateChanged(card)
	}
	notifyFHIR(h.fhirNotifier, card)

	return nil
}

// renderRenalSummaries picks the CLINICIAN and PATIENT fragments out of a
// loaded renal template and substitutes runtime values (eGFR, drug classes)
// into them via text/template. Pure function — exported for unit testing
// without a database or template loader.
func renderRenalSummaries(tmpl *models.CardTemplate, egfr float64, drugClasses []string) (clinician, patientEn, patientHi string) {
	data := struct {
		EGFR        string
		DrugClasses string
	}{
		EGFR:        fmt.Sprintf("%.1f", egfr),
		DrugClasses: strings.Join(drugClasses, ", "),
	}

	for _, frag := range tmpl.Fragments {
		switch frag.FragmentType {
		case models.FragClinician:
			clinician = executeRenalTemplate(frag.TextEn, data)
		case models.FragPatient:
			patientEn = executeRenalTemplate(frag.TextEn, data)
			patientHi = executeRenalTemplate(frag.TextHi, data)
		}
	}

	// Defensive fallback: if a template is somehow missing fragments,
	// synthesise a baseline clinician summary so the persisted card
	// still carries enough information to route the clinician to the
	// triggering eGFR and drug class list.
	if clinician == "" {
		clinician = fmt.Sprintf("Renal gating: eGFR %.1f mL/min/1.73m² — affected drug classes: %s",
			egfr, strings.Join(drugClasses, ", "))
	}
	return clinician, patientEn, patientHi
}

// executeRenalTemplate parses and executes a text/template snippet
// with the given data, returning the raw string on parse/execute
// failure so a malformed template never crashes the handler.
func executeRenalTemplate(raw string, data interface{}) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	tpl, err := texttemplate.New("renal").Parse(raw)
	if err != nil {
		return raw
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return raw
	}
	return strings.TrimSpace(buf.String())
}

// handleCKMTransition processes a CKM_STAGE_TRANSITION event published by
// KB-20's outbox relay. Phase 6 P6-6 landed the detection loop (KB-20
// publishes → KB-23 routes → handler filters for 4c → MandatoryMedChecker
// invoked → gaps detected). Phase 7 P7-B closes the card persistence
// loop via the CKM_4C_MANDATORY_MEDICATION YAML template and records
// kb23_ckm_stage_transitions_total / kb23_ckm_4c_mandatory_med_alerts_total.
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

	// Always-on audit counter: every transition observed on the priority
	// topic is tallied by from→to stage, regardless of whether KB-23 takes
	// action. This keeps the outbox → router → handler pipeline observable
	// even for non-4c transitions that are currently no-ops.
	if h.metrics != nil {
		fromLabel := payload.FromStage
		if fromLabel == "" {
			fromLabel = "unknown"
		}
		h.metrics.CKMStageTransitions.WithLabelValues(fromLabel, payload.ToStage).Inc()
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

	patientID, err := uuid.Parse(env.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id for CKM 4c card: %w", err)
	}

	if err := h.persistCKM4cCard(patientID, payload.FromStage, payload.HFType, missing); err != nil {
		h.log.Error("failed to persist CKM 4c mandatory-med card",
			zap.String("patient_id", env.PatientID),
			zap.Error(err))
		return nil
	}

	if h.metrics != nil {
		for _, drugClass := range missing {
			h.metrics.CKM4cMandatoryMedAlerts.WithLabelValues(drugClass).Inc()
		}
	}

	h.log.Info("CKM 4c transition: card persisted",
		zap.String("patient_id", env.PatientID),
		zap.String("from_stage", payload.FromStage),
		zap.String("hf_type", payload.HFType),
		zap.Int("gaps_found", len(gaps)),
		zap.Strings("missing_classes", missing),
	)
	return nil
}

// persistCKM4cCard looks up the CKM_4C_MANDATORY_MEDICATION template,
// renders fragments with {{.FromStage}} / {{.HFType}} / {{.MissingClasses}},
// assembles a DecisionCard, persists it, and emits the gate-changed
// event. Defensive when templateLoader / db / gateCache / kb19 are nil.
func (h *PrioritySignalHandler) persistCKM4cCard(
	patientID uuid.UUID,
	fromStage string,
	hfType string,
	missingClasses []string,
) error {
	if h.templateLoader == nil || h.db == nil {
		h.log.Warn("CKM 4c card persistence skipped: templateLoader or db not wired",
			zap.Int("missing_class_count", len(missingClasses)))
		return nil
	}

	tmpl, ok := h.templateLoader.Get(ckm4cMandatoryMedTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded (check templates/ckm/*.yaml)", ckm4cMandatoryMedTemplateID)
	}

	clinicianSummary, patientSummaryEn, patientSummaryHi := renderCKM4cSummaries(tmpl, fromStage, hfType, missingClasses)
	notes := clinicianSummary

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "CKM_4C_MANDATORY_MEDICATION",
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GateModify,
		MCUGateRationale:         clinicianSummary,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyUrgent,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         patientSummaryEn,
		PatientSummaryHi:         patientSummaryHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save CKM 4c card: %w", err)
	}

	if h.gateCache != nil {
		if err := h.gateCache.WriteGate(card); err != nil {
			h.log.Error("gate cache write failed for CKM 4c card",
				zap.String("card_id", card.CardID.String()),
				zap.Error(err))
		}
	}
	if h.kb19 != nil {
		go h.kb19.PublishGateChanged(card)
	}
	notifyFHIR(h.fhirNotifier, card)

	return nil
}

// renderCKM4cSummaries picks the CLINICIAN and PATIENT fragments out of
// the CKM 4c template and substitutes {{.FromStage}}, {{.HFType}}, and
// {{.MissingClasses}} into them. Pure function — exported for unit
// testing without a database or template loader.
func renderCKM4cSummaries(tmpl *models.CardTemplate, fromStage, hfType string, missingClasses []string) (clinician, patientEn, patientHi string) {
	data := struct {
		FromStage      string
		HFType         string
		MissingClasses string
	}{
		FromStage:      fromStage,
		HFType:         hfType,
		MissingClasses: strings.Join(missingClasses, ", "),
	}

	for _, frag := range tmpl.Fragments {
		switch frag.FragmentType {
		case models.FragClinician:
			clinician = executeRenalTemplate(frag.TextEn, data)
		case models.FragPatient:
			patientEn = executeRenalTemplate(frag.TextEn, data)
			patientHi = executeRenalTemplate(frag.TextHi, data)
		}
	}

	// Defensive fallback: synthesise a baseline clinician summary if
	// the template is somehow missing a CLINICIAN fragment.
	if clinician == "" {
		clinician = fmt.Sprintf("CKM 4c transition (%s→4c, %s): missing GDMT classes: %s",
			fromStage, hfType, strings.Join(missingClasses, ", "))
	}
	return clinician, patientEn, patientHi
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
	notifyFHIR(h.fhirNotifier, card)

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
	notifyFHIR(h.fhirNotifier, card)

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
	notifyFHIR(h.fhirNotifier, card)

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
	notifyFHIR(h.fhirNotifier, card)

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
