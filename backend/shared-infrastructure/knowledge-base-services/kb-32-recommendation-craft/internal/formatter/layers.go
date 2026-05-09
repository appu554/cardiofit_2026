// Package formatter implements Stage 6 of the six-stage rendering pipeline:
// the four-layer brevity formatter that enforces word-budget hard caps on the
// Signal and Reasoning layers before a recommendation may be surfaced to any
// clinical audience.
//
// VisibilityClass: AD — four-layer presentation per Guidelines §2;
// word-budget hard caps non-negotiable.
//
// The four-layer presentation architecture ensures that every recommendation
// carries information at exactly the right depth for each consumer: a clinician
// acting under time pressure reads Layer 1 only; a reviewer traces Layer 3 and
// Layer 4 for audit purposes. The word budgets for Layer 1 and Layer 2 are
// non-negotiable: any caller that exceeds them receives an error and MUST NOT
// surface the recommendation until the content is trimmed.
package formatter

// Layer descriptions
//
// Layer 1 (Signal, ≤25 words): the immediate clinical message.
// This is the headline shown to a busy clinician making a time-pressured
// decision. It must be complete, actionable, and free of jargon padding.
// Hard cap: Layer1MaxWords words.
//
// Layer 2 (Reasoning, ≤100 words): the rationale chain.
// Explains why the signal was raised: which patient data, which guideline
// clause, and what the clinical consequence is. Sufficient for a GP to
// understand the recommendation without consulting primary sources.
// Hard cap: Layer2MaxWords words.
//
// Layer 3 (Provenance, structured list): citations + substrate refs.
// An ordered list of evidence anchor IDs, guideline paragraph references,
// and KB-rule substrate identifiers. No word budget — the list must be
// complete. Consumers include pharmacist review and regulator audit.
//
// Layer 4 (Deep Audit, unbounded): full EvidenceTrace lineage.
// The complete serialised EvidenceTrace (see evidence package) that records
// every anchor, scoring step, and confidence calculation that contributed to
// the recommendation. Unbounded because truncation would break audit
// defensibility. Consumed by regulators and automated audit tools only.

// Layer1MaxWords is the inclusive upper bound on word count for the Signal
// layer. Any L1Signal string with more than Layer1MaxWords words must be
// rejected by Validate. This value is a non-negotiable commitment per
// Guidelines §2.
const Layer1MaxWords = 25

// Layer2MaxWords is the inclusive upper bound on word count for the Reasoning
// layer. Any L2Reasoning string with more than Layer2MaxWords words must be
// rejected by Validate. This value is a non-negotiable commitment per
// Guidelines §2.
const Layer2MaxWords = 100

// LayerOutput is the complete four-layer payload produced by the brevity
// formatter. All four fields must be populated (L3Provenance may be an empty
// slice; L4DeepAudit may be an empty string) before the recommendation may
// advance to the delivery stage. Callers must call Validate before storing
// or transmitting a LayerOutput.
type LayerOutput struct {
	// L1Signal is the immediate clinical message (Layer 1).
	// Must not exceed Layer1MaxWords words. This is the headline surfaced
	// to time-pressured clinicians.
	L1Signal string

	// L2Reasoning is the rationale chain (Layer 2).
	// Must not exceed Layer2MaxWords words. This is the supporting
	// explanation shown to reviewers and to clinicians who want context.
	L2Reasoning string

	// L3Provenance is the structured list of citations and substrate
	// references (Layer 3). No word budget — must be complete. Each
	// entry is typically an anchor ID or a guideline paragraph reference.
	L3Provenance []string

	// L4DeepAudit is the full EvidenceTrace lineage (Layer 4). Unbounded;
	// truncation is forbidden. May be empty during early pipeline stages
	// when the full trace is not yet available, but must be populated
	// before regulator delivery.
	L4DeepAudit string
}
