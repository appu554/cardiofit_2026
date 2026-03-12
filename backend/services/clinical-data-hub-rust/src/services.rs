pub mod grpc;
pub mod http;

use anyhow::Result;
use std::sync::Arc;
use tonic::{Request, Response, Status};

use crate::cache::manager::CacheManager;
use crate::models::performance::PerformanceMetrics;

/// Main service implementation for Clinical Data Hub
pub struct ClinicalDataHubService {
    cache_manager: Arc<CacheManager>,
}

impl ClinicalDataHubService {
    pub fn new(cache_manager: Arc<CacheManager>) -> Self {
        Self { cache_manager }
    }

    pub async fn get_cached_data(&self, key: &str) -> Result<Option<Vec<u8>>> {
        self.cache_manager.get(key).await
    }

    pub async fn set_cached_data(&self, key: &str, data: Vec<u8>) -> Result<()> {
        self.cache_manager.set(key, data).await
    }

    pub async fn invalidate_cache(&self, key: &str) -> Result<()> {
        self.cache_manager.delete(key).await
    }

    pub async fn get_performance_metrics(&self) -> Result<PerformanceMetrics> {
        let cache_metrics = self.cache_manager.get_metrics().await;

        let total_hits = cache_metrics.l1_hits + cache_metrics.l2_hits + cache_metrics.l3_hits;
        let total_requests = cache_metrics.total_requests;

        let hit_rate = if total_requests > 0 {
            (total_hits as f64) / (total_requests as f64)
        } else {
            0.0
        };

        Ok(PerformanceMetrics {
            cache_hit_rate: hit_rate,
            average_response_time_ms: 5.0, // Mock value
            throughput_per_second: 1000,   // Mock value
            memory_usage_bytes: 0,
            cpu_usage_percent: 0.0,
            timestamp: chrono::Utc::now().timestamp(),
        })
    }
}