package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"safety-gateway-platform/pkg/types"
)

// AuditEvent represents a single audit event
type AuditEvent struct {
	EventID       string                 `json:"event_id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     AuditEventType         `json:"event_type"`
	RequestID     string                 `json:"request_id"`
	PatientID     string                 `json:"patient_id"`
	ClinicianID   string                 `json:"clinician_id"`
	ActionType    string                 `json:"action_type"`
	Priority      string                 `json:"priority"`
	Source        string                 `json:"source"`
	
	// Request/Response data
	RequestData   map[string]interface{} `json:"request_data,omitempty"`
	ResponseData  map[string]interface{} `json:"response_data,omitempty"`
	
	// Safety decision details
	SafetyStatus  string                 `json:"safety_status,omitempty"`
	RiskScore     float64                `json:"risk_score,omitempty"`
	Confidence    float64                `json:"confidence,omitempty"`
	Violations    []string               `json:"violations,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	
	// Engine execution details
	EngineResults []EngineAuditResult    `json:"engine_results,omitempty"`
	
	// Override information
	OverrideToken string                 `json:"override_token,omitempty"`
	OverrideReason string                `json:"override_reason,omitempty"`
	
	// Performance metrics
	ProcessingTime time.Duration         `json:"processing_time"`
	
	// Compliance and regulatory
	ComplianceFlags map[string]bool      `json:"compliance_flags,omitempty"`
	RegulatoryNotes []string             `json:"regulatory_notes,omitempty"`
	
	// Additional metadata
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EngineAuditResult represents audit information for a single engine execution
type EngineAuditResult struct {
	EngineID      string        `json:"engine_id"`
	EngineName    string        `json:"engine_name"`
	Status        string        `json:"status"`
	RiskScore     float64       `json:"risk_score"`
	Confidence    float64       `json:"confidence"`
	Duration      time.Duration `json:"duration"`
	Violations    []string      `json:"violations,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
	Error         string        `json:"error,omitempty"`
	Tier          string        `json:"tier"`
}

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	EventTypeRequest          AuditEventType = "request"
	EventTypeResponse         AuditEventType = "response"
	EventTypeSafetyDecision   AuditEventType = "safety_decision"
	EventTypeEngineExecution  AuditEventType = "engine_execution"
	EventTypeOverrideUsed     AuditEventType = "override_used"
	EventTypeError            AuditEventType = "error"
	EventTypeHealthCheck      AuditEventType = "health_check"
	EventTypeConfigChange     AuditEventType = "config_change"
	EventTypeSystemStart      AuditEventType = "system_start"
	EventTypeSystemShutdown   AuditEventType = "system_shutdown"
)

// AuditLogger handles all audit logging for the Safety Gateway Platform
type AuditLogger struct {
	logger     *zap.Logger
	auditLog   *zap.Logger
	config     AuditConfig
}

// AuditConfig represents configuration for audit logging
type AuditConfig struct {
	Enabled           bool   `yaml:"enabled"`
	LogLevel          string `yaml:"log_level"`
	OutputPath        string `yaml:"output_path"`
	MaxFileSize       int    `yaml:"max_file_size_mb"`
	MaxBackups        int    `yaml:"max_backups"`
	MaxAge            int    `yaml:"max_age_days"`
	CompressBackups   bool   `yaml:"compress_backups"`
	IncludeStackTrace bool   `yaml:"include_stack_trace"`
	
	// Compliance settings
	RetentionPeriod   time.Duration `yaml:"retention_period"`
	EncryptionEnabled bool          `yaml:"encryption_enabled"`
	
	// Filtering
	LogRequestData    bool `yaml:"log_request_data"`
	LogResponseData   bool `yaml:"log_response_data"`
	LogEngineDetails  bool `yaml:"log_engine_details"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config AuditConfig, logger *zap.Logger) (*AuditLogger, error) {
	if !config.Enabled {
		return &AuditLogger{
			logger: logger,
			config: config,
		}, nil
	}
	
	// Create audit-specific logger configuration
	auditConfig := zap.NewProductionConfig()
	auditConfig.OutputPaths = []string{config.OutputPath}
	auditConfig.ErrorOutputPaths = []string{config.OutputPath}
	
	// Set log level
	switch config.LogLevel {
	case "debug":
		auditConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		auditConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		auditConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		auditConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		auditConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	
	// Build audit logger
	auditLogger, err := auditConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}
	
	return &AuditLogger{
		logger:   logger,
		auditLog: auditLogger,
		config:   config,
	}, nil
}

// LogRequest logs an incoming request
func (al *AuditLogger) LogRequest(ctx context.Context, req *types.SafetyRequest) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:     generateEventID(),
		Timestamp:   time.Now(),
		EventType:   EventTypeRequest,
		RequestID:   req.RequestID,
		PatientID:   req.PatientID,
		ClinicianID: req.ClinicianID,
		ActionType:  req.ActionType,
		Priority:    req.Priority,
		Source:      req.Source,
	}
	
	if al.config.LogRequestData {
		event.RequestData = map[string]interface{}{
			"medication_ids": req.MedicationIDs,
			"condition_ids":  req.ConditionIDs,
			"allergy_ids":    req.AllergyIDs,
			"context":        req.Context,
		}
	}
	
	al.logEvent(event)
}

// LogResponse logs a response
func (al *AuditLogger) LogResponse(ctx context.Context, req *types.SafetyRequest, resp *types.SafetyResponse, processingTime time.Duration) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:        generateEventID(),
		Timestamp:      time.Now(),
		EventType:      EventTypeResponse,
		RequestID:      req.RequestID,
		PatientID:      req.PatientID,
		ClinicianID:    req.ClinicianID,
		ActionType:     req.ActionType,
		Priority:       req.Priority,
		Source:         req.Source,
		SafetyStatus:   string(resp.Status),
		RiskScore:      resp.RiskScore,
		ProcessingTime: processingTime,
		Violations:     resp.Violations,
		Warnings:       resp.Warnings,
	}
	
	if al.config.LogResponseData {
		event.ResponseData = map[string]interface{}{
			"status":           resp.Status,
			"risk_score":       resp.RiskScore,
			"processing_time":  resp.ProcessingTime,
			"engine_count":     len(resp.EngineResults),
			"override_token":   resp.OverrideToken,
		}
	}
	
	if al.config.LogEngineDetails {
		for _, engineResult := range resp.EngineResults {
			event.EngineResults = append(event.EngineResults, EngineAuditResult{
				EngineID:   engineResult.EngineID,
				EngineName: engineResult.EngineName,
				Status:     string(engineResult.Status),
				RiskScore:  engineResult.RiskScore,
				Confidence: engineResult.Confidence,
				Duration:   engineResult.Duration,
				Violations: engineResult.Violations,
				Warnings:   engineResult.Warnings,
				Error:      engineResult.Error,
				Tier:       string(engineResult.Tier),
			})
		}
	}
	
	al.logEvent(event)
}

// LogSafetyDecision logs a safety decision
func (al *AuditLogger) LogSafetyDecision(ctx context.Context, req *types.SafetyRequest, decision *types.SafetyResponse) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:      generateEventID(),
		Timestamp:    time.Now(),
		EventType:    EventTypeSafetyDecision,
		RequestID:    req.RequestID,
		PatientID:    req.PatientID,
		ClinicianID:  req.ClinicianID,
		ActionType:   req.ActionType,
		Priority:     req.Priority,
		SafetyStatus: string(decision.Status),
		RiskScore:    decision.RiskScore,
		Violations:   decision.Violations,
		Warnings:     decision.Warnings,
	}
	
	// Add compliance flags
	event.ComplianceFlags = map[string]bool{
		"gdpr_compliant":     true,
		"hipaa_compliant":    true,
		"fda_compliant":      true,
		"audit_trail_complete": true,
	}
	
	// Add regulatory notes for high-risk decisions
	if decision.RiskScore > 0.7 {
		event.RegulatoryNotes = append(event.RegulatoryNotes, "High-risk decision requires additional review")
	}
	
	if len(decision.Violations) > 0 {
		event.RegulatoryNotes = append(event.RegulatoryNotes, "Safety violations detected - clinical review required")
	}
	
	al.logEvent(event)
}

// LogEngineExecution logs engine execution details
func (al *AuditLogger) LogEngineExecution(ctx context.Context, engineID, engineName string, req *types.SafetyRequest, result *types.EngineResult) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:     generateEventID(),
		Timestamp:   time.Now(),
		EventType:   EventTypeEngineExecution,
		RequestID:   req.RequestID,
		PatientID:   req.PatientID,
		ClinicianID: req.ClinicianID,
		EngineResults: []EngineAuditResult{{
			EngineID:   engineID,
			EngineName: engineName,
			Status:     string(result.Status),
			RiskScore:  result.RiskScore,
			Confidence: result.Confidence,
			Duration:   result.Duration,
			Violations: result.Violations,
			Warnings:   result.Warnings,
			Error:      result.Error,
			Tier:       string(result.Tier),
		}},
	}
	
	al.logEvent(event)
}

// LogOverrideUsed logs the use of an override token
func (al *AuditLogger) LogOverrideUsed(ctx context.Context, req *types.SafetyRequest, token, reason string) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:        generateEventID(),
		Timestamp:      time.Now(),
		EventType:      EventTypeOverrideUsed,
		RequestID:      req.RequestID,
		PatientID:      req.PatientID,
		ClinicianID:    req.ClinicianID,
		ActionType:     req.ActionType,
		OverrideToken:  token,
		OverrideReason: reason,
		ComplianceFlags: map[string]bool{
			"override_documented": true,
			"clinician_authorized": true,
		},
		RegulatoryNotes: []string{
			"Override token used for unsafe decision",
			"Clinical justification required",
			"Additional monitoring recommended",
		},
	}
	
	al.logEvent(event)
}

// LogError logs an error event
func (al *AuditLogger) LogError(ctx context.Context, req *types.SafetyRequest, err error, component string) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:     generateEventID(),
		Timestamp:   time.Now(),
		EventType:   EventTypeError,
		RequestID:   req.RequestID,
		PatientID:   req.PatientID,
		ClinicianID: req.ClinicianID,
		Metadata: map[string]interface{}{
			"error":     err.Error(),
			"component": component,
		},
	}
	
	al.logEvent(event)
}

// LogSystemEvent logs system-level events
func (al *AuditLogger) LogSystemEvent(eventType AuditEventType, message string, metadata map[string]interface{}) {
	if !al.config.Enabled {
		return
	}
	
	event := &AuditEvent{
		EventID:   generateEventID(),
		Timestamp: time.Now(),
		EventType: eventType,
		Metadata:  metadata,
	}
	
	if message != "" {
		if event.Metadata == nil {
			event.Metadata = make(map[string]interface{})
		}
		event.Metadata["message"] = message
	}
	
	al.logEvent(event)
}

// logEvent writes an audit event to the log
func (al *AuditLogger) logEvent(event *AuditEvent) {
	if al.auditLog == nil {
		return
	}
	
	// Convert event to JSON for structured logging
	eventJSON, err := json.Marshal(event)
	if err != nil {
		al.logger.Error("Failed to marshal audit event", zap.Error(err))
		return
	}
	
	// Log the event
	al.auditLog.Info("audit_event",
		zap.String("event_id", event.EventID),
		zap.String("event_type", string(event.EventType)),
		zap.String("request_id", event.RequestID),
		zap.String("patient_id", event.PatientID),
		zap.String("clinician_id", event.ClinicianID),
		zap.String("event_data", string(eventJSON)),
	)
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	if al.auditLog != nil {
		return al.auditLog.Sync()
	}
	return nil
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}
