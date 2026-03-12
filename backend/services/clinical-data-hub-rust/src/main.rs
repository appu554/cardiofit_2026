// Simplified Clinical Data Hub Rust Service - HTTP only for port 8118
use anyhow::{Context, Result};
use clap::Parser;
use serde_json::json;
use std::convert::Infallible;
use std::net::SocketAddr;
use tokio::signal;
use tracing::{info, error};
use warp::Filter;

/// Clinical Data Hub configuration
#[derive(Parser, Debug)]
#[command(name = "clinical-data-hub-rust")]
#[command(about = "Clinical Data Hub HTTP Service")]
struct Args {
    /// HTTP server port
    #[arg(long, default_value = "8118", env = "HTTP_PORT")]
    http_port: u16,

    /// Environment (development, staging, production)
    #[arg(long, default_value = "development", env = "ENVIRONMENT")]
    environment: String,

    /// Log level
    #[arg(long, default_value = "info", env = "LOG_LEVEL")]
    log_level: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();

    // Initialize tracing
    init_tracing(&args.log_level, &args.environment)?;

    info!("🚀 Starting Clinical Data Hub Rust Service (HTTP Only)");
    info!("   Environment: {}", args.environment);
    info!("   HTTP Port: {}", args.http_port);

    // Start HTTP server
    let http_server = start_http_server(args.http_port);

    info!("🎉 Clinical Data Hub Service is ready!");
    info!("   📊 Health Check: http://localhost:{}/health", args.http_port);
    info!("   📈 Metrics: http://localhost:{}/metrics", args.http_port);

    // Wait for shutdown signal
    tokio::select! {
        result = http_server => {
            if let Err(e) = result {
                error!("HTTP server error: {}", e);
            }
        }
        _ = signal::ctrl_c() => {
            info!("🛑 Received shutdown signal");
        }
    }

    info!("✅ Clinical Data Hub Service stopped gracefully");
    Ok(())
}

/// Initialize tracing/logging
fn init_tracing(log_level: &str, environment: &str) -> Result<()> {
    use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

    let env_filter = EnvFilter::try_new(log_level)
        .context("Failed to parse log level")?;

    if environment == "production" {
        // Structured logging for production
        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_subscriber::fmt::layer())
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

/// Start HTTP server for health checks and metrics
async fn start_http_server(port: u16) -> Result<()> {
    let health = warp::path("health")
        .and(warp::get())
        .and_then(health_handler);

    let ready = warp::path("ready")
        .and(warp::get())
        .and_then(ready_handler);

    let metrics = warp::path("metrics")
        .and(warp::get())
        .and_then(metrics_handler);

    // Federation endpoint for Apollo Federation integration
    let federation_get = warp::path("api")
        .and(warp::path("federation"))
        .and(warp::get())
        .and_then(federation_sdl_handler);

    let federation_post = warp::path("api")
        .and(warp::path("federation"))
        .and(warp::post())
        .and(warp::body::json())
        .and_then(federation_graphql_handler);

    let federation = federation_get.or(federation_post);

    let routes = health.or(ready).or(metrics).or(federation);

    let addr: SocketAddr = ([0, 0, 0, 0], port).into();
    info!("🌐 Starting HTTP server on {}", addr);

    warp::serve(routes)
        .run(addr)
        .await;

    Ok(())
}

async fn health_handler() -> Result<impl warp::Reply, Infallible> {
    let response = json!({
        "status": "healthy",
        "service": "clinical-data-hub-rust",
        "version": "1.0.0",
        "timestamp": chrono::Utc::now().to_rfc3339(),
        "uptime_seconds": 0
    });
    Ok(warp::reply::json(&response))
}

async fn ready_handler() -> Result<impl warp::Reply, Infallible> {
    let response = json!({
        "ready": true,
        "service": "clinical-data-hub-rust",
        "checks": {
            "http_server": "ok",
            "memory": "ok"
        },
        "timestamp": chrono::Utc::now().to_rfc3339()
    });
    Ok(warp::reply::json(&response))
}

async fn metrics_handler() -> Result<impl warp::Reply, Infallible> {
    let metrics = format!(
        "# HELP clinical_data_hub_uptime_seconds Total uptime in seconds\n\
         # TYPE clinical_data_hub_uptime_seconds counter\n\
         clinical_data_hub_uptime_seconds {}\n\
         # HELP clinical_data_hub_requests_total Total HTTP requests\n\
         # TYPE clinical_data_hub_requests_total counter\n\
         clinical_data_hub_requests_total {}\n\
         # HELP clinical_data_hub_memory_usage_bytes Memory usage in bytes\n\
         # TYPE clinical_data_hub_memory_usage_bytes gauge\n\
         clinical_data_hub_memory_usage_bytes {}\n",
        0, 0, 0
    );

    Ok(warp::reply::with_header(
        metrics,
        "content-type",
        "text/plain; version=0.0.4; charset=utf-8"
    ))
}

// GraphQL query structure
#[derive(serde::Deserialize, Debug)]
struct GraphQLRequest {
    query: String,
    variables: Option<serde_json::Value>,
    operation_name: Option<String>,
}

async fn federation_sdl_handler() -> Result<impl warp::Reply, Infallible> {
    let sdl = r#"
        directive @key(fields: String!) on OBJECT | INTERFACE
        directive @external on FIELD_DEFINITION | OBJECT

        scalar JSON
        scalar DateTime

        type ClinicalData @key(fields: "patientId") {
            patientId: ID!
            aggregatedData: JSON
            cacheLayer: String
            lastUpdated: DateTime
        }

        type Query {
            _entities(representations: [JSON!]!): [ClinicalData]!
            _service: _Service!
        }

        type _Service {
            sdl: String
        }
    "#;

    let response = json!({
        "data": {
            "_service": {
                "sdl": sdl
            }
        }
    });

    Ok(warp::reply::json(&response))
}

async fn federation_graphql_handler(request: GraphQLRequest) -> Result<impl warp::Reply, Infallible> {
    info!("Received GraphQL query: {}", request.query.chars().take(100).collect::<String>());

    // Handle different GraphQL queries
    if request.query.contains("_service") {
        // Handle _service query for schema introspection
        let sdl = r#"
        directive @key(fields: String!) on OBJECT | INTERFACE
        directive @external on FIELD_DEFINITION | OBJECT

        scalar JSON
        scalar DateTime

        type ClinicalData @key(fields: "patientId") {
            patientId: ID!
            aggregatedData: JSON
            cacheLayer: String
            lastUpdated: DateTime
        }

        type Query {
            _entities(representations: [JSON!]!): [ClinicalData]!
            _service: _Service!
        }

        type _Service {
            sdl: String
        }
    "#;

        let response = json!({
            "data": {
                "_service": {
                    "sdl": sdl
                }
            }
        });

        Ok(warp::reply::json(&response))
    } else if request.query.contains("_entities") {
        // Handle entity resolution queries
        let response = json!({
            "data": {
                "_entities": []
            }
        });

        Ok(warp::reply::json(&response))
    } else if request.query.contains("__schema") || request.query.contains("IntrospectionQuery") {
        // Handle schema introspection
        let response = json!({
            "data": {
                "__schema": {
                    "queryType": {"name": "Query"},
                    "mutationType": null,
                    "subscriptionType": null,
                    "types": []
                }
            }
        });

        Ok(warp::reply::json(&response))
    } else {
        // Handle unknown queries
        let response = json!({
            "errors": [{
                "message": "Query not supported",
                "extensions": {
                    "code": "QUERY_NOT_SUPPORTED"
                }
            }]
        });

        Ok(warp::reply::json(&response))
    }
}