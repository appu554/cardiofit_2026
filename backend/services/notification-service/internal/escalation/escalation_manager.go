package escalation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/cardiofit/notification-service/internal/models"
)

// EscalationConfig contains escalation manager configuration
type EscalationConfig struct {
	CriticalTimeoutMinutes int  // Default: 5
	HighTimeoutMinutes     int  // Default: 15
	MaxLevel               int  // Default: 3
	EnableVoiceEscalation  bool // Default: true
}

// DefaultConfig returns default escalation configuration
func DefaultConfig() EscalationConfig {
	return EscalationConfig{
		CriticalTimeoutMinutes: 5,
		HighTimeoutMinutes:     15,
		MaxLevel:               3,
		EnableVoiceEscalation:  true,
	}
}

// EscalationChain tracks the escalation state for an alert
type EscalationChain struct {
	AlertID        string
	CurrentLevel   int
	EscalatedTo    []*models.User
	AcknowledgedBy *models.User
	AcknowledgedAt *time.Time
	CreatedAt      time.Time
}

// EscalationLog represents an escalation log entry from the database
type EscalationLog struct {
	ID                 string
	AlertID            string
	EscalationLevel    int
	EscalatedToUser    string
	EscalatedToRole    string
	EscalatedAt        time.Time
	AcknowledgedAt     *time.Time
	AcknowledgedBy     *string
	Outcome            *string
	ResponseTimeMs     *int
	Metadata           map[string]interface{}
}

// EscalationManager manages timer-based escalation workflows with acknowledgment tracking
type EscalationManager struct {
	db              *pgxpool.Pool
	userService     UserPreferenceService
	deliveryService NotificationDeliveryService
	voiceProvider   VoiceCallProvider
	logger          *zap.Logger
	config          EscalationConfig

	// Timer management
	timers   map[string]*time.Timer // alertID -> timer
	chains   map[string]*EscalationChain // alertID -> chain state
	mu       sync.RWMutex
	shutdownCh chan struct{}
	wg         sync.WaitGroup
}

// UserPreferenceService interface for fetching users by role
type UserPreferenceService interface {
	GetAttendingPhysician(departmentID string) ([]*models.User, error)
	GetChargeNurse(departmentID string) ([]*models.User, error)
	GetPrimaryNurse(patientID string) ([]*models.User, error)
	GetPreferredChannels(user *models.User, severity models.AlertSeverity) []models.NotificationChannel
}

// NotificationDeliveryService interface for sending notifications
type NotificationDeliveryService interface {
	Send(ctx context.Context, notification *models.Notification) error
}

// VoiceCallProvider interface for making voice calls (Twilio)
type VoiceCallProvider interface {
	MakeCall(ctx context.Context, phoneNumber string, message string, metadata map[string]interface{}) (string, error)
}

// NewEscalationManager creates a new escalation manager
func NewEscalationManager(
	db *pgxpool.Pool,
	userService UserPreferenceService,
	deliveryService NotificationDeliveryService,
	voiceProvider VoiceCallProvider,
	logger *zap.Logger,
	config EscalationConfig,
) *EscalationManager {
	mgr := &EscalationManager{
		db:              db,
		userService:     userService,
		deliveryService: deliveryService,
		voiceProvider:   voiceProvider,
		logger:          logger,
		config:          config,
		timers:          make(map[string]*time.Timer),
		chains:          make(map[string]*EscalationChain),
		shutdownCh:      make(chan struct{}),
	}

	// Start background cleanup worker
	mgr.wg.Add(1)
	go mgr.startCleanupWorker()

	return mgr
}

// ScheduleEscalation schedules an escalation timer for an alert
func (e *EscalationManager) ScheduleEscalation(ctx context.Context, alert *models.Alert, timeout time.Duration) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if escalation already scheduled
	if _, exists := e.timers[alert.AlertID]; exists {
		e.logger.Warn("Escalation already scheduled for alert",
			zap.String("alert_id", alert.AlertID),
		)
		return nil
	}

	// Initialize escalation chain
	chain := &EscalationChain{
		AlertID:      alert.AlertID,
		CurrentLevel: 0, // Will start at level 1
		EscalatedTo:  make([]*models.User, 0),
		CreatedAt:    time.Now(),
	}
	e.chains[alert.AlertID] = chain

	// Create timer
	timer := time.AfterFunc(timeout, func() {
		e.handleEscalationTimeout(alert)
	})
	e.timers[alert.AlertID] = timer

	e.logger.Info("Escalation scheduled",
		zap.String("alert_id", alert.AlertID),
		zap.Duration("timeout", timeout),
	)

	return nil
}

// CancelEscalation cancels an escalation timer (called when alert acknowledged)
func (e *EscalationManager) CancelEscalation(ctx context.Context, alertID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	timer, exists := e.timers[alertID]
	if !exists {
		e.logger.Debug("No escalation timer found for alert",
			zap.String("alert_id", alertID),
		)
		return nil
	}

	// Stop timer
	timer.Stop()
	delete(e.timers, alertID)

	// Update chain state
	if chain, ok := e.chains[alertID]; ok {
		now := time.Now()
		chain.AcknowledgedAt = &now
	}

	e.logger.Info("Escalation cancelled",
		zap.String("alert_id", alertID),
	)

	return nil
}

// handleEscalationTimeout is called when an escalation timer fires
func (e *EscalationManager) handleEscalationTimeout(alert *models.Alert) {
	ctx := context.Background()

	e.mu.RLock()
	chain, exists := e.chains[alert.AlertID]
	e.mu.RUnlock()

	if !exists {
		e.logger.Error("Escalation chain not found for timeout",
			zap.String("alert_id", alert.AlertID),
		)
		return
	}

	// Check if already acknowledged
	if chain.AcknowledgedAt != nil {
		e.logger.Info("Alert already acknowledged, skipping escalation",
			zap.String("alert_id", alert.AlertID),
		)
		return
	}

	// Escalate to next level
	nextLevel := chain.CurrentLevel + 1
	if nextLevel > e.config.MaxLevel {
		e.logger.Warn("Maximum escalation level reached",
			zap.String("alert_id", alert.AlertID),
			zap.Int("max_level", e.config.MaxLevel),
		)
		// Log final outcome
		e.logEscalationOutcome(ctx, alert.AlertID, chain.CurrentLevel, "TIMEOUT", "Maximum escalation level reached")
		return
	}

	if err := e.escalateToNextLevel(ctx, alert, nextLevel); err != nil {
		e.logger.Error("Failed to escalate to next level",
			zap.String("alert_id", alert.AlertID),
			zap.Int("next_level", nextLevel),
			zap.Error(err),
		)
		return
	}

	// Schedule next escalation if not at max level
	if nextLevel < e.config.MaxLevel {
		timeout := e.getTimeoutForLevel(nextLevel, alert.Severity)
		e.mu.Lock()
		timer := time.AfterFunc(timeout, func() {
			e.handleEscalationTimeout(alert)
		})
		e.timers[alert.AlertID] = timer
		e.mu.Unlock()

		e.logger.Info("Next escalation scheduled",
			zap.String("alert_id", alert.AlertID),
			zap.Int("next_level", nextLevel+1),
			zap.Duration("timeout", timeout),
		)
	}
}

// escalateToNextLevel escalates an alert to the next level
func (e *EscalationManager) escalateToNextLevel(ctx context.Context, alert *models.Alert, level int) error {
	e.logger.Info("Escalating to level",
		zap.String("alert_id", alert.AlertID),
		zap.Int("level", level),
	)

	// Get target users for this level
	users, err := e.getEscalationUsers(alert, level)
	if err != nil {
		return fmt.Errorf("failed to get escalation users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no users found for escalation level %d", level)
	}

	// Update chain state
	e.mu.Lock()
	chain, exists := e.chains[alert.AlertID]
	if !exists {
		// Create chain if it doesn't exist (for direct escalation calls)
		chain = &EscalationChain{
			AlertID:      alert.AlertID,
			CurrentLevel: 0,
			EscalatedTo:  make([]*models.User, 0),
			CreatedAt:    time.Now(),
		}
		e.chains[alert.AlertID] = chain
	}
	chain.CurrentLevel = level
	chain.EscalatedTo = append(chain.EscalatedTo, users...)
	e.mu.Unlock()

	// Get channels for this level
	channels := e.getEscalationChannels(level, alert.Severity)

	// Send notifications to all users
	for _, user := range users {
		for _, channel := range channels {
			notification := e.buildEscalationNotification(alert, user, channel, level)

			if err := e.deliveryService.Send(ctx, notification); err != nil {
				e.logger.Error("Failed to send escalation notification",
					zap.String("alert_id", alert.AlertID),
					zap.String("user_id", user.ID),
					zap.String("channel", string(channel)),
					zap.Error(err),
				)
				// Continue with other notifications
			} else {
				e.logger.Info("Escalation notification sent",
					zap.String("alert_id", alert.AlertID),
					zap.String("user_id", user.ID),
					zap.String("channel", string(channel)),
					zap.Int("level", level),
				)
			}
		}

		// Log escalation to database
		if err := e.recordEscalation(ctx, alert.AlertID, level, user); err != nil {
			e.logger.Error("Failed to record escalation",
				zap.String("alert_id", alert.AlertID),
				zap.Error(err),
			)
		}
	}

	// Level 3 with voice call for critical alerts
	if level == 3 && e.config.EnableVoiceEscalation && alert.Severity == models.SeverityCritical {
		e.makeEscalationVoiceCall(ctx, alert, users)
	}

	return nil
}

// getEscalationUsers returns the users to escalate to for a given level
func (e *EscalationManager) getEscalationUsers(alert *models.Alert, level int) ([]*models.User, error) {
	switch level {
	case 1:
		// Level 1: Primary Nurse
		return e.userService.GetPrimaryNurse(alert.PatientID)

	case 2:
		// Level 2: Charge Nurse
		return e.userService.GetChargeNurse(alert.DepartmentID)

	case 3:
		// Level 3: Attending Physician
		return e.userService.GetAttendingPhysician(alert.DepartmentID)

	default:
		return nil, fmt.Errorf("invalid escalation level: %d", level)
	}
}

// getEscalationChannels returns the channels to use for a given level
func (e *EscalationManager) getEscalationChannels(level int, severity models.AlertSeverity) []models.NotificationChannel {
	switch level {
	case 1:
		// Level 1: SMS + Push
		return []models.NotificationChannel{models.ChannelSMS, models.ChannelPush}

	case 2:
		// Level 2: SMS + Pager
		return []models.NotificationChannel{models.ChannelSMS, models.ChannelPager}

	case 3:
		// Level 3: SMS + Pager + Voice (voice handled separately)
		if severity == models.SeverityCritical {
			return []models.NotificationChannel{models.ChannelSMS, models.ChannelPager}
		}
		return []models.NotificationChannel{models.ChannelSMS, models.ChannelPager}

	default:
		return []models.NotificationChannel{models.ChannelSMS}
	}
}

// getTimeoutForLevel returns the timeout duration for a given level
func (e *EscalationManager) getTimeoutForLevel(level int, severity models.AlertSeverity) time.Duration {
	if severity == models.SeverityCritical {
		return time.Duration(e.config.CriticalTimeoutMinutes) * time.Minute
	} else if severity == models.SeverityHigh {
		return time.Duration(e.config.HighTimeoutMinutes) * time.Minute
	}
	return 15 * time.Minute // Default
}

// buildEscalationNotification creates a notification for escalation
func (e *EscalationManager) buildEscalationNotification(
	alert *models.Alert,
	user *models.User,
	channel models.NotificationChannel,
	level int,
) *models.Notification {
	now := time.Now()

	message := fmt.Sprintf(
		"ESCALATION LEVEL %d: %s Alert - Patient %s in %s - %s",
		level,
		alert.Severity,
		alert.PatientID,
		alert.PatientLocation.Room,
		alert.Message,
	)

	notification := &models.Notification{
		ID:         fmt.Sprintf("esc-%s-%s-%d", alert.AlertID, user.ID, level),
		AlertID:    alert.AlertID,
		UserID:     user.ID,
		User:       user,
		Alert:      alert,
		Channel:    channel,
		Priority:   1, // Escalations are always highest priority
		Message:    message,
		Status:     models.StatusPending,
		RetryCount: 0,
		CreatedAt:  now,
		Metadata: map[string]interface{}{
			"escalation_level": level,
			"severity":         string(alert.Severity),
			"alert_type":       string(alert.AlertType),
			"patient_id":       alert.PatientID,
			"department_id":    alert.DepartmentID,
		},
	}

	return notification
}

// makeEscalationVoiceCall makes a voice call for level 3 critical escalations
func (e *EscalationManager) makeEscalationVoiceCall(ctx context.Context, alert *models.Alert, users []*models.User) {
	if e.voiceProvider == nil {
		e.logger.Warn("Voice provider not configured, skipping voice call",
			zap.String("alert_id", alert.AlertID),
		)
		return
	}

	message := fmt.Sprintf(
		"Critical alert. Patient %s in %s. Alert type: %s. Confidence: %.0f percent. Immediate attention required.",
		alert.PatientID,
		alert.PatientLocation.Room,
		alert.AlertType,
		alert.Confidence*100,
	)

	for _, user := range users {
		if user.PhoneNumber == "" {
			e.logger.Warn("User has no phone number for voice call",
				zap.String("user_id", user.ID),
				zap.String("user_name", user.Name),
			)
			continue
		}

		callSID, err := e.voiceProvider.MakeCall(ctx, user.PhoneNumber, message, map[string]interface{}{
			"alert_id":   alert.AlertID,
			"patient_id": alert.PatientID,
			"severity":   string(alert.Severity),
		})

		if err != nil {
			e.logger.Error("Failed to make escalation voice call",
				zap.String("alert_id", alert.AlertID),
				zap.String("user_id", user.ID),
				zap.String("phone_number", user.PhoneNumber),
				zap.Error(err),
			)
		} else {
			e.logger.Info("Escalation voice call initiated",
				zap.String("alert_id", alert.AlertID),
				zap.String("user_id", user.ID),
				zap.String("call_sid", callSID),
			)
		}
	}
}

// recordEscalation records an escalation event to the database
func (e *EscalationManager) recordEscalation(ctx context.Context, alertID string, level int, user *models.User) error {
	// Skip database operations if db is nil (testing mode)
	if e.db == nil {
		e.logger.Debug("Skipping database record (nil db)",
			zap.String("alert_id", alertID),
			zap.Int("level", level),
		)
		return nil
	}

	query := `
		INSERT INTO notification_service.escalation_log
		(alert_id, escalation_level, escalated_to_user, escalated_to_role, escalated_at, metadata)
		VALUES ($1, $2, $3, $4, NOW(), $5)
	`

	metadata := map[string]interface{}{
		"user_name":     user.Name,
		"user_email":    user.Email,
		"department_id": user.DepartmentID,
	}

	_, err := e.db.Exec(ctx, query, alertID, level, user.ID, user.Role, metadata)
	if err != nil {
		return fmt.Errorf("failed to insert escalation log: %w", err)
	}

	return nil
}

// RecordAcknowledgment records that an alert was acknowledged by a user
func (e *EscalationManager) RecordAcknowledgment(ctx context.Context, alertID, userID string) error {
	// Cancel escalation timer
	if err := e.CancelEscalation(ctx, alertID); err != nil {
		e.logger.Error("Failed to cancel escalation",
			zap.String("alert_id", alertID),
			zap.Error(err),
		)
	}

	// Skip database operations if db is nil (testing mode)
	if e.db != nil {
		// Update escalation log with acknowledgment
		query := `
			UPDATE notification_service.escalation_log
			SET acknowledged_at = NOW(),
			    acknowledged_by = $1,
			    outcome = 'ACKNOWLEDGED'
			WHERE alert_id = $2
			  AND acknowledged_at IS NULL
		`

		result, err := e.db.Exec(ctx, query, userID, alertID)
		if err != nil {
			return fmt.Errorf("failed to update escalation log: %w", err)
		}

		rowsAffected := result.RowsAffected()
		e.logger.Info("Acknowledgment recorded",
			zap.String("alert_id", alertID),
			zap.String("user_id", userID),
			zap.Int64("rows_affected", rowsAffected),
		)
	}

	// Update chain state
	e.mu.Lock()
	if chain, ok := e.chains[alertID]; ok {
		now := time.Now()
		chain.AcknowledgedAt = &now
		// Would need to fetch user object to set AcknowledgedBy
	}
	e.mu.Unlock()

	return nil
}

// IsAcknowledged checks if an alert has been acknowledged
func (e *EscalationManager) IsAcknowledged(ctx context.Context, alertID string) (bool, *models.User, error) {
	// Check in-memory chain first
	e.mu.RLock()
	chain, exists := e.chains[alertID]
	e.mu.RUnlock()

	if exists && chain.AcknowledgedAt != nil {
		return true, chain.AcknowledgedBy, nil
	}

	// Skip database query if db is nil (testing mode)
	if e.db == nil {
		return false, nil, nil
	}

	query := `
		SELECT acknowledged_by, acknowledged_at
		FROM notification_service.escalation_log
		WHERE alert_id = $1
		  AND acknowledged_at IS NOT NULL
		ORDER BY acknowledged_at DESC
		LIMIT 1
	`

	var acknowledgedBy *string
	var acknowledgedAt *time.Time

	err := e.db.QueryRow(ctx, query, alertID).Scan(&acknowledgedBy, &acknowledgedAt)
	if err != nil {
		// No rows means not acknowledged
		return false, nil, nil
	}

	if acknowledgedBy != nil && acknowledgedAt != nil {
		// Would need to fetch full user object - for now return minimal user
		user := &models.User{
			ID: *acknowledgedBy,
		}
		return true, user, nil
	}

	return false, nil, nil
}

// GetEscalationHistory returns the escalation history for an alert
func (e *EscalationManager) GetEscalationHistory(ctx context.Context, alertID string) ([]*EscalationLog, error) {
	// Skip database query if db is nil (testing mode)
	if e.db == nil {
		return []*EscalationLog{}, nil
	}

	query := `
		SELECT id, alert_id, escalation_level, escalated_to_user, escalated_to_role,
		       escalated_at, acknowledged_at, acknowledged_by, outcome, response_time_ms, metadata
		FROM notification_service.escalation_log
		WHERE alert_id = $1
		ORDER BY escalation_level, escalated_at
	`

	rows, err := e.db.Query(ctx, query, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to query escalation history: %w", err)
	}
	defer rows.Close()

	var logs []*EscalationLog
	for rows.Next() {
		log := &EscalationLog{}
		err := rows.Scan(
			&log.ID,
			&log.AlertID,
			&log.EscalationLevel,
			&log.EscalatedToUser,
			&log.EscalatedToRole,
			&log.EscalatedAt,
			&log.AcknowledgedAt,
			&log.AcknowledgedBy,
			&log.Outcome,
			&log.ResponseTimeMs,
			&log.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan escalation log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// logEscalationOutcome logs a final outcome for an escalation chain
func (e *EscalationManager) logEscalationOutcome(ctx context.Context, alertID string, level int, outcome string, reason string) {
	// Skip database operations if db is nil (testing mode)
	if e.db == nil {
		return
	}

	query := `
		UPDATE notification_service.escalation_log
		SET outcome = $1,
		    metadata = jsonb_set(metadata, '{outcome_reason}', to_jsonb($2::text))
		WHERE alert_id = $3
		  AND escalation_level = $4
		  AND outcome IS NULL
	`

	_, err := e.db.Exec(ctx, query, outcome, reason, alertID, level)
	if err != nil {
		e.logger.Error("Failed to log escalation outcome",
			zap.String("alert_id", alertID),
			zap.Error(err),
		)
	}
}

// startCleanupWorker runs background cleanup of completed timers and old chains
func (e *EscalationManager) startCleanupWorker() {
	defer e.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.cleanupCompletedChains()
		case <-e.shutdownCh:
			return
		}
	}
}

// cleanupCompletedChains removes acknowledged or timed-out chains from memory
func (e *EscalationManager) cleanupCompletedChains() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-30 * time.Minute) // Clean up chains older than 30 minutes

	for alertID, chain := range e.chains {
		// Remove if acknowledged or created more than 30 minutes ago
		if chain.AcknowledgedAt != nil || chain.CreatedAt.Before(cutoff) {
			delete(e.chains, alertID)
			delete(e.timers, alertID)
			e.logger.Debug("Cleaned up escalation chain",
				zap.String("alert_id", alertID),
			)
		}
	}
}

// Shutdown gracefully shuts down the escalation manager
func (e *EscalationManager) Shutdown(ctx context.Context) error {
	e.logger.Info("Shutting down escalation manager")

	// Stop all timers
	e.mu.Lock()
	for alertID, timer := range e.timers {
		timer.Stop()
		e.logger.Debug("Stopped escalation timer",
			zap.String("alert_id", alertID),
		)
	}
	e.mu.Unlock()

	// Signal cleanup worker to stop
	close(e.shutdownCh)

	// Wait for cleanup worker with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info("Escalation manager shutdown complete")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}
