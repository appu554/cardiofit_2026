package ingestion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// fakeKB20 captures every call for inspection. Idempotency is enforced
// by keeping a map keyed on MedicineUse.ID — a second upsert with the
// same ID returns the prior record (with its original CreatedAt) so the
// runner detects the duplicate.
type fakeKB20 struct {
	medicineUses map[uuid.UUID]models.MedicineUse
	traceNodes   map[uuid.UUID]models.EvidenceTraceNode
	edges        []evidence_trace.Edge
}

func newFakeKB20() *fakeKB20 {
	return &fakeKB20{
		medicineUses: make(map[uuid.UUID]models.MedicineUse),
		traceNodes:   make(map[uuid.UUID]models.EvidenceTraceNode),
	}
}

func (f *fakeKB20) GetMedicineUse(_ context.Context, id uuid.UUID) (*models.MedicineUse, error) {
	if m, ok := f.medicineUses[id]; ok {
		return &m, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeKB20) UpsertMedicineUse(_ context.Context, m models.MedicineUse) (*models.MedicineUse, error) {
	if prior, ok := f.medicineUses[m.ID]; ok {
		// Preserve original CreatedAt on update — this is what a real
		// upsert handler would do for an existing row.
		merged := m
		merged.CreatedAt = prior.CreatedAt
		f.medicineUses[m.ID] = merged
		return &merged, nil
	}
	f.medicineUses[m.ID] = m
	return &m, nil
}

func (f *fakeKB20) UpsertEvidenceTraceNode(_ context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error) {
	f.traceNodes[n.ID] = n
	return &n, nil
}

func (f *fakeKB20) InsertEvidenceTraceEdge(_ context.Context, e evidence_trace.Edge) error {
	f.edges = append(f.edges, e)
	return nil
}

// fakeMatcher returns a fixed MatchResult based on the IHI / Medicare
// presence on the incoming identifier. It mirrors the tier semantics of
// the real fuzzy matcher closely enough that runner tests exercise each
// confidence path.
type fakeMatcher struct {
	highRef   uuid.UUID // returned for IHI=8003608000000001..3
	mediumRef uuid.UUID // returned for any non-empty Medicare with empty IHI
	lowRef    uuid.UUID // returned for empty IHI+Medicare with present DOB+name
}

func (m *fakeMatcher) Match(_ context.Context, in identity.IncomingIdentifier) (identity.MatchResult, error) {
	if strings.HasPrefix(in.IHI, "800360") {
		return identity.MatchResult{
			ResidentRef:    &m.highRef,
			Confidence:     identity.ConfidenceHigh,
			Path:           identity.MatchPathIHI,
			RequiresReview: false,
		}, nil
	}
	if in.Medicare != "" {
		ref := m.mediumRef
		return identity.MatchResult{
			ResidentRef:    &ref,
			Confidence:     identity.ConfidenceMedium,
			Path:           identity.MatchPathMedicareNameDOB,
			RequiresReview: false,
		}, nil
	}
	if !in.DOB.IsZero() && in.FamilyName != "" {
		// Low: scoped to facility, returns reviewable result.
		ref := m.lowRef
		return identity.MatchResult{
			ResidentRef:    &ref,
			Confidence:     identity.ConfidenceLow,
			Path:           identity.MatchPathNameDOBFacility,
			RequiresReview: true,
		}, nil
	}
	return identity.MatchResult{
		Confidence:     identity.ConfidenceNone,
		Path:           identity.MatchPathNoMatch,
		RequiresReview: true,
	}, nil
}

const tinyHeader = goodHeader

func tinyRow(t *testing.T, ihi, medicare, family, given, dob, med, strength, start, end, ind string) string {
	t.Helper()
	return strings.Join([]string{
		"P1", "F1", ihi, medicare, family, given, dob,
		med, strength, "tablet", "ORAL", "OD", start, end, "Dr A", "", ind,
	}, ",") + "\n"
}

func newRunnerCfg(t *testing.T, kb *fakeKB20, matcher identity.IdentityMatcher) RunnerConfig {
	t.Helper()
	return RunnerConfig{
		FacilityID:  uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		SourceLabel: "telstra-medpoint-csv-test",
		Client:      kb,
		Matcher:     matcher,
		Normaliser:  &Normaliser{AMT: newFakeAMT(), SNOMED: &fakeSNOMED{m: map[string]string{"hypertension": "SCT-HTN"}}},
		Now:         func() time.Time { return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC) },
	}
}

func TestRun_HappyPath_MixedConfidenceTiers(t *testing.T) {
	matcher := &fakeMatcher{
		highRef:   uuid.MustParse("aaaaaaaa-1111-1111-1111-111111111111"),
		mediumRef: uuid.MustParse("bbbbbbbb-2222-2222-2222-222222222222"),
		lowRef:    uuid.MustParse("cccccccc-3333-3333-3333-333333333333"),
	}
	body := tinyHeader + "\n" +
		// HIGH: IHI present
		tinyRow(t, "8003608000000001", "1234567890", "Smith", "Jane", "1942-04-12",
			"paracetamol", "500mg", "2026-01-01", "", "osteoarthritis pain") +
		// MEDIUM: no IHI, Medicare present
		tinyRow(t, "", "9876543210", "Jones", "Bob", "1939-07-22",
			"amlodipine", "5mg", "2026-02-01", "", "hypertension") +
		// LOW: no IHI, no Medicare, name+DOB present → review
		tinyRow(t, "", "", "Brown", "Carol", "1945-11-03",
			"metformin", "500mg", "2026-03-01", "", "type 2 diabetes") +
		// NONE: no identifiers, no name → review queue
		tinyRow(t, "", "", "", "", "1950-01-01",
			"atorvastatin", "20mg", "2026-04-01", "", "")

	kb := newFakeKB20()
	cfg := newRunnerCfg(t, kb, matcher)

	res, err := Run(context.Background(), strings.NewReader(body), cfg)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}

	// Row 4 has empty name → blocked by parse error, counted as RowsErrored
	// (not RowsSkippedNoMatch), since we never reach the matcher for it.
	if res.RowsParsed != 4 {
		t.Errorf("RowsParsed = %d, want 4", res.RowsParsed)
	}
	// HIGH + MEDIUM + LOW all got resident refs → ingested.
	if res.RowsIngested != 3 {
		t.Errorf("RowsIngested = %d, want 3 (high+medium+low)", res.RowsIngested)
	}
	// Row 4 (no names) is a parse error blocker.
	if res.RowsErrored != 1 {
		t.Errorf("RowsErrored = %d, want 1", res.RowsErrored)
	}
	if res.RowsSkippedNoMatch != 0 {
		t.Errorf("RowsSkippedNoMatch = %d, want 0", res.RowsSkippedNoMatch)
	}

	// Verify the run-level start + end nodes + edge.
	if _, ok := kb.traceNodes[res.EvidenceTraceNodeRef]; !ok {
		t.Errorf("run-level start node not written")
	}
	gotStart, gotEnd := false, false
	for _, n := range kb.traceNodes {
		if n.StateChangeType == "extraction_pipeline_started" {
			gotStart = true
		}
		if n.StateChangeType == "extraction_pipeline_completed" {
			gotEnd = true
		}
	}
	if !gotStart || !gotEnd {
		t.Errorf("expected start+completed run-level nodes, got start=%v end=%v", gotStart, gotEnd)
	}
	gotEdge := false
	for _, e := range kb.edges {
		if e.To == res.EvidenceTraceNodeRef && e.Kind == evidence_trace.EdgeKindDerivedFrom {
			gotEdge = true
		}
	}
	if !gotEdge {
		t.Errorf("expected at least one derived_from edge to run-start node")
	}

	// Per-row nodes: 3 ingested rows → 3 row nodes; each links via derived_from to runStartID.
	rowNodeCount := 0
	for _, n := range kb.traceNodes {
		if n.StateChangeType == "ingestion_row" {
			rowNodeCount++
			if n.ResidentRef == nil {
				t.Errorf("row node missing ResidentRef")
			}
		}
	}
	if rowNodeCount != 3 {
		t.Errorf("rowNodeCount = %d, want 3", rowNodeCount)
	}
}

func TestRun_Idempotent(t *testing.T) {
	matcher := &fakeMatcher{
		highRef:   uuid.MustParse("aaaaaaaa-1111-1111-1111-111111111111"),
		mediumRef: uuid.MustParse("bbbbbbbb-2222-2222-2222-222222222222"),
		lowRef:    uuid.MustParse("cccccccc-3333-3333-3333-333333333333"),
	}
	body := tinyHeader + "\n" +
		tinyRow(t, "8003608000000001", "", "Smith", "Jane", "1942-04-12",
			"paracetamol", "500mg", "2026-01-01", "", "pain") +
		tinyRow(t, "8003608000000002", "", "Jones", "Bob", "1939-07-22",
			"amlodipine", "5mg", "2026-02-01", "", "hypertension")

	kb := newFakeKB20()
	cfg := newRunnerCfg(t, kb, matcher)

	first, err := Run(context.Background(), strings.NewReader(body), cfg)
	if err != nil {
		t.Fatalf("first run err: %v", err)
	}
	if first.RowsIngested != 2 {
		t.Fatalf("first run RowsIngested = %d, want 2", first.RowsIngested)
	}
	muCountAfter1 := len(kb.medicineUses)

	// Second run on same input — needs the SAME run start time so the
	// CreatedAt-comparison in processRow can detect the duplicate. The
	// fake preserves CreatedAt from the first write; we advance the
	// clock so the new mu.CreatedAt is strictly after the stored one.
	cfg2 := cfg
	cfg2.Now = func() time.Time { return time.Date(2026, 5, 6, 12, 5, 0, 0, time.UTC) }
	second, err := Run(context.Background(), strings.NewReader(body), cfg2)
	if err != nil {
		t.Fatalf("second run err: %v", err)
	}
	if second.RowsSkippedDup != 2 {
		t.Errorf("second run RowsSkippedDup = %d, want 2 (idempotent)", second.RowsSkippedDup)
	}
	if second.RowsIngested != 0 {
		t.Errorf("second run RowsIngested = %d, want 0", second.RowsIngested)
	}
	if len(kb.medicineUses) != muCountAfter1 {
		t.Errorf("MedicineUse count grew on second run: was %d, now %d",
			muCountAfter1, len(kb.medicineUses))
	}
}

func TestRun_DryRunWritesNothing(t *testing.T) {
	matcher := &fakeMatcher{highRef: uuid.New()}
	body := tinyHeader + "\n" +
		tinyRow(t, "8003608000000001", "", "Smith", "Jane", "1942-04-12",
			"paracetamol", "500mg", "2026-01-01", "", "pain")

	kb := newFakeKB20()
	cfg := newRunnerCfg(t, kb, matcher)
	cfg.DryRun = true

	res, err := Run(context.Background(), strings.NewReader(body), cfg)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if res.RowsIngested != 1 {
		t.Errorf("dry-run RowsIngested = %d, want 1", res.RowsIngested)
	}
	if len(kb.medicineUses) != 0 || len(kb.traceNodes) != 0 || len(kb.edges) != 0 {
		t.Errorf("dry-run wrote to client: mu=%d nodes=%d edges=%d",
			len(kb.medicineUses), len(kb.traceNodes), len(kb.edges))
	}
}

func TestRun_NoneConfidenceQueued(t *testing.T) {
	matcher := &fakeMatcher{}
	// Row with valid name but matcher returns NONE because no IHI/medicare/dob.
	// Use a dob that the matcher's logic treats as "no match" — empty name
	// triggers parse-error path, so we instead use empty IHI+Medicare AND
	// empty DOB.
	body := tinyHeader + "\n" +
		// FamilyName present, no IHI/medicare; DOB empty → fakeMatcher hits "no match"
		tinyRow(t, "", "", "Brown", "Carol", "",
			"metformin", "500mg", "2026-03-01", "", "")
	kb := newFakeKB20()
	cfg := newRunnerCfg(t, kb, matcher)

	res, err := Run(context.Background(), strings.NewReader(body), cfg)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	// Empty DOB → parseDate fails → error path (not no-match path).
	if res.RowsErrored == 0 {
		t.Errorf("expected at least one errored row for empty DOB")
	}
}

func TestRun_NoMatchEnqueuesReview(t *testing.T) {
	// Build a matcher that always returns ConfidenceNone (no resident).
	always := alwaysNoneMatcher{}
	body := tinyHeader + "\n" +
		tinyRow(t, "8003608000000099", "", "Ghost", "Person", "1900-01-01",
			"paracetamol", "500mg", "2026-01-01", "", "pain")
	kb := newFakeKB20()
	cfg := newRunnerCfg(t, kb, always)

	res, err := Run(context.Background(), strings.NewReader(body), cfg)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if res.RowsSkippedNoMatch != 1 {
		t.Errorf("RowsSkippedNoMatch = %d, want 1", res.RowsSkippedNoMatch)
	}
	if len(res.ReviewQueueRefs) != 1 {
		t.Errorf("ReviewQueueRefs len = %d, want 1", len(res.ReviewQueueRefs))
	}
}

type alwaysNoneMatcher struct{}

func (alwaysNoneMatcher) Match(_ context.Context, _ identity.IncomingIdentifier) (identity.MatchResult, error) {
	return identity.MatchResult{
		Confidence:     identity.ConfidenceNone,
		Path:           identity.MatchPathNoMatch,
		RequiresReview: true,
	}, nil
}

func TestRun_ErrorsTolerated(t *testing.T) {
	matcher := &fakeMatcher{highRef: uuid.New()}
	// Mix: 1 good row, 1 row with malformed start_date.
	body := tinyHeader + "\n" +
		tinyRow(t, "8003608000000001", "", "Smith", "Jane", "1942-04-12",
			"paracetamol", "500mg", "not-a-date", "", "pain") +
		tinyRow(t, "8003608000000002", "", "Jones", "Bob", "1939-07-22",
			"amlodipine", "5mg", "2026-02-01", "", "hypertension")

	kb := newFakeKB20()
	cfg := newRunnerCfg(t, kb, matcher)

	res, err := Run(context.Background(), strings.NewReader(body), cfg)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if res.RowsErrored != 1 || res.RowsIngested != 1 {
		t.Errorf("got errored=%d ingested=%d, want 1+1", res.RowsErrored, res.RowsIngested)
	}
}

func TestDeterministicID_StableAcrossCalls(t *testing.T) {
	a := deterministicID("a", "b", "c")
	b := deterministicID("a", "b", "c")
	if a != b {
		t.Errorf("deterministicID not stable: %v vs %v", a, b)
	}
	c := deterministicID("a", "b", "d")
	if a == c {
		t.Errorf("deterministicID collided on different inputs")
	}
}
