use serde::{Deserialize, Serialize};
use std::time::Duration;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    // Service Configuration
    pub project_name: String,
    pub version: String,
    pub environment: String,
    pub debug: bool,

    // Server Configuration
    pub host: String,
    pub port: u16,
    pub grpc_port: u16,

    // Database Configuration
    pub database_url: String,
    pub database_pool_size: u32,
    pub database_max_overflow: u32,
    pub database_pool_timeout: u64,

    // Kafka Configuration
    pub kafka_bootstrap_servers: String,
    pub kafka_api_key: String,
    pub kafka_api_secret: String,
    pub kafka_security_protocol: String,
    pub kafka_sasl_mechanism: String,

    // Publisher Configuration
    pub publisher_enabled: bool,
    pub publisher_poll_interval_secs: u64,
    pub publisher_batch_size: usize,
    pub publisher_max_workers: usize,

    // Retry Configuration
    pub max_retry_attempts: u32,
    pub retry_base_delay_secs: u64,
    pub retry_max_delay_secs: u64,
    pub retry_exponential_base: f64,
    pub retry_jitter: bool,

    // Dead Letter Queue Configuration
    pub dlq_enabled: bool,
    pub dlq_max_retries: u32,

    // Medical Circuit Breaker Configuration
    pub medical_circuit_breaker_enabled: bool,
    pub medical_circuit_breaker_max_queue_depth: usize,
    pub medical_circuit_breaker_critical_threshold: f64,
    pub medical_circuit_breaker_recovery_timeout_secs: u64,

    // Security Configuration
    pub grpc_api_key: String,
    pub enable_auth: bool,

    // Monitoring Configuration
    pub enable_metrics: bool,
    pub metrics_port: u16,
    pub log_level: String,

    // Supported Services
    pub supported_services: Vec<String>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            // Service Configuration
            project_name: "Global Outbox Service Rust".to_string(),
            version: "1.0.0".to_string(),
            environment: "development".to_string(),
            debug: false,

            // Server Configuration
            host: "0.0.0.0".to_string(),
            port: 8043,
            grpc_port: 50053,

            // Database Configuration
            database_url: "postgresql://postgres.auugxeqzgrnknklgwqrh:9FTqQnA4LRCsu8sw@aws-0-ap-south-1.pooler.supabase.com:5432/postgres".to_string(),
            database_pool_size: 20,
            database_max_overflow: 30,
            database_pool_timeout: 30,

            // Kafka Configuration
            kafka_bootstrap_servers: "pkc-619z3.us-east1.gcp.confluent.cloud:9092".to_string(),
            kafka_api_key: "LGJ3AQ2L6VRPW4S2".to_string(),
            kafka_api_secret: "2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl".to_string(),
            kafka_security_protocol: "SASL_SSL".to_string(),
            kafka_sasl_mechanism: "PLAIN".to_string(),

            // Publisher Configuration
            publisher_enabled: true,
            publisher_poll_interval_secs: 2,
            publisher_batch_size: 100,
            publisher_max_workers: 4,

            // Retry Configuration
            max_retry_attempts: 5,
            retry_base_delay_secs: 1,
            retry_max_delay_secs: 60,
            retry_exponential_base: 2.0,
            retry_jitter: true,

            // Dead Letter Queue Configuration
            dlq_enabled: true,
            dlq_max_retries: 10,

            // Medical Circuit Breaker Configuration
            medical_circuit_breaker_enabled: true,
            medical_circuit_breaker_max_queue_depth: 1000,
            medical_circuit_breaker_critical_threshold: 0.8,
            medical_circuit_breaker_recovery_timeout_secs: 30,

            // Security Configuration
            grpc_api_key: "global-outbox-service-rust-key".to_string(),
            enable_auth: false,

            // Monitoring Configuration
            enable_metrics: true,
            metrics_port: 8044,
            log_level: "INFO".to_string(),

            // Supported Services
            supported_services: vec![
                "patient-service".to_string(),
                "observation-service".to_string(),
                "condition-service".to_string(),
                "medication-service".to_string(),
                "encounter-service".to_string(),
                "timeline-service".to_string(),
                "workflow-engine-service".to_string(),
                "order-management-service".to_string(),
                "scheduling-service".to_string(),
                "organization-service".to_string(),
                "device-data-ingestion-service".to_string(),
                "lab-service".to_string(),
                "fhir-service".to_string(),
                "generic-service".to_string(),
            ],
        }
    }
}

impl Config {
    pub fn load() -> Result<Self, config::ConfigError> {
        let mut cfg = config::Config::builder()
            .add_source(config::Config::try_from(&Config::default())?)
            .add_source(config::File::with_name("config").required(false))
            .add_source(config::File::with_name("config.toml").required(false))
            .add_source(config::File::with_name("config.yaml").required(false))
            .add_source(config::Environment::with_prefix("OUTBOX").separator("_"))
            .build()?;

        let mut config: Config = cfg.try_deserialize()?;

        // Environment-specific adjustments
        if config.is_production() {
            config.debug = false;
            config.log_level = "INFO".to_string();
        } else if config.is_development() {
            config.debug = true;
            config.log_level = "DEBUG".to_string();
        }

        Ok(config)
    }

    pub fn is_production(&self) -> bool {
        self.environment.to_lowercase() == "production"
    }

    pub fn is_development(&self) -> bool {
        self.environment.to_lowercase() == "development"
    }

    pub fn publisher_poll_interval(&self) -> Duration {
        Duration::from_secs(self.publisher_poll_interval_secs)
    }

    pub fn retry_base_delay(&self) -> Duration {
        Duration::from_secs(self.retry_base_delay_secs)
    }

    pub fn retry_max_delay(&self) -> Duration {
        Duration::from_secs(self.retry_max_delay_secs)
    }

    pub fn database_pool_timeout(&self) -> Duration {
        Duration::from_secs(self.database_pool_timeout)
    }

    pub fn medical_circuit_breaker_recovery_timeout(&self) -> Duration {
        Duration::from_secs(self.medical_circuit_breaker_recovery_timeout_secs)
    }

    pub fn get_kafka_config(&self) -> std::collections::HashMap<String, String> {
        let mut config = std::collections::HashMap::new();
        config.insert("bootstrap.servers".to_string(), self.kafka_bootstrap_servers.clone());
        config.insert("security.protocol".to_string(), self.kafka_security_protocol.clone());
        config.insert("sasl.mechanism".to_string(), self.kafka_sasl_mechanism.clone());
        config.insert("sasl.username".to_string(), self.kafka_api_key.clone());
        config.insert("sasl.password".to_string(), self.kafka_api_secret.clone());
        config.insert(
            "client.id".to_string(),
            format!("{}-producer", self.project_name.to_lowercase().replace(" ", "-"))
        );
        config.insert("acks".to_string(), "all".to_string());
        config.insert("retries".to_string(), "3".to_string());
        config.insert("retry.backoff.ms".to_string(), "1000".to_string());
        config.insert("request.timeout.ms".to_string(), "30000".to_string());
        config.insert("delivery.timeout.ms".to_string(), "120000".to_string());
        config.insert("compression.type".to_string(), "snappy".to_string());
        config.insert("batch.size".to_string(), "16384".to_string());
        config.insert("linger.ms".to_string(), "10".to_string());
        config
    }
}