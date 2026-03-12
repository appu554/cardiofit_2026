package monitoring

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents logging levels
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info" 
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
)

// LoggingConfig holds configuration for structured logging
type LoggingConfig struct {
	Level             LogLevel `yaml:"level"`
	Environment       string   `yaml:"environment"`
	ServiceName       string   `yaml:"service_name"`
	ServiceVersion    string   `yaml:"service_version"`
	
	// Output configuration
	EnableConsole     bool   `yaml:"enable_console"`
	EnableFile        bool   `yaml:"enable_file"`
	EnableAuditFile   bool   `yaml:"enable_audit_file"`
	LogDirectory      string `yaml:"log_directory"`
	
	// File rotation
	MaxFileSize       int    `yaml:"max_file_size_mb"`
	MaxBackups        int    `yaml:"max_backups"`
	MaxAge            int    `yaml:"max_age_days"`
	CompressBackups   bool   `yaml:"compress_backups"`
	
	// Healthcare compliance
	EnableHIPAA       bool   `yaml:"enable_hipaa"`
	EnableAuditTrail  bool   `yaml:"enable_audit_trail"`
	RetentionDays     int    `yaml:"retention_days"`
	
	// Security
	EnableSanitization bool   `yaml:"enable_sanitization"`
	SanitizeFields    []string `yaml:"sanitize_fields"`
}

// AuditEvent represents a healthcare audit event
type AuditEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	EventCategory string                 `json:"event_category"`
	Timestamp     time.Time              `json:"timestamp"`
	
	// Actor information
	UserID        string `json:"user_id,omitempty"`
	UserRole      string `json:"user_role,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
	IPAddress     string `json:"ip_address,omitempty"`
	
	// Resource information
	ResourceType  string `json:"resource_type,omitempty"`
	ResourceID    string `json:"resource_id,omitempty"`
	PatientID     string `json:"patient_id,omitempty"` // Hashed in production
	
	// Operation details
	Operation     string                 `json:"operation"`
	Outcome       string                 `json:"outcome"`
	Description   string                 `json:"description"`
	
	// Technical context
	ServiceName   string                 `json:"service_name"`
	CorrelationID string                 `json:"correlation_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	
	// Healthcare-specific
	ClinicalContext map[string]interface{} `json:"clinical_context,omitempty"`
	SafetyImpact    string                `json:"safety_impact,omitempty"`
	ComplianceFlags []string              `json:"compliance_flags,omitempty"`
	
	// Additional metadata
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"` // authentication, authorization, data_access, etc.
	Severity      string                 `json:"severity"`   // low, medium, high, critical
	Timestamp     time.Time              `json:"timestamp"`
	
	// Security context
	ThreatLevel   string `json:"threat_level"`
	Action        string `json:"action"`
	Resource      string `json:"resource"`
	
	// Actor information
	UserID        string `json:"user_id,omitempty"`
	IPAddress     string `json:"ip_address,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
	
	// Technical details
	ServiceName   string                 `json:"service_name"`
	CorrelationID string                 `json:"correlation_id"`
	
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ClinicalDecisionEvent represents a clinical decision audit event
type ClinicalDecisionEvent struct {
	EventID       string                 `json:"event_id"`
	Timestamp     time.Time              `json:"timestamp"`
	
	// Clinical context
	PatientID     string `json:"patient_id"` // Hashed
	Indication    string `json:"indication"`
	DecisionType  string `json:"decision_type"`
	
	// Decision details
	InputData     map[string]interface{} `json:"input_data"`
	OutputData    map[string]interface{} `json:"output_data"`
	RulesApplied  []string              `json:"rules_applied"`
	Confidence    float64               `json:"confidence_score"`
	
	// Safety information
	SafetyChecks  []string `json:"safety_checks_performed"`
	Warnings      []string `json:"warnings,omitempty"`
	Alerts        []string `json:"alerts,omitempty"`
	
	// System information
	ServiceName   string `json:"service_name"`
	CorrelationID string `json:"correlation_id"`
	
	// Metadata
	ProcessingTime time.Duration         `json:"processing_time"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Logger wraps zap.Logger with healthcare-specific functionality
type Logger struct {
	zapLogger   *zap.Logger
	config      *LoggingConfig
	auditLogger *zap.Logger
	
	// Field sanitizers
	sanitizeFieldsMap map[string]bool
}

// NewLogger creates a new healthcare-compliant logger
func NewLogger(config *LoggingConfig) (*Logger, error) {
	// Create main logger
	zapLogger, err := createZapLogger(config, "application")
	if err != nil {
		return nil, fmt.Errorf("failed to create main logger: %w", err)
	}

	// Create audit logger if enabled
	var auditLogger *zap.Logger
	if config.EnableAuditTrail {
		auditLogger, err = createZapLogger(config, "audit")
		if err != nil {
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
	}

	// Create field sanitizer map
	sanitizeFields := make(map[string]bool)
	for _, field := range config.SanitizeFields {
		sanitizeFields[field] = true
	}

	logger := &Logger{
		zapLogger:      zapLogger,
		auditLogger:    auditLogger,
		config:         config,
		sanitizeFieldsMap: sanitizeFields,
	}

	logger.Info("Logger initialized",
		zap.String("level", string(config.Level)),
		zap.String("service", config.ServiceName),
		zap.Bool("hipaa_enabled", config.EnableHIPAA),
		zap.Bool("audit_enabled", config.EnableAuditTrail),
	)

	return logger, nil
}

// createZapLogger creates a configured zap logger
func createZapLogger(config *LoggingConfig, loggerType string) (*zap.Logger, error) {
	// Convert log level
	zapLevel := zapcore.InfoLevel
	switch config.Level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case PanicLevel:
		zapLevel = zapcore.PanicLevel
	case FatalLevel:
		zapLevel = zapcore.FatalLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create encoder
	var encoder zapcore.Encoder
	if config.Environment == "production" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create writers
	var writers []zapcore.WriteSyncer

	// Console writer
	if config.EnableConsole {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// File writer
	if config.EnableFile {
		filename := filepath.Join(config.LogDirectory, fmt.Sprintf("%s-%s.log", config.ServiceName, loggerType))
		fileWriter, err := createFileWriter(filename, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %w", err)
		}
		writers = append(writers, fileWriter)
	}

	// Audit file writer (for audit logger only)
	if loggerType == "audit" && config.EnableAuditFile {
		auditFilename := filepath.Join(config.LogDirectory, fmt.Sprintf("%s-audit.log", config.ServiceName))
		auditWriter, err := createFileWriter(auditFilename, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit file writer: %w", err)
		}
		writers = append(writers, auditWriter)
	}

	if len(writers) == 0 {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		zapLevel,
	)

	// Add healthcare-specific fields
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).With(
		zap.String("service", config.ServiceName),
		zap.String("version", config.ServiceVersion),
		zap.String("environment", config.Environment),
		zap.String("compliance", "hipaa"),
		zap.String("logger_type", loggerType),
	)

	return logger, nil
}

// createFileWriter creates a file writer with rotation
func createFileWriter(filename string, config *LoggingConfig) (zapcore.WriteSyncer, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// For now, use a simple file writer
	// In production, you'd want to use lumberjack for rotation
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return zapcore.AddSync(file), nil
}

// Standard logging methods

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	fields = l.sanitizeFields(fields)
	l.zapLogger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	fields = l.sanitizeFields(fields)
	l.zapLogger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	fields = l.sanitizeFields(fields)
	l.zapLogger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	fields = l.sanitizeFields(fields)
	l.zapLogger.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	fields = l.sanitizeFields(fields)
	l.zapLogger.Fatal(msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	fields = l.sanitizeFields(fields)
	l.zapLogger.Panic(msg, fields...)
}

// Context-aware logging

func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	correlationID := GetCorrelationID(ctx)
	return l.zapLogger.With(
		zap.String("correlation_id", correlationID),
		zap.String("context", "request"),
	)
}

func (l *Logger) WithFields(fields ...zap.Field) *zap.Logger {
	fields = l.sanitizeFields(fields)
	return l.zapLogger.With(fields...)
}

// Healthcare-specific logging methods

// LogAuditEvent logs a healthcare audit event
func (l *Logger) LogAuditEvent(ctx context.Context, event *AuditEvent) {
	if l.auditLogger == nil {
		return
	}

	// Add correlation ID from context
	if event.CorrelationID == "" {
		event.CorrelationID = GetCorrelationID(ctx)
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Set service name
	event.ServiceName = l.config.ServiceName

	// Log the audit event
	l.auditLogger.Info("audit_event",
		zap.String("event_id", event.EventID),
		zap.String("event_type", event.EventType),
		zap.String("event_category", event.EventCategory),
		zap.Time("timestamp", event.Timestamp),
		zap.String("user_id", event.UserID),
		zap.String("user_role", event.UserRole),
		zap.String("resource_type", event.ResourceType),
		zap.String("resource_id", event.ResourceID),
		zap.String("patient_id", hashSensitiveData(event.PatientID)),
		zap.String("operation", event.Operation),
		zap.String("outcome", event.Outcome),
		zap.String("description", event.Description),
		zap.String("correlation_id", event.CorrelationID),
		zap.Any("clinical_context", event.ClinicalContext),
		zap.String("safety_impact", event.SafetyImpact),
		zap.Strings("compliance_flags", event.ComplianceFlags),
		zap.Any("metadata", event.Metadata),
	)
}

// LogSecurityEvent logs a security-related event
func (l *Logger) LogSecurityEvent(ctx context.Context, event *SecurityEvent) {
	// Add correlation ID from context
	if event.CorrelationID == "" {
		event.CorrelationID = GetCorrelationID(ctx)
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Set service name
	event.ServiceName = l.config.ServiceName

	// Use main logger for security events with special field
	l.zapLogger.Warn("security_event",
		zap.String("security_event_id", event.EventID),
		zap.String("event_type", event.EventType),
		zap.String("severity", event.Severity),
		zap.Time("timestamp", event.Timestamp),
		zap.String("threat_level", event.ThreatLevel),
		zap.String("action", event.Action),
		zap.String("resource", event.Resource),
		zap.String("user_id", event.UserID),
		zap.String("ip_address", event.IPAddress),
		zap.String("correlation_id", event.CorrelationID),
		zap.Any("metadata", event.Metadata),
	)
}

// LogClinicalDecision logs a clinical decision for audit trail
func (l *Logger) LogClinicalDecision(ctx context.Context, event *ClinicalDecisionEvent) {
	if l.auditLogger == nil {
		return
	}

	// Add correlation ID from context
	if event.CorrelationID == "" {
		event.CorrelationID = GetCorrelationID(ctx)
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Set service name
	event.ServiceName = l.config.ServiceName

	l.auditLogger.Info("clinical_decision",
		zap.String("event_id", event.EventID),
		zap.Time("timestamp", event.Timestamp),
		zap.String("patient_id", hashSensitiveData(event.PatientID)),
		zap.String("indication", event.Indication),
		zap.String("decision_type", event.DecisionType),
		zap.Any("input_data", sanitizeHealthData(event.InputData)),
		zap.Any("output_data", sanitizeHealthData(event.OutputData)),
		zap.Strings("rules_applied", event.RulesApplied),
		zap.Float64("confidence", event.Confidence),
		zap.Strings("safety_checks", event.SafetyChecks),
		zap.Strings("warnings", event.Warnings),
		zap.Strings("alerts", event.Alerts),
		zap.String("correlation_id", event.CorrelationID),
		zap.Duration("processing_time", event.ProcessingTime),
		zap.Any("metadata", event.Metadata),
	)
}

// LogPatientAccess logs patient data access for HIPAA compliance
func (l *Logger) LogPatientAccess(ctx context.Context, userID, patientID, operation, purpose string) {
	auditEvent := &AuditEvent{
		EventID:       generateEventID(),
		EventType:     "patient_data_access",
		EventCategory: "data_access",
		UserID:        userID,
		ResourceType:  "patient_data",
		PatientID:     patientID,
		Operation:     operation,
		Description:   fmt.Sprintf("Patient data accessed for: %s", purpose),
		Outcome:       "success",
		ComplianceFlags: []string{"hipaa", "audit_trail"},
	}

	l.LogAuditEvent(ctx, auditEvent)
}

// LogMedicationDecision logs medication-related decisions
func (l *Logger) LogMedicationDecision(ctx context.Context, patientID, indication string, decision map[string]interface{}) {
	clinicalEvent := &ClinicalDecisionEvent{
		EventID:       generateEventID(),
		PatientID:     patientID,
		Indication:    indication,
		DecisionType:  "medication_recommendation",
		OutputData:    decision,
		SafetyChecks:  []string{"drug_interaction", "dosage_validation", "allergy_check"},
		ProcessingTime: time.Since(time.Now().Add(-100 * time.Millisecond)), // Mock processing time
	}

	l.LogClinicalDecision(ctx, clinicalEvent)
}

// Helper functions

// sanitizeFields sanitizes sensitive field values
func (l *Logger) sanitizeFields(fields []zap.Field) []zap.Field {
	if !l.config.EnableSanitization {
		return fields
	}

	sanitized := make([]zap.Field, len(fields))
	for i, field := range fields {
		if l.sanitizeFieldsMap[field.Key] {
			sanitized[i] = zap.String(field.Key, "[REDACTED]")
		} else {
			sanitized[i] = field
		}
	}
	return sanitized
}

// hashSensitiveData hashes sensitive data like patient IDs
func hashSensitiveData(data string) string {
	if data == "" {
		return ""
	}
	// In production, use proper cryptographic hashing
	return fmt.Sprintf("hash_%s", data[len(data)-4:])
}

// sanitizeHealthData removes or hashes sensitive healthcare data
func sanitizeHealthData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "ssn") ||
		   strings.Contains(lowerKey, "patient_id") ||
		   strings.Contains(lowerKey, "medical_record") {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = value
		}
	}
	return sanitized
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	err1 := l.zapLogger.Sync()
	var err2 error
	if l.auditLogger != nil {
		err2 = l.auditLogger.Sync()
	}

	if err1 != nil {
		return err1
	}
	return err2
}

// Close closes the logger and its resources
func (l *Logger) Close() error {
	return l.Sync()
}