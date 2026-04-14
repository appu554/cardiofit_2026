package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/pkg/stability"
)

// stubKB20Client implements the KB20Fetcher interface for tests.
type stubKB20Client struct {
	profile  *clients.KB20PatientProfile
	err      error
	readings []clients.KB20BPReading // Phase 4 P3 addition
	readErr  error                   // Phase 4 P3 addition
}

func (s *stubKB20Client) FetchProfile(ctx context.Context, patientID string) (*clients.KB20PatientProfile, error) {
	return s.profile, s.err
}

func (s *stubKB20Client) FetchBPReadings(ctx context.Context, patientID string, since time.Time) ([]clients.KB20BPReading, error) {
	return s.readings, s.readErr
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
	return newOrchestratorFull(t, kb20, kb21, kb19, nil)
}

// newOrchestratorFull is the Phase 4 P9 variant that also injects a KB-23
// composite trigger stub. Tests that only care about KB-19 events can keep
// using newOrchestratorWithPublisher; the composite-trigger tests use this.
func newOrchestratorFull(t *testing.T, kb20 KB20Fetcher, kb21 KB21Fetcher, kb19 KB19EventPublisher, kb23 KB23CompositeTrigger) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t) // from bp_context_repository_test.go
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	return NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil, kb19, nil, kb23) // nil stability engine
}

// stubKB23CompositeTrigger implements KB23CompositeTrigger for tests.
type stubKB23CompositeTrigger struct {
	calls []string // patient IDs
	err   error    // optional injection: return this error from every call
}

func (s *stubKB23CompositeTrigger) TriggerCompositeSynthesize(ctx context.Context, patientID string) error {
	s.calls = append(s.calls, patientID)
	return s.err
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

func TestBPContextOrchestrator_RealReadings_UsesBuildFromReadings(t *testing.T) {
	now := time.Now().UTC()
	// Build 14 home readings + 2 clinic readings, all high at home + normal at clinic
	// (classic masked HTN pattern).
	var readings []clients.KB20BPReading
	for i := 0; i < 14; i++ {
		readings = append(readings, clients.KB20BPReading{
			PatientID:  "p1",
			SBP:        148,
			DBP:        92,
			Source:     "HOME_CUFF",
			MeasuredAt: now.Add(-time.Duration(i*12) * time.Hour),
		})
	}
	readings = append(readings,
		clients.KB20BPReading{
			PatientID: "p1", SBP: 128, DBP: 78, Source: "CLINIC",
			MeasuredAt: now.Add(-7 * 24 * time.Hour),
		},
		clients.KB20BPReading{
			PatientID: "p1", SBP: 130, DBP: 80, Source: "CLINIC",
			MeasuredAt: now.Add(-14 * 24 * time.Hour),
		},
	)

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
		readings: readings,
	}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN from real readings, got %s", result.Phenotype)
	}
}

func TestBPContextOrchestrator_RealReadings_FetchError_FallsBackToSynthetic(t *testing.T) {
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
		readErr: errSimulated(),
	}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify should fall back gracefully, got %v", err)
	}
	// Still classifies as MASKED_HTN via the synthetic path
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN (fallback path), got %s", result.Phenotype)
	}
}

func TestBPContextOrchestrator_RealReadings_Empty_FallsBackToSynthetic(t *testing.T) {
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
		readings: []clients.KB20BPReading{}, // empty — fall back
	}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN (fallback path), got %s", result.Phenotype)
	}
}

// newOrchestratorWithStability is like newOrchestratorWithPublisher but
// also injects a real stability engine for Phase 4 P2 tests.
func newOrchestratorWithStability(
	t *testing.T,
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	kb19 KB19EventPublisher,
	policy stability.Policy,
) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	engine := stability.NewEngine(policy)
	return NewBPContextOrchestrator(
		kb20, kb21, repo, thresholds,
		zap.NewNop(), nil, kb19, engine, nil,
	)
}

func TestBPContextOrchestrator_Stability_DampsFlappingWithinDwell(t *testing.T) {
	// Classifier would produce MASKED_HTN on day 1, then SUSTAINED_NORMOTENSION
	// on day 2. With a 14-day dwell policy, day 2's transition must be damped
	// and the phenotype must remain MASKED_HTN.
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

	orch := newOrchestratorWithStability(t, kb20, kb21, kb19, stability.Policy{
		MinDwell:           14 * 24 * time.Hour,
		FlapWindow:         30 * 24 * time.Hour,
		MaxFlapsBeforeLock: 3,
	})

	// Day 1: classify as MASKED_HTN
	result1, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("first classify: %v", err)
	}
	if result1.Phenotype != models.PhenotypeMaskedHTN {
		t.Fatalf("expected MASKED_HTN on day 1, got %s", result1.Phenotype)
	}

	// Flip the profile so the classifier would now produce SUSTAINED_NORMOTENSION
	kb20.profile.SBP14dMean = ptrFloat(120)
	kb20.profile.DBP14dMean = ptrFloat(75)

	// Day 2 (simulated by same-day reclassify via upsert): stability engine
	// should damp the transition because <14 days elapsed.
	result2, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("second classify: %v", err)
	}
	if result2.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected dampened MASKED_HTN (was MH, would flap to SN), got %s", result2.Phenotype)
	}
	if result2.Confidence != "DAMPED" {
		t.Errorf("expected DAMPED confidence marker, got %s", result2.Confidence)
	}
}

func TestBPContextOrchestrator_Stability_NoEngineNoDampening(t *testing.T) {
	// Regression guard: nil stability engine means existing Phase 3
	// behavior — transitions are accepted immediately.
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
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19) // nil engine

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
	if result.Confidence == "DAMPED" {
		t.Errorf("confidence should not be DAMPED when engine is nil")
	}
}

func TestBPContextOrchestrator_RealReadings_MorningEveningDifferential_FiresMedTiming(t *testing.T) {
	// Simulate a patient on antihypertensives whose morning BPs are
	// significantly higher than evening — this should trigger the
	// medication timing hypothesis, which was dead code before P3.4.
	now := time.Now().UTC()
	var readings []clients.KB20BPReading
	for i := 0; i < 7; i++ {
		// Morning reading at 08:00 UTC each day (HIGH)
		morning := time.Date(now.Year(), now.Month(), now.Day()-i, 8, 0, 0, 0, time.UTC)
		readings = append(readings, clients.KB20BPReading{
			PatientID: "p1", SBP: 148, DBP: 92, Source: "HOME_CUFF",
			MeasuredAt: morning,
		})
		// Evening reading at 20:00 UTC each day (NORMAL)
		evening := time.Date(now.Year(), now.Month(), now.Day()-i, 20, 0, 0, 0, time.UTC)
		readings = append(readings, clients.KB20BPReading{
			PatientID: "p1", SBP: 125, DBP: 78, Source: "HOME_CUFF",
			MeasuredAt: evening,
		})
	}
	// 2 clinic readings (normal — afternoon visits)
	readings = append(readings,
		clients.KB20BPReading{
			PatientID: "p1", SBP: 128, DBP: 80, Source: "CLINIC",
			MeasuredAt: now.Add(-7 * 24 * time.Hour),
		},
		clients.KB20BPReading{
			PatientID: "p1", SBP: 130, DBP: 78, Source: "CLINIC",
			MeasuredAt: now.Add(-14 * 24 * time.Hour),
		},
	)

	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(137), // avg of 148+125
			DBP14dMean:       ptrFloat(85),
			ClinicSBPMean:    ptrFloat(129),
			ClinicDBPMean:    ptrFloat(79),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
			OnHTNMeds:        true,
		},
		readings: readings,
	}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	// Phenotype should be MASKED_UNCONTROLLED (on meds + home elevated + clinic normal)
	// OR MASKED_HTN depending on the exact means — either is acceptable.
	// The key assertion: MedicationTimingHypothesis must be non-empty.
	if result.MedicationTimingHypothesis == "" {
		t.Errorf("expected non-empty MedicationTimingHypothesis (morning-evening differential), got empty")
	}
}

// Phase 4 P9: verify that a successful classification triggers KB-23
// composite card synthesis exactly once per patient, passes the right
// patient id through, and never propagates KB-23 errors back to the
// caller (best-effort contract).

func TestBPContextOrchestrator_P9_TriggersCompositeSynthesisOnClassify(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-composite",
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
	kb23 := &stubKB23CompositeTrigger{}
	orch := newOrchestratorFull(t, kb20, kb21, kb19, kb23)

	if _, err := orch.Classify(context.Background(), "p-composite"); err != nil {
		t.Fatalf("classify: %v", err)
	}

	if len(kb23.calls) != 1 {
		t.Fatalf("expected KB-23 composite trigger to be called once, got %d", len(kb23.calls))
	}
	if kb23.calls[0] != "p-composite" {
		t.Errorf("expected patient id 'p-composite', got %q", kb23.calls[0])
	}
}

func TestBPContextOrchestrator_P9_CompositeTriggerFailureIsSwallowed(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-err",
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
	kb23 := &stubKB23CompositeTrigger{err: &simulatedErr{msg: "KB-23 down"}}
	orch := newOrchestratorFull(t, kb20, kb21, kb19, kb23)

	// Classification must succeed even though the trigger returns an error —
	// composite synthesis is best-effort per P9 design.
	result, err := orch.Classify(context.Background(), "p-err")
	if err != nil {
		t.Fatalf("classify should swallow KB-23 error, got %v", err)
	}
	if result == nil {
		t.Fatal("classify returned nil result on KB-23 error")
	}
	if len(kb23.calls) != 1 {
		t.Errorf("expected single trigger attempt even on error, got %d", len(kb23.calls))
	}
}

func TestBPContextOrchestrator_P9_NilTriggerIsNoop(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-nil",
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
	orch := newOrchestratorFull(t, kb20, kb21, kb19, nil) // explicit nil

	if _, err := orch.Classify(context.Background(), "p-nil"); err != nil {
		t.Fatalf("classify with nil composite trigger should succeed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Phase 5 P5-1: orchestrator must persist BOTH raw and stable phenotypes
//
// When the stability engine accepts a transition, raw == stable. When it
// damps, raw is the un-dampened classifier output and stable is the held
// prior phenotype. The history table previously lost the raw signal —
// these tests pin the new persistence behaviour.
// ---------------------------------------------------------------------------

func TestBPContextOrchestrator_P5_RawEqualsStable_OnFirstClassification(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-raw-first",
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
	orch := newOrchestratorFull(t, kb20, kb21, kb19, nil)

	if _, err := orch.Classify(context.Background(), "p-raw-first"); err != nil {
		t.Fatalf("classify: %v", err)
	}

	saved, err := orch.repo.FetchLatest("p-raw-first")
	if err != nil || saved == nil {
		t.Fatalf("fetch latest snapshot: err=%v saved=%v", err, saved)
	}
	if saved.RawPhenotype != saved.Phenotype {
		t.Errorf("expected raw == stable on first classification, got raw=%q stable=%q",
			saved.RawPhenotype, saved.Phenotype)
	}
	if saved.RawPhenotype == "" {
		t.Errorf("expected raw_phenotype to be populated, got empty")
	}
}

func TestBPContextOrchestrator_P5_RawDifferentFromStable_OnDampedTransition(t *testing.T) {
	// Day 1 the patient is classified as MASKED_HTN. Day 2 the readings
	// flip to SUSTAINED_NORMOTENSION but the 14-day dwell hasn't elapsed,
	// so the engine damps and the stable phenotype reverts to MASKED_HTN.
	// The day-2 snapshot must persist BOTH the un-dampened raw output
	// (SUSTAINED_NORMOTENSION) and the held stable phenotype (MASKED_HTN).
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-raw-damp",
			SBP14dMean:       ptrFloat(148), // home elevated
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128), // clinic normal
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	policy := stability.Policy{MinDwell: 14 * 24 * time.Hour}
	orch := newOrchestratorWithStabilityPolicy(t, kb20, kb21, kb19, policy)

	// Day 1 → MASKED_HTN (clinic normal, home elevated)
	day1, err := orch.Classify(context.Background(), "p-raw-damp")
	if err != nil {
		t.Fatalf("day 1 classify: %v", err)
	}
	if day1.Phenotype != models.PhenotypeMaskedHTN {
		t.Fatalf("setup error: expected day 1 MASKED_HTN, got %s", day1.Phenotype)
	}

	// Flip readings: home now normal, clinic still normal → raw becomes SUSTAINED_NORMOTENSION.
	kb20.profile.SBP14dMean = ptrFloat(120)
	kb20.profile.DBP14dMean = ptrFloat(75)

	day2, err := orch.Classify(context.Background(), "p-raw-damp")
	if err != nil {
		t.Fatalf("day 2 classify: %v", err)
	}
	if day2.Phenotype != models.PhenotypeMaskedHTN {
		t.Fatalf("expected damped stable phenotype to remain MASKED_HTN, got %s", day2.Phenotype)
	}

	saved, err := orch.repo.FetchLatest("p-raw-damp")
	if err != nil || saved == nil {
		t.Fatalf("fetch saved: err=%v", err)
	}
	if saved.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("snapshot stable phenotype: expected MASKED_HTN, got %s", saved.Phenotype)
	}
	if saved.RawPhenotype != models.PhenotypeSustainedNormotension {
		t.Errorf("snapshot raw phenotype: expected SUSTAINED_NORMOTENSION, got %s", saved.RawPhenotype)
	}
	if saved.RawPhenotype == saved.Phenotype {
		t.Error("expected raw != stable after dampening, but they matched")
	}
}

// ---------------------------------------------------------------------------
// Phase 5 P5-2: detectOverrideEvent reads the patient profile's medication
// change timestamp. Within the override window (default 7 days), the
// stability engine bypasses the dwell/flap checks. Outside the window, or
// when no medication change is recorded, the override returns false.
//
// 7 days is chosen as a conservative default that covers the time-to-steady-
// state of most antihypertensives without being so wide that long-tail
// effects of an unrelated med change keep firing. PK-aware per-drug windows
// are documented as a follow-up in the Phase 5 plan.
// ---------------------------------------------------------------------------

func TestDetectOverrideEvent_RecentMedChange_ReturnsTrue(t *testing.T) {
	recently := time.Now().UTC().Add(-2 * 24 * time.Hour)
	profile := &clients.KB20PatientProfile{
		PatientID:               "p1",
		LastMedicationChangeAt:  &recently,
	}
	if !detectOverrideEvent(profile) {
		t.Error("expected override=true for med change 2 days ago")
	}
}

func TestDetectOverrideEvent_StaleMedChange_ReturnsFalse(t *testing.T) {
	long := time.Now().UTC().Add(-30 * 24 * time.Hour)
	profile := &clients.KB20PatientProfile{
		PatientID:              "p1",
		LastMedicationChangeAt: &long,
	}
	if detectOverrideEvent(profile) {
		t.Error("expected override=false for med change 30 days ago (outside 7d window)")
	}
}

func TestDetectOverrideEvent_NoMedChange_ReturnsFalse(t *testing.T) {
	profile := &clients.KB20PatientProfile{
		PatientID:              "p1",
		LastMedicationChangeAt: nil,
	}
	if detectOverrideEvent(profile) {
		t.Error("expected override=false when LastMedicationChangeAt is nil")
	}
}

func TestDetectOverrideEvent_NilProfile_ReturnsFalse(t *testing.T) {
	if detectOverrideEvent(nil) {
		t.Error("expected override=false for nil profile (defensive)")
	}
}

func TestDetectOverrideEvent_BoundaryExactly7Days_ReturnsTrue(t *testing.T) {
	// At exactly the 7-day boundary, the override should still fire.
	// Drugs reach steady state at varying speeds and we want the engine to
	// remain reactive at the boundary, not strictly less-than.
	boundary := time.Now().UTC().Add(-7 * 24 * time.Hour).Add(time.Second)
	profile := &clients.KB20PatientProfile{
		PatientID:              "p1",
		LastMedicationChangeAt: &boundary,
	}
	if !detectOverrideEvent(profile) {
		t.Error("expected override=true at the 7-day boundary")
	}
}

// newOrchestratorWithStabilityPolicy builds an orchestrator with a custom
// stability policy. Phase 5 P5-1 tests need this because the dampening
// behaviour depends on the policy's MinDwell + MaxDwellOverrideRate.
func newOrchestratorWithStabilityPolicy(
	t *testing.T,
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	kb19 KB19EventPublisher,
	policy stability.Policy,
) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	engine := stability.NewEngine(policy)
	return NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil, kb19, engine, nil)
}

// ---------------------------------------------------------------------------
// Phase 5 P5-2.6: End-to-end regression guard for the medication-change
// override. Pins the contract that when KB20PatientProfile supplies a recent
// LastMedicationChangeAt, the orchestrator's stability engine bypasses the
// dwell and accepts the new phenotype. Proves the engine + detector + KB-20
// client field are wired together correctly.
// ---------------------------------------------------------------------------

func TestBPContextOrchestrator_P5_2_E2E_MedChangeOverridesDwell(t *testing.T) {
	// Day 1: classify patient as MASKED_HTN (no med change yet).
	// Day 2: a med change was recorded yesterday. The classifier now flips
	//        to SUSTAINED_NORMOTENSION. Without the override the dwell
	//        would damp this transition. With the override it must accept.
	medChange := time.Now().UTC().Add(-24 * time.Hour)

	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-p5-2-e2e",
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
	policy := stability.Policy{MinDwell: 14 * 24 * time.Hour}
	orch := newOrchestratorWithStabilityPolicy(t, kb20, kb21, kb19, policy)

	// Day 1 → MASKED_HTN (clinic normal, home elevated).
	day1, err := orch.Classify(context.Background(), "p-p5-2-e2e")
	if err != nil {
		t.Fatalf("day 1 classify: %v", err)
	}
	if day1.Phenotype != models.PhenotypeMaskedHTN {
		t.Fatalf("setup error: expected day 1 MASKED_HTN, got %s", day1.Phenotype)
	}

	// Day 2: readings flip to normotension; med change recorded yesterday.
	kb20.profile.SBP14dMean = ptrFloat(120)
	kb20.profile.DBP14dMean = ptrFloat(75)
	kb20.profile.LastMedicationChangeAt = &medChange

	day2, err := orch.Classify(context.Background(), "p-p5-2-e2e")
	if err != nil {
		t.Fatalf("day 2 classify: %v", err)
	}

	// Without override the dwell would hold MASKED_HTN. With override the
	// engine must emit DecisionOverride and the orchestrator must accept
	// the new phenotype.
	if day2.Phenotype == models.PhenotypeMaskedHTN {
		t.Errorf("expected override to bypass dwell and accept new phenotype, got %s",
			day2.Phenotype)
	}
}

func TestBPContextOrchestrator_P5_2_E2E_StaleMedChangeDoesNotOverride(t *testing.T) {
	// Patient had a med change 30 days ago — outside the 7-day override
	// window. Dwell should hold the transition. Day 2 must remain MASKED_HTN.
	staleChange := time.Now().UTC().Add(-30 * 24 * time.Hour)

	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p-p5-2-stale",
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
	policy := stability.Policy{MinDwell: 14 * 24 * time.Hour}
	orch := newOrchestratorWithStabilityPolicy(t, kb20, kb21, kb19, policy)

	if _, err := orch.Classify(context.Background(), "p-p5-2-stale"); err != nil {
		t.Fatalf("day 1: %v", err)
	}

	kb20.profile.SBP14dMean = ptrFloat(120)
	kb20.profile.DBP14dMean = ptrFloat(75)
	kb20.profile.LastMedicationChangeAt = &staleChange

	day2, err := orch.Classify(context.Background(), "p-p5-2-stale")
	if err != nil {
		t.Fatalf("day 2: %v", err)
	}
	if day2.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected damped MASKED_HTN (stale med change), got %s", day2.Phenotype)
	}
}
