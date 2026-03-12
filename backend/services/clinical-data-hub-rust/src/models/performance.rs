use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    pub cache_hit_rate: f64,
    pub average_response_time_ms: f64,
    pub throughput_per_second: u64,
    pub memory_usage_bytes: u64,
    pub cpu_usage_percent: f32,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheMetrics {
    pub l1_hits: u64,
    pub l1_misses: u64,
    pub l2_hits: u64,
    pub l2_misses: u64,
    pub l3_hits: u64,
    pub l3_misses: u64,
    pub total_requests: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LatencyMetrics {
    pub p50_ms: f64,
    pub p95_ms: f64,
    pub p99_ms: f64,
    pub max_ms: f64,
    pub min_ms: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ThroughputMetrics {
    pub requests_per_second: u64,
    pub bytes_per_second: u64,
    pub errors_per_second: u64,
}

impl Default for PerformanceMetrics {
    fn default() -> Self {
        Self {
            cache_hit_rate: 0.0,
            average_response_time_ms: 0.0,
            throughput_per_second: 0,
            memory_usage_bytes: 0,
            cpu_usage_percent: 0.0,
            timestamp: chrono::Utc::now().timestamp(),
        }
    }
}

impl Default for CacheMetrics {
    fn default() -> Self {
        Self {
            l1_hits: 0,
            l1_misses: 0,
            l2_hits: 0,
            l2_misses: 0,
            l3_hits: 0,
            l3_misses: 0,
            total_requests: 0,
        }
    }
}