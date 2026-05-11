package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/actions"
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/audit"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// fakeObservationFetcher returns a fixed observation by ID.
type fakeObservationFetcher struct {
	obs substrate_types.Observation
}

func (f *fakeObservationFetcher) GetObservationByID(_ context.Context, _ uuid.UUID) (substrate_types.Observation, error) {
	return f.obs, nil
}

// fakeAuditReader returns a fixed list, enforcing owner-only access.
type fakeAuditReader struct {
	owner  uuid.UUID
	events []audit.AuditEvent
}

func (f *fakeAuditReader) List(_ context.Context, _ uuid.UUID, requesterID uuid.UUID) ([]audit.AuditEvent, error) {
	if err := audit.EnforcePDPRead(requesterID, f.owner); err != nil {
		return nil, err
	}
	return f.events, nil
}

// buildTestServer constructs a Server wired with in-memory fakes that
// the action handler tests already exercise. Every handler endpoint can
// be hit without panicking; tests assert on the specific edge they care
// about.
func buildTestServer(t *testing.T) (*gin.Engine, *Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store := actions.NewInMemoryActionStore()
	sessions := actions.NewInMemorySessionStore()
	forwarder := actions.NewInMemoryOverrideForwarder()
	vb := aggregation.NewDefaultViewBuilder()
	h := actions.NewHandler(store, sessions, forwarder, vb)

	subClient := aggregation.NewInMemorySubstrateClient()

	srv := NewServer(Dependencies{
		ViewBuilder:        vb,
		ActionHandler:      h,
		SessionStore:       sessions,
		SubstrateClient:    subClient,
		ObservationFetcher: &fakeObservationFetcher{obs: substrate_types.Observation{Confidence: "moderate"}},
		AuditTrailReader:   nil, // wired per-test when needed
	})
	r := gin.New()
	srv.RegisterRoutes(r)
	return r, srv
}

func doJSON(r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestRoutes_GetResidentWorkspace_OK(t *testing.T) {
	r, _ := buildTestServer(t)
	body := workspaceReqBody{
		ResidentID:   uuid.New().String(),
		PharmacistID: uuid.New().String(),
		SessionID:    uuid.New().String(),
		EntryPath:    "worklist",
		AsOf:         time.Now().UTC(),
	}
	w := doJSON(r, http.MethodPost, "/v1/s2/workspace", body)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_GetResidentWorkspace_BadJSON(t *testing.T) {
	r, _ := buildTestServer(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/s2/workspace", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestRoutes_GetResidentWorkspace_BadUUID(t *testing.T) {
	r, _ := buildTestServer(t)
	body := workspaceReqBody{
		ResidentID:   "not-a-uuid",
		PharmacistID: uuid.New().String(),
		SessionID:    uuid.New().String(),
	}
	w := doJSON(r, http.MethodPost, "/v1/s2/workspace", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestRoutes_RefreshWorkspace_OK(t *testing.T) {
	r, _ := buildTestServer(t)
	body := workspaceReqBody{
		ResidentID:   uuid.New().String(),
		PharmacistID: uuid.New().String(),
		SessionID:    uuid.New().String(),
		EntryPath:    "search",
	}
	w := doJSON(r, http.MethodPost, "/v1/s2/workspace/refresh", body)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_SubstrateObservation_OK(t *testing.T) {
	r, _ := buildTestServer(t)
	path := "/v1/s2/substrate/" + uuid.New().String() + "/observation/" + uuid.New().String()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_SubstrateObservation_BadResident(t *testing.T) {
	r, _ := buildTestServer(t)
	path := "/v1/s2/substrate/bad-uuid/observation/" + uuid.New().String()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d", w.Code)
	}
}

func TestRoutes_TrajectoryHistory_OK(t *testing.T) {
	r, _ := buildTestServer(t)
	path := "/v1/s2/trajectory/" + uuid.New().String() + "/bp_systolic"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

// Helper: build a valid action body for the supplied action.
func validActionBody(t *testing.T, action actions.Action) actionReqBody {
	t.Helper()
	body := actionReqBody{
		PharmacistID: uuid.New().String(),
		ResidentID:   uuid.New().String(),
		SessionID:    uuid.New().String(),
		SubjectID:    uuid.New().String(),
	}
	// Add required fields per action.
	switch action {
	case actions.ActionModify,
		actions.ActionInvokeSafetyCriticalBypass:
		body.Reasoning = "documented clinical rationale here"
	case actions.ActionOverride:
		body.Reasoning = "documented clinical rationale here"
		body.OverrideReasonCode = "clinical_judgment"
	case actions.ActionAddNote:
		body.NoteBody = "documented clinical note body"
	}
	// Open session first so RecordActionInSession succeeds.
	sessID := uuid.New()
	body.SessionID = sessID.String()
	return body
}

func TestRoutes_ActionOpen_OK(t *testing.T) {
	r, srv := buildTestServer(t)
	// Pre-create a session matching the body.
	body := validActionBody(t, actions.ActionOpen)
	sessID, _ := uuid.Parse(body.SessionID)
	pharm, _ := uuid.Parse(body.PharmacistID)
	_ = srv.deps.SessionStore.Create(context.Background(), actions.SessionContext{
		SessionID: sessID, PharmacistID: pharm, StartedAt: time.Now().UTC(),
	})
	w := doJSON(r, http.MethodPost, "/v1/s2/actions/open", body)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_ActionModify_MissingReasoning_400(t *testing.T) {
	r, srv := buildTestServer(t)
	body := validActionBody(t, actions.ActionModify)
	body.Reasoning = "" // strip the mandatory field
	sessID, _ := uuid.Parse(body.SessionID)
	pharm, _ := uuid.Parse(body.PharmacistID)
	_ = srv.deps.SessionStore.Create(context.Background(), actions.SessionContext{
		SessionID: sessID, PharmacistID: pharm, StartedAt: time.Now().UTC(),
	})
	w := doJSON(r, http.MethodPost, "/v1/s2/actions/modify", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_ActionEscalateComplex_501(t *testing.T) {
	r, srv := buildTestServer(t)
	body := validActionBody(t, actions.ActionOpenComplexWorkspace)
	sessID, _ := uuid.Parse(body.SessionID)
	pharm, _ := uuid.Parse(body.PharmacistID)
	_ = srv.deps.SessionStore.Create(context.Background(), actions.SessionContext{
		SessionID: sessID, PharmacistID: pharm, StartedAt: time.Now().UTC(),
	})
	w := doJSON(r, http.MethodPost, "/v1/s2/actions/escalate_to_complex", body)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_Audit_NotWired_501(t *testing.T) {
	r, _ := buildTestServer(t)
	path := "/v1/s2/audit/" + uuid.New().String() + "?pharmacist_id=" + uuid.New().String()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("got %d", w.Code)
	}
}

func TestRoutes_Audit_CrossPharmacistRead_403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := actions.NewInMemoryActionStore()
	sessions := actions.NewInMemorySessionStore()
	forwarder := actions.NewInMemoryOverrideForwarder()
	vb := aggregation.NewDefaultViewBuilder()
	h := actions.NewHandler(store, sessions, forwarder, vb)
	owner := uuid.New()
	reader := &fakeAuditReader{owner: owner}
	srv := NewServer(Dependencies{
		ViewBuilder:      vb,
		ActionHandler:    h,
		AuditTrailReader: reader,
	})
	r := gin.New()
	srv.RegisterRoutes(r)
	// Request as a different pharmacist → cross-pharmacist read forbidden.
	path := "/v1/s2/audit/" + uuid.New().String() + "?pharmacist_id=" + uuid.New().String()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_SessionLifecycle(t *testing.T) {
	r, _ := buildTestServer(t)
	pharm := uuid.New().String()
	w := doJSON(r, http.MethodPost, "/v1/s2/session/start", sessionStartBody{PharmacistID: pharm})
	if w.Code != http.StatusOK {
		t.Fatalf("start got %d body=%s", w.Code, w.Body.String())
	}
	var sess actions.SessionContext
	if err := json.Unmarshal(w.Body.Bytes(), &sess); err != nil {
		t.Fatalf("decode: %v", err)
	}
	w2 := doJSON(r, http.MethodPost, "/v1/s2/session/end", sessionEndBody{SessionID: sess.SessionID.String()})
	if w2.Code != http.StatusOK {
		t.Fatalf("end got %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestRoutes_SessionEnd_NotFound(t *testing.T) {
	r, _ := buildTestServer(t)
	w := doJSON(r, http.MethodPost, "/v1/s2/session/end", sessionEndBody{SessionID: uuid.New().String()})
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRoutes_BadJSON_ActionEndpoint(t *testing.T) {
	r, _ := buildTestServer(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/s2/actions/open", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d", w.Code)
	}
}

func TestRoutes_OverrideForward_502(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := actions.NewInMemoryActionStore()
	sessions := actions.NewInMemorySessionStore()
	forwarder := actions.NewInMemoryOverrideForwarder()
	forwarder.FailNext()
	vb := aggregation.NewDefaultViewBuilder()
	h := actions.NewHandler(store, sessions, forwarder, vb)
	srv := NewServer(Dependencies{ViewBuilder: vb, ActionHandler: h, SessionStore: sessions})
	r := gin.New()
	srv.RegisterRoutes(r)

	body := validActionBody(t, actions.ActionOverride)
	sessID, _ := uuid.Parse(body.SessionID)
	pharm, _ := uuid.Parse(body.PharmacistID)
	_ = sessions.Create(context.Background(), actions.SessionContext{
		SessionID: sessID, PharmacistID: pharm, StartedAt: time.Now().UTC(),
	})
	w := doJSON(r, http.MethodPost, "/v1/s2/actions/override", body)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}
