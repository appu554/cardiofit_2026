package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"safety-gateway-platform/internal/config"
)

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	auditLogger *zap.Logger
}

// New creates a new logger instance based on configuration
func New(cfg config.LoggingConfig) (*Logger, error) {
	// Create main logger
	mainLogger, err := createLogger(cfg.Format, cfg.Output, cfg.AuditFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create main logger: %w", err)
	}

	// Create audit logger if needed
	var auditLogger *zap.Logger
	if cfg.AuditOutput == "file" && cfg.AuditFilePath != "" {
		auditLogger, err = createAuditLogger(cfg.AuditFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
	} else {
		auditLogger = mainLogger
	}

	return &Logger{
		Logger:      mainLogger,
		auditLogger: auditLogger,
	}, nil
}

// createLogger creates a zap logger with the specified configuration
func createLogger(format, output, auditPath string) (*zap.Logger, error) {
	var config zap.Config

	// Set base configuration based on format
	if strings.ToLower(format) == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Configure output paths
	switch strings.ToLower(output) {
	case "stdout":
		config.OutputPaths = []string{"stdout"}
	case "stderr":
		config.OutputPaths = []string{"stderr"}
	case "file":
		if auditPath == "" {
			return nil, fmt.Errorf("file path required for file output")
		}
		config.OutputPaths = []string{auditPath}
	default:
		config.OutputPaths = []string{"stdout"}
	}

	// Configure error output
	config.ErrorOutputPaths = []string{"stderr"}

	// Set encoding configuration for better readability
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	return config.Build()
}

// createAuditLogger creates a dedicated audit logger
func createAuditLogger(auditPath string) (*zap.Logger, error) {
	// Ensure audit directory exists
	if err := os.MkdirAll(strings.TrimSuffix(auditPath, "/audit.log"), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{auditPath}
	config.ErrorOutputPaths = []string{"stderr"}
	
	// Configure for audit logging
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "event"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	return config.Build()
}

// Audit logs an audit event
func (l *Logger) Audit(event string, fields ...zap.Field) {
	// Add audit-specific fields
	auditFields := append([]zap.Field{
		zap.String("event_type", "audit"),
		zap.String("service", "safety-gateway-platform"),
	}, fields...)

	l.auditLogger.Info(event, auditFields...)
}

// WithRequestID adds a request ID to the logger context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.String("request_id", requestID)),
		auditLogger: l.auditLogger.With(zap.String("request_id", requestID)),
	}
}

// WithPatientID adds a patient ID to the logger context (hashed for privacy)
func (l *Logger) WithPatientID(patientID string) *Logger {
	// Hash patient ID for privacy compliance
	hashedID := hashForLogging(patientID)
	return &Logger{
		Logger:      l.Logger.With(zap.String("patient_id_hash", hashedID)),
		auditLogger: l.auditLogger.With(zap.String("patient_id_hash", hashedID)),
	}
}

// WithEngine adds engine information to the logger context
func (l *Logger) WithEngine(engineID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.String("engine_id", engineID)),
		auditLogger: l.auditLogger.With(zap.String("engine_id", engineID)),
	}
}

// LogSafetyDecision logs a safety decision with full context
func (l *Logger) LogSafetyDecision(requestID, patientID, status string, riskScore float64, processingTime int64, engineResults []string) {
	l.Audit("safety_decision",
		zap.String("request_id", requestID),
		zap.String("patient_id_hash", hashForLogging(patientID)),
		zap.String("decision", status),
		zap.Float64("risk_score", riskScore),
		zap.Int64("processing_time_ms", processingTime),
		zap.Strings("engine_results", engineResults),
	)
}

// LogEngineExecution logs engine execution details
func (l *Logger) LogEngineExecution(engineID, status string, duration int64, error string) {
	fields := []zap.Field{
		zap.String("engine_id", engineID),
		zap.String("status", status),
		zap.Int64("duration_ms", duration),
	}

	if error != "" {
		fields = append(fields, zap.String("error", error))
	}

	l.Audit("engine_execution", fields...)
}

// LogOverrideAttempt logs an override attempt
func (l *Logger) LogOverrideAttempt(tokenID, clinicianID, reason string, success bool) {
	l.Audit("override_attempt",
		zap.String("token_id", tokenID),
		zap.String("clinician_id_hash", hashForLogging(clinicianID)),
		zap.String("reason", reason),
		zap.Bool("success", success),
	)
}

// LogContextAssembly logs context assembly performance
func (l *Logger) LogContextAssembly(patientID string, duration int64, sources []string, cacheHit bool) {
	l.Audit("context_assembly",
		zap.String("patient_id_hash", hashForLogging(patientID)),
		zap.Int64("duration_ms", duration),
		zap.Strings("data_sources", sources),
		zap.Bool("cache_hit", cacheHit),
	)
}

// LogCircuitBreakerEvent logs circuit breaker state changes
func (l *Logger) LogCircuitBreakerEvent(engineID, oldState, newState string, failureCount int) {
	l.Audit("circuit_breaker_event",
		zap.String("engine_id", engineID),
		zap.String("old_state", oldState),
		zap.String("new_state", newState),
		zap.Int("failure_count", failureCount),
	)
}

// hashForLogging creates a consistent hash for logging purposes
// This is a simple implementation - in production, use a proper cryptographic hash
func hashForLogging(input string) string {
	if input == "" {
		return ""
	}
	
	// Simple hash for demonstration - replace with proper crypto hash in production
	hash := 0
	for _, char := range input {
		hash = ((hash << 5) - hash) + int(char)
		hash = hash & hash // Convert to 32-bit integer
	}
	
	return fmt.Sprintf("hash_%x", hash)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	if err := l.Logger.Sync(); err != nil {
		return err
	}
	if l.auditLogger != l.Logger {
		return l.auditLogger.Sync()
	}
	return nil
}
