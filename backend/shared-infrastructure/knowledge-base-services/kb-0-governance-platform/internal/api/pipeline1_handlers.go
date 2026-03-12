package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	"kb-0-governance-platform/internal/database"
	"kb-0-governance-platform/internal/pipeline1"
)

// =============================================================================
// PIPELINE 1 SERVER
// =============================================================================
// Handles L2 extraction review endpoints for the Pipeline 1 reviewer UI.
// Mirrors FactServer pattern: own ServeMux, CORS, respondJSON/respondError.
// =============================================================================

// Pipeline1Server handles HTTP requests for Pipeline 1 span review.
type Pipeline1Server struct {
	store  *database.Pipeline1Store
	router *http.ServeMux
}

// NewPipeline1Server creates a new Pipeline 1 API server.
func NewPipeline1Server(store *database.Pipeline1Store) *Pipeline1Server {
	s := &Pipeline1Server{
		store:  store,
		router: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Pipeline1Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.router.ServeHTTP(w, r)
}

func (s *Pipeline1Server) registerRoutes() {
	// Health
	s.router.HandleFunc("GET /api/v2/pipeline1/health", s.handleHealth)

	// Jobs
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs", s.handleListJobs)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}", s.handleGetJob)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/metrics", s.handleGetMetrics)

	// Spans
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/spans", s.handleGetSpans)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/spans/{span_id}", s.handleGetSpan)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/spans/{span_id}/confirm", s.handleConfirmSpan)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/spans/{span_id}/reject", s.handleRejectSpan)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/spans/{span_id}/edit", s.handleEditSpan)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/spans/add", s.handleAddSpan)

	// Context
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/passages", s.handleGetPassages)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/tree", s.handleGetTree)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/text", s.handleGetText)

	// Page Decisions
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/pages", s.handleGetPages)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/pages/{page_number}/decide", s.handleDecidePage)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/pages/stats", s.handleGetPageStats)

	// Job Completion (Sprint 2 — Phase 5 Sign-Off)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/complete", s.handleCompleteJob)

	// Review Task Queue + Patched Passages
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/review-tasks", s.handleGetReviewTasks)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/passages/patched", s.handleGetPatchedPassages)

	// Reference Views (Pipeline HTML + Source PDF)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/highlight-html", s.handleGetHighlightHTML)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/source-pdf", s.handleGetSourcePDF)

	// Revalidation (Phase 4 — CoverageGuard delta check)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/revalidate", s.handleRunRevalidation)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/revalidation-history", s.handleGetRevalidationHistory)

	// Output Contract (Phase 5 — Pipeline 2 handoff)
	s.router.HandleFunc("GET /api/v2/pipeline1/jobs/{job_id}/output-contract/preview", s.handlePreviewOutputContract)
	s.router.HandleFunc("POST /api/v2/pipeline1/jobs/{job_id}/output-contract", s.handleAssembleOutputContract)
}

// =============================================================================
// HEALTH
// =============================================================================

func (s *Pipeline1Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "kb-0-pipeline1-review",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// =============================================================================
// JOB HANDLERS
// =============================================================================

func (s *Pipeline1Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r, 1, 20)

	jobs, total, err := s.store.GetJobs(r.Context(), page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":    jobs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"hasMore":  page*pageSize < total,
	})
}

func (s *Pipeline1Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	job, err := s.store.GetJob(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, job)
}

func (s *Pipeline1Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	metrics, err := s.store.GetJobMetrics(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

// =============================================================================
// JOB COMPLETION (Sprint 2 — Phase 5 Sign-Off)
// =============================================================================

func (s *Pipeline1Server) handleCompleteJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.CompleteJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}

	if err := s.store.CompleteJob(r.Context(), jobID, req.ReviewerID, req.Note); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"jobId":   jobID,
		"status":  "COMPLETED",
	})
}

// =============================================================================
// SPAN HANDLERS
// =============================================================================

func (s *Pipeline1Server) handleGetSpans(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	filters := parseSpanFilters(r)
	page, pageSize := parsePagination(r, 1, 50)

	spans, total, err := s.store.GetSpans(r.Context(), jobID, filters, page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items":    spans,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"hasMore":  page*pageSize < total,
	})
}

func (s *Pipeline1Server) handleGetSpan(w http.ResponseWriter, r *http.Request) {
	spanID, err := parseSpanID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	span, err := s.store.GetSpan(r.Context(), spanID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, span)
}

func (s *Pipeline1Server) handleConfirmSpan(w http.ResponseWriter, r *http.Request) {
	spanID, err := parseSpanID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.SpanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}

	if err := s.store.UpdateSpanStatus(r.Context(), spanID, pipeline1.ActionConfirm, nil, req.ReviewerID, req.Note, nil); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "spanId": spanID, "action": "CONFIRM"})
}

func (s *Pipeline1Server) handleRejectSpan(w http.ResponseWriter, r *http.Request) {
	spanID, err := parseSpanID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.SpanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}

	if err := s.store.UpdateSpanStatus(r.Context(), spanID, pipeline1.ActionReject, nil, req.ReviewerID, req.Note, req.RejectReason); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "spanId": spanID, "action": "REJECT"})
}

func (s *Pipeline1Server) handleEditSpan(w http.ResponseWriter, r *http.Request) {
	spanID, err := parseSpanID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.SpanReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}
	if req.EditedText == nil || *req.EditedText == "" {
		respondError(w, http.StatusBadRequest, "editedText required for edit action")
		return
	}

	if err := s.store.UpdateSpanStatus(r.Context(), spanID, pipeline1.ActionEdit, req.EditedText, req.ReviewerID, req.Note, nil); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "spanId": spanID, "action": "EDIT"})
}

func (s *Pipeline1Server) handleAddSpan(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.AddSpanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}
	if req.Text == "" {
		respondError(w, http.StatusBadRequest, "text required")
		return
	}

	newID := uuid.New()
	span := &pipeline1.MergedSpan{
		ID:          newID,
		JobID:       jobID,
		Text:        req.Text,
		StartOffset: req.Start,
		EndOffset:   req.End,
		PageNumber:  req.PageNumber,
		SectionID:   req.SectionID,
	}

	if err := s.store.InsertSpan(r.Context(), span, req.ReviewerID, req.Note); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "spanId": newID, "action": "ADD"})
}

// =============================================================================
// CONTEXT HANDLERS
// =============================================================================

func (s *Pipeline1Server) handleGetPassages(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	passages, err := s.store.GetPassages(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": passages})
}

func (s *Pipeline1Server) handleGetTree(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	tree, err := s.store.GetTree(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, tree)
}

func (s *Pipeline1Server) handleGetText(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	text, err := s.store.GetNormalizedText(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"text": text})
}

// =============================================================================
// HELPERS
// =============================================================================

func parseJobID(r *http.Request) (uuid.UUID, error) {
	s := r.PathValue("job_id")
	if s == "" {
		return uuid.UUID{}, fmt.Errorf("job_id required")
	}
	return uuid.Parse(s)
}

func parseSpanID(r *http.Request) (uuid.UUID, error) {
	s := r.PathValue("span_id")
	if s == "" {
		return uuid.UUID{}, fmt.Errorf("span_id required")
	}
	return uuid.Parse(s)
}

func parseSpanFilters(r *http.Request) pipeline1.SpanFilters {
	f := pipeline1.SpanFilters{}
	q := r.URL.Query()

	if v := q.Get("status"); v != "" {
		s := pipeline1.SpanReviewStatus(v)
		f.Status = &s
	}
	if v := q.Get("sectionId"); v != "" {
		f.SectionID = &v
	}
	if v := q.Get("pageNumber"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.PageNumber = &n
		}
	}
	if v := q.Get("minConfidence"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinConfidence = &n
		}
	}
	if v := q.Get("maxConfidence"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.MaxConfidence = &n
		}
	}
	if v := q.Get("hasDisagreement"); v != "" {
		b := v == "true"
		f.HasDisagreement = &b
	}
	if v := q.Get("search"); v != "" {
		f.Search = &v
	}
	if v := q.Get("tier"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 3 {
			f.Tier = &n
		}
	}

	return f
}

func parsePagination(r *http.Request, defaultPage, defaultPageSize int) (int, int) {
	page := defaultPage
	pageSize := defaultPageSize

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			pageSize = n
		}
	}

	return page, pageSize
}

// =============================================================================
// PAGE DECISION HANDLERS
// =============================================================================

func (s *Pipeline1Server) handleGetPages(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	pages, err := s.store.GetPages(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": pages})
}

func (s *Pipeline1Server) handleDecidePage(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	pageNumStr := r.PathValue("page_number")
	if pageNumStr == "" {
		respondError(w, http.StatusBadRequest, "page_number required")
		return
	}
	pageNumber, err := strconv.Atoi(pageNumStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "page_number must be an integer")
		return
	}

	var req pipeline1.PageDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}
	if req.Action != pipeline1.PageActionAccept && req.Action != pipeline1.PageActionFlag && req.Action != pipeline1.PageActionEscalate {
		respondError(w, http.StatusBadRequest, "action must be ACCEPT, FLAG, or ESCALATE")
		return
	}

	// ACCEPT guard: all Tier 1 (patient safety) spans on this page must be reviewed
	if req.Action == pipeline1.PageActionAccept {
		t1Pending, err := s.store.CountTier1PendingOnPage(r.Context(), jobID, pageNumber)
		if err == nil && t1Pending > 0 {
			respondError(w, http.StatusConflict, fmt.Sprintf(
				"Cannot ACCEPT page %d: %d Tier 1 (patient safety) spans still pending review",
				pageNumber, t1Pending,
			))
			return
		}
	}

	if err := s.store.DecidePage(r.Context(), jobID, pageNumber, req.Action, req.ReviewerID, req.Note); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"jobId":      jobID,
		"pageNumber": pageNumber,
		"action":     req.Action,
	})
}

func (s *Pipeline1Server) handleGetPageStats(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stats, err := s.store.GetPageStats(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// =============================================================================
// REVIEW TASK QUEUE + PATCHED PASSAGES
// =============================================================================

func (s *Pipeline1Server) handleGetReviewTasks(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	tasks, err := s.store.GetReviewTasks(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": tasks})
}

func (s *Pipeline1Server) handleGetPatchedPassages(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	passages, err := s.store.GetPatchedPassages(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": passages})
}

// =============================================================================
// REFERENCE VIEW HANDLERS (Pipeline HTML + Source PDF)
// =============================================================================

func (s *Pipeline1Server) handleGetHighlightHTML(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	html, err := s.store.GetHighlightHTML(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if html == "" {
		respondError(w, http.StatusNotFound, "no highlight HTML available for this job")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// =============================================================================
// REVALIDATION HANDLERS (Phase 4)
// =============================================================================

func (s *Pipeline1Server) handleRunRevalidation(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.RevalidateRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	run, err := s.store.RunRevalidation(r.Context(), jobID, req.ReviewerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, run)
}

func (s *Pipeline1Server) handleGetRevalidationHistory(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	runs, err := s.store.GetRevalidationHistory(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": runs})
}

// =============================================================================
// OUTPUT CONTRACT HANDLERS (Phase 5)
// =============================================================================

func (s *Pipeline1Server) handlePreviewOutputContract(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	contract, err := s.store.PreviewOutputContract(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, contract)
}

func (s *Pipeline1Server) handleAssembleOutputContract(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req pipeline1.AssembleContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "reviewerId required")
		return
	}

	contract, err := s.store.AssembleOutputContract(r.Context(), jobID, req.ReviewerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, contract)
}

// =============================================================================
// REFERENCE VIEW HANDLERS (Pipeline HTML + Source PDF) — continued
// =============================================================================

func (s *Pipeline1Server) handleGetSourcePDF(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	pdfPath, err := s.store.GetSourcePDFPath(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if pdfPath == "" {
		respondError(w, http.StatusNotFound, "no source PDF path configured for this job")
		return
	}

	// Verify file exists before serving
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		respondError(w, http.StatusNotFound, "source PDF file not found on disk")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, pdfPath)
}
