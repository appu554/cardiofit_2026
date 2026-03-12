// CAE Engine Configuration
//
// This module handles configuration for the Clinical Assertion Engine,
// including database connections, cache settings, and performance tuning.

use serde::{Deserialize, Serialize};
use std::time::Duration;
use crate::cae::CAECapability;

/// Configuration for the Clinical Assertion Engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CAEConfig {
    /// Path to clinical rules database
    pub rules_path: String,
    
    /// Path to knowledge database
    pub knowledge_db_path: String,
    
    /// Cache configuration
    pub cache: CacheConfig,
    
    /// Performance settings
    pub performance: PerformanceConfig,
    
    /// Database connection settings
    pub database: DatabaseConfig,
    
    /// Enabled capabilities
    pub enabled_capabilities: Vec<String>,
    
    /// Logging configuration
    pub logging: LoggingConfig,
}

/// Cache configuration settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheConfig {
    /// Maximum number of entries in the cache
    pub size: usize,
    
    /// Cache entry TTL in seconds
    pub ttl_seconds: u64,
    
    /// Enable/disable caching
    pub enabled: bool,
    
    /// Cache hit rate threshold for warnings
    pub hit_rate_threshold: f64,
}

/// Performance tuning configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceConfig {
    /// Maximum evaluation time in milliseconds
    pub max_evaluation_time_ms: u64,
    
    /// Number of worker threads for parallel processing
    pub worker_threads: usize,
    
    /// Enable performance monitoring
    pub monitoring_enabled: bool,
    
    /// Batch size for bulk operations
    pub batch_size: usize,
}

/// Database connection configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    /// Drug interactions database path
    pub drug_interactions_path: String,
    
    /// Contraindications database path
    pub contraindications_path: String,
    
    /// Dosing rules database path
    pub dosing_rules_path: String,
    
    /// Database connection timeout in milliseconds
    pub connection_timeout_ms: u64,
    
    /// Enable database connection pooling
    pub pooling_enabled: bool,
}

/// Logging configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoggingConfig {
    /// Log level (debug, info, warn, error)
    pub level: String,
    
    /// Enable structured logging
    pub structured: bool,
    
    /// Enable performance logging
    pub performance_logging: bool,
    
    /// Enable audit logging for safety decisions
    pub audit_logging: bool,
}

impl Default for CAEConfig {
    fn default() -> Self {
        Self {
            rules_path: "./data/clinical_rules".to_string(),
            knowledge_db_path: "./data/knowledge_graph".to_string(),
            cache: CacheConfig::default(),
            performance: PerformanceConfig::default(),
            database: DatabaseConfig::default(),
            enabled_capabilities: CAECapability::all()
                .iter()
                .map(|cap| cap.as_str().to_string())
                .collect(),
            logging: LoggingConfig::default(),
        }
    }
}

impl Default for CacheConfig {
    fn default() -> Self {
        Self {
            size: 10000,
            ttl_seconds: 3600, // 1 hour
            enabled: true,
            hit_rate_threshold: 0.7,
        }
    }
}

impl Default for PerformanceConfig {
    fn default() -> Self {
        Self {
            max_evaluation_time_ms: 100, // Sub-100ms target
            worker_threads: num_cpus::get().min(8),
            monitoring_enabled: true,
            batch_size: 100,
        }
    }
}

impl Default for DatabaseConfig {
    fn default() -> Self {
        Self {
            drug_interactions_path: "./data/drug_interactions.db".to_string(),
            contraindications_path: "./data/contraindications.db".to_string(),
            dosing_rules_path: "./data/dosing_rules.db".to_string(),
            connection_timeout_ms: 5000,
            pooling_enabled: true,
        }
    }
}

impl Default for LoggingConfig {
    fn default() -> Self {
        Self {
            level: "info".to_string(),
            structured: true,
            performance_logging: true,
            audit_logging: true,
        }
    }
}

impl CAEConfig {
    /// Create a test configuration with minimal settings
    pub fn test_config() -> Self {
        Self {
            rules_path: "test_data".to_string(),
            knowledge_db_path: "test_data".to_string(),
            cache: CacheConfig {
                size: 100,
                ttl_seconds: 60,
                enabled: true,
                hit_rate_threshold: 0.5,
            },
            performance: PerformanceConfig {
                max_evaluation_time_ms: 50,
                worker_threads: 2,
                monitoring_enabled: false,
                batch_size: 10,
            },
            database: DatabaseConfig {
                drug_interactions_path: ":memory:".to_string(),
                contraindications_path: ":memory:".to_string(),
                dosing_rules_path: ":memory:".to_string(),
                connection_timeout_ms: 1000,
                pooling_enabled: false,
            },
            enabled_capabilities: vec![
                "drug_interaction".to_string(),
                "contraindication".to_string(),
                "dosing_validation".to_string(),
            ],
            logging: LoggingConfig {
                level: "warn".to_string(),
                structured: false,
                performance_logging: false,
                audit_logging: false,
            },
        }
    }
    
    /// Validate the configuration
    pub fn validate(&self) -> Result<(), String> {
        // Validate cache size
        if self.cache.size == 0 {
            return Err("Cache size must be greater than 0".to_string());
        }
        
        // Validate performance settings
        if self.performance.max_evaluation_time_ms == 0 {
            return Err("Max evaluation time must be greater than 0".to_string());
        }
        
        if self.performance.worker_threads == 0 {
            return Err("Worker threads must be greater than 0".to_string());
        }
        
        // Validate log level
        match self.logging.level.as_str() {
            "debug" | "info" | "warn" | "error" => {},
            _ => return Err("Invalid log level. Must be debug, info, warn, or error".to_string()),
        }
        
        // Validate capabilities
        for capability in &self.enabled_capabilities {
            match capability.as_str() {
                "drug_interaction" | "contraindication" | "dosing_validation" |
                "allergy_check" | "duplicate_therapy" | "clinical_protocol" => {},
                _ => return Err(format!("Invalid capability: {}", capability)),
            }
        }
        
        Ok(())
    }
    
    /// Get maximum evaluation timeout as Duration
    pub fn max_evaluation_timeout(&self) -> Duration {
        Duration::from_millis(self.performance.max_evaluation_time_ms)
    }
    
    /// Get cache TTL as Duration
    pub fn cache_ttl(&self) -> Duration {
        Duration::from_secs(self.cache.ttl_seconds)
    }
    
    /// Check if a capability is enabled
    pub fn is_capability_enabled(&self, capability: &CAECapability) -> bool {
        self.enabled_capabilities.contains(&capability.as_str().to_string())
    }
    
    /// Get the number of worker threads
    pub fn worker_threads(&self) -> usize {
        self.performance.worker_threads
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_default_config() {
        let config = CAEConfig::default();
        assert!(config.validate().is_ok());
        assert!(config.cache.enabled);
        assert!(config.performance.monitoring_enabled);
        assert!(config.logging.structured);
    }
    
    #[test]
    fn test_test_config() {
        let config = CAEConfig::test_config();
        assert!(config.validate().is_ok());
        assert_eq!(config.cache.size, 100);
        assert_eq!(config.performance.worker_threads, 2);
        assert!(!config.performance.monitoring_enabled);
    }
    
    #[test]
    fn test_config_validation() {
        let mut config = CAEConfig::default();
        
        // Valid config should pass
        assert!(config.validate().is_ok());
        
        // Invalid cache size
        config.cache.size = 0;
        assert!(config.validate().is_err());
        
        // Reset and test invalid log level
        config = CAEConfig::default();
        config.logging.level = "invalid".to_string();
        assert!(config.validate().is_err());
        
        // Reset and test invalid capability
        config = CAEConfig::default();
        config.enabled_capabilities.push("invalid_capability".to_string());
        assert!(config.validate().is_err());
    }
    
    #[test]
    fn test_capability_check() {
        let config = CAEConfig::default();
        assert!(config.is_capability_enabled(&CAECapability::DrugInteraction));
        assert!(config.is_capability_enabled(&CAECapability::Contraindication));
        
        let mut limited_config = CAEConfig::test_config();
        limited_config.enabled_capabilities = vec!["drug_interaction".to_string()];
        assert!(limited_config.is_capability_enabled(&CAECapability::DrugInteraction));
        assert!(!limited_config.is_capability_enabled(&CAECapability::AllergyCheck));
    }
    
    #[test]
    fn test_duration_conversions() {
        let config = CAEConfig::default();
        
        let timeout = config.max_evaluation_timeout();
        assert_eq!(timeout.as_millis() as u64, config.performance.max_evaluation_time_ms);
        
        let ttl = config.cache_ttl();
        assert_eq!(ttl.as_secs(), config.cache.ttl_seconds);
    }
    
    #[test]
    fn test_serialization() {
        let config = CAEConfig::default();
        
        // Test JSON serialization
        let json = serde_json::to_string(&config).unwrap();
        let deserialized: CAEConfig = serde_json::from_str(&json).unwrap();
        
        assert_eq!(config.cache.size, deserialized.cache.size);
        assert_eq!(config.performance.max_evaluation_time_ms, 
                   deserialized.performance.max_evaluation_time_ms);
        assert_eq!(config.enabled_capabilities, deserialized.enabled_capabilities);
    }
}