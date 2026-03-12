use crate::config::Config;
use crate::database::models::*;
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use sqlx::{PgPool, Row};
use std::collections::HashMap;
use tracing::{debug, error, info, warn};
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct Repository {
    pool: PgPool,
    config: Config,
}

impl Repository {
    pub async fn new(config: Config) -> Result<Self> {
        let pool = PgPool::connect(&config.database_url)
            .await
            .context("Failed to connect to database")?;

        // Test the connection
        sqlx::query("SELECT 1").execute(&pool).await
            .context("Failed to ping database")?;

        info!("Database connection established successfully");

        Ok(Self { pool, config })
    }

    pub async fn close(&self) {
        self.pool.close().await;
    }

    pub async fn create_partitioned_table(&self, service_name: &str) -> Result<()> {
        let table_name = format!("outbox_events_{}", service_name.replace("-", "_"));
        
        let query = format!(r#"
            CREATE TABLE IF NOT EXISTS {} (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                service_name VARCHAR(255) NOT NULL,
                event_type VARCHAR(255) NOT NULL,
                event_data JSONB NOT NULL,
                topic VARCHAR(255) NOT NULL,
                correlation_id VARCHAR(255),
                priority INTEGER NOT NULL DEFAULT 5,
                metadata JSONB DEFAULT '{{}}',
                medical_context VARCHAR(50) NOT NULL DEFAULT 'routine',
                created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                published_at TIMESTAMP WITH TIME ZONE,
                retry_count INTEGER NOT NULL DEFAULT 0,
                status VARCHAR(20) NOT NULL DEFAULT 'pending',
                error_message TEXT,
                next_retry_at TIMESTAMP WITH TIME ZONE
            );
            
            CREATE INDEX IF NOT EXISTS idx_{}_status ON {} (status);
            CREATE INDEX IF NOT EXISTS idx_{}_created_at ON {} (created_at);
            CREATE INDEX IF NOT EXISTS idx_{}_priority ON {} (priority DESC);
            CREATE INDEX IF NOT EXISTS idx_{}_medical_context ON {} (medical_context);
            CREATE INDEX IF NOT EXISTS idx_{}_next_retry ON {} (next_retry_at) WHERE next_retry_at IS NOT NULL;
        "#, table_name, table_name, table_name, table_name, table_name, table_name, table_name, table_name, table_name, table_name);

        sqlx::query(&query)
            .execute(&self.pool)
            .await
            .with_context(|| format!("Failed to create partitioned table for service {}", service_name))?;

        info!("Created/verified partitioned table for service: {}", service_name);
        Ok(())
    }

    pub async fn insert_event(&self, event: &OutboxEvent) -> Result<()> {
        let table_name = format!("outbox_events_{}", event.service_name.replace("-", "_"));
        
        // Ensure the partitioned table exists
        self.create_partitioned_table(&event.service_name).await
            .context("Failed to ensure table exists")?;

        let query = format!(r#"
            INSERT INTO {} (
                id, service_name, event_type, event_data, topic, correlation_id,
                priority, metadata, medical_context, created_at, retry_count, status
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        "#, table_name);

        sqlx::query(&query)
            .bind(&event.id)
            .bind(&event.service_name)
            .bind(&event.event_type)
            .bind(&event.event_data)
            .bind(&event.topic)
            .bind(&event.correlation_id)
            .bind(event.priority)
            .bind(&event.metadata)
            .bind(&event.medical_context)
            .bind(event.created_at)
            .bind(event.retry_count)
            .bind(&event.status)
            .execute(&self.pool)
            .await
            .context("Failed to insert event")?;

        debug!("Inserted event {} for service {}", event.id, event.service_name);
        Ok(())
    }

    pub async fn get_pending_events(&self, limit: i64) -> Result<Vec<OutboxEvent>> {
        let mut events = Vec::new();

        // Query all service tables for pending events
        for service_name in &self.config.supported_services {
            let table_name = format!("outbox_events_{}", service_name.replace("-", "_"));
            
            let query = format!(r#"
                SELECT id, service_name, event_type, event_data, topic, correlation_id,
                       priority, metadata, medical_context, created_at, published_at,
                       retry_count, status, error_message, next_retry_at
                FROM {}
                WHERE status IN ('pending', 'failed')
                  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
                ORDER BY 
                    CASE medical_context
                        WHEN 'critical' THEN 1
                        WHEN 'urgent' THEN 2
                        WHEN 'routine' THEN 3
                        ELSE 4
                    END,
                    priority DESC,
                    created_at ASC
                LIMIT $1
            "#, table_name);

            match sqlx::query_as::<_, OutboxEvent>(&query)
                .bind(limit)
                .fetch_all(&self.pool)
                .await
            {
                Ok(service_events) => {
                    events.extend(service_events);
                }
                Err(e) => {
                    // Table might not exist yet, skip
                    debug!("Skipping table {}: {}", table_name, e);
                    continue;
                }
            }
        }

        // Sort events by priority across all services
        events.sort_by(|a, b| {
            a.priority_order()
                .cmp(&b.priority_order())
                .then_with(|| b.priority.cmp(&a.priority))
                .then_with(|| a.created_at.cmp(&b.created_at))
        });

        // Limit total results
        events.truncate(limit as usize);

        Ok(events)
    }

    pub async fn update_event_status(&self, event: &OutboxEvent) -> Result<()> {
        let table_name = format!("outbox_events_{}", event.service_name.replace("-", "_"));
        
        let query = format!(r#"
            UPDATE {} SET
                status = $2,
                published_at = $3,
                retry_count = $4,
                error_message = $5,
                next_retry_at = $6
            WHERE id = $1
        "#, table_name);

        let result = sqlx::query(&query)
            .bind(event.id)
            .bind(&event.status)
            .bind(event.published_at)
            .bind(event.retry_count)
            .bind(&event.error_message)
            .bind(event.next_retry_at)
            .execute(&self.pool)
            .await
            .context("Failed to update event status")?;

        if result.rows_affected() == 0 {
            return Err(anyhow::anyhow!("Event {} not found", event.id));
        }

        Ok(())
    }

    pub async fn get_outbox_stats(&self) -> Result<OutboxStats> {
        let mut stats = OutboxStats::default();

        // Get queue depths and success rates for each service
        for service_name in &self.config.supported_services {
            let table_name = format!("outbox_events_{}", service_name.replace("-", "_"));
            
            // Queue depth
            let queue_query = format!(
                "SELECT COUNT(*) as queue_depth FROM {} WHERE status IN ('pending', 'failed')",
                table_name
            );
            
            let queue_depth: i64 = match sqlx::query(&queue_query)
                .fetch_one(&self.pool)
                .await
            {
                Ok(row) => row.get("queue_depth"),
                Err(_) => 0, // Table might not exist
            };
            
            stats.queue_depths.insert(service_name.clone(), queue_depth);

            // Success rate (last 24 hours)
            let success_query = format!(r#"
                SELECT 
                    COUNT(*) as total,
                    COUNT(*) FILTER (WHERE status = 'published') as successful
                FROM {} 
                WHERE created_at >= NOW() - INTERVAL '24 hours'
            "#, table_name);
            
            match sqlx::query(&success_query)
                .fetch_one(&self.pool)
                .await
            {
                Ok(row) => {
                    let total: i64 = row.get("total");
                    let successful: i64 = row.get("successful");
                    
                    let success_rate = if total > 0 {
                        successful as f64 / total as f64
                    } else {
                        1.0 // No events = 100% success
                    };
                    
                    stats.success_rates.insert(service_name.clone(), success_rate);
                }
                Err(_) => {
                    stats.success_rates.insert(service_name.clone(), 0.0);
                }
            }
        }

        // Get total processed in 24h across all services
        let mut total_processed_24h = 0i64;
        for service_name in &self.config.supported_services {
            let table_name = format!("outbox_events_{}", service_name.replace("-", "_"));
            let total_query = format!(
                "SELECT COUNT(*) as count FROM {} WHERE status = 'published' AND published_at >= NOW() - INTERVAL '24 hours'",
                table_name
            );
            
            if let Ok(row) = sqlx::query(&total_query).fetch_one(&self.pool).await {
                let count: i64 = row.get("count");
                total_processed_24h += count;
            }
        }
        stats.total_processed_24h = total_processed_24h;

        // Get dead letter count across all services
        let mut dead_letter_count = 0i64;
        for service_name in &self.config.supported_services {
            let table_name = format!("outbox_events_{}", service_name.replace("-", "_"));
            let dlq_query = format!(
                "SELECT COUNT(*) as count FROM {} WHERE status = 'dead_letter'",
                table_name
            );
            
            if let Ok(row) = sqlx::query(&dlq_query).fetch_one(&self.pool).await {
                let count: i64 = row.get("count");
                dead_letter_count += count;
            }
        }
        stats.dead_letter_count = dead_letter_count;

        Ok(stats)
    }

    pub async fn health_check(&self) -> HashMap<String, serde_json::Value> {
        let mut health = HashMap::new();
        
        // Test connection
        match sqlx::query("SELECT 1").execute(&self.pool).await {
            Ok(_) => {
                health.insert("status".to_string(), serde_json::Value::String("healthy".to_string()));
                
                // Get connection pool stats
                health.insert("total_connections".to_string(), serde_json::Value::Number(
                    serde_json::Number::from(self.pool.size() as u64)
                ));
                health.insert("idle_connections".to_string(), serde_json::Value::Number(
                    serde_json::Number::from(self.pool.num_idle() as u64)
                ));
            }
            Err(e) => {
                health.insert("status".to_string(), serde_json::Value::String("unhealthy".to_string()));
                health.insert("error".to_string(), serde_json::Value::String(e.to_string()));
            }
        }

        health
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }
}