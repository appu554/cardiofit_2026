//! Configuration management for the REST API server

use serde::{Deserialize, Serialize};
use std::env;
use std::time::Duration;

/// Server configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub server: ServerSettings,
    pub security: SecuritySettings,
    pub performance: PerformanceSettings,
    pub logging: LoggingSettings,
    pub knowledge_base: KnowledgeBaseSettings,
}

/// Server settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerSettings {
    pub host: String,
    pub port: u16,
    pub workers: Option<usize>,
    pub keep_alive: Duration,
    pub shutdown_timeout: Duration,
}

/// Security settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecuritySettings {
    pub enable_auth: bool,
    pub api_keys: Vec<String>,
    pub cors_origins: Vec<String>,
    pub rate_limit: RateLimitSettings,
    pub request_timeout: Duration,
    pub max_payload_size: usize,
}

/// Rate limiting settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimitSettings {
    pub enabled: bool,
    pub max_requests: usize,
    pub window_duration: Duration,
    pub burst_size: Option<usize>,
}

/// Performance settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceSettings {
    pub enable_compression: bool,
    pub compression_level: u32,
    pub enable_caching: bool,
    pub cache_size: usize,
    pub cache_ttl: Duration,
    pub connection_pool_size: usize,
}

/// Logging settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoggingSettings {
    pub level: String,
    pub format: String,
    pub enable_request_logging: bool,
    pub enable_performance_logging: bool,
    pub log_file: Option<String>,
}

/// Knowledge base settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnowledgeBaseSettings {
    pub path: String,
    pub auto_reload: bool,
    pub reload_interval: Duration,
    pub validation_on_startup: bool,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            server: ServerSettings::default(),
            security: SecuritySettings::default(),
            performance: PerformanceSettings::default(),
            logging: LoggingSettings::default(),
            knowledge_base: KnowledgeBaseSettings::default(),
        }
    }
}

impl Default for ServerSettings {
    fn default() -> Self {
        Self {
            host: "0.0.0.0".to_string(),
            port: 8080,
            workers: None, // Auto-detect based on CPU cores
            keep_alive: Duration::from_secs(75),
            shutdown_timeout: Duration::from_secs(30),
        }
    }
}

impl Default for SecuritySettings {
    fn default() -> Self {
        Self {
            enable_auth: false, // Disabled by default for development
            api_keys: vec!["development-token".to_string()],
            cors_origins: vec!["*".to_string()],
            rate_limit: RateLimitSettings::default(),
            request_timeout: Duration::from_secs(30),
            max_payload_size: 10 * 1024 * 1024, // 10MB
        }
    }
}

impl Default for RateLimitSettings {
    fn default() -> Self {
        Self {
            enabled: true,
            max_requests: 100,
            window_duration: Duration::from_secs(60),
            burst_size: Some(10),
        }
    }
}

impl Default for PerformanceSettings {
    fn default() -> Self {
        Self {
            enable_compression: true,
            compression_level: 6,
            enable_caching: true,
            cache_size: 1000,
            cache_ttl: Duration::from_secs(300), // 5 minutes
            connection_pool_size: 10,
        }
    }
}

impl Default for LoggingSettings {
    fn default() -> Self {
        Self {
            level: "info".to_string(),
            format: "json".to_string(),
            enable_request_logging: true,
            enable_performance_logging: true,
            log_file: None,
        }
    }
}

impl Default for KnowledgeBaseSettings {
    fn default() -> Self {
        Self {
            path: "../knowledge".to_string(),
            auto_reload: false,
            reload_interval: Duration::from_secs(300), // 5 minutes
            validation_on_startup: true,
        }
    }
}

impl ServerConfig {
    /// Load configuration from environment variables
    pub fn from_env() -> Self {
        let mut config = Self::default();

        // Server settings
        if let Ok(host) = env::var("SERVER_HOST") {
            config.server.host = host;
        }
        if let Ok(port) = env::var("SERVER_PORT") {
            config.server.port = port.parse().unwrap_or(config.server.port);
        }
        if let Ok(workers) = env::var("SERVER_WORKERS") {
            config.server.workers = Some(workers.parse().unwrap_or(num_cpus::get()));
        }

        // Security settings
        if let Ok(enable_auth) = env::var("SECURITY_ENABLE_AUTH") {
            config.security.enable_auth = enable_auth.parse().unwrap_or(false);
        }
        if let Ok(api_keys) = env::var("SECURITY_API_KEYS") {
            config.security.api_keys = api_keys.split(',').map(|s| s.trim().to_string()).collect();
        }
        if let Ok(cors_origins) = env::var("SECURITY_CORS_ORIGINS") {
            config.security.cors_origins = cors_origins.split(',').map(|s| s.trim().to_string()).collect();
        }

        // Rate limiting
        if let Ok(rate_limit_enabled) = env::var("RATE_LIMIT_ENABLED") {
            config.security.rate_limit.enabled = rate_limit_enabled.parse().unwrap_or(true);
        }
        if let Ok(max_requests) = env::var("RATE_LIMIT_MAX_REQUESTS") {
            config.security.rate_limit.max_requests = max_requests.parse().unwrap_or(100);
        }

        // Performance settings
        if let Ok(enable_compression) = env::var("PERFORMANCE_ENABLE_COMPRESSION") {
            config.performance.enable_compression = enable_compression.parse().unwrap_or(true);
        }
        if let Ok(enable_caching) = env::var("PERFORMANCE_ENABLE_CACHING") {
            config.performance.enable_caching = enable_caching.parse().unwrap_or(true);
        }

        // Logging settings
        if let Ok(log_level) = env::var("LOG_LEVEL") {
            config.logging.level = log_level;
        }
        if let Ok(log_format) = env::var("LOG_FORMAT") {
            config.logging.format = log_format;
        }
        if let Ok(log_file) = env::var("LOG_FILE") {
            config.logging.log_file = Some(log_file);
        }

        // Knowledge base settings
        if let Ok(kb_path) = env::var("KNOWLEDGE_BASE_PATH") {
            config.knowledge_base.path = kb_path;
        }
        if let Ok(auto_reload) = env::var("KNOWLEDGE_BASE_AUTO_RELOAD") {
            config.knowledge_base.auto_reload = auto_reload.parse().unwrap_or(false);
        }

        config
    }

    /// Load configuration from file
    pub fn from_file(path: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let content = std::fs::read_to_string(path)?;
        let config: Self = match path.ends_with(".yaml") || path.ends_with(".yml") {
            true => serde_yaml::from_str(&content)?,
            false => serde_json::from_str(&content)?,
        };
        Ok(config)
    }

    /// Save configuration to file
    pub fn save_to_file(&self, path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let content = match path.ends_with(".yaml") || path.ends_with(".yml") {
            true => serde_yaml::to_string(self)?,
            false => serde_json::to_string_pretty(self)?,
        };
        std::fs::write(path, content)?;
        Ok(())
    }

    /// Validate configuration
    pub fn validate(&self) -> Result<(), String> {
        // Validate server settings
        if self.server.port == 0 {
            return Err("Server port cannot be 0".to_string());
        }
        if self.server.host.is_empty() {
            return Err("Server host cannot be empty".to_string());
        }

        // Validate security settings
        if self.security.enable_auth && self.security.api_keys.is_empty() {
            return Err("API keys must be provided when authentication is enabled".to_string());
        }
        if self.security.max_payload_size == 0 {
            return Err("Max payload size cannot be 0".to_string());
        }

        // Validate rate limiting
        if self.security.rate_limit.enabled && self.security.rate_limit.max_requests == 0 {
            return Err("Rate limit max requests cannot be 0 when rate limiting is enabled".to_string());
        }

        // Validate knowledge base settings
        if self.knowledge_base.path.is_empty() {
            return Err("Knowledge base path cannot be empty".to_string());
        }

        Ok(())
    }

    /// Get effective number of workers
    pub fn effective_workers(&self) -> usize {
        self.server.workers.unwrap_or_else(num_cpus::get)
    }

    /// Check if development mode is enabled
    pub fn is_development(&self) -> bool {
        !self.security.enable_auth || 
        self.security.api_keys.contains(&"development-token".to_string()) ||
        env::var("RUST_ENV").unwrap_or_default() == "development"
    }

    /// Get log level as tracing level
    pub fn tracing_level(&self) -> tracing::Level {
        match self.logging.level.to_lowercase().as_str() {
            "trace" => tracing::Level::TRACE,
            "debug" => tracing::Level::DEBUG,
            "info" => tracing::Level::INFO,
            "warn" => tracing::Level::WARN,
            "error" => tracing::Level::ERROR,
            _ => tracing::Level::INFO,
        }
    }
}
