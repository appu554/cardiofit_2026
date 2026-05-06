package reconciliation

import "testing"

func TestClassifyIntent_AcuteKeywords(t *testing.T) {
	cases := []string{
		"started after sepsis on day 2",
		"post-op pain control",
		"acute exacerbation of COPD",
		"DVT prophylaxis course",
	}
	for _, txt := range cases {
		got := ClassifyIntent(DiffEntry{Class: DiffNewMedication}, txt)
		if got != IntentAcuteTemporary {
			t.Errorf("text %q: want acute_illness_temporary, got %s", txt, got)
		}
	}
}

func TestClassifyIntent_ChronicMarkers(t *testing.T) {
	got := ClassifyIntent(DiffEntry{Class: DiffNewMedication},
		"started for ongoing hypertension management — long-term therapy")
	if got != IntentNewChronic {
		t.Errorf("want new_chronic, got %s", got)
	}
}

func TestClassifyIntent_ReconciledChange(t *testing.T) {
	got := ClassifyIntent(DiffEntry{Class: DiffDoseChange},
		"dose increased to optimise BP control")
	if got != IntentReconciledChange {
		t.Errorf("want reconciled_change, got %s", got)
	}
}

func TestClassifyIntent_DefaultUnclear(t *testing.T) {
	got := ClassifyIntent(DiffEntry{Class: DiffNewMedication}, "")
	if got != IntentUnclear {
		t.Errorf("empty text must be unclear, got %s", got)
	}
	got = ClassifyIntent(DiffEntry{Class: DiffNewMedication}, "no relevant signals here")
	if got != IntentUnclear {
		t.Errorf("text with no markers must be unclear, got %s", got)
	}
}

func TestClassifyIntent_CeasedAndUnchangedAlwaysUnclear(t *testing.T) {
	for _, c := range []DiffClass{DiffCeasedMedication, DiffUnchanged} {
		got := ClassifyIntent(DiffEntry{Class: c}, "post-op infection")
		if got != IntentUnclear {
			t.Errorf("class %s must always classify unclear, got %s", c, got)
		}
	}
}

func TestClassifyIntent_AcuteWinsOverChronicWhenBothPresent(t *testing.T) {
	got := ClassifyIntent(DiffEntry{Class: DiffNewMedication},
		"started for ongoing care — initial reason was sepsis")
	if got != IntentAcuteTemporary {
		t.Errorf("acute should win, got %s", got)
	}
}

func TestComposeDischargeText(t *testing.T) {
	line := DischargeLineSummary{IndicationText: "atrial fibrillation", Notes: "long-term anticoagulation"}
	got := ComposeDischargeText(&line)
	want := "atrial fibrillation long-term anticoagulation"
	if got != want {
		t.Errorf("compose: want %q got %q", want, got)
	}
	if ComposeDischargeText(nil) != "" {
		t.Errorf("nil line must yield empty text")
	}
}
