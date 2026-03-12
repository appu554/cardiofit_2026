package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RollbackManager implements the 5-minute rollback mechanism from document 13_9.2
type RollbackManager struct {
	redisClient *redis.Client
	logger      *zap.Logger
	config      *RollbackConfiguration
}

// RollbackConfiguration defines rollback behavior settings
type RollbackConfiguration struct {
	RollbackWindow    time.Duration `json:"rollback_window"`    // 5 minutes per document
	GracePeriod       time.Duration `json:"grace_period"`       // Additional 30 seconds for processing
	MaxRetries        int           `json:"max_retries"`        // Maximum rollback attempts
	RetryDelay        time.Duration `json:"retry_delay"`        // Delay between retry attempts
	NotificationDelay time.Duration `json:"notification_delay"` // Delay before notifying systems
}

// RollbackToken represents a rollback authorization token
type RollbackToken struct {
	Token           string                 `json:"token"`
	CommitID        string                 `json:"commit_id"`
	ProposalID      string                 `json:"proposal_id"`
	WorkflowID      string                 `json:"workflow_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	Status          RollbackTokenStatus    `json:"status"`
	CreatedBy       string                 `json:"created_by,omitempty"`
	OriginalContext map[string]interface{} `json:"original_context"`
}

// RollbackRequest represents a rollback operation request
type RollbackRequest struct {
	Token           string                 `json:"token"`
	Reason          string                 `json:"reason"`
	RequestedBy     string                 `json:"requested_by"`
	Documentation   string                 `json:"documentation,omitempty"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// RollbackResult represents the outcome of a rollback operation
type RollbackResult struct {
	RollbackID        string                 `json:"rollback_id"`
	Status            RollbackStatus         `json:"status"`
	CompensationActions []CompensationAction `json:"compensation_actions"`
	AuditTrail        []AuditEntry           `json:"audit_trail"`
	NotificationsSent []SystemNotification   `json:"notifications_sent"`
	ExecutionTime     time.Duration          `json:"execution_time"`
	ErrorDetails      string                 `json:"error_details,omitempty"`
}

// CompensationAction represents a single compensation action in rollback
type CompensationAction struct {
	ActionID      string                 `json:"action_id"`
	ActionType    CompensationActionType `json:"action_type"`
	TargetSystem  string                 `json:"target_system"`
	ActionData    map[string]interface{} `json:"action_data"`
	Status        ActionStatus           `json:"status"`
	AttemptCount  int                    `json:"attempt_count"`
	ExecutedAt    *time.Time             `json:"executed_at,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

// SystemNotification represents notifications sent during rollback
type SystemNotification struct {
	NotificationID string                 `json:"notification_id"`
	TargetSystem   string                 `json:"target_system"`
	NotificationType NotificationType     `json:"notification_type"`
	Payload        map[string]interface{} `json:"payload"`
	SentAt         time.Time              `json:"sent_at"`
	Acknowledged   bool                   `json:"acknowledged"`
}

// Enums
type RollbackTokenStatus string

const (
	RollbackTokenStatusActive   RollbackTokenStatus = "ACTIVE"
	RollbackTokenStatusUsed     RollbackTokenStatus = "USED"
	RollbackTokenStatusExpired  RollbackTokenStatus = "EXPIRED"
	RollbackTokenStatusRevoked  RollbackTokenStatus = "REVOKED"
)

type RollbackStatus string

const (
	RollbackStatusPending    RollbackStatus = "PENDING"
	RollbackStatusInProgress RollbackStatus = "IN_PROGRESS"
	RollbackStatusCompleted  RollbackStatus = "COMPLETED"
	RollbackStatusFailed     RollbackStatus = "FAILED"
	RollbackStatusPartial    RollbackStatus = "PARTIAL"
)

type CompensationActionType string

const (
	CompensationActionSoftDelete       CompensationActionType = "SOFT_DELETE"
	CompensationActionStateReversion   CompensationActionType = "STATE_REVERSION"
	CompensationActionKafkaCompensation CompensationActionType = "KAFKA_COMPENSATION"
	CompensationActionUINotification   CompensationActionType = "UI_NOTIFICATION"
	CompensationActionAuditReversal    CompensationActionType = "AUDIT_REVERSAL"
)

type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "PENDING"
	ActionStatusExecuting ActionStatus = "EXECUTING"
	ActionStatusCompleted ActionStatus = "COMPLETED"
	ActionStatusFailed    ActionStatus = "FAILED"
	ActionStatusSkipped   ActionStatus = "SKIPPED"
)

// NewRollbackManager creates a new rollback manager with default configuration
func NewRollbackManager(redisClient *redis.Client, logger *zap.Logger) *RollbackManager {
	return &RollbackManager{
		redisClient: redisClient,
		logger:      logger,
		config:      DefaultRollbackConfiguration(),
	}
}

// DefaultRollbackConfiguration returns default rollback settings per document 13_9.2
func DefaultRollbackConfiguration() *RollbackConfiguration {
	return &RollbackConfiguration{
		RollbackWindow:    5 * time.Minute,  // 5-minute window as specified
		GracePeriod:       30 * time.Second, // Additional processing grace period
		MaxRetries:        3,                // Maximum retry attempts for compensation
		RetryDelay:        5 * time.Second,  // Delay between retries
		NotificationDelay: 1 * time.Second,  // Delay before notifying systems
	}
}

// CreateRollbackToken creates a new rollback token with 5-minute expiration
func (r *RollbackManager) CreateRollbackToken(ctx context.Context, commitID, proposalID string) (string, time.Time) {
	now := time.Now()
	expiresAt := now.Add(r.config.RollbackWindow)

	token := r.generateRollbackToken(commitID, proposalID)

	rollbackToken := &RollbackToken{
		Token:      token,
		CommitID:   commitID,
		ProposalID: proposalID,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		Status:     RollbackTokenStatusActive,
		OriginalContext: map[string]interface{}{
			"commit_id":   commitID,
			"proposal_id": proposalID,
		},
	}

	// Store token in Redis with TTL
	if err := r.storeRollbackToken(ctx, rollbackToken); err != nil {
		r.logger.Error("Failed to store rollback token", zap.Error(err))
		// Return token anyway - storage failure shouldn't prevent rollback capability
	}

	r.logger.Info("Rollback token created",
		zap.String("token", token),
		zap.String("commit_id", commitID),
		zap.Time("expires_at", expiresAt))

	return token, expiresAt
}

// ExecuteRollback performs the rollback operation with compensation actions
func (r *RollbackManager) ExecuteRollback(ctx context.Context, request *RollbackRequest) (*RollbackResult, error) {
	startTime := time.Now()
	rollbackID := r.generateRollbackID(request.Token)

	r.logger.Info("Starting rollback execution",
		zap.String("rollback_id", rollbackID),
		zap.String("token", request.Token),
		zap.String("reason", request.Reason),
		zap.String("requested_by", request.RequestedBy))

	// Validate rollback token
	rollbackToken, err := r.validateRollbackToken(ctx, request.Token)
	if err != nil {
		return r.buildFailureResult(rollbackID, startTime, fmt.Sprintf("Token validation failed: %v", err)), err
	}

	// Create audit trail
	auditTrail := []AuditEntry{{
		EntryID:   fmt.Sprintf("%s_start", rollbackID),
		Action:    "ROLLBACK_INITIATED",
		Actor:     request.RequestedBy,
		Context: map[string]interface{}{
			"token":      request.Token,
			"reason":     request.Reason,
			"commit_id":  rollbackToken.CommitID,
		},
		Timestamp: startTime,
	}}

	// Plan compensation actions
	compensationActions := r.planCompensationActions(rollbackToken)

	r.logger.Info("Planned compensation actions",
		zap.String("rollback_id", rollbackID),
		zap.Int("action_count", len(compensationActions)))

	// Execute compensation actions in sequence
	notifications := make([]SystemNotification, 0)
	for i := range compensationActions {
		action := &compensationActions[i]

		if err := r.executeCompensationAction(ctx, rollbackToken, action); err != nil {
			r.logger.Error("Compensation action failed",
				zap.String("action_id", action.ActionID),
				zap.String("action_type", string(action.ActionType)),
				zap.Error(err))

			action.Status = ActionStatusFailed
			action.ErrorMessage = err.Error()

			// Continue with other actions - don't fail entire rollback for one action
		} else {
			action.Status = ActionStatusCompleted
			now := time.Now()
			action.ExecutedAt = &now
		}
	}

	// Send notifications to all affected systems
	systemNotifications := r.generateSystemNotifications(rollbackToken, rollbackID, request.Reason)
	for _, notification := range systemNotifications {
		if err := r.sendSystemNotification(ctx, &notification); err != nil {
			r.logger.Warn("Failed to send system notification",
				zap.String("target_system", notification.TargetSystem),
				zap.Error(err))
			notification.Acknowledged = false
		} else {
			notification.Acknowledged = true
		}
		notifications = append(notifications, notification)
	}

	// Mark token as used
	rollbackToken.Status = RollbackTokenStatusUsed
	r.updateRollbackToken(ctx, rollbackToken)

	// Add completion audit entry
	auditTrail = append(auditTrail, AuditEntry{
		EntryID: fmt.Sprintf("%s_complete", rollbackID),
		Action:  "ROLLBACK_COMPLETED",
		Actor:   request.RequestedBy,
		Context: map[string]interface{}{
			"rollback_id":      rollbackID,
			"compensation_count": len(compensationActions),
			"notifications_sent": len(notifications),
		},
		Timestamp: time.Now(),
	})

	// Determine final status
	status := r.determineRollbackStatus(compensationActions)

	result := &RollbackResult{
		RollbackID:          rollbackID,
		Status:              status,
		CompensationActions: compensationActions,
		AuditTrail:          auditTrail,
		NotificationsSent:   notifications,
		ExecutionTime:       time.Since(startTime),
	}

	r.logger.Info("Rollback execution completed",
		zap.String("rollback_id", rollbackID),
		zap.String("status", string(status)),
		zap.Duration("execution_time", result.ExecutionTime))

	return result, nil
}

// ValidateRollbackToken checks if a rollback token is valid and active
func (r *RollbackManager) ValidateRollbackToken(ctx context.Context, token string) error {
	_, err := r.validateRollbackToken(ctx, token)
	return err
}

// RevokeRollbackToken invalidates a rollback token before expiration
func (r *RollbackManager) RevokeRollbackToken(ctx context.Context, token string, reason string) error {
	rollbackToken, err := r.getRollbackToken(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollback token: %w", err)
	}

	rollbackToken.Status = RollbackTokenStatusRevoked

	if err := r.updateRollbackToken(ctx, rollbackToken); err != nil {
		return fmt.Errorf("failed to revoke rollback token: %w", err)
	}

	r.logger.Info("Rollback token revoked",
		zap.String("token", token),
		zap.String("reason", reason))

	return nil
}

// Helper methods

func (r *RollbackManager) generateRollbackToken(commitID, proposalID string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("rollback_%s_%s_%d", commitID[:8], proposalID[:8], timestamp)
}

func (r *RollbackManager) generateRollbackID(token string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("rb_%s_%d", token[9:17], timestamp) // Extract middle part of token
}

func (r *RollbackManager) storeRollbackToken(ctx context.Context, token *RollbackToken) error {
	key := fmt.Sprintf("rollback:token:%s", token.Token)
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	// Store with TTL slightly longer than rollback window for cleanup
	expiration := r.config.RollbackWindow + r.config.GracePeriod
	return r.redisClient.Set(ctx, key, data, expiration).Err()
}

func (r *RollbackManager) getRollbackToken(ctx context.Context, token string) (*RollbackToken, error) {
	key := fmt.Sprintf("rollback:token:%s", token)
	data, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var rollbackToken RollbackToken
	if err := json.Unmarshal([]byte(data), &rollbackToken); err != nil {
		return nil, err
	}

	return &rollbackToken, nil
}

func (r *RollbackManager) updateRollbackToken(ctx context.Context, token *RollbackToken) error {
	return r.storeRollbackToken(ctx, token)
}

func (r *RollbackManager) validateRollbackToken(ctx context.Context, token string) (*RollbackToken, error) {
	rollbackToken, err := r.getRollbackToken(ctx, token)
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("rollback token not found or expired")
		}
		return nil, fmt.Errorf("failed to retrieve rollback token: %w", err)
	}

	// Check token status
	if rollbackToken.Status != RollbackTokenStatusActive {
		return nil, fmt.Errorf("rollback token is not active: %s", rollbackToken.Status)
	}

	// Check expiration
	if time.Now().After(rollbackToken.ExpiresAt) {
		rollbackToken.Status = RollbackTokenStatusExpired
		r.updateRollbackToken(ctx, rollbackToken)
		return nil, fmt.Errorf("rollback token has expired")
	}

	return rollbackToken, nil
}

func (r *RollbackManager) planCompensationActions(token *RollbackToken) []CompensationAction {
	actions := []CompensationAction{
		// Step 1: Soft delete the committed proposal
		{
			ActionID:     fmt.Sprintf("%s_soft_delete", token.CommitID),
			ActionType:   CompensationActionSoftDelete,
			TargetSystem: "medication_service",
			ActionData: map[string]interface{}{
				"proposal_id": token.ProposalID,
				"commit_id":   token.CommitID,
				"operation":   "soft_delete",
			},
			Status: ActionStatusPending,
		},

		// Step 2: Revert state in workflow engine
		{
			ActionID:     fmt.Sprintf("%s_state_revert", token.CommitID),
			ActionType:   CompensationActionStateReversion,
			TargetSystem: "workflow_engine",
			ActionData: map[string]interface{}{
				"commit_id":    token.CommitID,
				"revert_to":    "pre_commit",
				"preserve_audit": true,
			},
			Status: ActionStatusPending,
		},

		// Step 3: Send compensation events to Kafka
		{
			ActionID:     fmt.Sprintf("%s_kafka_compensation", token.CommitID),
			ActionType:   CompensationActionKafkaCompensation,
			TargetSystem: "kafka",
			ActionData: map[string]interface{}{
				"topic":       "medication-rollbacks",
				"commit_id":   token.CommitID,
				"proposal_id": token.ProposalID,
				"event_type":  "COMPENSATION",
			},
			Status: ActionStatusPending,
		},

		// Step 4: Create audit reversal entry
		{
			ActionID:     fmt.Sprintf("%s_audit_reversal", token.CommitID),
			ActionType:   CompensationActionAuditReversal,
			TargetSystem: "audit_service",
			ActionData: map[string]interface{}{
				"original_commit_id": token.CommitID,
				"reversal_reason":    "rollback_executed",
			},
			Status: ActionStatusPending,
		},
	}

	return actions
}

func (r *RollbackManager) executeCompensationAction(ctx context.Context, token *RollbackToken, action *CompensationAction) error {
	action.Status = ActionStatusExecuting
	action.AttemptCount++

	r.logger.Info("Executing compensation action",
		zap.String("action_id", action.ActionID),
		zap.String("action_type", string(action.ActionType)),
		zap.String("target_system", action.TargetSystem))

	// Add delay for notification timing
	if action.AttemptCount > 1 {
		time.Sleep(r.config.RetryDelay)
	}

	switch action.ActionType {
	case CompensationActionSoftDelete:
		return r.executeSoftDelete(ctx, action)
	case CompensationActionStateReversion:
		return r.executeStateReversion(ctx, action)
	case CompensationActionKafkaCompensation:
		return r.executeKafkaCompensation(ctx, action)
	case CompensationActionAuditReversal:
		return r.executeAuditReversal(ctx, action)
	default:
		return fmt.Errorf("unknown compensation action type: %s", action.ActionType)
	}
}

func (r *RollbackManager) executeSoftDelete(ctx context.Context, action *CompensationAction) error {
	// Implementation would call medication service to soft delete the proposal
	r.logger.Info("Executing soft delete compensation", zap.String("proposal_id", action.ActionData["proposal_id"].(string)))

	// Simulate soft delete operation
	// In real implementation, this would call the medication service API
	time.Sleep(100 * time.Millisecond) // Simulate processing time

	return nil
}

func (r *RollbackManager) executeStateReversion(ctx context.Context, action *CompensationAction) error {
	// Implementation would revert workflow state
	r.logger.Info("Executing state reversion", zap.String("commit_id", action.ActionData["commit_id"].(string)))

	// Simulate state reversion
	time.Sleep(50 * time.Millisecond)

	return nil
}

func (r *RollbackManager) executeKafkaCompensation(ctx context.Context, action *CompensationAction) error {
	// Implementation would publish compensation event to Kafka
	r.logger.Info("Publishing Kafka compensation event", zap.String("topic", action.ActionData["topic"].(string)))

	// Simulate Kafka publish
	time.Sleep(25 * time.Millisecond)

	return nil
}

func (r *RollbackManager) executeAuditReversal(ctx context.Context, action *CompensationAction) error {
	// Implementation would create audit reversal entry
	r.logger.Info("Creating audit reversal entry", zap.String("commit_id", action.ActionData["original_commit_id"].(string)))

	// Simulate audit entry creation
	time.Sleep(75 * time.Millisecond)

	return nil
}

func (r *RollbackManager) generateSystemNotifications(token *RollbackToken, rollbackID, reason string) []SystemNotification {
	return []SystemNotification{
		{
			NotificationID:   fmt.Sprintf("%s_medication_service", rollbackID),
			TargetSystem:     "medication_service",
			NotificationType: NotificationTypeEscalation,
			Payload: map[string]interface{}{
				"rollback_id":   rollbackID,
				"proposal_id":   token.ProposalID,
				"commit_id":     token.CommitID,
				"reason":        reason,
				"action_required": "verify_rollback_completion",
			},
			SentAt: time.Now(),
		},
		{
			NotificationID:   fmt.Sprintf("%s_ui_service", rollbackID),
			TargetSystem:     "ui_service",
			NotificationType: NotificationTypeEscalation,
			Payload: map[string]interface{}{
				"rollback_id": rollbackID,
				"commit_id":   token.CommitID,
				"message":     "Medication proposal has been rolled back",
				"ui_action":   "refresh_proposal_status",
			},
			SentAt: time.Now(),
		},
	}
}

func (r *RollbackManager) sendSystemNotification(ctx context.Context, notification *SystemNotification) error {
	// Implementation would send actual notifications to target systems
	r.logger.Info("Sending system notification",
		zap.String("target_system", notification.TargetSystem),
		zap.String("notification_id", notification.NotificationID))

	// Simulate notification sending
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (r *RollbackManager) determineRollbackStatus(actions []CompensationAction) RollbackStatus {
	completedCount := 0
	failedCount := 0

	for _, action := range actions {
		switch action.Status {
		case ActionStatusCompleted:
			completedCount++
		case ActionStatusFailed:
			failedCount++
		}
	}

	if failedCount == 0 {
		return RollbackStatusCompleted
	} else if completedCount > 0 {
		return RollbackStatusPartial
	} else {
		return RollbackStatusFailed
	}
}

func (r *RollbackManager) buildFailureResult(rollbackID string, startTime time.Time, errorDetails string) *RollbackResult {
	return &RollbackResult{
		RollbackID:        rollbackID,
		Status:            RollbackStatusFailed,
		CompensationActions: []CompensationAction{},
		AuditTrail:        []AuditEntry{},
		NotificationsSent: []SystemNotification{},
		ExecutionTime:     time.Since(startTime),
		ErrorDetails:      errorDetails,
	}
}