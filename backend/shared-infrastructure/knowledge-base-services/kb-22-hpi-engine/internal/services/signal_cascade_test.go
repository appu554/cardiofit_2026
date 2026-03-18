package services

import (
	"context"
	"errors"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// stubDeteriorationEvaluator
// ---------------------------------------------------------------------------

// stubDeteriorationEvaluator is a controllable implementation of DeteriorationEvaluator.
// Each nodeID can be mapped to a fixed response (event + error).
type stubDeteriorationEvaluator struct {
	// responses maps nodeID → (event, err)
	responses map[string]stubResponse
	// calls records which nodeIDs were evaluated and with what CascadeContext.
	calls []stubCall
}

type stubResponse struct {
	event *models.ClinicalSignalEvent
	err   error
}

type stubCall struct {
	nodeID     string
	cascadeCtx *CascadeContext
}

func newStubEvaluator() *stubDeteriorationEvaluator {
	return &stubDeteriorationEvaluator{
		responses: make(map[string]stubResponse),
	}
}

func (s *stubDeteriorationEvaluator) setResponse(nodeID string, event *models.ClinicalSignalEvent, err error) {
	s.responses[nodeID] = stubResponse{event: event, err: err}
}

func (s *stubDeteriorationEvaluator) Evaluate(
	_ context.Context,
	nodeID, _, _ string,
	cascadeCtx *CascadeContext,
) (*models.ClinicalSignalEvent, error) {
	s.calls = append(s.calls, stubCall{nodeID: nodeID, cascadeCtx: cascadeCtx})
	resp, ok := s.responses[nodeID]
	if !ok {
		return nil, nil // default: node not found, no event
	}
	return resp.event, resp.err
}

func (s *stubDeteriorationEvaluator) calledNodes() []string {
	nodes := make([]string, len(s.calls))
	for i, c := range s.calls {
		nodes[i] = c.nodeID
	}
	return nodes
}

func (s *stubDeteriorationEvaluator) callFor(nodeID string) (stubCall, bool) {
	for _, c := range s.calls {
		if c.nodeID == nodeID {
			return c, true
		}
	}
	return stubCall{}, false
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// buildMDEvent creates a minimal MD ClinicalSignalEvent with the given severity.
func buildMDEvent(nodeID, severity string) *models.ClinicalSignalEvent {
	gate := "SAFE"
	return &models.ClinicalSignalEvent{
		EventID:   "test-event-" + nodeID,
		EventType: "CLINICAL_SIGNAL",
		SignalType: models.SignalDeteriorationSignal,
		NodeID:    nodeID,
		DeteriorationSignal: &models.DeteriorationResult{
			Signal:   "TEST_SIGNAL",
			Severity: severity,
		},
		MCUGateSuggestion: &gate,
	}
}

// buildPMEvent creates a minimal PM ClinicalSignalEvent.
// The Category field on Classification holds the severity string (matches how
// MonitoringNodeEngine stores data).
func buildPMEvent(nodeID, severityCategory string) *models.ClinicalSignalEvent {
	gate := "SAFE"
	return &models.ClinicalSignalEvent{
		EventID:   "test-pm-event-" + nodeID,
		EventType: "CLINICAL_SIGNAL",
		SignalType: models.SignalMonitoringClassification,
		NodeID:    nodeID,
		Classification: &models.ClassificationResult{
			Category:        severityCategory,
			DataSufficiency: string(models.DataSufficient),
		},
		MCUGateSuggestion: &gate,
	}
}

// buildCascade creates a SignalCascade using a pre-wired MonitoringNodeLoader,
// DeteriorationNodeLoader, and stub evaluator.
func buildCascade(
	pmNodes []*models.MonitoringNodeDefinition,
	mdNodes []*models.DeteriorationNodeDefinition,
	stub *stubDeteriorationEvaluator,
) *SignalCascade {
	log := testLogger()

	monLoader := NewMonitoringNodeLoader("", log)
	monLoader.mu.Lock()
	for _, n := range pmNodes {
		monLoader.nodes[n.NodeID] = n
	}
	monLoader.mu.Unlock()

	deterLoader := NewDeteriorationNodeLoader("", log)
	deterLoader.mu.Lock()
	for _, n := range mdNodes {
		deterLoader.nodes[n.NodeID] = n
	}
	deterLoader.mu.Unlock()

	return NewSignalCascade(monLoader, deterLoader, stub, log)
}

// ---------------------------------------------------------------------------
// Test node builders
// ---------------------------------------------------------------------------

func pmNodeWithCascade(nodeID string, cascadeTo []string) *models.MonitoringNodeDefinition {
	return &models.MonitoringNodeDefinition{
		NodeID:  nodeID,
		Version: "1.0.0",
		Type:    "MONITORING",
		Classifications: []models.ClassificationDef{
			{Category: "NORMAL", Condition: "", MCUGateSuggestion: "SAFE"},
		},
		CascadeTo: cascadeTo,
	}
}

func mdNodeWithContributing(nodeID string, contributing []string) *models.DeteriorationNodeDefinition {
	return &models.DeteriorationNodeDefinition{
		NodeID:              nodeID,
		Version:             "1.0.0",
		Type:                "DETERIORATION",
		ContributingSignals: contributing,
		ComputedFields: []models.ComputedFieldDef{
			{Name: "score", Formula: "0"},
		},
		Thresholds: []models.ThresholdDef{
			{Signal: "STABLE", Condition: "", Severity: "NONE", MCUGateSuggestion: "SAFE"},
		},
		InsufficientData: models.InsufficientDataPolicy{Action: "USE_SNAPSHOT"},
	}
}

// ---------------------------------------------------------------------------
// TestSignalCascade_PMToMD
// PM-04 has cascade_to=[MD-01, MD-05]. Trigger with PM-04 → both MD-01 and MD-05 evaluated.
// ---------------------------------------------------------------------------

func TestSignalCascade_PMToMD(t *testing.T) {
	pm04 := pmNodeWithCascade("PM-04", []string{"MD-01", "MD-05"})
	md01 := mdNodeWithContributing("MD-01", nil)
	md05 := mdNodeWithContributing("MD-05", nil)

	stub := newStubEvaluator()
	stub.setResponse("MD-01", buildMDEvent("MD-01", "MILD"), nil)
	stub.setResponse("MD-05", buildMDEvent("MD-05", "NONE"), nil)

	cascade := buildCascade(
		[]*models.MonitoringNodeDefinition{pm04},
		[]*models.DeteriorationNodeDefinition{md01, md05},
		stub,
	)

	events := cascade.Trigger(context.Background(), "PM-04", "patient-c001", "T2D", 1.0)

	// Verify both MD-01 and MD-05 were evaluated.
	called := stub.calledNodes()
	hasMD01 := false
	hasMD05 := false
	for _, id := range called {
		if id == "MD-01" {
			hasMD01 = true
		}
		if id == "MD-05" {
			hasMD05 = true
		}
	}
	if !hasMD01 {
		t.Error("expected MD-01 to be evaluated in Pass 1")
	}
	if !hasMD05 {
		t.Error("expected MD-05 to be evaluated in Pass 1")
	}

	// Both events returned.
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

// ---------------------------------------------------------------------------
// TestSignalCascade_MDToMD06
// PM-01 triggers MD-02 (fires with MODERATE). MD-02 is in MD-06's contributing_signals
// → Pass 2 evaluates MD-06.
// ---------------------------------------------------------------------------

func TestSignalCascade_MDToMD06(t *testing.T) {
	pm01 := pmNodeWithCascade("PM-01", []string{"MD-02"})
	md02 := mdNodeWithContributing("MD-02", nil)
	// MD-06 lists MD-02 as contributing signal.
	md06 := mdNodeWithContributing("MD-06", []string{"MD-02"})

	stub := newStubEvaluator()
	stub.setResponse("MD-02", buildMDEvent("MD-02", "MODERATE"), nil)
	stub.setResponse("MD-06", buildMDEvent("MD-06", "CRITICAL"), nil)

	cascade := buildCascade(
		[]*models.MonitoringNodeDefinition{pm01},
		[]*models.DeteriorationNodeDefinition{md02, md06},
		stub,
	)

	events := cascade.Trigger(context.Background(), "PM-01", "patient-c002", "T2D", 1.0)

	// Should have 2 events: MD-02 (Pass 1) + MD-06 (Pass 2).
	if len(events) != 2 {
		t.Errorf("expected 2 events (MD-02 + MD-06), got %d", len(events))
	}

	// Verify MD-06 was called in Pass 2.
	called := stub.calledNodes()
	hasMD06 := false
	for _, id := range called {
		if id == "MD-06" {
			hasMD06 = true
		}
	}
	if !hasMD06 {
		t.Error("expected MD-06 to be evaluated in Pass 2")
	}
}

// ---------------------------------------------------------------------------
// TestSignalCascade_NonFatalFailure
// MD-01 evaluation fails → MD-05 still evaluated, error logged (not returned).
// ---------------------------------------------------------------------------

func TestSignalCascade_NonFatalFailure(t *testing.T) {
	pm04 := pmNodeWithCascade("PM-04", []string{"MD-01", "MD-05"})
	md01 := mdNodeWithContributing("MD-01", nil)
	md05 := mdNodeWithContributing("MD-05", nil)

	stub := newStubEvaluator()
	// MD-01 fails with an error.
	stub.setResponse("MD-01", nil, errors.New("simulated MD-01 evaluation failure"))
	// MD-05 succeeds.
	stub.setResponse("MD-05", buildMDEvent("MD-05", "MILD"), nil)

	cascade := buildCascade(
		[]*models.MonitoringNodeDefinition{pm04},
		[]*models.DeteriorationNodeDefinition{md01, md05},
		stub,
	)

	events := cascade.Trigger(context.Background(), "PM-04", "patient-c003", "T2D", 1.0)

	// MD-01 error is non-fatal; MD-05 event is still returned.
	if len(events) != 1 {
		t.Errorf("expected 1 event (MD-05 only), got %d", len(events))
	}
	if len(events) == 1 && events[0].NodeID != "MD-05" {
		t.Errorf("expected event from MD-05, got node %q", events[0].NodeID)
	}

	// MD-05 must still have been called.
	called := stub.calledNodes()
	hasMD05 := false
	for _, id := range called {
		if id == "MD-05" {
			hasMD05 = true
		}
	}
	if !hasMD05 {
		t.Error("expected MD-05 to be evaluated despite MD-01 failure")
	}
}

// ---------------------------------------------------------------------------
// TestSignalCascade_NoCascade
// PM node with empty cascade_to → no MD nodes evaluated, returns empty slice.
// ---------------------------------------------------------------------------

func TestSignalCascade_NoCascade(t *testing.T) {
	pm08 := pmNodeWithCascade("PM-08", nil) // no cascade_to

	stub := newStubEvaluator()

	cascade := buildCascade(
		[]*models.MonitoringNodeDefinition{pm08},
		[]*models.DeteriorationNodeDefinition{},
		stub,
	)

	events := cascade.Trigger(context.Background(), "PM-08", "patient-c004", "T2D", 0.0)

	if len(events) != 0 {
		t.Errorf("expected 0 events for PM node with no cascade_to, got %d", len(events))
	}
	if len(stub.calls) != 0 {
		t.Errorf("expected no evaluator calls, got %d", len(stub.calls))
	}
}

// ---------------------------------------------------------------------------
// TestSignalCascade_MD06NotTriggeredIfNoContributor
// PM-08 cascades to MD-04 only. MD-04 fires but is NOT in MD-06's contributing_signals
// → MD-06 NOT evaluated.
// ---------------------------------------------------------------------------

func TestSignalCascade_MD06NotTriggeredIfNoContributor(t *testing.T) {
	pm08 := pmNodeWithCascade("PM-08", []string{"MD-04"})
	// MD-04 is not listed in MD-06's contributing_signals.
	md04 := mdNodeWithContributing("MD-04", nil)
	md06 := mdNodeWithContributing("MD-06", []string{"MD-02"}) // MD-02, NOT MD-04

	stub := newStubEvaluator()
	stub.setResponse("MD-04", buildMDEvent("MD-04", "MODERATE"), nil)
	stub.setResponse("MD-06", buildMDEvent("MD-06", "CRITICAL"), nil) // should never be called

	cascade := buildCascade(
		[]*models.MonitoringNodeDefinition{pm08},
		[]*models.DeteriorationNodeDefinition{md04, md06},
		stub,
	)

	events := cascade.Trigger(context.Background(), "PM-08", "patient-c005", "T2D", 2.0)

	// Only MD-04 event returned; MD-06 NOT evaluated.
	if len(events) != 1 {
		t.Errorf("expected 1 event (MD-04 only), got %d", len(events))
	}

	// Verify MD-06 was NOT called.
	called := stub.calledNodes()
	for _, id := range called {
		if id == "MD-06" {
			t.Error("MD-06 should NOT have been evaluated — MD-04 is not in MD-06's contributing_signals")
		}
	}
}

// ---------------------------------------------------------------------------
// TestSignalCascade_BuildsCascadeContext
// Verify PM severity score is passed to MD engine in CascadeContext.PMSignals.
// ---------------------------------------------------------------------------

func TestSignalCascade_BuildsCascadeContext(t *testing.T) {
	pm04 := pmNodeWithCascade("PM-04", []string{"MD-01"})
	md01 := mdNodeWithContributing("MD-01", nil)

	stub := newStubEvaluator()
	stub.setResponse("MD-01", buildMDEvent("MD-01", "MILD"), nil)

	cascade := buildCascade(
		[]*models.MonitoringNodeDefinition{pm04},
		[]*models.DeteriorationNodeDefinition{md01},
		stub,
	)

	// Trigger with CRITICAL severity = 3.0.
	cascade.Trigger(context.Background(), "PM-04", "patient-c006", "T2D", 3.0)

	// MD-01 should have been called with a CascadeContext containing pm_04=3.0.
	call, found := stub.callFor("MD-01")
	if !found {
		t.Fatal("expected MD-01 to have been called")
	}
	if call.cascadeCtx == nil {
		t.Fatal("CascadeContext passed to MD-01 must not be nil")
	}

	// PM key uses underscore convention: PM-04 → pm_04.
	pmKey := nodeIDToFieldKey("PM-04") // "pm_04"
	score, ok := call.cascadeCtx.PMSignals[pmKey]
	if !ok {
		t.Errorf("CascadeContext.PMSignals should contain key %q (from PM-04)", pmKey)
	}
	if score != 3.0 {
		t.Errorf("CascadeContext.PMSignals[%q]: expected 3.0, got %.1f", pmKey, score)
	}
}
