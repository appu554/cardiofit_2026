package reconciliation

import (
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func mkPre(name, amt, dose, freq string, status string) models.MedicineUse {
	if status == "" {
		status = models.MedicineUseStatusActive
	}
	return models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  uuid.New(),
		AMTCode:     amt,
		DisplayName: name,
		Dose:        dose,
		Frequency:   freq,
		Route:       "oral",
		Status:      status,
	}
}

func mkLine(name, amt, dose, freq string) DischargeLineSummary {
	return DischargeLineSummary{
		LineRef:     uuid.New(),
		AMTCode:     amt,
		DisplayName: name,
		Dose:        dose,
		Frequency:   freq,
		Route:       "oral",
	}
}

func classOf(entries []DiffEntry, want DiffClass) []DiffEntry {
	out := []DiffEntry{}
	for _, e := range entries {
		if e.Class == want {
			out = append(out, e)
		}
	}
	return out
}

func TestComputeDiff_AllFourClasses(t *testing.T) {
	preMet := mkPre("metformin", "AMT-MET-500", "500mg", "BID", "")
	preWar := mkPre("warfarin", "AMT-WAR-2", "2mg", "QD", "") // ceased on discharge
	preRam := mkPre("ramipril", "AMT-RAM-5", "5mg", "QD", "") // dose changed

	pre := []models.MedicineUse{preMet, preWar, preRam}

	lineMet := mkLine("metformin", "AMT-MET-500", "500mg", "BID") // unchanged
	lineRam := mkLine("ramipril", "AMT-RAM-5", "10mg", "QD")      // dose change
	lineApix := mkLine("apixaban", "AMT-APX-5", "5mg", "BID")     // new

	discharge := []DischargeLineSummary{lineMet, lineRam, lineApix}

	got := ComputeDiff(pre, discharge)

	if len(classOf(got, DiffUnchanged)) != 1 {
		t.Fatalf("want 1 unchanged, got %v", got)
	}
	if len(classOf(got, DiffDoseChange)) != 1 {
		t.Fatalf("want 1 dose_change, got %v", got)
	}
	if len(classOf(got, DiffCeasedMedication)) != 1 {
		t.Fatalf("want 1 ceased, got %v", got)
	}
	newRows := classOf(got, DiffNewMedication)
	if len(newRows) != 1 {
		t.Fatalf("want 1 new_medication, got %v", got)
	}
	if newRows[0].DischargeLineRef == nil || *newRows[0].DischargeLineRef != lineApix.LineRef {
		t.Fatalf("new_medication should point at apixaban line; got %+v", newRows[0])
	}

	// Dose-change summary must contain the changed dose.
	dc := classOf(got, DiffDoseChange)[0]
	if dc.DoseChangeSummary == "" {
		t.Fatalf("dose_change must populate summary")
	}
}

func TestComputeDiff_NameFallbackMatch(t *testing.T) {
	// AMT codes intentionally absent on both sides — must still match by
	// shared head token.
	pre := []models.MedicineUse{mkPre("metformin", "", "500mg", "BID", "")}
	discharge := []DischargeLineSummary{mkLine("metformin XR", "", "500mg", "BID")}

	got := ComputeDiff(pre, discharge)
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d (%v)", len(got), got)
	}
	// Names differ in suffix → dose comparison sees same dose/freq → unchanged.
	if got[0].Class != DiffUnchanged {
		t.Fatalf("expected unchanged via name fallback, got %s", got[0].Class)
	}
}

func TestComputeDiff_InactivePreFiltered(t *testing.T) {
	// A ceased pre-admission row should NOT show up in the diff.
	pre := []models.MedicineUse{mkPre("aspirin", "AMT-ASP-100", "100mg", "QD", models.MedicineUseStatusCeased)}
	discharge := []DischargeLineSummary{mkLine("aspirin", "AMT-ASP-100", "100mg", "QD")}

	got := ComputeDiff(pre, discharge)
	// The ceased pre-admission row is filtered, so the discharge line
	// has no pre-admission counterpart and is classified as new.
	if len(got) != 1 || got[0].Class != DiffNewMedication {
		t.Fatalf("expected single new_medication entry, got %v", got)
	}
}

func TestComputeDiff_UnmatchedDischargeLineByName(t *testing.T) {
	// Unique drug on discharge with no pre-admission counterpart.
	pre := []models.MedicineUse{mkPre("metformin", "AMT-MET-500", "500mg", "BID", "")}
	discharge := []DischargeLineSummary{
		mkLine("metformin", "AMT-MET-500", "500mg", "BID"),
		mkLine("clopidogrel", "AMT-CLP-75", "75mg", "QD"),
	}
	got := ComputeDiff(pre, discharge)
	newRows := classOf(got, DiffNewMedication)
	if len(newRows) != 1 || newRows[0].DischargeLineMedicine.DisplayName != "clopidogrel" {
		t.Fatalf("expected single new_medication for clopidogrel, got %v", got)
	}
}

func TestComputeDiff_OrderingDeterministic(t *testing.T) {
	// pre-order: A, B, C. Match A. Cease B. Dose-change C. New D.
	pa := mkPre("amoxicillin", "AMT-AMX-500", "500mg", "TID", "")
	pb := mkPre("bisoprolol", "AMT-BIS-5", "5mg", "QD", "")
	pc := mkPre("candesartan", "AMT-CAN-8", "8mg", "QD", "")
	pre := []models.MedicineUse{pa, pb, pc}

	la := mkLine("amoxicillin", "AMT-AMX-500", "500mg", "TID")
	lc := mkLine("candesartan", "AMT-CAN-8", "16mg", "QD") // dose change
	ld := mkLine("dapagliflozin", "AMT-DAP-10", "10mg", "QD")

	got := ComputeDiff(pre, []DischargeLineSummary{la, lc, ld})

	// First three entries should be matched-pairs in pre order: A unchanged,
	// B ceased, C dose_change. Then D as new.
	if len(got) != 4 {
		t.Fatalf("expected 4 entries got %d (%v)", len(got), got)
	}
	wantClasses := []DiffClass{DiffUnchanged, DiffCeasedMedication, DiffDoseChange, DiffNewMedication}
	for i, w := range wantClasses {
		if got[i].Class != w {
			t.Fatalf("entry %d: want %s got %s", i, w, got[i].Class)
		}
	}
}
