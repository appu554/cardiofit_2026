// Clinical Data Hub Rust Service - Ultra-high performance clinical data intelligence
use anyhow::{Context, Result};
use clap::Parser;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::signal;
use tracing::{info, warn, error};

mod api;
mod models;
mod cache;
mod services;
mod proto {
    tonic::include_proto!("clinical_data_hub");
}

/// Clinical Data Hub configuration
#[derive(Parser, Debug)]
#[command(name = "clinical-data-hub-rust")]
#[command(about = "Ultra-high performance clinical data intelligence hub")]
struct Args {
    /// gRPC server port
    #[arg(long, default_value = "8018", env = "GRPC_PORT")]
    grpc_port: u16,
    
    /// HTTP server port for metrics and health checks
    #[arg(long, default_value = "8118", env = "HTTP_PORT")]
    http_port: u16,
    
    /// Redis cluster addresses (comma-separated)
    #[arg(long, default_value = "localhost:6379", env = "REDIS_ADDRS")]
    redis_addrs: String,
    
    /// PostgreSQL connection string
    #[arg(long, default_value = "postgresql://localhost:5432/clinical_data_hub", env = "POSTGRES_URL")]
    postgres_url: String,
    
    /// Kafka brokers (comma-separated)
    #[arg(long, default_value = "localhost:9092", env = "KAFKA_BROKERS")]
    kafka_brokers: String,
    
    /// Environment (development, staging, production)
    #[arg(long, default_value = "development", env = "ENVIRONMENT")]
    environment: String,
    
    /// Log level
    #[arg(long, default_value = "info", env = "LOG_LEVEL")]
    log_level: String,
    
    /// Maximum memory usage for L1 cache (MB)
    #[arg(long, default_value = "512", env = "L1_CACHE_SIZE_MB")]
    l1_cache_size_mb: usize,
    
    /// Enable performance profiling
    #[arg(long, default_value = "false", env = "ENABLE_PROFILING")]
    enable_profiling: bool,
}

#[global_allocator]
static GLOBAL: mimalloc::MiMalloc = mimalloc::MiMalloc;

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();
    
    // Initialize tracing
    init_tracing(&args.log_level, &args.environment)?;
    
    info!("🚀 Starting Clinical Data Hub Rust Service");
    info!("   Environment: {}", args.environment);
    info!("   gRPC Port: {}", args.grpc_port);
    info!("   HTTP Port: {}", args.http_port);
    info!("   Redis: {}", args.redis_addrs);
    info!("   PostgreSQL: {}", args.postgres_url);
    info!("   Kafka: {}", args.kafka_brokers);
    info!("   L1 Cache Size: {} MB", args.l1_cache_size_mb);
    
    // Initialize services
    let app_state = initialize_services(&args).await?;
    
    // Start HTTP server for metrics and health checks
    let http_server = start_http_server(args.http_port, app_state.clone());
    
    // Start gRPC server
    let grpc_server = start_grpc_server(args.grpc_port, app_state.clone());
    
    info!("🎉 Clinical Data Hub Rust Service is ready!");
    info!("   📊 Health Check: http://localhost:{}/health", args.http_port);
    info!("   📈 Metrics: http://localhost:{}/metrics", args.http_port);
    info!("   🔧 gRPC: localhost:{}", args.grpc_port);
    
    // Wait for shutdown signal
    tokio::select! {
        result = http_server => {
            if let Err(e) = result {
                error!("HTTP server error: {}", e);
            }
        }
        result = grpc_server => {
            if let Err(e) = result {
                error!("gRPC server error: {}", e);
            }
        }
        _ = signal::ctrl_c() => {
            info!("🛑 Received shutdown signal");
        }
    }
    
    // Graceful shutdown
    info!("🔄 Starting graceful shutdown...");
    
    // Flush cache and close connections
    if let Err(e) = app_state.shutdown().await {
        error!("Error during shutdown: {}", e);
    }
    
    info!("✅ Clinical Data Hub Rust Service stopped gracefully");
    Ok(())
}

/// Initialize tracing/logging
fn init_tracing(log_level: &str, environment: &str) -> Result<()> {
    use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};
    
    let env_filter = EnvFilter::try_new(log_level)
        .context("Failed to parse log level")?;
    
    if environment == "production" {
        // JSON logging for production
        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_subscriber::fmt::layer().json())
            .init();
    } else {
        // Pretty logging for development
        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_subscriber::fmt::layer().pretty())
            .init();
    }
    
    Ok(())
}

/// Application state container
#[derive(Clone)]
pub struct AppState {
    cache_manager: Arc<cache::manager::CacheManager>,
    data_aggregator: Arc<services::DataAggregator>,
    performance_monitor: Arc<services::PerformanceMonitor>,
    stream_processor: Arc<services::StreamProcessor>,
}

impl AppState {
    /// Graceful shutdown of all services
    async fn shutdown(&self) -> Result<()> {
        info!("Shutting down cache manager...");
        self.cache_manager.shutdown().await?;
        
        info!("Shutting down data aggregator...");
        self.data_aggregator.shutdown().await?;
        
        info!("Shutting down stream processor...");
        self.stream_processor.shutdown().await?;
        
        Ok(())
    }
}

/// Initialize all services and connections
async fn initialize_services(args: &Args) -> Result<AppState> {
    info!("🔧 Initializing services...");
    
    // Parse Redis addresses
    let redis_addrs: Vec<String> = args.redis_addrs
        .split(',')
        .map(|s| s.trim().to_string())
        .collect();
    
    // Parse Kafka brokers
    let kafka_brokers: Vec<String> = args.kafka_brokers
        .split(',')
        .map(|s| s.trim().to_string())
        .collect();
    
    // Initialize cache manager
    info!("📦 Initializing cache manager...");
    let cache_config = cache::manager::CacheManagerConfig {
        l1_max_size_mb: args.l1_cache_size_mb,
        l2_redis_addrs: redis_addrs,
        l3_postgres_url: args.postgres_url.clone(),
        enable_compression: true,
        enable_metrics: true,
    };
    let cache_manager = Arc::new(
        cache::manager::CacheManager::new(cache_config).await
            .context("Failed to initialize cache manager")?
    );
    
    // Initialize data aggregator
    info!("🔄 Initializing data aggregator...");
    let data_aggregator = Arc::new(
        services::DataAggregator::new(cache_manager.clone()).await
            .context("Failed to initialize data aggregator")?
    );
    
    // Initialize performance monitor
    info!("📊 Initializing performance monitor...");
    let performance_monitor = Arc::new(
        services::PerformanceMonitor::new()
    );
    
    // Initialize stream processor
    info!("🌊 Initializing stream processor...");
    let stream_processor = Arc::new(
        services::StreamProcessor::new(kafka_brokers, cache_manager.clone()).await
            .context("Failed to initialize stream processor")?
    );
    
    info!("✅ All services initialized successfully");
    
    Ok(AppState {
        cache_manager,
        data_aggregator,
        performance_monitor,
        stream_processor,
    })
}

/// Start HTTP server for metrics and health checks
async fn start_http_server(port: u16, app_state: AppState) -> Result<()> {
    use warp::Filter;
    
    let health = warp::path("health")
        .map(move || {
            warp::reply::json(&serde_json::json!({
                "status": "healthy",
                "service": "clinical-data-hub-rust",
                "timestamp": chrono::Utc::now().to_rfc3339()
            }))
        });
    
    let ready = warp::path("ready")
        .map(move || {
            // TODO: Add readiness checks
            warp::reply::json(&serde_json::json!({
                "ready": true,
                "service": "clinical-data-hub-rust",
                "timestamp": chrono::Utc::now().to_rfc3339()
            }))
        });
    
    let state_for_metrics = app_state.clone();
    let metrics = warp::path("metrics")
        .and_then(move || {
            let state = state_for_metrics.clone();
            async move {
                match state.performance_monitor.get_prometheus_metrics().await {
                    Ok(metrics) => Ok(warp::reply::with_header(
                        metrics,
                        "content-type",
                        "text/plain; version=0.0.4; charset=utf-8"
                    )),
                    Err(e) => {
                        error!("Failed to get metrics: {}", e);
                        Err(warp::reject())
                    }
                }
            }
        });
    
    // Add federation endpoints for Apollo Federation integration
    let federation_routes = api::federation::federation_routes();
    
    let routes = health.or(ready).or(metrics).or(federation_routes);
    
    let addr: SocketAddr = ([0, 0, 0, 0], port).into();
    info!("🌐 Starting HTTP server on {}", addr);
    
    warp::serve(routes)
        .run(addr)
        .await;
    
    Ok(())
}

/// Start gRPC server
async fn start_grpc_server(port: u16, app_state: AppState) -> Result<()> {
    use tonic::transport::Server;
    use proto::clinical_data_hub_server::ClinicalDataHubServer;
    
    let service = services::ClinicalDataHubService::new(
        app_state.cache_manager,
        app_state.data_aggregator,
        app_state.performance_monitor,
        app_state.stream_processor,
    );
    
    let addr: SocketAddr = ([0, 0, 0, 0], port).into();
    info!("🔌 Starting gRPC server on {}", addr);
    
    Server::builder()
        .add_service(ClinicalDataHubServer::new(service))
        .serve(addr)
        .await
        .context("gRPC server failed")?;
    
    Ok(())
}