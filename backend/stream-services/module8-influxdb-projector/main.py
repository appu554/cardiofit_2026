"""Main entry point for InfluxDB Projector Service."""
import logging
import signal
import sys
import threading
from contextlib import asynccontextmanager
from typing import Dict, Any

from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
import uvicorn

from config import config
from influxdb_manager import influxdb_manager
from projector import projector

# Configure logging
logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


# Lifecycle management
@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage service lifecycle."""
    logger.info(f"Starting {config.SERVICE_NAME}...")

    try:
        # Validate configuration
        config.validate()

        # Connect to InfluxDB
        influxdb_manager.connect()

        # Setup buckets and downsampling
        influxdb_manager.setup_buckets()
        influxdb_manager.setup_downsampling_tasks()

        # Start Kafka consumer in background thread to avoid blocking FastAPI startup
        def run_projector():
            try:
                projector.start()
            except Exception as e:
                logger.error(f"Projector thread error: {e}")

        projector_thread = threading.Thread(target=run_projector, daemon=True)
        projector_thread.start()

        logger.info(f"{config.SERVICE_NAME} started successfully on port {config.SERVICE_PORT}")
        logger.info("InfluxDB projector running in background thread")

    except Exception as e:
        logger.error(f"Failed to start service: {e}")
        raise

    yield

    # Shutdown
    logger.info(f"Shutting down {config.SERVICE_NAME}...")
    projector.stop()
    influxdb_manager.close()
    logger.info("Service shutdown complete")


# Create FastAPI app
app = FastAPI(
    title="InfluxDB Projector Service",
    description="Projects enriched EHR events to InfluxDB time-series database",
    version="1.0.0",
    lifespan=lifespan
)


@app.get("/health")
async def health_check() -> Dict[str, Any]:
    """Health check endpoint."""
    try:
        # Check InfluxDB connection
        if influxdb_manager.client:
            health = influxdb_manager.client.health()
            influxdb_status = health.status
        else:
            influxdb_status = "disconnected"

        # Check Kafka consumer (check if consumer exists)
        kafka_status = "running" if hasattr(projector, 'consumer') and projector.consumer else "stopped"

        stats = projector.get_stats()

        return {
            "status": "healthy" if kafka_status == "running" else "degraded",
            "service": config.SERVICE_NAME,
            "influxdb_status": influxdb_status,
            "kafka_status": kafka_status,
            "statistics": stats
        }
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=str(e))


@app.get("/stats")
async def get_statistics() -> Dict[str, Any]:
    """Get detailed projector statistics."""
    return projector.get_stats()


@app.get("/buckets")
async def get_buckets() -> Dict[str, Any]:
    """Get information about InfluxDB buckets."""
    try:
        buckets = []
        for bucket_name in [
            config.INFLUXDB_BUCKET_REALTIME,
            config.INFLUXDB_BUCKET_1MIN,
            config.INFLUXDB_BUCKET_1HOUR
        ]:
            bucket = influxdb_manager.buckets_api.find_bucket_by_name(bucket_name)
            if bucket:
                buckets.append({
                    "name": bucket.name,
                    "id": bucket.id,
                    "retention_seconds": bucket.retention_rules[0].every_seconds if bucket.retention_rules else None,
                    "description": bucket.description
                })

        return {"buckets": buckets}
    except Exception as e:
        logger.error(f"Failed to get buckets: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/reset")
async def reset_stats() -> Dict[str, str]:
    """Reset projector statistics."""
    projector.stats = {
        "total_events_processed": 0,
        "vitals_written": 0,
        "heart_rate_count": 0,
        "blood_pressure_count": 0,
        "spo2_count": 0,
        "temperature_count": 0,
        "non_vital_skipped": 0,
        "errors": 0
    }
    return {"message": "Statistics reset successfully"}


def signal_handler(sig, frame):
    """Handle shutdown signals."""
    logger.info("Received shutdown signal")
    sys.exit(0)


if __name__ == "__main__":
    # Register signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # Start service
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=config.SERVICE_PORT,
        log_level=config.LOG_LEVEL.lower()
    )
