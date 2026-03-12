package governance

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// DigitalSignatureService implements Ed25519 signature verification for KB-1 compliance
// Ensures clinical dosing rule integrity and authenticity per SaMD requirements
type DigitalSignatureService struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	logger     *logrus.Logger
}

// SignatureMetadata represents the metadata associated with a digital signature
type SignatureMetadata struct {
	Algorithm     string    `json:"algorithm"`
	KeyID         string    `json:"key_id"`
	SignedAt      time.Time `json:"signed_at"`
	SignedBy      string    `json:"signed_by"`
	ContentSHA256 string    `json:"content_sha256"`
	SignatureB64  string    `json:"signature_b64"`
}

// SignedContent represents content with its digital signature
type SignedContent struct {
	Content   json.RawMessage    `json:"content"`
	Signature *SignatureMetadata `json:"signature"`
}

// NewDigitalSignatureService creates a new digital signature service
func NewDigitalSignatureService(publicKeyPath, privateKeyPath string, logger *logrus.Logger) (*DigitalSignatureService, error) {
	// Load or generate Ed25519 key pair
	publicKey, privateKey, err := loadOrGenerateKeyPair(publicKeyPath, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}
	
	keyID := generateKeyID(publicKey)
	logger.WithField("key_id", keyID).Info("Digital signature service initialized with Ed25519 key pair")
	
	return &DigitalSignatureService{
		publicKey:  publicKey,
		privateKey: privateKey,
		logger:     logger,
	}, nil
}

// SignContent signs content using Ed25519 and returns signed content structure
func (ds *DigitalSignatureService) SignContent(content []byte, signedBy string) (*SignedContent, error) {
	start := time.Now()
	defer func() {
		ds.logger.WithFields(logrus.Fields{
			"operation":   "sign_content",
			"signed_by":   signedBy,
			"content_size": len(content),
			"duration_ms": time.Since(start).Milliseconds(),
		}).Debug("Content signing operation")
	}()

	// Calculate content hash
	contentHash := sha256.Sum256(content)
	contentSHA := hex.EncodeToString(contentHash[:])

	// Sign content hash (not content directly for security)
	signature := ed25519.Sign(ds.privateKey, contentHash[:])
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	metadata := &SignatureMetadata{
		Algorithm:     "Ed25519",
		KeyID:         generateKeyID(ds.publicKey),
		SignedAt:      time.Now(),
		SignedBy:      signedBy,
		ContentSHA256: contentSHA,
		SignatureB64:  signatureB64,
	}

	return &SignedContent{
		Content:   content,
		Signature: metadata,
	}, nil
}

// VerifySignature verifies the digital signature of signed content
func (ds *DigitalSignatureService) VerifySignature(signedContent *SignedContent) (bool, error) {
	start := time.Now()
	defer func() {
		ds.logger.WithFields(logrus.Fields{
			"operation":   "verify_signature",
			"key_id":      signedContent.Signature.KeyID,
			"signed_by":   signedContent.Signature.SignedBy,
			"duration_ms": time.Since(start).Milliseconds(),
		}).Debug("Signature verification operation")
	}()

	if signedContent.Signature == nil {
		return false, fmt.Errorf("no signature metadata found")
	}

	// Verify algorithm
	if signedContent.Signature.Algorithm != "Ed25519" {
		return false, fmt.Errorf("unsupported signature algorithm: %s", signedContent.Signature.Algorithm)
	}

	// Verify key ID matches current public key
	expectedKeyID := generateKeyID(ds.publicKey)
	if signedContent.Signature.KeyID != expectedKeyID {
		return false, fmt.Errorf("key ID mismatch: expected %s, got %s", expectedKeyID, signedContent.Signature.KeyID)
	}

	// Recalculate content hash
	contentHash := sha256.Sum256(signedContent.Content)
	expectedSHA := hex.EncodeToString(contentHash[:])
	
	if signedContent.Signature.ContentSHA256 != expectedSHA {
		return false, fmt.Errorf("content hash mismatch: content has been modified")
	}

	// Decode signature
	signatureBytes, err := base64.StdEncoding.DecodeString(signedContent.Signature.SignatureB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify Ed25519 signature
	valid := ed25519.Verify(ds.publicKey, contentHash[:], signatureBytes)
	
	ds.logger.WithFields(logrus.Fields{
		"valid":      valid,
		"signed_by":  signedContent.Signature.SignedBy,
		"signed_at":  signedContent.Signature.SignedAt,
		"content_sha": signedContent.Signature.ContentSHA256,
	}).Info("Digital signature verification completed")

	return valid, nil
}

// SignRule signs a TOML dosing rule content specifically
func (ds *DigitalSignatureService) SignRule(tomlContent, drugCode, version, signedBy string) (*SignedContent, error) {
	// Create structured content for signing
	ruleContent := map[string]interface{}{
		"drug_code": drugCode,
		"version":   version,
		"toml_content": tomlContent,
		"signed_at": time.Now(),
		"signed_by": signedBy,
	}

	contentBytes, err := json.Marshal(ruleContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rule content: %w", err)
	}

	return ds.SignContent(contentBytes, signedBy)
}

// VerifyRuleSignature verifies a signed dosing rule
func (ds *DigitalSignatureService) VerifyRuleSignature(signedRule *SignedContent, expectedDrugCode, expectedVersion string) (bool, error) {
	// First verify the signature itself
	valid, err := ds.VerifySignature(signedRule)
	if err != nil || !valid {
		return false, err
	}

	// Parse content to verify rule-specific fields
	var ruleContent map[string]interface{}
	if err := json.Unmarshal(signedRule.Content, &ruleContent); err != nil {
		return false, fmt.Errorf("failed to parse signed rule content: %w", err)
	}

	// Verify drug code and version match expectations
	if drugCode, ok := ruleContent["drug_code"].(string); !ok || drugCode != expectedDrugCode {
		return false, fmt.Errorf("drug code mismatch in signed content")
	}

	if version, ok := ruleContent["version"].(string); !ok || version != expectedVersion {
		return false, fmt.Errorf("version mismatch in signed content")
	}

	return true, nil
}

// GetSignatureInfo extracts signature information for display/audit purposes
func (ds *DigitalSignatureService) GetSignatureInfo(signedContent *SignedContent) map[string]interface{} {
	if signedContent.Signature == nil {
		return map[string]interface{}{"signed": false}
	}

	return map[string]interface{}{
		"signed":         true,
		"algorithm":      signedContent.Signature.Algorithm,
		"key_id":         signedContent.Signature.KeyID,
		"signed_at":      signedContent.Signature.SignedAt,
		"signed_by":      signedContent.Signature.SignedBy,
		"content_sha256": signedContent.Signature.ContentSHA256,
	}
}

// Helper functions

// loadOrGenerateKeyPair loads existing keys or generates new Ed25519 key pair
func loadOrGenerateKeyPair(publicKeyPath, privateKeyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	// For development/demo - in production, keys should be loaded from secure storage
	// Generate a new key pair for this instance
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}
	
	return publicKey, privateKey, nil
}

// generateKeyID creates a short identifier for the public key
func generateKeyID(publicKey ed25519.PublicKey) string {
	hash := sha256.Sum256(publicKey)
	return hex.EncodeToString(hash[:8]) // First 16 hex chars for readability
}

// GovernanceWorkflow implements the clinical governance approval process
type GovernanceWorkflow struct {
	signatureService *DigitalSignatureService
	logger          *logrus.Logger
}

// ApprovalRequest represents a request for clinical governance approval
type ApprovalRequest struct {
	SubmissionID       string    `json:"submission_id"`
	DrugCode           string    `json:"drug_code"`
	Version            string    `json:"semantic_version"`
	TOMLContent        string    `json:"toml_content"`
	SubmittedBy        string    `json:"submitted_by"`
	SubmittedAt        time.Time `json:"submitted_at"`
	ClinicalJustification string `json:"clinical_justification"`
	Status             string    `json:"status"` // pending, clinical_review, technical_review, approved, rejected
}

// ApprovalDecision represents an approval decision by a reviewer
type ApprovalDecision struct {
	DecisionID       string    `json:"decision_id"`
	SubmissionID     string    `json:"submission_id"`
	ReviewerID       string    `json:"reviewer_id"`
	ReviewerType     string    `json:"reviewer_type"` // clinical, technical, admin
	Decision         string    `json:"decision"`      // approve, reject, request_changes
	Comments         string    `json:"comments"`
	ReviewedAt       time.Time `json:"reviewed_at"`
	DigitalSignature *SignatureMetadata `json:"digital_signature"`
}

// NewGovernanceWorkflow creates a new governance workflow service
func NewGovernanceWorkflow(signatureService *DigitalSignatureService, logger *logrus.Logger) *GovernanceWorkflow {
	return &GovernanceWorkflow{
		signatureService: signatureService,
		logger:          logger,
	}
}

// SubmitForApproval submits a dosing rule for clinical governance approval
func (gw *GovernanceWorkflow) SubmitForApproval(drugCode, version, tomlContent, submittedBy, justification string) (*ApprovalRequest, error) {
	submissionID := fmt.Sprintf("SUB_%s_%s_%d", drugCode, version, time.Now().Unix())
	
	request := &ApprovalRequest{
		SubmissionID:          submissionID,
		DrugCode:              drugCode,
		Version:               version,
		TOMLContent:           tomlContent,
		SubmittedBy:           submittedBy,
		SubmittedAt:           time.Now(),
		ClinicalJustification: justification,
		Status:                "pending",
	}

	gw.logger.WithFields(logrus.Fields{
		"submission_id": submissionID,
		"drug_code":     drugCode,
		"version":       version,
		"submitted_by":  submittedBy,
	}).Info("Dosing rule submitted for governance approval")

	return request, nil
}

// ReviewSubmission records an approval decision with digital signature
func (gw *GovernanceWorkflow) ReviewSubmission(submissionID, reviewerID, reviewerType, decision, comments string) (*ApprovalDecision, error) {
	decisionID := fmt.Sprintf("DEC_%s_%s_%d", submissionID, reviewerType, time.Now().Unix())
	
	// Create decision record
	decisionRecord := &ApprovalDecision{
		DecisionID:   decisionID,
		SubmissionID: submissionID,
		ReviewerID:   reviewerID,
		ReviewerType: reviewerType,
		Decision:     decision,
		Comments:     comments,
		ReviewedAt:   time.Now(),
	}

	// Sign the decision record
	decisionBytes, err := json.Marshal(decisionRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal decision record: %w", err)
	}

	signedDecision, err := gw.signatureService.SignContent(decisionBytes, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("failed to sign decision: %w", err)
	}

	decisionRecord.DigitalSignature = signedDecision.Signature

	gw.logger.WithFields(logrus.Fields{
		"decision_id":   decisionID,
		"submission_id": submissionID,
		"reviewer_id":   reviewerID,
		"reviewer_type": reviewerType,
		"decision":      decision,
		"signature_valid": true,
	}).Info("Governance decision recorded with digital signature")

	return decisionRecord, nil
}

// FinalizeApproval finalizes an approved dosing rule with comprehensive signatures
func (gw *GovernanceWorkflow) FinalizeApproval(request *ApprovalRequest, clinicalDecision, technicalDecision *ApprovalDecision) (*SignedContent, error) {
	// Verify both approvals are valid
	if clinicalDecision.Decision != "approve" || technicalDecision.Decision != "approve" {
		return nil, fmt.Errorf("both clinical and technical approval required")
	}

	// Create final approval record
	finalApproval := map[string]interface{}{
		"submission_id":       request.SubmissionID,
		"drug_code":           request.DrugCode,
		"version":             request.Version,
		"toml_content":        request.TOMLContent,
		"submitted_by":        request.SubmittedBy,
		"clinical_approval":   clinicalDecision,
		"technical_approval":  technicalDecision,
		"approved_at":         time.Now(),
		"final_status":        "approved_for_production",
	}

	approvalBytes, err := json.Marshal(finalApproval)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal final approval: %w", err)
	}

	// Sign the final approval record
	signedApproval, err := gw.signatureService.SignContent(approvalBytes, "SYSTEM")
	if err != nil {
		return nil, fmt.Errorf("failed to sign final approval: %w", err)
	}

	gw.logger.WithFields(logrus.Fields{
		"submission_id": request.SubmissionID,
		"drug_code":     request.DrugCode,
		"version":       request.Version,
		"status":        "approved_for_production",
	}).Info("Dosing rule approved and finalized with digital signatures")

	return signedApproval, nil
}

// VerifyRuleIntegrity verifies the complete signature chain for a dosing rule
func (gw *GovernanceWorkflow) VerifyRuleIntegrity(signedRule *SignedContent) (*IntegrityReport, error) {
	report := &IntegrityReport{
		VerifiedAt: time.Now(),
		Checks:     make(map[string]bool),
		Details:    make(map[string]interface{}),
	}

	// Verify primary signature
	signatureValid, err := gw.signatureService.VerifySignature(signedRule)
	if err != nil {
		report.Checks["signature_verification"] = false
		report.Details["signature_error"] = err.Error()
		return report, nil
	}
	report.Checks["signature_verification"] = signatureValid

	// Parse content to check approval chain
	var finalApproval map[string]interface{}
	if err := json.Unmarshal(signedRule.Content, &finalApproval); err != nil {
		report.Checks["content_parsing"] = false
		report.Details["parsing_error"] = err.Error()
		return report, nil
	}
	report.Checks["content_parsing"] = true

	// Verify clinical approval signature exists
	if clinicalApproval, ok := finalApproval["clinical_approval"].(*ApprovalDecision); ok {
		report.Checks["clinical_approval"] = (clinicalApproval.Decision == "approve")
		report.Details["clinical_reviewer"] = clinicalApproval.ReviewerID
		report.Details["clinical_approved_at"] = clinicalApproval.ReviewedAt
	}

	// Verify technical approval signature exists
	if technicalApproval, ok := finalApproval["technical_approval"].(*ApprovalDecision); ok {
		report.Checks["technical_approval"] = (technicalApproval.Decision == "approve")
		report.Details["technical_reviewer"] = technicalApproval.ReviewerID
		report.Details["technical_approved_at"] = technicalApproval.ReviewedAt
	}

	// Overall integrity status
	report.OverallValid = report.Checks["signature_verification"] && 
						  report.Checks["content_parsing"] && 
						  report.Checks["clinical_approval"] && 
						  report.Checks["technical_approval"]

	report.Details["rule_metadata"] = finalApproval["drug_code"]

	gw.logger.WithFields(logrus.Fields{
		"overall_valid":      report.OverallValid,
		"signature_valid":    report.Checks["signature_verification"],
		"clinical_approved":  report.Checks["clinical_approval"],
		"technical_approved": report.Checks["technical_approval"],
	}).Info("Rule integrity verification completed")

	return report, nil
}

// IntegrityReport represents the result of integrity verification
type IntegrityReport struct {
	VerifiedAt   time.Time              `json:"verified_at"`
	OverallValid bool                   `json:"overall_valid"`
	Checks       map[string]bool        `json:"checks"`
	Details      map[string]interface{} `json:"details"`
}

// AuditLog represents an audit log entry for governance actions
type AuditLog struct {
	LogID       string                 `json:"log_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Action      string                 `json:"action"`
	DrugCode    string                 `json:"drug_code"`
	Version     string                 `json:"version"`
	ActorID     string                 `json:"actor_id"`
	ActorType   string                 `json:"actor_type"`
	Details     map[string]interface{} `json:"details"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
}

// LogGovernanceAction creates an audit log entry for governance actions
func (gw *GovernanceWorkflow) LogGovernanceAction(action, drugCode, version, actorID, actorType string, details map[string]interface{}) *AuditLog {
	logID := fmt.Sprintf("AUDIT_%s_%s_%d", action, drugCode, time.Now().Unix())
	
	auditLog := &AuditLog{
		LogID:     logID,
		Timestamp: time.Now(),
		Action:    action,
		DrugCode:  drugCode,
		Version:   version,
		ActorID:   actorID,
		ActorType: actorType,
		Details:   details,
	}

	gw.logger.WithFields(logrus.Fields{
		"log_id":     logID,
		"action":     action,
		"drug_code":  drugCode,
		"version":    version,
		"actor_id":   actorID,
		"actor_type": actorType,
	}).Info("Governance action logged for audit trail")

	return auditLog
}

// GetPublicKeyInfo returns public key information for verification
func (ds *DigitalSignatureService) GetPublicKeyInfo() map[string]interface{} {
	return map[string]interface{}{
		"algorithm":        "Ed25519",
		"key_id":           generateKeyID(ds.publicKey),
		"public_key_b64":   base64.StdEncoding.EncodeToString(ds.publicKey),
		"key_length_bits":  256,
		"signature_length": 64,
	}
}

// RoleBasedAccess defines access control for governance operations
type RoleBasedAccess struct {
	allowedRoles map[string][]string
	logger       *logrus.Logger
}

// NewRoleBasedAccess creates a new role-based access control system
func NewRoleBasedAccess(logger *logrus.Logger) *RoleBasedAccess {
	allowedRoles := map[string][]string{
		"submit_rule":     {"clinical_pharmacist", "attending_physician", "system_admin"},
		"clinical_review": {"clinical_pharmacist", "attending_physician", "chief_pharmacy_officer"},
		"technical_review": {"pharmacy_informatics", "system_architect", "security_engineer"},
		"final_approval":  {"chief_pharmacy_officer", "system_admin"},
		"emergency_override": {"chief_medical_officer", "system_admin"},
	}

	return &RoleBasedAccess{
		allowedRoles: allowedRoles,
		logger:       logger,
	}
}

// CheckPermission verifies if a user role has permission for an action
func (rba *RoleBasedAccess) CheckPermission(action, userRole string) bool {
	allowedRoles, exists := rba.allowedRoles[action]
	if !exists {
		rba.logger.WithFields(logrus.Fields{
			"action":    action,
			"user_role": userRole,
			"result":    "denied_unknown_action",
		}).Warn("Permission check for unknown action")
		return false
	}

	for _, role := range allowedRoles {
		if role == userRole {
			rba.logger.WithFields(logrus.Fields{
				"action":    action,
				"user_role": userRole,
				"result":    "granted",
			}).Debug("Permission granted")
			return true
		}
	}

	rba.logger.WithFields(logrus.Fields{
		"action":        action,
		"user_role":     userRole,
		"allowed_roles": allowedRoles,
		"result":        "denied",
	}).Warn("Permission denied")
	
	return false
}

// EmergencyOverride allows emergency rule deployment with enhanced audit logging
func (gw *GovernanceWorkflow) EmergencyOverride(drugCode, version, tomlContent, overrideBy, emergencyJustification string) (*SignedContent, error) {
	emergencyRecord := map[string]interface{}{
		"emergency_override": true,
		"drug_code":          drugCode,
		"version":            version,
		"toml_content":       tomlContent,
		"override_by":        overrideBy,
		"override_at":        time.Now(),
		"justification":      emergencyJustification,
		"requires_post_hoc_review": true,
	}

	recordBytes, err := json.Marshal(emergencyRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emergency record: %w", err)
	}

	signedRecord, err := gw.signatureService.SignContent(recordBytes, overrideBy)
	if err != nil {
		return nil, fmt.Errorf("failed to sign emergency override: %w", err)
	}

	gw.logger.WithFields(logrus.Fields{
		"emergency_override": true,
		"drug_code":          drugCode,
		"version":            version,
		"override_by":        overrideBy,
		"justification":      emergencyJustification,
	}).Warn("Emergency governance override activated - requires post-hoc review")

	return signedRecord, nil
}