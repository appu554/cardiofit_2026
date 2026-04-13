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
	return newOrchestratorWithPublisher(t, kb20, kb21, nil)
}

func newOrchestratorWithPublisher(t *testing.T, kb20 KB20Fetcher, kb21 KB21Fetcher, kb19 KB19EventPublisher) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t) // from bp_context_repository_test.go
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	return NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil, kb19)
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

// stubKB19Publisher implements KB19EventPublisher for tests.
type stubKB19Publisher struct {
	maskedHTNCalls       []string    // patientIDs for MASKED_HTN_DETECTED
	phenotypeChangeCalls [][3]string // {patientID, old, new}
}

func (s *stubKB19Publisher) PublishMaskedHTNDetected(ctx context.Context, patientID, phenotype, urgency string) error {
	s.maskedHTNCalls = append(s.maskedHTNCalls, patientID)
	return nil
}

func (s *stubKB19Publisher) PublishPhenotypeChanged(ctx context.Context, patientID, oldPhenotype, newPhenotype string) error {
	s.phenotypeChangeCalls = append(s.phenotypeChangeCalls, [3]string{patientID, oldPhenotype, newPhenotype})
	return nil
}

func TestBPContextOrchestrator_NewMaskedHTN_PublishesDetectedEvent(t *testing.T) {
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
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19)

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("classify: %v", err)
	}

	if len(kb19.maskedHTNCalls) != 1 {
		t.Errorf("expected 1 MASKED_HTN_DETECTED call, got %d", len(kb19.maskedHTNCalls))
	}
	if len(kb19.phenotypeChangeCalls) != 0 {
		t.Errorf("first detection should not emit BP_PHENOTYPE_CHANGED, got %d", len(kb19.phenotypeChangeCalls))
	}
}

func TestBPContextOrchestrator_PhenotypeUnchanged_NoEvent(t *testing.T) {
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
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19)

	// First classification — emits MASKED_HTN_DETECTED
	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("first classify: %v", err)
	}
	// Second classification of identical state — should not emit anything new
	kb19.maskedHTNCalls = nil
	kb19.phenotypeChangeCalls = nil
	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("second classify: %v", err)
	}
	if len(kb19.maskedHTNCalls) != 0 {
		t.Errorf("expected 0 MASKED_HTN_DETECTED on re-classify, got %d", len(kb19.maskedHTNCalls))
	}
	if len(kb19.phenotypeChangeCalls) != 0 {
		t.Errorf("expected 0 BP_PHENOTYPE_CHANGED on re-classify, got %d", len(kb19.phenotypeChangeCalls))
	}
}

func TestBPContextOrchestrator_PhenotypeChanged_PublishesTransition(t *testing.T) {
	// First call: home reading high (148) -> MASKED_HTN
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
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19)

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("first classify: %v", err)
	}

	// Second call: home reading dropped to normal (120) -> SUSTAINED_NORMOTENSION
	kb20.profile.SBP14dMean = ptrFloat(120)
	kb20.profile.DBP14dMean = ptrFloat(75)
	kb19.maskedHTNCalls = nil
	kb19.phenotypeChangeCalls = nil

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("second classify: %v", err)
	}

	if len(kb19.phenotypeChangeCalls) != 1 {
		t.Fatalf("expected 1 BP_PHENOTYPE_CHANGED, got %d", len(kb19.phenotypeChangeCalls))
	}
	got := kb19.phenotypeChangeCalls[0]
	if got[1] != "MASKED_HTN" || got[2] != "SUSTAINED_NORMOTENSION" {
		t.Errorf("expected MH->SN transition, got %v", got)
	}
}

func TestBPContextOrchestrator_NilPublisher_NoEventsNoErrors(t *testing.T) {
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
		},
	}
	kb21 := &stubKB21Client{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, nil) // explicit nil publisher

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("classify with nil publisher should not error: %v", err)
	}
}
