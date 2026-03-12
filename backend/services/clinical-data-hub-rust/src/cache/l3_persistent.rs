use async_trait::async_trait;
use sqlx::{PgPool, postgres::PgPoolOptions};
use anyhow::{Result, Context};
use std::collections::HashMap;
use chrono::Utc;

use crate::models::cache::{CacheEntry, CacheKey};

/// L3 Persistent Cache - PostgreSQL-backed storage
pub struct L3PersistentCache {
    pool: Option<PgPool>,
    connection_string: String,
}

impl L3PersistentCache {
    pub fn new(connection_string: &str) -> Self {
        Self {
            pool: None,
            connection_string: connection_string.to_string(),
        }
    }

    pub async fn connect(&mut self) -> Result<()> {
        let pool = PgPoolOptions::new()
            .max_connections(10)
            .connect(&self.connection_string)
            .await
            .context("Failed to connect to PostgreSQL")?;

        // Create cache table if it doesn't exist
        sqlx::query(
            r#"
            CREATE TABLE IF NOT EXISTS cache_entries (
                key VARCHAR(255) PRIMARY KEY,
                value BYTEA NOT NULL,
                metadata JSONB,
                created_at TIMESTAMPTZ DEFAULT NOW(),
                accessed_at TIMESTAMPTZ DEFAULT NOW(),
                ttl INTEGER,
                compressed BOOLEAN DEFAULT FALSE
            )
            "#,
        )
        .execute(&pool)
        .await
        .context("Failed to create cache table")?;

        self.pool = Some(pool);
        Ok(())
    }

    pub async fn get(&self, key: &str) -> Result<Option<Vec<u8>>> {
        if let Some(ref pool) = self.pool {
            let row = sqlx::query!(
                r#"
                UPDATE cache_entries
                SET accessed_at = NOW()
                WHERE key = $1
                RETURNING value
                "#,
                key
            )
            .fetch_optional(pool)
            .await?;

            Ok(row.map(|r| r.value))
        } else {
            Ok(None)
        }
    }

    pub async fn set(&self, key: &str, value: &[u8], ttl_seconds: Option<i32>) -> Result<()> {
        if let Some(ref pool) = self.pool {
            sqlx::query!(
                r#"
                INSERT INTO cache_entries (key, value, ttl)
                VALUES ($1, $2, $3)
                ON CONFLICT (key)
                DO UPDATE SET
                    value = EXCLUDED.value,
                    ttl = EXCLUDED.ttl,
                    accessed_at = NOW()
                "#,
                key,
                value,
                ttl_seconds
            )
            .execute(pool)
            .await?;
        }
        Ok(())
    }

    pub async fn delete(&self, key: &str) -> Result<()> {
        if let Some(ref pool) = self.pool {
            sqlx::query!(
                "DELETE FROM cache_entries WHERE key = $1",
                key
            )
            .execute(pool)
            .await?;
        }
        Ok(())
    }

    pub async fn clear_expired(&self) -> Result<u64> {
        if let Some(ref pool) = self.pool {
            let result = sqlx::query!(
                r#"
                DELETE FROM cache_entries
                WHERE ttl IS NOT NULL
                AND created_at + (ttl || ' seconds')::INTERVAL < NOW()
                "#,
            )
            .execute(pool)
            .await?;

            Ok(result.rows_affected())
        } else {
            Ok(0)
        }
    }

    pub async fn stats(&self) -> Result<HashMap<String, i64>> {
        let mut stats = HashMap::new();

        if let Some(ref pool) = self.pool {
            let row = sqlx::query!(
                r#"
                SELECT
                    COUNT(*) as count,
                    SUM(LENGTH(value)) as total_size
                FROM cache_entries
                "#,
            )
            .fetch_one(pool)
            .await?;

            stats.insert("entries".to_string(), row.count.unwrap_or(0));
            stats.insert("total_bytes".to_string(), row.total_size.unwrap_or(0) as i64);
        }

        Ok(stats)
    }
}