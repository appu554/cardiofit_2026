package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/appropriateness"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/reasoning"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Fakes for Pipeline dependencies
// ---------------------------------------------------------------------------

// fakeSubstrateClient returns a fixed ClinicalSnapshot.
type fakeSubstrateClient struct {
	snap kb32ctx.ClinicalSnapshot
	err  error
}

func (f *fakeSubstrateClient) SnapshotFor(_ context.Context, residentID uuid.UUID) (kb32ctx.ClinicalSnapshot, error) {
	if f.err != nil {
		return kb32ctx.ClinicalSnapshot{}, f.err
	}
	snap := f.snap
	snap.ResidentID = residentID
	return snap, nil
}

// fakeReasoningSource returns a fixed EvaluateRuleResult.
type fakeReasoningSource struct {
	result *reasoning.EvaluateRuleResult
	err    error
}

func (f *fakeReasoningSource) EvaluateRule(_ context.Context, ruleID string, _ uuid.UUID) (*reasoning.EvaluateRuleResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.result == nil {
		return &reasoning.EvaluateRuleResult{RuleID: ruleID, Triggered: false}, nil
	}
	res := *f.result
	res.RuleID = ruleID
	return &res, nil
}

// fakeAppropriatenessSource returns a fixed Assessment.
type fakeAppropriatenessSource struct {
	assessment appropriateness.Assessment
	err        error
}

func (f *fakeAppropriatenessSource) Assess(_ context.Context, _ *generator.Packet,
	_ kb32ctx.ClinicalSnapshot, _ reasoning.ApplicableRule) (appropriateness.Assessment, error) {
	return f.assessment, f.err
}

// ---------------------------------------------------------------------------
// Helper: build a Pipeline with test fakes
// ---------------------------------------------------------------------------

func buildTestPipeline(
	snapClient *fakeSubstrateClient,
	reasoningSrc *fakeReasoningSource,
	appSrc AppropriatenessSource,
) *Pipeline {
	assembler := kb32ctx.NewAssembler(snapClient)
	chain := reasoning.NewChainBuilder(reasoningSrc)
	return NewPipeline(assembler, chain, appSrc, nil)
}

// standardPassingSnap returns a minimal ClinicalSnapshot that produces
// urgency=red (RecentFall72h=true) and a valid care intensity.
func standardPassingSnap() kb32ctx.ClinicalSnapshot {
	return kb32ctx.ClinicalSnapshot{
		EGFR:          60.0,
		DBI:           0.5,
		ACB:           1,
		CFS:           4,
		CareIntensity: "active",
		RecentFall72h: true,
		AssessedAt:    time.Now(),
	}
}

// standardPassingAssessment returns a passing appropriateness assessment.
func standardPassingAssessment() appropriateness.Assessment {
	return appropriateness.Assessment{
		ClinicalWarrant:        3,
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}
}

// ---------------------------------------------------------------------------
// Handler tests
// ---------------------------------------------------------------------------

func TestHandleDraft_HappyPath_Drafted(t *testing.T) {
	residentID := uuid.New()
	authorID := uuid.New()

	snap := standardPassingSnap()
	snap.ResidentID = residentID

	pipeline := buildTestPipeline(
		&fakeSubstrateClient{snap: snap},
		&fakeReasoningSource{result: &reasoning.EvaluateRuleResult{
			Triggered: true,
			Type:      "MONITOR",
			Urgency:   "red",
		}},
		&fakeAppropriatenessSource{assessment: standardPassingAssessment()},
	)

	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	body, _ := json.Marshal(DraftRequest{
		RuleID:     "PostFall",
		ResidentID: residentID.String(),
		AuthorID:   authorID.String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body: %s", w.Code, w.Body.String())
	}

	var resp DraftResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.State != "drafted" {
		t.Errorf("state = %q; want drafted", resp.State)
	}
	if resp.ContentHash == "" {
		t.Errorf("content_hash is empty for drafted recommendation")
	}
	if resp.RecommendationID == "" {
		t.Errorf("recommendation_id is empty")
	}
	if resp.UrgencyTag != "red" {
		t.Errorf("urgency_tag = %q; want red (RecentFall72h=true)", resp.UrgencyTag)
	}
	if resp.HoldReason != "" {
		t.Errorf("hold_reason should be empty for drafted state; got %q", resp.HoldReason)
	}
}

func TestHandleDraft_AppropriatenessHold_ReturnsDetected(t *testing.T) {
	residentID := uuid.New()
	authorID := uuid.New()

	snap := standardPassingSnap()
	holdAssessment := appropriateness.Assessment{
		ClinicalWarrant:        2, // at threshold → holds
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}

	pipeline := buildTestPipeline(
		&fakeSubstrateClient{snap: snap},
		&fakeReasoningSource{result: &reasoning.EvaluateRuleResult{
			Triggered: true,
			Type:      "STOP",
			Urgency:   "red",
		}},
		&fakeAppropriatenessSource{assessment: holdAssessment},
	)

	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	body, _ := json.Marshal(DraftRequest{
		RuleID:     "RuleX",
		ResidentID: residentID.String(),
		AuthorID:   authorID.String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}

	var resp DraftResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.State != "detected" {
		t.Errorf("state = %q; want detected", resp.State)
	}
	if resp.HoldReason == "" {
		t.Errorf("hold_reason should be set for detected state")
	}
	if resp.ContentHash != "" {
		t.Errorf("content_hash should be empty when gate holds; got %q", resp.ContentHash)
	}
}

func TestHandleDraft_BadJSON_Returns400(t *testing.T) {
	pipeline := buildTestPipeline(
		&fakeSubstrateClient{},
		&fakeReasoningSource{},
		DefaultAppropriatenessSource{},
	)
	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400", w.Code)
	}
}

func TestHandleDraft_MissingField_Returns400(t *testing.T) {
	pipeline := buildTestPipeline(
		&fakeSubstrateClient{},
		&fakeReasoningSource{},
		DefaultAppropriatenessSource{},
	)
	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	// Missing AuthorID
	body, _ := json.Marshal(map[string]string{
		"rule_id":     "X",
		"resident_id": uuid.New().String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400 for missing author_id", w.Code)
	}
}

func TestHandleDraft_InvalidUUID_Returns400(t *testing.T) {
	pipeline := buildTestPipeline(
		&fakeSubstrateClient{},
		&fakeReasoningSource{},
		DefaultAppropriatenessSource{},
	)
	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	body, _ := json.Marshal(DraftRequest{
		RuleID:     "X",
		ResidentID: "not-a-uuid",
		AuthorID:   uuid.New().String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400 for invalid UUID", w.Code)
	}
}

func TestHandleDraft_SubstrateError_Returns500(t *testing.T) {
	pipeline := buildTestPipeline(
		&fakeSubstrateClient{err: context.DeadlineExceeded},
		&fakeReasoningSource{},
		DefaultAppropriatenessSource{},
	)
	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	body, _ := json.Marshal(DraftRequest{
		RuleID:     "X",
		ResidentID: uuid.New().String(),
		AuthorID:   uuid.New().String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d; want 500 for substrate error", w.Code)
	}
}

func TestHandleDraft_NoApplicableRules_Returns500(t *testing.T) {
	// fakeReasoningSource with Triggered=false → no applicable rules → generator returns ErrNoApplicableRules.
	snap := standardPassingSnap()
	pipeline := buildTestPipeline(
		&fakeSubstrateClient{snap: snap},
		&fakeReasoningSource{result: &reasoning.EvaluateRuleResult{Triggered: false}},
		DefaultAppropriatenessSource{},
	)
	handler := NewHandler(pipeline)
	r := gin.New()
	r.POST("/v1/craft/draft", handler.HandleDraft)

	body, _ := json.Marshal(DraftRequest{
		RuleID:     "NoFire",
		ResidentID: uuid.New().String(),
		AuthorID:   uuid.New().String(),
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d; want 500 when no applicable rules", w.Code)
	}
}
