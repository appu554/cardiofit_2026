package views

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRegulatorSource struct {
	missingHash bool
}

func (f *fakeRegulatorSource) AuditPackFor(_ context.Context, scope string, since time.Time, until time.Time) (AuditPack, error) {
	hash := "sha256:abc123def456"
	if f.missingHash {
		hash = ""
	}
	return AuditPack{
		PackID:      uuid.New(),
		ContentHash: hash,
		Items: []AuditItem{
			{
				ID:        uuid.New(),
				Class:     "recommendation_decision",
				OriginRef: uuid.New(),
			},
		},
	}, nil
}

func TestRegulatorView_MissingHash(t *testing.T) {
	src := &fakeRegulatorSource{missingHash: true}
	v := NewRegulatorView(src)

	_, err := v.AuditPack(context.Background(), "pharmacy:AU:NSW", time.Now().Add(-24*time.Hour), time.Now())
	if err == nil {
		t.Fatalf("expected ErrMissingProvenance, got nil")
	}
	if err != ErrMissingProvenance {
		t.Errorf("expected ErrMissingProvenance, got: %v", err)
	}
}

func TestRegulatorView_HappyPath(t *testing.T) {
	src := &fakeRegulatorSource{}
	v := NewRegulatorView(src)

	since := time.Now().Add(-24 * time.Hour)
	until := time.Now()
	pack, err := v.AuditPack(context.Background(), "pharmacy:AU:NSW", since, until)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pack.ContentHash == "" {
		t.Errorf("expected non-empty ContentHash")
	}
	if len(pack.Items) == 0 {
		t.Errorf("expected at least one AuditItem")
	}
}
