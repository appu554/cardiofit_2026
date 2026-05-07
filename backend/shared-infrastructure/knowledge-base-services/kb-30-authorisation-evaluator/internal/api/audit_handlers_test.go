package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/audit"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

func newServerForTest(t *testing.T) (*Server, *audit.Service) {
	t.Helper()
	a := audit.NewService()
	s := &Server{Audit: a}
	return s, a
}

func seedAudit(a *audit.Service, residentRef, credID uuid.UUID) audit.EvaluationRecord {
	rec := audit.EvaluationRecord{
		ID: uuid.New(),
		Query: evaluator.Query{
			Jurisdiction:       "AU/VIC",
			Role:               "personal_care_worker",
			ActionClass:        dsl.ActionAdminister,
			MedicationSchedule: "S4",
			ResidentRef:        residentRef,
			ActorRef:           uuid.New(),
		},
		Result: evaluator.Result{
			Decision:             dsl.DecisionDenied,
			Reason:               "Vic PCW exclusion",
			RuleID:               "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01",
			LegislativeReference: "DPCSA Amendment Act 2025 (Vic)",
		},
		EvaluatedAt:   time.Now(),
		CredentialIDs: []uuid.UUID{credID},
	}
	a.Record(rec)
	return rec
}

func TestAuditResident_FHIRBundle(t *testing.T) {
	s, a := newServerForTest(t)
	resident := uuid.New()
	cred := uuid.New()
	seedAudit(a, resident, cred)

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/resident/"+resident.String(), nil)
	w := httptest.NewRecorder()
	s.handleAuditResident(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var bundle map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bundle))
	assert.Equal(t, "Bundle", bundle["resourceType"])
	assert.Equal(t, "searchset", bundle["type"])
	assert.EqualValues(t, 1, bundle["total"])
}

func TestAuditResident_CSV(t *testing.T) {
	s, a := newServerForTest(t)
	resident := uuid.New()
	seedAudit(a, resident, uuid.New())

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/resident/"+resident.String()+"?format=csv", nil)
	w := httptest.NewRecorder()
	s.handleAuditResident(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Body.String(), "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01")
}

func TestAuditResident_JSON(t *testing.T) {
	s, a := newServerForTest(t)
	resident := uuid.New()
	seedAudit(a, resident, uuid.New())

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/resident/"+resident.String()+"?format=json", nil)
	w := httptest.NewRecorder()
	s.handleAuditResident(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var records []audit.EvaluationRecord
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &records))
	require.Len(t, records, 1)
	assert.Equal(t, dsl.DecisionDenied, records[0].Result.Decision)
}

func TestAuditCredential(t *testing.T) {
	s, a := newServerForTest(t)
	cred := uuid.New()
	seedAudit(a, uuid.New(), cred)

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/credential/"+cred.String()+"?format=json", nil)
	w := httptest.NewRecorder()
	s.handleAuditCredential(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var records []audit.EvaluationRecord
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &records))
	require.Len(t, records, 1)
}

func TestAuditJurisdiction(t *testing.T) {
	s, a := newServerForTest(t)
	seedAudit(a, uuid.New(), uuid.New())

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/jurisdiction/AU/VIC/medications/S4?format=json", nil)
	w := httptest.NewRecorder()
	s.handleAuditJurisdiction(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var records []audit.EvaluationRecord
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &records))
	require.Len(t, records, 1)
}

func TestAuditChain(t *testing.T) {
	s, a := newServerForTest(t)
	rec := seedAudit(a, uuid.New(), uuid.New())

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/authorisation/"+rec.ID.String()+"/chain", nil)
	w := httptest.NewRecorder()
	s.handleAuditChain(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var chain audit.AuthorisationChain
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &chain))
	assert.Equal(t, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", chain.RuleID)
	assert.Contains(t, chain.LegislativeReference, "DPCSA")
}

func TestAuditChain_NotFound(t *testing.T) {
	s, _ := newServerForTest(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/audit/authorisation/"+uuid.New().String()+"/chain", nil)
	w := httptest.NewRecorder()
	s.handleAuditChain(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
