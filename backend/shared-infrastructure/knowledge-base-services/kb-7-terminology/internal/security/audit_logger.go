package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	EventTypeAuthentication AuditEventType = "authentication"
	EventTypeAuthorization  AuditEventType = "authorization"
	EventTypeAccess         AuditEventType = "access"
	EventTypeDataAccess     AuditEventType = "data_access"
	EventTypeRateLimit      AuditEventType = "rate_limit"
	EventTypeLicenseCheck   AuditEventType = "license_check"
	EventTypeSecurityError  AuditEventType = "security_error"
	EventTypeSystemEvent    AuditEventType = "system_event"
	EventTypeDataModification AuditEventType = "data_modification"
)

// AuditSeverity represents the severity level of an audit event
type AuditSeverity string

const (
	SeverityInfo     AuditSeverity = "info"
	SeverityWarning  AuditSeverity = "warning"
	SeverityError    AuditSeverity = "error"
	SeverityCritical AuditSeverity = "critical"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID               string                 `json:"id" db:"id"`
	Timestamp        time.Time              `json:"timestamp" db:"timestamp"`
	EventType        AuditEventType         `json:"event_type" db:"event_type"`
	Severity         AuditSeverity          `json:"severity" db:"severity"`
	UserID           string                 `json:"user_id" db:"user_id"`
	SessionID        string                 `json:"session_id,omitempty" db:"session_id"`
	IPAddress        string                 `json:"ip_address" db:"ip_address"`
	UserAgent        string                 `json:"user_agent,omitempty" db:"user_agent"`
	Resource         string                 `json:"resource,omitempty" db:"resource"`
	Action           string                 `json:"action" db:"action"`
	System           string                 `json:"system,omitempty" db:"system"`
	Result           string                 `json:"result" db:"result"` // success, failure, blocked
	Message          string                 `json:"message" db:"message"`
	Details          map[string]interface{} `json:"details,omitempty" db:"details"`
	RequestID        string                 `json:"request_id,omitempty" db:"request_id"`
	Organization     string                 `json:"organization,omitempty" db:"organization"`
	ComplianceFlags  []string               `json:"compliance_flags,omitempty" db:"compliance_flags"`
	RiskScore        int                    `json:"risk_score,omitempty" db:"risk_score"`
	GeoLocation      string                 `json:"geo_location,omitempty" db:"geo_location"`
	DeviceFingerprint string                `json:"device_fingerprint,omitempty" db:"device_fingerprint"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	db     *sql.DB
	logger *zap.Logger
	config *AuditConfig
}

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	EnableAuditLogging    bool          `json:"enable_audit_logging"`
	LogLevel              AuditSeverity `json:"log_level"`
	RetentionPeriod       time.Duration `json:"retention_period"`
	EnableRealTimeAlerts  bool          `json:"enable_real_time_alerts"`
	HighRiskThreshold     int           `json:"high_risk_threshold"`
	EnableComplianceMode  bool          `json:"enable_compliance_mode"`
	EnableGeoTracking     bool          `json:"enable_geo_tracking"`
	MaxEventsPerBatch     int           `json:"max_events_per_batch"`
	BatchFlushInterval    time.Duration `json:"batch_flush_interval"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *sql.DB, logger *zap.Logger, config *AuditConfig) *AuditLogger {
	if config == nil {
		config = &AuditConfig{
			EnableAuditLogging:   true,
			LogLevel:             SeverityInfo,
			RetentionPeriod:      365 * 24 * time.Hour, // 1 year
			EnableRealTimeAlerts: true,
			HighRiskThreshold:    80,
			EnableComplianceMode: true,
			MaxEventsPerBatch:    100,
			BatchFlushInterval:   time.Second * 30,
		}
	}

	al := &AuditLogger{
		db:     db,
		logger: logger,
		config: config,
	}

	// Initialize audit tables
	al.initializeAuditTables()

	return al
}

// LogEvent logs a security audit event
func (al *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	if !al.config.EnableAuditLogging {
		return nil
	}

	// Set default values
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Calculate risk score if not set
	if event.RiskScore == 0 {
		event.RiskScore = al.calculateRiskScore(event)
	}

	// Add compliance flags
	if al.config.EnableComplianceMode {
		event.ComplianceFlags = al.generateComplianceFlags(event)
	}

	// Log to structured logger
	al.logToStructuredLogger(event)

	// Persist to database
	if err := al.persistEvent(ctx, event); err != nil {
		al.logger.Error("Failed to persist audit event", zap.Error(err))
		return err
	}

	// Check for high-risk events and send alerts
	if al.config.EnableRealTimeAlerts && event.RiskScore >= al.config.HighRiskThreshold {
		al.sendSecurityAlert(event)
	}

	return nil
}

// LogAuthentication logs authentication events
func (al *AuditLogger) LogAuthentication(ctx context.Context, userID, method, result, ipAddress string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: EventTypeAuthentication,
		Severity:  al.getSeverityForResult(result),
		UserID:    userID,
		IPAddress: ipAddress,
		Action:    fmt.Sprintf("authenticate_%s", method),
		Result:    result,
		Message:   fmt.Sprintf("User authentication attempt using %s", method),
		Details:   details,
	}

	al.LogEvent(ctx, event)
}

// LogAuthorization logs authorization events
func (al *AuditLogger) LogAuthorization(ctx context.Context, userID, resource, action, result, system string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: EventTypeAuthorization,
		Severity:  al.getSeverityForResult(result),
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		System:    system,
		Result:    result,
		Message:   fmt.Sprintf("Authorization check for %s on %s", action, resource),
		Details:   details,
	}

	al.LogEvent(ctx, event)
}

// LogDataAccess logs data access events
func (al *AuditLogger) LogDataAccess(ctx context.Context, userCtx *UserContext, system, operation, resource string, recordCount int) {
	details := map[string]interface{}{
		"record_count": recordCount,
		"operation":    operation,
		"auth_method":  userCtx.AuthMethod,
	}

	if userCtx.Organization != "" {
		details["organization"] = userCtx.Organization
	}

	event := &AuditEvent{
		EventType:    EventTypeDataAccess,
		Severity:     SeverityInfo,
		UserID:       userCtx.UserID,
		IPAddress:    userCtx.IPAddress,
		UserAgent:    userCtx.UserAgent,
		Resource:     resource,
		Action:       operation,
		System:       system,
		Result:       "success",
		Message:      fmt.Sprintf("Accessed %s data: %s", system, resource),
		Details:      details,
		Organization: userCtx.Organization,
	}

	al.LogEvent(ctx, event)
}

// LogRateLimitViolation logs rate limiting violations
func (al *AuditLogger) LogRateLimitViolation(ctx context.Context, userID, rateLimitKey, operation, limitType string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: EventTypeRateLimit,
		Severity:  SeverityWarning,
		UserID:    userID,
		Action:    operation,
		Result:    "blocked",
		Message:   fmt.Sprintf("Rate limit exceeded: %s", limitType),
		Details: map[string]interface{}{
			"rate_limit_key": rateLimitKey,
			"limit_type":     limitType,
			"operation":      operation,
			"details":        details,
		},
	}

	al.LogEvent(ctx, event)
}

// LogLicenseViolation logs license violations
func (al *AuditLogger) LogLicenseViolation(ctx context.Context, userID, system, violation string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: EventTypeLicenseCheck,
		Severity:  SeverityError,
		UserID:    userID,
		System:    system,
		Action:    "license_check",
		Result:    "violation",
		Message:   fmt.Sprintf("License violation: %s", violation),
		Details:   details,
	}

	al.LogEvent(ctx, event)
}

// LogSecurityError logs security-related errors
func (al *AuditLogger) LogSecurityError(ctx context.Context, userID, errorType, message string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: EventTypeSecurityError,
		Severity:  SeverityError,
		UserID:    userID,
		Action:    errorType,
		Result:    "error",
		Message:   message,
		Details:   details,
	}

	al.LogEvent(ctx, event)
}

// GetAuditTrail retrieves audit trail for a user or resource
func (al *AuditLogger) GetAuditTrail(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*AuditEvent, error) {
	query := `
		SELECT id, timestamp, event_type, severity, user_id, session_id, 
		       ip_address, user_agent, resource, action, system, result, 
		       message, details, request_id, organization, compliance_flags,
		       risk_score, geo_location, device_fingerprint
		FROM audit_events 
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if userID, ok := filters["user_id"].(string); ok && userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if system, ok := filters["system"].(string); ok && system != "" {
		query += fmt.Sprintf(" AND system = $%d", argIndex)
		args = append(args, system)
		argIndex++
	}

	if eventType, ok := filters["event_type"].(string); ok && eventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, eventType)
		argIndex++
	}

	if from, ok := filters["from"].(time.Time); ok {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, from)
		argIndex++
	}

	if to, ok := filters["to"].(time.Time); ok {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, to)
		argIndex++
	}

	query += " ORDER BY timestamp DESC"
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := al.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit trail: %w", err)
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var detailsJSON, complianceFlagsJSON sql.NullString

		err := rows.Scan(
			&event.ID, &event.Timestamp, &event.EventType, &event.Severity,
			&event.UserID, &event.SessionID, &event.IPAddress, &event.UserAgent,
			&event.Resource, &event.Action, &event.System, &event.Result,
			&event.Message, &detailsJSON, &event.RequestID, &event.Organization,
			&complianceFlagsJSON, &event.RiskScore, &event.GeoLocation,
			&event.DeviceFingerprint,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}

		// Parse JSON fields
		if detailsJSON.Valid && detailsJSON.String != "" {
			json.Unmarshal([]byte(detailsJSON.String), &event.Details)
		}

		if complianceFlagsJSON.Valid && complianceFlagsJSON.String != "" {
			json.Unmarshal([]byte(complianceFlagsJSON.String), &event.ComplianceFlags)
		}

		events = append(events, event)
	}

	return events, nil
}

// initializeAuditTables creates audit tables if they don't exist
func (al *AuditLogger) initializeAuditTables() {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS audit_events (
			id VARCHAR(255) PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			event_type VARCHAR(50) NOT NULL,
			severity VARCHAR(20) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			session_id VARCHAR(255),
			ip_address INET,
			user_agent TEXT,
			resource VARCHAR(500),
			action VARCHAR(100) NOT NULL,
			system VARCHAR(50),
			result VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			details JSONB,
			request_id VARCHAR(255),
			organization VARCHAR(255),
			compliance_flags JSONB,
			risk_score INTEGER DEFAULT 0,
			geo_location VARCHAR(255),
			device_fingerprint VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);
		CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id);
		CREATE INDEX IF NOT EXISTS idx_audit_events_system ON audit_events(system);
		CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
		CREATE INDEX IF NOT EXISTS idx_audit_events_risk_score ON audit_events(risk_score);
	`

	_, err := al.db.Exec(createTableSQL)
	if err != nil {
		al.logger.Error("Failed to create audit tables", zap.Error(err))
	}
}

// persistEvent saves an audit event to the database
func (al *AuditLogger) persistEvent(ctx context.Context, event *AuditEvent) error {
	query := `
		INSERT INTO audit_events (
			id, timestamp, event_type, severity, user_id, session_id,
			ip_address, user_agent, resource, action, system, result,
			message, details, request_id, organization, compliance_flags,
			risk_score, geo_location, device_fingerprint
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	detailsJSON, _ := json.Marshal(event.Details)
	complianceFlagsJSON, _ := json.Marshal(event.ComplianceFlags)

	_, err := al.db.ExecContext(ctx, query,
		event.ID, event.Timestamp, event.EventType, event.Severity,
		event.UserID, event.SessionID, event.IPAddress, event.UserAgent,
		event.Resource, event.Action, event.System, event.Result,
		event.Message, string(detailsJSON), event.RequestID, event.Organization,
		string(complianceFlagsJSON), event.RiskScore, event.GeoLocation,
		event.DeviceFingerprint,
	)

	return err
}

// calculateRiskScore calculates a risk score for an audit event
func (al *AuditLogger) calculateRiskScore(event *AuditEvent) int {
	score := 0

	// Base score by event type
	switch event.EventType {
	case EventTypeAuthentication:
		score = 10
	case EventTypeAuthorization:
		score = 20
	case EventTypeDataAccess:
		score = 15
	case EventTypeRateLimit:
		score = 30
	case EventTypeLicenseCheck:
		score = 40
	case EventTypeSecurityError:
		score = 60
	default:
		score = 5
	}

	// Adjust by severity
	switch event.Severity {
	case SeverityInfo:
		score += 0
	case SeverityWarning:
		score += 10
	case SeverityError:
		score += 30
	case SeverityCritical:
		score += 50
	}

	// Adjust by result
	if event.Result == "failure" || event.Result == "blocked" || event.Result == "violation" {
		score += 20
	}

	// Additional risk factors could be added here
	// (e.g., unusual IP, time of day, frequency)

	return min(score, 100) // Cap at 100
}

// generateComplianceFlags generates compliance flags for an event
func (al *AuditLogger) generateComplianceFlags(event *AuditEvent) []string {
	var flags []string

	// HIPAA compliance flags
	if event.EventType == EventTypeDataAccess {
		flags = append(flags, "HIPAA_AUDIT_REQUIRED")
	}

	if event.Severity == SeverityError || event.Severity == SeverityCritical {
		flags = append(flags, "SECURITY_INCIDENT")
	}

	if event.EventType == EventTypeAuthentication && event.Result == "failure" {
		flags = append(flags, "ACCESS_ATTEMPT_FAILED")
	}

	if event.RiskScore >= 70 {
		flags = append(flags, "HIGH_RISK_EVENT")
	}

	return flags
}

// getSeverityForResult determines severity based on result
func (al *AuditLogger) getSeverityForResult(result string) AuditSeverity {
	switch result {
	case "success":
		return SeverityInfo
	case "failure", "blocked":
		return SeverityWarning
	case "violation", "error":
		return SeverityError
	default:
		return SeverityInfo
	}
}

// logToStructuredLogger logs to the structured logger
func (al *AuditLogger) logToStructuredLogger(event *AuditEvent) {
	fields := []zap.Field{
		zap.String("audit_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("user_id", event.UserID),
		zap.String("action", event.Action),
		zap.String("result", event.Result),
		zap.Int("risk_score", event.RiskScore),
	}

	if event.System != "" {
		fields = append(fields, zap.String("system", event.System))
	}

	if event.Resource != "" {
		fields = append(fields, zap.String("resource", event.Resource))
	}

	switch event.Severity {
	case SeverityError, SeverityCritical:
		al.logger.Error(event.Message, fields...)
	case SeverityWarning:
		al.logger.Warn(event.Message, fields...)
	default:
		al.logger.Info(event.Message, fields...)
	}
}

// sendSecurityAlert sends real-time security alerts for high-risk events
func (al *AuditLogger) sendSecurityAlert(event *AuditEvent) {
	// This would integrate with your alerting system
	// For now, just log at critical level
	al.logger.Error("HIGH RISK SECURITY EVENT DETECTED",
		zap.String("audit_id", event.ID),
		zap.String("event_type", string(event.EventType)),
		zap.String("user_id", event.UserID),
		zap.Int("risk_score", event.RiskScore),
		zap.String("message", event.Message))
}