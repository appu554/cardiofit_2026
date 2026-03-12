use async_trait::async_trait;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use anyhow::Result;
use chrono::Utc;

use crate::models::cache::{CacheEntry, CacheKey};

/// L1 In-Memory Cache - Ultra-fast access
pub struct L1MemoryCache {
    data: Arc<RwLock<HashMap<String, CacheEntry>>>,
    max_size: usize,
    size: Arc<RwLock<usize>>,
}

impl L1MemoryCache {
    pub fn new(max_size_mb: usize) -> Self {
        Self {
            data: Arc::new(RwLock::new(HashMap::new())),
            max_size: max_size_mb * 1024 * 1024, // Convert MB to bytes
            size: Arc::new(RwLock::new(0)),
        }
    }

    pub async fn get(&self, key: &str) -> Option<CacheEntry> {
        let cache = self.data.read().await;
        cache.get(key).cloned()
    }

    pub async fn set(&self, key: String, entry: CacheEntry) -> Result<()> {
        let entry_size = entry.data.len();

        // Check if we need to evict entries
        if *self.size.read().await + entry_size > self.max_size {
            self.evict_lru().await?;
        }

        let mut cache = self.data.write().await;
        cache.insert(key, entry);

        let mut size = self.size.write().await;
        *size += entry_size;

        Ok(())
    }

    pub async fn delete(&self, key: &str) -> Result<()> {
        let mut cache = self.data.write().await;
        if let Some(entry) = cache.remove(key) {
            let mut size = self.size.write().await;
            *size -= entry.data.len();
        }
        Ok(())
    }

    pub async fn clear(&self) -> Result<()> {
        let mut cache = self.data.write().await;
        cache.clear();

        let mut size = self.size.write().await;
        *size = 0;

        Ok(())
    }

    async fn evict_lru(&self) -> Result<()> {
        let mut cache = self.data.write().await;

        // Find the least recently used entry
        if let Some((key_to_remove, _)) = cache.iter()
            .min_by_key(|(_, entry)| entry.last_accessed) {
            let key = key_to_remove.clone();
            if let Some(entry) = cache.remove(&key) {
                let mut size = self.size.write().await;
                *size -= entry.data.len();
            }
        }

        Ok(())
    }

    pub async fn stats(&self) -> HashMap<String, u64> {
        let cache = self.data.read().await;
        let mut stats = HashMap::new();
        stats.insert("entries".to_string(), cache.len() as u64);
        stats.insert("size_bytes".to_string(), *self.size.read().await as u64);
        stats.insert("max_size_bytes".to_string(), self.max_size as u64);
        stats
    }
}