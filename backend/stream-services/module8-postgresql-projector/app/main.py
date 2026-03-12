"""
FastAPI application for PostgreSQL Projector Service
Provides health checks, metrics, and status endpoints
"""
import os
import sys
from pathlib import Path
from contextlib import asynccontextmanager
from datetime import datetime

import structlog
from fastapi import FastAPI, HTTPException
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
    POSTGRES_CONFIG,
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
    logger.info("Starting PostgreSQL Projector Service")

    # Validate configuration
    security_protocol = os.getenv("KAFKA_SECURITY_PROTOCOL", "SASL_SSL")
    if security_protocol == "SASL_SSL" and not os.getenv("KAFKA_API_KEY"):
        logger.error("KAFKA_API_KEY environment variable not set for SASL_SSL")
        raise RuntimeError("Missing KAFKA_API_KEY configuration")

    if not os.getenv("POSTGRES_PASSWORD"):
        logger.warning("POSTGRES_PASSWORD not set, using default")

    # Initialize consumer service
    consumer_service = KafkaConsumerService(
        kafka_config=KAFKA_CONFIG,
        postgres_config=POSTGRES_CONFIG,
    )

    # Start consumer in background
    consumer_service.start()

    logger.info(
        "Service started",
        topics=TOPICS,
        batch_size=BATCH_SIZE,
        batch_timeout=BATCH_TIMEOUT_SECONDS,
    )

    yield

    # Shutdown
    logger.info("Shutting down PostgreSQL Projector Service")
    if consumer_service:
        consumer_service.shutdown()
    logger.info("Service shutdown complete")


# Create FastAPI app
app = FastAPI(
    title="PostgreSQL Projector Service",
    description="Projects enriched clinical events from Kafka to PostgreSQL",
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
        f'projector_messages_consumed_total{{projector="postgresql-projector"}} {metrics_data["messages_consumed"]}',
        "",
        "# HELP projector_messages_processed_total Total messages successfully processed",
        "# TYPE projector_messages_processed_total counter",
        f'projector_messages_processed_total{{projector="postgresql-projector"}} {metrics_data["messages_processed"]}',
        "",
        "# HELP projector_messages_failed_total Total messages failed",
        "# TYPE projector_messages_failed_total counter",
        f'projector_messages_failed_total{{projector="postgresql-projector"}} {metrics_data["messages_failed"]}',
        "",
        "# HELP projector_batches_processed_total Total batches processed",
        "# TYPE projector_batches_processed_total counter",
        f'projector_batches_processed_total{{projector="postgresql-projector"}} {metrics_data["batches_processed"]}',
        "",
        "# HELP projector_consumer_lag Current consumer lag",
        "# TYPE projector_consumer_lag gauge",
        f'projector_consumer_lag{{projector="postgresql-projector"}} {metrics_data["consumer_lag"]}',
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

    # Test PostgreSQL connection
    postgres_connected = True
    try:
        import psycopg2
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        conn.close()
    except Exception as e:
        logger.warning("PostgreSQL connection check failed", error=str(e))
        postgres_connected = False

    return StatusResponse(
        status="running" if status_data["running"] else "stopped",
        kafka_connected=status_data["running"],
        postgres_connected=postgres_connected,
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


@app.get("/")
async def root():
    """Root endpoint with service information"""
    return {
        "service": "postgresql-projector",
        "version": "1.0.0",
        "endpoints": {
            "health": "/health",
            "metrics": "/metrics",
            "status": "/status",
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
