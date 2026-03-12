package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	contextrouter "context-router"
	"kb-drug-interactions/internal/services"
)

// ExecutionHandlers handles the ONC → OHDSI → LOINC execution contract
// This handler orchestrates:
//   1. OHDSI Expansion Service (produces DDI projections from constitutional rules)
//   2. Shared Context Router (evaluates LOINC context to produce decisions)
//
// Golden Rule: "Class Expansion NEVER checks LOINC. Context Router ALWAYS does."
type ExecutionHandlers struct {
	expansionService *services.OHDSIExpansionService
	contextRouter    *contextrouter.ContextRouter
}

// NewExecutionHandlers creates execution contract handlers
func NewExecutionHandlers(
	expansionService *services.OHDSIExpansionService,
	contextRouter *contextrouter.ContextRouter,
) *ExecutionHandlers {
	return &ExecutionHandlers{
		expansionService: expansionService,
		contextRouter:    contextRouter,
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
	PatientID      string                 `json:"patient_id" binding:"required"`
	DrugConceptIDs []int64                `json:"drug_concept_ids" binding:"required,min=2"`
	PatientLabs    map[string]LabValueDTO `json:"patient_labs"`
}

// LabValueDTO represents a lab value in the API request
type LabValueDTO struct {
	Value     float64    `json:"value"`
	Unit      string     `json:"unit,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

// EvaluateDDI evaluates drug-drug interactions using the full execution contract
// @Summary Evaluate DDI with full ONC → OHDSI → LOINC pipeline
// @Description Runs complete DDI evaluation with context-aware alerting
// @Tags Execution Contract
// @Accept json
// @Produce json
// @Param request body EvaluateDDIRequest true "Evaluation request"
// @Success 200 {object} contextrouter.ContextRouterResponse
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

	// ═══════════════════════════════════════════════════════════════════
	// LAYER 1-2: PROJECTION + EXPANSION (ONC Rules → OHDSI Vocabulary)
	// ═══════════════════════════════════════════════════════════════════
	// The expansion service checks constitutional rules and expands
	// class-based rules to concrete drug pairs using OHDSI vocabulary.
	// ✅ Expansion answers: "CAN this interaction exist?"
	// ❌ Expansion NEVER checks LOINC values
	checkResult, err := h.expansionService.CheckDDI(ctx, req.DrugConceptIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "DDI expansion failed",
			"details": err.Error(),
		})
		return
	}

	// ═══════════════════════════════════════════════════════════════════
	// LAYER 3-4: CONTEXT + OUTPUT (LOINC Evaluation → Decisions)
	// ═══════════════════════════════════════════════════════════════════
	// Convert OHDSI projections to Context Router format
	projections := convertToContextRouterProjections(checkResult.Interactions)

	// Build patient context from request
	patientContext := buildPatientContext(req.PatientID, req.PatientLabs)

	// ✅ Context Router ALWAYS evaluates LOINC when context_required=true
	// ✅ Context Router answers: "DOES it matter for THIS patient NOW?"
	response := h.contextRouter.Evaluate(projections, patientContext)

	c.JSON(http.StatusOK, response)
}

// convertToContextRouterProjections converts OHDSI projections to Context Router format
func convertToContextRouterProjections(ohdsiProjections []services.DDIProjection) []contextrouter.DDIProjection {
	projections := make([]contextrouter.DDIProjection, len(ohdsiProjections))

	for i, p := range ohdsiProjections {
		projections[i] = contextrouter.DDIProjection{
			RuleID:               p.RuleID,
			DrugAConceptID:       p.DrugAConceptID,
			DrugAName:            p.DrugAName,
			DrugAClassName:       p.DrugAClassName,
			DrugBConceptID:       p.DrugBConceptID,
			DrugBName:            p.DrugBName,
			DrugBClassName:       p.DrugBClassName,
			RiskLevel:            p.RiskLevel,
			AlertMessage:         p.AlertMessage,
			RuleAuthority:        p.RuleAuthority,
			RuleVersion:          p.RuleVersion,
			ContextRequired:      p.ContextRequired,
			ContextLOINCID:       p.ContextLOINCID,
			ContextThreshold:     p.ContextThreshold,
			ContextOperator:      p.ContextOperator,
			EvaluationTier:       convertEvaluationTier(p.EvaluationTier),
			InteractionDirection: convertInteractionDirection(p.InteractionDirection),
			LazyEvaluate:         p.LazyEvaluate,
			AffectedDrugRole:     p.AffectedDrugRole,
		}
	}

	return projections
}

// convertEvaluationTier converts OHDSI tier to Context Router tier
func convertEvaluationTier(tier services.EvaluationTier) contextrouter.EvaluationTier {
	switch tier {
	case services.TierONCHigh:
		return contextrouter.TierONCHigh
	case services.TierSevere:
		return contextrouter.TierSevere
	case services.TierModerate:
		return contextrouter.TierModerate
	case services.TierMechanism:
		return contextrouter.TierMechanism
	default:
		return contextrouter.TierModerate
	}
}

// convertInteractionDirection converts OHDSI direction to Context Router direction
func convertInteractionDirection(dir services.InteractionDirection) contextrouter.InteractionDirection {
	switch dir {
	case services.DirectionBidirectional:
		return contextrouter.DirectionBidirectional
	case services.DirectionAffectsTrigger:
		return contextrouter.DirectionAffectsTrigger
	case services.DirectionAffectsTarget:
		return contextrouter.DirectionAffectsTarget
	default:
		return contextrouter.DirectionBidirectional
	}
}

// buildPatientContext converts request labs to PatientContext
func buildPatientContext(patientID string, labs map[string]LabValueDTO) *contextrouter.PatientContext {
	context := &contextrouter.PatientContext{
		PatientID: patientID,
		Labs:      make(map[string]contextrouter.LabValue),
	}

	for loincCode, lab := range labs {
		labValue := contextrouter.LabValue{
			Value: lab.Value,
			Unit:  lab.Unit,
		}
		if lab.Timestamp != nil {
			labValue.Timestamp = *lab.Timestamp
		}
		context.Labs[loincCode] = labValue
	}

	return context
}

// GetContractSpec returns the execution contract specification
// @Summary Get execution contract specification
// @Description Returns documentation of the ONC → OHDSI → LOINC pipeline
// @Tags Execution Contract
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /execution/contract [get]
func (h *ExecutionHandlers) GetContractSpec(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "ONC → OHDSI → LOINC Execution Contract",
		"version": "v2.0",
		"architecture": gin.H{
			"expansion_layer": "kb-5-drug-interactions/services/ohdsi_expansion_service.go",
			"context_layer":   "shared-infrastructure/orchestration/context_router/",
			"separation":      "Strict downstream relationship - expansion produces, context evaluates",
		},
		"layers": []map[string]interface{}{
			{
				"layer":       1,
				"name":        "PROJECTION",
				"source":      "ONC Constitutional Rules (canonical_facts.ddi_constitutional_rules)",
				"purpose":     "Identify which class-based rules COULD apply",
				"output":      "List of potentially applicable rules",
				"constraints": "MUST NOT filter based on labs",
			},
			{
				"layer":       2,
				"name":        "EXPANSION",
				"source":      "OHDSI Vocabulary (73,842 drug→class mappings)",
				"purpose":     "Resolve class rules to concrete drug pairs",
				"output":      "DDIProjection (intentional over-generation)",
				"constraints": "Cartesian expansion, canonical ordering, NEVER checks LOINC",
			},
			{
				"layer":       3,
				"name":        "CONTEXT",
				"source":      "Shared Context Router (shared-infrastructure/orchestration)",
				"purpose":     "Apply clinical context to produce decisions",
				"output":      "DDIDecision (BLOCK, INTERRUPT, INFORMATIONAL, SUPPRESSED, NEEDS_CONTEXT)",
				"constraints": "ALWAYS evaluates LOINC when context_required=true, fail-safe behavior",
			},
			{
				"layer":       4,
				"name":        "OUTPUT",
				"source":      "Context Router Response",
				"purpose":     "Generate final tiered decisions with audit trail",
				"output":      "ContextRouterResponse with full governance metadata",
				"constraints": "CMS-ready audit trail, tier-based lazy evaluation",
			},
		},
		"golden_rules": []string{
			"Class Expansion NEVER checks LOINC",
			"Context Router ALWAYS checks LOINC (when required)",
			"TIER_0 (ONC Constitutional) rules cannot be suppressed in strict mode",
			"Expansion answers: CAN this interaction exist?",
			"Context answers: DOES it matter for THIS patient NOW?",
			"All decisions have audit trail",
			"Projections are immutable after expansion",
		},
		"decision_types": map[string]string{
			"BLOCK":         "Absolute contraindication, cannot proceed",
			"INTERRUPT":     "Requires clinician acknowledgment before proceeding",
			"INFORMATIONAL": "Display information but allow to proceed",
			"SUPPRESSED":    "Do not display, context indicates low risk",
			"NEEDS_CONTEXT": "Cannot evaluate, required context is missing",
		},
		"context_logic": map[string]interface{}{
			"semantic_contract_version": "2.0",
			"context_required_true": map[string]interface{}{
				"semantics":          "HARD GATE - interaction only meaningful when context abnormal",
				"threshold_exceeded": "Alert fires: WARNING→INTERRUPT, HIGH→INTERRUPT, MODERATE→INFORMATIONAL",
				"threshold_not_met":  "SUPPRESS (default) - context indicates safe, no alert",
				"context_missing":    "FAIL-SAFE: INTERRUPT (ONC) or NEEDS_CONTEXT (others)",
			},
			"context_required_false": map[string]interface{}{
				"semantics": "FAIL OPEN - alert fires based on risk level, context not evaluated",
				"behavior":  "CRITICAL→BLOCK, HIGH→INTERRUPT, WARNING/MODERATE→INFORMATIONAL",
			},
			"policy_modes": map[string]interface{}{
				"StrictONCMode":            "ONC Constitutional rules (TIER_0) cannot be suppressed",
				"ConservativeHighRiskMode": "Opt-in: HIGH-risk stays INFORMATIONAL even with safe context",
			},
		},
		"example_request": map[string]interface{}{
			"patient_id":       "P12345",
			"drug_concept_ids": []int64{1310149, 1177480},
			"patient_labs": map[string]interface{}{
				"6301-6": map[string]interface{}{
					"value": 4.2,
					"unit":  "INR",
				},
			},
		},
	})
}
