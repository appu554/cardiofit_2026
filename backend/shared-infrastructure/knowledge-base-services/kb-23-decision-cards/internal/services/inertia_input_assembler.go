package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// InertiaInterventionTimelineFetcher is the narrow interface the
// assembler needs for intervention-timeline lookups. Production wires
// this to KB20Client.FetchInterventionTimeline; tests inject a stub.
type InertiaInterventionTimelineFetcher interface {
	FetchInterventionTimeline(ctx context.Context, patientID string) (*KB20InterventionTimeline, error)
}

// InertiaPatientContextFetcher is the narrow interface the assembler
// needs for KB-20 summary-context + renal-status lookups. KB20Client
// satisfies this via its existing FetchSummaryContext + FetchRenalStatus
// methods.
type InertiaPatientContextFetcher interface {
	FetchSummaryContext(ctx context.Context, patientID string) (*PatientContext, error)
	FetchRenalStatus(ctx context.Context, patientID string) (*KB20RenalStatus, error)
}

// InertiaTargetStatusFetcher is the narrow interface the assembler needs
// for KB-26 target-status compute.
type InertiaTargetStatusFetcher interface {
	FetchTargetStatus(ctx context.Context, patientID string, req KB26TargetStatusRequest) (*KB26TargetStatusResponse, error)
}

// InertiaCGMLatestFetcher is the narrow interface the assembler needs
// to pull the latest CGM period report from KB-26. Phase 7 P7-E
// Milestone 2: when a recent report is available, its TIR replaces
// the HbA1c-based glycaemic target status with a HIGH-confidence
// CGM_TIR branch reading.
type InertiaCGMLatestFetcher interface {
	FetchLatestCGMReport(ctx context.Context, patientID string) (*KB26CGMLatestReport, error)
}

// ConcreteInertiaInputAssembler fetches KB-20 + KB-26 data and assembles
// an InertiaDetectorInput for the given patient. Phase 7 P7-D + P7-E.
//
// The assembler is deliberately not an interface on its own — it is the
// concrete implementation of the InertiaInputAssembler interface that
// InertiaWeeklyBatch already consumes. Tests can construct it with stub
// fetchers; production wires it from main.go with real KB-20/KB-26 clients.
//
// Phase 7 P7-E Milestone 2: cgmLatestFetcher is optional — when a
// patient has CGM period reports available, TIR is preferred over
// HbA1c for the glycaemic inertia branch (HIGH vs MODERATE confidence).
// When nil or the fetch returns a nil report, the assembler falls back
// to the HbA1c-based target status path unchanged.
type ConcreteInertiaInputAssembler struct {
	timelineFetcher  InertiaInterventionTimelineFetcher
	contextFetcher   InertiaPatientContextFetcher
	targetFetcher    InertiaTargetStatusFetcher
	cgmLatestFetcher InertiaCGMLatestFetcher
	log              *zap.Logger
}

// NewInertiaInputAssembler constructs the assembler.
func NewInertiaInputAssembler(
	timelineFetcher InertiaInterventionTimelineFetcher,
	contextFetcher InertiaPatientContextFetcher,
	targetFetcher InertiaTargetStatusFetcher,
	cgmLatestFetcher InertiaCGMLatestFetcher,
	log *zap.Logger,
) *ConcreteInertiaInputAssembler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ConcreteInertiaInputAssembler{
		timelineFetcher:  timelineFetcher,
		contextFetcher:   contextFetcher,
		targetFetcher:    targetFetcher,
		cgmLatestFetcher: cgmLatestFetcher,
		log:              log,
	}
}

// AssembleInertiaInput implements the InertiaInputAssembler interface
// (defined in inertia_weekly_batch.go). Returns an InertiaDetectorInput
// populated from three upstream fetches:
//
//   1. KB-20 summary context (patient stratum, active meds, latest HbA1c/FBG/weight)
//   2. KB-20 renal status (eGFR + slope, CKD stage) — needed for renal domain input
//   3. KB-20 intervention timeline (latest clinical action per domain)
//   4. KB-26 target status (compute from the raw measurements)
//
// The assembler is tolerant of individual fetch failures: if the target-
// status compute fails (e.g. KB-26 unreachable), the resulting input
// simply has nil domain slots, and DetectInertia degrades to producing
// no verdicts for those domains rather than erroring out.
func (a *ConcreteInertiaInputAssembler) AssembleInertiaInput(ctx context.Context, patientID string) (InertiaDetectorInput, error) {
	input := InertiaDetectorInput{PatientID: patientID}

	// 1. KB-20 summary context — active medications, latest HbA1c/FBG,
	// stratum. This call must succeed — without it we cannot know which
	// medications the patient is on.
	summary, err := a.contextFetcher.FetchSummaryContext(ctx, patientID)
	if err != nil {
		return input, fmt.Errorf("fetch KB-20 summary context: %w", err)
	}
	if summary == nil {
		return input, fmt.Errorf("KB-20 summary context nil for %s", patientID)
	}

	// Phase 9 P9-A: thread engagement context into the detector input
	// so the orchestrator can check adherence before running
	// DetectInertia. Nil EngagementComposite is fine — the orchestrator
	// treats it as "no engagement data available, assume engaged" which
	// biases toward surfacing inertia cards rather than suppressing them.
	input.EngagementStatus = summary.EngagementStatus
	input.EngagementComposite = summary.EngagementComposite

	// 2. KB-20 renal status — eGFR + slope for the renal domain. Non-fatal
	// if missing; renal domain input simply stays nil.
	var renalStatus *KB20RenalStatus
	if rs, rsErr := a.contextFetcher.FetchRenalStatus(ctx, patientID); rsErr == nil {
		renalStatus = rs
	} else {
		a.log.Debug("inertia assembler: renal status fetch failed",
			zap.String("patient_id", patientID),
			zap.Error(rsErr))
	}

	// 3. KB-20 intervention timeline — latest action per domain.
	timeline, err := a.timelineFetcher.FetchInterventionTimeline(ctx, patientID)
	if err != nil {
		a.log.Debug("inertia assembler: intervention timeline fetch failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
		timeline = &KB20InterventionTimeline{PatientID: patientID, ByDomain: map[string]KB20LatestDomainAction{}}
	}

	// 4. KB-26 target status — computed from the raw summary measurements.
	//    Uses HbA1c + mean SBP from KB-20 summary context; eGFR from renal
	//    status. If the context is missing any field, KB-26 returns the
	//    corresponding domain AtTarget=false with CurrentValue=0, which
	//    produces no false-positive inertia (the detector requires
	//    ConsecutiveReadings > 0 for a verdict).
	targetReq := buildKB26TargetStatusRequest(summary, renalStatus)
	targetResp, err := a.targetFetcher.FetchTargetStatus(ctx, patientID, targetReq)
	if err != nil {
		a.log.Debug("inertia assembler: target status fetch failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
		// Continue with nil targetResp — domain inputs will stay nil.
	}

	// Populate per-domain inputs from target status + timeline.
	if targetResp != nil {
		if glyc := buildDomainInertiaInput(targetResp.Glycaemic, timeline.ByDomain["GLYCAEMIC"], summary.Medications); glyc != nil {
			input.Glycaemic = glyc
		}
		if hemo := buildDomainInertiaInput(targetResp.Hemodynamic, timeline.ByDomain["HEMODYNAMIC"], summary.Medications); hemo != nil {
			input.Hemodynamic = hemo
		}
		if renal := buildDomainInertiaInput(targetResp.Renal, timeline.ByDomain["RENAL"], summary.Medications); renal != nil {
			input.Renal = renal
		}
	}

	// Phase 7 P7-E Milestone 2: CGM TIR override for the glycaemic
	// domain. When KB-26 has a recent CGM period report for this
	// patient, replace the HbA1c-driven glycaemic input with a
	// CGM_TIR reading (HIGH confidence, 14-day window). The inertia
	// detector honours DataSource="CGM_TIR" via its cgmMinDays=14
	// branch — the detector's pure logic does not change, only the
	// input assembly does.
	if a.cgmLatestFetcher != nil {
		if cgm, cgmErr := a.cgmLatestFetcher.FetchLatestCGMReport(ctx, patientID); cgmErr == nil && cgm != nil {
			input.Glycaemic = buildCGMGlycaemicInput(cgm, timeline.ByDomain["GLYCAEMIC"], summary.Medications)
			a.log.Debug("inertia assembler: CGM_TIR override applied",
				zap.String("patient_id", patientID),
				zap.Float64("tir_pct", cgm.TIRPct),
				zap.String("gri_zone", cgm.GRIZone))
		} else if cgmErr != nil {
			a.log.Debug("inertia assembler: CGM latest fetch failed",
				zap.String("patient_id", patientID),
				zap.Error(cgmErr))
		}
	}

	return input, nil
}

// buildCGMGlycaemicInput constructs a DomainInertiaInput from a KB-26
// CGM period report. TIR is the clinical indicator (target: ≥70 per
// ADA SOC 2025 §6), the period-end date is both the
// "most-recent-reading" anchor and — when under target — the
// first-uncontrolled-at proxy. DataSource = CGM_TIR to select the
// detector's cgmMinDays=14 branch. Pure function — exported for unit
// testing without a database or HTTP client.
func buildCGMGlycaemicInput(
	cgm *KB26CGMLatestReport,
	timeline KB20LatestDomainAction,
	meds []string,
) *DomainInertiaInput {
	const tirTarget = 70.0
	input := &DomainInertiaInput{
		AtTarget:            cgm.TIRPct >= tirTarget,
		CurrentValue:        cgm.TIRPct,
		TargetValue:         tirTarget,
		ConsecutiveReadings: 1,
		DataSource:          "CGM_TIR",
		CurrentMeds:         meds,
	}

	// DaysUncontrolled: approximated by the span from the CGM period
	// start to today when the patient is off-target. A CGM period
	// report already aggregates 14 days, so "uncontrolled for at
	// least 14 days" is the minimum — if the report came in today
	// with TIR<70, the detector's cgmMinDays=14 threshold is met.
	if !input.AtTarget {
		input.DaysUncontrolled = 14
	}

	// Last intervention from the GLYCAEMIC-domain timeline entry, if
	// any — matches the HbA1c-path behaviour for consistency.
	if !timeline.ActionDate.IsZero() {
		t := timeline.ActionDate
		input.LastIntervention = &t
	}

	return input
}

// buildKB26TargetStatusRequest assembles the POST body for KB-26's
// target-status endpoint from KB-20 summary context + renal status.
func buildKB26TargetStatusRequest(summary *PatientContext, renal *KB20RenalStatus) KB26TargetStatusRequest {
	req := KB26TargetStatusRequest{}
	if summary != nil {
		if summary.LatestHbA1c > 0 {
			h := summary.LatestHbA1c
			req.HbA1c = &h
		}
	}
	if renal != nil {
		if renal.EGFR > 0 {
			e := renal.EGFR
			req.EGFR = &e
		}
	}
	// Note: PatientContext does not currently carry mean SBP 7d — when
	// a future KB-20 summary-context expansion adds it, plumb it here.
	return req
}

// buildDomainInertiaInput translates a KB-26 target-status verdict and
// KB-20 intervention-timeline entry into a DomainInertiaInput for the
// inertia detector. Returns nil when there's nothing worth feeding
// (no target measurement OR patient is at target AND no timeline entry).
func buildDomainInertiaInput(status KB26DomainTargetStatus, timeline KB20LatestDomainAction, meds []string) *DomainInertiaInput {
	if status.CurrentValue == 0 && timeline.InterventionID == "" {
		return nil
	}

	input := &DomainInertiaInput{
		AtTarget:            status.AtTarget,
		CurrentValue:        status.CurrentValue,
		TargetValue:         status.TargetValue,
		DaysUncontrolled:    status.DaysUncontrolled,
		ConsecutiveReadings: status.ConsecutiveReadings,
		DataSource:          status.DataSource,
		CurrentMeds:         meds,
	}
	if !timeline.ActionDate.IsZero() {
		t := timeline.ActionDate
		input.LastIntervention = &t
	}

	// Flag AtMaxDose only when the timeline carries the dose and we can
	// compare against a canonical max. For now this is left false —
	// the detector treats AtMaxDose + TargetUnmet as the Ceiling trigger,
	// and we don't want to false-positive on missing dose data.
	input.AtMaxDose = false

	return input
}

// contextHasIntervention returns true if the patient has at least one
// active medication in any domain. Used by the batch to skip no-op
// patients. Currently unused — kept for the P7-D follow-up that
// short-circuits evaluation on empty regimens.
func contextHasIntervention(summary *PatientContext) bool {
	if summary == nil {
		return false
	}
	for _, med := range summary.Medications {
		if strings.TrimSpace(med) != "" {
			return true
		}
	}
	return false
}

// toInertiaTime parses an ISO-8601 date string to time.Time; returns
// zero time on parse failure.
func toInertiaTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Ensure ConcreteInertiaInputAssembler satisfies the batch's expected
// InertiaInputAssembler interface at compile time.
var _ InertiaInputAssembler = (*ConcreteInertiaInputAssembler)(nil)
