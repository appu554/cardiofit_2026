package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// Test 1: PM evaluation cascades to MD nodes via SignalCascade.
// Flow: PM-01 (SEVERELY_ABOVE) → cascade triggers MD-02 + MD-06.
// ---------------------------------------------------------------------------

func TestIntegration_PMEvaluation_CascadesToMD(t *testing.T) {
	// --- Setup PM-01 node: Home BP with SEVERELY_ABOVE classification ---
	pm01 := &models.MonitoringNodeDefinition{
		NodeID:  "PM-01",
		Version: "1.0.0",
		Type:    "MONITORING",
		TitleEN: "Home BP Monitoring",
		ComputedFields: []models.ComputedFieldDef{
			{Name: "sbp_delta", Formula: "sbp_home_mean - bp_target_sbp"},
		},
		Classifications: []models.ClassificationDef{
			{Category: "HYPOTENSIVE", Condition: "sbp_home_mean < 90", Severity: "CRITICAL", MCUGateSuggestion: "PAUSE"},
			{Category: "SEVERELY_ABOVE", Condition: "sbp_delta > 30", Severity: "CRITICAL", MCUGateSuggestion: "PAUSE"},
			{Category: "ABOVE_TARGET", Condition: "sbp_delta > 10", Severity: "MODERATE", MCUGateSuggestion: "MODIFY"},
			{Category: "AT_TARGET", Condition: "", Severity: "NONE", MCUGateSuggestion: "SAFE"},
		},
		InsufficientData: models.InsufficientDataPolicy{Action: "SKIP"},
		CascadeTo:        []string{"MD-02", "MD-06"},
	}

	// Mock resolver: sbp=165, target=130 → delta=35 → SEVERELY_ABOVE (delta > 30)
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_home_mean": 165.0,
				"dbp_home_mean": 95.0,
				"bp_target_sbp": 130.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	// Build monitoring engine with PM-01
	monLoader := NewMonitoringNodeLoader("", testLogger())
	monLoader.nodes = map[string]*models.MonitoringNodeDefinition{"PM-01": pm01}

	monEngine := NewMonitoringNodeEngine(monLoader, resolver, NewExpressionEvaluator(), NewTrajectoryComputer(testLogger()), nil, testLogger())

	// Evaluate PM-01
	ctx := context.Background()
	pmEvent, err := monEngine.Evaluate(ctx, "PM-01", "patient-int01", "CKD_G3a_DM")
	if err != nil {
		t.Fatalf("PM-01 Evaluate() error: %v", err)
	}
	if pmEvent == nil {
		t.Fatal("PM-01 expected non-nil event")
	}
	if pmEvent.Classification == nil {
		t.Fatal("PM-01 expected classification")
	}
	if pmEvent.Classification.Category != "SEVERELY_ABOVE" {
		t.Errorf("PM-01 category: got %q, want SEVERELY_ABOVE", pmEvent.Classification.Category)
	}

	// --- Setup cascade with stub MD evaluator ---
	// MD-02 and MD-06 are targets from PM-01.cascade_to
	md02 := &models.DeteriorationNodeDefinition{
		NodeID: "MD-02", Version: "1.0.0", Type: "DETERIORATION",
		Thresholds: []models.ThresholdDef{{Signal: "VR_STABLE", Condition: "", Severity: "NONE", MCUGateSuggestion: "SAFE"}},
		InsufficientData: models.InsufficientDataPolicy{Action: "SKIP"},
	}
	md06 := &models.DeteriorationNodeDefinition{
		NodeID: "MD-06", Version: "1.0.0", Type: "DETERIORATION",
		ContributingSignals: []string{"MD-02"},
		Thresholds:          []models.ThresholdDef{{Signal: "CV_RISK_LOW", Condition: "", Severity: "NONE", MCUGateSuggestion: "SAFE"}},
		InsufficientData:    models.InsufficientDataPolicy{Action: "SKIP"},
	}

	deterLoader := NewDeteriorationNodeLoader("", testLogger())
	deterLoader.nodes = map[string]*models.DeteriorationNodeDefinition{
		"MD-02": md02,
		"MD-06": md06,
	}

	// Use stub evaluator that returns events for both MD nodes
	stub := newStubEvaluator()
	stub.setResponse("MD-02", buildMDEvent("MD-02", "MODERATE"), nil)
	stub.setResponse("MD-06", buildMDEvent("MD-06", "NONE"), nil)

	cascade := NewSignalCascade(monLoader, deterLoader, stub, testLogger())

	// Trigger cascade from PM-01 with CRITICAL severity (score=3)
	mdEvents := cascade.Trigger(ctx, "PM-01", "patient-int01", "CKD_G3a_DM", 3.0)

	// Assert: cascade evaluated MD-02, and since MD-02 is in MD-06.contributing_signals, also MD-06
	calledNodes := stub.calledNodes()
	if !sliceContains(calledNodes, "MD-02") {
		t.Errorf("expected MD-02 to be called, got %v", calledNodes)
	}
	if !sliceContains(calledNodes, "MD-06") {
		t.Errorf("expected MD-06 to be called (MD-02 is contributor), got %v", calledNodes)
	}
	if len(mdEvents) < 2 {
		t.Errorf("expected >=2 cascade events, got %d", len(mdEvents))
	}

	// Assert: CascadeContext.PMSignals has pm_01 severity score
	call, ok := stub.callFor("MD-02")
	if !ok {
		t.Fatal("no call recorded for MD-02")
	}
	if call.cascadeCtx == nil {
		t.Fatal("cascade context is nil for MD-02")
	}
	if score, exists := call.cascadeCtx.PMSignals["pm_01"]; !exists || score != 3.0 {
		t.Errorf("PMSignals[pm_01]: got %v, want 3.0", call.cascadeCtx.PMSignals)
	}
}

// ---------------------------------------------------------------------------
// Test 2: Safety trigger fires and produces SafetyFlags on the event.
// ---------------------------------------------------------------------------

func TestIntegration_SafetyTriggerProducesSafetyFlags(t *testing.T) {
	pm01 := &models.MonitoringNodeDefinition{
		NodeID:  "PM-01",
		Version: "1.0.0",
		Type:    "MONITORING",
		TitleEN: "Home BP",
		ComputedFields: []models.ComputedFieldDef{
			{Name: "sbp_delta", Formula: "sbp_home_mean - bp_target_sbp"},
		},
		Classifications: []models.ClassificationDef{
			{Category: "SEVERELY_ABOVE", Condition: "sbp_delta > 30", Severity: "CRITICAL", MCUGateSuggestion: "PAUSE"},
			{Category: "AT_TARGET", Condition: "", Severity: "NONE", MCUGateSuggestion: "SAFE"},
		},
		SafetyTriggers: []models.MonitoringSafetyTrigger{
			{
				ID:        "PM01_ST01",
				Condition: "sbp_home_mean > 180",
				Severity:  "IMMEDIATE",
				Action:    "Hypertensive crisis. Urgent physician review required.",
			},
		},
		InsufficientData: models.InsufficientDataPolicy{Action: "SKIP"},
	}

	// SBP=185 → triggers safety (>180)
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_home_mean": 185.0,
				"bp_target_sbp": 130.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	monLoader := NewMonitoringNodeLoader("", testLogger())
	monLoader.nodes = map[string]*models.MonitoringNodeDefinition{"PM-01": pm01}

	engine := NewMonitoringNodeEngine(monLoader, resolver, NewExpressionEvaluator(), NewTrajectoryComputer(testLogger()), nil, testLogger())

	event, err := engine.Evaluate(context.Background(), "PM-01", "patient-int02", "CKD_G3a_DM")
	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	// Assert: safety flag present with IMMEDIATE severity
	if len(event.SafetyFlags) == 0 {
		t.Fatal("expected safety flags, got none")
	}
	found := false
	for _, sf := range event.SafetyFlags {
		if sf.FlagID == "PM01_ST01" && sf.Severity == "IMMEDIATE" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected PM01_ST01 IMMEDIATE flag, got %+v", event.SafetyFlags)
	}
}

// ---------------------------------------------------------------------------
// Test 3: MD-06 HALT gate flows through cascade.
// ---------------------------------------------------------------------------

func TestIntegration_MD06HaltGateViaCascade(t *testing.T) {
	// PM-01 cascades to MD-02; MD-06 has MD-02 as contributing signal
	pm01 := &models.MonitoringNodeDefinition{
		NodeID: "PM-01", Version: "1.0.0", Type: "MONITORING",
		Classifications: []models.ClassificationDef{
			{Category: "ABOVE_TARGET", Condition: "", Severity: "MODERATE", MCUGateSuggestion: "MODIFY"},
		},
		InsufficientData: models.InsufficientDataPolicy{Action: "SKIP"},
		CascadeTo:        []string{"MD-02"},
	}

	md02 := &models.DeteriorationNodeDefinition{
		NodeID: "MD-02", Version: "1.0.0", Type: "DETERIORATION",
		Thresholds:      []models.ThresholdDef{{Signal: "VR_CRITICAL_RISE", Condition: "", Severity: "CRITICAL", MCUGateSuggestion: "PAUSE"}},
		InsufficientData: models.InsufficientDataPolicy{Action: "SKIP"},
	}
	md06 := &models.DeteriorationNodeDefinition{
		NodeID: "MD-06", Version: "1.0.0", Type: "DETERIORATION",
		ContributingSignals: []string{"MD-02"},
		Thresholds:          []models.ThresholdDef{{Signal: "CV_RISK_CRITICAL", Condition: "", Severity: "CRITICAL", MCUGateSuggestion: "HALT"}},
		InsufficientData:    models.InsufficientDataPolicy{Action: "SKIP"},
	}

	monLoader := NewMonitoringNodeLoader("", testLogger())
	monLoader.nodes = map[string]*models.MonitoringNodeDefinition{"PM-01": pm01}
	deterLoader := NewDeteriorationNodeLoader("", testLogger())
	deterLoader.nodes = map[string]*models.DeteriorationNodeDefinition{"MD-02": md02, "MD-06": md06}

	// Stub: MD-02 fires with CRITICAL, MD-06 fires with HALT
	stub := newStubEvaluator()
	stub.setResponse("MD-02", buildMDEvent("MD-02", "CRITICAL"), nil)
	haltGate := "HALT"
	md06Event := &models.ClinicalSignalEvent{
		EventID:   "evt-md06-halt",
		NodeID:    "MD-06",
		SignalType: models.SignalDeteriorationSignal,
		DeteriorationSignal: &models.DeteriorationResult{
			Signal:   "CV_RISK_CRITICAL",
			Severity: "CRITICAL",
		},
		MCUGateSuggestion: &haltGate,
	}
	stub.setResponse("MD-06", md06Event, nil)

	cascade := NewSignalCascade(monLoader, deterLoader, stub, testLogger())
	events := cascade.Trigger(context.Background(), "PM-01", "patient-int03", "CKD_G3a_DM", 2.0)

	// Assert: MD-06 event has HALT gate
	var foundHalt bool
	for _, evt := range events {
		if evt.NodeID == "MD-06" && evt.MCUGateSuggestion != nil && *evt.MCUGateSuggestion == "HALT" {
			foundHalt = true
		}
	}
	if !foundHalt {
		t.Errorf("expected MD-06 HALT event in cascade results, got %d events", len(events))
	}
}

// ---------------------------------------------------------------------------
// Test 4: Insufficient data triggers FLAG_FOR_REVIEW event.
// ---------------------------------------------------------------------------

func TestIntegration_InsufficientData_FlagForReview(t *testing.T) {
	pm03 := testPM03Node() // Uses FLAG_FOR_REVIEW policy

	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields:      map[string]float64{},
			Sufficiency: models.DataInsufficient,
		},
	}

	monLoader := NewMonitoringNodeLoader("", testLogger())
	monLoader.nodes = map[string]*models.MonitoringNodeDefinition{"PM-03": pm03}

	engine := NewMonitoringNodeEngine(monLoader, resolver, NewExpressionEvaluator(), NewTrajectoryComputer(testLogger()), nil, testLogger())

	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-int04", "CKD_G3a_DM")
	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}

	// FLAG_FOR_REVIEW should still produce an event (not nil)
	if event == nil {
		t.Fatal("FLAG_FOR_REVIEW should produce a non-nil event")
	}
}

// ---------------------------------------------------------------------------
// Test 5: SignalPublisher delivers event to KB-23 mock and Kafka mock.
// ---------------------------------------------------------------------------

func TestIntegration_PublisherDeliversToKB23AndKafka(t *testing.T) {
	// KB-23 mock returns 201
	var receivedBody []byte
	kb23 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/clinical-signals" {
			t.Errorf("expected /api/v1/clinical-signals, got %s", r.URL.Path)
		}
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		receivedBody = buf[:n]
		w.WriteHeader(http.StatusCreated)
	}))
	defer kb23.Close()

	kafka := &mockKafkaPublisher{}
	publisher := newTestSignalPublisher(kb23.URL, kafka, 1, time.Millisecond)

	event := sampleSignalEvent()
	err := publisher.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	// Assert: KB-23 received the event
	if len(receivedBody) == 0 {
		t.Fatal("KB-23 mock received empty body")
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("KB-23 body not valid JSON: %v", err)
	}
	if decoded["event_id"] != event.EventID {
		t.Errorf("event_id mismatch: got %v, want %s", decoded["event_id"], event.EventID)
	}

	// Assert: Kafka received the event with correct topic
	if len(kafka.calls) == 0 {
		t.Fatal("Kafka mock received no calls")
	}
	if kafka.calls[0].topic != "clinical.signal.events" {
		t.Errorf("Kafka topic: got %q, want clinical.signal.events", kafka.calls[0].topic)
	}
	if kafka.calls[0].key != event.PatientID {
		t.Errorf("Kafka key: got %q, want %s", kafka.calls[0].key, event.PatientID)
	}
}

func sliceContains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
