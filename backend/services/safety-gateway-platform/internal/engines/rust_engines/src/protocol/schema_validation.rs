//! Schema Validation and Serialization
//!
//! This module provides comprehensive schema validation and serialization
//! capabilities for protocol events, messages, and data structures to ensure
//! data integrity and compatibility across service boundaries.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use serde_json::{Value, Map};
use chrono::{DateTime, Utc};
use tokio::sync::RwLock;
use uuid::Uuid;

use crate::protocol::{
    types::*,
    error::*,
    event_publisher::{ProtocolEvent, ProtocolEventType},
    message_router::{ServiceMessage, ServiceMessageType},
    approval_workflow::{ApprovalRequest, ApprovalStatus},
    cae_integration::{CaeEvaluationRequest, CaeEvaluationResponse},
};

/// Schema validation configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaValidationConfig {
    /// Enable schema validation
    pub enabled: bool,
    /// Schema registry configuration
    pub registry_config: SchemaRegistryConfig,
    /// Validation rules
    pub validation_rules: ValidationRulesConfig,
    /// Serialization configuration
    pub serialization_config: SerializationConfig,
    /// Schema evolution settings
    pub evolution_config: SchemaEvolutionConfig,
}

/// Schema registry configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaRegistryConfig {
    /// Registry type
    pub registry_type: SchemaRegistryType,
    /// Registry endpoint URL
    pub endpoint_url: Option<String>,
    /// Authentication configuration
    pub auth_config: Option<RegistryAuthConfig>,
    /// Local schema cache settings
    pub cache_config: SchemaCacheConfig,
    /// Schema loading behavior
    pub loading_behavior: SchemaLoadingBehavior,
}

/// Schema registry types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SchemaRegistryType {
    /// In-memory schema registry
    InMemory,
    /// File-based schema registry
    FileSystem(String),
    /// Confluent Schema Registry
    Confluent,
    /// Custom schema registry
    Custom(String),
    /// Azure Schema Registry
    Azure,
    /// AWS Glue Schema Registry
    AwsGlue,
}

/// Registry authentication configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegistryAuthConfig {
    /// Authentication method
    pub method: RegistryAuthMethod,
    /// Username
    pub username: Option<String>,
    /// Password or token
    pub password: Option<String>,
    /// API key
    pub api_key: Option<String>,
}

/// Registry authentication methods
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RegistryAuthMethod {
    /// No authentication
    None,
    /// Basic authentication
    Basic,
    /// Bearer token
    Bearer,
    /// API key authentication
    ApiKey,
    /// OAuth 2.0
    OAuth2,
}

/// Schema cache configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaCacheConfig {
    /// Enable caching
    pub enabled: bool,
    /// Maximum cache size
    pub max_size: usize,
    /// Cache TTL in seconds
    pub ttl_seconds: u64,
    /// Cache refresh interval
    pub refresh_interval_seconds: u64,
}

/// Schema loading behavior
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SchemaLoadingBehavior {
    /// Load all schemas on startup
    Eager,
    /// Load schemas on demand
    Lazy,
    /// Load schemas with background refresh
    Background,
}

/// Validation rules configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationRulesConfig {
    /// Strict validation mode
    pub strict_mode: bool,
    /// Allow unknown fields
    pub allow_unknown_fields: bool,
    /// Custom validation rules
    pub custom_rules: HashMap<String, ValidationRule>,
    /// Field-level validation
    pub field_validations: HashMap<String, FieldValidation>,
    /// Cross-field validation rules
    pub cross_field_rules: Vec<CrossFieldRule>,
}

/// Custom validation rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationRule {
    /// Rule name
    pub name: String,
    /// Rule description
    pub description: String,
    /// Rule type
    pub rule_type: ValidationRuleType,
    /// Rule parameters
    pub parameters: HashMap<String, Value>,
    /// Error message template
    pub error_message: String,
}

/// Validation rule types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ValidationRuleType {
    /// Format validation (e.g., email, phone)
    Format,
    /// Range validation (min/max values)
    Range,
    /// Length validation (string/array length)
    Length,
    /// Pattern matching (regex)
    Pattern,
    /// Enumeration validation
    Enum,
    /// Custom function validation
    Custom,
    /// Clinical-specific validations
    Clinical,
}

/// Field-level validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FieldValidation {
    /// Field path (dot notation)
    pub field_path: String,
    /// Required field
    pub required: bool,
    /// Data type validation
    pub data_type: Option<DataType>,
    /// Value constraints
    pub constraints: Vec<FieldConstraint>,
    /// Custom validators
    pub validators: Vec<String>,
}

/// Data types for validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DataType {
    String,
    Number,
    Integer,
    Boolean,
    Array,
    Object,
    DateTime,
    UUID,
    Email,
    Phone,
    URL,
    /// Clinical-specific types
    PatientId,
    ProtocolId,
    MedicationCode,
    DiagnosisCode,
    /// FHIR resource types
    FhirResource(String),
}

/// Field constraint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FieldConstraint {
    /// Constraint type
    pub constraint_type: ConstraintType,
    /// Constraint value
    pub value: Value,
    /// Error message
    pub error_message: Option<String>,
}

/// Constraint types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConstraintType {
    /// Minimum value/length
    MinValue,
    /// Maximum value/length
    MaxValue,
    /// Exact value/length
    ExactValue,
    /// Pattern match
    Pattern,
    /// Enumeration
    Enum,
    /// Not null
    NotNull,
    /// Unique value
    Unique,
    /// Range validation
    Range,
}

/// Cross-field validation rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossFieldRule {
    /// Rule name
    pub name: String,
    /// Fields involved in validation
    pub fields: Vec<String>,
    /// Validation logic
    pub logic: CrossFieldLogic,
    /// Error message
    pub error_message: String,
}

/// Cross-field validation logic
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CrossFieldLogic {
    /// All fields must be present together
    AllOrNone,
    /// Only one field should be present
    OneOf,
    /// Field A requires field B
    Requires,
    /// Fields are mutually exclusive
    MutuallyExclusive,
    /// Custom validation function
    Custom(String),
}

/// Serialization configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SerializationConfig {
    /// Default serialization format
    pub default_format: SerializationFormat,
    /// Pretty print JSON
    pub pretty_print: bool,
    /// Include null fields
    pub include_nulls: bool,
    /// Date/time format
    pub datetime_format: DateTimeFormat,
    /// Custom serializers
    pub custom_serializers: HashMap<String, String>,
    /// Compression settings
    pub compression: CompressionConfig,
}

/// Serialization formats
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SerializationFormat {
    /// JSON format
    Json,
    /// YAML format
    Yaml,
    /// XML format
    Xml,
    /// MessagePack format
    MessagePack,
    /// Protocol Buffers
    Protobuf,
    /// Apache Avro
    Avro,
}

/// Date/time format options
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DateTimeFormat {
    /// RFC 3339 format
    Rfc3339,
    /// ISO 8601 format
    Iso8601,
    /// Unix timestamp
    Unix,
    /// Custom format string
    Custom(String),
}

/// Compression configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompressionConfig {
    /// Enable compression
    pub enabled: bool,
    /// Compression algorithm
    pub algorithm: CompressionAlgorithm,
    /// Compression level (1-9)
    pub level: u8,
    /// Minimum size threshold for compression
    pub min_size_bytes: usize,
}

/// Compression algorithms
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CompressionAlgorithm {
    /// Gzip compression
    Gzip,
    /// Zlib compression
    Zlib,
    /// LZ4 compression
    Lz4,
    /// Snappy compression
    Snappy,
}

/// Schema evolution configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaEvolutionConfig {
    /// Evolution strategy
    pub strategy: EvolutionStrategy,
    /// Compatibility rules
    pub compatibility: CompatibilityMode,
    /// Version management
    pub versioning: VersioningStrategy,
    /// Migration rules
    pub migration_rules: Vec<MigrationRule>,
}

/// Schema evolution strategies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvolutionStrategy {
    /// Strict compatibility required
    Strict,
    /// Forward compatibility only
    Forward,
    /// Backward compatibility only
    Backward,
    /// Full compatibility (forward + backward)
    Full,
    /// No compatibility checks
    None,
}

/// Schema compatibility modes
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CompatibilityMode {
    /// Backward compatible
    Backward,
    /// Forward compatible
    Forward,
    /// Full compatibility
    Full,
    /// No compatibility required
    None,
}

/// Versioning strategies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VersioningStrategy {
    /// Semantic versioning (major.minor.patch)
    Semantic,
    /// Integer versioning (1, 2, 3, ...)
    Integer,
    /// Date-based versioning
    Date,
    /// Git commit hash
    GitHash,
    /// Custom versioning scheme
    Custom(String),
}

/// Schema migration rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationRule {
    /// Source schema version
    pub from_version: String,
    /// Target schema version
    pub to_version: String,
    /// Migration type
    pub migration_type: MigrationType,
    /// Migration instructions
    pub instructions: Vec<MigrationInstruction>,
}

/// Migration types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MigrationType {
    /// Automatic migration
    Automatic,
    /// Manual migration required
    Manual,
    /// Gradual migration
    Gradual,
    /// Breaking change
    Breaking,
}

/// Migration instruction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationInstruction {
    /// Instruction type
    pub instruction_type: MigrationInstructionType,
    /// Field path
    pub field_path: String,
    /// Instruction parameters
    pub parameters: HashMap<String, Value>,
}

/// Migration instruction types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MigrationInstructionType {
    /// Add field
    AddField,
    /// Remove field
    RemoveField,
    /// Rename field
    RenameField,
    /// Change field type
    ChangeType,
    /// Set default value
    SetDefault,
    /// Transform value
    Transform,
}

/// Schema definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaDefinition {
    /// Schema identifier
    pub schema_id: String,
    /// Schema name
    pub name: String,
    /// Schema version
    pub version: String,
    /// Schema description
    pub description: String,
    /// Schema type (event, message, etc.)
    pub schema_type: SchemaType,
    /// JSON schema definition
    pub schema: Value,
    /// Schema metadata
    pub metadata: SchemaMetadata,
    /// Creation timestamp
    pub created_at: DateTime<Utc>,
    /// Last modified timestamp
    pub modified_at: DateTime<Utc>,
}

/// Schema types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SchemaType {
    /// Protocol event schema
    ProtocolEvent,
    /// Service message schema
    ServiceMessage,
    /// Approval request schema
    ApprovalRequest,
    /// CAE request/response schema
    CaeRequest,
    /// Custom schema type
    Custom(String),
}

/// Schema metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaMetadata {
    /// Schema author
    pub author: String,
    /// Schema tags
    pub tags: Vec<String>,
    /// Schema category
    pub category: String,
    /// Usage documentation
    pub documentation_url: Option<String>,
    /// Example data
    pub examples: Vec<Value>,
    /// Related schemas
    pub related_schemas: Vec<String>,
}

/// Validation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    /// Validation success
    pub valid: bool,
    /// Schema version used
    pub schema_version: String,
    /// Validation errors
    pub errors: Vec<ValidationError>,
    /// Validation warnings
    pub warnings: Vec<ValidationWarning>,
    /// Validation metadata
    pub metadata: ValidationMetadata,
}

/// Validation error
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationError {
    /// Error code
    pub code: String,
    /// Error message
    pub message: String,
    /// Field path where error occurred
    pub field_path: Option<String>,
    /// Error severity
    pub severity: ValidationSeverity,
    /// Suggested fix
    pub suggested_fix: Option<String>,
}

/// Validation warning
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationWarning {
    /// Warning code
    pub code: String,
    /// Warning message
    pub message: String,
    /// Field path where warning occurred
    pub field_path: Option<String>,
    /// Warning category
    pub category: WarningCategory,
}

/// Validation severity levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ValidationSeverity {
    /// Critical error - blocks processing
    Critical,
    /// Error - should be fixed
    Error,
    /// Warning - should be reviewed
    Warning,
    /// Info - informational only
    Info,
}

/// Warning categories
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum WarningCategory {
    /// Deprecated field usage
    Deprecated,
    /// Performance concern
    Performance,
    /// Best practice violation
    BestPractice,
    /// Schema evolution concern
    Evolution,
    /// Clinical safety concern
    ClinicalSafety,
}

/// Validation metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationMetadata {
    /// Validation timestamp
    pub timestamp: DateTime<Utc>,
    /// Validator version
    pub validator_version: String,
    /// Validation duration
    pub duration_ms: u64,
    /// Schema cache hit
    pub cache_hit: bool,
}

/// Schema validator engine
pub struct SchemaValidator {
    /// Configuration
    config: SchemaValidationConfig,
    /// Schema registry
    schema_registry: Arc<RwLock<HashMap<String, SchemaDefinition>>>,
    /// Validation cache
    validation_cache: Arc<RwLock<HashMap<String, ValidationResult>>>,
    /// Custom validators
    custom_validators: HashMap<String, Box<dyn CustomValidator>>,
    /// Validation metrics
    metrics: Arc<ValidationMetrics>,
}

/// Custom validator trait
pub trait CustomValidator: Send + Sync {
    /// Validate data against custom rules
    fn validate(&self, data: &Value, context: &ValidationContext) -> ValidationResult;
    
    /// Get validator name
    fn name(&self) -> &str;
    
    /// Get validator description
    fn description(&self) -> &str;
}

/// Validation context
#[derive(Debug, Clone)]
pub struct ValidationContext {
    /// Schema being validated against
    pub schema: SchemaDefinition,
    /// Validation configuration
    pub config: SchemaValidationConfig,
    /// Additional context data
    pub context_data: HashMap<String, Value>,
}

/// Validation metrics
#[derive(Debug, Default)]
pub struct ValidationMetrics {
    /// Total validations performed
    pub total_validations: std::sync::atomic::AtomicU64,
    /// Successful validations
    pub successful_validations: std::sync::atomic::AtomicU64,
    /// Failed validations
    pub failed_validations: std::sync::atomic::AtomicU64,
    /// Schema cache hits
    pub cache_hits: std::sync::atomic::AtomicU64,
    /// Schema cache misses
    pub cache_misses: std::sync::atomic::AtomicU64,
    /// Average validation time
    pub avg_validation_time_ms: std::sync::atomic::AtomicU64,
}

impl SchemaValidator {
    /// Create new schema validator
    pub async fn new(config: SchemaValidationConfig) -> ProtocolResult<Self> {
        let mut validator = Self {
            config: config.clone(),
            schema_registry: Arc::new(RwLock::new(HashMap::new())),
            validation_cache: Arc::new(RwLock::new(HashMap::new())),
            custom_validators: HashMap::new(),
            metrics: Arc::new(ValidationMetrics::default()),
        };

        // Load built-in schemas
        validator.load_builtin_schemas().await?;

        // Register custom validators
        validator.register_builtin_validators();

        Ok(validator)
    }

    /// Load built-in schema definitions
    async fn load_builtin_schemas(&self) -> ProtocolResult<()> {
        let mut registry = self.schema_registry.write().await;

        // Protocol Event schema
        let protocol_event_schema = self.create_protocol_event_schema();
        registry.insert("protocol-event-v1".to_string(), protocol_event_schema);

        // Service Message schema
        let service_message_schema = self.create_service_message_schema();
        registry.insert("service-message-v1".to_string(), service_message_schema);

        // Approval Request schema
        let approval_request_schema = self.create_approval_request_schema();
        registry.insert("approval-request-v1".to_string(), approval_request_schema);

        // CAE Request schema
        let cae_request_schema = self.create_cae_request_schema();
        registry.insert("cae-request-v1".to_string(), cae_request_schema);

        Ok(())
    }

    /// Register built-in custom validators
    fn register_builtin_validators(&mut self) {
        // Clinical validators
        self.register_custom_validator(Box::new(PatientIdValidator));
        self.register_custom_validator(Box::new(ProtocolIdValidator));
        self.register_custom_validator(Box::new(MedicationCodeValidator));
        self.register_custom_validator(Box::new(ClinicalTimestampValidator));
    }

    /// Register custom validator
    pub fn register_custom_validator(&mut self, validator: Box<dyn CustomValidator>) {
        self.custom_validators.insert(validator.name().to_string(), validator);
    }

    /// Validate protocol event
    pub async fn validate_protocol_event(&self, event: &ProtocolEvent) -> ValidationResult {
        let data = match serde_json::to_value(event) {
            Ok(value) => value,
            Err(e) => return ValidationResult {
                valid: false,
                schema_version: "protocol-event-v1".to_string(),
                errors: vec![ValidationError {
                    code: "SERIALIZATION_ERROR".to_string(),
                    message: format!("Failed to serialize event: {}", e),
                    field_path: None,
                    severity: ValidationSeverity::Critical,
                    suggested_fix: Some("Check event structure".to_string()),
                }],
                warnings: vec![],
                metadata: ValidationMetadata {
                    timestamp: Utc::now(),
                    validator_version: "1.0.0".to_string(),
                    duration_ms: 0,
                    cache_hit: false,
                },
            },
        };

        self.validate_data(&data, "protocol-event-v1").await
    }

    /// Validate service message
    pub async fn validate_service_message(&self, message: &ServiceMessage) -> ValidationResult {
        let data = match serde_json::to_value(message) {
            Ok(value) => value,
            Err(e) => return ValidationResult {
                valid: false,
                schema_version: "service-message-v1".to_string(),
                errors: vec![ValidationError {
                    code: "SERIALIZATION_ERROR".to_string(),
                    message: format!("Failed to serialize message: {}", e),
                    field_path: None,
                    severity: ValidationSeverity::Critical,
                    suggested_fix: Some("Check message structure".to_string()),
                }],
                warnings: vec![],
                metadata: ValidationMetadata {
                    timestamp: Utc::now(),
                    validator_version: "1.0.0".to_string(),
                    duration_ms: 0,
                    cache_hit: false,
                },
            },
        };

        self.validate_data(&data, "service-message-v1").await
    }

    /// Validate approval request
    pub async fn validate_approval_request(&self, request: &ApprovalRequest) -> ValidationResult {
        let data = match serde_json::to_value(request) {
            Ok(value) => value,
            Err(e) => return ValidationResult {
                valid: false,
                schema_version: "approval-request-v1".to_string(),
                errors: vec![ValidationError {
                    code: "SERIALIZATION_ERROR".to_string(),
                    message: format!("Failed to serialize request: {}", e),
                    field_path: None,
                    severity: ValidationSeverity::Critical,
                    suggested_fix: Some("Check request structure".to_string()),
                }],
                warnings: vec![],
                metadata: ValidationMetadata {
                    timestamp: Utc::now(),
                    validator_version: "1.0.0".to_string(),
                    duration_ms: 0,
                    cache_hit: false,
                },
            },
        };

        self.validate_data(&data, "approval-request-v1").await
    }

    /// Validate data against schema
    pub async fn validate_data(&self, data: &Value, schema_id: &str) -> ValidationResult {
        let start_time = std::time::Instant::now();
        self.metrics.total_validations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);

        // Check validation cache
        let cache_key = format!("{}-{}", schema_id, self.calculate_data_hash(data));
        if let Some(cached_result) = self.get_cached_result(&cache_key).await {
            self.metrics.cache_hits.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
            return cached_result;
        }

        self.metrics.cache_misses.fetch_add(1, std::sync::atomic::Ordering::Relaxed);

        // Get schema definition
        let schema = match self.get_schema_definition(schema_id).await {
            Some(schema) => schema,
            None => return ValidationResult {
                valid: false,
                schema_version: schema_id.to_string(),
                errors: vec![ValidationError {
                    code: "SCHEMA_NOT_FOUND".to_string(),
                    message: format!("Schema not found: {}", schema_id),
                    field_path: None,
                    severity: ValidationSeverity::Critical,
                    suggested_fix: Some("Register schema definition".to_string()),
                }],
                warnings: vec![],
                metadata: ValidationMetadata {
                    timestamp: Utc::now(),
                    validator_version: "1.0.0".to_string(),
                    duration_ms: start_time.elapsed().as_millis() as u64,
                    cache_hit: false,
                },
            },
        };

        // Perform validation
        let result = self.perform_validation(data, &schema).await;

        // Cache result
        self.cache_validation_result(&cache_key, &result).await;

        // Update metrics
        let duration_ms = start_time.elapsed().as_millis() as u64;
        self.update_validation_metrics(&result, duration_ms);

        result
    }

    /// Perform actual validation
    async fn perform_validation(&self, data: &Value, schema: &SchemaDefinition) -> ValidationResult {
        let mut errors = Vec::new();
        let mut warnings = Vec::new();

        // Basic JSON schema validation
        if let Err(validation_errors) = self.validate_json_schema(data, &schema.schema) {
            errors.extend(validation_errors);
        }

        // Custom validation rules
        if let Err(custom_errors) = self.validate_custom_rules(data, schema).await {
            errors.extend(custom_errors);
        }

        // Clinical-specific validation
        if let Err(clinical_errors) = self.validate_clinical_rules(data, schema).await {
            errors.extend(clinical_errors);
        }

        // Cross-field validation
        if let Err(cross_field_errors) = self.validate_cross_field_rules(data, schema).await {
            errors.extend(cross_field_errors);
        }

        ValidationResult {
            valid: errors.is_empty(),
            schema_version: schema.version.clone(),
            errors,
            warnings,
            metadata: ValidationMetadata {
                timestamp: Utc::now(),
                validator_version: "1.0.0".to_string(),
                duration_ms: 0, // Will be set by caller
                cache_hit: false,
            },
        }
    }

    /// Validate against JSON schema
    fn validate_json_schema(&self, _data: &Value, _schema: &Value) -> Result<(), Vec<ValidationError>> {
        // TODO: Implement JSON Schema validation
        // This would typically use a library like jsonschema-rs
        Ok(())
    }

    /// Validate custom rules
    async fn validate_custom_rules(&self, data: &Value, schema: &SchemaDefinition) -> Result<(), Vec<ValidationError>> {
        let mut errors = Vec::new();

        for (rule_name, _rule) in &self.config.validation_rules.custom_rules {
            if let Some(validator) = self.custom_validators.get(rule_name) {
                let context = ValidationContext {
                    schema: schema.clone(),
                    config: self.config.clone(),
                    context_data: HashMap::new(),
                };

                let result = validator.validate(data, &context);
                if !result.valid {
                    errors.extend(result.errors);
                }
            }
        }

        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }

    /// Validate clinical-specific rules
    async fn validate_clinical_rules(&self, _data: &Value, _schema: &SchemaDefinition) -> Result<(), Vec<ValidationError>> {
        // TODO: Implement clinical-specific validation rules
        // This would include validation for:
        // - Patient ID format
        // - Protocol ID format
        // - Medication codes
        // - Diagnosis codes
        // - Clinical timestamps
        // - FHIR resource structure
        Ok(())
    }

    /// Validate cross-field rules
    async fn validate_cross_field_rules(&self, _data: &Value, _schema: &SchemaDefinition) -> Result<(), Vec<ValidationError>> {
        // TODO: Implement cross-field validation
        // This would validate relationships between fields
        Ok(())
    }

    /// Get schema definition
    async fn get_schema_definition(&self, schema_id: &str) -> Option<SchemaDefinition> {
        let registry = self.schema_registry.read().await;
        registry.get(schema_id).cloned()
    }

    /// Get cached validation result
    async fn get_cached_result(&self, cache_key: &str) -> Option<ValidationResult> {
        let cache = self.validation_cache.read().await;
        cache.get(cache_key).cloned()
    }

    /// Cache validation result
    async fn cache_validation_result(&self, cache_key: &str, result: &ValidationResult) {
        let mut cache = self.validation_cache.write().await;
        cache.insert(cache_key.to_string(), result.clone());
        
        // Simple cache eviction (TODO: implement LRU)
        if cache.len() > 1000 {
            let keys_to_remove: Vec<String> = cache.keys().take(100).cloned().collect();
            for key in keys_to_remove {
                cache.remove(&key);
            }
        }
    }

    /// Calculate hash of data for caching
    fn calculate_data_hash(&self, data: &Value) -> String {
        use std::hash::{Hash, Hasher};
        use std::collections::hash_map::DefaultHasher;

        let data_str = serde_json::to_string(data).unwrap_or_default();
        let mut hasher = DefaultHasher::new();
        data_str.hash(&mut hasher);
        format!("{:x}", hasher.finish())
    }

    /// Update validation metrics
    fn update_validation_metrics(&self, result: &ValidationResult, duration_ms: u64) {
        if result.valid {
            self.metrics.successful_validations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        } else {
            self.metrics.failed_validations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        }

        // Update average validation time
        let current_avg = self.metrics.avg_validation_time_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_validations = self.metrics.total_validations.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_validations > 1 {
            let new_avg = (current_avg * (total_validations - 1) + duration_ms) / total_validations;
            self.metrics.avg_validation_time_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        } else {
            self.metrics.avg_validation_time_ms.store(duration_ms, std::sync::atomic::Ordering::Relaxed);
        }
    }

    /// Create protocol event schema
    fn create_protocol_event_schema(&self) -> SchemaDefinition {
        SchemaDefinition {
            schema_id: "protocol-event-v1".to_string(),
            name: "Protocol Event".to_string(),
            version: "1.0.0".to_string(),
            description: "Schema for protocol events".to_string(),
            schema_type: SchemaType::ProtocolEvent,
            schema: serde_json::json!({
                "type": "object",
                "required": ["event_id", "event_type", "patient_id", "protocol_id", "timestamp"],
                "properties": {
                    "event_id": {
                        "type": "string",
                        "format": "uuid"
                    },
                    "event_type": {
                        "type": "string"
                    },
                    "patient_id": {
                        "type": "string",
                        "minLength": 1
                    },
                    "protocol_id": {
                        "type": "string",
                        "minLength": 1
                    },
                    "timestamp": {
                        "type": "string",
                        "format": "date-time"
                    }
                }
            }),
            metadata: SchemaMetadata {
                author: "Protocol Engine".to_string(),
                tags: vec!["event".to_string(), "protocol".to_string()],
                category: "clinical".to_string(),
                documentation_url: None,
                examples: vec![],
                related_schemas: vec![],
            },
            created_at: Utc::now(),
            modified_at: Utc::now(),
        }
    }

    /// Create service message schema
    fn create_service_message_schema(&self) -> SchemaDefinition {
        SchemaDefinition {
            schema_id: "service-message-v1".to_string(),
            name: "Service Message".to_string(),
            version: "1.0.0".to_string(),
            description: "Schema for cross-service messages".to_string(),
            schema_type: SchemaType::ServiceMessage,
            schema: serde_json::json!({
                "type": "object",
                "required": ["message_id", "message_type", "source_service", "target_service", "timestamp"],
                "properties": {
                    "message_id": {
                        "type": "string",
                        "format": "uuid"
                    },
                    "message_type": {
                        "type": "string"
                    },
                    "source_service": {
                        "type": "string",
                        "minLength": 1
                    },
                    "target_service": {
                        "type": "string",
                        "minLength": 1
                    },
                    "timestamp": {
                        "type": "string",
                        "format": "date-time"
                    }
                }
            }),
            metadata: SchemaMetadata {
                author: "Message Router".to_string(),
                tags: vec!["message".to_string(), "service".to_string()],
                category: "communication".to_string(),
                documentation_url: None,
                examples: vec![],
                related_schemas: vec![],
            },
            created_at: Utc::now(),
            modified_at: Utc::now(),
        }
    }

    /// Create approval request schema
    fn create_approval_request_schema(&self) -> SchemaDefinition {
        SchemaDefinition {
            schema_id: "approval-request-v1".to_string(),
            name: "Approval Request".to_string(),
            version: "1.0.0".to_string(),
            description: "Schema for approval requests".to_string(),
            schema_type: SchemaType::ApprovalRequest,
            schema: serde_json::json!({
                "type": "object",
                "required": ["request_id", "approval_type", "patient_id", "protocol_id", "requester"],
                "properties": {
                    "request_id": {
                        "type": "string",
                        "format": "uuid"
                    },
                    "approval_type": {
                        "type": "string"
                    },
                    "patient_id": {
                        "type": "string",
                        "minLength": 1
                    },
                    "protocol_id": {
                        "type": "string",
                        "minLength": 1
                    },
                    "requester": {
                        "type": "object",
                        "required": ["user_id", "name", "role"],
                        "properties": {
                            "user_id": {"type": "string"},
                            "name": {"type": "string"},
                            "role": {"type": "string"}
                        }
                    }
                }
            }),
            metadata: SchemaMetadata {
                author: "Approval Workflow".to_string(),
                tags: vec!["approval".to_string(), "workflow".to_string()],
                category: "governance".to_string(),
                documentation_url: None,
                examples: vec![],
                related_schemas: vec![],
            },
            created_at: Utc::now(),
            modified_at: Utc::now(),
        }
    }

    /// Create CAE request schema
    fn create_cae_request_schema(&self) -> SchemaDefinition {
        SchemaDefinition {
            schema_id: "cae-request-v1".to_string(),
            name: "CAE Request".to_string(),
            version: "1.0.0".to_string(),
            description: "Schema for CAE evaluation requests".to_string(),
            schema_type: SchemaType::CaeRequest,
            schema: serde_json::json!({
                "type": "object",
                "required": ["request_id", "patient_id", "protocol_id", "clinical_context"],
                "properties": {
                    "request_id": {
                        "type": "string",
                        "format": "uuid"
                    },
                    "patient_id": {
                        "type": "string",
                        "minLength": 1
                    },
                    "protocol_id": {
                        "type": "string",
                        "minLength": 1
                    },
                    "clinical_context": {
                        "type": "object"
                    }
                }
            }),
            metadata: SchemaMetadata {
                author: "CAE Integration".to_string(),
                tags: vec!["cae".to_string(), "assessment".to_string()],
                category: "clinical".to_string(),
                documentation_url: None,
                examples: vec![],
                related_schemas: vec![],
            },
            created_at: Utc::now(),
            modified_at: Utc::now(),
        }
    }

    /// Get validation metrics
    pub fn get_metrics(&self) -> ValidationMetricsSnapshot {
        ValidationMetricsSnapshot {
            total_validations: self.metrics.total_validations.load(std::sync::atomic::Ordering::Relaxed),
            successful_validations: self.metrics.successful_validations.load(std::sync::atomic::Ordering::Relaxed),
            failed_validations: self.metrics.failed_validations.load(std::sync::atomic::Ordering::Relaxed),
            cache_hits: self.metrics.cache_hits.load(std::sync::atomic::Ordering::Relaxed),
            cache_misses: self.metrics.cache_misses.load(std::sync::atomic::Ordering::Relaxed),
            avg_validation_time_ms: self.metrics.avg_validation_time_ms.load(std::sync::atomic::Ordering::Relaxed),
        }
    }
}

/// Validation metrics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationMetricsSnapshot {
    pub total_validations: u64,
    pub successful_validations: u64,
    pub failed_validations: u64,
    pub cache_hits: u64,
    pub cache_misses: u64,
    pub avg_validation_time_ms: u64,
}

/// Built-in custom validators

/// Patient ID validator
pub struct PatientIdValidator;

impl CustomValidator for PatientIdValidator {
    fn validate(&self, data: &Value, _context: &ValidationContext) -> ValidationResult {
        let mut errors = Vec::new();
        
        if let Some(patient_id) = data.get("patient_id").and_then(|v| v.as_str()) {
            if patient_id.is_empty() {
                errors.push(ValidationError {
                    code: "PATIENT_ID_EMPTY".to_string(),
                    message: "Patient ID cannot be empty".to_string(),
                    field_path: Some("patient_id".to_string()),
                    severity: ValidationSeverity::Error,
                    suggested_fix: Some("Provide valid patient ID".to_string()),
                });
            } else if patient_id.len() < 3 {
                errors.push(ValidationError {
                    code: "PATIENT_ID_TOO_SHORT".to_string(),
                    message: "Patient ID too short".to_string(),
                    field_path: Some("patient_id".to_string()),
                    severity: ValidationSeverity::Error,
                    suggested_fix: Some("Patient ID should be at least 3 characters".to_string()),
                });
            }
        }

        ValidationResult {
            valid: errors.is_empty(),
            schema_version: "patient-id-validator-v1".to_string(),
            errors,
            warnings: vec![],
            metadata: ValidationMetadata {
                timestamp: Utc::now(),
                validator_version: "1.0.0".to_string(),
                duration_ms: 1,
                cache_hit: false,
            },
        }
    }

    fn name(&self) -> &str {
        "patient-id-validator"
    }

    fn description(&self) -> &str {
        "Validates patient ID format and constraints"
    }
}

/// Protocol ID validator
pub struct ProtocolIdValidator;

impl CustomValidator for ProtocolIdValidator {
    fn validate(&self, data: &Value, _context: &ValidationContext) -> ValidationResult {
        let mut errors = Vec::new();
        
        if let Some(protocol_id) = data.get("protocol_id").and_then(|v| v.as_str()) {
            if protocol_id.is_empty() {
                errors.push(ValidationError {
                    code: "PROTOCOL_ID_EMPTY".to_string(),
                    message: "Protocol ID cannot be empty".to_string(),
                    field_path: Some("protocol_id".to_string()),
                    severity: ValidationSeverity::Error,
                    suggested_fix: Some("Provide valid protocol ID".to_string()),
                });
            } else if !protocol_id.contains('-') {
                errors.push(ValidationError {
                    code: "PROTOCOL_ID_INVALID_FORMAT".to_string(),
                    message: "Protocol ID should contain version separator".to_string(),
                    field_path: Some("protocol_id".to_string()),
                    severity: ValidationSeverity::Warning,
                    suggested_fix: Some("Use format: protocol-name-version".to_string()),
                });
            }
        }

        ValidationResult {
            valid: errors.is_empty(),
            schema_version: "protocol-id-validator-v1".to_string(),
            errors,
            warnings: vec![],
            metadata: ValidationMetadata {
                timestamp: Utc::now(),
                validator_version: "1.0.0".to_string(),
                duration_ms: 1,
                cache_hit: false,
            },
        }
    }

    fn name(&self) -> &str {
        "protocol-id-validator"
    }

    fn description(&self) -> &str {
        "Validates protocol ID format and constraints"
    }
}

/// Medication code validator
pub struct MedicationCodeValidator;

impl CustomValidator for MedicationCodeValidator {
    fn validate(&self, _data: &Value, _context: &ValidationContext) -> ValidationResult {
        // TODO: Implement medication code validation
        ValidationResult {
            valid: true,
            schema_version: "medication-code-validator-v1".to_string(),
            errors: vec![],
            warnings: vec![],
            metadata: ValidationMetadata {
                timestamp: Utc::now(),
                validator_version: "1.0.0".to_string(),
                duration_ms: 1,
                cache_hit: false,
            },
        }
    }

    fn name(&self) -> &str {
        "medication-code-validator"
    }

    fn description(&self) -> &str {
        "Validates medication codes against standard vocabularies"
    }
}

/// Clinical timestamp validator
pub struct ClinicalTimestampValidator;

impl CustomValidator for ClinicalTimestampValidator {
    fn validate(&self, data: &Value, _context: &ValidationContext) -> ValidationResult {
        let mut errors = Vec::new();
        let mut warnings = Vec::new();
        
        if let Some(timestamp_str) = data.get("timestamp").and_then(|v| v.as_str()) {
            if let Ok(timestamp) = DateTime::parse_from_rfc3339(timestamp_str) {
                let now = Utc::now();
                let timestamp_utc = timestamp.with_timezone(&Utc);
                
                // Check if timestamp is too far in the future
                if timestamp_utc > now + Duration::hours(1) {
                    warnings.push(ValidationWarning {
                        code: "TIMESTAMP_FUTURE".to_string(),
                        message: "Timestamp is in the future".to_string(),
                        field_path: Some("timestamp".to_string()),
                        category: WarningCategory::ClinicalSafety,
                    });
                }
                
                // Check if timestamp is too far in the past
                if timestamp_utc < now - Duration::days(30) {
                    warnings.push(ValidationWarning {
                        code: "TIMESTAMP_OLD".to_string(),
                        message: "Timestamp is more than 30 days old".to_string(),
                        field_path: Some("timestamp".to_string()),
                        category: WarningCategory::ClinicalSafety,
                    });
                }
            } else {
                errors.push(ValidationError {
                    code: "TIMESTAMP_INVALID_FORMAT".to_string(),
                    message: "Invalid timestamp format".to_string(),
                    field_path: Some("timestamp".to_string()),
                    severity: ValidationSeverity::Error,
                    suggested_fix: Some("Use RFC 3339 format".to_string()),
                });
            }
        }

        ValidationResult {
            valid: errors.is_empty(),
            schema_version: "clinical-timestamp-validator-v1".to_string(),
            errors,
            warnings,
            metadata: ValidationMetadata {
                timestamp: Utc::now(),
                validator_version: "1.0.0".to_string(),
                duration_ms: 2,
                cache_hit: false,
            },
        }
    }

    fn name(&self) -> &str {
        "clinical-timestamp-validator"
    }

    fn description(&self) -> &str {
        "Validates clinical timestamps for safety and consistency"
    }
}

impl Default for SchemaValidationConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            registry_config: SchemaRegistryConfig {
                registry_type: SchemaRegistryType::InMemory,
                endpoint_url: None,
                auth_config: None,
                cache_config: SchemaCacheConfig {
                    enabled: true,
                    max_size: 1000,
                    ttl_seconds: 3600,
                    refresh_interval_seconds: 600,
                },
                loading_behavior: SchemaLoadingBehavior::Eager,
            },
            validation_rules: ValidationRulesConfig {
                strict_mode: false,
                allow_unknown_fields: true,
                custom_rules: HashMap::new(),
                field_validations: HashMap::new(),
                cross_field_rules: vec![],
            },
            serialization_config: SerializationConfig {
                default_format: SerializationFormat::Json,
                pretty_print: false,
                include_nulls: false,
                datetime_format: DateTimeFormat::Rfc3339,
                custom_serializers: HashMap::new(),
                compression: CompressionConfig {
                    enabled: false,
                    algorithm: CompressionAlgorithm::Gzip,
                    level: 6,
                    min_size_bytes: 1024,
                },
            },
            evolution_config: SchemaEvolutionConfig {
                strategy: EvolutionStrategy::Backward,
                compatibility: CompatibilityMode::Backward,
                versioning: VersioningStrategy::Semantic,
                migration_rules: vec![],
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_schema_validator_creation() {
        let config = SchemaValidationConfig::default();
        let validator = SchemaValidator::new(config).await;
        assert!(validator.is_ok());
    }

    #[tokio::test]
    async fn test_patient_id_validation() {
        let validator = PatientIdValidator;
        let context = ValidationContext {
            schema: SchemaDefinition {
                schema_id: "test".to_string(),
                name: "test".to_string(),
                version: "1.0.0".to_string(),
                description: "test".to_string(),
                schema_type: SchemaType::Custom("test".to_string()),
                schema: serde_json::json!({}),
                metadata: SchemaMetadata {
                    author: "test".to_string(),
                    tags: vec![],
                    category: "test".to_string(),
                    documentation_url: None,
                    examples: vec![],
                    related_schemas: vec![],
                },
                created_at: Utc::now(),
                modified_at: Utc::now(),
            },
            config: SchemaValidationConfig::default(),
            context_data: HashMap::new(),
        };

        // Valid patient ID
        let valid_data = serde_json::json!({"patient_id": "PAT12345"});
        let result = validator.validate(&valid_data, &context);
        assert!(result.valid);

        // Invalid patient ID (too short)
        let invalid_data = serde_json::json!({"patient_id": "PA"});
        let result = validator.validate(&invalid_data, &context);
        assert!(!result.valid);
        assert_eq!(result.errors.len(), 1);
    }
}