package eba_register

import (
	"context"
	"testing"
)

func TestInMemoryRegister_RoundTrip(t *testing.T) {
	r := NewInMemoryRegister()
	ctx := context.Background()
	f := Finding{
		FindingType: "acceptance_appropriateness_divergence",
		Severity:    3,
		Description: "rule X diverged",
	}
	if err := r.Append(ctx, f); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got := r.List()
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	if got[0].FindingType != "acceptance_appropriateness_divergence" {
		t.Errorf("FindingType=%q", got[0].FindingType)
	}
	if got[0].Status != "open" {
		t.Errorf("default Status=%q, want open", got[0].Status)
	}
	if got[0].ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected non-nil ID")
	}
	if got[0].DetectedAt.IsZero() {
		t.Errorf("DetectedAt not defaulted")
	}
}

func TestLogOnlyRegister_AppendNil(t *testing.T) {
	if err := (LogOnlyRegister{}).Append(context.Background(), Finding{
		FindingType: "smoke",
		Severity:    1,
		Description: "smoke",
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
}
