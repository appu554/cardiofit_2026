use warp::{Filter, Reply, Rejection, reject};
use std::sync::Arc;
use serde_json::json;

use crate::services::ClinicalDataHubService;

/// HTTP service for health checks and metrics
pub fn routes(service: Arc<ClinicalDataHubService>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    health()
        .or(ready())
        .or(metrics(service.clone()))
        .or(federation(service))
}

/// Health check endpoint
fn health() -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path("health")
        .and(warp::get())
        .map(|| {
            let response = json!({
                "status": "healthy",
                "service": "clinical-data-hub-rust",
                "timestamp": chrono::Utc::now().to_rfc3339()
            });
            warp::reply::json(&response)
        })
}

/// Readiness probe endpoint
fn ready() -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path("ready")
        .and(warp::get())
        .map(|| {
            let response = json!({
                "status": "ready",
                "service": "clinical-data-hub-rust",
                "timestamp": chrono::Utc::now().to_rfc3339()
            });
            warp::reply::json(&response)
        })
}

/// Metrics endpoint
fn metrics(service: Arc<ClinicalDataHubService>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path("metrics")
        .and(warp::get())
        .and(with_service(service))
        .and_then(handle_metrics)
}

/// Federation GraphQL endpoint
fn federation(service: Arc<ClinicalDataHubService>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path("api")
        .and(warp::path("federation"))
        .and(warp::get().or(warp::post()).unify())
        .and(with_service(service))
        .and_then(handle_federation)
}

fn with_service(service: Arc<ClinicalDataHubService>) -> impl Filter<Extract = (Arc<ClinicalDataHubService>,), Error = std::convert::Infallible> + Clone {
    warp::any().map(move || service.clone())
}

async fn handle_metrics(service: Arc<ClinicalDataHubService>) -> Result<impl Reply, Rejection> {
    match service.get_performance_metrics().await {
        Ok(metrics) => {
            let prometheus_format = format!(
                "# HELP clinical_data_hub_cache_hit_rate Cache hit rate\n\
                 # TYPE clinical_data_hub_cache_hit_rate gauge\n\
                 clinical_data_hub_cache_hit_rate {{}} {}\n\
                 # HELP clinical_data_hub_response_time_ms Average response time in milliseconds\n\
                 # TYPE clinical_data_hub_response_time_ms gauge\n\
                 clinical_data_hub_response_time_ms {{}} {}\n\
                 # HELP clinical_data_hub_throughput Throughput per second\n\
                 # TYPE clinical_data_hub_throughput gauge\n\
                 clinical_data_hub_throughput {{}} {}\n",
                metrics.cache_hit_rate,
                metrics.average_response_time_ms,
                metrics.throughput_per_second
            );
            Ok(warp::reply::with_header(
                prometheus_format,
                "Content-Type",
                "text/plain; version=0.0.4",
            ))
        }
        Err(_) => Err(reject::reject()),
    }
}

async fn handle_federation(service: Arc<ClinicalDataHubService>) -> Result<impl Reply, Rejection> {
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