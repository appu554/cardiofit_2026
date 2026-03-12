// Multi-layer cache implementation for ultra-high performance clinical data access
pub mod l1_memory;
pub mod l2_redis;
pub mod l3_persistent;
pub mod manager;

use crate::models::cache::{CacheEntry, CacheError, CacheLayer, CacheResult, CacheStats};
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use std::collections::HashMap;

/// Cache trait for all cache layer implementations
#[async_trait]
pub trait Cache: Send + Sync {
    /// Get data from cache
    async fn get(&self, key: &str) -> CacheResult<Option<CacheEntry>>;
    
    /// Set data in cache
    async fn set(&self, key: String, entry: CacheEntry) -> CacheResult<()>;
    
    /// Delete data from cache
    async fn delete(&self, key: &str) -> CacheResult<bool>;
    
    /// Check if key exists
    async fn exists(&self, key: &str) -> CacheResult<bool>;
    
    /// Get multiple keys
    async fn get_multi(&self, keys: &[String]) -> CacheResult<HashMap<String, CacheEntry>>;
    
    /// Set multiple keys
    async fn set_multi(&self, entries: HashMap<String, CacheEntry>) -> CacheResult<()>;
    
    /// Delete multiple keys
    async fn delete_multi(&self, keys: &[String]) -> CacheResult<usize>;
    
    /// Delete keys matching pattern
    async fn delete_pattern(&self, pattern: &str) -> CacheResult<usize>;
    
    /// Get cache statistics
    async fn get_stats(&self) -> CacheResult<CacheStats>;
    
    /// Clear all entries
    async fn clear(&self) -> CacheResult<()>;
    
    /// Get cache layer type
    fn layer(&self) -> CacheLayer;
    
    /// Health check
    async fn health_check(&self) -> CacheResult<bool>;
}

/// Cache warming strategies
#[derive(Debug, Clone)]
pub enum WarmingStrategy {
    /// Predictive warming based on usage patterns
    Predictive {
        lookback_hours: u32,
        confidence_threshold: f64,
    },
    /// Historical warming based on previous access patterns
    Historical {
        time_window_hours: u32,
        min_access_count: u32,
    },
    /// Scheduled warming for known access patterns
    Scheduled {
        schedule: Vec<WarmingSchedule>,
    },
}

/// Warming schedule entry
#[derive(Debug, Clone)]
pub struct WarmingSchedule {
    pub hour: u8,
    pub minute: u8,
    pub patient_patterns: Vec<String>,
    pub data_types: Vec<String>,
}

/// Cache warming result
#[derive(Debug, Clone)]
pub struct WarmingResult {
    pub keys_warmed: usize,
    pub keys_failed: usize,
    pub total_size_bytes: u64,
    pub warming_time_ms: u64,
    pub strategy_used: String,
}

/// Cache invalidation patterns
#[derive(Debug, Clone)]
pub enum InvalidationPattern {
    /// Exact key match
    Exact(String),
    /// Prefix match
    Prefix(String),
    /// Regex pattern
    Regex(String),
    /// Patient-specific invalidation
    Patient(String),
    /// Data type invalidation
    DataType(String),
}

/// Cache optimization operations
#[derive(Debug, Clone)]
pub enum OptimizationOperation {
    /// Compress entries to save space
    Compress,
    /// Remove expired entries
    EvictExpired,
    /// Reorganize for better locality
    Reorganize,
    /// Prefetch predicted entries
    Prefetch,
}

/// Performance hints for cache operations
#[derive(Debug, Clone, Default)]
pub struct PerformanceHints {
    /// Prefer speed over compression
    pub favor_speed: bool,
    /// Allow stale data if fresh data is slow
    pub allow_stale: bool,
    /// Maximum staleness in seconds
    pub max_staleness_seconds: Option<u32>,
    /// Preferred cache layers in order
    pub preferred_layers: Vec<CacheLayer>,
    /// Expected data size for optimization
    pub expected_size_bytes: Option<usize>,
}

/// Cache operation metrics
#[derive(Debug, Clone)]
pub struct CacheMetrics {
    pub operation_type: String,
    pub layer: CacheLayer,
    pub duration_microseconds: u64,
    pub data_size_bytes: usize,
    pub hit: bool,
    pub error: Option<String>,
    pub timestamp: DateTime<Utc>,
}

/// Cache health status
#[derive(Debug, Clone)]
pub struct CacheHealth {
    pub layer: CacheLayer,
    pub healthy: bool,
    pub response_time_ms: f64,
    pub memory_usage_percent: f64,
    pub error_rate_percent: f64,
    pub last_check: DateTime<Utc>,
    pub details: HashMap<String, String>,
}

/// Multi-layer cache configuration
#[derive(Debug, Clone)]
pub struct MultiLayerConfig {
    pub enable_l1: bool,
    pub enable_l2: bool,
    pub enable_l3: bool,
    pub write_through: bool,
    pub read_through: bool,
    pub consistency_level: ConsistencyLevel,
    pub performance_hints: PerformanceHints,
}

/// Cache consistency levels
#[derive(Debug, Clone)]
pub enum ConsistencyLevel {
    /// Best performance, eventual consistency
    Eventual,
    /// Balanced performance and consistency
    Monotonic,
    /// Strong consistency, may impact performance
    Strong,
}

impl Default for MultiLayerConfig {
    fn default() -> Self {
        Self {
            enable_l1: true,
            enable_l2: true,
            enable_l3: true,
            write_through: true,
            read_through: true,
            consistency_level: ConsistencyLevel::Monotonic,
            performance_hints: PerformanceHints::default(),
        }
    }
}

/// Utility functions for cache operations
pub mod utils {
    use super::*;
    use std::time::{SystemTime, UNIX_EPOCH};
    
    /// Generate a unique cache key with timestamp
    pub fn generate_timestamped_key(base_key: &str) -> String {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis();
        format!("{}:{}", base_key, timestamp)
    }
    
    /// Extract patient ID from cache key
    pub fn extract_patient_id(key: &str) -> Option<String> {
        // Assuming key format: prefix:patient:patient_id:...
        let parts: Vec<&str> = key.split(':').collect();
        if parts.len() >= 3 && parts[1] == "patient" {
            Some(parts[2].to_string())
        } else {
            None
        }
    }
    
    /// Calculate optimal compression type based on data characteristics
    pub fn optimal_compression(data_size: usize, data_type: &str) -> crate::models::CompressionType {
        match data_type {
            "json" | "text" if data_size > 1024 => crate::models::CompressionType::Zstd,
            "binary" if data_size > 512 => crate::models::CompressionType::Lz4,
            "structured" => crate::models::CompressionType::MessagePack,
            _ => crate::models::CompressionType::None,
        }
    }
    
    /// Estimate memory usage for cache entry
    pub fn estimate_memory_usage(entry: &CacheEntry) -> usize {
        // Base overhead for the struct
        let base_overhead = std::mem::size_of::<CacheEntry>();
        
        // Key size
        let key_size = entry.key.len();
        
        // Data size
        let data_size = entry.data.len();
        
        // Metadata size
        let metadata_size: usize = entry.metadata.iter()
            .map(|(k, v)| k.len() + v.len())
            .sum();
        
        base_overhead + key_size + data_size + metadata_size
    }
    
    /// Check if cache key matches pattern
    pub fn key_matches_pattern(key: &str, pattern: &InvalidationPattern) -> bool {
        match pattern {
            InvalidationPattern::Exact(exact) => key == exact,
            InvalidationPattern::Prefix(prefix) => key.starts_with(prefix),
            InvalidationPattern::Regex(regex_str) => {
                if let Ok(regex) = regex::Regex::new(regex_str) {
                    regex.is_match(key)
                } else {
                    false
                }
            }
            InvalidationPattern::Patient(patient_id) => {
                key.contains(&format!(":patient:{}:", patient_id)) ||
                key.contains(&format!("patient:{}", patient_id))
            }
            InvalidationPattern::DataType(data_type) => {
                key.contains(&format!(":data:{}", data_type)) ||
                key.contains(&format!("data_type:{}", data_type))
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::cache::CompressionType;
    
    #[test]
    fn test_extract_patient_id() {
        assert_eq!(
            utils::extract_patient_id("clinical:patient:12345:demographics"),
            Some("12345".to_string())
        );
        assert_eq!(
            utils::extract_patient_id("invalid_key"),
            None
        );
    }
    
    #[test]
    fn test_optimal_compression() {
        assert_eq!(
            utils::optimal_compression(2048, "json"),
            CompressionType::Zstd
        );
        assert_eq!(
            utils::optimal_compression(1024, "binary"),
            CompressionType::Lz4
        );
        assert_eq!(
            utils::optimal_compression(100, "text"),
            CompressionType::None
        );
    }
    
    #[test]
    fn test_key_pattern_matching() {
        assert!(utils::key_matches_pattern(
            "clinical:patient:12345:data", 
            &InvalidationPattern::Patient("12345".to_string())
        ));
        assert!(utils::key_matches_pattern(
            "clinical:prefix:test", 
            &InvalidationPattern::Prefix("clinical:".to_string())
        ));
    }
}