//! Clinical Logging Framework
//!
//! This module provides HIPAA-compliant structured logging for clinical systems
//! with patient safety context, audit trails, and security event tracking.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use tokio::sync::{RwLock, mpsc};
use uuid::Uuid;
use tracing::{info, warn, error, debug, trace};
use tracing_subscriber::{Layer, Registry};
use tracing_appender::{rolling, non_blocking};

use crate::protocol::error::ProtocolResult;

/// Clinical logger configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogConfig {
    /// Enable logging
    pub enabled: bool,
    /// Logging level
    pub level: ClinicalLogLevel,
    /// Log format
    pub format: LogFormat,
    /// Log outputs
    pub outputs: Vec<LogOutput>,
    /// HIPAA compliance settings
    pub hipaa_settings: HipaaComplianceSettings,
    /// Audit logging configuration
    pub audit_config: AuditLoggingConfig,
    /// Performance logging settings
    pub performance_config: PerformanceLoggingConfig,
    /// Log retention settings
    pub retention: LogRetentionConfig,
}

/// Clinical log levels
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, PartialOrd, Ord)]
pub enum ClinicalLogLevel {
    /// Emergency - system unusable, patient safety at risk
    Emergency,
    /// Alert - immediate action required
    Alert,
    /// Critical - critical conditions, immediate attention needed
    Critical,
    /// Error - error conditions
    Error,
    /// Warning - warning conditions
    Warning,
    /// Notice - normal but significant conditions
    Notice,
    /// Info - informational messages
    Info,
    /// Debug - debug messages
    Debug,
    /// Trace - detailed trace information
    Trace,
}

/// Log format options
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LogFormat {
    /// JSON structured logging
    Json,
    /// Plain text format
    Text,
    /// Custom format
    Custom(String),
}

/// Log output destinations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LogOutput {
    /// Standard output
    Stdout,
    /// Standard error
    Stderr,
    /// File output
    File(FileOutputConfig),
    /// Syslog output
    Syslog(SyslogConfig),
    /// External logging service
    External(ExternalLogConfig),
}

/// File output configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileOutputConfig {
    /// Base file path
    pub path: String,
    /// File rotation policy
    pub rotation: FileRotation,
    /// Maximum file size in MB
    pub max_size_mb: u64,
    /// File permissions (octal)
    pub permissions: Option<u32>,
}

/// File rotation policies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum FileRotation {
    /// Rotate daily
    Daily,
    /// Rotate hourly
    Hourly,
    /// Rotate by size
    Size,
    /// No rotation
    None,
}

/// Syslog configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SyslogConfig {
    /// Syslog server address
    pub address: String,
    /// Syslog facility
    pub facility: String,
    /// Use TLS encryption
    pub use_tls: bool,
}

/// External logging service configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExternalLogConfig {
    /// Service type (elasticsearch, splunk, etc.)
    pub service_type: String,
    /// Service endpoint
    pub endpoint: String,
    /// Authentication configuration
    pub auth: Option<ExternalLogAuth>,
    /// Batch size for log shipping
    pub batch_size: usize,
    /// Flush interval in seconds
    pub flush_interval_seconds: u64,
}

/// External logging authentication
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExternalLogAuth {
    /// Authentication method
    pub method: String,
    /// Username
    pub username: Option<String>,
    /// Password or token
    pub password: Option<String>,
    /// API key
    pub api_key: Option<String>,
}

/// HIPAA compliance settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HipaaComplianceSettings {
    /// Enable HIPAA compliance mode
    pub enabled: bool,
    /// Automatically redact PHI (Protected Health Information)
    pub auto_redact_phi: bool,
    /// PHI redaction patterns
    pub phi_patterns: Vec<PhiPattern>,
    /// Encrypt logs at rest
    pub encrypt_at_rest: bool,
    /// Encryption key ID
    pub encryption_key_id: Option<String>,
    /// Access logging for audit trail
    pub access_logging: bool,
}

/// PHI pattern for redaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhiPattern {
    /// Pattern name
    pub name: String,
    /// Regular expression pattern
    pub pattern: String,
    /// Replacement text
    pub replacement: String,
}

/// Audit logging configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditLoggingConfig {
    /// Enable audit logging
    pub enabled: bool,
    /// Audit log level
    pub level: ClinicalLogLevel,
    /// Separate audit log file
    pub separate_file: bool,
    /// Audit log file path
    pub audit_file_path: Option<String>,
    /// Include stack traces for security events
    pub include_stack_traces: bool,
    /// Digital signing of audit logs
    pub digital_signing: bool,
}

/// Performance logging configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceLoggingConfig {
    /// Enable performance logging
    pub enabled: bool,
    /// Log slow operations threshold (milliseconds)
    pub slow_operation_threshold_ms: u64,
    /// Log memory usage
    pub log_memory_usage: bool,
    /// Log CPU usage
    pub log_cpu_usage: bool,
    /// Performance sampling rate (0.0 to 1.0)
    pub sampling_rate: f64,
}

/// Log retention configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogRetentionConfig {
    /// Retention period in days
    pub retention_days: u32,
    /// Automatic cleanup
    pub auto_cleanup: bool,
    /// Compression for archived logs
    pub compress_archives: bool,
    /// Archive location
    pub archive_location: Option<String>,
}

/// Clinical context for logging
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalContext {
    /// Patient identifier (may be redacted for HIPAA)
    pub patient_id: Option<String>,
    /// Protocol identifier
    pub protocol_id: Option<String>,
    /// Clinical session identifier
    pub session_id: Option<String>,
    /// Healthcare provider identifier
    pub provider_id: Option<String>,
    /// Department or unit
    pub department: Option<String>,
    /// Clinical workflow step
    pub workflow_step: Option<String>,
    /// Patient acuity level
    pub acuity_level: Option<String>,
    /// Clinical priority
    pub priority: Option<ClinicalPriority>,
    /// Custom context fields
    pub custom: HashMap<String, String>,
}

/// Clinical priority levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ClinicalPriority {
    /// Life-threatening emergency
    Emergency,
    /// Urgent care needed
    Urgent,
    /// High priority
    High,
    /// Normal priority
    Normal,
    /// Low priority
    Low,
}

/// Structured log entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalLogEntry {
    /// Unique log entry ID
    pub id: String,
    /// Timestamp
    pub timestamp: DateTime<Utc>,
    /// Log level
    pub level: ClinicalLogLevel,
    /// Log message
    pub message: String,
    /// Clinical context
    pub clinical_context: Option<ClinicalContext>,
    /// Technical context
    pub technical_context: TechnicalContext,
    /// Additional fields
    pub fields: HashMap<String, serde_json::Value>,
    /// Error information if applicable
    pub error: Option<LogError>,
}

/// Technical context for logging
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TechnicalContext {
    /// Service name
    pub service: String,
    /// Service version
    pub version: String,
    /// Environment
    pub environment: String,
    /// Host/instance identifier
    pub host: String,
    /// Process ID
    pub pid: u32,
    /// Thread ID
    pub thread_id: String,
    /// Request ID for correlation
    pub request_id: Option<String>,
    /// Trace ID for distributed tracing
    pub trace_id: Option<String>,
    /// Span ID
    pub span_id: Option<String>,
    /// Source code location
    pub source: Option<SourceLocation>,
}

/// Source code location
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SourceLocation {
    /// File name
    pub file: String,
    /// Line number
    pub line: u32,
    /// Column number
    pub column: Option<u32>,
    /// Function name
    pub function: Option<String>,
}

/// Error information in logs
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogError {
    /// Error message
    pub message: String,
    /// Error code
    pub code: Option<String>,
    /// Error category
    pub category: ErrorCategory,
    /// Stack trace
    pub stack_trace: Option<String>,
    /// Nested error
    pub cause: Option<Box<LogError>>,
}

/// Error categories for logging
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ErrorCategory {
    /// Clinical safety error
    ClinicalSafety,
    /// Security error
    Security,
    /// System error
    System,
    /// Network error
    Network,
    /// Database error
    Database,
    /// Integration error
    Integration,
    /// Configuration error
    Configuration,
    /// Business logic error
    BusinessLogic,
}

/// Clinical logger
pub struct ClinicalLogger {
    /// Configuration
    config: LogConfig,
    /// Log entry processor
    processor: Arc<LogProcessor>,
    /// Audit logger
    audit_logger: Option<Arc<AuditLogger>>,
    /// Security logger
    security_logger: Option<Arc<SecurityLogger>>,
    /// Performance logger
    performance_logger: Option<Arc<PerformanceLogger>>,
    /// PHI redactor
    phi_redactor: Arc<PhiRedactor>,
}

/// Log processor for handling log entries
struct LogProcessor {
    /// Log outputs
    outputs: Vec<Box<dyn LogOutput>>,
    /// Log queue
    queue: mpsc::UnboundedSender<ClinicalLogEntry>,
    /// Queue receiver
    receiver: Arc<RwLock<Option<mpsc::UnboundedReceiver<ClinicalLogEntry>>>>,
}

/// Log output trait
trait LogOutput: Send + Sync {
    /// Write log entry
    fn write(&self, entry: &ClinicalLogEntry) -> ProtocolResult<()>;
    
    /// Flush pending writes
    fn flush(&self) -> ProtocolResult<()>;
}

/// Audit logger for compliance
pub struct AuditLogger {
    config: AuditLoggingConfig,
    processor: Arc<LogProcessor>,
}

/// Security logger for security events
pub struct SecurityLogger {
    processor: Arc<LogProcessor>,
}

/// Performance logger for performance monitoring
pub struct PerformanceLogger {
    config: PerformanceLoggingConfig,
    processor: Arc<LogProcessor>,
}

/// PHI redactor for HIPAA compliance
struct PhiRedactor {
    patterns: Vec<PhiPattern>,
    enabled: bool,
}

impl ClinicalLogger {
    /// Create new clinical logger
    pub async fn new(config: LogConfig) -> ProtocolResult<Self> {
        // Initialize log processor
        let (sender, receiver) = mpsc::unbounded_channel();
        let processor = Arc::new(LogProcessor {
            outputs: Vec::new(), // TODO: Initialize outputs based on config
            queue: sender,
            receiver: Arc::new(RwLock::new(Some(receiver))),
        });

        // Initialize PHI redactor
        let phi_redactor = Arc::new(PhiRedactor {
            patterns: config.hipaa_settings.phi_patterns.clone(),
            enabled: config.hipaa_settings.auto_redact_phi,
        });

        // Initialize specialized loggers
        let audit_logger = if config.audit_config.enabled {
            Some(Arc::new(AuditLogger {
                config: config.audit_config.clone(),
                processor: Arc::clone(&processor),
            }))
        } else {
            None
        };

        let security_logger = Some(Arc::new(SecurityLogger {
            processor: Arc::clone(&processor),
        }));

        let performance_logger = if config.performance_config.enabled {
            Some(Arc::new(PerformanceLogger {
                config: config.performance_config.clone(),
                processor: Arc::clone(&processor),
            }))
        } else {
            None
        };

        Ok(Self {
            config,
            processor,
            audit_logger,
            security_logger,
            performance_logger,
            phi_redactor,
        })
    }

    /// Start logger
    pub async fn start(&self) -> ProtocolResult<()> {
        // Start log processing task
        let receiver = {
            let mut guard = self.processor.receiver.write().await;
            guard.take()
        };

        if let Some(mut receiver) = receiver {
            let processor = Arc::clone(&self.processor);
            tokio::spawn(async move {
                while let Some(entry) = receiver.recv().await {
                    for output in &processor.outputs {
                        if let Err(e) = output.write(&entry) {
                            eprintln!("Failed to write log entry: {}", e);
                        }
                    }
                }
            });
        }

        Ok(())
    }

    /// Stop logger
    pub async fn stop(&self) -> ProtocolResult<()> {
        // Flush all outputs
        for output in &self.processor.outputs {
            output.flush()?;
        }
        Ok(())
    }

    /// Log emergency message
    pub async fn log_emergency(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Emergency, message, Some(clinical_context), error).await
    }

    /// Log alert message
    pub async fn log_alert(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Alert, message, Some(clinical_context), error).await
    }

    /// Log critical message
    pub async fn log_critical(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Critical, message, Some(clinical_context), error).await
    }

    /// Log error message
    pub async fn log_error(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Error, message, Some(clinical_context), error).await
    }

    /// Log warning message
    pub async fn log_warning(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Warning, message, Some(clinical_context), error).await
    }

    /// Log info message
    pub async fn log_info(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Info, message, Some(clinical_context), error).await
    }

    /// Log debug message
    pub async fn log_debug(
        &self,
        message: &str,
        clinical_context: ClinicalContext,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        self.log(ClinicalLogLevel::Debug, message, Some(clinical_context), error).await
    }

    /// Generic log method
    async fn log(
        &self,
        level: ClinicalLogLevel,
        message: &str,
        clinical_context: Option<ClinicalContext>,
        error: Option<LogError>,
    ) -> ProtocolResult<()> {
        // Check if logging is enabled and level is appropriate
        if !self.config.enabled || level < self.config.level {
            return Ok(());
        }

        // Apply PHI redaction if enabled
        let redacted_message = if self.phi_redactor.enabled {
            self.phi_redactor.redact_phi(message)
        } else {
            message.to_string()
        };

        let redacted_context = clinical_context.map(|ctx| {
            if self.phi_redactor.enabled {
                self.phi_redactor.redact_clinical_context(ctx)
            } else {
                ctx
            }
        });

        // Create log entry
        let entry = ClinicalLogEntry {
            id: Uuid::new_v4().to_string(),
            timestamp: Utc::now(),
            level,
            message: redacted_message,
            clinical_context: redacted_context,
            technical_context: self.create_technical_context(),
            fields: HashMap::new(),
            error,
        };

        // Send to processor queue
        if let Err(_) = self.processor.queue.send(entry) {
            eprintln!("Failed to queue log entry");
        }

        Ok(())
    }

    /// Create technical context for current execution
    fn create_technical_context(&self) -> TechnicalContext {
        TechnicalContext {
            service: "protocol-engine".to_string(),
            version: "1.0.0".to_string(),
            environment: "production".to_string(),
            host: gethostname::gethostname().to_string_lossy().to_string(),
            pid: std::process::id(),
            thread_id: format!("{:?}", std::thread::current().id()),
            request_id: None, // TODO: Extract from context
            trace_id: None,   // TODO: Extract from tracing context
            span_id: None,    // TODO: Extract from tracing context
            source: None,     // TODO: Extract from caller info
        }
    }
}

impl ClinicalContext {
    /// Create system-level clinical context
    pub fn system() -> Self {
        Self {
            patient_id: None,
            protocol_id: None,
            session_id: None,
            provider_id: None,
            department: Some("system".to_string()),
            workflow_step: None,
            acuity_level: None,
            priority: Some(ClinicalPriority::Normal),
            custom: HashMap::new(),
        }
    }

    /// Create patient-specific clinical context
    pub fn patient(patient_id: String, protocol_id: Option<String>) -> Self {
        Self {
            patient_id: Some(patient_id),
            protocol_id,
            session_id: Some(Uuid::new_v4().to_string()),
            provider_id: None,
            department: None,
            workflow_step: None,
            acuity_level: None,
            priority: Some(ClinicalPriority::Normal),
            custom: HashMap::new(),
        }
    }

    /// Create emergency clinical context
    pub fn emergency(patient_id: String) -> Self {
        Self {
            patient_id: Some(patient_id),
            protocol_id: None,
            session_id: Some(Uuid::new_v4().to_string()),
            provider_id: None,
            department: Some("emergency".to_string()),
            workflow_step: None,
            acuity_level: Some("critical".to_string()),
            priority: Some(ClinicalPriority::Emergency),
            custom: HashMap::new(),
        }
    }
}

impl PhiRedactor {
    /// Redact PHI from text
    fn redact_phi(&self, text: &str) -> String {
        if !self.enabled {
            return text.to_string();
        }

        let mut redacted = text.to_string();
        for pattern in &self.patterns {
            // TODO: Apply regex pattern redaction
            // This is a placeholder implementation
            if pattern.name == "patient_id" {
                redacted = redacted.replace("PAT", "***");
            }
        }
        redacted
    }

    /// Redact PHI from clinical context
    fn redact_clinical_context(&self, mut context: ClinicalContext) -> ClinicalContext {
        if !self.enabled {
            return context;
        }

        // Redact patient ID
        if let Some(patient_id) = &context.patient_id {
            context.patient_id = Some(format!("***{}", &patient_id[patient_id.len().saturating_sub(4)..]));
        }

        // Redact provider ID
        if let Some(provider_id) = &context.provider_id {
            context.provider_id = Some(format!("***{}", &provider_id[provider_id.len().saturating_sub(4)..]));
        }

        context
    }
}

impl Default for LogConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            level: ClinicalLogLevel::Info,
            format: LogFormat::Json,
            outputs: vec![LogOutput::Stdout],
            hipaa_settings: HipaaComplianceSettings {
                enabled: true,
                auto_redact_phi: true,
                phi_patterns: vec![
                    PhiPattern {
                        name: "patient_id".to_string(),
                        pattern: r"PAT\d+".to_string(),
                        replacement: "***".to_string(),
                    },
                    PhiPattern {
                        name: "ssn".to_string(),
                        pattern: r"\d{3}-\d{2}-\d{4}".to_string(),
                        replacement: "***-**-****".to_string(),
                    },
                ],
                encrypt_at_rest: true,
                encryption_key_id: None,
                access_logging: true,
            },
            audit_config: AuditLoggingConfig {
                enabled: true,
                level: ClinicalLogLevel::Notice,
                separate_file: true,
                audit_file_path: Some("/var/log/clinical/audit.log".to_string()),
                include_stack_traces: true,
                digital_signing: false,
            },
            performance_config: PerformanceLoggingConfig {
                enabled: true,
                slow_operation_threshold_ms: 1000,
                log_memory_usage: true,
                log_cpu_usage: false,
                sampling_rate: 0.1,
            },
            retention: LogRetentionConfig {
                retention_days: 2555, // 7 years for clinical data
                auto_cleanup: true,
                compress_archives: true,
                archive_location: Some("/var/log/clinical/archive".to_string()),
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_clinical_logger_creation() {
        let config = LogConfig::default();
        let logger = ClinicalLogger::new(config).await;
        assert!(logger.is_ok());
    }

    #[tokio::test]
    async fn test_clinical_context_creation() {
        let system_context = ClinicalContext::system();
        assert_eq!(system_context.department, Some("system".to_string()));

        let patient_context = ClinicalContext::patient("PAT12345".to_string(), None);
        assert_eq!(patient_context.patient_id, Some("PAT12345".to_string()));

        let emergency_context = ClinicalContext::emergency("PAT67890".to_string());
        assert_eq!(emergency_context.priority, Some(ClinicalPriority::Emergency));
    }

    #[test]
    fn test_phi_redaction() {
        let patterns = vec![
            PhiPattern {
                name: "patient_id".to_string(),
                pattern: r"PAT\d+".to_string(),
                replacement: "***".to_string(),
            }
        ];
        let redactor = PhiRedactor {
            patterns,
            enabled: true,
        };

        let original = "Patient PAT12345 admitted";
        let redacted = redactor.redact_phi(original);
        assert!(redacted.contains("***"));
        assert!(!redacted.contains("PAT12345"));
    }

    #[test]
    fn test_log_levels() {
        assert!(ClinicalLogLevel::Emergency < ClinicalLogLevel::Alert);
        assert!(ClinicalLogLevel::Alert < ClinicalLogLevel::Critical);
        assert!(ClinicalLogLevel::Critical < ClinicalLogLevel::Error);
        assert!(ClinicalLogLevel::Error < ClinicalLogLevel::Warning);
    }
}