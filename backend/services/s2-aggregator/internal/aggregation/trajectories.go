package aggregation

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// Trajectory is the per-parameter rendering unit per S2 v1.0 Part 5.1.
// Each Trajectory carries at least one SubstrateRef so the verification-
// not-belief discipline (v1.0 Part 17 critical structural test) is
// enforceable at test time.
//
// CurrentValue, Velocity, Baseline90d are pointers so absence is
// distinguishable from zero — required by v1.0 Part 5.3 sparse-data
// graceful degradation.
type Trajectory struct {
	Parameter      string
	Unit           string
	CurrentValue   *float64
	Velocity       *float64 // per-year rate of change; nil when <3 observations
	Baseline90d    *float64 // mean over 90-day prior window
	ThresholdFlags []ThresholdFlag
	SparseDataFlag bool // true when <3 observations in the trajectory window
	AssessedAt     time.Time
	SubstrateRefs  []SubstrateRef
	Confidence     string // "high" | "moderate" | "low" per v1.0 Part 10.3
}

// ThresholdFlag is a clinically-meaningful boundary crossing per v1.0
// Part 5.1 ("Threshold flags for clinically relevant boundaries").
//
// Kind is a stable lower_snake_case label (e.g., "egfr_below_30").
// CrossedAt is the timestamp of the first observation at-or-past the
// threshold within the trajectory window.
type ThresholdFlag struct {
	Kind        string
	CrossedAt   time.Time
	Description string
}

// MultiParameterComposition is the v1.0 Part 5.4 multi-parameter shift
// pattern. The pattern catalogue itself is senior-pharmacist-authoring
// content; ComputeMultiParameterCompositions ships the structural
// framework + two examples for tests.
type MultiParameterComposition struct {
	CompositionLabel       string
	ContributingParameters []string
	SubstrateRefs          []SubstrateRef
}

// trajectoryParam binds a Snapshot field accessor to the parameter name
// + unit + threshold ruleset. This keeps BuildTrajectories table-driven
// instead of a long switch.
type trajectoryParam struct {
	name             string
	unit             string
	snapshotAccessor func(s Snapshot) *float64
	thresholds       func(value float64, observedAt time.Time) []ThresholdFlag
}

// parameterCatalogue lists every parameter rendered as a trajectory in
// Layer 1 per S2 v1.0 Part 5.2. The catalogue is intentionally narrower
// than v1.0's full list (which names MMSE/MoCA/4AT, ADL scores, drug
// levels, TSH/sodium/potassium/FBC, BPSD): Phase 1 substrate (kb-20)
// carries the parameters below; the rest are
// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1).
var parameterCatalogue = []trajectoryParam{
	{
		name: "egfr", unit: "mL/min/1.73m²",
		snapshotAccessor: func(s Snapshot) *float64 { return s.EGFR },
		thresholds:       egfrThresholds,
	},
	{
		name: "dbi", unit: "index",
		snapshotAccessor: func(s Snapshot) *float64 { return s.DBI },
		// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1):
		// DBI clinical threshold (literature commonly cites ≥1.0 as elevated
		// risk; conservative placeholder pending senior pharmacist authoring).
		thresholds: dbiThresholdsPlaceholder,
	},
	{
		name: "acb", unit: "index",
		snapshotAccessor: func(s Snapshot) *float64 { return s.ACB },
		// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1):
		// ACB threshold (literature commonly cites ≥3 as clinically significant).
		thresholds: acbThresholdsPlaceholder,
	},
	{
		name: "cfs", unit: "scale",
		snapshotAccessor: func(s Snapshot) *float64 { return s.CFS },
		// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1):
		// CFS threshold per v1.0 Part 11.1 names CFS ≥6 as complex-workspace
		// activation criterion — used as a placeholder boundary.
		thresholds: cfsThresholdsPlaceholder,
	},
	{
		name: "weight", unit: "kg",
		snapshotAccessor: func(s Snapshot) *float64 { return s.Weight },
		// TODO(senior consultant pharmacist authoring): weight loss is
		// typically flagged as %-change rather than absolute value;
		// computed in Velocity field, no absolute-threshold flag here.
		thresholds: func(_ float64, _ time.Time) []ThresholdFlag { return nil },
	},
	{
		name: "bp_systolic", unit: "mmHg",
		snapshotAccessor: func(s Snapshot) *float64 { return s.BPSystolic },
		// TODO(senior consultant pharmacist authoring): BP thresholds in
		// aged care are goals-of-care-dependent; no placeholder here.
		thresholds: func(_ float64, _ time.Time) []ThresholdFlag { return nil },
	},
	{
		name: "bp_diastolic", unit: "mmHg",
		snapshotAccessor: func(s Snapshot) *float64 { return s.BPDiastolic },
		thresholds: func(_ float64, _ time.Time) []ThresholdFlag { return nil },
	},
}

// egfrThresholds names the canonical CKD-stage-3b boundary. eGFR <30
// indicates CKD stage 4 (severely reduced kidney function) and is widely
// used as a renal-dose-adjustment threshold. v1.0 Part 5.1 cites
// "CKD stage boundaries" as a threshold flag class.
func egfrThresholds(value float64, observedAt time.Time) []ThresholdFlag {
	if value < 30 {
		return []ThresholdFlag{{
			Kind:        "egfr_below_30",
			CrossedAt:   observedAt,
			Description: "eGFR <30 mL/min/1.73m² (CKD stage 4) — renal dose adjustment review indicated",
		}}
	}
	if value < 45 {
		return []ThresholdFlag{{
			Kind:        "egfr_below_45",
			CrossedAt:   observedAt,
			Description: "eGFR <45 mL/min/1.73m² (CKD stage 3b)",
		}}
	}
	return nil
}

func dbiThresholdsPlaceholder(value float64, observedAt time.Time) []ThresholdFlag {
	// Placeholder: ≥1.0 is the literature-cited "elevated risk" threshold.
	if value >= 1.0 {
		return []ThresholdFlag{{
			Kind:        "dbi_elevated_placeholder",
			CrossedAt:   observedAt,
			Description: "DBI ≥1.0 (placeholder threshold — pending senior pharmacist authoring)",
		}}
	}
	return nil
}

func acbThresholdsPlaceholder(value float64, observedAt time.Time) []ThresholdFlag {
	// Placeholder: ≥3 is the literature-cited "clinically significant" threshold.
	if value >= 3 {
		return []ThresholdFlag{{
			Kind:        "acb_elevated_placeholder",
			CrossedAt:   observedAt,
			Description: "ACB ≥3 (placeholder threshold — pending senior pharmacist authoring)",
		}}
	}
	return nil
}

func cfsThresholdsPlaceholder(value float64, observedAt time.Time) []ThresholdFlag {
	// Per v1.0 Part 11.1, CFS ≥6 is a complex-workspace activation criterion.
	if value >= 6 {
		return []ThresholdFlag{{
			Kind:        "cfs_severely_frail",
			CrossedAt:   observedAt,
			Description: "CFS ≥6 (severely frail or worse — per v1.0 Part 11.1)",
		}}
	}
	return nil
}

// BuildTrajectories computes the per-parameter trajectory set for a
// resident as of asOf. The returned slice is ordered to match
// parameterCatalogue. Parameters with zero observations on record are
// included with SparseDataFlag=true so the renderer can show
// "no observation on record" per v1.0 Part 5.3.
//
// Each returned Trajectory carries at least one SubstrateRef; if no
// observation exists for the parameter, the SubstrateRef points at the
// resident with a "no observation" description. This preserves the
// verification-not-belief invariant structurally — even an absence is
// auditable.
func BuildTrajectories(ctx context.Context, client SubstrateClient, residentID uuid.UUID, asOf time.Time) ([]Trajectory, error) {
	if client == nil {
		return nil, fmt.Errorf("BuildTrajectories: nil SubstrateClient")
	}

	out := make([]Trajectory, 0, len(parameterCatalogue)+3)

	for _, p := range parameterCatalogue {
		obs, err := client.TrajectoryHistory(ctx, residentID, p.name)
		if err != nil {
			return nil, fmt.Errorf("BuildTrajectories(%s): %w", p.name, err)
		}
		out = append(out, buildOne(residentID, p, obs, asOf))
	}

	// PRN velocity trajectories — one per Phase 1 class.
	for _, cls := range []substrate_types.PRNClass{
		substrate_types.PRNBenzodiazepine,
		substrate_types.PRNAntipsychotic,
		substrate_types.PRNAnalgesic,
	} {
		adm, err := client.RecentPRNAdministrations(ctx, residentID, cls, asOf)
		if err != nil {
			return nil, fmt.Errorf("BuildTrajectories(prn:%s): %w", cls, err)
		}
		out = append(out, buildPRNTrajectory(residentID, cls, adm, asOf))
	}

	return out, nil
}

// buildOne computes a Trajectory for one numeric parameter given its
// observation series. Implements v1.0 Part 5.3 graceful degradation:
// 0 obs → SparseDataFlag, no values; 1–2 obs → SparseDataFlag, no
// velocity; 3+ obs → full trajectory with velocity + baseline.
func buildOne(residentID uuid.UUID, p trajectoryParam, obs []substrate_types.Observation, asOf time.Time) Trajectory {
	t := Trajectory{
		Parameter:  p.name,
		Unit:       p.unit,
		AssessedAt: asOf,
	}

	if len(obs) == 0 {
		t.SparseDataFlag = true
		t.Confidence = "low"
		// Verification-not-belief: even "no observation" carries a ref so
		// the absence is auditable. The ref points to the resident with a
		// descriptive "no observation" label.
		t.SubstrateRefs = []SubstrateRef{{
			Source:      "kb-20",
			ID:          residentID,
			Description: fmt.Sprintf("no %s observation on record", p.name),
		}}
		return t
	}

	// Observations come sorted oldest-first from the SubstrateClient
	// contract; defensive sort here anyway.
	sort.Slice(obs, func(i, j int) bool { return obs[i].ObservedAt.Before(obs[j].ObservedAt) })

	latest := obs[len(obs)-1]
	cv := latest.Value
	t.CurrentValue = &cv
	t.Confidence = latest.Confidence
	if t.Confidence == "" {
		t.Confidence = "high"
	}

	// SubstrateRefs: latest + first observation in window — enough for
	// drill-through to anchor both endpoints.
	for _, o := range obs {
		t.SubstrateRefs = append(t.SubstrateRefs, SubstrateRef{
			Source:      o.Source,
			ID:          o.ID,
			Description: fmt.Sprintf("%s=%v %s @ %s", p.name, o.Value, o.Unit, o.ObservedAt.Format("2006-01-02")),
		})
	}

	// Threshold flags computed from current value.
	if p.thresholds != nil {
		t.ThresholdFlags = p.thresholds(latest.Value, latest.ObservedAt)
	}

	if len(obs) < 3 {
		// v1.0 Part 5.3: "2–3 observations: render observations with
		// first-to-last delta; no formal velocity." We mark sparse but
		// still expose current value + refs.
		t.SparseDataFlag = true
		return t
	}

	// 3+ observations: compute velocity (per-year) and baseline (90d mean).
	t.Velocity = computeVelocityPerYear(obs)
	t.Baseline90d = computeBaseline90d(obs, asOf)
	return t
}

// computeVelocityPerYear fits a simple two-point velocity using oldest
// and newest observations in the series, expressed as units per year.
// A more sophisticated fit (linear regression with confidence interval)
// is named in v1.0 Part 5.1 as "trend line with confidence interval" —
// that work is TODO(senior consultant pharmacist + biostatistician
// authoring per S2 Addendum Part 6.1).
func computeVelocityPerYear(obs []substrate_types.Observation) *float64 {
	if len(obs) < 2 {
		return nil
	}
	first := obs[0]
	last := obs[len(obs)-1]
	years := last.ObservedAt.Sub(first.ObservedAt).Hours() / (24 * 365.25)
	if years <= 0 || math.IsNaN(years) {
		return nil
	}
	v := (last.Value - first.Value) / years
	return &v
}

// computeBaseline90d returns the mean over observations whose
// ObservedAt falls inside (asOf-180d, asOf-90d] — the prior-90d window
// per v1.0 Part 5.1 "Baseline comparison". Returns nil if no
// observations fall in the window.
func computeBaseline90d(obs []substrate_types.Observation, asOf time.Time) *float64 {
	start := asOf.Add(-180 * 24 * time.Hour)
	end := asOf.Add(-90 * 24 * time.Hour)
	var sum float64
	var n int
	for _, o := range obs {
		if o.ObservedAt.After(start) && !o.ObservedAt.After(end) {
			sum += o.Value
			n++
		}
	}
	if n == 0 {
		return nil
	}
	mean := sum / float64(n)
	return &mean
}

// buildPRNTrajectory wraps prn_velocity-shaped computation into a
// Trajectory. The full prn_velocity.Compute lives in the shared module
// (see substrate_types pin tests); s2-aggregator re-implements the
// 30d-vs-prior-90d-mean ratio locally to keep the dependency surface
// small. Severity bucketing matches CAPE Guidelines v1.1 lines 283–289.
func buildPRNTrajectory(residentID uuid.UUID, cls substrate_types.PRNClass, adm []substrate_types.PRNAdministration, asOf time.Time) Trajectory {
	t := Trajectory{
		Parameter:  fmt.Sprintf("prn_velocity_%s", cls),
		Unit:       "admins/30d",
		AssessedAt: asOf,
	}

	recentStart := asOf.Add(-30 * 24 * time.Hour)
	baselineStart := asOf.Add(-120 * 24 * time.Hour)

	var recent, baseline int
	for _, a := range adm {
		switch {
		case a.AdministeredAt.After(recentStart) && !a.AdministeredAt.After(asOf):
			recent++
		case a.AdministeredAt.After(baselineStart) && !a.AdministeredAt.After(recentStart):
			baseline++
		}
	}
	baselineAvg := float64(baseline) / 3.0

	recentF := float64(recent)
	t.CurrentValue = &recentF
	t.Baseline90d = &baselineAvg

	// Sparse-data flag: <3 administrations in the full 120d window means
	// velocity is not interpretable. v1.0 Part 5.3 discipline applied.
	if len(adm) < 3 {
		t.SparseDataFlag = true
	}

	var ratio float64
	switch {
	case baselineAvg == 0 && recent > 0:
		ratio = math.Inf(+1)
	case baselineAvg == 0:
		ratio = 0
	default:
		ratio = recentF / baselineAvg
	}
	if !math.IsInf(ratio, 0) {
		t.Velocity = &ratio
	}

	// Severity flag per CAPE bucket table.
	if sev := prnSeverity(ratio); sev >= 3 {
		t.ThresholdFlags = []ThresholdFlag{{
			Kind:        fmt.Sprintf("prn_velocity_severity_%d", sev),
			CrossedAt:   asOf,
			Description: fmt.Sprintf("PRN %s velocity severity %d (recent=%d, baseline_avg=%.2f)", cls, sev, recent, baselineAvg),
		}}
	}

	// Verification-not-belief: at least one ref. If administrations are
	// present, ref the most recent; otherwise ref the resident with
	// "no recent PRN administrations".
	if len(adm) > 0 {
		// Find most recent.
		latest := adm[0]
		for _, a := range adm {
			if a.AdministeredAt.After(latest.AdministeredAt) {
				latest = a
			}
		}
		t.SubstrateRefs = []SubstrateRef{{
			Source:      "prn_velocity",
			ID:          latest.ResidentID,
			Description: fmt.Sprintf("most-recent %s administration @ %s", cls, latest.AdministeredAt.Format("2006-01-02")),
		}}
	} else {
		t.SubstrateRefs = []SubstrateRef{{
			Source:      "prn_velocity",
			ID:          residentID,
			Description: fmt.Sprintf("no recent %s PRN administrations", cls),
		}}
	}

	t.Confidence = "high"
	return t
}

// prnSeverity applies the CAPE Guidelines v1.1 lines 283–289 bucket table.
func prnSeverity(ratio float64) int {
	switch {
	case math.IsInf(ratio, +1):
		return 5
	case ratio > 4.0:
		return 5
	case ratio > 2.5:
		return 4
	case ratio > 1.5:
		return 3
	case ratio > 1.0:
		return 2
	default:
		return 1
	}
}

// ComputeMultiParameterCompositions detects concurrent shifts across
// trajectories per v1.0 Part 5.4. Pattern-matching is the structural
// framework; the full catalogue of compositions is
// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1).
//
// Two examples ship for tests:
//
//  1. Anticholinergic burden + renal decline: ACB elevated AND eGFR
//     declining (eGFR <45 threshold flag present).
//  2. Frailty + PRN escalation: CFS ≥6 AND any PRN class severity ≥3.
//
// These examples are STRUCTURAL — they prove the pattern-matching shape
// works. Clinical patterns must be authored by a senior pharmacist
// before pilot deployment.
func ComputeMultiParameterCompositions(trajectories []Trajectory) []MultiParameterComposition {
	byParam := make(map[string]Trajectory, len(trajectories))
	for _, t := range trajectories {
		byParam[t.Parameter] = t
	}

	var out []MultiParameterComposition

	// Example 1: anticholinergic burden + renal decline.
	if hasThresholdLike(byParam["acb"], "acb_elevated") && hasThresholdLike(byParam["egfr"], "egfr_below") {
		out = append(out, MultiParameterComposition{
			CompositionLabel:       "anticholinergic burden + renal decline",
			ContributingParameters: []string{"acb", "egfr"},
			SubstrateRefs: mergeRefs(
				byParam["acb"].SubstrateRefs,
				byParam["egfr"].SubstrateRefs,
			),
		})
	}

	// Example 2: severe frailty + PRN escalation (any class).
	if hasThresholdLike(byParam["cfs"], "cfs_severely_frail") {
		for _, cls := range []substrate_types.PRNClass{
			substrate_types.PRNBenzodiazepine,
			substrate_types.PRNAntipsychotic,
			substrate_types.PRNAnalgesic,
		} {
			key := fmt.Sprintf("prn_velocity_%s", cls)
			if hasThresholdLike(byParam[key], "prn_velocity_severity_") {
				out = append(out, MultiParameterComposition{
					CompositionLabel:       fmt.Sprintf("severe frailty + escalating %s use", cls),
					ContributingParameters: []string{"cfs", key},
					SubstrateRefs: mergeRefs(
						byParam["cfs"].SubstrateRefs,
						byParam[key].SubstrateRefs,
					),
				})
			}
		}
	}

	// TODO(senior consultant pharmacist authoring per S2 Addendum Part 6.1):
	// extend the composition catalogue. Candidate patterns named in
	// v1.0 Part 5.4 include weight loss + cognitive decline + renal
	// decline (composite "general decline"), polypharmacy + falls,
	// orthostatic instability + antihypertensive intensification, etc.
	return out
}

// hasThresholdLike reports whether t carries a ThresholdFlag whose Kind
// starts with prefix. Empty prefix or zero Trajectory → false.
func hasThresholdLike(t Trajectory, prefix string) bool {
	if prefix == "" {
		return false
	}
	for _, f := range t.ThresholdFlags {
		if len(f.Kind) >= len(prefix) && f.Kind[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// mergeRefs concatenates SubstrateRef slices preserving order. Caller
// must own the slices.
func mergeRefs(a, b []SubstrateRef) []SubstrateRef {
	out := make([]SubstrateRef, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}
