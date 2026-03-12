"""
FastAPI application for FHIR Store Projector health endpoints
"""

from fastapi import FastAPI
from typing import Dict, Any
import structlog

from app.config import Config

logger = structlog.get_logger(__name__)

app = FastAPI(
    title="Module 8 FHIR Store Projector",
    description="Consumes from prod.ehr.fhir.upsert and writes to Google Cloud Healthcare FHIR Store",
    version="1.0.0",
)

# Global projector instance (will be set by run.py)
projector_instance = None


def set_projector_instance(projector):
    """Set global projector instance for health checks"""
    global projector_instance
    projector_instance = projector


@app.get("/health")
async def health_check() -> Dict[str, Any]:
    """
    Health check endpoint

    Returns service health status and basic metrics
    """
    if projector_instance is None:
        return {
            "status": "starting",
            "message": "Projector not initialized",
        }

    try:
        # Get handler stats
        handler_stats = projector_instance.handler.get_stats()

        return {
            "status": "healthy",
            "service": "fhir-store-projector",
            "fhir_store": {
                "project_id": Config.GOOGLE_CLOUD_PROJECT_ID,
                "location": Config.GOOGLE_CLOUD_LOCATION,
                "dataset_id": Config.GOOGLE_CLOUD_DATASET_ID,
                "store_id": Config.GOOGLE_CLOUD_FHIR_STORE_ID,
            },
            "kafka": {
                "topic": Config.KAFKA_TOPIC_FHIR_UPSERT,
                "group_id": Config.KAFKA_GROUP_ID,
            },
            "stats": {
                "total_upserts": handler_stats['total_upserts'],
                "successful_creates": handler_stats['successful_creates'],
                "successful_updates": handler_stats['successful_updates'],
                "failed_upserts": handler_stats['failed_upserts'],
                "validation_errors": handler_stats['validation_errors'],
                "success_rate": handler_stats['success_rate'],
            },
        }
    except Exception as e:
        logger.error("Health check failed", error=str(e))
        return {
            "status": "unhealthy",
            "error": str(e),
        }


@app.get("/metrics")
async def metrics() -> Dict[str, Any]:
    """
    Detailed metrics endpoint

    Returns comprehensive processing statistics
    """
    if projector_instance is None:
        return {
            "error": "Projector not initialized",
        }

    try:
        return projector_instance.get_processing_summary()
    except Exception as e:
        logger.error("Metrics retrieval failed", error=str(e))
        return {
            "error": str(e),
        }


@app.get("/stats/reset")
async def reset_stats() -> Dict[str, str]:
    """
    Reset statistics counters

    Useful for testing and monitoring
    """
    if projector_instance is None:
        return {
            "status": "error",
            "message": "Projector not initialized",
        }

    try:
        projector_instance.handler.reset_stats()
        projector_instance.processing_stats = {
            'total_processed': 0,
            'successful_upserts': 0,
            'failed_upserts': 0,
            'validation_errors': 0,
            'resource_type_counts': {},
        }

        return {
            "status": "success",
            "message": "Statistics reset",
        }
    except Exception as e:
        logger.error("Stats reset failed", error=str(e))
        return {
            "status": "error",
            "message": str(e),
        }


@app.get("/")
async def root() -> Dict[str, str]:
    """Root endpoint"""
    return {
        "service": "Module 8 FHIR Store Projector",
        "version": "1.0.0",
        "status": "running",
    }
