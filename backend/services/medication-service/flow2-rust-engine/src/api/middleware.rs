//! Production-grade middleware for the REST API server

use axum::{
    extract::{Request, State},
    http::{HeaderMap, StatusCode},
    middleware::Next,
    response::Response,
    Json,
};
use crate::api::server::ApiState;
use std::time::Instant;
use tracing::{info, warn, error, Span};
use uuid::Uuid;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tokio::time::{Duration, sleep};

/// Request tracking middleware
pub async fn request_tracking_middleware(
    mut request: Request,
    next: Next,
) -> Result<Response, StatusCode> {
    let start_time = Instant::now();
    
    // Generate or extract request ID
    let request_id = request
        .headers()
        .get("x-request-id")
        .and_then(|h| h.to_str().ok())
        .unwrap_or(&Uuid::new_v4().to_string())
        .to_string();

    // Add request ID to headers for downstream services
    request.headers_mut().insert(
        "x-request-id",
        request_id.parse().unwrap(),
    );

    // Create tracing span with request context
    let span = tracing::info_span!(
        "api_request",
        request_id = %request_id,
        method = %request.method(),
        uri = %request.uri(),
    );

    let _enter = span.enter();

    info!("📥 Request started: {} {}", request.method(), request.uri());

    // Process request
    let response = next.run(request).await;

    let duration = start_time.elapsed();
    let status = response.status();

    // Log completion
    if status.is_success() {
        info!("✅ Request completed: {} ({:?})", status, duration);
    } else if status.is_client_error() {
        warn!("⚠️  Request failed (client error): {} ({:?})", status, duration);
    } else {
        error!("❌ Request failed (server error): {} ({:?})", status, duration);
    }

    Ok(response)
}

/// Rate limiting middleware
#[derive(Clone)]
pub struct RateLimiter {
    requests: Arc<Mutex<HashMap<String, Vec<Instant>>>>,
    max_requests: usize,
    window_duration: Duration,
}

impl RateLimiter {
    /// Create a new rate limiter
    pub fn new(max_requests: usize, window_duration: Duration) -> Self {
        Self {
            requests: Arc::new(Mutex::new(HashMap::new())),
            max_requests,
            window_duration,
        }
    }

    /// Check if request is allowed
    pub fn is_allowed(&self, client_id: &str) -> bool {
        let mut requests = self.requests.lock().unwrap();
        let now = Instant::now();
        
        // Get or create request history for this client
        let client_requests = requests.entry(client_id.to_string()).or_insert_with(Vec::new);
        
        // Remove old requests outside the window
        client_requests.retain(|&request_time| {
            now.duration_since(request_time) < self.window_duration
        });
        
        // Check if under limit
        if client_requests.len() < self.max_requests {
            client_requests.push(now);
            true
        } else {
            false
        }
    }

    /// Clean up old entries periodically
    pub async fn cleanup_task(&self) {
        let mut interval = tokio::time::interval(Duration::from_secs(60));
        
        loop {
            interval.tick().await;
            
            let mut requests = self.requests.lock().unwrap();
            let now = Instant::now();
            
            // Remove clients with no recent requests
            requests.retain(|_, client_requests| {
                client_requests.retain(|&request_time| {
                    now.duration_since(request_time) < self.window_duration * 2
                });
                !client_requests.is_empty()
            });
        }
    }
}

/// Rate limiting middleware function
pub async fn rate_limiting_middleware(
    headers: HeaderMap,
    State(rate_limiter): State<RateLimiter>,
    request: Request,
    next: Next,
) -> Result<Response, (StatusCode, Json<serde_json::Value>)> {
    // Extract client identifier (IP address or API key)
    let client_id = headers
        .get("x-forwarded-for")
        .or_else(|| headers.get("x-real-ip"))
        .and_then(|h| h.to_str().ok())
        .unwrap_or("unknown")
        .to_string();

    if !rate_limiter.is_allowed(&client_id) {
        warn!("🚫 Rate limit exceeded for client: {}", client_id);
        return Err((
            StatusCode::TOO_MANY_REQUESTS,
            Json(serde_json::json!({
                "error": "Rate limit exceeded",
                "message": "Too many requests. Please try again later.",
                "retry_after": "60 seconds"
            }))
        ));
    }

    Ok(next.run(request).await)
}

/// Authentication middleware
pub async fn auth_middleware(
    State(state): State<ApiState>,
    headers: HeaderMap,
    request: Request,
    next: Next,
) -> Result<Response, (StatusCode, Json<serde_json::Value>)> {
    // Skip auth for health endpoints
    let path = request.uri().path();
    if path.starts_with("/health") || path.starts_with("/metrics") || path.starts_with("/status") {
        return Ok(next.run(request).await);
    }

    // Check if authentication is enabled in configuration
    if !state.config.security.enable_auth {
        // Authentication is disabled, allow all requests
        return Ok(next.run(request).await);
    }

    // Check for API key or authorization header
    let auth_header = headers
        .get("authorization")
        .or_else(|| headers.get("x-api-key"))
        .and_then(|h| h.to_str().ok());

    match auth_header {
        Some(token) => {
            // Validate token against configured API keys
            if state.config.security.api_keys.contains(&token.to_string()) || validate_token(token) {
                info!("🔐 Authentication successful");
                Ok(next.run(request).await)
            } else {
                warn!("🚫 Authentication failed: invalid token");
                Err((
                    StatusCode::UNAUTHORIZED,
                    Json(serde_json::json!({
                        "error": "Authentication failed",
                        "message": "Invalid or expired token"
                    }))
                ))
            }
        }
        None => {
            warn!("🚫 Authentication failed: missing token");
            Err((
                StatusCode::UNAUTHORIZED,
                Json(serde_json::json!({
                    "error": "Authentication required",
                    "message": "Missing authorization header or API key"
                }))
            ))
        }
    }
}

/// Validate authentication token (simplified implementation)
fn validate_token(token: &str) -> bool {
    // In production, this would validate against a proper auth service
    // For demo purposes, accept any token that starts with "Bearer " or "ApiKey "
    token.starts_with("Bearer ") || token.starts_with("ApiKey ") || token == "development-token"
}

/// CORS middleware
pub async fn cors_middleware(
    request: Request,
    next: Next,
) -> Result<Response, StatusCode> {
    let mut response = next.run(request).await;
    
    let headers = response.headers_mut();
    headers.insert("Access-Control-Allow-Origin", "*".parse().unwrap());
    headers.insert("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS".parse().unwrap());
    headers.insert("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Request-ID".parse().unwrap());
    headers.insert("Access-Control-Max-Age", "86400".parse().unwrap());
    
    Ok(response)
}

/// Security headers middleware
pub async fn security_headers_middleware(
    request: Request,
    next: Next,
) -> Result<Response, StatusCode> {
    let mut response = next.run(request).await;
    
    let headers = response.headers_mut();
    
    // Security headers
    headers.insert("X-Content-Type-Options", "nosniff".parse().unwrap());
    headers.insert("X-Frame-Options", "DENY".parse().unwrap());
    headers.insert("X-XSS-Protection", "1; mode=block".parse().unwrap());
    headers.insert("Referrer-Policy", "strict-origin-when-cross-origin".parse().unwrap());
    headers.insert("Content-Security-Policy", "default-src 'self'".parse().unwrap());
    
    // API-specific headers
    headers.insert("X-API-Version", "2.0.0".parse().unwrap());
    headers.insert("X-Engine", "rust-recipe-engine".parse().unwrap());
    
    Ok(response)
}

/// Request validation middleware
pub async fn request_validation_middleware(
    headers: HeaderMap,
    request: Request,
    next: Next,
) -> Result<Response, (StatusCode, Json<serde_json::Value>)> {
    // Validate Content-Type for POST requests
    if request.method() == "POST" {
        let content_type = headers
            .get("content-type")
            .and_then(|h| h.to_str().ok())
            .unwrap_or("");

        if !content_type.starts_with("application/json") {
            return Err((
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "error": "Invalid Content-Type",
                    "message": "Content-Type must be application/json for POST requests"
                }))
            ));
        }
    }

    // Validate request size (prevent large payloads)
    if let Some(content_length) = headers.get("content-length") {
        if let Ok(length_str) = content_length.to_str() {
            if let Ok(length) = length_str.parse::<usize>() {
                const MAX_PAYLOAD_SIZE: usize = 10 * 1024 * 1024; // 10MB
                if length > MAX_PAYLOAD_SIZE {
                    return Err((
                        StatusCode::PAYLOAD_TOO_LARGE,
                        Json(serde_json::json!({
                            "error": "Payload too large",
                            "message": format!("Request payload exceeds maximum size of {} bytes", MAX_PAYLOAD_SIZE)
                        }))
                    ));
                }
            }
        }
    }

    Ok(next.run(request).await)
}

/// Timeout middleware
pub async fn timeout_middleware(
    request: Request,
    next: Next,
) -> Result<Response, (StatusCode, Json<serde_json::Value>)> {
    const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);
    
    match tokio::time::timeout(REQUEST_TIMEOUT, next.run(request)).await {
        Ok(response) => Ok(response),
        Err(_) => {
            error!("⏰ Request timeout after {:?}", REQUEST_TIMEOUT);
            Err((
                StatusCode::REQUEST_TIMEOUT,
                Json(serde_json::json!({
                    "error": "Request timeout",
                    "message": "Request took too long to process"
                }))
            ))
        }
    }
}
