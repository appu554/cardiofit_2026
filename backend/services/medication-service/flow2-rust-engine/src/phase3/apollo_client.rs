//! Apollo Federation GraphQL Client
//! 
//! Provides versioned knowledge base access via GraphQL Federation.
//! Supports circuit breaker patterns for resilience and caching for performance.

use std::collections::HashMap;
use std::time::{Duration, Instant};
use anyhow::{Result, anyhow};
use serde_json::{Value, json};
use reqwest::{Client, header::HeaderMap};
use tracing::{info, warn, error};

/// Apollo Federation client for versioned KB access
#[derive(Clone)]
pub struct ApolloFederationClient {
    client: Client,
    endpoint_url: String,
    circuit_breaker: CircuitBreaker,
    cache: QueryCache,
}

impl ApolloFederationClient {
    /// Create new Apollo Federation client
    pub fn new(endpoint_url: String) -> Result<Self> {
        let client = Client::builder()
            .timeout(Duration::from_millis(5000)) // 5s timeout
            .build()?;
        
        let circuit_breaker = CircuitBreaker::new(
            5,   // failure_threshold
            Duration::from_secs(60), // recovery_timeout
        );
        
        let cache = QueryCache::new(Duration::from_secs(300)); // 5-minute cache
        
        Ok(Self {
            client,
            endpoint_url,
            circuit_breaker,
            cache,
        })
    }
    
    /// Execute GraphQL query with variables
    pub async fn query(
        &self,
        query: &str,
        variables: Value,
    ) -> Result<Value> {
        // Check circuit breaker
        if !self.circuit_breaker.can_execute().await {
            return Err(anyhow!("Circuit breaker is open"));
        }
        
        // Check cache first
        let cache_key = self.create_cache_key(query, &variables);
        if let Some(cached_result) = self.cache.get(&cache_key).await {
            info!("📋 Cache hit for GraphQL query");
            return Ok(cached_result);
        }
        
        let start_time = Instant::now();
        
        // Prepare GraphQL request
        let graphql_request = json!({
            "query": query,
            "variables": variables
        });
        
        // Execute request
        let response = match self.execute_request(&graphql_request).await {
            Ok(response) => {
                self.circuit_breaker.record_success().await;
                response
            }
            Err(e) => {
                self.circuit_breaker.record_failure().await;
                return Err(e);
            }
        };
        
        let query_duration = start_time.elapsed();
        
        info!(
            "🌐 GraphQL query completed in {:?}",
            query_duration
        );
        
        // Cache successful results
        self.cache.set(cache_key, response.clone()).await;
        
        Ok(response)
    }
    
    /// Execute GraphQL query with custom headers
    pub async fn query_with_headers(
        &self,
        query: &str,
        variables: Value,
        headers: HashMap<String, String>,
    ) -> Result<Value> {
        // Check circuit breaker
        if !self.circuit_breaker.can_execute().await {
            return Err(anyhow!("Circuit breaker is open"));
        }
        
        let start_time = Instant::now();
        
        // Prepare GraphQL request
        let graphql_request = json!({
            "query": query,
            "variables": variables
        });
        
        // Convert headers
        let mut header_map = HeaderMap::new();
        for (key, value) in headers {
            header_map.insert(
                key.parse().map_err(|e| anyhow!("Invalid header key: {}", e))?,
                value.parse().map_err(|e| anyhow!("Invalid header value: {}", e))?,
            );
        }
        
        // Execute request with custom headers
        let response = match self.execute_request_with_headers(&graphql_request, header_map).await {
            Ok(response) => {
                self.circuit_breaker.record_success().await;
                response
            }
            Err(e) => {
                self.circuit_breaker.record_failure().await;
                return Err(e);
            }
        };
        
        let query_duration = start_time.elapsed();
        
        info!(
            "🌐 GraphQL query with headers completed in {:?}",
            query_duration
        );
        
        Ok(response)
    }
    
    /// Health check for Apollo Federation endpoint
    pub async fn health_check(&self) -> Result<bool> {
        let health_query = r#"
            query HealthCheck {
                __schema {
                    queryType {
                        name
                    }
                }
            }
        "#;
        
        match self.query(health_query, json!({})).await {
            Ok(response) => {
                // Check if response contains expected schema information
                if response.get("data").is_some() {
                    info!("✅ Apollo Federation health check passed");
                    Ok(true)
                } else {
                    warn!("⚠️ Apollo Federation health check failed: no data in response");
                    Ok(false)
                }
            }
            Err(e) => {
                error!("❌ Apollo Federation health check failed: {}", e);
                Ok(false)
            }
        }
    }
    
    /// Get available knowledge bases
    pub async fn get_available_knowledge_bases(&self) -> Result<Vec<String>> {
        let introspection_query = r#"
            query GetKnowledgeBases {
                __schema {
                    types {
                        name
                        fields {
                            name
                        }
                    }
                }
            }
        "#;
        
        let response = self.query(introspection_query, json!({})).await?;
        
        let mut kb_names = Vec::new();
        
        if let Some(types) = response
            .get("data")
            .and_then(|d| d.get("__schema"))
            .and_then(|s| s.get("types"))
            .and_then(|t| t.as_array())
        {
            for type_info in types {
                if let Some(type_name) = type_info.get("name").and_then(|n| n.as_str()) {
                    if type_name.starts_with("kb_") {
                        kb_names.push(type_name.to_string());
                    }
                }
            }
        }
        
        Ok(kb_names)
    }
    
    /// Execute HTTP request to GraphQL endpoint
    async fn execute_request(&self, graphql_request: &Value) -> Result<Value> {
        let response = self.client
            .post(&self.endpoint_url)
            .header("Content-Type", "application/json")
            .json(graphql_request)
            .send()
            .await?;
        
        if !response.status().is_success() {
            return Err(anyhow!(
                "GraphQL request failed with status: {}",
                response.status()
            ));
        }
        
        let response_body: Value = response.json().await?;
        
        // Check for GraphQL errors
        if let Some(errors) = response_body.get("errors") {
            return Err(anyhow!(
                "GraphQL errors: {}",
                serde_json::to_string_pretty(errors)?
            ));
        }
        
        Ok(response_body)
    }
    
    /// Execute HTTP request with custom headers
    async fn execute_request_with_headers(
        &self,
        graphql_request: &Value,
        headers: HeaderMap,
    ) -> Result<Value> {
        let mut request_builder = self.client
            .post(&self.endpoint_url)
            .header("Content-Type", "application/json");
        
        // Add custom headers
        for (key, value) in headers.iter() {
            request_builder = request_builder.header(key, value);
        }
        
        let response = request_builder
            .json(graphql_request)
            .send()
            .await?;
        
        if !response.status().is_success() {
            return Err(anyhow!(
                "GraphQL request failed with status: {}",
                response.status()
            ));
        }
        
        let response_body: Value = response.json().await?;
        
        // Check for GraphQL errors
        if let Some(errors) = response_body.get("errors") {
            return Err(anyhow!(
                "GraphQL errors: {}",
                serde_json::to_string_pretty(errors)?
            ));
        }
        
        Ok(response_body)
    }
    
    /// Create cache key from query and variables
    fn create_cache_key(&self, query: &str, variables: &Value) -> String {
        use sha2::{Sha256, Digest};
        
        let combined = format!("{}{}", query, variables.to_string());
        let mut hasher = Sha256::new();
        hasher.update(combined.as_bytes());
        
        format!("{:x}", hasher.finalize())
    }
}

/// Circuit breaker for resilience
struct CircuitBreaker {
    failure_count: std::sync::atomic::AtomicU32,
    failure_threshold: u32,
    recovery_timeout: Duration,
    last_failure_time: std::sync::Mutex<Option<Instant>>,
}

impl CircuitBreaker {
    fn new(failure_threshold: u32, recovery_timeout: Duration) -> Self {
        Self {
            failure_count: std::sync::atomic::AtomicU32::new(0),
            failure_threshold,
            recovery_timeout,
            last_failure_time: std::sync::Mutex::new(None),
        }
    }
    
    async fn can_execute(&self) -> bool {
        let current_failures = self.failure_count.load(std::sync::atomic::Ordering::Relaxed);
        
        if current_failures >= self.failure_threshold {
            // Check if recovery timeout has passed
            if let Ok(last_failure) = self.last_failure_time.lock() {
                if let Some(last_time) = *last_failure {
                    if last_time.elapsed() > self.recovery_timeout {
                        // Reset circuit breaker
                        self.failure_count.store(0, std::sync::atomic::Ordering::Relaxed);
                        info!("🔄 Circuit breaker reset after recovery timeout");
                        return true;
                    }
                }
            }
            
            warn!("🚫 Circuit breaker is open (failures: {})", current_failures);
            return false;
        }
        
        true
    }
    
    async fn record_success(&self) {
        self.failure_count.store(0, std::sync::atomic::Ordering::Relaxed);
    }
    
    async fn record_failure(&self) {
        let new_count = self.failure_count.fetch_add(1, std::sync::atomic::Ordering::Relaxed) + 1;
        
        if let Ok(mut last_failure) = self.last_failure_time.lock() {
            *last_failure = Some(Instant::now());
        }
        
        if new_count >= self.failure_threshold {
            warn!("⚠️ Circuit breaker opened after {} failures", new_count);
        }
    }
}

/// Simple in-memory cache for query results
struct QueryCache {
    cache: std::sync::Arc<std::sync::Mutex<HashMap<String, (Value, Instant)>>>,
    ttl: Duration,
}

impl QueryCache {
    fn new(ttl: Duration) -> Self {
        Self {
            cache: std::sync::Arc::new(std::sync::Mutex::new(HashMap::new())),
            ttl,
        }
    }
    
    async fn get(&self, key: &str) -> Option<Value> {
        if let Ok(cache) = self.cache.lock() {
            if let Some((value, timestamp)) = cache.get(key) {
                if timestamp.elapsed() < self.ttl {
                    return Some(value.clone());
                }
            }
        }
        None
    }
    
    async fn set(&self, key: String, value: Value) {
        if let Ok(mut cache) = self.cache.lock() {
            cache.insert(key, (value, Instant::now()));
            
            // Simple cleanup: remove expired entries if cache gets too large
            if cache.len() > 1000 {
                let now = Instant::now();
                cache.retain(|_, (_, timestamp)| now.duration_since(*timestamp) < self.ttl);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_apollo_client_creation() {
        let client = ApolloFederationClient::new("http://localhost:4000/graphql".to_string());
        assert!(client.is_ok());
    }
    
    #[test]
    fn test_cache_key_generation() {
        let client = ApolloFederationClient::new("http://localhost:4000/graphql".to_string()).unwrap();
        
        let query = "query Test { test }";
        let variables = json!({"var1": "value1"});
        
        let key1 = client.create_cache_key(query, &variables);
        let key2 = client.create_cache_key(query, &variables);
        let key3 = client.create_cache_key("different query", &variables);
        
        assert_eq!(key1, key2);
        assert_ne!(key1, key3);
    }
    
    #[tokio::test]
    async fn test_circuit_breaker() {
        let circuit_breaker = CircuitBreaker::new(3, Duration::from_millis(100));
        
        // Should be able to execute initially
        assert!(circuit_breaker.can_execute().await);
        
        // Record failures up to threshold
        circuit_breaker.record_failure().await;
        circuit_breaker.record_failure().await;
        circuit_breaker.record_failure().await;
        
        // Should be open now
        assert!(!circuit_breaker.can_execute().await);
        
        // Wait for recovery timeout
        tokio::time::sleep(Duration::from_millis(150)).await;
        
        // Should be able to execute again
        assert!(circuit_breaker.can_execute().await);
    }
    
    #[tokio::test]
    async fn test_query_cache() {
        let cache = QueryCache::new(Duration::from_millis(100));
        
        let key = "test_key".to_string();
        let value = json!({"test": "value"});
        
        // Initially empty
        assert!(cache.get(&key).await.is_none());
        
        // Set value
        cache.set(key.clone(), value.clone()).await;
        
        // Should retrieve value
        let retrieved = cache.get(&key).await;
        assert_eq!(retrieved, Some(value));
        
        // Wait for expiration
        tokio::time::sleep(Duration::from_millis(150)).await;
        
        // Should be expired
        assert!(cache.get(&key).await.is_none());
    }
}