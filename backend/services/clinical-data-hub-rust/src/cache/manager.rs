use anyhow::{Result, Context};
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, debug, warn};

use crate::cache::{l1_memory::L1MemoryCache, l2_redis::L2RedisCache, l3_persistent::L3PersistentCache};
use crate::models::cache::{CacheEntry, CacheKey};
use crate::models::performance::CacheMetrics;

/// Multi-tier cache manager orchestrating L1, L2, and L3 caches
pub struct CacheManager {
    l1_cache: Arc<L1MemoryCache>,
    l2_cache: Arc<RwLock<L2RedisCache>>,
    l3_cache: Arc<RwLock<L3PersistentCache>>,
    metrics: Arc<RwLock<CacheMetrics>>,
}

impl CacheManager {
    pub fn new(
        l1_cache: L1MemoryCache,
        l2_cache: L2RedisCache,
        l3_cache: L3PersistentCache,
    ) -> Self {
        Self {
            l1_cache: Arc::new(l1_cache),
            l2_cache: Arc::new(RwLock::new(l2_cache)),
            l3_cache: Arc::new(RwLock::new(l3_cache)),
            metrics: Arc::new(RwLock::new(CacheMetrics::default())),
        }
    }

    /// Get data from cache, checking L1 -> L2 -> L3 in order
    pub async fn get(&self, key: &str) -> Result<Option<Vec<u8>>> {
        let mut metrics = self.metrics.write().await;
        metrics.total_requests += 1;

        // Check L1 cache first
        if let Some(entry) = self.l1_cache.get(key).await {
            debug!("L1 cache hit for key: {}", key);
            metrics.l1_hits += 1;
            return Ok(Some(entry.data));
        }
        metrics.l1_misses += 1;

        // Check L2 cache
        let mut l2_cache = self.l2_cache.write().await;
        if let Some(data) = l2_cache.get(key).await? {
            debug!("L2 cache hit for key: {}", key);
            metrics.l2_hits += 1;

            // Promote to L1
            let entry = CacheEntry {
                key: key.to_string(),
                data: data.clone(),
                metadata: None,
                created_at: chrono::Utc::now().timestamp(),
                last_accessed: chrono::Utc::now().timestamp(),
                ttl_seconds: 3600,
                compressed: false,
            };
            let _ = self.l1_cache.set(key.to_string(), entry).await;

            return Ok(Some(data));
        }
        metrics.l2_misses += 1;

        // Check L3 cache
        let l3_cache = self.l3_cache.read().await;
        if let Some(data) = l3_cache.get(key).await? {
            debug!("L3 cache hit for key: {}", key);
            metrics.l3_hits += 1;

            // Promote to L2 and L1
            let _ = l2_cache.set(key, &data, 3600).await;

            let entry = CacheEntry {
                key: key.to_string(),
                data: data.clone(),
                metadata: None,
                created_at: chrono::Utc::now().timestamp(),
                last_accessed: chrono::Utc::now().timestamp(),
                ttl_seconds: 3600,
                compressed: false,
            };
            let _ = self.l1_cache.set(key.to_string(), entry).await;

            return Ok(Some(data));
        }
        metrics.l3_misses += 1;

        debug!("Cache miss for key: {}", key);
        Ok(None)
    }

    /// Set data in all cache layers
    pub async fn set(&self, key: &str, data: Vec<u8>) -> Result<()> {
        // Set in L1
        let entry = CacheEntry {
            key: key.to_string(),
            data: data.clone(),
            metadata: None,
            created_at: chrono::Utc::now().timestamp(),
            last_accessed: chrono::Utc::now().timestamp(),
            ttl_seconds: 3600,
            compressed: false,
        };
        self.l1_cache.set(key.to_string(), entry).await?;

        // Set in L2
        let mut l2_cache = self.l2_cache.write().await;
        l2_cache.set(key, &data, 3600).await?;

        // Set in L3
        let l3_cache = self.l3_cache.read().await;
        l3_cache.set(key, &data, Some(86400)).await?;

        info!("Data set in all cache layers for key: {}", key);
        Ok(())
    }

    /// Delete data from all cache layers
    pub async fn delete(&self, key: &str) -> Result<()> {
        self.l1_cache.delete(key).await?;

        let mut l2_cache = self.l2_cache.write().await;
        l2_cache.delete(key).await?;

        let l3_cache = self.l3_cache.read().await;
        l3_cache.delete(key).await?;

        info!("Data deleted from all cache layers for key: {}", key);
        Ok(())
    }

    /// Clear all caches
    pub async fn clear_all(&self) -> Result<()> {
        self.l1_cache.clear().await?;

        let mut l2_cache = self.l2_cache.write().await;
        l2_cache.clear_pattern("*").await?;

        warn!("All cache layers have been cleared");
        Ok(())
    }

    /// Get cache metrics
    pub async fn get_metrics(&self) -> CacheMetrics {
        self.metrics.read().await.clone()
    }

    /// Initialize connections for L2 and L3 caches
    pub async fn initialize(&self) -> Result<()> {
        let mut l2_cache = self.l2_cache.write().await;
        l2_cache.connect().await
            .context("Failed to connect to L2 Redis cache")?;

        let mut l3_cache = self.l3_cache.write().await;
        l3_cache.connect().await
            .context("Failed to connect to L3 persistent cache")?;

        info!("Cache manager initialized successfully");
        Ok(())
    }
}