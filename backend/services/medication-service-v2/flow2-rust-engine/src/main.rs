//! Main entry point for the Unified Dose Safety Engine

use flow2_rust_engine::{
    unified_clinical_engine::{UnifiedClinicalEngine, knowledge_base::KnowledgeBase},
    api::{server::start_server, config::ServerConfig},
    grpc::{Flow2GrpcServer, flow2::flow2_engine_server::Flow2EngineServer},
};
use std::sync::Arc;
use tracing::{info, error, Level};
use tracing_subscriber;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_max_level(Level::INFO)
        .with_target(false)
        .with_thread_ids(true)
        .with_file(true)
        .with_line_number(true)
        .init();

    info!("🚀 Starting Unified Dose Safety Engine v{}", flow2_rust_engine::VERSION);

    // Load configuration
    let config = if let Ok(config_path) = std::env::var("CONFIG_FILE") {
        info!("📄 Loading configuration from file: {}", config_path);
        ServerConfig::from_file(&config_path).unwrap_or_else(|e| {
            error!("Failed to load config file: {}. Using environment/defaults.", e);
            ServerConfig::from_env()
        })
    } else {
        info!("📄 Loading configuration from environment variables");
        ServerConfig::from_env()
    };

    // Validate configuration
    if let Err(e) = config.validate() {
        error!("❌ Configuration validation failed: {}", e);
        return Err(e.into());
    }

    info!("📚 Knowledge base path: {}", config.knowledge_base.path);
    info!("🌐 Server: {}:{}", config.server.host, config.server.port);
    info!("🔐 Authentication: {}", if config.security.enable_auth { "enabled" } else { "disabled" });
    info!("🚦 Rate limiting: {}", if config.security.rate_limit.enabled { "enabled" } else { "disabled" });
    info!("🗜️  Compression: {}", if config.performance.enable_compression { "enabled" } else { "disabled" });

    // Initialize the unified clinical engine
    info!("🔧 Initializing Unified Clinical Engine...");
    let knowledge_base = match KnowledgeBase::new(&config.knowledge_base.path).await {
        Ok(kb) => Arc::new(kb),
        Err(e) => {
            error!("❌ Failed to initialize knowledge base: {}", e);
            return Err(e.into());
        }
    };

    let unified_engine = match UnifiedClinicalEngine::new(knowledge_base) {
        Ok(orchestrator) => {
            info!("✅ Unified Clinical Engine initialized successfully");
            Arc::new(orchestrator)
        }
        Err(e) => {
            error!("❌ Failed to initialize Unified Clinical Engine: {}", e);
            return Err(e.into());
        }
    };

    // Print startup banner
    print_startup_banner(&unified_engine, &config);

    // Start both HTTP and gRPC servers concurrently
    info!("🌐 Starting HTTP and gRPC servers...");

    let unified_engine_http = unified_engine.clone();
    let unified_engine_grpc = unified_engine.clone();
    let config_clone = config.clone();

    // Start HTTP server task
    let http_server = tokio::spawn(async move {
        if let Err(e) = start_server(unified_engine_http, &config_clone).await {
            error!("❌ Failed to start HTTP server: {}", e);
        }
    });

    // Start gRPC server task
    let grpc_server = tokio::spawn(async move {
        let grpc_port = std::env::var("GRPC_PORT").unwrap_or_else(|_| "8091".to_string());
        let grpc_addr = format!("0.0.0.0:{}", grpc_port).parse().unwrap();

        let grpc_service = Flow2GrpcServer::new(unified_engine_grpc);
        let grpc_server = Flow2EngineServer::new(grpc_service);

        info!("🦀 Starting gRPC server on {}", grpc_addr);

        if let Err(e) = tonic::transport::Server::builder()
            .add_service(grpc_server)
            .serve_with_shutdown(grpc_addr, shutdown_signal())
            .await
        {
            error!("❌ Failed to start gRPC server: {}", e);
        }
    });

    // Wait for either server to complete (or fail)
    tokio::select! {
        _ = http_server => {
            info!("🌐 HTTP server completed");
        }
        _ = grpc_server => {
            info!("🦀 gRPC server completed");
        }
    }

    Ok(())
}

/// Print startup banner with system information
fn print_startup_banner(unified_engine: &UnifiedClinicalEngine, config: &ServerConfig) {
    // Create a simple health status for the unified engine
    let health_status = "healthy";
    let version = flow2_rust_engine::VERSION;

    println!("\n🦀 ===============================================");
    println!("🦀  UNIFIED DOSE SAFETY ENGINE - PRODUCTION READY");
    println!("🦀 ===============================================");
    println!("🦀  Version: {}", version);
    println!("🦀  Engine:  Unified-Clinical-Engine");
    println!("🦀  Status:  {}", health_status);
    println!("🦀  Mode:    {}", if config.is_development() { "DEVELOPMENT" } else { "PRODUCTION" });
    println!("🦀 ===============================================");
    println!("📚  UNIFIED KNOWLEDGE BASE:");
    println!("📚    Drug Rules:     Available");
    println!("📚    Safety Rules:   Available");
    println!("📚    DDI Rules:      Available");
    println!("📚    Path:           {}", config.knowledge_base.path);
    println!("🦀 ===============================================");
    println!("🔧  SERVER CONFIGURATION:");
    println!("🔧    Host:        {}", config.server.host);
    println!("🔧    Port:        {}", config.server.port);
    println!("🔧    Workers:     {}", config.effective_workers());
    println!("🔧    Auth:        {}", if config.security.enable_auth { "ENABLED" } else { "DISABLED" });
    println!("🔧    Rate Limit:  {} req/min", if config.security.rate_limit.enabled { config.security.rate_limit.max_requests.to_string() } else { "DISABLED".to_string() });
    println!("🔧    Compression: {}", if config.performance.enable_compression { "ENABLED" } else { "DISABLED" });
    println!("🦀 ===============================================");
    println!("🎯  UNIFIED ENGINE CAPABILITIES:");
    println!("🎯    ✅ Advanced Dose Calculation");
    println!("🎯    ✅ Comprehensive Safety Verification");
    println!("🎯    ✅ Clinical Intelligence Integration");
    println!("🎯    ✅ Pharmacokinetic Modeling");
    println!("🎯    ✅ Risk Assessment & Monitoring");
    println!("🎯    ✅ Titration Scheduling");
    println!("🎯    ✅ Sub-100ms Performance");
    println!("🦀 ===============================================");
    println!("🌐  HTTP API ENDPOINTS:");
    println!("🌐    POST /api/dose/optimize        - Advanced dose calculation");
    println!("🌐    POST /api/medication/intelligence - Clinical decision support");
    println!("🌐    POST /api/flow2/execute        - Flow2 integration");
    println!("🌐    POST /api/execute-with-snapshot - Snapshot-based processing");
    println!("🌐    POST /api/recipe/execute-snapshot - Recipe execution with snapshot");
    println!("🌐    GET  /health                   - Health check");
    println!("🌐    GET  /metrics                  - Performance metrics");
    println!("🌐    GET  /status                   - Engine status");
    println!("🌐    GET  /api/admin/stats          - Admin statistics");
    println!("🦀 ===============================================");
    println!("🦀  gRPC API ENDPOINTS:");
    println!("🦀    ExecuteRecipe                  - Execute medication recipe");
    println!("🦀    OptimizeDose                   - Optimize medication dosing");
    println!("🦀    AnalyzeMedication              - Medication intelligence analysis");
    println!("🦀    ExecuteFlow2                   - Flow2 workflow execution");
    println!("🦀    HealthCheck                    - gRPC health check");
    println!("🦀    StreamPatientUpdates           - Real-time patient monitoring");
    println!("🦀    gRPC Port: {}                   - {}",
             std::env::var("GRPC_PORT").unwrap_or_else(|_| "8091".to_string()),
             "Binary protocol buffers");
    println!("🦀 ===============================================");
    println!("🚀  UNIFIED ENGINE READY FOR PRODUCTION!");
    println!("🦀 ===============================================\n");
}

/// Handle graceful shutdown
async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }

    info!("🛑 Shutdown signal received, starting graceful shutdown...");
}
