package entry_paths

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestFromCrossReference_HappyPath(t *testing.T) {
	origin := uuid.New()
	target := uuid.New()
	meta, err := FromCrossReference(context.Background(), uuid.New(), target, CrossReferenceContext{
		OriginResidentID: origin,
		ReasonCode:       "medication_class_cross_reference",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Path != EntryPathCrossReference {
		t.Errorf("Path = %q", meta.Path)
	}
	xref := meta.Context.(CrossReferenceContext)
	if xref.OriginResidentID != origin {
		t.Errorf("OriginResidentID not preserved")
	}
	if xref.Kind() != EntryPathCrossReference {
		t.Errorf("Kind() = %q", xref.Kind())
	}
}

func TestFromCrossReference_AcceptsAllCanonicalReasons(t *testing.T) {
	for code := range ValidCrossReferenceReasons {
		_, err := FromCrossReference(context.Background(), uuid.New(), uuid.New(), CrossReferenceContext{
			OriginResidentID: uuid.New(),
			ReasonCode:       code,
		})
		if err != nil {
			t.Errorf("canonical reason %q rejected: %v", code, err)
		}
	}
}

func TestFromCrossReference_RejectsSelfReference(t *testing.T) {
	id := uuid.New()
	_, err := FromCrossReference(context.Background(), uuid.New(), id, CrossReferenceContext{
		OriginResidentID: id,
		ReasonCode:       "family_member",
	})
	if err == nil {
		t.Fatal("expected error when origin_resident_id == target resident_id")
	}
}

func TestFromCrossReference_RejectsZeroOrigin(t *testing.T) {
	_, err := FromCrossReference(context.Background(), uuid.New(), uuid.New(), CrossReferenceContext{
		ReasonCode: "family_member",
	})
	if err == nil {
		t.Fatal("expected error for zero origin_resident_id")
	}
}

func TestFromCrossReference_RejectsUnknownReason(t *testing.T) {
	_, err := FromCrossReference(context.Background(), uuid.New(), uuid.New(), CrossReferenceContext{
		OriginResidentID: uuid.New(),
		ReasonCode:       "made_up_reason_code",
	})
	if err == nil {
		t.Fatal("expected error for non-canonical reason code")
	}
}

func TestFromCrossReference_RejectsEmptyReason(t *testing.T) {
	_, err := FromCrossReference(context.Background(), uuid.New(), uuid.New(), CrossReferenceContext{
		OriginResidentID: uuid.New(),
		ReasonCode:       "",
	})
	if err == nil {
		t.Fatal("expected error for empty reason_code")
	}
}

func TestIsValidEntryPath(t *testing.T) {
	cases := map[string]bool{
		"worklist":        true,
		"search":          true,
		"notification":    true,
		"cross_reference": true,
		"":                false,
		"bogus":           false,
	}
	for in, want := range cases {
		if got := IsValidEntryPath(in); got != want {
			t.Errorf("IsValidEntryPath(%q) = %v want %v", in, got, want)
		}
	}
}
