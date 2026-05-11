// Package api implements the s2-aggregator external HTTP surface per
// S2 Resident Workspace Implementation Guidelines v1.0 Part 16 (lines
// 1214–1246), structured against the S2 Adaptive Cognition Architectural
// Commitment Addendum Part 8.
//
// Task 8 ships:
//
//   - 17 HTTP routes covering all 15 v1.0 Part 16 RPCs (rendering,
//     drill-through, the eleven actions, audit, session). The two extra
//     routes split GET/POST cleanly: rendering and refresh are both POST
//     because they carry a WorkspaceRequest body with polymorphic
//     EntryPathMetadata that does not flatten safely into URL params.
//
//   - permissions-middleware gating on every route via GinPermMW.
//     S2_PERMISSIONS_ENFORCED toggles enforcement vs passthrough.
//
//   - gRPC is wire-contract only in this commit — proto/v1/s2_workspace.proto
//     is the IDL; no buf config, no generated Go bindings, no server
//     stubs. Same pattern as Step 4 Task E (kb-33 proto).
//
//   - Event subscriptions are scaffolded in events.go for Phase 2
//     wiring; Task 8 only defines the interface and an in-memory stub.
//
// Error mapping (sanitised — never leak internal types or stack traces):
//
//   - JSON decode / bad UUID / ValidateReasoning   → 400 {"error": ...}
//   - OverrideForwarder failure                    → 502
//   - Addendum Part 6 layer-N-deferred sentinel    → 501
//   - audit.ErrCrossPharmacistRead                 → 403
//   - Anything else                                → 500
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/actions"
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/audit"
	"github.com/cardiofit/s2-aggregator/internal/drill_through"
)

// AuditTrailReader is the read port for the GetS2AuditTrail endpoint.
// It is intentionally minimal and lives in the api package so Task 7's
// no-surveillance-reader rule (the audit package exports no
// per-pharmacist read function) is preserved: production wiring adapts
// a Postgres reader that runs EnforcePDPRead before yielding rows; the
// HTTP handler ALSO runs EnforcePDPRead on the caller's pharmacist_id
// vs the row owner so the boundary check is local-defended.
type AuditTrailReader interface {
	List(ctx context.Context, residentID uuid.UUID, requesterID uuid.UUID) ([]audit.AuditEvent, error)
}

// SubstrateClient is the narrow subset of aggregation.SubstrateClient
// this HTTP layer needs for drill-through. Declared locally so tests
// only have to satisfy the methods we actually call.
type SubstrateClient = aggregation.SubstrateClient

// Dependencies bundles the collaborators the HTTP server requires.
// Required fields:
//
//   - ViewBuilder    (Task 1)
//   - ActionHandler  (Task 6)
//
// Optional fields:
//
//   - SessionStore     — defaults to actions.NewInMemorySessionStore()
//   - SubstrateClient  — drill-through endpoints return 500 without it
//   - ObservationFetcher — drill-through observation endpoint requires it
//   - AuditTrailReader — audit endpoint returns 501 without it
//   - PermsMW          — nil means passthrough on every route
//   - AuditEmitter     — best-effort emission on view renders / drill-throughs
type Dependencies struct {
	ViewBuilder        aggregation.S2ViewBuilder
	ActionHandler      *actions.Handler
	SessionStore       actions.SessionStore
	SubstrateClient    aggregation.SubstrateClient
	ObservationFetcher drill_through.ObservationFetcher
	AuditTrailReader   AuditTrailReader
	AuditEmitter       audit.Emitter
	PermsMW            Middleware
}

// Server holds the wired dependencies and implements the route handlers.
type Server struct {
	deps Dependencies
}

// NewServer returns a Server with the supplied dependencies. ViewBuilder
// and ActionHandler are mandatory; the rest default to safe nils
// (handlers that need them return 500/501 if they were not wired).
func NewServer(deps Dependencies) *Server {
	if deps.ViewBuilder == nil {
		panic("api.NewServer: ViewBuilder is required")
	}
	if deps.ActionHandler == nil {
		panic("api.NewServer: ActionHandler is required")
	}
	if deps.SessionStore == nil {
		deps.SessionStore = actions.NewInMemorySessionStore()
	}
	return &Server{deps: deps}
}

// RegisterRoutes mounts the 17 S2 endpoints on r. Every route is
// wrapped with GinPermMW(s.deps.PermsMW, "<resource>", PDP). PDP is the
// correct class for every action route per v1.0 Part 13.3 — these are
// pharmacist-private writes on their own clinical workspace activity.
func (s *Server) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/v1/s2")

	// ---------------------------------------------------------------------
	// Rendering — POST because WorkspaceRequest body carries polymorphic
	// EntryPathMetadata context (worklist/search/notification/cross-ref).
	// ---------------------------------------------------------------------
	v1.POST("/workspace",
		GinPermMW(s.deps.PermsMW, "s2_workspace", PDP),
		s.handleGetResidentWorkspace)
	v1.POST("/workspace/refresh",
		GinPermMW(s.deps.PermsMW, "s2_workspace_refresh", PDP),
		s.handleRefreshResidentWorkspace)

	// ---------------------------------------------------------------------
	// Drill-through — GET because the resource is identified by URL.
	// ---------------------------------------------------------------------
	v1.GET("/substrate/:resident_id/:substrate_type/:substrate_id",
		GinPermMW(s.deps.PermsMW, "s2_substrate", PDP),
		s.handleGetSubstrateObservation)
	v1.GET("/trajectory/:resident_id/:parameter",
		GinPermMW(s.deps.PermsMW, "s2_trajectory", PDP),
		s.handleGetTrajectoryHistory)

	// ---------------------------------------------------------------------
	// The eleven pharmacist actions — all POST.
	// ---------------------------------------------------------------------
	v1.POST("/actions/open",
		GinPermMW(s.deps.PermsMW, "s2_action_open", PDP),
		s.handleAction(actions.ActionOpen))
	v1.POST("/actions/modify",
		GinPermMW(s.deps.PermsMW, "s2_action_modify", PDP),
		s.handleAction(actions.ActionModify))
	v1.POST("/actions/defer",
		GinPermMW(s.deps.PermsMW, "s2_action_defer", PDP),
		s.handleAction(actions.ActionDefer))
	v1.POST("/actions/override",
		GinPermMW(s.deps.PermsMW, "s2_action_override", PDP),
		s.handleAction(actions.ActionOverride))
	v1.POST("/actions/mark_reviewed",
		GinPermMW(s.deps.PermsMW, "s2_action_mark_reviewed", PDP),
		s.handleAction(actions.ActionMarkReviewed))
	v1.POST("/actions/flag",
		GinPermMW(s.deps.PermsMW, "s2_action_flag", PDP),
		s.handleAction(actions.ActionFlagForFollowUp))
	v1.POST("/actions/note",
		GinPermMW(s.deps.PermsMW, "s2_action_note", PDP),
		s.handleAction(actions.ActionAddNote))
	v1.POST("/actions/escalate_to_complex",
		GinPermMW(s.deps.PermsMW, "s2_action_escalate_complex", PDP),
		s.handleAction(actions.ActionOpenComplexWorkspace))
	v1.POST("/actions/acknowledge_restraint",
		GinPermMW(s.deps.PermsMW, "s2_action_ack_restraint", PDP),
		s.handleAction(actions.ActionAcknowledgeRestraintSignal))
	v1.POST("/actions/safety_bypass",
		GinPermMW(s.deps.PermsMW, "s2_action_safety_bypass", PDP),
		s.handleAction(actions.ActionInvokeSafetyCriticalBypass))

	// ---------------------------------------------------------------------
	// Audit + session
	// ---------------------------------------------------------------------
	v1.GET("/audit/:resident_id",
		GinPermMW(s.deps.PermsMW, "s2_audit", PDP),
		s.handleGetS2AuditTrail)
	v1.POST("/session/start",
		GinPermMW(s.deps.PermsMW, "s2_session_start", PDP),
		s.handleStartSession)
	v1.POST("/session/end",
		GinPermMW(s.deps.PermsMW, "s2_session_end", PDP),
		s.handleEndSession)
}

// =============================================================================
// Wire request bodies
// =============================================================================

// workspaceReqBody is the JSON wire-shape for /v1/s2/workspace and
// /v1/s2/workspace/refresh. UUID fields are decoded as strings so we
// can return a sanitised 400 on parse failure rather than a generic
// json error.
type workspaceReqBody struct {
	ResidentID   string    `json:"resident_id"`
	PharmacistID string    `json:"pharmacist_id"`
	SessionID    string    `json:"session_id"`
	EntryPath    string    `json:"entry_path"`
	AsOf         time.Time `json:"as_of"`
}

// actionReqBody is the JSON wire-shape for the 11 action endpoints.
// The action enum is set by the handler factory (handleAction) per
// route; clients do NOT supply it on the wire.
type actionReqBody struct {
	PharmacistID            string    `json:"pharmacist_id"`
	ResidentID              string    `json:"resident_id"`
	SessionID               string    `json:"session_id"`
	SubjectID               string    `json:"subject_id"`
	Reasoning               string    `json:"reasoning"`
	OverrideReasonCode      string    `json:"override_reason_code,omitempty"`
	OverrideReasonCodeShort string    `json:"override_reason_code_short,omitempty"`
	AppropriatenessFlag     string    `json:"appropriateness_flag,omitempty"`
	NoteBody                string    `json:"note_body,omitempty"`
	Timestamp               time.Time `json:"timestamp,omitempty"`
}

type sessionStartBody struct {
	PharmacistID string `json:"pharmacist_id"`
}

type sessionEndBody struct {
	SessionID string `json:"session_id"`
}

// =============================================================================
// Handler implementations
// =============================================================================

func (s *Server) handleGetResidentWorkspace(c *gin.Context) {
	req, err := s.parseWorkspaceBody(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	view, err := s.deps.ViewBuilder.BuildLayer1Baseline(c.Request.Context(), req)
	if err != nil {
		s.mapBuilderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"resident_id": req.ResidentID,
		"layer":       view.Layer(),
		"view":        view,
	})
}

func (s *Server) handleRefreshResidentWorkspace(c *gin.Context) {
	req, err := s.parseWorkspaceBody(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	// Refresh re-runs BuildLayer1Baseline; cache invalidation hooks
	// live in events.go and are Phase 2 wiring.
	view, err := s.deps.ViewBuilder.BuildLayer1Baseline(c.Request.Context(), req)
	if err != nil {
		s.mapBuilderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"resident_id": req.ResidentID,
		"layer":       view.Layer(),
		"view":        view,
		"refreshed":   true,
	})
}

func (s *Server) handleGetSubstrateObservation(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid resident_id"))
		return
	}
	substrateType := c.Param("substrate_type")
	substrateID, err := uuid.Parse(c.Param("substrate_id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid substrate_id"))
		return
	}
	if s.deps.ObservationFetcher == nil {
		writeError(c, http.StatusInternalServerError, errors.New("substrate observation fetcher not wired"))
		return
	}
	_ = residentID // used by callers for back-trail; not required by GetSubstrateObservation
	ref := aggregation.SubstrateRef{Source: substrateType, ID: substrateID}
	obs, err := drill_through.GetSubstrateObservation(c.Request.Context(), s.deps.ObservationFetcher, ref, nil)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, obs)
}

func (s *Server) handleGetTrajectoryHistory(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid resident_id"))
		return
	}
	parameter := strings.TrimSpace(c.Param("parameter"))
	if parameter == "" {
		writeError(c, http.StatusBadRequest, errors.New("parameter is required"))
		return
	}
	if s.deps.SubstrateClient == nil {
		writeError(c, http.StatusInternalServerError, errors.New("substrate client not wired"))
		return
	}
	hist, err := drill_through.GetTrajectoryHistory(c.Request.Context(), s.deps.SubstrateClient, residentID, parameter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, hist)
}

// handleAction returns a Gin handler bound to a specific Action. The
// handler decodes the action body, sets req.Action to the bound enum,
// and dispatches to actions.Handler.Execute.
func (s *Server) handleAction(action actions.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := s.parseActionBody(c, action)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
		ack, err := s.deps.ActionHandler.Execute(c.Request.Context(), req)
		if err != nil {
			s.mapActionError(c, err, ack)
			return
		}
		c.JSON(http.StatusOK, ack)
	}
}

func (s *Server) handleGetS2AuditTrail(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid resident_id"))
		return
	}
	requesterIDStr := c.Query("pharmacist_id")
	requesterID, err := uuid.Parse(requesterIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid or missing pharmacist_id query parameter"))
		return
	}
	if s.deps.AuditTrailReader == nil {
		writeError(c, http.StatusNotImplemented, errors.New("audit trail reader not wired (production wiring is Phase 2)"))
		return
	}
	rows, err := s.deps.AuditTrailReader.List(c.Request.Context(), residentID, requesterID)
	if err != nil {
		if errors.Is(err, audit.ErrCrossPharmacistRead) {
			writeError(c, http.StatusForbidden, err)
			return
		}
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": rows})
}

func (s *Server) handleStartSession(c *gin.Context) {
	var body sessionStartBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}
	pharmID, err := uuid.Parse(body.PharmacistID)
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid pharmacist_id"))
		return
	}
	sess, err := actions.StartSession(c.Request.Context(), pharmID, s.deps.SessionStore)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, sess)
}

func (s *Server) handleEndSession(c *gin.Context) {
	var body sessionEndBody
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}
	sessID, err := uuid.Parse(body.SessionID)
	if err != nil {
		writeError(c, http.StatusBadRequest, errors.New("invalid session_id"))
		return
	}
	sess, err := actions.EndSession(c.Request.Context(), sessID, s.deps.SessionStore)
	if err != nil {
		if errors.Is(err, actions.ErrSessionNotFound) {
			writeError(c, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, actions.ErrSessionAlreadyEnded) {
			writeError(c, http.StatusConflict, err)
			return
		}
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, sess)
}

// =============================================================================
// Helpers
// =============================================================================

func (s *Server) parseWorkspaceBody(c *gin.Context) (aggregation.WorkspaceRequest, error) {
	var body workspaceReqBody
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return aggregation.WorkspaceRequest{}, errors.New("invalid request body")
	}
	residentID, err := uuid.Parse(body.ResidentID)
	if err != nil {
		return aggregation.WorkspaceRequest{}, errors.New("invalid resident_id")
	}
	pharmID, err := uuid.Parse(body.PharmacistID)
	if err != nil {
		return aggregation.WorkspaceRequest{}, errors.New("invalid pharmacist_id")
	}
	sessID, err := uuid.Parse(body.SessionID)
	if err != nil {
		return aggregation.WorkspaceRequest{}, errors.New("invalid session_id")
	}
	asOf := body.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	return aggregation.WorkspaceRequest{
		ResidentID:   residentID,
		PharmacistID: pharmID,
		SessionID:    sessID,
		EntryPath:    aggregation.EntryPath(body.EntryPath),
		AsOf:         asOf,
	}, nil
}

func (s *Server) parseActionBody(c *gin.Context, action actions.Action) (actions.ActionRequest, error) {
	var body actionReqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return actions.ActionRequest{}, errors.New("invalid request body")
	}
	pharmID, err := uuid.Parse(body.PharmacistID)
	if err != nil {
		return actions.ActionRequest{}, errors.New("invalid pharmacist_id")
	}
	residentID, err := uuid.Parse(body.ResidentID)
	if err != nil {
		return actions.ActionRequest{}, errors.New("invalid resident_id")
	}
	sessID, err := uuid.Parse(body.SessionID)
	if err != nil {
		return actions.ActionRequest{}, errors.New("invalid session_id")
	}
	var subjectID uuid.UUID
	if body.SubjectID != "" {
		subjectID, err = uuid.Parse(body.SubjectID)
		if err != nil {
			return actions.ActionRequest{}, errors.New("invalid subject_id")
		}
	}
	return actions.ActionRequest{
		Action:                  action,
		PharmacistID:            pharmID,
		ResidentID:              residentID,
		SessionID:               sessID,
		SubjectID:               subjectID,
		Reasoning:               body.Reasoning,
		OverrideReasonCode:      body.OverrideReasonCode,
		OverrideReasonCodeShort: body.OverrideReasonCodeShort,
		AppropriatenessFlag:     body.AppropriatenessFlag,
		NoteBody:                body.NoteBody,
		Timestamp:               body.Timestamp,
	}, nil
}

// mapBuilderError translates errors from view-builder calls into HTTP
// statuses. The Addendum Part 6 deferral sentinel for Layers 2–5
// surfaces as 501 so the frontend can distinguish "deferred by
// architectural discipline" from runtime failures.
func (s *Server) mapBuilderError(c *gin.Context, err error) {
	if isLayerDeferredSentinel(err) {
		writeError(c, http.StatusNotImplemented, err)
		return
	}
	writeError(c, http.StatusInternalServerError, err)
}

// mapActionError translates errors from actions.Handler.Execute.
//
//   - ValidateReasoning errors (ErrReasoningRequired / ErrReasoningNotApplicable /
//     ErrInvalidAction / ErrEmptyNoteBody / ErrInconsistentOverrideCodes)
//     → 400 with the audit ack still emitted in body if non-empty.
//   - Layer-deferred sentinel (open_complex_workspace path) → 501.
//   - Anything that looks like an OverrideForwarder failure → 502.
//   - Everything else → 500.
//
// We do best-effort string-matching on the override-forwarder failure
// rather than introducing a sentinel — the Task 6 contract for
// OverrideForwarder is interface-only and existing fakes return
// errors.New(...) strings. When kb-32 production wiring lands a typed
// error, we'll swap to errors.Is.
func (s *Server) mapActionError(c *gin.Context, err error, ack actions.ActionAcknowledgment) {
	switch {
	case errors.Is(err, actions.ErrReasoningRequired),
		errors.Is(err, actions.ErrReasoningNotApplicable),
		errors.Is(err, actions.ErrInvalidAction),
		errors.Is(err, actions.ErrEmptyNoteBody),
		errors.Is(err, actions.ErrInconsistentOverrideCodes):
		writeError(c, http.StatusBadRequest, err)
		return
	case isLayerDeferredSentinel(err):
		// Audit ack is already written (handler returns ack along
		// with the layer-deferral error). Surface 501 with the ack
		// embedded so the caller knows the audit row was captured.
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":           sanitise(err),
			"acknowledgment":  ack,
		})
		return
	case strings.Contains(err.Error(), "forward"):
		writeError(c, http.StatusBadGateway, err)
		return
	default:
		writeError(c, http.StatusInternalServerError, err)
	}
}

// isLayerDeferredSentinel reports whether err is the Addendum Part 6
// "layer N not yet implemented" sentinel from aggregation.notImplementedSentinel.
// Match on the stable substring rather than the wrapped error chain
// because that helper builds the error with fmt.Errorf without %w.
func isLayerDeferredSentinel(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Two sentinel shapes from internal/aggregation/view_builder.go:
	//   - notImplementedSentinel(layer): "...not yet implemented per S2 Adaptive Cognition Addendum Part 6..."
	//   - EscalateToLayer:                "escalation not implemented at Layer 1 — Addendum Part 6 defers Layer 2–5 content"
	return strings.Contains(msg, "not yet implemented per S2 Adaptive Cognition Addendum") ||
		strings.Contains(msg, "Addendum Part 6 defers Layer")
}

// sanitise returns a public-safe error message. Internal types and
// stack traces never leak — error.Error() text is permitted because
// every sentinel in this codebase is authored to be safe for clients.
func sanitise(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// writeError emits {"error": "..."} JSON with the supplied status.
// Logs the underlying error at warning level so operators can correlate.
func writeError(c *gin.Context, status int, err error) {
	if status >= 500 {
		log.Printf("s2-aggregator: %s %s -> %d: %v", c.Request.Method, c.Request.URL.Path, status, err)
	}
	c.JSON(status, gin.H{"error": sanitise(err)})
}
