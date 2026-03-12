package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"kb-drug-interactions/internal/models"
	"kb-drug-interactions/internal/services"
)

// GovernanceHandlers handles governance and attribution endpoints
type GovernanceHandlers struct {
	governanceEngine *services.GovernancePolicyEngine
	interactionService *services.InteractionService
	integrationService *services.EnhancedIntegrationService
}

// NewGovernanceHandlers creates new governance handlers
func NewGovernanceHandlers(
	governanceEngine *services.GovernancePolicyEngine,
	interactionService *services.InteractionService,
	integrationService *services.EnhancedIntegrationService,
) *GovernanceHandlers {
	return &GovernanceHandlers{
		governanceEngine:   governanceEngine,
		interactionService: interactionService,
		integrationService: integrationService,
	}
}

// ============================================================================
// GOVERNED INTERACTION CHECK
// POST /api/v1/interactions/governed-check
// Returns interactions with governance actions and attribution
// ============================================================================

// GovernedInteractionCheckRequest extends request with governance parameters
type GovernedInteractionCheckRequest struct {
	DrugCodes       []string                     `json:"drug_codes" binding:"required,min=2"`
	PatientContext  *models.PatientContextData   `json:"patient_context,omitempty"`
	InstitutionID   string                       `json:"institution_id,omitempty"`
	DatasetVersion  string                       `json:"dataset_version,omitempty"`
	IncludeGuidance bool                         `json:"include_guidance"`
}

func (h *GovernanceHandlers) governedInteractionCheck(c *gin.Context) {
	startTime := time.Now()

	if h.governanceEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Governance engine not available", "ENGINE_UNAVAILABLE", nil)
		return
	}

	var request GovernedInteractionCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// First, get base interactions from interaction service
	baseRequest := models.InteractionCheckRequest{
		DrugCodes:           request.DrugCodes,
		IncludeAlternatives: true,
		IncludeMonitoring:   true,
	}

	baseResponse, err := h.interactionService.CheckInteractions(baseRequest)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to check interactions", "CHECK_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Convert to enhanced results and apply governance
	governedInteractions := make([]models.GovernedInteractionResult, 0, len(baseResponse.InteractionsFound))
	enginesUsed := []string{"core_ddi"}

	for _, interaction := range baseResponse.InteractionsFound {
		// Convert to enhanced result format
		enhanced := convertToEnhancedResult(interaction)

		// Apply governance layer
		governed := h.governanceEngine.EnhanceWithGovernance(
			enhanced,
			request.InstitutionID,
			request.PatientContext,
		)
		governedInteractions = append(governedInteractions, governed)
	}

	// Build governed summary
	summary := h.governanceEngine.BuildGovernedSummary(governedInteractions)

	// Build response attribution
	processingTime := float64(time.Since(startTime).Milliseconds())
	attribution := h.governanceEngine.BuildResponseAttribution(
		governedInteractions,
		enginesUsed,
		processingTime,
		request.InstitutionID,
	)
	attribution.RequestHash = h.governanceEngine.HashRequest(request)

	// Build full response
	response := models.GovernedInteractionCheckResponse{
		TransactionID:    uuid.New().String(),
		DatasetVersion:   h.governanceEngine.GetPolicy(request.InstitutionID).PolicyName,
		Interactions:     governedInteractions,
		Summary:          summary,
		PolicyApplied:    h.governanceEngine.GetPolicy(request.InstitutionID).PolicyName,
		PolicyVersion:    "1.0",
		InstitutionID:    request.InstitutionID,
		Attribution:      attribution,
		Recommendations:  baseResponse.Recommendations,
		CheckTimestamp:   time.Now(),
		CacheHit:         baseResponse.CacheHit,
	}

	// Return response with governance metadata
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"meta": map[string]interface{}{
			"total_interactions":        len(governedInteractions),
			"highest_severity":          summary.HighestSeverity,
			"highest_governance_action": summary.HighestGovernanceAction,
			"is_order_blocked":          summary.IsOrderBlocked,
			"documentation_required":    summary.DocumentationRequired,
			"processing_time_ms":        processingTime,
		},
	})
}

// ============================================================================
// GOVERNANCE POLICY ENDPOINTS
// ============================================================================

// getGovernancePolicy handles GET /api/v1/governance/policy
func (h *GovernanceHandlers) getGovernancePolicy(c *gin.Context) {
	institutionID := c.Query("institution_id")

	policy := h.governanceEngine.GetPolicy(institutionID)

	sendSuccess(c, map[string]interface{}{
		"policy_name":             policy.PolicyName,
		"institution_id":          policy.InstitutionID,
		"effective_date":          policy.EffectiveDate,
		"severity_mappings": map[string]string{
			"contraindicated": string(policy.ContraindicatedAction),
			"major":           string(policy.MajorAction),
			"moderate":        string(policy.ModerateAction),
			"minor":           string(policy.MinorAction),
			"unknown":         string(policy.UnknownAction),
		},
		"context_escalations": map[string]bool{
			"pediatric_escalation":       policy.PediatricEscalation,
			"geriatric_escalation":       policy.GeriatricEscalation,
			"renal_impairment_upgrade":   policy.RenalImpairmentUpgrade,
			"hepatic_impairment_upgrade": policy.HepaticImpairmentUpgrade,
		},
		"governance_action_descriptions": map[string]string{
			"ignore":                     models.GetGovernanceActionDescription(models.GovernanceIgnore),
			"notify":                     models.GetGovernanceActionDescription(models.GovernanceNotify),
			"warn_acknowledge":           models.GetGovernanceActionDescription(models.GovernanceWarnAcknowledge),
			"hard_block":                 models.GetGovernanceActionDescription(models.GovernanceHardBlock),
			"hard_block_governance_override": models.GetGovernanceActionDescription(models.GovernanceHardBlockOverride),
			"mandatory_escalation":       models.GetGovernanceActionDescription(models.GovernanceMandatoryEscalation),
		},
	}, nil)
}

// translateSeverity handles POST /api/v1/governance/translate
// Translates clinical severity to governance action with context
func (h *GovernanceHandlers) translateSeverity(c *gin.Context) {
	var request struct {
		Severity       string                      `json:"severity" binding:"required"`
		InstitutionID  string                      `json:"institution_id,omitempty"`
		PatientContext *models.PatientContextData  `json:"patient_context,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	// Convert string to DDISeverity
	severity := models.DDISeverity(request.Severity)

	// Translate to governance action
	action := h.governanceEngine.TranslateSeverity(
		severity,
		request.InstitutionID,
		request.PatientContext,
	)

	sendSuccess(c, map[string]interface{}{
		"input_severity":      request.Severity,
		"governance_action":   action,
		"action_description":  models.GetGovernanceActionDescription(action),
		"action_priority":     action.Priority(),
		"is_blocking":         action.IsBlocking(),
		"requires_acknowledgment": action.RequiresAcknowledgment(),
		"allows_clinical_override": action.AllowsClinicalOverride(),
		"override_level":      models.GetOverrideLevel(action),
		"policy_applied":      h.governanceEngine.GetPolicy(request.InstitutionID).PolicyName,
	}, map[string]interface{}{
		"context_applied": request.PatientContext != nil,
	})
}

// ============================================================================
// ATTRIBUTION LOOKUP ENDPOINTS
// ============================================================================

// getAttributionTemplate handles GET /api/v1/governance/attribution/template
// Returns the attribution envelope template for integration
func (h *GovernanceHandlers) getAttributionTemplate(c *gin.Context) {
	sendSuccess(c, map[string]interface{}{
		"attribution_envelope_schema": map[string]string{
			"dataset_version":      "string - Version of the clinical knowledge dataset",
			"dataset_date":         "datetime - When the dataset was published",
			"harmonization_id":     "string - Unique ID for data harmonization run",
			"clinical_sources":     "array - List of clinical knowledge sources used",
			"evidence_level":       "string - Evidence grading (A, B, C, D, ExpertOpinion, Unknown)",
			"evidence_strength":    "string - Evidence quality (HIGH, MODERATE, LOW, VERY_LOW, UNGRADED)",
			"confidence_score":     "decimal - Model confidence 0.0-1.0",
			"governance_relevance": "array - Governance categories (REGULATORY, LEGAL, ACCREDITATION, etc.)",
			"governance_action":    "string - Workflow action (ignore, notify, warn_acknowledge, etc.)",
			"program_flags":        "array - Special program applicability (ONCOLOGY, PEDIATRIC, etc.)",
		},
		"clinical_sources": []string{
			"FDA_LABEL", "DRUGBANK", "RXNORM", "CPIC", "PHARMGKB",
			"MICROMEDEX", "LEXICOMP", "UPTODATE", "CLINICAL_TRIALS",
			"PEER_REVIEWED", "EXPERT_CONSENSUS", "INSTITUTIONAL", "UNKNOWN",
		},
		"governance_relevance_types": []string{
			"REGULATORY", "LEGAL", "ACCREDITATION", "FORMULARY",
			"PROTOCOL", "ADVISORY", "INFORMATIONAL",
		},
		"program_flags": []string{
			"ONCOLOGY", "PEDIATRIC", "GERIATRIC", "PREGNANCY",
			"RENAL_DOSING", "HEPATIC_DOSING", "PHARMACOGENOMICS",
			"CONTROLLED_SUBSTANCE", "HIGH_ALERT", "LASA",
			"NARROW_TI", "BLACK_BOX_WARNING", "REMS",
		},
	}, nil)
}

// getGovernanceActions handles GET /api/v1/governance/actions
// Returns all governance action definitions
func (h *GovernanceHandlers) getGovernanceActions(c *gin.Context) {
	actions := []map[string]interface{}{
		{
			"action":       "ignore",
			"priority":     0,
			"description":  "No action required - informational only",
			"is_blocking":  false,
			"requires_ack": false,
			"override_by":  "none",
		},
		{
			"action":       "notify",
			"priority":     1,
			"description":  "Notification sent to relevant parties",
			"is_blocking":  false,
			"requires_ack": false,
			"override_by":  "none",
		},
		{
			"action":       "warn_acknowledge",
			"priority":     2,
			"description":  "Warning displayed - clinician acknowledgment required before proceeding",
			"is_blocking":  false,
			"requires_ack": true,
			"override_by":  "clinical",
		},
		{
			"action":       "mandatory_escalation",
			"priority":     3,
			"description":  "Mandatory escalation to senior clinician/pharmacist required",
			"is_blocking":  false,
			"requires_ack": true,
			"override_by":  "senior_clinical",
		},
		{
			"action":       "hard_block_governance_override",
			"priority":     4,
			"description":  "Order blocked - requires governance/P&T committee override",
			"is_blocking":  true,
			"requires_ack": true,
			"override_by":  "governance",
		},
		{
			"action":       "hard_block",
			"priority":     5,
			"description":  "Order blocked - cannot proceed under any circumstances",
			"is_blocking":  true,
			"requires_ack": false,
			"override_by":  "not_overridable",
		},
	}

	sendSuccess(c, map[string]interface{}{
		"governance_actions": actions,
		"severity_mapping_default": map[string]string{
			"contraindicated": "hard_block",
			"major":           "warn_acknowledge",
			"moderate":        "notify",
			"minor":           "ignore",
			"unknown":         "notify",
		},
	}, nil)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// convertToEnhancedResult converts base InteractionResult to EnhancedInteractionResult
func convertToEnhancedResult(ir models.InteractionResult) models.EnhancedInteractionResult {
	// Convert severity string to DDISeverity enum
	severity := models.DDISeverity(ir.Severity)

	// Convert evidence level
	evidence := models.EvidenceLevel(ir.EvidenceLevel)

	// Convert mechanism
	mechanism := models.MechanismPKPD // Default to combined

	// Build confidence
	var confidence *decimal.Decimal
	if ir.ClinicalSignificance != nil {
		conf := decimal.NewFromFloat(*ir.ClinicalSignificance)
		confidence = &conf
	}

	return models.EnhancedInteractionResult{
		InteractionID:          ir.InteractionID,
		Drug1:                  ir.DrugA,
		Drug2:                  ir.DrugB,
		Severity:               severity,
		Mechanism:              mechanism,
		ClinicalEffects:        ir.ClinicalEffect,
		ManagementStrategy:     ir.ManagementStrategy,
		Evidence:               evidence,
		Confidence:             confidence,
		DoseAdjustmentRequired: ir.DoseAdjustmentRequired,
		TimeToOnset:            ir.TimeToOnset,
		Duration:               ir.Duration,
		AlternativeDrugs:       ir.AlternativeDrugs,
	}
}
