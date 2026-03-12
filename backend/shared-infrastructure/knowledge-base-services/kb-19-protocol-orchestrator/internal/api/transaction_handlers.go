// Package api provides transaction HTTP handlers for KB-19.
// These handlers expose the Transaction Authority functionality via REST API.
// Part of V3 Architecture: KB-19 as Transaction Authority (Clerk)
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-19-protocol-orchestrator/internal/transaction"
	"kb-19-protocol-orchestrator/pkg/contracts"
)

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	manager *transaction.Manager
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(manager *transaction.Manager) *TransactionHandler {
	return &TransactionHandler{manager: manager}
}

// RegisterTransactionRoutes registers transaction routes on the router
func (h *TransactionHandler) RegisterTransactionRoutes(rg *gin.RouterGroup) {
	txn := rg.Group("/transaction")
	{
		// Transaction lifecycle
		txn.POST("/create", h.HandleCreateTransaction)
		txn.POST("/validate", h.HandleValidateTransaction)
		txn.POST("/override", h.HandleOverrideBlock)
		txn.POST("/commit", h.HandleCommitTransaction)

		// Query endpoints
		txn.GET("/:id", h.HandleGetTransaction)
		txn.GET("/:id/audit", h.HandleGetAuditTrail)
		txn.GET("/patient/:patientId", h.HandleGetPatientTransactions)
	}
}

// =============================================================================
// CREATE TRANSACTION
// =============================================================================

// HandleCreateTransaction handles POST /api/v1/transaction/create
// Creates a new medication transaction in CREATED state
func (h *TransactionHandler) HandleCreateTransaction(c *gin.Context) {
	startTime := time.Now()

	var req contracts.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Convert proposed medication to ClinicalCode
	proposedMed := transaction.ClinicalCode{
		System:  "RxNorm",
		Code:    req.ProposedMedication.RxNormCode,
		Display: req.ProposedMedication.DrugName,
	}

	// Convert current medications to ClinicalCodes (V3: Store for DDI checking)
	var currentMeds []transaction.ClinicalCode
	for _, med := range req.CurrentMedications {
		currentMeds = append(currentMeds, transaction.ClinicalCode{
			System:  "RxNorm",
			Code:    med.RxNormCode,
			Display: med.DrugName,
		})
	}

	// Create transaction
	txn, err := h.manager.CreateTransaction(
		c.Request.Context(),
		req.PatientID,
		req.EncounterID,
		proposedMed,
		currentMeds,
		req.RequestedBy,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, contracts.ErrorResponse{
			Error:   "Failed to create transaction",
			Code:    "CREATE_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Build response
	c.JSON(http.StatusCreated, contracts.CreateTransactionResponse{
		TransactionID: txn.ID,
		State:         txn.State,
		CreatedAt:     txn.CreatedAt,
		SafetyAssessment: contracts.SafetyAssessmentSummary{
			IsBlocked:         false,
			BlockCount:        0,
			RecommendedAction: "validate",
		},
		NextAction:       "validate",
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	})
}

// =============================================================================
// VALIDATE TRANSACTION
// =============================================================================

// HandleValidateTransaction handles POST /api/v1/transaction/validate
// Validates a transaction against DDI rules, Lab rules, and excluded drugs
func (h *TransactionHandler) HandleValidateTransaction(c *gin.Context) {
	startTime := time.Now()

	var req contracts.ValidateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Convert current medications
	var currentMeds []transaction.ClinicalCode
	if req.CurrentSnapshot != nil {
		for _, med := range req.CurrentSnapshot.Medications {
			currentMeds = append(currentMeds, transaction.ClinicalCode{
				System:  "RxNorm",
				Code:    med.RxNormCode,
				Display: med.DrugName,
			})
		}
	}

	// Convert patient labs
	var patientLabs []transaction.LabValue
	if req.CurrentSnapshot != nil {
		for _, lab := range req.CurrentSnapshot.LabResults {
			patientLabs = append(patientLabs, transaction.LabValue{
				Code:      lab.LOINCCode,
				Display:   lab.TestName,
				Value:     lab.Value,
				Unit:      lab.Unit,
				Timestamp: lab.CollectedAt.Format(time.RFC3339),
			})
		}
	}

	// Build patient context with full clinical data for risk assessment
	patientContext := transaction.PatientContext{}
	if req.CurrentSnapshot != nil {
		// Demographics
		patientContext.Sex = req.CurrentSnapshot.Demographics.Gender
		patientContext.Age = req.CurrentSnapshot.Demographics.Age
		if req.CurrentSnapshot.Demographics.WeightKg != nil {
			patientContext.WeightKg = *req.CurrentSnapshot.Demographics.WeightKg
		}
		if req.CurrentSnapshot.Demographics.EGFR != nil {
			patientContext.EGFR = *req.CurrentSnapshot.Demographics.EGFR
		}

		// Pregnancy/Lactation status (critical for teratogenic drug checks)
		patientContext.IsPregnant = req.CurrentSnapshot.Demographics.IsPregnant

		// Allergies (critical for allergy contraindication checks)
		for _, allergy := range req.CurrentSnapshot.Allergies {
			patientContext.Allergies = append(patientContext.Allergies, transaction.ClinicalCode{
				System:  "Allergen",
				Code:    allergy.Allergen,
				Display: allergy.Allergen,
			})
		}

		// Conditions (for contraindication checks)
		for _, condition := range req.CurrentSnapshot.Conditions {
			patientContext.Conditions = append(patientContext.Conditions, transaction.ClinicalCode{
				System:  "ICD-10",
				Code:    condition.ICD10Code,
				Display: condition.ConditionName,
			})
		}
	}

	// Validate transaction
	txn, err := h.manager.ValidateTransaction(
		c.Request.Context(),
		req.TransactionID,
		currentMeds,
		patientLabs,
		[]transaction.ExcludedDrug{}, // Would come from Med-Advisor /risk-profile
		patientContext,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, contracts.ErrorResponse{
			Error:   "Validation failed",
			Code:    "VALIDATION_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Convert hard blocks to response format
	var hardBlocks []contracts.HardBlockSummary
	for _, block := range txn.HardBlocks {
		hardBlocks = append(hardBlocks, contracts.HardBlockSummary{
			ID:          block.ID,
			BlockType:   block.BlockType,
			Severity:    block.Severity,
			Medication:  block.Medication.Display,
			TriggerCode: block.TriggerCondition.Code,
			TriggerName: block.TriggerCondition.Display,
			Reason:      block.Reason,
			RequiresAck: block.RequiresAck,
			AckText:     block.AckText,
			KBSource:    block.KBSource,
			RuleID:      block.RuleID,
		})
	}

	// Determine next action
	nextAction := "commit"
	if len(txn.HardBlocks) > 0 {
		nextAction = "resolve_blocks"
	}

	// Build response
	c.JSON(http.StatusOK, contracts.ValidateTransactionResponse{
		TransactionID: txn.ID,
		State:         txn.State,
		ValidatedAt:   time.Now(),
		IsValid:       len(txn.HardBlocks) == 0,
		PendingBlocks: hardBlocks,
		NextAction:    nextAction,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	})
}

// =============================================================================
// OVERRIDE BLOCK
// =============================================================================

// OverrideBlockRequest represents an override request
type OverrideBlockRequest struct {
	TransactionID  uuid.UUID `json:"transaction_id" binding:"required"`
	BlockID        uuid.UUID `json:"block_id" binding:"required"`
	AcknowledgedBy string    `json:"acknowledged_by" binding:"required"`
	AckText        string    `json:"ack_text" binding:"required"`
	ClinicalReason string    `json:"clinical_reason,omitempty"`
}

// OverrideBlockResponse represents an override response
type OverrideBlockResponse struct {
	TransactionID    uuid.UUID                    `json:"transaction_id"`
	State            transaction.TransactionState `json:"state"`
	BlockOverridden  bool                         `json:"block_overridden"`
	RemainingBlocks  int                          `json:"remaining_blocks"`
	NextAction       string                       `json:"next_action"`
	ProcessingTimeMs int64                        `json:"processing_time_ms"`
}

// HandleOverrideBlock handles POST /api/v1/transaction/override
// Records a provider override for a hard block (requires identity binding)
func (h *TransactionHandler) HandleOverrideBlock(c *gin.Context) {
	startTime := time.Now()

	var req OverrideBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Override the block
	txn, err := h.manager.OverrideBlock(
		c.Request.Context(),
		req.TransactionID,
		req.BlockID,
		req.AcknowledgedBy,
		req.AckText,
		req.ClinicalReason,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Override failed",
			Code:    "OVERRIDE_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Count remaining blocks
	remainingBlocks := len(txn.HardBlocks) - len(txn.OverrideDecisions)
	if remainingBlocks < 0 {
		remainingBlocks = 0
	}

	// Determine next action
	nextAction := "override_remaining"
	if remainingBlocks == 0 {
		nextAction = "commit"
	}

	c.JSON(http.StatusOK, OverrideBlockResponse{
		TransactionID:    txn.ID,
		State:            txn.State,
		BlockOverridden:  true,
		RemainingBlocks:  remainingBlocks,
		NextAction:       nextAction,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	})
}

// =============================================================================
// COMMIT TRANSACTION
// =============================================================================

// CommitRequest represents a commit request
type CommitRequest struct {
	TransactionID uuid.UUID         `json:"transaction_id" binding:"required"`
	CommittedBy   string            `json:"committed_by" binding:"required"`
	KBVersions    map[string]string `json:"kb_versions,omitempty"`
}

// HandleCommitTransaction handles POST /api/v1/transaction/commit
// Finalizes a transaction with governance events and audit trail
func (h *TransactionHandler) HandleCommitTransaction(c *gin.Context) {
	startTime := time.Now()

	var req CommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Default KB versions if not provided
	if req.KBVersions == nil {
		req.KBVersions = map[string]string{
			"KB-5":  "1.0.0", // DDI
			"KB-16": "1.0.0", // Lab rules
			"KB-19": "1.0.0", // Protocol orchestrator
		}
	}

	// Commit transaction
	txn, err := h.manager.CommitTransaction(
		c.Request.Context(),
		req.TransactionID,
		req.CommittedBy,
		req.KBVersions,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Commit failed",
			Code:    "COMMIT_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Convert governance events to response format
	var govEvents []contracts.GovernanceEventSummary
	for _, event := range txn.GovernanceEvents {
		govEvents = append(govEvents, contracts.GovernanceEventSummary{
			EventID:     event.ID,
			EventType:   string(event.EventType),
			Description: event.Description,
			Timestamp:   event.Timestamp,
			Tier:        7, // Tier-7 compliance
		})
	}

	// Convert generated tasks to response format
	var tasks []contracts.GeneratedTaskSummary
	for _, task := range txn.GeneratedTasks {
		tasks = append(tasks, contracts.GeneratedTaskSummary{
			TaskID:      uuid.New(),
			TaskType:    task.TaskType,
			Description: task.Description,
			Priority:    task.Priority,
		})
	}

	// Get audit hash
	auditHash := ""
	if txn.AuditTrail != nil {
		auditHash = txn.AuditTrail.AuditHash
	}

	c.JSON(http.StatusOK, contracts.CommitTransactionResponse{
		TransactionID:    txn.ID,
		State:            txn.State,
		CommittedAt:      *txn.CommittedAt,
		Disposition:      string(txn.Disposition),
		DispositionCode:  string(txn.Disposition),
		GovernanceEvents: govEvents,
		AuditID:          txn.ID,
		AuditHash:        auditHash,
		GeneratedTasks:   tasks,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	})
}

// =============================================================================
// QUERY HANDLERS
// =============================================================================

// HandleGetTransaction handles GET /api/v1/transaction/:id
func (h *TransactionHandler) HandleGetTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid transaction ID",
			Code:    "INVALID_ID",
			Details: err.Error(),
		})
		return
	}

	txn, err := h.manager.GetTransaction(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, contracts.ErrorResponse{
			Error:   "Failed to get transaction",
			Code:    "GET_FAILED",
			Details: err.Error(),
		})
		return
	}
	if txn == nil {
		c.JSON(http.StatusNotFound, contracts.ErrorResponse{
			Error: "Transaction not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Convert hard blocks
	var hardBlocks []contracts.HardBlockSummary
	for _, block := range txn.HardBlocks {
		hardBlocks = append(hardBlocks, contracts.HardBlockSummary{
			ID:          block.ID,
			BlockType:   block.BlockType,
			Severity:    block.Severity,
			Medication:  block.Medication.Display,
			TriggerCode: block.TriggerCondition.Code,
			TriggerName: block.TriggerCondition.Display,
			Reason:      block.Reason,
			RequiresAck: block.RequiresAck,
			AckText:     block.AckText,
			KBSource:    block.KBSource,
			RuleID:      block.RuleID,
		})
	}

	c.JSON(http.StatusOK, contracts.GetTransactionResponse{
		TransactionID: txn.ID,
		PatientID:     txn.PatientID,
		EncounterID:   txn.EncounterID,
		State:         txn.State,
		CreatedAt:     txn.CreatedAt,
		UpdatedAt:     txn.CreatedAt, // Would need to track this
		ProposedMedication: contracts.ProposedMedicationInfo{
			RxNormCode: txn.ProposedMedication.Code,
			DrugName:   txn.ProposedMedication.Display,
		},
		SafetyAssessment: contracts.SafetyAssessmentSummary{
			IsBlocked:  len(txn.HardBlocks) > 0,
			BlockCount: len(txn.HardBlocks),
		},
		HardBlocks: hardBlocks,
	})
}

// HandleGetAuditTrail handles GET /api/v1/transaction/:id/audit
func (h *TransactionHandler) HandleGetAuditTrail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid transaction ID",
			Code:    "INVALID_ID",
			Details: err.Error(),
		})
		return
	}

	auditTrail, err := h.manager.GetAuditTrail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, contracts.ErrorResponse{
			Error:   "Failed to get audit trail",
			Code:    "GET_AUDIT_FAILED",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": id,
		"audit_summary":  auditTrail,
	})
}

// HandleGetPatientTransactions handles GET /api/v1/transaction/patient/:patientId
func (h *TransactionHandler) HandleGetPatientTransactions(c *gin.Context) {
	patientIDStr := c.Param("patientId")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, contracts.ErrorResponse{
			Error:   "Invalid patient ID",
			Code:    "INVALID_ID",
			Details: err.Error(),
		})
		return
	}

	txns, err := h.manager.GetPatientTransactions(c.Request.Context(), patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, contracts.ErrorResponse{
			Error:   "Failed to get transactions",
			Code:    "GET_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Convert to summary list
	var summaries []contracts.TransactionSummary
	for _, txn := range txns {
		summaries = append(summaries, contracts.TransactionSummary{
			TransactionID:  txn.ID,
			PatientID:      txn.PatientID,
			State:          txn.State,
			MedicationName: txn.ProposedMedication.Display,
			MedicationCode: txn.ProposedMedication.Code,
			HasBlocks:      len(txn.HardBlocks) > 0,
			BlockCount:     len(txn.HardBlocks),
			CreatedAt:      txn.CreatedAt,
			UpdatedAt:      txn.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, contracts.ListTransactionsResponse{
		Transactions: summaries,
		Total:        len(summaries),
		HasMore:      false,
	})
}
