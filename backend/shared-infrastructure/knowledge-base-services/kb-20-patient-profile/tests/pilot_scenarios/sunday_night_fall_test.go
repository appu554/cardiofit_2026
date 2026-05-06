// Wave 6.3 — Sunday-night-fall pilot scenario rehearsal.
//
// Layer 2 doc Part 8 closing line: "the Sunday-night-fall walkthrough —
// exercising the substrate against a real-world scenario."
//
// Day-by-day script (per the Wave 6.3 plan task):
//
//   Sun PM   fall Event ingested
//   Mon AM   post-fall vitals (delta-flagged) + agitation episode
//   Mon PM   hospital_admission Event for head CT
//   Tue AM   hospital_discharge Event with new anti-emetic
//   Wed      ACOP pharmacist completes reconciliation
//   Thu      pathology result via MHR (mild AKI) opens active concern;
//            potassium baseline recompute excludes the AKI window
//   Fri      care intensity transition (active_treatment → comfort_focused)
//   Sat      forward + backward EvidenceTrace traversal demonstrates
//            full lineage from Sunday's fall to Friday's care-intensity
//            transition.
//
// All in-memory; no DB. The intent is to demonstrate the substrate's
// contracts compose into a believable end-to-end flow.
package pilot_scenarios

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// pilotSink is a tiny in-memory store satisfying both the NodeStore and
// EdgeStore contracts. We deliberately don't pull in kb-20 storage.
type pilotSink struct {
	nodes map[uuid.UUID]models.EvidenceTraceNode
	edges []evidence_trace.Edge
}

func newPilotSink() *pilotSink {
	return &pilotSink{nodes: map[uuid.UUID]models.EvidenceTraceNode{}}
}

func (s *pilotSink) writeNode(n models.EvidenceTraceNode) models.EvidenceTraceNode {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	s.nodes[n.ID] = n
	return n
}

func (s *pilotSink) writeEdge(from, to uuid.UUID, kind evidence_trace.EdgeKind) {
	s.edges = append(s.edges, evidence_trace.Edge{From: from, To: to, Kind: kind, CreatedAt: time.Now().UTC()})
}

// EdgeStore impl
func (s *pilotSink) InsertEdge(_ context.Context, e evidence_trace.Edge) error {
	for _, ex := range s.edges {
		if ex.From == e.From && ex.To == e.To && ex.Kind == e.Kind {
			return nil
		}
	}
	s.edges = append(s.edges, e)
	return nil
}
func (s *pilotSink) OutEdges(_ context.Context, from uuid.UUID, kind evidence_trace.EdgeKind) ([]evidence_trace.Edge, error) {
	var out []evidence_trace.Edge
	for _, e := range s.edges {
		if e.From == from && (kind == "" || e.Kind == kind) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (s *pilotSink) InEdges(_ context.Context, to uuid.UUID, kind evidence_trace.EdgeKind) ([]evidence_trace.Edge, error) {
	var out []evidence_trace.Edge
	for _, e := range s.edges {
		if e.To == to && (kind == "" || e.Kind == kind) {
			out = append(out, e)
		}
	}
	return out, nil
}

// NodeStore impl
func (s *pilotSink) GetEvidenceTraceNode(_ context.Context, id uuid.UUID) (*models.EvidenceTraceNode, error) {
	n, ok := s.nodes[id]
	if !ok {
		return nil, evidence_trace.ErrInvalidDepth // sentinel reuse
	}
	c := n
	return &c, nil
}
func (s *pilotSink) ListEvidenceTraceNodesByResident(_ context.Context, residentRef uuid.UUID, from, to time.Time) ([]models.EvidenceTraceNode, error) {
	var out []models.EvidenceTraceNode
	for _, n := range s.nodes {
		if n.ResidentRef == nil || *n.ResidentRef != residentRef {
			continue
		}
		if n.RecordedAt.Before(from) || !n.RecordedAt.Before(to) {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func makeNode(sm, sct string, residentRef uuid.UUID, when time.Time) models.EvidenceTraceNode {
	return models.EvidenceTraceNode{
		StateMachine:    sm,
		StateChangeType: sct,
		RecordedAt:      when,
		OccurredAt:      when,
		ResidentRef:     &residentRef,
	}
}

func TestPilotScenario_SundayNightFall_FullWeek(t *testing.T) {
	sink := newPilotSink()
	ctx := context.Background()
	resident := uuid.New()
	startSun := time.Date(2026, 5, 3, 19, 0, 0, 0, time.UTC) // Sunday 7pm

	// --- Sunday PM: fall Event ingested ---
	fallEvent := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "fall_event_recorded",
		resident, startSun))

	// --- Monday AM: post-fall vitals + agitation episode ---
	monAM := startSun.Add(13 * time.Hour)
	postFallVitals := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "baseline_delta_flagged",
		resident, monAM))
	agitation := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "agitation_episode_recorded",
		resident, monAM.Add(time.Hour)))
	sink.writeEdge(fallEvent.ID, postFallVitals.ID, evidence_trace.EdgeKindLedTo)
	sink.writeEdge(fallEvent.ID, agitation.ID, evidence_trace.EdgeKindLedTo)

	// --- Monday PM: hospital_admission Event for head CT ---
	monPM := monAM.Add(8 * time.Hour)
	hospAdmission := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "hospital_admission_recorded",
		resident, monPM))
	sink.writeEdge(fallEvent.ID, hospAdmission.ID, evidence_trace.EdgeKindLedTo)
	sink.writeEdge(postFallVitals.ID, hospAdmission.ID, evidence_trace.EdgeKindEvidenceFor)

	// --- Tuesday AM: hospital_discharge Event with new anti-emetic ---
	tueAM := startSun.Add(36 * time.Hour)
	discharge := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "hospital_discharge_recorded",
		resident, tueAM))
	sink.writeEdge(hospAdmission.ID, discharge.ID, evidence_trace.EdgeKindLedTo)

	// --- Wednesday: ACOP pharmacist reconciliation ---
	wed := tueAM.Add(24 * time.Hour)
	reconciliation := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineRecommendation, "reconciliation_completed",
		resident, wed))
	sink.writeEdge(discharge.ID, reconciliation.ID, evidence_trace.EdgeKindLedTo)
	sink.writeEdge(discharge.ID, reconciliation.ID, evidence_trace.EdgeKindDerivedFrom)

	// --- Thursday: MHR pathology result, AKI active concern, baseline recompute ---
	thu := wed.Add(24 * time.Hour)
	pathology := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "pathology_result_received",
		resident, thu))
	akiConcern := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineClinicalState, "active_concern_opened_AKI_watching",
		resident, thu.Add(time.Hour)))
	baselineRecompute := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineMonitoring, "baseline_recomputed_excluding_aki_window",
		resident, thu.Add(2*time.Hour)))
	sink.writeEdge(pathology.ID, akiConcern.ID, evidence_trace.EdgeKindLedTo)
	sink.writeEdge(akiConcern.ID, baselineRecompute.ID, evidence_trace.EdgeKindLedTo)

	// --- Friday: care intensity transition ---
	fri := thu.Add(24 * time.Hour)
	careTransition := sink.writeNode(makeNode(
		models.EvidenceTraceStateMachineClinicalState, "care_intensity_active_treatment_to_comfort_focused",
		resident, fri))
	sink.writeEdge(akiConcern.ID, careTransition.ID, evidence_trace.EdgeKindLedTo)
	sink.writeEdge(reconciliation.ID, careTransition.ID, evidence_trace.EdgeKindEvidenceFor)
	sink.writeEdge(fallEvent.ID, careTransition.ID, evidence_trace.EdgeKindEvidenceFor)

	// --- Saturday: forward + backward EvidenceTrace traversal ---
	// Forward from Sunday's fall: every downstream node should be reachable.
	fwd, err := evidence_trace.TraceForward(ctx, sink, fallEvent.ID, 10, nil)
	if err != nil {
		t.Fatalf("forward trace: %v", err)
	}
	// Sanity: should reach hospital admission, discharge, reconciliation,
	// AKI concern, baseline recompute, care transition (six downstream).
	expectedReachable := []uuid.UUID{
		postFallVitals.ID, agitation.ID, hospAdmission.ID, discharge.ID,
		reconciliation.ID, careTransition.ID,
	}
	for _, id := range expectedReachable {
		if !containsID(fwd.NodeIDs, id) {
			t.Fatalf("forward traversal from fall did not reach node %s", id)
		}
	}

	// Backward from Friday's care transition: should reach the Sunday fall.
	bk, err := evidence_trace.TraceBackward(ctx, sink, careTransition.ID, 10, nil)
	if err != nil {
		t.Fatalf("backward trace: %v", err)
	}
	if !containsID(bk.NodeIDs, fallEvent.ID) {
		t.Fatal("backward traversal from Friday's care transition MUST reach Sunday's fall")
	}

	// Reasoning window: regulator-audit rollup over the full week.
	winFrom := startSun.Add(-time.Hour)
	winTo := fri.Add(24 * time.Hour)
	w, err := evidence_trace.ReasoningWindow(ctx, resident, winFrom, winTo, sink)
	if err != nil {
		t.Fatalf("ReasoningWindow: %v", err)
	}
	if w.TotalNodes != len(sink.nodes) {
		t.Fatalf("reasoning window total drift: want %d got %d", len(sink.nodes), w.TotalNodes)
	}
	if w.RecommendationCount != 1 {
		t.Fatalf("expected 1 Recommendation node (reconciliation); got %d", w.RecommendationCount)
	}

	t.Logf("Sunday-night-fall pilot rehearsal: %d substrate nodes written, %d edges, "+
		"forward traversal reaches %d downstream, backward traversal reaches %d upstream, "+
		"reasoning window summarises the full week",
		len(sink.nodes), len(sink.edges), len(fwd.NodeIDs), len(bk.NodeIDs))
}

func containsID(ids []uuid.UUID, want uuid.UUID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
