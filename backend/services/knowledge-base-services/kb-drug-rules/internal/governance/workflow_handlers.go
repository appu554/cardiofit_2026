package governance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// WorkflowHandlers provides HTTP handlers for governance workflow operations
type WorkflowHandlers struct {
	db               *gorm.DB
	workflow         *GovernanceWorkflow
	signatureService *DigitalSignatureService
	rbac             *RoleBasedAccess
	logger           *logrus.Logger
}

// NewWorkflowHandlers creates new governance workflow HTTP handlers
func NewWorkflowHandlers(db *gorm.DB, workflow *GovernanceWorkflow, signatureService *DigitalSignatureService, logger *logrus.Logger) *WorkflowHandlers {
	return &WorkflowHandlers{
		db:               db,
		workflow:         workflow,
		signatureService: signatureService,
		rbac:             NewRoleBasedAccess(logger),
		logger:           logger,
	}
}

// SubmitForApprovalRequest represents the request body for submitting a rule for approval
type SubmitForApprovalRequest struct {
	DrugCode              string `json:"drug_code" binding:"required"`
	Version               string `json:"version" binding:"required"`
	TOMLContent           string `json:"toml_content" binding:"required"`
	ClinicalJustification string `json:"clinical_justification" binding:"required"`
}

// ReviewDecisionRequest represents the request body for reviewing a submission
type ReviewDecisionRequest struct {
	SubmissionID string `json:"submission_id" binding:"required"`
	Decision     string `json:"decision" binding:"required,oneof=approve reject request_changes"`
	Comments     string `json:"comments"`
}

// EmergencyOverrideRequest represents the request body for emergency override
type EmergencyOverrideRequest struct {
	DrugCode             string `json:"drug_code" binding:"required"`
	Version              string `json:"version" binding:"required"`
	TOMLContent          string `json:"toml_content" binding:"required"`
	EmergencyJustification string `json:"emergency_justification" binding:"required"`
}

// RegisterRoutes registers governance workflow routes with the Gin router
func (wh *WorkflowHandlers) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/v1/governance")
	{
		// Submission endpoints
		v1.POST("/submit", wh.HandleSubmitForApproval)
		v1.GET("/submissions", wh.HandleListSubmissions)
		v1.GET("/submissions/:id", wh.HandleGetSubmission)
		
		// Review endpoints
		v1.POST("/review/clinical", wh.HandleClinicalReview)
		v1.POST("/review/technical", wh.HandleTechnicalReview)
		v1.GET("/reviews/:submission_id", wh.HandleGetReviews)
		
		// Emergency override
		v1.POST("/emergency-override", wh.HandleEmergencyOverride)
		
		// Signature verification
		v1.POST("/verify-signature", wh.HandleVerifySignature)
		v1.GET("/public-key", wh.HandleGetPublicKey)
		
		// Audit endpoints
		v1.GET("/audit-log", wh.HandleGetAuditLog)
		v1.GET("/integrity-report/:drug_code/:version", wh.HandleIntegrityReport)
	}
}

// HandleSubmitForApproval handles submission of dosing rules for approval
func (wh *WorkflowHandlers) HandleSubmitForApproval(c *gin.Context) {
	var req SubmitForApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Extract user information from context (would come from JWT middleware)
	submittedBy := c.GetHeader("X-User-ID")
	userRole := c.GetHeader("X-User-Role")
	
	if submittedBy == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	// Check permissions
	if !wh.rbac.CheckPermission("submit_rule", userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for rule submission"})
		return
	}

	// Submit for approval
	submission, err := wh.workflow.SubmitForApproval(
		req.DrugCode,
		req.Version,
		req.TOMLContent,
		submittedBy,
		req.ClinicalJustification,
	)
	if err != nil {
		wh.logger.WithError(err).Error("Failed to submit rule for approval")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Submission failed", "details": err.Error()})
		return
	}

	// Store submission in database
	if err := wh.storeSubmission(submission); err != nil {
		wh.logger.WithError(err).Error("Failed to store submission")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store submission"})
		return
	}

	// Log governance action
	auditLog := wh.workflow.LogGovernanceAction(
		"submit_for_approval",
		req.DrugCode,
		req.Version,
		submittedBy,
		userRole,
		map[string]interface{}{
			"submission_id": submission.SubmissionID,
			"justification": req.ClinicalJustification,
		},
	)
	wh.storeAuditLog(auditLog)

	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"submission_id": submission.SubmissionID,
		"status":        submission.Status,
		"estimated_review_time": "2-5 business days",
	})
}

// HandleClinicalReview handles clinical reviewer decisions
func (wh *WorkflowHandlers) HandleClinicalReview(c *gin.Context) {
	var req ReviewDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	reviewerID := c.GetHeader("X-User-ID")
	userRole := c.GetHeader("X-User-Role")

	if !wh.rbac.CheckPermission("clinical_review", userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for clinical review"})
		return
	}

	decision, err := wh.workflow.ReviewSubmission(
		req.SubmissionID,
		reviewerID,
		"clinical",
		req.Decision,
		req.Comments,
	)
	if err != nil {
		wh.logger.WithError(err).Error("Failed to record clinical review")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Review recording failed"})
		return
	}

	// Store decision in database
	if err := wh.storeDecision(decision); err != nil {
		wh.logger.WithError(err).Error("Failed to store clinical decision")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"decision_id": decision.DecisionID,
		"decision":    decision.Decision,
		"signed":      decision.DigitalSignature != nil,
	})
}

// HandleTechnicalReview handles technical reviewer decisions
func (wh *WorkflowHandlers) HandleTechnicalReview(c *gin.Context) {
	var req ReviewDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	reviewerID := c.GetHeader("X-User-ID")
	userRole := c.GetHeader("X-User-Role")

	if !wh.rbac.CheckPermission("technical_review", userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for technical review"})
		return
	}

	decision, err := wh.workflow.ReviewSubmission(
		req.SubmissionID,
		reviewerID,
		"technical",
		req.Decision,
		req.Comments,
	)
	if err != nil {
		wh.logger.WithError(err).Error("Failed to record technical review")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Review recording failed"})
		return
	}

	// Store decision in database
	if err := wh.storeDecision(decision); err != nil {
		wh.logger.WithError(err).Error("Failed to store technical decision")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"decision_id": decision.DecisionID,
		"decision":    decision.Decision,
		"signed":      decision.DigitalSignature != nil,
	})
}

// HandleEmergencyOverride handles emergency rule deployment
func (wh *WorkflowHandlers) HandleEmergencyOverride(c *gin.Context) {
	var req EmergencyOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	overrideBy := c.GetHeader("X-User-ID")
	userRole := c.GetHeader("X-User-Role")

	if !wh.rbac.CheckPermission("emergency_override", userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for emergency override"})
		return
	}

	signedOverride, err := wh.workflow.EmergencyOverride(
		req.DrugCode,
		req.Version,
		req.TOMLContent,
		overrideBy,
		req.EmergencyJustification,
	)
	if err != nil {
		wh.logger.WithError(err).Error("Failed to process emergency override")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Emergency override failed"})
		return
	}

	// Log critical governance action
	auditLog := wh.workflow.LogGovernanceAction(
		"emergency_override",
		req.DrugCode,
		req.Version,
		overrideBy,
		userRole,
		map[string]interface{}{
			"justification": req.EmergencyJustification,
			"requires_post_hoc_review": true,
			"signature_id": signedOverride.Signature.KeyID,
		},
	)
	wh.storeAuditLog(auditLog)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"override_id":  signedOverride.Signature.KeyID,
		"status":       "deployed_emergency",
		"post_hoc_review_required": true,
	})
}

// HandleVerifySignature handles signature verification requests
func (wh *WorkflowHandlers) HandleVerifySignature(c *gin.Context) {
	var signedContent SignedContent
	if err := c.ShouldBindJSON(&signedContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signed content format"})
		return
	}

	valid, err := wh.signatureService.VerifySignature(&signedContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	signatureInfo := wh.signatureService.GetSignatureInfo(&signedContent)
	
	c.JSON(http.StatusOK, gin.H{
		"valid":          valid,
		"signature_info": signatureInfo,
		"verified_at":    time.Now(),
	})
}

// HandleGetPublicKey returns the public key information for signature verification
func (wh *WorkflowHandlers) HandleGetPublicKey(c *gin.Context) {
	publicKeyInfo := wh.signatureService.GetPublicKeyInfo()
	c.JSON(http.StatusOK, publicKeyInfo)
}

// HandleIntegrityReport generates comprehensive integrity report for a specific rule
func (wh *WorkflowHandlers) HandleIntegrityReport(c *gin.Context) {
	drugCode := c.Param("drug_code")
	version := c.Param("version")

	if drugCode == "" || version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Drug code and version required"})
		return
	}

	// Retrieve the signed rule from database
	var storedRule struct {
		SignedContent json.RawMessage `json:"signed_content"`
	}
	
	err := wh.db.Raw(`
		SELECT signed_content 
		FROM dosing_rules 
		WHERE drug_code = ? AND semantic_version = ?
	`, drugCode, version).Scan(&storedRule).Error
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// Parse signed content
	var signedContent SignedContent
	if err := json.Unmarshal(storedRule.SignedContent, &signedContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse signed content"})
		return
	}

	// Generate integrity report
	report, err := wh.workflow.VerifyRuleIntegrity(&signedContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Integrity verification failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drug_code":        drugCode,
		"version":          version,
		"integrity_report": report,
	})
}

// HandleGetAuditLog retrieves audit log entries with filtering
func (wh *WorkflowHandlers) HandleGetAuditLog(c *gin.Context) {
	// Parse query parameters
	drugCode := c.Query("drug_code")
	action := c.Query("action")
	actorID := c.Query("actor_id")
	
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit > 1000 {
		limit = 1000 // Cap at 1000 entries
	}

	// Build dynamic query
	query := "SELECT * FROM governance_audit_log WHERE 1=1"
	var args []interface{}
	argIndex := 1

	if drugCode != "" {
		query += fmt.Sprintf(" AND drug_code = $%d", argIndex)
		args = append(args, drugCode)
		argIndex++
	}
	
	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, action)
		argIndex++
	}
	
	if actorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argIndex)
		args = append(args, actorID)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	var auditEntries []AuditLog
	if err := wh.db.Raw(query, args...).Scan(&auditEntries).Error; err != nil {
		wh.logger.WithError(err).Error("Failed to query audit log")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Audit log query failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_entries": auditEntries,
		"total_count":   len(auditEntries),
		"filters_applied": map[string]string{
			"drug_code": drugCode,
			"action":    action,
			"actor_id":  actorID,
		},
	})
}

// HandleListSubmissions lists pending and recent submissions
func (wh *WorkflowHandlers) HandleListSubmissions(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	query := "SELECT * FROM approval_requests WHERE 1=1"
	var args []interface{}
	argIndex := 1

	if status != "all" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY submitted_at DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	var submissions []ApprovalRequest
	if err := wh.db.Raw(query, args...).Scan(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query submissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"count":       len(submissions),
		"status_filter": status,
	})
}

// HandleGetSubmission retrieves a specific submission by ID
func (wh *WorkflowHandlers) HandleGetSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	
	var submission ApprovalRequest
	if err := wh.db.Raw("SELECT * FROM approval_requests WHERE submission_id = ?", submissionID).Scan(&submission).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	c.JSON(http.StatusOK, submission)
}

// HandleGetReviews retrieves all review decisions for a submission
func (wh *WorkflowHandlers) HandleGetReviews(c *gin.Context) {
	submissionID := c.Param("submission_id")
	
	var decisions []ApprovalDecision
	if err := wh.db.Raw("SELECT * FROM approval_decisions WHERE submission_id = ? ORDER BY reviewed_at", submissionID).Scan(&decisions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submission_id": submissionID,
		"reviews":       decisions,
		"review_count":  len(decisions),
	})
}

// Database helper methods

func (wh *WorkflowHandlers) storeSubmission(submission *ApprovalRequest) error {
	// Marshal for validation, though we store individual fields
	if _, err := json.Marshal(submission); err != nil {
		return err
	}

	return wh.db.Exec(`
		INSERT INTO approval_requests 
		(submission_id, drug_code, semantic_version, toml_content, submitted_by, submitted_at, status, clinical_justification)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, submission.SubmissionID, submission.DrugCode, submission.Version, submission.TOMLContent,
		submission.SubmittedBy, submission.SubmittedAt, submission.Status, submission.ClinicalJustification).Error
}

func (wh *WorkflowHandlers) storeDecision(decision *ApprovalDecision) error {
	signatureJSON, _ := json.Marshal(decision.DigitalSignature)
	
	return wh.db.Exec(`
		INSERT INTO approval_decisions 
		(decision_id, submission_id, reviewer_id, reviewer_type, decision, comments, reviewed_at, digital_signature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, decision.DecisionID, decision.SubmissionID, decision.ReviewerID, decision.ReviewerType,
		decision.Decision, decision.Comments, decision.ReviewedAt, signatureJSON).Error
}

func (wh *WorkflowHandlers) storeAuditLog(auditLog *AuditLog) error {
	detailsJSON, _ := json.Marshal(auditLog.Details)
	
	return wh.db.Exec(`
		INSERT INTO governance_audit_log 
		(log_id, timestamp, action, drug_code, version, actor_id, actor_type, details, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, auditLog.LogID, auditLog.Timestamp, auditLog.Action, auditLog.DrugCode, auditLog.Version,
		auditLog.ActorID, auditLog.ActorType, detailsJSON, auditLog.IPAddress, auditLog.UserAgent).Error
}

// Middleware for extracting user information from JWT tokens
func (wh *WorkflowHandlers) GovernanceAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract JWT token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// In a real implementation, validate JWT and extract user info
		// For now, use headers for user information
		userID := c.GetHeader("X-User-ID")
		userRole := c.GetHeader("X-User-Role")
		
		if userID == "" || userRole == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID and role required"})
			c.Abort()
			return
		}

		// Log governance access
		wh.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"user_role":  userRole,
			"endpoint":   c.Request.URL.Path,
			"method":     c.Request.Method,
			"ip_address": c.ClientIP(),
		}).Info("Governance endpoint access")

		c.Next()
	}
}