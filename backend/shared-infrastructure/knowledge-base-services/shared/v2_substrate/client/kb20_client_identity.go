package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
)

// IdentityMatchResponse mirrors the kb-20 wire format for
// POST /v2/identity/match. ReviewQueueEntryID is non-nil when the
// matcher returned LOW or NONE confidence and the service layer
// enqueued the decision for human verification.
type IdentityMatchResponse struct {
	Match                identity.MatchResult `json:"match"`
	EvidenceTraceNodeRef uuid.UUID            `json:"evidence_trace_node_ref"`
	ReviewQueueEntryID   *uuid.UUID           `json:"review_queue_entry_id,omitempty"`
}

// IdentityResolveResponse mirrors the kb-20 wire format for
// POST /v2/identity/review/:id/resolve. Rerouted is the count of
// prior identity_mappings rows that were repointed at the resolved
// resident as part of the post-hoc correction (Layer 2 §3.3).
type IdentityResolveResponse struct {
	Entry    *interfaces.IdentityReviewQueueEntry `json:"entry"`
	Rerouted int                                  `json:"rerouted_mappings"`
}

// MatchIdentity POSTs an IncomingIdentifier to /v2/identity/match and
// returns the matcher's MatchResult plus the audit cross-refs.
func (c *KB20Client) MatchIdentity(ctx context.Context, in identity.IncomingIdentifier) (*IdentityMatchResponse, error) {
	return doJSON[IdentityMatchResponse](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/identity/match", in)
}

// ListIdentityReviewQueue GETs /v2/identity/review-queue with pagination
// and an optional status filter. Pass status="" to get every status;
// the kb-20 handler treats that as "no filter".
func (c *KB20Client) ListIdentityReviewQueue(ctx context.Context, status string, limit, offset int) ([]interfaces.IdentityReviewQueueEntry, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u := c.baseURL + "/v2/identity/review-queue?" + q.Encode()
	out, err := doJSON[[]interfaces.IdentityReviewQueueEntry](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ResolveIdentityReview POSTs to /v2/identity/review/:id/resolve.
// resolvedRef must be a non-zero UUID — the kb-20 handler rejects
// nil refs with a 400.
func (c *KB20Client) ResolveIdentityReview(ctx context.Context, id, resolvedRef, resolvedBy uuid.UUID, note string) (*IdentityResolveResponse, error) {
	body := struct {
		ResolvedResidentRef uuid.UUID `json:"resolved_resident_ref"`
		ResolvedBy          uuid.UUID `json:"resolved_by"`
		ResolutionNote      string    `json:"resolution_note,omitempty"`
	}{
		ResolvedResidentRef: resolvedRef,
		ResolvedBy:          resolvedBy,
		ResolutionNote:      note,
	}
	return doJSON[IdentityResolveResponse](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/identity/review/"+id.String()+"/resolve", body)
}
