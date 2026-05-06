package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

var _ = models.EvidenceTraceNode{} // keep models import live across edits

// ---------------------------------------------------------------------------
// Wave 5.2 — Lineage / Consequences / ReasoningWindow query API
// ---------------------------------------------------------------------------

// GetRecommendationLineage GETs the backward-evidence rollup for one
// Recommendation node.
func (c *KB20Client) GetRecommendationLineage(ctx context.Context, recID uuid.UUID, depth int) (*evidence_trace.Lineage, error) {
	u := c.baseURL + "/v2/evidence-trace/recommendations/" + recID.String() + "/lineage?depth=" + strconv.Itoa(depth)
	return doJSON[evidence_trace.Lineage](ctx, c.http, http.MethodGet, u, nil)
}

// GetObservationConsequences GETs the forward-consequence rollup for one
// observation-seed node.
func (c *KB20Client) GetObservationConsequences(ctx context.Context, obsID uuid.UUID, depth int) (*evidence_trace.Consequences, error) {
	u := c.baseURL + "/v2/evidence-trace/observations/" + obsID.String() + "/consequences?depth=" + strconv.Itoa(depth)
	return doJSON[evidence_trace.Consequences](ctx, c.http, http.MethodGet, u, nil)
}

// GetReasoningWindow GETs the regulator-audit window rollup for one
// resident across [from, to).
func (c *KB20Client) GetReasoningWindow(ctx context.Context, residentRef uuid.UUID, from, to time.Time) (*evidence_trace.ReasoningSummaryWindow, error) {
	q := url.Values{}
	q.Set("from", from.UTC().Format(time.RFC3339))
	q.Set("to", to.UTC().Format(time.RFC3339))
	u := c.baseURL + "/v2/residents/" + residentRef.String() + "/reasoning-window?" + q.Encode()
	return doJSON[evidence_trace.ReasoningSummaryWindow](ctx, c.http, http.MethodGet, u, nil)
}

// ---------------------------------------------------------------------------
// EvidenceTrace
// ---------------------------------------------------------------------------

// UpsertEvidenceTraceNode POSTs a node to /v2/evidence-trace/nodes.
func (c *KB20Client) UpsertEvidenceTraceNode(ctx context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error) {
	return doJSON[models.EvidenceTraceNode](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/evidence-trace/nodes", n)
}

// GetEvidenceTraceNode GETs a node by id.
func (c *KB20Client) GetEvidenceTraceNode(ctx context.Context, id uuid.UUID) (*models.EvidenceTraceNode, error) {
	return doJSON[models.EvidenceTraceNode](ctx, c.http, http.MethodGet,
		c.baseURL+"/v2/evidence-trace/nodes/"+id.String(), nil)
}

// InsertEvidenceTraceEdge POSTs an edge to /v2/evidence-trace/edges.
// The endpoint is idempotent on (from_node, to_node, edge_kind).
func (c *KB20Client) InsertEvidenceTraceEdge(ctx context.Context, e evidence_trace.Edge) error {
	_, err := doJSON[map[string]interface{}](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/evidence-trace/edges", e)
	return err
}

// TraceEvidenceForward GETs forward traversal from startNode capped at depth.
func (c *KB20Client) TraceEvidenceForward(ctx context.Context, startNode uuid.UUID, depth int) ([]models.EvidenceTraceNode, error) {
	u := c.baseURL + "/v2/evidence-trace/" + startNode.String() + "/forward?depth=" + strconv.Itoa(depth)
	out, err := doJSON[[]models.EvidenceTraceNode](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// TraceEvidenceBackward GETs backward traversal from startNode capped at depth.
func (c *KB20Client) TraceEvidenceBackward(ctx context.Context, startNode uuid.UUID, depth int) ([]models.EvidenceTraceNode, error) {
	u := c.baseURL + "/v2/evidence-trace/" + startNode.String() + "/backward?depth=" + strconv.Itoa(depth)
	out, err := doJSON[[]models.EvidenceTraceNode](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
