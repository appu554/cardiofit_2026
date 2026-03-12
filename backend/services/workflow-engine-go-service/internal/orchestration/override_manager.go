package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OverrideLevel represents the hierarchy of override authority
type OverrideLevel string

const (
	OverrideLevelClinicalJudgment OverrideLevel = "CLINICAL_JUDGMENT"
	OverrideLevelPeerReview       OverrideLevel = "PEER_REVIEW"
	OverrideLevelSupervisory      OverrideLevel = "SUPERVISORY"
	OverrideLevelEmergency        OverrideLevel = "EMERGENCY"
)

// OverrideManager handles clinical override governance and validation
type OverrideManager struct {
	redisClient *redis.Client
	logger      *zap.Logger
}

// NewOverrideManager creates a new override manager instance
func NewOverrideManager(redisClient *redis.Client, logger *zap.Logger) *OverrideManager {
	return &OverrideManager{
		redisClient: redisClient,
		logger:      logger,
	}
}

// OverrideSession represents an active override session
type OverrideSession struct {
	SessionID          string                   `json:"session_id"`
	WorkflowID         string                   `json:"workflow_id"`
	ValidationID       string                   `json:"validation_id"`
	RequiredLevel      OverrideLevel            `json:"required_level"`
	ClinicianID        string                   `json:"clinician_id"`
	Status             string                   `json:"status"`
	CreatedAt          time.Time                `json:"created_at"`
	ExpiresAt          time.Time                `json:"expires_at"`
	ValidationFindings []interface{}            `json:"validation_findings"`
}

// ValidateOverrideAuthority checks if a clinician has the required authority level
func (m *OverrideManager) ValidateOverrideAuthority(
	ctx context.Context,
	clinicianAuthority string,
	requiredLevel OverrideLevel,
) (bool, error) {
	// Map clinician authority to override level
	clinicianLevel := m.mapAuthorityToLevel(clinicianAuthority)

	// Check if clinician level meets or exceeds required level
	return m.meetsRequiredLevel(clinicianLevel, requiredLevel), nil
}

// CreateOverrideSession creates a new override session in Redis
func (m *OverrideManager) CreateOverrideSession(
	ctx context.Context,
	session *OverrideSession,
) error {
	// Serialize session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal override session: %w", err)
	}

	// Store in Redis with expiration
	key := fmt.Sprintf("override:session:%s", session.SessionID)
	ttl := time.Until(session.ExpiresAt)

	if err := m.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store override session: %w", err)
	}

	m.logger.Info("Created override session",
		zap.String("session_id", session.SessionID),
		zap.String("workflow_id", session.WorkflowID),
		zap.String("required_level", string(session.RequiredLevel)))

	return nil
}

// GetOverrideSession retrieves an override session from Redis
func (m *OverrideManager) GetOverrideSession(
	ctx context.Context,
	sessionID string,
) (*OverrideSession, error) {
	key := fmt.Sprintf("override:session:%s", sessionID)

	data, err := m.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("override session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to retrieve override session: %w", err)
	}

	var session OverrideSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal override session: %w", err)
	}

	return &session, nil
}

// ProcessOverrideDecision processes a clinician's override decision
func (m *OverrideManager) ProcessOverrideDecision(
	ctx context.Context,
	sessionID string,
	decision string,
	justification string,
	clinicianID string,
) error {
	// Retrieve the session
	session, err := m.GetOverrideSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Verify the clinician matches the session
	if session.ClinicianID != clinicianID {
		return fmt.Errorf("clinician mismatch for override session")
	}

	// Update session status
	session.Status = decision

	// Store the decision
	decisionKey := fmt.Sprintf("override:decision:%s", sessionID)
	decisionData := map[string]interface{}{
		"session_id":    sessionID,
		"workflow_id":   session.WorkflowID,
		"decision":      decision,
		"justification": justification,
		"clinician_id":  clinicianID,
		"timestamp":     time.Now(),
	}

	data, err := json.Marshal(decisionData)
	if err != nil {
		return fmt.Errorf("failed to marshal decision: %w", err)
	}

	if err := m.redisClient.Set(ctx, decisionKey, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store decision: %w", err)
	}

	m.logger.Info("Processed override decision",
		zap.String("session_id", sessionID),
		zap.String("decision", decision),
		zap.String("clinician_id", clinicianID))

	return nil
}

// mapAuthorityToLevel maps clinician authority string to override level
func (m *OverrideManager) mapAuthorityToLevel(authority string) OverrideLevel {
	switch authority {
	case "emergency":
		return OverrideLevelEmergency
	case "supervisory":
		return OverrideLevelSupervisory
	case "peer_review":
		return OverrideLevelPeerReview
	case "clinical_judgment":
		return OverrideLevelClinicalJudgment
	default:
		return OverrideLevelClinicalJudgment
	}
}

// meetsRequiredLevel checks if clinician level meets required level
func (m *OverrideManager) meetsRequiredLevel(clinicianLevel, requiredLevel OverrideLevel) bool {
	// Define hierarchy levels
	hierarchy := map[OverrideLevel]int{
		OverrideLevelClinicalJudgment: 1,
		OverrideLevelPeerReview:       2,
		OverrideLevelSupervisory:      3,
		OverrideLevelEmergency:        4,
	}

	clinicianPower := hierarchy[clinicianLevel]
	requiredPower := hierarchy[requiredLevel]

	return clinicianPower >= requiredPower
}

// HealthCheck verifies the override manager is operational
func (m *OverrideManager) HealthCheck(ctx context.Context) error {
	// Check Redis connectivity
	if err := m.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}