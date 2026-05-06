package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
)

// Reconciliation client methods (Wave 4 of Layer 2 substrate plan;
// Layer 2 doc §3.2).

// IngestDischargeDocument posts a parsed discharge document to kb-20
// and returns the persisted row + its medication lines.
func (c *KB20Client) IngestDischargeDocument(ctx context.Context, doc interfaces.DischargeDocument) (*interfaces.DischargeDocument, error) {
	return doJSON[interfaces.DischargeDocument](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/discharge-documents", doc)
}

// GetDischargeDocument fetches a discharge_documents row + lines by id.
func (c *KB20Client) GetDischargeDocument(ctx context.Context, id uuid.UUID) (*interfaces.DischargeDocument, error) {
	return doJSON[interfaces.DischargeDocument](ctx, c.http, http.MethodGet,
		c.baseURL+"/v2/discharge-documents/"+id.String(), nil)
}

// StartReconciliation creates the reconciliation worklist + decision
// rows for a previously-ingested discharge document.
func (c *KB20Client) StartReconciliation(ctx context.Context, in interfaces.ReconciliationStartInputs) (*interfaces.ReconciliationStartResult, error) {
	return doJSON[interfaces.ReconciliationStartResult](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/reconciliation/start", in)
}

// GetReconciliationWorklist fetches a worklist + its decision rows.
func (c *KB20Client) GetReconciliationWorklist(ctx context.Context, worklistRef uuid.UUID) (*interfaces.ReconciliationWorklist, []interfaces.ReconciliationDecision, error) {
	type bundle struct {
		Worklist  interfaces.ReconciliationWorklist  `json:"worklist"`
		Decisions []interfaces.ReconciliationDecision `json:"decisions"`
	}
	out, err := doJSON[bundle](ctx, c.http, http.MethodGet,
		c.baseURL+"/v2/reconciliation/"+worklistRef.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	return &out.Worklist, out.Decisions, nil
}

// ListReconciliationWorklists queries worklists filtered by role,
// facility, and status. Pass nil refs to omit the filter.
func (c *KB20Client) ListReconciliationWorklists(ctx context.Context, roleRef, facilityID *uuid.UUID, status string, limit, offset int) ([]interfaces.ReconciliationWorklist, error) {
	q := url.Values{}
	if roleRef != nil {
		q.Set("role_ref", roleRef.String())
	}
	if facilityID != nil {
		q.Set("facility_id", facilityID.String())
	}
	if status != "" {
		q.Set("status", status)
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	out, err := doJSON[[]interfaces.ReconciliationWorklist](ctx, c.http, http.MethodGet,
		c.baseURL+"/v2/reconciliation?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// DecideReconciliation records one ACOP decision against a
// reconciliation_decisions row.
func (c *KB20Client) DecideReconciliation(ctx context.Context, in interfaces.DecideReconciliationInputs) (*interfaces.ReconciliationDecision, error) {
	return doJSON[interfaces.ReconciliationDecision](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/reconciliation/"+in.WorklistRef.String()+"/lines/"+in.DecisionRef.String()+"/decide", in)
}

// FinaliseReconciliationWorklist marks the worklist completed and
// runs the write-back. Returns the resulting MedicineUse refs.
func (c *KB20Client) FinaliseReconciliationWorklist(ctx context.Context, worklistRef uuid.UUID, completedByRoleRef uuid.UUID) (*interfaces.FinaliseReconciliationResult, error) {
	body := map[string]string{"completed_by_role_ref": completedByRoleRef.String()}
	return doJSON[interfaces.FinaliseReconciliationResult](ctx, c.http, http.MethodPost,
		c.baseURL+"/v2/reconciliation/"+worklistRef.String()+"/finalise", body)
}
