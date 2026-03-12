mod config;
mod database;
mod api;
mod publisher;
mod circuit_breaker;
mod services;
mod metrics;

use anyhow::{Context, Result};
use config::Config;
use database::Repository;
use std::sync::Arc;
use tokio::signal;
use tracing::{error, info, warn};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[tokio::main]
async fn main() -> Result<()> {
    // Load configuration
    let config = Config::load().context("Failed to load configuration")?;

    // Setup logging
    setup_logging(&config)?;

    info!("Starting {} v{}", config.project_name, config.version);
    info!("Environment: {}", config.environment);

    // Run the application
    if let Err(e) = run(config).await {
        error!("Application failed: {}", e);
        std::process::exit(1);
    }

    info!("Application shutdown complete");
    Ok(())
}

async fn run(config: Config) -> Result<()> {
    // Initialize database repository
    info!("Initializing database connection...");
    let repo = Repository::new(config.clone()).await
        .context("Failed to initialize database repository")?;
    let repo = Arc::new(repo);

    // Initialize circuit breaker
    info!("Initializing medical circuit breaker...");
    let circuit_breaker = Arc::new(
        circuit_breaker::MedicalCircuitBreaker::new(config.clone())
    );

    // Initialize and start servers
    let http_server = api::http::Server::new(
        repo.clone(),
        circuit_breaker.clone(),
        config.clone(),
    );

    let grpc_server = api::grpc::Server::new(
        repo.clone(),
        circuit_breaker.clone(),
        config.clone(),
    );

    // Start HTTP server
    let http_handle = {
        let server = http_server.clone();
        tokio::spawn(async move {
            if let Err(e) = server.start().await {
                error!("HTTP server error: {}", e);
            }
        })
    };

    // Start gRPC server
    let grpc_handle = {
        let server = grpc_server.clone();
        tokio::spawn(async move {
            if let Err(e) = server.start().await {
                error!("gRPC server error: {}", e);
            }
        })
    };

    // Initialize and start Kafka publisher if enabled
    let publisher_handle = if config.publisher_enabled {
        info!("Initializing Kafka publisher...");
        let publisher = publisher::KafkaPublisher::new(
            repo.clone(),
            circuit_breaker.clone(),
            config.clone(),
        ).await.context("Failed to initialize Kafka publisher")?;

        let publisher = Arc::new(publisher);
        let publisher_clone = publisher.clone();
        
        Some(tokio::spawn(async move {
            if let Err(e) = publisher_clone.start().await {
                error!("Kafka publisher error: {}", e);
            }
        }))
    } else {
        None
    };

    info!("All services started successfully!");
    info!("HTTP server listening on {}:{}", config.host, config.port);
    info!("gRPC server listening on {}:{}", config.host, config.grpc_port);

    // Wait for shutdown signal
    match signal::ctrl_c().await {
        Ok(()) => {
            info!("Received shutdown signal, initiating graceful shutdown...");
        }
        Err(err) => {
            error!("Unable to listen for shutdown signal: {}", err);
        }
    }

    // Graceful shutdown
    info!("Shutting down services...");

    // Stop HTTP server
    http_server.stop().await;

    // Stop gRPC server
    grpc_server.stop().await;

    // Stop Kafka publisher
    if let Some(handle) = publisher_handle {
        handle.abort();
        if let Err(e) = handle.await {
            if !e.is_cancelled() {
                warn!("Error stopping Kafka publisher: {}", e);
            }
        }
    }

    // Stop background tasks
    http_handle.abort();
    grpc_handle.abort();

    // Close database connections
    repo.close().await;

    info!("Graceful shutdown completed");
    Ok(())
}

fn setup_logging(config: &Config) -> Result<()> {
    let log_level = match config.log_level.to_lowercase().as_str() {
        "trace" => tracing::Level::TRACE,
        "debug" => tracing::Level::DEBUG,
        "info" => tracing::Level::INFO,
        "warn" => tracing::Level::WARN,
        "error" => tracing::Level::ERROR,
        _ => tracing::Level::INFO,
    };

    if config.is_production() {
        // JSON logging for production
        tracing_subscriber::registry()
            .with(
                tracing_subscriber::EnvFilter::try_from_default_env()
                    .unwrap_or_else(|_| format!("{}={}", env!("CARGO_PKG_NAME"), log_level).into()),
            )
            .with(tracing_subscriber::fmt::layer().json())
            .try_init()
            .context("Failed to initialize JSON logger")?;
    } else {
        // Pretty logging for development
        tracing_subscriber::registry()
            .with(
                tracing_subscriber::EnvFilter::try_from_default_env()
                    .unwrap_or_else(|_| format!("{}={}", env!("CARGO_PKG_NAME"), log_level).into()),
            )
            .with(tracing_subscriber::fmt::layer().pretty())
            .try_init()
            .context("Failed to initialize pretty logger")?;
    }

    Ok(())
}