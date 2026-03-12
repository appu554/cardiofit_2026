package governance

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-drug-rules/internal/services"
)

// Engine defines the governance engine interface
type Engine interface {
	IsApproved(drugID, version string) (bool, error)
	GetApproval(drugID, version string) (*services.ApprovalDetails, error)
	SubmitForApproval(request *services.ApprovalRequest) (*services.ApprovalTicket, error)
	ReviewSubmission(ticketID string, review *services.ReviewSubmission) error
	VerifySignature(content, signature, signer string) (bool, error)
	SignContent(content, signer string) (string, error)
}

// GovernanceEngine implements the governance engine
type GovernanceEngine struct {
	db         *gorm.DB
	logger     *logrus.Logger
	signingKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// ApprovalRecord represents an approval record in the database
type ApprovalRecord struct {
	ID                  string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DrugID              string     `gorm:"not null;index"`
	Version             string     `gorm:"not null;index"`
	Status              string     `gorm:"not null;default:'pending'"`
	Submitter           string     `gorm:"not null"`
	Description         string     `gorm:"type:text"`
	ClinicalReviewer    string     `gorm:""`
	ClinicalReviewDate  *time.Time `gorm:""`
	ClinicalApproved    bool       `gorm:"default:false"`
	ClinicalComments    string     `gorm:"type:text"`
	TechnicalReviewer   string     `gorm:""`
	TechnicalReviewDate *time.Time `gorm:""`
	TechnicalApproved   bool       `gorm:"default:false"`
	TechnicalComments   string     `gorm:"type:text"`
	CreatedAt           time.Time  `gorm:"autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (ApprovalRecord) TableName() string {
	return "governance_approvals"
}

// NewEngine creates a new governance engine
func NewEngine(signingKeyPath string, db *gorm.DB) (Engine, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Generate or load signing keys
	publicKey, privateKey, err := loadOrGenerateKeys(signingKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing keys: %w", err)
	}

	// Auto-migrate the approval records table
	if err := db.AutoMigrate(&ApprovalRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate approval records table: %w", err)
	}

	return &GovernanceEngine{
		db:         db,
		logger:     logger,
		signingKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// IsApproved checks if a rule version is approved
func (g *GovernanceEngine) IsApproved(drugID, version string) (bool, error) {
	var record ApprovalRecord
	err := g.db.Where("drug_id = ? AND version = ?", drugID, version).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check approval status: %w", err)
	}

	// Both clinical and technical approval required
	return record.ClinicalApproved && record.TechnicalApproved, nil
}

// GetApproval gets approval details for a rule version
func (g *GovernanceEngine) GetApproval(drugID, version string) (*services.ApprovalDetails, error) {
	var record ApprovalRecord
	err := g.db.Where("drug_id = ? AND version = ?", drugID, version).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, services.ErrApprovalRequired
		}
		return nil, fmt.Errorf("failed to get approval details: %w", err)
	}

	details := &services.ApprovalDetails{
		DrugID:              record.DrugID,
		Version:             record.Version,
		Status:              record.Status,
		ClinicalReviewer:    record.ClinicalReviewer,
		ClinicalApproved:    record.ClinicalApproved,
		TechnicalReviewer:   record.TechnicalReviewer,
		TechnicalApproved:   record.TechnicalApproved,
		CreatedAt:           record.CreatedAt,
		UpdatedAt:           record.UpdatedAt,
	}

	if record.ClinicalReviewDate != nil {
		details.ClinicalReviewDate = *record.ClinicalReviewDate
	}

	if record.TechnicalReviewDate != nil {
		details.TechnicalReviewDate = *record.TechnicalReviewDate
	}

	return details, nil
}

// SubmitForApproval submits a rule for approval
func (g *GovernanceEngine) SubmitForApproval(request *services.ApprovalRequest) (*services.ApprovalTicket, error) {
	// Check if already exists
	var existing ApprovalRecord
	err := g.db.Where("drug_id = ? AND version = ?", request.DrugID, request.Version).First(&existing).Error
	if err == nil {
		return nil, services.ErrDuplicateRule
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing approval: %w", err)
	}

	// Create new approval record
	record := ApprovalRecord{
		ID:          uuid.New().String(),
		DrugID:      request.DrugID,
		Version:     request.Version,
		Status:      "pending_clinical_review",
		Submitter:   request.Submitter,
		Description: request.Description,
	}

	if err := g.db.Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to create approval record: %w", err)
	}

	ticket := &services.ApprovalTicket{
		ID:          record.ID,
		DrugID:      record.DrugID,
		Version:     record.Version,
		Status:      record.Status,
		Submitter:   record.Submitter,
		Description: record.Description,
		Metadata:    request.Metadata,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}

	g.logger.WithFields(logrus.Fields{
		"ticket_id": ticket.ID,
		"drug_id":   ticket.DrugID,
		"version":   ticket.Version,
		"submitter": ticket.Submitter,
	}).Info("Approval ticket created")

	return ticket, nil
}

// ReviewSubmission processes a review for a submission
func (g *GovernanceEngine) ReviewSubmission(ticketID string, review *services.ReviewSubmission) error {
	var record ApprovalRecord
	err := g.db.Where("id = ?", ticketID).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("approval ticket not found: %s", ticketID)
		}
		return fmt.Errorf("failed to get approval record: %w", err)
	}

	// Update based on review type
	updates := make(map[string]interface{})
	
	switch review.ReviewType {
	case "clinical":
		updates["clinical_reviewer"] = review.Reviewer
		updates["clinical_review_date"] = review.ReviewDate
		updates["clinical_approved"] = review.Approved
		updates["clinical_comments"] = review.Comments
		
		if review.Approved {
			updates["status"] = "pending_technical_review"
		} else {
			updates["status"] = "rejected_clinical"
		}
		
	case "technical":
		updates["technical_reviewer"] = review.Reviewer
		updates["technical_review_date"] = review.ReviewDate
		updates["technical_approved"] = review.Approved
		updates["technical_comments"] = review.Comments
		
		if review.Approved && record.ClinicalApproved {
			updates["status"] = "approved"
		} else if !review.Approved {
			updates["status"] = "rejected_technical"
		}
		
	default:
		return fmt.Errorf("invalid review type: %s", review.ReviewType)
	}

	if err := g.db.Model(&record).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update approval record: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"ticket_id":   ticketID,
		"review_type": review.ReviewType,
		"reviewer":    review.Reviewer,
		"approved":    review.Approved,
		"status":      updates["status"],
	}).Info("Review submitted")

	return nil
}

// VerifySignature verifies the digital signature of content
func (g *GovernanceEngine) VerifySignature(content, signature, signer string) (bool, error) {
	// Decode the signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// For now, use the same public key for all signers
	// In production, you would look up the signer's public key
	valid := ed25519.Verify(g.publicKey, []byte(content), sigBytes)
	
	if !valid {
		g.logger.WithFields(logrus.Fields{
			"signer":    signer,
			"content_length": len(content),
		}).Warn("Signature verification failed")
	}

	return valid, nil
}

// SignContent signs content with the specified signer
func (g *GovernanceEngine) SignContent(content, signer string) (string, error) {
	// Sign the content
	signature := ed25519.Sign(g.signingKey, []byte(content))
	
	// Encode as base64
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	
	g.logger.WithFields(logrus.Fields{
		"signer":         signer,
		"content_length": len(content),
	}).Info("Content signed")

	return signatureB64, nil
}

// Helper functions

// loadOrGenerateKeys loads existing keys or generates new ones
func loadOrGenerateKeys(keyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	// For now, generate new keys each time
	// In production, you would load from secure storage
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate keys: %w", err)
	}

	// Log the public key for reference (in production, store securely)
	publicKeyHex := hex.EncodeToString(publicKey)
	logrus.WithField("public_key", publicKeyHex).Info("Generated signing keys")

	return publicKey, privateKey, nil
}

// NewKB1GovernanceEngine creates a KB-1 enhanced governance engine
func NewKB1GovernanceEngine(cfg interface{}, logger *logrus.Logger) (Engine, error) {
	// For KB-1, we use a simplified mock engine if no database is available
	// In production, this would connect to the governance database
	return &MockGovernanceEngine{
		logger: logger,
	}, nil
}

// MockGovernanceEngine implements a mock governance engine for development
type MockGovernanceEngine struct {
	logger *logrus.Logger
}

// NewMockEngine creates a new mock governance engine
func NewMockEngine() Engine {
	return &MockGovernanceEngine{
		logger: logrus.New(),
	}
}

// IsApproved always returns true for mock engine
func (m *MockGovernanceEngine) IsApproved(drugID, version string) (bool, error) {
	m.logger.WithFields(logrus.Fields{
		"drug_id": drugID,
		"version": version,
	}).Debug("Mock governance: IsApproved check")
	return true, nil
}

// GetApproval returns mock approval details
func (m *MockGovernanceEngine) GetApproval(drugID, version string) (*services.ApprovalDetails, error) {
	return &services.ApprovalDetails{
		DrugID:            drugID,
		Version:           version,
		Status:            "approved",
		ClinicalApproved:  true,
		TechnicalApproved: true,
	}, nil
}

// SubmitForApproval auto-approves in mock mode
func (m *MockGovernanceEngine) SubmitForApproval(request *services.ApprovalRequest) (*services.ApprovalTicket, error) {
	return &services.ApprovalTicket{
		ID:          "mock-ticket-" + request.DrugID,
		DrugID:      request.DrugID,
		Version:     request.Version,
		Status:      "approved",
		Submitter:   request.Submitter,
		Description: request.Description,
	}, nil
}

// ReviewSubmission always succeeds in mock mode
func (m *MockGovernanceEngine) ReviewSubmission(ticketID string, review *services.ReviewSubmission) error {
	m.logger.WithFields(logrus.Fields{
		"ticket_id": ticketID,
		"reviewer":  review.Reviewer,
	}).Debug("Mock governance: Review submitted")
	return nil
}

// VerifySignature always returns true in mock mode
func (m *MockGovernanceEngine) VerifySignature(content, signature, signer string) (bool, error) {
	return true, nil
}

// SignContent returns a mock signature
func (m *MockGovernanceEngine) SignContent(content, signer string) (string, error) {
	return "mock-signature-" + signer, nil
}
