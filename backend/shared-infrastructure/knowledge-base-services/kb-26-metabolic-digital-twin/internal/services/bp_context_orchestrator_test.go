package services

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/models"
)

// stubKB20Client implements the KB20Fetcher interface for tests.
type stubKB20Client struct {
	profile *clients.KB20PatientProfile
	err     error
}

func (s *stubKB20Client) FetchProfile(ctx context.Context, patientID string) (*clients.KB20PatientProfile, error) {
	return s.profile, s.err
}

// stubKB21Client implements the KB21Fetcher interface for tests.
type stubKB21Client struct {
	profile *clients.KB21EngagementProfile
	err     error
}

func (s *stubKB21Client) FetchEngagement(ctx context.Context, patientID string) (*clients.KB21EngagementProfile, error) {
	return s.profile, s.err
}

func ptrFloat(v float64) *float64 { return &v }

func newOrchestrator(t *testing.T, kb20 KB20Fetcher, kb21 KB21Fetcher) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t) // from bp_context_repository_test.go
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	return NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil)
}

func TestBPContextOrchestrator_MaskedHTN_Persists(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(148),
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128),
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
			IsDiabetic:       false,
			HasCKD:           false,
			OnHTNMeds:        false,
		},
	}
	kb21 := &stubKB21Client{
		profile: &clients.KB21EngagementProfile{
			PatientID: "p1",
			Phenotype: "STEADY",
		},
	}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}

	// Verify snapshot was persisted.
	latest, err := orch.repo.FetchLatest("p1")
	if err != nil {
		t.Fatalf("FetchLatest failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected snapshot to be persisted")
	}
	if latest.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected persisted MASKED_HTN, got %s", latest.Phenotype)
	}
}

func TestBPContextOrchestrator_KB20Unavailable_ReturnsError(t *testing.T) {
	kb20 := &stubKB20Client{err: errSimulated()}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	_, err := orch.Classify(context.Background(), "p1")
	if err == nil {
		t.Error("expected error when KB-20 unavailable")
	}
}

func TestBPContextOrchestrator_KB21Unavailable_ContinuesWithoutEngagement(t *testing.T) {
	// KB-21 is non-critical: the classifier still works without engagement
	// data, just without selection bias detection.
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(120),
			DBP14dMean:       ptrFloat(75),
			ClinicSBPMean:    ptrFloat(118),
			ClinicDBPMean:    ptrFloat(74),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{err: errSimulated()}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify should tolerate KB-21 outage, got %v", err)
	}
	if result.Phenotype != models.PhenotypeSustainedNormotension {
		t.Errorf("expected SUSTAINED_NORMOTENSION, got %s", result.Phenotype)
	}
	if result.SelectionBiasRisk {
		t.Error("expected no selection bias when KB-21 unavailable")
	}
}

func TestBPContextOrchestrator_PatientNotFound(t *testing.T) {
	kb20 := &stubKB20Client{profile: nil}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	_, err := orch.Classify(context.Background(), "ghost")
	if err == nil {
		t.Error("expected error for unknown patient")
	}
}

// Local error helper.
func errSimulated() error {
	return &simulatedErr{msg: "simulated outage"}
}

type simulatedErr struct{ msg string }

func (e *simulatedErr) Error() string { return e.msg }
