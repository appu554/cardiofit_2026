package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kb-26-metabolic-digital-twin/internal/services"
)

// TargetStatusRequest is the POST body for /api/v1/patient/:id/target-status.
// KB-23's InertiaInputAssembler populates this from the raw patient
// measurements already fetched from KB-20 (HbA1c, SBP, eGFR) and KB-26
// converts them into per-domain DomainTargetStatusResult verdicts via
// the existing ComputeGlycaemicTargetStatus / ComputeHemodynamicTargetStatus
// functions. Phase 7 P7-D.
type TargetStatusRequest struct {
	HbA1c        *float64 `json:"hba1c,omitempty"`
	HbA1cDate    *string  `json:"hba1c_date,omitempty"`
	HbA1cTarget  float64  `json:"hba1c_target,omitempty"`
	CGMTIR       *float64 `json:"cgm_tir,omitempty"`
	CGMReportDate *string `json:"cgm_report_date,omitempty"`
	TIRTarget    float64  `json:"tir_target,omitempty"`

	MeanSBP7d *float64 `json:"mean_sbp_7d,omitempty"`
	SBPTarget float64  `json:"sbp_target,omitempty"`

	// Renal inputs carried through so KB-23's inertia assembler can
	// populate the Renal DomainInertiaInput. KB-26 does not run a
	// formal renal-target compute function yet (Phase 8); the handler
	// currently returns the inputs echoed through a simple threshold
	// check (eGFR ≥ 45 → at target).
	EGFR         *float64 `json:"egfr,omitempty"`
	EGFRTarget   float64  `json:"egfr_target,omitempty"`
}

// TargetStatusResponse is the handler's envelope body.
type TargetStatusResponse struct {
	Glycaemic   services.DomainTargetStatusResult `json:"glycaemic"`
	Hemodynamic services.DomainTargetStatusResult `json:"hemodynamic"`
	Renal       services.DomainTargetStatusResult `json:"renal"`
}

// getTargetStatus returns per-domain target status verdicts for the
// given patient using the raw measurements supplied in the request body.
// Stateless compute — KB-26 does not fetch anything from its database
// in this handler. Phase 7 P7-D.
func (s *Server) getTargetStatus(c *gin.Context) {
	var req TargetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid target-status request: "+err.Error(), "BAD_REQUEST", nil)
		return
	}

	// Default targets if the caller omits them. These mirror the
	// Vaidshala runtime defaults — callers may override per-patient.
	if req.HbA1cTarget == 0 {
		req.HbA1cTarget = 7.0
	}
	if req.TIRTarget == 0 {
		req.TIRTarget = 70.0
	}
	if req.SBPTarget == 0 {
		req.SBPTarget = 130.0
	}
	if req.EGFRTarget == 0 {
		req.EGFRTarget = 45.0
	}

	glycInput := services.TargetStatusInput{
		HbA1c:       req.HbA1c,
		HbA1cTarget: req.HbA1cTarget,
		CGMTIR:      req.CGMTIR,
		TIRTarget:   req.TIRTarget,
	}
	if req.HbA1cDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.HbA1cDate); err == nil {
			glycInput.HbA1cDate = &t
		}
	}
	if req.CGMTIR != nil && req.CGMReportDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.CGMReportDate); err == nil {
			glycInput.CGMReportDate = &t
			glycInput.CGMAvailable = true
			glycInput.CGMSufficientData = true
		}
	}

	bpInput := services.BPTargetStatusInput{
		MeanSBP7d: req.MeanSBP7d,
		SBPTarget: req.SBPTarget,
	}

	resp := TargetStatusResponse{
		Glycaemic:   services.ComputeGlycaemicTargetStatus(glycInput),
		Hemodynamic: services.ComputeHemodynamicTargetStatus(bpInput),
		Renal:       computeRenalTargetStatus(req.EGFR, req.EGFRTarget),
	}

	sendSuccess(c, resp, nil)
}

// computeRenalTargetStatus is a lightweight inline check: a patient is
// "at renal target" when their eGFR is at or above the threshold
// (default 45 mL/min/1.73m²). KB-26 does not yet ship a full renal
// compute function in target_status.go — this keeps the handler
// self-contained until the Phase 8 addition lands.
func computeRenalTargetStatus(egfr *float64, target float64) services.DomainTargetStatusResult {
	result := services.DomainTargetStatusResult{
		Domain:      "RENAL",
		TargetValue: target,
		DataSource:  "EGFR",
		Confidence:  "MODERATE",
	}
	if egfr != nil {
		result.CurrentValue = *egfr
		result.AtTarget = *egfr >= target
		result.ConsecutiveReadings = 1
	}
	return result
}
