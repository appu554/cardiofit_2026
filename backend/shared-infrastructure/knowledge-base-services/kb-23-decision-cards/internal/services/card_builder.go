package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/models"
)

// CardBuilder assembles a DecisionCard from a template, HPI event, and
// patient context. It orchestrates confidence tier computation, MCU gate
// evaluation, recommendation composition, and fragment-based summaries.
//
// Implements:
//   - N-05: dose_adjustment_notes required for MODIFY gate.
//   - A-02: confidence_tier_decayed placeholder (enriched in Phase 3).
//   - A-05: observation_reliability via perturbation check (Phase 3).
//   - N-01: hysteresis filtering on MCU gate transitions (Phase 3).
//   - A-01: perturbation-based observation reliability downgrade (Phase 3).
//   - A-04: KB-21 adherence gain factor enrichment (Phase 3).
type CardBuilder struct {
	confidenceTier    *ConfidenceTierService
	mcuGateManager    *MCUGateManager
	recommendComposer *RecommendationComposer
	fragmentLoader    *FragmentLoader
	db                *database.Database
	log               *zap.Logger

	// Phase 3 enrichment dependencies
	hysteresis    *HysteresisEngine
	perturbations *PerturbationService
	kb21Client    *KB21Client
	gateCache     *MCUGateCache
}

// NewCardBuilder creates a CardBuilder with all required service dependencies.
// Phase 3 enrichment services (hysteresis, perturbations, kb21Client, gateCache)
// are optional -- pass nil to disable the corresponding enrichment step.
func NewCardBuilder(
	ct *ConfidenceTierService,
	gm *MCUGateManager,
	rc *RecommendationComposer,
	fl *FragmentLoader,
	db *database.Database,
	log *zap.Logger,
	hysteresis *HysteresisEngine,
	perturbations *PerturbationService,
	kb21Client *KB21Client,
	gateCache *MCUGateCache,
) *CardBuilder {
	return &CardBuilder{
		confidenceTier:    ct,
		mcuGateManager:    gm,
		recommendComposer: rc,
		fragmentLoader:    fl,
		db:                db,
		log:               log,
		hysteresis:        hysteresis,
		perturbations:     perturbations,
		kb21Client:        kb21Client,
		gateCache:         gateCache,
	}
}

// Build creates a DecisionCard from the given template, HPI event, and
// patient context. The card and its recommendations are persisted to the
// database within this call.
func (b *CardBuilder) Build(
	ctx context.Context,
	tmpl *models.CardTemplate,
	event *models.HPICompleteEvent,
	patientCtx *PatientContext,
) (*models.DecisionCard, error) {
	// 1. Compute confidence tier (V-01)
	tier := b.confidenceTier.ComputeTier(event.TopPosterior, tmpl)
	isFirmMedChange := b.confidenceTier.IsFirmForMedicationChange(event.TopPosterior, tmpl)

	// 2. Evaluate MCU_GATE (V-06, N-05)
	gate, rationale, adjustmentNotes := b.mcuGateManager.EvaluateGate(tmpl, tier, patientCtx)

	// Phase 3 enrichment: applied between gate evaluation and recommendation composition

	// 2a. A-01: Perturbation enrichment -- active perturbations downgrade observation reliability
	observationReliability := models.ReliabilityHigh
	if b.perturbations != nil {
		if activePerturbations, err := b.perturbations.GetActive(ctx, event.PatientID.String()); err == nil && len(activePerturbations) > 0 {
			observationReliability = models.ReliabilityModerate
			b.log.Debug("A-01: observation reliability downgraded due to active perturbations",
				zap.String("patient_id", event.PatientID.String()),
				zap.Int("active_perturbation_count", len(activePerturbations)),
			)
		}
	}

	// 2b. N-01: Hysteresis filtering -- prevent rapid gate oscillation
	if b.hysteresis != nil && b.gateCache != nil {
		currentGate := models.GateSafe
		if cached, err := b.gateCache.ReadGate(event.PatientID.String()); err == nil {
			currentGate = cached.MCUGate
		}
		effectiveGate, hystRationale := b.hysteresis.Apply(event.PatientID, currentGate, gate)
		if hystRationale != "" {
			rationale = rationale + "; " + hystRationale
		}
		gate = effectiveGate
	}

	// 2c. A-04: KB-21 adherence enrichment -- fetch gain factor for gate cache
	adherenceGain := 1.0
	if b.kb21Client != nil {
		if adherence, err := b.kb21Client.FetchAdherence(ctx, event.PatientID.String()); err == nil && adherence != nil {
			adherenceGain = adherence.GainFactor
			b.log.Debug("A-04: KB-21 adherence enriched",
				zap.String("patient_id", event.PatientID.String()),
				zap.Float64("gain_factor", adherenceGain),
			)
		}
	}

	// 3. Compose recommendations (V-01, V-04, V-05)
	recommendations := b.recommendComposer.Compose(tmpl, tier, isFirmMedChange)

	// 4. Determine safety tier from safety flags
	safetyTier := b.determineSafetyTier(event.SafetyFlags)

	// 5. Build fragments for summaries
	clinicianSummary := b.buildClinicianSummary(tmpl, event)
	patientSummaryEn, patientSummaryHi := b.buildPatientSummaries(tmpl)

	// 6. Assemble the card
	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                event.PatientID,
		SessionID:                &event.SessionID,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    event.TopDiagnosis,
		PrimaryPosterior:         event.TopPosterior,
		DiagnosticConfidenceTier: tier,
		MCUGate:                  gate,
		MCUGateRationale:         rationale,
		ObservationReliability:   observationReliability, // A-05: enriched by A-01 perturbation check
		ClinicianSummary:         clinicianSummary,
		PatientSummaryEn:         patientSummaryEn,
		PatientSummaryHi:         patientSummaryHi,
		SafetyTier:               safetyTier,
		CardSource:               models.SourceKB22Session,
		Status:                   models.StatusActive,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// N-05: dose_adjustment_notes required for MODIFY
	if adjustmentNotes != "" {
		card.DoseAdjustmentNotes = &adjustmentNotes
	}

	// A-04: store adherence gain factor on transient field for gate cache writer
	card.AdherenceGainFactor = adherenceGain

	// CTL Panel 1: Structured patient state snapshot
	card.PatientStateSnapshot = b.buildPatientStateSnapshot(patientCtx)

	// CTL Panel 2: Overall guideline condition status from recommendations
	card.GuidelineConditionStatus = b.evaluateGuidelineConditions(recommendations)

	// CTL Panel 3: Safety check summary
	card.SafetyCheckSummary = b.buildSafetyCheckSummary(gate, rationale, observationReliability, adjustmentNotes, event.SafetyFlags)

	// CTL Panel 4: Reasoning chain (pass-through from KB-22 via HPI event)
	if len(event.ReasoningChain) > 0 {
		card.ReasoningChain = models.JSONB(event.ReasoningChain)
	}

	// 7. Save card to database
	if err := b.db.DB.WithContext(ctx).Create(card).Error; err != nil {
		return nil, err
	}

	// 8. Save recommendations with card_id reference
	for i := range recommendations {
		recommendations[i].CardID = card.CardID
	}
	if len(recommendations) > 0 {
		if err := b.db.DB.WithContext(ctx).Create(&recommendations).Error; err != nil {
			b.log.Error("failed to save recommendations", zap.Error(err))
		}
	}
	card.Recommendations = recommendations

	b.log.Info("decision card created",
		zap.String("card_id", card.CardID.String()),
		zap.String("template_id", card.TemplateID),
		zap.String("tier", string(card.DiagnosticConfidenceTier)),
		zap.String("gate", string(card.MCUGate)),
		zap.Int("recommendations", len(recommendations)),
	)

	return card, nil
}

// determineSafetyTier selects the highest safety tier from the event's
// safety flags. IMMEDIATE takes precedence over URGENT, which takes
// precedence over ROUTINE.
func (b *CardBuilder) determineSafetyTier(flags []models.SafetyFlagEntry) models.SafetyTier {
	for _, flag := range flags {
		if flag.Severity == "IMMEDIATE" {
			return models.SafetyImmediate
		}
	}
	for _, flag := range flags {
		if flag.Severity == "URGENT" {
			return models.SafetyUrgent
		}
	}
	return models.SafetyRoutine
}

// buildClinicianSummary returns the clinician-facing summary text from the
// template's fragments. Falls back to a generated summary if no clinician
// fragment is found.
func (b *CardBuilder) buildClinicianSummary(tmpl *models.CardTemplate, event *models.HPICompleteEvent) string {
	fragments := b.fragmentLoader.GetByTemplate(tmpl.TemplateID)
	for _, frag := range fragments {
		if frag.FragmentType == models.FragClinician {
			return frag.TextEn
		}
	}
	return "Decision card generated for " + event.TopDiagnosis
}

// buildPatientStateSnapshot assembles CTL Panel 1 from the PatientContext
// already fetched from KB-20 during gate evaluation.
func (b *CardBuilder) buildPatientStateSnapshot(ctx *PatientContext) models.JSONB {
	if ctx == nil {
		return nil
	}
	entry := models.PatientStateEntry{
		Stratum:     ctx.Stratum,
		EGFRValue:   ctx.EGFRValue,
		LatestHbA1c: ctx.LatestHbA1c,
		LatestFBG:   ctx.LatestFBG,
		WeightKg:    ctx.WeightKg,
		Medications: ctx.Medications,
		IsAcuteIll:  ctx.IsAcuteIll,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		b.log.Error("failed to marshal patient state snapshot", zap.Error(err))
		return nil
	}
	return models.JSONB(data)
}

// evaluateGuidelineConditions derives the overall guideline condition status
// for CTL Panel 2. It returns the "worst" status across all recommendations:
// any CRITERIA_NOT_MET → NOT_MET; any CRITERIA_PARTIAL → PARTIAL; else MET.
func (b *CardBuilder) evaluateGuidelineConditions(recommendations []models.CardRecommendation) *models.ConditionStatus {
	if len(recommendations) == 0 {
		return nil
	}

	overall := models.ConditionMet
	for _, rec := range recommendations {
		if rec.ConditionStatus == nil {
			continue
		}
		switch *rec.ConditionStatus {
		case models.ConditionNotMet:
			overall = models.ConditionNotMet
			result := overall
			return &result
		case models.ConditionPartial:
			overall = models.ConditionPartial
		}
	}

	result := overall
	return &result
}

// buildSafetyCheckSummary assembles CTL Panel 3 from the gate evaluation
// results and safety flags already computed in the Build flow.
func (b *CardBuilder) buildSafetyCheckSummary(
	gate models.MCUGate,
	rationale string,
	reliability models.ObservationReliability,
	adjustmentNotes string,
	flags []models.SafetyFlagEntry,
) models.JSONB {
	entry := models.SafetyCheckEntry{
		CheckType:              "MCU_GATE_EVALUATION",
		Gate:                   gate,
		GateRationale:          rationale,
		ObservationReliability: reliability,
		SafetyFlags:            flags,
		StressHyperglycaemia:   gate == models.GatePause && rationale == "V-06: stress hyperglycaemia -- acute illness, medication intensification paused",
		DoseAdjustmentNotes:    adjustmentNotes,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		b.log.Error("failed to marshal safety check summary", zap.Error(err))
		return nil
	}
	return models.JSONB(data)
}

// ---------------------------------------------------------------------------
// AD-07: Re-Escalation Pathway
// ---------------------------------------------------------------------------

// ReEscalationSpec defines the restart behaviour when a deprescribing
// step-down fails. Each drug class has specific re-escalation rules that
// balance safety (e.g. beta-blocker rebound tachycardia) with clinical
// judgement (e.g. ACEi/ARB always restore full RAAS protection).
type ReEscalationSpec struct {
	DrugClass     string `json:"drug_class"`
	RestartDose   string `json:"restart_dose"`
	CardNote      string `json:"card_note"`
	ReassessWeeks int    `json:"reassess_weeks"`
}

// GetReEscalationSpec returns the drug-class-specific re-escalation plan
// when a deprescribing step-down fails.
//
// failedAtPhase indicates which step failed:
//   - "DOSE_REDUCTION" → Step 1 failed at half-dose
//   - "REMOVAL"        → Step 2 failed after full removal
func GetReEscalationSpec(drugClass string, failedAtPhase string) ReEscalationSpec {
	switch drugClass {
	case "THIAZIDE":
		if failedAtPhase == "REMOVAL" {
			return ReEscalationSpec{
				DrugClass:     drugClass,
				RestartDose:   "Restart at 12.5 mg (not full dose)",
				CardNote:      "Restart at lowest effective dose. Reassess in 90 days.",
				ReassessWeeks: 13, // ~90 days
			}
		}
		return ReEscalationSpec{
			DrugClass:     drugClass,
			RestartDose:   "Restore full dose",
			CardNote:      "Not a candidate for dose reduction now. Re-attempt in 6 months.",
			ReassessWeeks: 26, // ~6 months
		}

	case "CCB":
		if failedAtPhase == "REMOVAL" {
			return ReEscalationSpec{
				DrugClass:     drugClass,
				RestartDose:   "Restart at lowest effective dose",
				CardNote:      "Surface oedema-vs-BP trade-off if applicable. Reassess in 90 days.",
				ReassessWeeks: 13,
			}
		}
		return ReEscalationSpec{
			DrugClass:     drugClass,
			RestartDose:   "Restore full dose",
			CardNote:      "Not a candidate for dose reduction now. Re-attempt in 6 months.",
			ReassessWeeks: 26,
		}

	case "BETA_BLOCKER":
		return ReEscalationSpec{
			DrugClass:     drugClass,
			RestartDose:   "Restart at half-dose, taper up",
			CardNote:      "Rebound tachycardia risk — restart at lower dose, taper upward over 2 weeks.",
			ReassessWeeks: 26,
		}

	case "ACE_INHIBITOR", "ARB":
		return ReEscalationSpec{
			DrugClass:     drugClass,
			RestartDose:   "Restore full RAAS dose",
			CardNote:      "ACR worsening overrides stable BP — restore full RAAS dose. Recheck ACR in 6 weeks.",
			ReassessWeeks: 6,
		}

	default:
		return ReEscalationSpec{
			DrugClass:     drugClass,
			RestartDose:   "Restore previous dose",
			CardNote:      "Step-down failed. Restore previous dose and reassess in 6 months.",
			ReassessWeeks: 26,
		}
	}
}

// ---------------------------------------------------------------------------
// Wave 3.2 / Amendment 10: Chronotherapy — Timing-Before-Escalation
// ---------------------------------------------------------------------------

// DoseTiming mirrors the KB-20 DoseTiming enum for use in KB-23 card logic.
type DoseTiming = string

const (
	DoseTimingMorning DoseTiming = "MORNING"
	DoseTimingBedtime DoseTiming = "BEDTIME"
)

// ShouldSuggestChronotherapy checks if a timing change should be recommended
// before dose escalation. Returns true if the patient has MORNING_SURGE BP pattern
// and the relevant medication is taken in the MORNING.
func ShouldSuggestChronotherapy(bpPattern string, currentTiming DoseTiming) bool {
	return bpPattern == "MORNING_SURGE" && currentTiming == "MORNING"
}

// AppendChronotherapyFragment adds a chronotherapy recommendation to the card
// when ShouldSuggestChronotherapy returns true. The fragment advises switching
// dose timing to bedtime before increasing dose.
func AppendChronotherapyFragment(card *models.DecisionCard, bpPattern string, currentTiming DoseTiming) {
	if !ShouldSuggestChronotherapy(bpPattern, currentTiming) {
		return
	}
	enNote := "Consider switching dose timing to bedtime before increasing dose. Morning surge pattern detected with morning dosing."
	hiNote := "खुराक बढ़ाने से पहले रात को खुराक लेने पर विचार करें। सुबह की खुराक के साथ सुबह का रक्तचाप बढ़ने का पैटर्न पाया गया।"

	card.PatientSummaryEn = card.PatientSummaryEn + "\n\n" + enNote
	card.PatientSummaryHi = card.PatientSummaryHi + "\n\n" + hiNote
	card.ClinicianSummary = card.ClinicianSummary + "\n[Chronotherapy] " + enNote
}

// ---------------------------------------------------------------------------
// BP Variability Annotation
// ---------------------------------------------------------------------------

// AddVariabilityNote appends a variability warning to the card when BP variability is HIGH.
func AddVariabilityNote(card *models.DecisionCard, variabilityStatus string, sbpSD float64) {
	if variabilityStatus != "HIGH" {
		return
	}
	enNote := fmt.Sprintf("High visit-to-visit BP variability (SD: %.1f mmHg). Consider home BP monitoring and assessing white-coat or masked hypertension.", sbpSD)
	hiNote := fmt.Sprintf("रक्तचाप में अधिक परिवर्तनशीलता (SD: %.1f mmHg)। घर पर बीपी मॉनिटरिंग और व्हाइट-कोट या मास्क्ड हाइपरटेंशन का मूल्यांकन करें।", sbpSD)

	card.ClinicianSummary = card.ClinicianSummary + "\n[BP Variability] " + enNote
	card.PatientSummaryEn = card.PatientSummaryEn + "\n\n" + enNote
	card.PatientSummaryHi = card.PatientSummaryHi + "\n\n" + hiNote
}

// ---------------------------------------------------------------------------
// Pulse Pressure Annotation
// ---------------------------------------------------------------------------

// AddPulsePressureNote appends an arterial stiffness warning when PP > 60 mmHg.
func AddPulsePressureNote(card *models.DecisionCard, pulsePressureMean float64, trend string) {
	if pulsePressureMean <= 60 {
		return
	}
	enNote := fmt.Sprintf("Wide pulse pressure (%.0f mmHg, trend: %s). Further SBP lowering may reduce DBP below safe threshold. Consider arterial stiffness assessment.", pulsePressureMean, trend)
	hiNote := fmt.Sprintf("अधिक पल्स प्रेशर (%.0f mmHg, रुझान: %s)। SBP और कम करने से DBP सुरक्षित सीमा से नीचे जा सकता है। धमनी कठोरता का मूल्यांकन करें।", pulsePressureMean, trend)

	card.ClinicianSummary = card.ClinicianSummary + "\n[Pulse Pressure] " + enNote
	card.PatientSummaryEn = card.PatientSummaryEn + "\n\n" + enNote
	card.PatientSummaryHi = card.PatientSummaryHi + "\n\n" + hiNote
}

// ---------------------------------------------------------------------------
// Salt Sensitivity — Lifestyle-First Sequencing
// ---------------------------------------------------------------------------

// ShouldPrioritizeDietaryIntervention checks if dietary sodium reduction should
// be recommended before dose escalation.
// Returns true if sodium is HIGH and reduction potential is HIGH (>= 0.6).
func ShouldPrioritizeDietaryIntervention(sodiumEstimate string, reductionPotential float64) bool {
	return sodiumEstimate == "HIGH" && reductionPotential >= 0.6
}

// AppendDietaryInterventionFragment adds a LIFESTYLE_INTERVENTION recommendation
// to the card when ShouldPrioritizeDietaryIntervention returns true, advising
// dietary sodium reduction before dose escalation.
func AppendDietaryInterventionFragment(card *models.DecisionCard, sodiumEstimate string, reductionPotential float64) {
	if !ShouldPrioritizeDietaryIntervention(sodiumEstimate, reductionPotential) {
		return
	}
	enNote := "High dietary sodium with significant reduction potential. Recommend dietary sodium reduction before considering dose increase. Focus: reduce pickles/papads, avoid post-cooking salt, limit processed foods."
	hiNote := "आहार में अधिक नमक और कम करने की अच्छी संभावना। खुराक बढ़ाने से पहले आहार में नमक कम करने की सिफारिश। ध्यान दें: अचार/पापड़ कम करें, पका हुआ खाना खाने से पहले ऊपर से नमक न डालें, प्रोसेस्ड फूड सीमित करें।"

	card.PatientSummaryEn = card.PatientSummaryEn + "\n\n" + enNote
	card.PatientSummaryHi = card.PatientSummaryHi + "\n\n" + hiNote
	card.ClinicianSummary = card.ClinicianSummary + "\n[Lifestyle-First] " + enNote
}

// ---------------------------------------------------------------------------
// Wave 2 Track E: Thiazide K+ Causal Context Annotation
// ---------------------------------------------------------------------------

// AddThiazideKPlusCausalContext annotates the card when thiazide-induced
// hypokalemia is the likely cause of low K+.
func AddThiazideKPlusCausalContext(card *models.DecisionCard, thiazideActive bool, potassiumLevel *float64) {
	if !thiazideActive || potassiumLevel == nil || *potassiumLevel >= 3.5 {
		return
	}
	enNote := fmt.Sprintf("Low potassium (%.1f mEq/L) likely thiazide-induced. Consider potassium supplementation or switch to potassium-sparing combination before attributing to other causes.", *potassiumLevel)
	hiNote := fmt.Sprintf("कम पोटैशियम (%.1f mEq/L) संभवतः थायज़ाइड के कारण। अन्य कारणों को मानने से पहले पोटैशियम सप्लीमेंट या पोटैशियम-स्पेयरिंग संयोजन पर विचार करें।", *potassiumLevel)

	card.ClinicianSummary = card.ClinicianSummary + "\n[Thiazide K+] " + enNote
	card.PatientSummaryEn = card.PatientSummaryEn + "\n\n" + enNote
	card.PatientSummaryHi = card.PatientSummaryHi + "\n\n" + hiNote
}

// buildPatientSummaries returns English and Hindi patient-facing summary
// text from the template's fragments.
func (b *CardBuilder) buildPatientSummaries(tmpl *models.CardTemplate) (string, string) {
	fragments := b.fragmentLoader.GetByTemplate(tmpl.TemplateID)
	var en, hi string
	for _, frag := range fragments {
		if frag.FragmentType == models.FragPatient {
			en = frag.TextEn
			hi = frag.TextHi
			break
		}
	}
	return en, hi
}
