package client

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

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
