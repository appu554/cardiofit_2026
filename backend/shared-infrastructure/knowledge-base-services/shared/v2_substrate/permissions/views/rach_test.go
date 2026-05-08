package views

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeRACHSource struct {
	mismatchRACH bool
}

func (f *fakeRACHSource) RollupForRACH(_ context.Context, rachID uuid.UUID, metric string) (RACHRollup, error) {
	id := rachID
	if f.mismatchRACH {
		id = uuid.New() // deliberately different
	}
	return RACHRollup{
		RACHID:         id,
		Metric:         metric,
		AggregateValue: 0.91,
	}, nil
}

func TestRACHView_HappyPath(t *testing.T) {
	src := &fakeRACHSource{}
	v := NewRACHView(src)
	rachID := uuid.New()

	rollup, err := v.Rollup(context.Background(), rachID, "rir_mean")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rollup.RACHID != rachID {
		t.Errorf("expected RACHID %v, got %v", rachID, rollup.RACHID)
	}
	if rollup.Metric != "rir_mean" {
		t.Errorf("expected metric rir_mean, got %s", rollup.Metric)
	}
}

func TestRACHView_MismatchedRACHID(t *testing.T) {
	src := &fakeRACHSource{mismatchRACH: true}
	v := NewRACHView(src)

	_, err := v.Rollup(context.Background(), uuid.New(), "rir_mean")
	if err == nil {
		t.Errorf("expected error for mismatched RACH ID, got nil")
	}
}
