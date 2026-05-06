package ingestion

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeAMT is an in-memory AMTLookup keyed on lowercase medication name.
// Strength and form are appended for exact-match scoring.
type fakeAMT struct {
	exact map[string]string // name|strength|form → code
	name  map[string]string // name → code (fuzzy: confidence 0.7)
}

func newFakeAMT() *fakeAMT {
	return &fakeAMT{
		exact: map[string]string{
			"paracetamol|500mg|tablet": "AMT-PCM-500-TAB",
			"amlodipine|5mg|tablet":    "AMT-AML-5-TAB",
			"metformin|500mg|tablet":   "AMT-MET-500-TAB",
		},
		name: map[string]string{
			"paracetamol": "AMT-PCM-GENERIC",
			"amlodipine":  "AMT-AML-GENERIC",
			"metformin":   "AMT-MET-GENERIC",
		},
	}
}

func (f *fakeAMT) LookupByName(_ context.Context, name, strength, form string) (string, float64, error) {
	key := strings.ToLower(name) + "|" + strings.ToLower(strength) + "|" + strings.ToLower(form)
	if code, ok := f.exact[key]; ok {
		return code, 1.0, nil
	}
	if code, ok := f.name[strings.ToLower(name)]; ok {
		return code, 0.7, nil
	}
	return "", 0.0, nil
}

type fakeSNOMED struct {
	m map[string]string
}

func (f *fakeSNOMED) LookupIndication(_ context.Context, text string) (string, float64, error) {
	key := strings.ToLower(strings.TrimSpace(text))
	if code, ok := f.m[key]; ok {
		return code, 1.0, nil
	}
	// crude fuzzy: any exact-keyword substring → 0.6
	for k, code := range f.m {
		if strings.Contains(key, k) {
			return code, 0.6, nil
		}
	}
	return "", 0.0, nil
}

type erroringAMT struct{}

func (erroringAMT) LookupByName(_ context.Context, _, _, _ string) (string, float64, error) {
	return "", 0.0, errors.New("transport boom")
}

func TestNormalise_AMTHappyPath(t *testing.T) {
	n := &Normaliser{AMT: newFakeAMT(), SNOMED: &fakeSNOMED{m: map[string]string{
		"osteoarthritis pain": "SCT-OA-PAIN",
	}}}
	row := CSVRow{
		MedicationName: "paracetamol", Strength: "500mg", Form: "tablet",
		IndicationText: "osteoarthritis pain",
	}
	got, err := n.Normalise(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.AMTCode != "AMT-PCM-500-TAB" || got.AMTConfidence != 1.0 {
		t.Errorf("AMT: got %q conf %.2f", got.AMTCode, got.AMTConfidence)
	}
	if got.PrimaryIndication != "SCT-OA-PAIN" || got.IndicationConfidence != 1.0 {
		t.Errorf("SNOMED: got %q conf %.2f", got.PrimaryIndication, got.IndicationConfidence)
	}
}

func TestNormalise_AMTNotFound(t *testing.T) {
	n := &Normaliser{AMT: newFakeAMT()}
	row := CSVRow{MedicationName: "obscure-investigational-drug", Strength: "1mg"}
	got, err := n.Normalise(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.AMTCode != "" || got.AMTConfidence != 0.0 {
		t.Errorf("expected empty AMT, got %q conf %.2f", got.AMTCode, got.AMTConfidence)
	}
}

func TestNormalise_AMTPartialMatchConfidence(t *testing.T) {
	n := &Normaliser{AMT: newFakeAMT()}
	// strength/form mismatch → fuzzy name-only path → confidence 0.7
	row := CSVRow{MedicationName: "paracetamol", Strength: "1g", Form: "syrup"}
	got, err := n.Normalise(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.AMTCode != "AMT-PCM-GENERIC" || got.AMTConfidence != 0.7 {
		t.Errorf("expected fuzzy match, got %q conf %.2f", got.AMTCode, got.AMTConfidence)
	}
}

func TestNormalise_EmptyIndicationOK(t *testing.T) {
	n := &Normaliser{AMT: newFakeAMT(), SNOMED: &fakeSNOMED{m: map[string]string{}}}
	row := CSVRow{MedicationName: "amlodipine", Strength: "5mg", Form: "tablet"}
	got, err := n.Normalise(context.Background(), row)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.PrimaryIndication != "" {
		t.Errorf("expected empty SNOMED for empty indication, got %q", got.PrimaryIndication)
	}
}

func TestNormalise_NilAMTReturnsConfigError(t *testing.T) {
	n := &Normaliser{}
	_, err := n.Normalise(context.Background(), CSVRow{MedicationName: "x"})
	if !errors.Is(err, ErrNoAMTLookup) {
		t.Errorf("expected ErrNoAMTLookup, got %v", err)
	}
}

func TestNormalise_AMTTransportErrorPropagates(t *testing.T) {
	n := &Normaliser{AMT: erroringAMT{}}
	_, err := n.Normalise(context.Background(), CSVRow{MedicationName: "x"})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected transport error, got %v", err)
	}
}

func TestStubLookups_AlwaysNotFound(t *testing.T) {
	a := StubAMTLookup{}
	c, conf, err := a.LookupByName(context.Background(), "anything", "", "")
	if err != nil || c != "" || conf != 0.0 {
		t.Errorf("stub AMT: got %q conf %.2f err %v", c, conf, err)
	}
	s := StubSNOMEDLookup{}
	c, conf, err = s.LookupIndication(context.Background(), "anything")
	if err != nil || c != "" || conf != 0.0 {
		t.Errorf("stub SNOMED: got %q conf %.2f err %v", c, conf, err)
	}
}
