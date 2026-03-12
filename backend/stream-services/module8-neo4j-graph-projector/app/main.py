"""
FastAPI application for Neo4j Graph Projector Service
Provides health checks, metrics, status, and graph query endpoints
"""
import os
import sys
from pathlib import Path
from contextlib import asynccontextmanager
from datetime import datetime

import structlog
from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import PlainTextResponse

# Configure structured logging
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer()
    ]
)

logger = structlog.get_logger(__name__)

# Add shared module to path
shared_module_path = Path(__file__).parent.parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from app.services import KafkaConsumerService
from app.models import (
    HealthResponse,
    MetricsResponse,
    StatusResponse,
    ErrorResponse,
)
from app.config import (
    KAFKA_CONFIG,
    NEO4J_CONFIG,
    SERVICE_PORT,
    SERVICE_HOST,
    TOPICS,
    BATCH_SIZE,
    BATCH_TIMEOUT_SECONDS,
)

# Global consumer service
consumer_service: KafkaConsumerService = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle"""
    global consumer_service

    # Startup
    logger.info("Starting Neo4j Graph Projector Service")

    # Validate configuration
    security_protocol = os.getenv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT")
    if security_protocol == "SASL_SSL" and not os.getenv("KAFKA_API_KEY"):
        logger.error("KAFKA_API_KEY required for SASL_SSL security protocol")
        raise RuntimeError("Missing KAFKA_API_KEY configuration")

    if not os.getenv("NEO4J_PASSWORD"):
        logger.warning("NEO4J_PASSWORD not set, using default")

    # Initialize consumer service
    consumer_service = KafkaConsumerService(
        kafka_config=KAFKA_CONFIG,
        neo4j_config=NEO4J_CONFIG,
    )

    # Start consumer in background
    consumer_service.start()

    logger.info(
        "Service started",
        topics=TOPICS,
        batch_size=BATCH_SIZE,
        batch_timeout=BATCH_TIMEOUT_SECONDS,
        neo4j_uri=NEO4J_CONFIG["uri"],
    )

    yield

    # Shutdown
    logger.info("Shutting down Neo4j Graph Projector Service")
    if consumer_service:
        consumer_service.shutdown()
    logger.info("Service shutdown complete")


# Create FastAPI app
app = FastAPI(
    title="Neo4j Graph Projector Service",
    description="Projects graph mutations from Kafka to Neo4j patient journey graphs",
    version="1.0.0",
    lifespan=lifespan,
)


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """
    Health check endpoint
    Returns 200 if service is healthy, 503 otherwise
    """
    if not consumer_service or not consumer_service.is_healthy():
        raise HTTPException(
            status_code=503,
            detail="Service unhealthy - consumer not running"
        )

    return HealthResponse(
        status="healthy",
        timestamp=datetime.utcnow(),
    )


@app.get("/metrics", response_class=PlainTextResponse)
async def metrics():
    """
    Prometheus-compatible metrics endpoint
    Returns metrics in Prometheus text format
    """
    if not consumer_service:
        raise HTTPException(status_code=503, detail="Service not initialized")

    metrics_data = consumer_service.get_metrics()

    # Format as Prometheus text
    lines = [
        "# HELP projector_messages_consumed_total Total messages consumed from Kafka",
        "# TYPE projector_messages_consumed_total counter",
        f'projector_messages_consumed_total{{projector="neo4j-graph-projector"}} {metrics_data["messages_consumed"]}',
        "",
        "# HELP projector_messages_processed_total Total messages successfully processed",
        "# TYPE projector_messages_processed_total counter",
        f'projector_messages_processed_total{{projector="neo4j-graph-projector"}} {metrics_data["messages_processed"]}',
        "",
        "# HELP projector_messages_failed_total Total messages failed",
        "# TYPE projector_messages_failed_total counter",
        f'projector_messages_failed_total{{projector="neo4j-graph-projector"}} {metrics_data["messages_failed"]}',
        "",
        "# HELP projector_batches_processed_total Total batches processed",
        "# TYPE projector_batches_processed_total counter",
        f'projector_batches_processed_total{{projector="neo4j-graph-projector"}} {metrics_data["batches_processed"]}',
        "",
        "# HELP projector_consumer_lag Current consumer lag",
        "# TYPE projector_consumer_lag gauge",
        f'projector_consumer_lag{{projector="neo4j-graph-projector"}} {metrics_data["consumer_lag"]}',
        "",
    ]

    return "\n".join(lines)


@app.get("/status", response_model=StatusResponse)
async def status():
    """
    Service status endpoint
    Returns detailed service status and metrics
    """
    if not consumer_service:
        raise HTTPException(status_code=503, detail="Service not initialized")

    status_data = consumer_service.get_status()
    metrics_data = status_data["metrics"]

    # Test Neo4j connection
    neo4j_connected = True
    try:
        from neo4j import GraphDatabase
        driver = GraphDatabase.driver(
            NEO4J_CONFIG["uri"],
            auth=(NEO4J_CONFIG["username"], NEO4J_CONFIG["password"])
        )
        with driver.session(database=NEO4J_CONFIG["database"]) as session:
            session.run("RETURN 1")
        driver.close()
    except Exception as e:
        logger.warning("Neo4j connection check failed", error=str(e))
        neo4j_connected = False

    return StatusResponse(
        status="running" if status_data["running"] else "stopped",
        kafka_connected=status_data["running"],
        neo4j_connected=neo4j_connected,
        consumer_group=KAFKA_CONFIG["group.id"],
        topics=TOPICS,
        batch_size=BATCH_SIZE,
        batch_timeout_seconds=BATCH_TIMEOUT_SECONDS,
        metrics=MetricsResponse(
            messages_consumed=metrics_data["messages_consumed"],
            messages_processed=metrics_data["messages_processed"],
            messages_failed=metrics_data["messages_failed"],
            batches_processed=metrics_data["batches_processed"],
            consumer_lag=metrics_data["consumer_lag"],
            uptime_seconds=status_data["uptime_seconds"],
        ),
        last_processed=status_data["last_processed"],
    )


@app.get("/graph/stats")
async def graph_stats():
    """
    Neo4j graph statistics endpoint
    Returns node and relationship counts
    """
    if not consumer_service:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        stats = consumer_service.get_graph_stats()
        return stats
    except Exception as e:
        logger.error("Failed to get graph stats", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/graph/patient-journey/{patient_id}")
async def patient_journey(patient_id: str):
    """
    Query patient journey from Neo4j
    Returns chronological list of clinical events for a patient
    """
    if not consumer_service:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        events = consumer_service.query_patient_journey(patient_id)
        return {
            "patient_id": patient_id,
            "event_count": len(events),
            "events": events,
        }
    except Exception as e:
        logger.error("Failed to query patient journey", error=str(e), patient_id=patient_id)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/")
async def root():
    """Root endpoint with service information"""
    return {
        "service": "neo4j-graph-projector",
        "version": "1.0.0",
        "description": "Projects graph mutations from Kafka to Neo4j patient journey graphs",
        "endpoints": {
            "health": "/health",
            "metrics": "/metrics",
            "status": "/status",
            "graph_stats": "/graph/stats",
            "patient_journey": "/graph/patient-journey/{patient_id}",
            "docs": "/docs",
        }
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        app,
        host=SERVICE_HOST,
        port=SERVICE_PORT,
        log_level="info",
    )
