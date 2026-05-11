package aggregation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildFamilyCommunicationContext_EmptyState(t *testing.T) {
	rid := uuid.New()
	client := NewInMemorySubstrateClient()
	fc, err := BuildFamilyCommunicationContext(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.LastMeetingDate != nil {
		t.Error("expected nil LastMeetingDate when none seeded")
	}
	if fc.SubstrateRefs == nil {
		t.Error("SubstrateRefs must be non-nil empty slice")
	}
	if len(fc.SubstrateRefs) != 0 {
		t.Errorf("expected empty SubstrateRefs, got %d", len(fc.SubstrateRefs))
	}
}

func TestBuildFamilyCommunicationContext_WithMeeting(t *testing.T) {
	rid := uuid.New()
	when := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	client := NewInMemorySubstrateClient().WithLastFamilyMeeting(rid, when)
	fc, err := BuildFamilyCommunicationContext(context.Background(), client, rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.LastMeetingDate == nil || !fc.LastMeetingDate.Equal(when) {
		t.Errorf("LastMeetingDate = %v, want %v", fc.LastMeetingDate, when)
	}
	if len(fc.SubstrateRefs) != 1 {
		t.Fatalf("expected 1 SubstrateRef, got %d", len(fc.SubstrateRefs))
	}
}

func TestBuildFamilyCommunicationContext_NilClient(t *testing.T) {
	_, err := BuildFamilyCommunicationContext(context.Background(), nil, uuid.New())
	if err == nil {
		t.Fatal("expected error on nil SubstrateClient")
	}
}
