package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/citations"
	"github.com/cardiofit/shared/v2_substrate/ethics/decision_metadata"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// newExplainTestRouter wires an ExplainHandler at /v1/explain/:decision_id
// using the supplied stores/linker.
func newExplainTestRouter(
	t *testing.T,
	md decision_metadata.Store,
	log ethics_log.Store,
	reg citations.Registry,
	linker EvidenceTraceLinker,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewExplainHandler(md, log, reg, linker)
	r.GET("/v1/explain/:decision_id", h.HandleExplain)
	return r
}

// stubLinker returns a fixed slice of node IDs.
type stubLinker struct {
	nodes []uuid.UUID
	err   error
}

func (s stubLinker) LinkedNodes(_ context.Context, _ uuid.UUID, _ int) ([]uuid.UUID, error) {
	return s.nodes, s.err
}

// erroringRegistry wraps an InMemoryRegistry and returns an error from
// ListCitations to verify graceful degradation: a registry failure must
// degrade to an empty citations slice rather than surface as a 500.
type erroringRegistry struct {
	citations.Registry
}

func (erroringRegistry) ListCitations(_ context.Context, _ string) ([]citations.RecommendationCitation, error) {
	return nil, errors.New("simulated registry failure")
}

func TestExplain_KnownDecision_Returns200WithFullPayload(t *testing.T) {
	mdStore := decision_metadata.NewInMemoryStore()
	logStore := ethics_log.NewInMemoryStore()
	reg := citations.NewInMemoryRegistry()

	decisionID := uuid.New()
	recID := uuid.New()
	now := time.Now().UTC()
	outcome := "approved"
	if err := mdStore.Put(context.Background(), decision_metadata.Metadata{
		DecisionID:           decisionID,
		Component:            "kb-32-recommendation-craft",
		DecisionType:         "draft_recommendation",
		AffectedSubjectID:    "resident-001",
		AffectedSubjectClass: "resident",
		PrinciplesImplicated: []string{"P1", "P6"},
		ERMReviewed:          true,
		ERMOutcome:           &outcome,
		ContestationEnabled:  true,
		AuditTraceRef:        uuid.New(),
		Timestamp:            now,
		RecommendationID:     recID,
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	// Seed two pinned citations for recID so the handler returns them via
	// the now-wired ListCitations call.
	for _, c := range []citations.RecommendationCitation{
		{RecommendationID: recID.String(), SourceID: "src-A", Version: "v1", PinnedAt: now},
		{RecommendationID: recID.String(), SourceID: "src-B", Version: "v2", PinnedAt: now},
	} {
		if err := reg.SaveCitation(context.Background(), c); err != nil {
			t.Fatalf("seed citation: %v", err)
		}
	}
	// Unrelated citation under a different rec — must NOT appear.
	if err := reg.SaveCitation(context.Background(), citations.RecommendationCitation{
		RecommendationID: uuid.New().String(), SourceID: "src-X", Version: "v1", PinnedAt: now,
	}); err != nil {
		t.Fatalf("seed unrelated citation: %v", err)
	}

	logger := ethics_log.NewLogger(logStore)
	if err := logger.Append(context.Background(), ethics_log.Entry{
		DecisionID:  decisionID,
		EntryType:   ethics_log.EntryTypeDecision,
		Severity:    1,
		Description: "primary decision",
	}); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	if err := logger.Append(context.Background(), ethics_log.Entry{
		DecisionID:  decisionID,
		EntryType:   ethics_log.EntryTypeReviewRequested,
		Severity:    2,
		Description: "review requested",
	}); err != nil {
		t.Fatalf("seed log 2: %v", err)
	}
	// Unrelated entry on a different decision; must NOT appear.
	if err := logger.Append(context.Background(), ethics_log.Entry{
		DecisionID:  uuid.New(),
		EntryType:   ethics_log.EntryTypeDecision,
		Severity:    1,
		Description: "unrelated",
	}); err != nil {
		t.Fatalf("seed log 3: %v", err)
	}

	linkedNodes := []uuid.UUID{uuid.New(), uuid.New()}
	r := newExplainTestRouter(t, mdStore, logStore, reg, stubLinker{nodes: linkedNodes})

	req := httptest.NewRequest(http.MethodGet, "/v1/explain/"+decisionID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp ExplainResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, w.Body.String())
	}
	if resp.DecisionID != decisionID {
		t.Errorf("decision_id mismatch: got %s want %s", resp.DecisionID, decisionID)
	}
	if resp.Metadata == nil || resp.Metadata.DecisionID != decisionID {
		t.Errorf("metadata not populated correctly: %+v", resp.Metadata)
	}
	if len(resp.EthicsLog) != 2 {
		t.Errorf("expected 2 ethics log entries (filtered), got %d", len(resp.EthicsLog))
	}
	if len(resp.LinkedTrace) != len(linkedNodes) {
		t.Errorf("expected %d linked nodes, got %d", len(linkedNodes), len(resp.LinkedTrace))
	}
	if resp.Citations == nil {
		t.Error("citations must be a non-nil slice (even if empty)")
	}
	if len(resp.Citations) != 2 {
		t.Errorf("expected 2 citations for rec_id=%s, got %d (%+v)", recID, len(resp.Citations), resp.Citations)
	}
	for _, c := range resp.Citations {
		if c.RecommendationID != recID.String() {
			t.Errorf("citation rec-ID mismatch: got %s want %s", c.RecommendationID, recID)
		}
	}
}

func TestExplain_UnknownDecision_Returns404(t *testing.T) {
	r := newExplainTestRouter(t,
		decision_metadata.NewInMemoryStore(),
		ethics_log.NewInMemoryStore(),
		citations.NewInMemoryRegistry(),
		NoOpEvidenceTraceLinker{},
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/explain/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "decision_not_found" {
		t.Errorf("expected error=decision_not_found, got %q", body["error"])
	}
}

func TestExplain_MalformedUUID_Returns400(t *testing.T) {
	r := newExplainTestRouter(t,
		decision_metadata.NewInMemoryStore(),
		ethics_log.NewInMemoryStore(),
		citations.NewInMemoryRegistry(),
		NoOpEvidenceTraceLinker{},
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/explain/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "bad_decision_id" {
		t.Errorf("expected error=bad_decision_id, got %q", body["error"])
	}
}

func TestExplain_LinkedTraceTruncated(t *testing.T) {
	mdStore := decision_metadata.NewInMemoryStore()
	decisionID := uuid.New()
	if err := mdStore.Put(context.Background(), decision_metadata.Metadata{
		DecisionID:    decisionID,
		Component:     "kb-32",
		DecisionType:  "draft",
		AuditTraceRef: uuid.New(),
		Timestamp:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Linker returns 10 nodes — handler must pass through all of them.
	// Truncation policy lives in the linker, not the handler.
	nodes := make([]uuid.UUID, 10)
	for i := range nodes {
		nodes[i] = uuid.New()
	}
	r := newExplainTestRouter(t,
		mdStore,
		ethics_log.NewInMemoryStore(),
		citations.NewInMemoryRegistry(),
		stubLinker{nodes: nodes},
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/explain/"+decisionID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp ExplainResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.LinkedTrace) != 10 {
		t.Errorf("expected handler to pass through all 10 nodes, got %d", len(resp.LinkedTrace))
	}
}

// TestExplain_MetadataWithoutRecommendationLink_EmptyCitations is the
// backwards-compatibility guard for the uuid.Nil sentinel: when a decision
// is not linked to a recommendation (e.g., legacy fixtures or non-
// recommendation decisions), the handler must return 200 with an empty
// citations slice and must NOT call into the registry.
func TestExplain_MetadataWithoutRecommendationLink_EmptyCitations(t *testing.T) {
	mdStore := decision_metadata.NewInMemoryStore()
	decisionID := uuid.New()
	if err := mdStore.Put(context.Background(), decision_metadata.Metadata{
		DecisionID:    decisionID,
		Component:     "kb-32",
		DecisionType:  "draft",
		AuditTraceRef: uuid.New(),
		Timestamp:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	r := newExplainTestRouter(t,
		mdStore,
		ethics_log.NewInMemoryStore(),
		citations.NewInMemoryRegistry(),
		NoOpEvidenceTraceLinker{},
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/explain/"+decisionID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp ExplainResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Citations == nil {
		t.Fatal("citations must be a non-nil slice")
	}
	if len(resp.Citations) != 0 {
		t.Errorf("expected 0 citations (no rec-ID linkage in Metadata), got %d", len(resp.Citations))
	}
}

func TestExplain_RegistryError_DegradesGracefully(t *testing.T) {
	mdStore := decision_metadata.NewInMemoryStore()
	decisionID := uuid.New()
	if err := mdStore.Put(context.Background(), decision_metadata.Metadata{
		DecisionID:       decisionID,
		Component:        "kb-32",
		DecisionType:     "draft",
		AuditTraceRef:    uuid.New(),
		Timestamp:        time.Now().UTC(),
		RecommendationID: uuid.New(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// With a non-nil RecommendationID the handler now calls ListCitations.
	// The wrapped registry errors on every call — the handler must still
	// return 200 with an empty citations slice.
	reg := erroringRegistry{Registry: citations.NewInMemoryRegistry()}
	r := newExplainTestRouter(t,
		mdStore,
		ethics_log.NewInMemoryStore(),
		reg,
		NoOpEvidenceTraceLinker{},
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/explain/"+decisionID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (graceful degradation), got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp ExplainResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Citations) != 0 {
		t.Errorf("expected empty citations on registry error, got %d", len(resp.Citations))
	}
}

// erroringMetadataStore implements decision_metadata.Store and returns a
// fixed error from Get. Used to verify that a substrate failure surfaces as
// 500 metadata_lookup_failed rather than a misleading 404 decision_not_found.
type erroringMetadataStore struct {
	err error
}

func (e erroringMetadataStore) Put(_ context.Context, _ decision_metadata.Metadata) error {
	return nil
}

func (e erroringMetadataStore) Get(_ context.Context, _ uuid.UUID) (*decision_metadata.Metadata, error) {
	return nil, e.err
}

func (e erroringMetadataStore) QueryBySubject(_ context.Context, _ string) ([]decision_metadata.Metadata, error) {
	return nil, nil
}

func TestExplain_MetadataLookupError_Returns500(t *testing.T) {
	r := newExplainTestRouter(t,
		erroringMetadataStore{err: errors.New("simulated db outage")},
		ethics_log.NewInMemoryStore(),
		citations.NewInMemoryRegistry(),
		NoOpEvidenceTraceLinker{},
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/explain/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (substrate failure must not masquerade as 404), got %d (body=%s)", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "metadata_lookup_failed" {
		t.Errorf("expected error=metadata_lookup_failed, got %q", body["error"])
	}
}

func TestNoOpEvidenceTraceLinker_ReturnsNothing(t *testing.T) {
	nodes, err := NoOpEvidenceTraceLinker{}.LinkedNodes(context.Background(), uuid.New(), 5)
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if nodes != nil {
		t.Errorf("expected nil nodes, got %+v", nodes)
	}
}
