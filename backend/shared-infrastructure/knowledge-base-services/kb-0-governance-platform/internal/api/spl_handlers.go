package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"kb-0-governance-platform/internal/database"
)

// =============================================================================
// SPL REVIEW SERVER
// =============================================================================
// Handlers for SPL FactStore Pipeline review workflows. Consumed by the
// Next.js governance dashboard's SPL Review pages for pharmacist triage,
// fact review, and drug sign-off.
//
// Route prefix: /api/v2/spl/
//
// Endpoints:
//   GET  /completeness               — All drugs, latest reports
//   GET  /completeness/{drug}        — Single drug report
//   GET  /facts                      — Paginated facts with filters
//   GET  /facts/pending/{drug}       — Pending review queue for drug
//   GET  /facts/sample/{drug}        — Random sample of approved facts
//   POST /facts/{id}/review          — Submit review decision
//   GET  /source/{docId}/section/{code} — Section HTML for source panel
//   GET  /triage                     — Triage dashboard data
//   POST /signoff/{drug}             — Submit sign-off attestation
//   GET  /signoff/{drug}             — Get existing sign-off
// =============================================================================

// SPLServer handles HTTP requests for SPL review workflows.
type SPLServer struct {
	store  *database.SPLStore
	router *http.ServeMux
}

// NewSPLServer creates a new SPL review API server.
func NewSPLServer(store *database.SPLStore) *SPLServer {
	s := &SPLServer{
		store:  store,
		router: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler with CORS for the Next.js frontend.
func (s *SPLServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-ID, X-Reviewer-ID")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.router.ServeHTTP(w, r)
}

func (s *SPLServer) registerRoutes() {
	// Completeness reports
	s.router.HandleFunc("GET /api/v2/spl/completeness", s.handleGetAllCompleteness)
	s.router.HandleFunc("GET /api/v2/spl/completeness/{drug}", s.handleGetCompletenessByDrug)

	// Derived facts
	s.router.HandleFunc("GET /api/v2/spl/facts", s.handleGetFacts)
	s.router.HandleFunc("GET /api/v2/spl/facts/pending/{drug}", s.handleGetPendingFacts)
	s.router.HandleFunc("GET /api/v2/spl/facts/sample/{drug}", s.handleGetSampleFacts)
	s.router.HandleFunc("POST /api/v2/spl/facts/{id}/review", s.handleReviewFact)

	// Source HTML
	s.router.HandleFunc("GET /api/v2/spl/source/{docId}/section/{code}", s.handleGetSourceSection)

	// Triage
	s.router.HandleFunc("GET /api/v2/spl/triage", s.handleGetTriage)

	// Sign-off
	s.router.HandleFunc("POST /api/v2/spl/signoff/{drug}", s.handleSubmitSignOff)
	s.router.HandleFunc("GET /api/v2/spl/signoff/{drug}", s.handleGetSignOff)
}

// =============================================================================
// COMPLETENESS HANDLERS
// =============================================================================

func (s *SPLServer) handleGetAllCompleteness(w http.ResponseWriter, r *http.Request) {
	reports, err := s.store.GetAllCompleteness(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": reports,
		"total": len(reports),
	})
}

func (s *SPLServer) handleGetCompletenessByDrug(w http.ResponseWriter, r *http.Request) {
	drug := r.PathValue("drug")
	if drug == "" {
		respondError(w, http.StatusBadRequest, "drug name required")
		return
	}

	report, err := s.store.GetCompletenessByDrug(r.Context(), drug)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, report)
}

// =============================================================================
// FACT HANDLERS
// =============================================================================

func (s *SPLServer) handleGetFacts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	drug := q.Get("drug")
	if drug == "" {
		respondError(w, http.StatusBadRequest, "drug query parameter required")
		return
	}

	filters := database.SPLFactFilters{
		DrugName:         drug,
		FactType:         q.Get("factType"),
		GovernanceStatus: q.Get("status"),
		ExtractionMethod: q.Get("method"),
	}

	page, pageSize := database.ParsePagination(q.Get("page"), q.Get("pageSize"), 1, 100)

	facts, total, err := s.store.GetFactsByDrug(r.Context(), drug, filters, page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":    facts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"hasMore":  page*pageSize < total,
	})
}

func (s *SPLServer) handleGetPendingFacts(w http.ResponseWriter, r *http.Request) {
	drug := r.PathValue("drug")
	if drug == "" {
		respondError(w, http.StatusBadRequest, "drug name required")
		return
	}

	limit := 500
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parsePositiveInt(v); err == nil && n <= 1000 {
			limit = n
		}
	}

	facts, err := s.store.GetPendingReviewFacts(r.Context(), drug, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": facts,
		"total": len(facts),
	})
}

func (s *SPLServer) handleGetSampleFacts(w http.ResponseWriter, r *http.Request) {
	drug := r.PathValue("drug")
	if drug == "" {
		respondError(w, http.StatusBadRequest, "drug name required")
		return
	}

	size := 10
	if v := r.URL.Query().Get("size"); v != "" {
		if n, err := parsePositiveInt(v); err == nil && n <= 50 {
			size = n
		}
	}

	facts, err := s.store.GetAutoApprovedSample(r.Context(), drug, size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": facts,
		"total": len(facts),
	})
}

// ReviewDecisionRequest represents a pharmacist review action from the UI.
type ReviewDecisionRequest struct {
	Decision   string `json:"decision"`   // CONFIRM, REJECT, EDIT, ESCALATE
	ReviewerID string `json:"reviewerId"`
	Reason     string `json:"reason"`
	EditedData json.RawMessage `json:"editedData,omitempty"`
}

func (s *SPLServer) handleReviewFact(w http.ResponseWriter, r *http.Request) {
	factID := r.PathValue("id")
	if factID == "" {
		respondError(w, http.StatusBadRequest, "fact ID required")
		return
	}

	var req ReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}
	if req.Decision == "" {
		respondError(w, http.StatusBadRequest, "decision required")
		return
	}

	err := s.store.SubmitReview(r.Context(), factID, req.Decision, req.ReviewerID, req.Reason)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"factId":  factID,
		"status":  req.Decision,
	})
}

// =============================================================================
// SOURCE HTML HANDLER
// =============================================================================

func (s *SPLServer) handleGetSourceSection(w http.ResponseWriter, r *http.Request) {
	docID := r.PathValue("docId")
	code := r.PathValue("code")

	if docID == "" || code == "" {
		respondError(w, http.StatusBadRequest, "docId and section code required")
		return
	}

	rawXML, sectionName, err := s.store.GetSectionHTML(r.Context(), docID, code)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	// If raw_html is empty in DB, fetch on-demand from DailyMed
	if rawXML == "" {
		setID, err := s.store.GetDocumentSetID(r.Context(), docID)
		if err != nil {
			log.Printf("[SPL] Failed to get set_id for doc %s: %v", docID, err)
			respondError(w, http.StatusNotFound, "source document not found")
			return
		}

		fetched, err := FetchSectionHTML(r.Context(), setID, code)
		if err != nil {
			log.Printf("[SPL] DailyMed fetch failed for set_id=%s section=%s: %v", setID, code, err)
			respondError(w, http.StatusNotFound, "section HTML not available")
			return
		}

		rawXML = fetched

		// Cache back to DB so future requests don't re-fetch
		if cacheErr := s.store.UpdateSectionRawHTML(r.Context(), docID, code, rawXML); cacheErr != nil {
			log.Printf("[SPL] Failed to cache raw_html for doc=%s section=%s: %v", docID, code, cacheErr)
			// Non-fatal — continue serving the fetched content
		}
	}

	// Transform SPL XML → browser-renderable HTML
	html := TransformSPLToHTML(rawXML)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"html":        html,
		"sectionName": sectionName,
		"sectionCode": code,
		"documentId":  docID,
	})
}

// =============================================================================
// TRIAGE HANDLER
// =============================================================================

func (s *SPLServer) handleGetTriage(w http.ResponseWriter, r *http.Request) {
	states, err := s.store.GetTriageDashboard(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": states,
		"total": len(states),
	})
}

// =============================================================================
// SIGN-OFF HANDLERS
// =============================================================================

// SignOffRequest represents a sign-off submission from the UI.
type SignOffRequest struct {
	DrugName                  string          `json:"drugName"`
	RxCUI                     string          `json:"rxcui"`
	TotalFacts                int             `json:"totalFacts"`
	Confirmed                 int             `json:"confirmed"`
	Edited                    int             `json:"edited"`
	Rejected                  int             `json:"rejected"`
	Added                     int             `json:"added"`
	AutoApprovedSampleSize    int             `json:"autoApprovedSampleSize"`
	AutoApprovedSampleErrors  int             `json:"autoApprovedSampleErrors"`
	FactTypeCoverage          json.RawMessage `json:"factTypeCoverage"`
	ReviewerID                string          `json:"reviewerId"`
	Attestation               string          `json:"attestation"`
	SignedAt                  string          `json:"signedAt"`
}

func (s *SPLServer) handleSubmitSignOff(w http.ResponseWriter, r *http.Request) {
	drug := r.PathValue("drug")
	if drug == "" {
		respondError(w, http.StatusBadRequest, "drug name required")
		return
	}

	var req SignOffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}
	if req.Attestation == "" {
		respondError(w, http.StatusBadRequest, "attestation required")
		return
	}

	signedAt, err := time.Parse(time.RFC3339, req.SignedAt)
	if err != nil {
		signedAt = time.Now()
	}

	signOff := &database.SPLSignOff{
		DrugName:                  drug,
		RxCUI:                     req.RxCUI,
		TotalFacts:                req.TotalFacts,
		Confirmed:                 req.Confirmed,
		Edited:                    req.Edited,
		Rejected:                  req.Rejected,
		Added:                     req.Added,
		AutoApprovedSampleSize:    req.AutoApprovedSampleSize,
		AutoApprovedSampleErrors:  req.AutoApprovedSampleErrors,
		FactTypeCoverage:          req.FactTypeCoverage,
		ReviewerID:                req.ReviewerID,
		Attestation:               req.Attestation,
		SignedAt:                  signedAt,
	}

	if err := s.store.SubmitSignOff(r.Context(), signOff); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"id":       signOff.ID,
		"drug":     drug,
		"signedAt": signOff.SignedAt,
	})
}

func (s *SPLServer) handleGetSignOff(w http.ResponseWriter, r *http.Request) {
	drug := r.PathValue("drug")
	if drug == "" {
		respondError(w, http.StatusBadRequest, "drug name required")
		return
	}

	signOff, err := s.store.GetSignOff(r.Context(), drug)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if signOff == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"exists": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, signOff)
}

// =============================================================================
// HELPERS
// =============================================================================

func parsePositiveInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("value must be positive: %d", n)
	}
	return n, nil
}
