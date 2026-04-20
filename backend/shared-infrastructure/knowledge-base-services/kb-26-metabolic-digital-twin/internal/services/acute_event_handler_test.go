package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"kb-26-metabolic-digital-twin/internal/models"
)

// fakeLifecycleResolver captures calls for assertions in tests.
type fakeLifecycleResolver struct {
	mu    sync.Mutex
	calls []struct {
		PatientID     string
		DetectionType string
		Outcome       string
	}
}

func (f *fakeLifecycleResolver) ResolveLifecycle(_ context.Context, patientID, detectionType, outcome string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, struct {
		PatientID     string
		DetectionType string
		Outcome       string
	}{patientID, detectionType, outcome})
	return nil
}

func (f *fakeLifecycleResolver) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// makeTimestamps generates n timestamps within the last lookbackDays, spaced evenly.
func makeTimestamps(n int, lookbackDays int) []time.Time {
	now := time.Now().UTC()
	stamps := make([]time.Time, n)
	for i := 0; i < n; i++ {
		// Space evenly within the lookback window.
		hoursBack := float64(lookbackDays*24) * float64(n-1-i) / float64(n)
		stamps[i] = now.Add(-time.Duration(hoursBack) * time.Hour)
	}
	return stamps
}

// ---------------------------------------------------------------------------
// Test 1: TestHandler_EGFRBelowThreshold_CreatesEvent
// Baseline readings [42,40,43,41,38,42,40] (median~41), new reading 30 (27% drop)
// → AcuteEvent with EventType ACUTE_KIDNEY_INJURY, Severity HIGH
// ---------------------------------------------------------------------------

func TestHandler_EGFRBelowThreshold_CreatesEvent(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)

	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	ctx := DeviationContext{}
	now := time.Now().UTC()

	event, resolvedIDs := handler.HandleNewReading(
		"patient-001", "EGFR", 30.0, now,
		readings, timestamps,
		ctx,
		nil, // no recent deviations
		CompoundContext{},
		nil, // no active events
	)

	if event == nil {
		t.Fatal("expected an AcuteEvent, got nil")
	}
	if event.EventType != string(models.AcuteKidneyInjury) {
		t.Errorf("expected EventType %q, got %q", models.AcuteKidneyInjury, event.EventType)
	}
	if event.Severity != "HIGH" {
		t.Errorf("expected Severity HIGH, got %q", event.Severity)
	}
	if event.EscalationTier != "IMMEDIATE" {
		t.Errorf("expected EscalationTier IMMEDIATE, got %q", event.EscalationTier)
	}
	if event.PatientID != "patient-001" {
		t.Errorf("expected PatientID patient-001, got %q", event.PatientID)
	}
	if event.Direction != "BELOW_BASELINE" {
		t.Errorf("expected Direction BELOW_BASELINE, got %q", event.Direction)
	}
	if event.SuggestedAction == "" {
		t.Error("expected non-empty SuggestedAction")
	}
	if len(resolvedIDs) != 0 {
		t.Errorf("expected no resolved IDs, got %v", resolvedIDs)
	}
}

// ---------------------------------------------------------------------------
// Test 2: TestHandler_NormalReading_NoEvent
// Baseline median ~41, new reading 39 (5% drop) → nil (below threshold)
// ---------------------------------------------------------------------------

func TestHandler_NormalReading_NoEvent(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)

	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	ctx := DeviationContext{}
	now := time.Now().UTC()

	event, resolvedIDs := handler.HandleNewReading(
		"patient-001", "EGFR", 39.0, now,
		readings, timestamps,
		ctx,
		nil,
		CompoundContext{},
		nil,
	)

	if event != nil {
		t.Errorf("expected nil event for normal reading, got EventType=%q Severity=%q",
			event.EventType, event.Severity)
	}
	if len(resolvedIDs) != 0 {
		t.Errorf("expected no resolved IDs, got %v", resolvedIDs)
	}
}

// ---------------------------------------------------------------------------
// Test 3: TestHandler_CompoundTriggered
// Recent SBP drop deviation + new eGFR drop → compound CARDIORENAL_SYNDROME
// ---------------------------------------------------------------------------

func TestHandler_CompoundTriggered(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)

	// eGFR baseline readings and a significant drop.
	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	now := time.Now().UTC()
	ctx := DeviationContext{}

	// Simulate recent SBP drop deviation (from last 72h).
	recentDeviations := []models.DeviationResult{
		{
			VitalSignType:        "SBP",
			CurrentValue:         100.0,
			BaselineMedian:       130.0,
			DeviationAbsolute:    30.0,
			DeviationPercent:     23.08,
			Direction:            "BELOW_BASELINE",
			ClinicalSignificance: "HIGH",
		},
	}

	event, _ := handler.HandleNewReading(
		"patient-002", "EGFR", 30.0, now,
		readings, timestamps,
		ctx,
		recentDeviations,
		CompoundContext{},
		nil,
	)

	if event == nil {
		t.Fatal("expected compound AcuteEvent, got nil")
	}
	if event.EventType != string(models.AcuteCompoundCardiorenal) {
		t.Errorf("expected EventType %q, got %q", models.AcuteCompoundCardiorenal, event.EventType)
	}
	if event.CompoundPattern != "CARDIORENAL_SYNDROME" {
		t.Errorf("expected CompoundPattern CARDIORENAL_SYNDROME, got %q", event.CompoundPattern)
	}
	// Compound severity should be escalated from the base.
	if event.Severity == "" {
		t.Error("expected non-empty Severity for compound event")
	}
}

// ---------------------------------------------------------------------------
// Test 4: TestHandler_Resolution_MarksResolved
// Active event for same vital type + recovery reading (within 10% of baseline)
// → handler returns the resolved event ID
// ---------------------------------------------------------------------------

func TestHandler_Resolution_MarksResolved(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)

	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	now := time.Now().UTC()
	ctx := DeviationContext{}

	// Simulate an active event that should be resolved.
	activeEventID := uuid.New()
	activeEvents := []models.AcuteEvent{
		{
			ID:            activeEventID,
			PatientID:     "patient-003",
			EventType:     string(models.AcuteKidneyInjury),
			VitalSignType: "EGFR",
			Severity:      "HIGH",
			ResolvedAt:    nil,
		},
	}

	// Recovery reading: 40 is within 10% of baseline median ~41.
	event, resolvedIDs := handler.HandleNewReading(
		"patient-003", "EGFR", 40.0, now,
		readings, timestamps,
		ctx,
		nil,
		CompoundContext{},
		activeEvents,
	)

	// No new event since reading is back to normal.
	if event != nil {
		t.Errorf("expected nil event for recovery reading, got EventType=%q", event.EventType)
	}

	if len(resolvedIDs) != 1 {
		t.Fatalf("expected 1 resolved ID, got %d", len(resolvedIDs))
	}
	if resolvedIDs[0] != activeEventID.String() {
		t.Errorf("expected resolved ID %q, got %q", activeEventID.String(), resolvedIDs[0])
	}
}

// TestHandler_Resolution_BridgesT4ToKB23 — when an acute event resolves,
// the handler must call LifecycleResolver.ResolveLifecycle so KB-23 can
// close the Gap 19 T4 outcome lifecycle. Without this bridge the pilot
// metrics report "actions taken" but never "outcomes observed."
func TestHandler_Resolution_BridgesT4ToKB23(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)
	fake := &fakeLifecycleResolver{}
	handler.SetLifecycleResolver(fake)

	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	now := time.Now().UTC()
	activeEvents := []models.AcuteEvent{
		{
			ID:            uuid.New(),
			PatientID:     "patient-t4",
			EventType:     string(models.AcuteKidneyInjury),
			VitalSignType: "EGFR",
			Severity:      "HIGH",
			ResolvedAt:    nil,
		},
	}

	_, resolvedIDs := handler.HandleNewReading(
		"patient-t4", "EGFR", 40.0, now,
		readings, timestamps,
		DeviationContext{},
		nil, CompoundContext{}, activeEvents,
	)
	if len(resolvedIDs) != 1 {
		t.Fatalf("expected 1 resolved event, got %d", len(resolvedIDs))
	}

	// Bridge fires on a goroutine — give it a moment to run.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && fake.callCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if fake.callCount() != 1 {
		t.Fatalf("expected 1 ResolveLifecycle call, got %d", fake.callCount())
	}
	call := fake.calls[0]
	if call.PatientID != "patient-t4" {
		t.Errorf("wrong patient_id: %q", call.PatientID)
	}
	if call.DetectionType != "ACUTE_EVENT_EGFR" {
		t.Errorf("wrong detection_type: %q", call.DetectionType)
	}
}

// ---------------------------------------------------------------------------
// Test 5: TestHandler_CriticalSeverity_SafetyEscalation
// eGFR 35% drop (CRITICAL) → event.EscalationTier = "SAFETY"
// ---------------------------------------------------------------------------

func TestHandler_CriticalSeverity_SafetyEscalation(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)

	// Baseline median ~41. Reading of 26 → ~36.6% drop → CRITICAL.
	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	now := time.Now().UTC()
	ctx := DeviationContext{}

	event, _ := handler.HandleNewReading(
		"patient-004", "EGFR", 26.0, now,
		readings, timestamps,
		ctx,
		nil,
		CompoundContext{},
		nil,
	)

	if event == nil {
		t.Fatal("expected an AcuteEvent for critical drop, got nil")
	}
	if event.Severity != "CRITICAL" {
		t.Errorf("expected Severity CRITICAL, got %q", event.Severity)
	}
	if event.EscalationTier != "SAFETY" {
		t.Errorf("expected EscalationTier SAFETY, got %q", event.EscalationTier)
	}
}

// ---------------------------------------------------------------------------
// Test 6: TestHandler_ConfounderDampening_Applied
// eGFR 25% drop + active confounder → severity dampened, ConfounderDampened = true
// ---------------------------------------------------------------------------

func TestHandler_ConfounderDampening_Applied(t *testing.T) {
	handler := NewAcuteEventHandler(DefaultAcuteDetectionConfig(), nil, nil)

	readings := []float64{42, 40, 43, 41, 38, 42, 40}
	timestamps := makeTimestamps(len(readings), 7)
	now := time.Now().UTC()

	// Active confounder dampens severity by 1 level.
	ctx := DeviationContext{
		ActiveConfounderName: "STEROID_COURSE",
	}

	// Reading of 30 → ~27% drop → would be HIGH → dampened to MODERATE.
	event, _ := handler.HandleNewReading(
		"patient-005", "EGFR", 30.0, now,
		readings, timestamps,
		ctx,
		nil,
		CompoundContext{},
		nil,
	)

	if event == nil {
		t.Fatal("expected an AcuteEvent (dampened), got nil")
	}
	if !event.ConfounderDampened {
		t.Error("expected ConfounderDampened=true")
	}
	// HIGH dampened by 1 level → MODERATE.
	if event.Severity != "MODERATE" {
		t.Errorf("expected Severity MODERATE (dampened from HIGH), got %q", event.Severity)
	}
	if event.EscalationTier != "URGENT" {
		t.Errorf("expected EscalationTier URGENT (for MODERATE), got %q", event.EscalationTier)
	}
}
