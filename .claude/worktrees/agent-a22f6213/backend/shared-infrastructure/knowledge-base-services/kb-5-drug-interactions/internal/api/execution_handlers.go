package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-drug-interactions/internal/services"
)

// ExecutionHandlers handles the ONC → MED-RT → OHDSI → LOINC execution contract
type ExecutionHandlers struct {
	executionService *services.ExecutionContractService
}

// NewExecutionHandlers creates execution contract handlers
func NewExecutionHandlers(executionService *services.ExecutionContractService) *ExecutionHandlers {
	return &ExecutionHandlers{
		executionService: executionService,
	}
}

// RegisterRoutes registers execution contract routes
func (h *ExecutionHandlers) RegisterRoutes(r *gin.RouterGroup) {
	execution := r.Group("/execution")
	{
		// Primary DDI evaluation endpoint
		execution.POST("/evaluate", h.EvaluateDDI)

		// Contract documentation
		execution.GET("/contract", h.GetContractSpec)
	}
}

// EvaluateDDIRequest is the request for DDI evaluation
type EvaluateDDIRequest struct {
	PatientID      string             `json:"patient_id" binding:"required"`
	DrugConceptIDs []int64            `json:"drug_concept_ids" binding:"required,min=2"`
	PatientLabs    map[string]float64 `json:"patient_labs"`
}

// EvaluateDDI evaluates drug-drug interactions using the full execution contract
// @Summary Evaluate DDI with full ONC → OHDSI → LOINC pipeline
// @Description Runs complete DDI evaluation with context-aware alerting
// @Tags Execution Contract
// @Accept json
// @Produce json
// @Param request body EvaluateDDIRequest true "Evaluation request"
// @Success 200 {object} services.DDIEvaluationResponse
// @Router /execution/evaluate [post]
func (h *ExecutionHandlers) EvaluateDDI(c *gin.Context) {
	ctx := c.Request.Context()

	var req EvaluateDDIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": "Provide patient_id and at least 2 drug_concept_ids",
		})
		return
	}

	// Initialize labs map if not provided
	if req.PatientLabs == nil {
		req.PatientLabs = make(map[string]float64)
	}

	// Create evaluation request
	evalReq := services.DDIEvaluationRequest{
		PatientID:      req.PatientID,
		DrugConceptIDs: req.DrugConceptIDs,
		PatientLabs:    req.PatientLabs,
	}

	// Execute the contract
	response, err := h.executionService.EvaluateDDI(ctx, evalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "DDI evaluation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetContractSpec returns the execution contract specification
// @Summary Get execution contract specification
// @Description Returns documentation of the ONC → MED-RT → OHDSI → LOINC pipeline
// @Tags Execution Contract
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /execution/contract [get]
func (h *ExecutionHandlers) GetContractSpec(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "ONC → MED-RT → OHDSI → LOINC Execution Contract",
		"version": "v1.0",
		"layers": []map[string]interface{}{
			{
				"layer":       1,
				"name":        "PROJECTION",
				"source":      "ONC Constitutional Rules",
				"purpose":     "Identify which class-based rules COULD apply",
				"output":      "List of potentially applicable rules",
				"constraints": "MUST NOT filter based on labs",
			},
			{
				"layer":       2,
				"name":        "EXPANSION",
				"source":      "OHDSI Vocabulary (73,842 drug→class mappings)",
				"purpose":     "Resolve class rules to concrete drug pairs",
				"output":      "DDI projections (intentional over-generation)",
				"constraints": "Cartesian expansion, canonical ordering",
			},
			{
				"layer":       3,
				"name":        "CONTEXT",
				"source":      "LOINC Labs (KB-16)",
				"purpose":     "Apply clinical context to modify/suppress alerts",
				"output":      "Context-evaluated interactions",
				"constraints": "Fail-safe behavior when labs missing",
			},
			{
				"layer":       4,
				"name":        "OUTPUT",
				"source":      "Execution Contract",
				"purpose":     "Generate final tiered alerts with audit trail",
				"output":      "FinalAlert objects with governance metadata",
				"constraints": "CMS-ready audit trail required",
			},
		},
		"context_logic": map[string]interface{}{
			"no_context_defined":             "Always alert at base severity",
			"context_required_true_lab_missing":  "FAIL-SAFE: Alert anyway",
			"context_required_false_lab_missing": "Alert at base severity",
			"threshold_exceeded":             "Escalate severity",
			"threshold_not_exceeded_required": "Alert at base severity",
			"threshold_not_exceeded_optional": "SUPPRESS (can be filtered)",
		},
		"severity_escalation": map[string]string{
			"WARNING":  "→ HIGH (when threshold exceeded)",
			"HIGH":     "→ CRITICAL (when threshold exceeded)",
			"CRITICAL": "→ CRITICAL (already maximum)",
		},
		"governance": map[string]interface{}{
			"rule_authority_preserved": true,
			"rule_version_preserved":   true,
			"audit_trail_generated":    true,
			"cms_reporting_ready":      true,
		},
		"example_request": map[string]interface{}{
			"patient_id":       "P12345",
			"drug_concept_ids": []int64{1310149, 1177480},
			"patient_labs": map[string]float64{
				"5902-2": 4.2,
			},
		},
	})
}
