package aggregation

import (
	"errors"
	"time"
)

// CAPEContextBand is the top-region rendering carrier for the CAPE
// context band per S2 v1.0 Part 4.1 Component 2. It surfaces the primary
// CAPE signals that drove worklist prioritisation, retained from kb-33 —
// not re-derived in S2 (v1.0 Part 3.1 + Addendum Part 4.8 carry-through).
//
// A zero-value band (no signals) is the correct rendering when the
// entry path was anything other than worklist.
type CAPEContextBand struct {
	Signals       []SignalDisplay
	CAPEScore     float64
	TriagedAt     time.Time
	SubstrateRefs []SubstrateRef
}

// SignalDisplay is the per-signal rendering payload. HumanReadable is
// rendered from Code via the lookup table in capeSignalDisplay below.
// Severity is an integer 1–5 extracted from the signal code where the
// code encodes severity (e.g., acute_event_severity_5_fall → 5).
type SignalDisplay struct {
	Code          string
	HumanReadable string
	Severity      int
}

// capeSignalDisplay maps canonical CAPE signal codes to display strings
// for the CAPE context band. This is a Phase 1 stub — five plausible
// codes covering acute events, trajectory velocities, and recommendation
// aging suffice for tests.
//
// TODO(senior consultant pharmacist authoring): canonical CAPE signal
// display vocabulary. The full taxonomy lives in kb-33's signal catalogue;
// the displayed strings are a clinical-communication decision that needs
// senior pharmacist sign-off before pilot.
var capeSignalDisplay = map[string]struct {
	HumanReadable string
	Severity      int
}{
	"acute_event_severity_5_fall":         {"Fall with injury 3 days ago", 5},
	"acute_event_severity_4_delirium":     {"Acute delirium episode this week", 4},
	"trajectory_velocity_4_egfr_decline":  {"eGFR declining at clinically meaningful velocity", 4},
	"trajectory_velocity_3_weight_loss":   {"Weight loss trajectory crossing threshold", 3},
	"recommendation_aging_overdue_review": {"Pending recommendation overdue for GP review", 2},
	"monitoring_overdue_lithium_level":    {"Lithium level monitoring overdue", 3},
}

// ErrInvalidWorklistContext is returned by BuildCAPEContextBand when the
// EntryPathMetadata Context is not a WorklistContext despite Path being
// EntryPathWorklist.
var ErrInvalidWorklistContext = errors.New("worklist entry-path metadata missing WorklistContext")

// BuildCAPEContextBand renders the CAPE context band for the top region
// of S2 per v1.0 Part 4.1 Component 2 + Addendum Part 4.8.
//
// When the entry path is not worklist, an empty band is returned — the
// band only renders for triage-driven entries per v1.0 Part 3.1.
//
// Every signal carries at least one SubstrateRef per the
// verification-not-belief discipline (v1.0 Part 10). Until kb-33 ships
// substrate IDs alongside CAPE signals, a sentinel pending-integration
// ref is attached.
func BuildCAPEContextBand(meta EntryPathMetadata) (CAPEContextBand, error) {
	if meta.Path != EntryPathWorklist {
		// Empty band — the only correct rendering for non-worklist entries.
		return CAPEContextBand{}, nil
	}
	wc, ok := meta.Context.(WorklistContext)
	if !ok {
		return CAPEContextBand{}, ErrInvalidWorklistContext
	}
	signals := make([]SignalDisplay, 0, len(wc.PrimarySignals))
	refs := make([]SubstrateRef, 0, len(wc.PrimarySignals))
	for _, code := range wc.PrimarySignals {
		display, known := capeSignalDisplay[code]
		if !known {
			// TODO(senior consultant pharmacist authoring): canonical
			// CAPE signal display vocabulary. Unknown codes render verbatim
			// rather than fabricating a clinical phrase.
			signals = append(signals, SignalDisplay{
				Code:          code,
				HumanReadable: code,
				Severity:      0,
			})
		} else {
			signals = append(signals, SignalDisplay{
				Code:          code,
				HumanReadable: display.HumanReadable,
				Severity:      display.Severity,
			})
		}
		// TODO(kb-33 Step 5 integration): replace with actual substrate
		// IDs carried alongside CAPE signals from kb-33-triage-engine.
		refs = append(refs, SubstrateRef{
			Source:      "kb-33",
			Description: "substrate ref pending kb-33 integration for signal " + code,
		})
	}
	return CAPEContextBand{
		Signals:       signals,
		CAPEScore:     wc.CAPEScore,
		TriagedAt:     wc.TriagedAt,
		SubstrateRefs: refs,
	}, nil
}
