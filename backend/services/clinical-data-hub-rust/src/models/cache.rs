// Multi-layer cache models and structures
use crate::models::{CompressionType, ClinicalData};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Cache layer types for the multi-layer architecture
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum CacheLayer {
    /// L1: In-memory cache for ultra-fast access (< 1ms)
    L1Memory,
    /// L2: Redis cluster for distributed caching (1-5ms)
    L2Redis,
    /// L3: Persistent storage with compression (5-50ms)
    L3Persistent,
}

impl CacheLayer {
    /// Get expected latency range for the cache layer
    pub fn expected_latency_ms(&self) -> (f64, f64) {
        match self {
            CacheLayer::L1Memory => (0.01, 1.0),
            CacheLayer::L2Redis => (1.0, 5.0),
            CacheLayer::L3Persistent => (5.0, 50.0),
        }
    }
    
    /// Get typical capacity for the cache layer
    pub fn typical_capacity_mb(&self) -> usize {
        match self {
            CacheLayer::L1Memory => 512,      // 512 MB
            CacheLayer::L2Redis => 8192,      // 8 GB  
            CacheLayer::L3Persistent => 102400, // 100 GB
        }
    }
}

/// Cache entry with metadata and performance tracking
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheEntry {
    pub key: String,
    pub data: Vec<u8>,
    pub compressed: bool,
    pub compression_type: CompressionType,
    pub original_size: usize,
    pub compressed_size: usize,
    pub created_at: DateTime<Utc>,
    pub last_accessed: DateTime<Utc>,
    pub access_count: u64,
    pub ttl_seconds: Option<u32>,
    pub checksum: String,
    pub metadata: HashMap<String, String>,
}

impl CacheEntry {
    /// Create a new cache entry
    pub fn new(
        key: String,
        data: Vec<u8>,
        ttl_seconds: Option<u32>,
        compression_type: CompressionType,
    ) -> Result<Self, CacheError> {
        let now = Utc::now();
        let original_size = data.len();
        let checksum = Self::calculate_checksum(&data);
        
        let (compressed_data, compressed_size) = if compression_type != CompressionType::None {
            let compressed = Self::compress_data(&data, compression_type)?;
            let size = compressed.len();
            (compressed, size)
        } else {
            (data, original_size)
        };
        
        Ok(Self {
            key,
            data: compressed_data,
            compressed: compression_type != CompressionType::None,
            compression_type,
            original_size,
            compressed_size,
            created_at: now,
            last_accessed: now,
            access_count: 0,
            ttl_seconds,
            checksum,
            metadata: HashMap::new(),
        })
    }
    
    /// Calculate SHA-256 checksum
    fn calculate_checksum(data: &[u8]) -> String {
        use sha2::{Sha256, Digest};
        let mut hasher = Sha256::new();
        hasher.update(data);
        format!("{:x}", hasher.finalize())
    }
    
    /// Compress data using specified compression type
    fn compress_data(data: &[u8], compression_type: CompressionType) -> Result<Vec<u8>, CacheError> {
        match compression_type {
            CompressionType::None => Ok(data.to_vec()),
            CompressionType::Lz4 => {
                lz4::compress(data, None, false)
                    .map_err(|e| CacheError::CompressionFailed(e.to_string()))
            }
            CompressionType::Zstd => {
                zstd::encode_all(data, 3)
                    .map_err(|e| CacheError::CompressionFailed(e.to_string()))
            }
            CompressionType::MessagePack => {
                // For MessagePack, we assume the data is already serialized JSON
                let json: serde_json::Value = serde_json::from_slice(data)
                    .map_err(|e| CacheError::CompressionFailed(e.to_string()))?;
                rmp_serde::to_vec(&json)
                    .map_err(|e| CacheError::CompressionFailed(e.to_string()))
            }
        }
    }
    
    /// Decompress data
    pub fn decompress_data(&self) -> Result<Vec<u8>, CacheError> {
        if !self.compressed {
            return Ok(self.data.clone());
        }
        
        match self.compression_type {
            CompressionType::None => Ok(self.data.clone()),
            CompressionType::Lz4 => {
                lz4::decompress(&self.data, Some(self.original_size as i32))
                    .map_err(|e| CacheError::DecompressionFailed(e.to_string()))
            }
            CompressionType::Zstd => {
                zstd::decode_all(&self.data[..])
                    .map_err(|e| CacheError::DecompressionFailed(e.to_string()))
            }
            CompressionType::MessagePack => {
                let value: serde_json::Value = rmp_serde::from_slice(&self.data)
                    .map_err(|e| CacheError::DecompressionFailed(e.to_string()))?;
                serde_json::to_vec(&value)
                    .map_err(|e| CacheError::DecompressionFailed(e.to_string()))
            }
        }
    }
    
    /// Check if entry has expired
    pub fn is_expired(&self) -> bool {
        if let Some(ttl) = self.ttl_seconds {
            let age = (Utc::now() - self.created_at).num_seconds();
            age > ttl as i64
        } else {
            false
        }
    }
    
    /// Record access to the entry
    pub fn record_access(&mut self) {
        self.last_accessed = Utc::now();
        self.access_count += 1;
    }
    
    /// Get compression ratio
    pub fn compression_ratio(&self) -> f64 {
        if self.original_size == 0 {
            1.0
        } else {
            self.compressed_size as f64 / self.original_size as f64
        }
    }
    
    /// Get entry age in seconds
    pub fn age_seconds(&self) -> i64 {
        (Utc::now() - self.created_at).num_seconds()
    }
    
    /// Verify data integrity
    pub fn verify_integrity(&self) -> Result<bool, CacheError> {
        let decompressed = self.decompress_data()?;
        let computed_checksum = Self::calculate_checksum(&decompressed);
        Ok(computed_checksum == self.checksum)
    }
}

/// Cache statistics for performance monitoring
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheStats {
    pub layer: CacheLayer,
    pub total_entries: u64,
    pub memory_usage_bytes: u64,
    pub hit_count: u64,
    pub miss_count: u64,
    pub eviction_count: u64,
    pub error_count: u64,
    pub average_access_time_microseconds: f64,
    pub compression_stats: CompressionStats,
    pub last_updated: DateTime<Utc>,
}

impl CacheStats {
    /// Create new cache statistics
    pub fn new(layer: CacheLayer) -> Self {
        Self {
            layer,
            total_entries: 0,
            memory_usage_bytes: 0,
            hit_count: 0,
            miss_count: 0,
            eviction_count: 0,
            error_count: 0,
            average_access_time_microseconds: 0.0,
            compression_stats: CompressionStats::new(),
            last_updated: Utc::now(),
        }
    }
    
    /// Calculate hit ratio
    pub fn hit_ratio(&self) -> f64 {
        let total = self.hit_count + self.miss_count;
        if total == 0 {
            0.0
        } else {
            self.hit_count as f64 / total as f64
        }
    }
    
    /// Calculate operations per second
    pub fn operations_per_second(&self) -> f64 {
        let age_seconds = (Utc::now() - self.last_updated).num_seconds() as f64;
        if age_seconds == 0.0 {
            0.0
        } else {
            (self.hit_count + self.miss_count) as f64 / age_seconds
        }
    }
}

/// Compression statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompressionStats {
    pub total_compressed: u64,
    pub total_original_bytes: u64,
    pub total_compressed_bytes: u64,
    pub average_compression_ratio: f64,
    pub compression_time_microseconds: u64,
    pub decompression_time_microseconds: u64,
}

impl CompressionStats {
    pub fn new() -> Self {
        Self {
            total_compressed: 0,
            total_original_bytes: 0,
            total_compressed_bytes: 0,
            average_compression_ratio: 1.0,
            compression_time_microseconds: 0,
            decompression_time_microseconds: 0,
        }
    }
    
    /// Calculate space savings percentage
    pub fn space_savings_percent(&self) -> f64 {
        if self.total_original_bytes == 0 {
            0.0
        } else {
            let savings = self.total_original_bytes - self.total_compressed_bytes;
            (savings as f64 / self.total_original_bytes as f64) * 100.0
        }
    }
}

/// Cache configuration for different layers
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheConfig {
    pub layer: CacheLayer,
    pub max_size_bytes: u64,
    pub max_entries: u64,
    pub default_ttl_seconds: u32,
    pub compression_type: CompressionType,
    pub eviction_policy: EvictionPolicy,
    pub enable_metrics: bool,
}

/// Cache eviction policies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvictionPolicy {
    /// Least Recently Used
    Lru,
    /// Least Frequently Used
    Lfu,
    /// Time To Live based
    Ttl,
    /// Random eviction
    Random,
    /// First In First Out
    Fifo,
}

/// Cache errors
#[derive(Debug, thiserror::Error)]
pub enum CacheError {
    #[error("Cache entry not found: {0}")]
    NotFound(String),
    
    #[error("Cache is full")]
    CacheFull,
    
    #[error("Compression failed: {0}")]
    CompressionFailed(String),
    
    #[error("Decompression failed: {0}")]
    DecompressionFailed(String),
    
    #[error("Serialization error: {0}")]
    SerializationError(String),
    
    #[error("Network error: {0}")]
    NetworkError(String),
    
    #[error("Redis error: {0}")]
    RedisError(String),
    
    #[error("IO error: {0}")]
    IoError(String),
    
    #[error("Configuration error: {0}")]
    ConfigError(String),
}

/// Cache operation result
pub type CacheResult<T> = Result<T, CacheError>;

/// Cache key builder for consistent key generation
pub struct CacheKeyBuilder {
    prefix: String,
    separator: String,
}

impl CacheKeyBuilder {
    /// Create a new cache key builder
    pub fn new(prefix: &str) -> Self {
        Self {
            prefix: prefix.to_string(),
            separator: ":".to_string(),
        }
    }
    
    /// Build a cache key for patient data
    pub fn patient_data_key(&self, patient_id: &str, data_type: &str) -> String {
        format!("{}{}patient{}{}{}{}{}",
            self.prefix, self.separator,
            self.separator, patient_id,
            self.separator, data_type, self.separator, "data")
    }
    
    /// Build a cache key for aggregated data
    pub fn aggregated_data_key(&self, patient_id: &str, sources: &[String]) -> String {
        let sources_hash = Self::hash_sources(sources);
        format!("{}{}aggregated{}{}{}hash{}{}", 
            self.prefix, self.separator, 
            self.separator, patient_id, 
            self.separator, self.separator, sources_hash)
    }
    
    /// Build a cache key for metrics
    pub fn metrics_key(&self, metric_type: &str, time_window: &str) -> String {
        format!("{}{}metrics{}{}{}window{}{}", 
            self.prefix, self.separator, 
            self.separator, metric_type, 
            self.separator, self.separator, time_window)
    }
    
    /// Hash a list of sources for consistent key generation
    fn hash_sources(sources: &[String]) -> String {
        use sha2::{Sha256, Digest};
        let mut hasher = Sha256::new();
        for source in sources {
            hasher.update(source.as_bytes());
        }
        format!("{:x}", hasher.finalize())[..16].to_string()
    }
}