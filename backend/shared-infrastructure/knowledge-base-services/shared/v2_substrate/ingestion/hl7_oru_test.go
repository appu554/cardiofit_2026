package ingestion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseORUR01_SyntheticFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "synthetic_oru_r01.hl7"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	res, err := ParseORUR01(raw, "generic")
	if err != nil {
		t.Fatalf("ParseORUR01: %v", err)
	}

	if res.DocumentID != "MSG-SYN-0001" {
		t.Errorf("DocumentID = %q, want MSG-SYN-0001", res.DocumentID)
	}
	if res.PatientIHI != "8003608000000001" {
		t.Errorf("PatientIHI = %q, want 8003608000000001", res.PatientIHI)
	}
	if res.AuthoredAt.IsZero() {
		t.Errorf("AuthoredAt should be parsed")
	}

	if got, want := len(res.Observations), 3; got != want {
		t.Fatalf("len(Observations) = %d, want %d", got, want)
	}

	po := res.Observations[0]
	if po.LOINCCode != "2823-3" {
		t.Errorf("obs[0].LOINCCode = %q", po.LOINCCode)
	}
	if po.Value == nil || *po.Value != 5.8 {
		t.Errorf("obs[0].Value = %v, want 5.8", po.Value)
	}
	if po.Unit != "mmol/L" {
		t.Errorf("obs[0].Unit = %q", po.Unit)
	}
	if po.AbnormalFlag != "high" {
		t.Errorf("obs[0].AbnormalFlag = %q, want high", po.AbnormalFlag)
	}

	po = res.Observations[1]
	if po.LOINCCode != "33914-3" {
		t.Errorf("obs[1].LOINCCode = %q", po.LOINCCode)
	}
	if po.Value == nil || *po.Value != 42 {
		t.Errorf("obs[1].Value = %v", po.Value)
	}
	if po.AbnormalFlag != "low" {
		t.Errorf("obs[1].AbnormalFlag = %q, want low", po.AbnormalFlag)
	}

	po = res.Observations[2]
	if po.LOINCCode != "11556-8" {
		t.Errorf("obs[2].LOINCCode = %q", po.LOINCCode)
	}
	if po.Value != nil {
		t.Errorf("obs[2].Value should be nil for ST")
	}
	if po.ValueText != "No organisms seen on Gram stain." {
		t.Errorf("obs[2].ValueText = %q", po.ValueText)
	}
}

func TestParseORUR01_NoMSH(t *testing.T) {
	_, err := ParseORUR01([]byte("PID|1|||DOE^JANE\n"), "")
	if err == nil {
		t.Fatalf("expected error when first segment is not MSH")
	}
}

// rewriteUnitVendorAdapter is a tiny per-vendor adapter used to
// exercise the registry contract: rewrites "mmol/L" to "mmol_per_L".
type rewriteUnitVendorAdapter struct{}

func (r *rewriteUnitVendorAdapter) Adapt(po ParsedObservation) ParsedObservation {
	if po.Unit == "mmol/L" {
		po.Unit = "mmol_per_L"
	}
	return po
}

func TestVendorAdapterRegistry_AppliesAdapter(t *testing.T) {
	RegisterVendorAdapter("test-vendor-rewrite-units", &rewriteUnitVendorAdapter{})
	raw, err := os.ReadFile(filepath.Join("testdata", "synthetic_oru_r01.hl7"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	res, err := ParseORUR01(raw, "test-vendor-rewrite-units")
	if err != nil {
		t.Fatalf("ParseORUR01: %v", err)
	}
	if res.Observations[0].Unit != "mmol_per_L" {
		t.Errorf("vendor adapter did not rewrite unit; got %q", res.Observations[0].Unit)
	}
}

func TestVendorAdapterRegistry_UnknownFallsToGeneric(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "synthetic_oru_r01.hl7"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	res, err := ParseORUR01(raw, "completely-unregistered-vendor-xyz")
	if err != nil {
		t.Fatalf("ParseORUR01 should not fail for unknown vendor: %v", err)
	}
	if res.Observations[0].Unit != "mmol/L" {
		t.Errorf("expected pass-through for unknown vendor; got Unit=%q", res.Observations[0].Unit)
	}
}
