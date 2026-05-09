package exports_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cardiofit/pharmacist-self-visibility/internal/exports"
)

// stubRPLSource is a test double for RPLSource.
type stubRPLSource struct {
	data map[string][]exports.EvidenceItem
	err  error
}

func (s *stubRPLSource) CandidatesForDimension(ctx context.Context, pharmacistID, dimension string) ([]exports.EvidenceItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data[dimension], nil
}

// TestRPLPack_FiveCompetencyDimensions — verbatim test from plan.
// Verifies that a pack generated across all 5 APC dimensions contains exactly
// one item per dimension, with anonymisation applied.
func TestRPLPack_FiveCompetencyDimensions(t *testing.T) {
	dims := []string{
		"clinical_assessment",
		"medication_review",
		"communication",
		"quality_use_of_medicines",
		"professional_practice",
	}

	data := map[string][]exports.EvidenceItem{}
	for _, d := range dims {
		data[d] = []exports.EvidenceItem{
			{Title: "Evidence for " + d, Dimension: d, Anonymised: false, Annotation: "some annotation", OriginRef: "ref-" + d},
		}
	}

	src := &stubRPLSource{data: data}
	gen := exports.NewRPLGenerator(src)

	pack, err := gen.Generate(context.Background(), "pharm-1", dims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pack.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(pack.Items))
	}

	seen := map[string]bool{}
	for _, item := range pack.Items {
		if !item.Anonymised {
			t.Errorf("item %q was not anonymised", item.Title)
		}
		seen[item.Dimension] = true
	}

	for _, d := range dims {
		if !seen[d] {
			t.Errorf("missing dimension %q in pack", d)
		}
	}

	if pack.PharmacistID != "pharm-1" {
		t.Errorf("unexpected pharmacist ID %q", pack.PharmacistID)
	}
	if pack.ID == "" {
		t.Error("pack ID must not be empty")
	}
	if pack.GeneratedAt.IsZero() {
		t.Error("pack GeneratedAt must be set")
	}
}

// TestRPLPack_PropagatesSourceError — source error propagates to caller.
func TestRPLPack_PropagatesSourceError(t *testing.T) {
	wantErr := errors.New("db connection failed")
	src := &stubRPLSource{err: wantErr}
	gen := exports.NewRPLGenerator(src)

	_, err := gen.Generate(context.Background(), "pharm-2", []string{"clinical_assessment"})
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

// TestRPLPack_EmptyDimensionsReturnsEmptyPack — calling Generate with an empty
// dimensions slice returns a valid RPLPack with no Items.
func TestRPLPack_EmptyDimensionsReturnsEmptyPack(t *testing.T) {
	src := &stubRPLSource{data: map[string][]exports.EvidenceItem{}}
	gen := exports.NewRPLGenerator(src)

	pack, err := gen.Generate(context.Background(), "pharm-3", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pack.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(pack.Items))
	}
	if pack.ID == "" {
		t.Error("pack ID must not be empty even for empty pack")
	}
	if pack.PharmacistID != "pharm-3" {
		t.Errorf("unexpected pharmacist ID %q", pack.PharmacistID)
	}
	if pack.GeneratedAt.IsZero() {
		t.Error("pack GeneratedAt must be set")
	}
}

// TestRPLPack_DimensionWithNoCandidates — when a dimension has no candidates
// that dimension is skipped; other dimensions are still populated.
func TestRPLPack_DimensionWithNoCandidates(t *testing.T) {
	data := map[string][]exports.EvidenceItem{
		"clinical_assessment": {
			{Title: "Assessment evidence", Dimension: "clinical_assessment", OriginRef: "ref-1"},
		},
		// "medication_review" intentionally absent (returns empty slice)
		"communication": {
			{Title: "Communication evidence", Dimension: "communication", OriginRef: "ref-3"},
		},
	}
	src := &stubRPLSource{data: data}
	gen := exports.NewRPLGenerator(src)

	dims := []string{"clinical_assessment", "medication_review", "communication"}
	pack, err := gen.Generate(context.Background(), "pharm-4", dims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 2 dimensions had candidates; medication_review should be skipped.
	if len(pack.Items) != 2 {
		t.Errorf("expected 2 items (skipping empty dimension), got %d", len(pack.Items))
	}

	dims_present := map[string]bool{}
	for _, item := range pack.Items {
		dims_present[item.Dimension] = true
	}
	if dims_present["medication_review"] {
		t.Error("medication_review should have been skipped (no candidates)")
	}
	if !dims_present["clinical_assessment"] {
		t.Error("clinical_assessment should be present")
	}
	if !dims_present["communication"] {
		t.Error("communication should be present")
	}
}
