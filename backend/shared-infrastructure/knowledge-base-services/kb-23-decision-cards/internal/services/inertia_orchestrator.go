package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

// Phase 7 P7-D + Phase 9 P9-A + P9-F: template IDs for the inertia cards.
const (
	inertiaDetectedTemplateID           = "dc-inertia-detected-v1"
	dualDomainInertiaDetectedTemplateID = "dc-dual-domain-inertia-detected-v1"
	adherenceGapTemplateID              = "dc-adherence-gap-v1"
	deprescribingReviewTemplateID       = "dc-deprescribing-review-v1"
)

// Polypharmacy-elderly thresholds (Beers Criteria screen).
// A patient meeting BOTH criteria gets a DEPRESCRIBING_REVIEW card
// when inertia is detected, advising the clinician to consider
// reducing medication burden before adding another agent.
const (
	polypharmacyElderlyAgeThreshold = 75
	polypharmacyElderlyMedThreshold = 5
)

// IsPolypharmacyElderly returns true when the patient meets the
// conservative frailty proxy: age >= 75 AND >= 5 active medications.
// Pure function — exported for unit testing. Phase 9 P9-F.
func IsPolypharmacyElderly(age, medicationCount int) bool {
	return age >= polypharmacyElderlyAgeThreshold && medicationCount >= polypharmacyElderlyMedThreshold
}

// InertiaOrchestrator is the coordination layer that wraps DetectInertia,
// applies stability dampening against the previous week's verdict,
// persists the current verdict to history, and writes DecisionCards for
// every detected inertia pattern.
//
// Phase 6 P6-1 shipped this as a stateless pass-through that just ran
// DetectInertia and logged. Phase 7 P7-D now wires:
//
//   - InertiaVerdictHistory for previous-week lookup + dampening
//   - TemplateLoader + database for card persistence
//   - MCUGateCache + KB19Publisher for downstream gate propagation
//   - metrics.Collector for inertia counters
type InertiaOrchestrator struct {
	history           InertiaVerdictHistory
	templateLoader    *TemplateLoader
	db                *database.Database
	gateCache         *MCUGateCache
	kb19              *KB19Publisher
	fhirNotifier      FHIRCardNotifier
	escalationManager *EscalationManager
	metrics           *metrics.Collector
	log               *zap.Logger
}

// FHIRCardNotifier is the interface for sending FHIR CommunicationRequest
// notifications when a decision card is persisted. Implemented by KB20Client.
// Nil-safe: when unset, FHIR notification is skipped (local dev, tests).
type FHIRCardNotifier interface {
	NotifyCardGenerated(patientID, cardID, templateID, clinicianSummary, safetyTier, mcuGate string)
}

// notifyFHIR fires a FHIR webhook for a persisted card. Nil-safe, fire-and-forget.
func notifyFHIR(notifier FHIRCardNotifier, card *models.DecisionCard) {
	if notifier == nil || card == nil {
		return
	}
	go notifier.NotifyCardGenerated(
		card.PatientID.String(),
		card.CardID.String(),
		card.TemplateID,
		card.ClinicianSummary,
		string(card.SafetyTier),
		string(card.MCUGate),
	)
}

// NewInertiaOrchestrator constructs the orchestrator. All dependencies
// are optional for degraded / test modes:
//
//   - history nil → dampening disabled, every raw verdict is published
//   - templateLoader / db nil → card persistence disabled (log-only mode)
//   - gateCache / kb19 / metrics nil → corresponding side effects skipped
//
// Production main.go should wire everything.
func NewInertiaOrchestrator(
	history InertiaVerdictHistory,
	templateLoader *TemplateLoader,
	db *database.Database,
	gateCache *MCUGateCache,
	kb19 *KB19Publisher,
	m *metrics.Collector,
	log *zap.Logger,
) *InertiaOrchestrator {
	if log == nil {
		log = zap.NewNop()
	}
	return &InertiaOrchestrator{
		history:        history,
		templateLoader: templateLoader,
		db:             db,
		gateCache:      gateCache,
		kb19:           kb19,
		metrics:        m,
		log:            log,
	}
}

// SetFHIRNotifier injects the FHIR notification client after construction.
// Called from main.go once the KB20Client is instantiated. Phase 10 Gap 9.
func (o *InertiaOrchestrator) SetFHIRNotifier(n FHIRCardNotifier) {
	o.fhirNotifier = n
}

// SetEscalationManager injects the escalation manager after construction.
// Called from main.go once the EscalationManager is instantiated. Gap 15.
func (o *InertiaOrchestrator) SetEscalationManager(em *EscalationManager) {
	o.escalationManager = em
}

// Evaluate runs DetectInertia, applies stability dampening, persists
// verdicts to history, and writes one DecisionCard per detected verdict
// (plus a DUAL_DOMAIN_INERTIA_DETECTED card when two or more domains
// fire in the same week).
//
// Errors from card persistence are logged but do not abort the
// evaluation — the verdict history write still happens so the next
// week's dampening check sees the current verdict.
func (o *InertiaOrchestrator) Evaluate(ctx context.Context, input InertiaDetectorInput) models.PatientInertiaReport {
	// Phase 9 P9-A: adherence-exclusion gate. If the patient is
	// disengaged, skip DetectInertia entirely and produce an
	// ADHERENCE_GAP report instead. This prevents false-positive
	// inertia cards when the target gap is driven by non-adherence
	// (Patient 17 in the 20-patient thought experiment).
	//
	// Threshold: EngagementStatus == "DISENGAGED" OR
	// EngagementComposite != nil && *EngagementComposite < 0.4.
	// Nil EngagementComposite → no engagement data → assume
	// engaged (bias toward surfacing inertia, not suppressing).
	if isDisengaged(input) {
		return o.handleAdherenceGap(ctx, input)
	}

	// V4-7: phenotype stability gate. A patient stably classified as
	// STABLE_CONTROLLED for 4+ weeks (enforced by the KB-20 stability
	// engine's dwell gate) is in a well-managed cluster where "no
	// medication change" is appropriate maintenance, not inertia.
	// Suppressing inertia cards for these patients avoids false
	// positives that would confuse clinicians with contradictory
	// signals ("patient is well controlled" + "therapeutic inertia
	// detected").
	if isPhenotypeStableGood(input) {
		o.log.Debug("inertia: suppressed by stable-good phenotype",
			zap.String("patient_id", input.PatientID),
			zap.String("phenotype_cluster", input.PhenotypeCluster))
		if o.metrics != nil {
			o.metrics.InertiaSuppressedByPhenotype.Inc()
		}
		return models.PatientInertiaReport{
			PatientID:   input.PatientID,
			EvaluatedAt: time.Now(),
		}
	}

	report := DetectInertia(input)

	// Stability dampening: if the raw verdict differs from the previous
	// week's persisted verdict AND the patient's current target status
	// is unchanged, hold the previous verdict. This suppresses one-week
	// flip-flop patterns where the patient oscillates between "just
	// over target" and "just under target" without any clinical change.
	if o.history != nil {
		if prev, _, ok := o.history.FetchLatest(input.PatientID); ok {
			if shouldDampen(prev, report, input) {
				o.log.Debug("inertia: dampening to previous verdict",
					zap.String("patient_id", input.PatientID))
				report = prev
			}
		}
		weekStart := startOfWeek(time.Now())
		if err := o.history.SaveVerdict(input.PatientID, weekStart, report); err != nil {
			o.log.Warn("inertia: failed to save verdict history",
				zap.String("patient_id", input.PatientID),
				zap.Error(err))
		}
	}

	if len(report.Verdicts) == 0 {
		o.log.Debug("inertia: no verdicts",
			zap.String("patient_id", input.PatientID))
		return report
	}

	// Persist one DecisionCard per detected verdict, plus one
	// dual-domain card when two or more domains fire.
	detectedCount := 0
	for _, v := range report.Verdicts {
		if !v.Detected {
			continue
		}
		detectedCount++
		if err := o.persistInertiaCard(input.PatientID, v); err != nil {
			o.log.Error("inertia: failed to persist card",
				zap.String("patient_id", input.PatientID),
				zap.String("domain", string(v.Domain)),
				zap.Error(err))
		} else if o.metrics != nil {
			o.metrics.InertiaVerdictsDetected.WithLabelValues(string(v.Domain), string(v.Severity)).Inc()
		}
	}

	if detectedCount >= 2 {
		if err := o.persistDualDomainCard(input.PatientID, report); err != nil {
			o.log.Error("inertia: failed to persist dual-domain card",
				zap.String("patient_id", input.PatientID),
				zap.Error(err))
		} else if o.metrics != nil {
			o.metrics.DualDomainInertiaDetected.Inc()
		}
	}

	// Phase 9 P9-F: polypharmacy-elderly deprescribing review. When
	// the patient is >= 75 years old AND on >= 5 medications AND the
	// detector just generated inertia verdicts, add a
	// DEPRESCRIBING_REVIEW card that advises the clinician to
	// consider reducing medication burden before adding another agent.
	// This does NOT suppress the inertia verdicts — it adds a
	// parallel card that provides frailty context so the clinician
	// can make an informed decision between escalation and
	// deprescribing.
	if detectedCount > 0 && IsPolypharmacyElderly(input.Age, input.MedicationCount) {
		if err := o.persistDeprescribingCard(input.PatientID, input.Age, input.MedicationCount); err != nil {
			o.log.Error("inertia: failed to persist deprescribing card",
				zap.String("patient_id", input.PatientID),
				zap.Error(err))
		} else if o.metrics != nil {
			o.metrics.DeprescribingReviewGenerated.Inc()
		}
	}

	o.log.Info("inertia: verdicts detected",
		zap.String("patient_id", input.PatientID),
		zap.Int("verdict_count", detectedCount),
		zap.Bool("dual_domain", detectedCount >= 2),
		zap.Bool("deprescribing_review", detectedCount > 0 && IsPolypharmacyElderly(input.Age, input.MedicationCount)),
	)
	return report
}

// shouldDampen returns true when the current raw verdict differs from
// the previous week's persisted verdict AND the patient's underlying
// target status has not changed. This is the one-week flip-flop guard.
//
// The check is intentionally conservative: if the previous verdict had
// zero detected domains and the current verdict has ≥1 detected, we do
// NOT dampen — a new detection is always honoured. Dampening only
// suppresses oscillation between two previously-seen states.
func shouldDampen(prev, current models.PatientInertiaReport, input InertiaDetectorInput) bool {
	prevDetected := countDetected(prev)
	currDetected := countDetected(current)

	// Never suppress a new detection where the previous week had none.
	if prevDetected == 0 && currDetected > 0 {
		return false
	}
	// Never suppress a clearance either — if the previous week detected
	// and the current week does not, clearance is honoured.
	if prevDetected > 0 && currDetected == 0 {
		return false
	}
	// Both weeks have the same count — check if the detected domain set
	// flipped. If identical domains fired, no dampening needed.
	if currDetected == prevDetected && sameDetectedDomains(prev, current) {
		return false
	}
	// Different detected domain sets with same cardinality → flip-flop.
	// Dampen iff the patient's target status hasn't visibly changed
	// (simple proxy: none of the DaysUncontrolled counters moved
	// meaningfully since last week).
	if input.Glycaemic != nil && input.Glycaemic.DaysUncontrolled > 0 {
		// Any non-trivial uncontrolled window → allow the new verdict.
		return false
	}
	return true
}

// countDetected returns the number of verdicts in a report with Detected=true.
func countDetected(r models.PatientInertiaReport) int {
	n := 0
	for _, v := range r.Verdicts {
		if v.Detected {
			n++
		}
	}
	return n
}

// sameDetectedDomains returns true if the two reports have the same
// set of detected domains (ignoring severity + ordering).
func sameDetectedDomains(a, b models.PatientInertiaReport) bool {
	set := map[models.InertiaDomain]bool{}
	for _, v := range a.Verdicts {
		if v.Detected {
			set[v.Domain] = true
		}
	}
	for _, v := range b.Verdicts {
		if v.Detected {
			if !set[v.Domain] {
				return false
			}
			delete(set, v.Domain)
		}
	}
	return len(set) == 0
}

// startOfWeek returns the Monday 00:00 UTC of the week containing t.
// Used as the week-key for verdict history entries.
func startOfWeek(t time.Time) time.Time {
	weekday := int(t.UTC().Weekday())
	// Go's Weekday: Sunday=0, Monday=1, … Saturday=6. Shift so Monday=0.
	offset := (weekday + 6) % 7
	monday := t.UTC().AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// persistInertiaCard builds and persists a single INERTIA_DETECTED
// DecisionCard for the given verdict.
func (o *InertiaOrchestrator) persistInertiaCard(patientID string, verdict models.InertiaVerdict) error {
	if o.templateLoader == nil || o.db == nil {
		return nil
	}
	tmpl, ok := o.templateLoader.Get(inertiaDetectedTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", inertiaDetectedTemplateID)
	}
	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinician, patientEn, patientHi := renderInertiaSummaries(tmpl, verdict)
	notes := clinician

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "INERTIA_DETECTED_" + string(verdict.Domain),
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  models.GateModify,
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               severityToSafetyTier(verdict.Severity),
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinician,
		PatientSummaryEn:         patientEn,
		PatientSummaryHi:         patientHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := o.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save inertia card: %w", err)
	}
	if o.gateCache != nil {
		_ = o.gateCache.WriteGate(card)
	}
	if o.kb19 != nil {
		go o.kb19.PublishGateChanged(card)
	}
	notifyFHIR(o.fhirNotifier, card)
	if o.escalationManager != nil {
		go o.escalationManager.HandleCardCreated(card, "", 0)
	}
	return nil
}

// persistDualDomainCard builds and persists a DUAL_DOMAIN_INERTIA_DETECTED
// card when the patient has ≥2 detected domains in one evaluation.
func (o *InertiaOrchestrator) persistDualDomainCard(patientID string, report models.PatientInertiaReport) error {
	if o.templateLoader == nil || o.db == nil {
		return nil
	}
	tmpl, ok := o.templateLoader.Get(dualDomainInertiaDetectedTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", dualDomainInertiaDetectedTemplateID)
	}
	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	detected := make([]string, 0, len(report.Verdicts))
	for _, v := range report.Verdicts {
		if v.Detected {
			detected = append(detected, string(v.Domain))
		}
	}
	clinician, patientEn, patientHi := renderDualDomainSummaries(tmpl, detected)
	notes := clinician

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "DUAL_DOMAIN_INERTIA_DETECTED",
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GateModify,
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyUrgent,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinician,
		PatientSummaryEn:         patientEn,
		PatientSummaryHi:         patientHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := o.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save dual-domain inertia card: %w", err)
	}
	if o.gateCache != nil {
		_ = o.gateCache.WriteGate(card)
	}
	if o.kb19 != nil {
		go o.kb19.PublishGateChanged(card)
	}
	notifyFHIR(o.fhirNotifier, card)
	return nil
}

// renderInertiaSummaries substitutes verdict placeholders into the
// inertia-detected template.
func renderInertiaSummaries(tmpl *models.CardTemplate, verdict models.InertiaVerdict) (clinician, patientEn, patientHi string) {
	data := struct {
		Domain              string
		Severity            string
		InertiaDurationDays string
		Pattern             string
	}{
		Domain:              string(verdict.Domain),
		Severity:            string(verdict.Severity),
		InertiaDurationDays: fmt.Sprintf("%d", verdict.InertiaDurationDays),
		Pattern:             string(verdict.Pattern),
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
	if clinician == "" {
		clinician = fmt.Sprintf("Therapeutic inertia detected: %s domain, %s severity, %d days uncontrolled",
			verdict.Domain, verdict.Severity, verdict.InertiaDurationDays)
	}
	return clinician, patientEn, patientHi
}

// renderDualDomainSummaries substitutes the detected-domain list into
// the dual-domain template.
func renderDualDomainSummaries(tmpl *models.CardTemplate, detectedDomains []string) (clinician, patientEn, patientHi string) {
	data := struct {
		DetectedDomains string
		DomainCount     string
	}{
		DetectedDomains: strings.Join(detectedDomains, ", "),
		DomainCount:     fmt.Sprintf("%d", len(detectedDomains)),
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
	if clinician == "" {
		clinician = fmt.Sprintf("Dual-domain therapeutic inertia: %s — composite risk warrants combined escalation review",
			strings.Join(detectedDomains, ", "))
	}
	return clinician, patientEn, patientHi
}

// isDisengaged returns true when the patient's engagement context
// indicates non-adherence. Phase 9 P9-A.
//
// Threshold logic:
//   - EngagementStatus == "DISENGAGED" → definitely disengaged
//   - EngagementComposite != nil && *EngagementComposite < 0.4 → low
//     composite score means <40% engagement across measurement domains
//   - EngagementComposite == nil → no engagement data available →
//     assume engaged (bias toward surfacing inertia cards)
// isPhenotypeStableGood returns true when the patient's stable phenotype
// cluster is STABLE_CONTROLLED — meaning the stability engine has confirmed
// the patient has been in this well-managed cluster for at least 4 weeks
// (or 8 weeks, since STABLE_CONTROLLED has rank 1 = extended dwell).
// V4-7 cross-domain signal: suppresses inertia for these patients.
func isPhenotypeStableGood(input InertiaDetectorInput) bool {
	return input.PhenotypeCluster == "STABLE_CONTROLLED"
}

func isDisengaged(input InertiaDetectorInput) bool {
	if input.EngagementStatus == "DISENGAGED" {
		return true
	}
	if input.EngagementComposite != nil && *input.EngagementComposite < 0.4 {
		return true
	}
	return false
}

// handleAdherenceGap produces an ADHERENCE_GAP report instead of an
// inertia report. It identifies which domains are uncontrolled (from
// the raw domain inputs) and generates one card per uncontrolled
// domain with the adherence-gap template. Phase 9 P9-A.
func (o *InertiaOrchestrator) handleAdherenceGap(ctx context.Context, input InertiaDetectorInput) models.PatientInertiaReport {
	report := models.PatientInertiaReport{
		PatientID:   input.PatientID,
		EvaluatedAt: time.Now(),
	}

	// Identify uncontrolled domains for the adherence-gap verdicts.
	// We DON'T run the full DetectInertia because we don't want to
	// produce inertia verdicts — but we still need to know which
	// domains are off-target so the adherence-gap card can say
	// "glycaemic adherence gap" vs "hemodynamic adherence gap."
	var uncontrolledDomains []models.InertiaDomain
	if input.Glycaemic != nil && !input.Glycaemic.AtTarget {
		uncontrolledDomains = append(uncontrolledDomains, models.DomainGlycaemic)
	}
	if input.Hemodynamic != nil && !input.Hemodynamic.AtTarget {
		uncontrolledDomains = append(uncontrolledDomains, models.DomainHemodynamic)
	}
	if input.Renal != nil && !input.Renal.AtTarget {
		uncontrolledDomains = append(uncontrolledDomains, models.DomainRenal)
	}

	if len(uncontrolledDomains) == 0 {
		// Patient is disengaged but all domains are at target — no
		// adherence gap to surface. This is a patient who is non-
		// adherent but still controlled (possible if they're over-
		// medicated or the non-adherence is to non-critical drugs).
		o.log.Debug("inertia: disengaged patient but all domains at target",
			zap.String("patient_id", input.PatientID))
		return report
	}

	for _, domain := range uncontrolledDomains {
		verdict := models.InertiaVerdict{
			Domain:   domain,
			Pattern:  models.PatternAdherenceGap,
			Detected: true,
			Severity: models.SeverityModerate,
		}
		report.Verdicts = append(report.Verdicts, verdict)

		if err := o.persistAdherenceGapCard(input.PatientID, verdict); err != nil {
			o.log.Error("inertia: failed to persist adherence gap card",
				zap.String("patient_id", input.PatientID),
				zap.String("domain", string(domain)),
				zap.Error(err))
		} else if o.metrics != nil {
			o.metrics.AdherenceGapDetected.WithLabelValues(string(domain)).Inc()
			o.metrics.InertiaSuppressedByAdherence.Inc()
		}
	}

	report.HasAnyInertia = false // adherence gap is NOT inertia
	report.OverallUrgency = "ROUTINE"

	o.log.Info("inertia: adherence gap detected (inertia suppressed)",
		zap.String("patient_id", input.PatientID),
		zap.Int("uncontrolled_domains", len(uncontrolledDomains)),
		zap.String("engagement_status", input.EngagementStatus))

	return report
}

// persistAdherenceGapCard builds and persists a single ADHERENCE_GAP
// card. Uses the same persistence pattern as persistInertiaCard but
// with the adherence-gap template + SAFE gate (advisory only).
func (o *InertiaOrchestrator) persistAdherenceGapCard(patientID string, verdict models.InertiaVerdict) error {
	if o.templateLoader == nil || o.db == nil {
		return nil
	}
	tmpl, ok := o.templateLoader.Get(adherenceGapTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", adherenceGapTemplateID)
	}
	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinician, patientEn, patientHi := renderInertiaSummaries(tmpl, verdict)
	notes := clinician

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "ADHERENCE_GAP_" + string(verdict.Domain),
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  models.GateSafe, // advisory only — don't modify therapy
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityModerate,
		SafetyTier:               models.SafetyRoutine,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinician,
		PatientSummaryEn:         patientEn,
		PatientSummaryHi:         patientHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := o.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save adherence gap card: %w", err)
	}
	if o.gateCache != nil {
		_ = o.gateCache.WriteGate(card)
	}
	if o.kb19 != nil {
		go o.kb19.PublishGateChanged(card)
	}
	notifyFHIR(o.fhirNotifier, card)
	return nil
}

// persistDeprescribingCard builds and persists a DEPRESCRIBING_REVIEW
// card for a polypharmacy-elderly patient. This card is generated in
// addition to (not instead of) the inertia verdicts — it provides
// frailty context so the clinician can decide between escalation and
// deprescribing. Phase 9 P9-F.
func (o *InertiaOrchestrator) persistDeprescribingCard(patientID string, age, medCount int) error {
	if o.templateLoader == nil || o.db == nil {
		return nil
	}
	tmpl, ok := o.templateLoader.Get(deprescribingReviewTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", deprescribingReviewTemplateID)
	}
	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	data := struct {
		Age      string
		MedCount string
	}{
		Age:      fmt.Sprintf("%d", age),
		MedCount: fmt.Sprintf("%d", medCount),
	}

	var clinician, patientEn, patientHi string
	for _, frag := range tmpl.Fragments {
		switch frag.FragmentType {
		case models.FragClinician:
			clinician = executeRenalTemplate(frag.TextEn, data)
		case models.FragPatient:
			patientEn = executeRenalTemplate(frag.TextEn, data)
			patientHi = executeRenalTemplate(frag.TextHi, data)
		}
	}
	if clinician == "" {
		clinician = fmt.Sprintf("DEPRESCRIBING REVIEW: patient is %d years old on %d medications — consider reducing medication burden before adding another agent",
			age, medCount)
	}

	notes := clinician
	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "DEPRESCRIBING_REVIEW",
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  models.GateSafe, // advisory
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityModerate,
		SafetyTier:               models.SafetyRoutine,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinician,
		PatientSummaryEn:         patientEn,
		PatientSummaryHi:         patientHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := o.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save deprescribing review card: %w", err)
	}
	if o.gateCache != nil {
		_ = o.gateCache.WriteGate(card)
	}
	if o.kb19 != nil {
		go o.kb19.PublishGateChanged(card)
	}
	notifyFHIR(o.fhirNotifier, card)
	return nil
}

// severityToSafetyTier maps the inertia detector's severity bracket to
// KB-23's SafetyTier enum.
func severityToSafetyTier(severity models.InertiaSeverity) models.SafetyTier {
	switch severity {
	case models.SeverityCritical, models.SeveritySevere:
		return models.SafetyUrgent
	case models.SeverityModerate:
		return models.SafetyRoutine
	default:
		return models.SafetyRoutine
	}
}
