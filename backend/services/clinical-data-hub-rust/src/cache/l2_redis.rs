use async_trait::async_trait;
use redis::aio::ConnectionManager;
use redis::{AsyncCommands, Client};
use anyhow::{Result, Context};
use std::collections::HashMap;

use crate::models::cache::{CacheEntry, CacheKey};

/// L2 Redis Cache - Distributed cache layer
pub struct L2RedisCache {
    client: Client,
    connection: Option<ConnectionManager>,
}

impl L2RedisCache {
    pub fn new(redis_url: &str) -> Result<Self> {
        let client = Client::open(redis_url)
            .context("Failed to create Redis client")?;

        Ok(Self {
            client,
            connection: None,
        })
    }

    pub async fn connect(&mut self) -> Result<()> {
        let connection = self.client
            .get_tokio_connection_manager()
            .await
            .context("Failed to get Redis connection")?;
        self.connection = Some(connection);
        Ok(())
    }

    pub async fn get(&mut self, key: &str) -> Result<Option<Vec<u8>>> {
        if let Some(ref mut conn) = self.connection {
            let result: Option<Vec<u8>> = conn.get(key).await?;
            Ok(result)
        } else {
            Ok(None)
        }
    }

    pub async fn set(&mut self, key: &str, value: &[u8], ttl_seconds: u64) -> Result<()> {
        if let Some(ref mut conn) = self.connection {
            conn.set_ex(key, value, ttl_seconds).await?;
        }
        Ok(())
    }

    pub async fn delete(&mut self, key: &str) -> Result<()> {
        if let Some(ref mut conn) = self.connection {
            conn.del(key).await?;
        }
        Ok(())
    }

    pub async fn exists(&mut self, key: &str) -> Result<bool> {
        if let Some(ref mut conn) = self.connection {
            let exists: bool = conn.exists(key).await?;
            Ok(exists)
        } else {
            Ok(false)
        }
    }

    pub async fn clear_pattern(&mut self, pattern: &str) -> Result<u64> {
        if let Some(ref mut conn) = self.connection {
            let keys: Vec<String> = conn.keys(pattern).await?;
            if !keys.is_empty() {
                let deleted: u64 = conn.del(&keys).await?;
                Ok(deleted)
            } else {
                Ok(0)
            }
        } else {
            Ok(0)
        }
    }

    pub async fn stats(&mut self) -> Result<HashMap<String, String>> {
        let mut stats = HashMap::new();
        if let Some(ref mut conn) = self.connection {
            let info: String = redis::cmd("INFO")
                .arg("stats")
                .query_async(conn)
                .await?;

            for line in info.lines() {
                if line.contains(':') {
                    let parts: Vec<&str> = line.split(':').collect();
                    if parts.len() == 2 {
                        stats.insert(
                            parts[0].to_string(),
                            parts[1].to_string(),
                        );
                    }
                }
            }
        }
        Ok(stats)
    }
}