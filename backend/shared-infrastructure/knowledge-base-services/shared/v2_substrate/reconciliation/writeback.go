package reconciliation

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// AcuteReviewWindow is the auto-review window applied to MedicineUse rows
// produced from a discharge line classified as IntentAcuteTemporary
// (Layer 2 §3.2 step 8). Discharge_date + 14d → MedicineUse
// expected_review_date, which the Wave 2.3 active-concern engine
// monitors to fire a post_deprescribing_monitoring concern automatically.
const AcuteReviewWindow = 14 * 24 * time.Hour

// DecisionContext bundles everything ApplyDecision needs to compute
// the substrate mutation. It is the call-site contract: the storage
// layer assembles this from the persisted decision row + diff + ACOP
// payload before invoking ApplyDecision.
type DecisionContext struct {
	Decision      ACOPDecision
	IntentClass   IntentClass
	Diff          DiffEntry
	DischargeAt   time.Time
	Notes         string
	// Override is applied when Decision == ACOPModify. Non-empty fields
	// replace the corresponding values from the discharge line. Other
	// fields are left at their discharge-line defaults.
	Override *DecisionOverride
}

// DecisionOverride carries ACOP-specified replacements for an
// "accept-with-changes" path. Empty fields mean "keep the discharge line
// value"; non-empty fields replace.
type DecisionOverride struct {
	Dose        string
	Frequency   string
	Route       string
	IntentClass IntentClass
	// ReviewDate, when non-nil, replaces the auto-computed
	// expected_review_date for an accepted line.
	ReviewDate *time.Time
}

// MutationKind classifies the MedicineUse mutation a decision implies.
type MutationKind string

const (
	MutationInsert  MutationKind = "insert"  // create a new MedicineUse
	MutationEnd     MutationKind = "end"     // status -> ceased, EndedAt set
	MutationUpdate  MutationKind = "update"  // dose/freq/route changed
	MutationNoop    MutationKind = "noop"    // reject / defer / unchanged
)

// Mutation is the substrate change ApplyDecision returns. The caller
// (storage layer) is responsible for executing the change and writing
// the EvidenceTrace node + edges.
//
// Only one of Insert / Update is populated based on Kind:
//   - Insert: brand-new MedicineUse row to write
//   - End:    Update.ID + Update.EndedAt + Update.Status (ceased) +
//             Update.UpdatedAt are set
//   - Update: existing row id + the changed fields populated
//   - Noop:   no fields populated; caller writes the audit row only
type Mutation struct {
	Kind                  MutationKind
	Insert                *models.MedicineUse
	Update                *MedicineUseChange
	IntentClass           IntentClass // captured for audit/EvidenceTrace
	ExpectedReviewAtSet   bool
	// HumanReadableSummary is a short description of the mutation, useful
	// for EvidenceTrace ReasoningSummary.Text and audit logs.
	HumanReadableSummary string
}

// MedicineUseChange is a sparse update payload — only fields relevant to
// the change are non-zero. The storage layer translates this into a
// SET clause.
type MedicineUseChange struct {
	ID                  uuid.UUID
	Status              string     // when non-empty, set status
	EndedAt             *time.Time // when non-nil, set ended_at
	Dose                string     // when non-empty, replace dose
	Frequency           string     // when non-empty, replace frequency
	Route               string     // when non-empty, replace route
	ExpectedReviewDate  *time.Time // optional: set on stop_criteria.review_date
	ReviewOutcomeNote   string     // append to review_outcome_history (audit)
}

// ApplyDecision is the pure-engine writeback logic. Given the diff
// class, ACOP decision, and optional override, it returns the Mutation
// the storage layer should execute.
//
// Decision matrix (Layer 2 §3.2 step 7):
//
//   class                acop      → mutation
//   ----------------------------------------------
//   new_medication       accept    → insert MedicineUse with intent populated
//                                    (acute_illness_temporary also sets
//                                     expected_review_date = discharge+14d)
//   ceased_medication    accept    → end pre-admission MedicineUse
//                                    (status=ceased, ended_at=discharge_at)
//   dose_change          accept    → update pre-admission MedicineUse with
//                                    discharge dose/freq/route + audit note
//   *                    reject    → noop
//   *                    defer     → noop
//   *                    modify    → like accept, but Override fields
//                                    replace discharge-line values
//   unchanged            *         → noop (should never reach a worklist row)
//
// The function never mutates DecisionContext or its embedded entities.
func ApplyDecision(ctx DecisionContext, residentID uuid.UUID, prescriberID *uuid.UUID, now time.Time) (Mutation, error) {
	if !IsValidACOPDecision(string(ctx.Decision)) {
		return Mutation{}, fmt.Errorf("reconciliation: invalid acop decision %q", ctx.Decision)
	}
	// Reject and defer are no-ops in substrate terms — the audit row alone
	// captures the ACOP's call.
	if ctx.Decision == ACOPReject || ctx.Decision == ACOPDefer {
		return Mutation{
			Kind:                 MutationNoop,
			IntentClass:          ctx.IntentClass,
			HumanReadableSummary: fmt.Sprintf("acop %s: no substrate change", ctx.Decision),
		}, nil
	}

	// Resolve effective intent class — Override wins when present.
	intent := ctx.IntentClass
	if ctx.Decision == ACOPModify && ctx.Override != nil && ctx.Override.IntentClass != "" {
		intent = ctx.Override.IntentClass
	}

	switch ctx.Diff.Class {
	case DiffNewMedication:
		if ctx.Diff.DischargeLineMedicine == nil {
			return Mutation{}, fmt.Errorf("reconciliation: new_medication missing discharge line")
		}
		insert := buildInsert(ctx.Diff.DischargeLineMedicine, residentID, prescriberID, ctx.DischargeAt, intent, ctx.Override)
		// expected_review_date for acute lines.
		reviewSet := false
		if intent == IntentAcuteTemporary {
			rd := ctx.DischargeAt.Add(AcuteReviewWindow)
			if ctx.Override != nil && ctx.Override.ReviewDate != nil {
				rd = *ctx.Override.ReviewDate
			}
			insert.StopCriteria.ReviewDate = &rd
			reviewSet = true
		} else if ctx.Override != nil && ctx.Override.ReviewDate != nil {
			rd := *ctx.Override.ReviewDate
			insert.StopCriteria.ReviewDate = &rd
			reviewSet = true
		}
		return Mutation{
			Kind:                MutationInsert,
			Insert:              insert,
			IntentClass:         intent,
			ExpectedReviewAtSet: reviewSet,
			HumanReadableSummary: fmt.Sprintf("acop %s: new MedicineUse %q (intent=%s)",
				ctx.Decision, insert.DisplayName, intent),
		}, nil

	case DiffCeasedMedication:
		if ctx.Diff.PreAdmissionMedicine == nil {
			return Mutation{}, fmt.Errorf("reconciliation: ceased_medication missing pre-admission row")
		}
		end := ctx.DischargeAt
		change := &MedicineUseChange{
			ID:                ctx.Diff.PreAdmissionMedicine.ID,
			Status:            models.MedicineUseStatusCeased,
			EndedAt:           &end,
			ReviewOutcomeNote: composeAuditNote("ceased on discharge", ctx.Notes),
		}
		return Mutation{
			Kind:        MutationEnd,
			Update:      change,
			IntentClass: intent,
			HumanReadableSummary: fmt.Sprintf("acop %s: ceased MedicineUse %q",
				ctx.Decision, ctx.Diff.PreAdmissionMedicine.DisplayName),
		}, nil

	case DiffDoseChange:
		if ctx.Diff.PreAdmissionMedicine == nil || ctx.Diff.DischargeLineMedicine == nil {
			return Mutation{}, fmt.Errorf("reconciliation: dose_change missing one side of the diff")
		}
		dose := ctx.Diff.DischargeLineMedicine.Dose
		freq := ctx.Diff.DischargeLineMedicine.Frequency
		route := ctx.Diff.DischargeLineMedicine.Route
		if ctx.Decision == ACOPModify && ctx.Override != nil {
			if ctx.Override.Dose != "" {
				dose = ctx.Override.Dose
			}
			if ctx.Override.Frequency != "" {
				freq = ctx.Override.Frequency
			}
			if ctx.Override.Route != "" {
				route = ctx.Override.Route
			}
		}
		change := &MedicineUseChange{
			ID:                ctx.Diff.PreAdmissionMedicine.ID,
			Dose:              dose,
			Frequency:         freq,
			Route:             route,
			ReviewOutcomeNote: composeAuditNote(ctx.Diff.DoseChangeSummary, ctx.Notes),
		}
		return Mutation{
			Kind:        MutationUpdate,
			Update:      change,
			IntentClass: intent,
			HumanReadableSummary: fmt.Sprintf("acop %s: dose change on %q (%s)",
				ctx.Decision, ctx.Diff.PreAdmissionMedicine.DisplayName, ctx.Diff.DoseChangeSummary),
		}, nil

	case DiffUnchanged:
		// Should never reach a worklist row, but be defensive.
		return Mutation{Kind: MutationNoop, IntentClass: intent,
			HumanReadableSummary: "unchanged: no substrate change"}, nil
	}

	return Mutation{}, fmt.Errorf("reconciliation: unhandled diff class %q", ctx.Diff.Class)
}

// buildInsert constructs a fresh MedicineUse from a discharge line.
func buildInsert(line *DischargeLineSummary, residentID uuid.UUID, prescriberID *uuid.UUID, startedAt time.Time, intent IntentClass, override *DecisionOverride) *models.MedicineUse {
	dose := line.Dose
	freq := line.Frequency
	route := line.Route
	if override != nil {
		if override.Dose != "" {
			dose = override.Dose
		}
		if override.Frequency != "" {
			freq = override.Frequency
		}
		if override.Route != "" {
			route = override.Route
		}
	}
	return &models.MedicineUse{
		ID:           uuid.New(),
		ResidentID:   residentID,
		AMTCode:      line.AMTCode,
		DisplayName:  line.DisplayName,
		Dose:         dose,
		Frequency:    freq,
		Route:        route,
		PrescriberID: prescriberID,
		StartedAt:    startedAt,
		Status:       models.MedicineUseStatusActive,
		Intent: models.Intent{
			Category:   mapReconciliationIntentToCategory(intent),
			Indication: line.IndicationText,
			Notes:      composeIntentNotes(intent, line.Notes),
		},
	}
}

// mapReconciliationIntentToCategory translates the reconciliation-time
// IntentClass onto the long-lived MedicineUse intent category. The
// reconciliation classifier is finer-grained than the substrate
// taxonomy, so mapping is many-to-few:
//
//	acute_illness_temporary → therapeutic (with notes/expected_review_date)
//	new_chronic             → therapeutic
//	reconciled_change       → therapeutic
//	unclear                 → unspecified (legacy/migration sentinel — ACOP
//	                          must clarify before substrate is fully populated)
func mapReconciliationIntentToCategory(c IntentClass) string {
	switch c {
	case IntentAcuteTemporary, IntentNewChronic, IntentReconciledChange:
		return models.IntentTherapeutic
	}
	return models.IntentUnspecified
}

// composeIntentNotes records the reconciliation IntentClass on the
// Intent.Notes field so downstream readers can recover the finer
// classification without re-running the classifier.
func composeIntentNotes(intent IntentClass, lineNotes string) string {
	notes := fmt.Sprintf("reconciliation:%s", intent)
	if s := strings.TrimSpace(lineNotes); s != "" {
		notes += " | " + s
	}
	return notes
}

func composeAuditNote(primary, free string) string {
	parts := []string{}
	if s := strings.TrimSpace(primary); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(free); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " | ")
}
