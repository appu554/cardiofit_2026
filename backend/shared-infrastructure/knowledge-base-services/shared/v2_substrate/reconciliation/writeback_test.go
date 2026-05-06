package reconciliation

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestApplyDecision_NewMedicationAcceptInsertsWithIntent(t *testing.T) {
	dischargeAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	line := &DischargeLineSummary{
		LineRef:        uuid.New(),
		AMTCode:        "AMT-APX-5",
		DisplayName:    "apixaban",
		Dose:           "5mg",
		Frequency:      "BID",
		Route:          "oral",
		IndicationText: "atrial fibrillation",
	}
	ctx := DecisionContext{
		Decision:    ACOPAccept,
		IntentClass: IntentNewChronic,
		Diff:        DiffEntry{Class: DiffNewMedication, DischargeLineMedicine: line},
		DischargeAt: dischargeAt,
	}
	resident := uuid.New()
	mut, err := ApplyDecision(ctx, resident, nil, dischargeAt)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if mut.Kind != MutationInsert {
		t.Fatalf("want insert, got %s", mut.Kind)
	}
	if mut.Insert.ResidentID != resident || mut.Insert.AMTCode != "AMT-APX-5" {
		t.Fatalf("inserted row mismatch: %+v", mut.Insert)
	}
	if mut.Insert.Intent.Category != models.IntentTherapeutic {
		t.Fatalf("new_chronic must map to therapeutic, got %s", mut.Insert.Intent.Category)
	}
	if !strings.Contains(mut.Insert.Intent.Notes, "reconciliation:new_chronic") {
		t.Fatalf("intent notes must record reconciliation class: %q", mut.Insert.Intent.Notes)
	}
	if mut.ExpectedReviewAtSet {
		t.Fatalf("non-acute insert must NOT set expected_review_date")
	}
}

func TestApplyDecision_AcuteTemporarySetsReviewDate(t *testing.T) {
	dischargeAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	line := &DischargeLineSummary{
		LineRef: uuid.New(), DisplayName: "amoxicillin", Dose: "500mg", Frequency: "TID",
		IndicationText: "post-op infection",
	}
	ctx := DecisionContext{
		Decision: ACOPAccept, IntentClass: IntentAcuteTemporary,
		Diff:        DiffEntry{Class: DiffNewMedication, DischargeLineMedicine: line},
		DischargeAt: dischargeAt,
	}
	mut, err := ApplyDecision(ctx, uuid.New(), nil, dischargeAt)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if mut.Kind != MutationInsert || mut.Insert == nil {
		t.Fatalf("want insert")
	}
	if !mut.ExpectedReviewAtSet {
		t.Fatalf("acute_illness_temporary must set expected_review_date")
	}
	want := dischargeAt.Add(AcuteReviewWindow)
	if mut.Insert.StopCriteria.ReviewDate == nil || !mut.Insert.StopCriteria.ReviewDate.Equal(want) {
		t.Fatalf("expected_review_date should be discharge+14d, got %v", mut.Insert.StopCriteria.ReviewDate)
	}
}

func TestApplyDecision_CeasedAcceptEndsMedicineUse(t *testing.T) {
	pre := models.MedicineUse{ID: uuid.New(), DisplayName: "warfarin", Status: models.MedicineUseStatusActive}
	dischargeAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	ctx := DecisionContext{
		Decision: ACOPAccept, IntentClass: IntentUnclear,
		Diff:        DiffEntry{Class: DiffCeasedMedication, PreAdmissionMedicine: &pre},
		DischargeAt: dischargeAt,
		Notes:       "switched to apixaban",
	}
	mut, err := ApplyDecision(ctx, uuid.New(), nil, dischargeAt)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if mut.Kind != MutationEnd || mut.Update == nil {
		t.Fatalf("want end mutation")
	}
	if mut.Update.ID != pre.ID || mut.Update.Status != models.MedicineUseStatusCeased {
		t.Fatalf("end mismatch: %+v", mut.Update)
	}
	if mut.Update.EndedAt == nil || !mut.Update.EndedAt.Equal(dischargeAt) {
		t.Fatalf("ended_at should equal discharge_at")
	}
	if !strings.Contains(mut.Update.ReviewOutcomeNote, "switched to apixaban") {
		t.Fatalf("audit note should carry ACOP free text: %q", mut.Update.ReviewOutcomeNote)
	}
}

func TestApplyDecision_DoseChangeAcceptUpdatesFields(t *testing.T) {
	pre := models.MedicineUse{ID: uuid.New(), DisplayName: "ramipril", Dose: "5mg", Frequency: "QD",
		Route: "oral", Status: models.MedicineUseStatusActive}
	line := &DischargeLineSummary{Dose: "10mg", Frequency: "QD", Route: "oral"}
	ctx := DecisionContext{
		Decision: ACOPAccept, IntentClass: IntentReconciledChange,
		Diff: DiffEntry{Class: DiffDoseChange, PreAdmissionMedicine: &pre, DischargeLineMedicine: line,
			DoseChangeSummary: `dose "5mg"→"10mg"`},
		DischargeAt: time.Now(),
	}
	mut, err := ApplyDecision(ctx, uuid.New(), nil, time.Now())
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if mut.Kind != MutationUpdate || mut.Update == nil {
		t.Fatalf("want update")
	}
	if mut.Update.Dose != "10mg" {
		t.Fatalf("dose not updated, got %q", mut.Update.Dose)
	}
	if !strings.Contains(mut.Update.ReviewOutcomeNote, `dose "5mg"→"10mg"`) {
		t.Fatalf("audit note must carry diff summary: %q", mut.Update.ReviewOutcomeNote)
	}
}

func TestApplyDecision_RejectAndDeferAreNoops(t *testing.T) {
	for _, dec := range []ACOPDecision{ACOPReject, ACOPDefer} {
		ctx := DecisionContext{
			Decision: dec, IntentClass: IntentNewChronic,
			Diff:        DiffEntry{Class: DiffNewMedication, DischargeLineMedicine: &DischargeLineSummary{}},
			DischargeAt: time.Now(),
		}
		mut, err := ApplyDecision(ctx, uuid.New(), nil, time.Now())
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		if mut.Kind != MutationNoop {
			t.Errorf("decision %s should be noop, got %s", dec, mut.Kind)
		}
	}
}

func TestApplyDecision_ModifyAppliesOverrides(t *testing.T) {
	pre := models.MedicineUse{ID: uuid.New(), DisplayName: "ramipril", Dose: "5mg", Frequency: "QD"}
	line := &DischargeLineSummary{Dose: "10mg", Frequency: "QD"}
	ctx := DecisionContext{
		Decision: ACOPModify, IntentClass: IntentReconciledChange,
		Diff: DiffEntry{Class: DiffDoseChange, PreAdmissionMedicine: &pre, DischargeLineMedicine: line},
		Override: &DecisionOverride{Dose: "7.5mg"},
		DischargeAt: time.Now(),
	}
	mut, err := ApplyDecision(ctx, uuid.New(), nil, time.Now())
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if mut.Update.Dose != "7.5mg" {
		t.Fatalf("override dose not honoured, got %q", mut.Update.Dose)
	}
}

func TestApplyDecision_InvalidDecisionRejected(t *testing.T) {
	_, err := ApplyDecision(DecisionContext{Decision: "weirdo"}, uuid.New(), nil, time.Now())
	if err == nil {
		t.Fatalf("expected error for invalid decision")
	}
}
