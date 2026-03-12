"""FastAPI application for MongoDB Projector service."""

import logging
import asyncio
from contextlib import asynccontextmanager
from typing import Dict, Any

from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse

from .config import get_settings
from .services.projector import MongoDBProjector

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# Global projector instance
projector: MongoDBProjector = None
projector_task: asyncio.Task = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan context manager for startup and shutdown."""
    global projector, projector_task

    # Startup
    logger.info("Starting MongoDB Projector service...")
    try:
        projector = MongoDBProjector()
        projector.connect_mongodb()

        # Start projector in background
        projector_task = asyncio.create_task(run_projector())
        logger.info("MongoDB Projector service started successfully")

        yield

    except Exception as e:
        logger.error(f"Failed to start MongoDB Projector service: {e}")
        raise

    finally:
        # Shutdown
        logger.info("Shutting down MongoDB Projector service...")
        if projector_task:
            projector_task.cancel()
            try:
                await projector_task
            except asyncio.CancelledError:
                pass

        if projector:
            projector.cleanup()

        logger.info("MongoDB Projector service shut down")


async def run_projector():
    """Run the projector in async context."""
    try:
        await asyncio.get_event_loop().run_in_executor(None, projector.start)
    except Exception as e:
        logger.error(f"Projector error: {e}")


# Create FastAPI app
settings = get_settings()
app = FastAPI(
    title="MongoDB Projector Service",
    description="Module 8 MongoDB Projector - consumes enriched events and projects to MongoDB",
    version="1.0.0",
    lifespan=lifespan,
)


@app.get("/health")
async def health_check() -> Dict[str, Any]:
    """Health check endpoint."""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    try:
        # Check MongoDB connection
        projector.client.admin.command("ping")
        mongodb_status = "healthy"
    except Exception as e:
        mongodb_status = f"unhealthy: {str(e)}"

    return {
        "status": "healthy" if mongodb_status == "healthy" else "degraded",
        "service": settings.service_name,
        "mongodb": mongodb_status,
        "kafka_topic": settings.kafka_topic,
    }


@app.get("/metrics")
async def get_metrics() -> Dict[str, Any]:
    """Get projector metrics."""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    return projector.get_statistics()


@app.get("/status")
async def get_status() -> Dict[str, Any]:
    """Get detailed projector status."""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    stats = projector.get_statistics()

    return {
        "service": settings.service_name,
        "kafka": {
            "topic": settings.kafka_topic,
            "group_id": settings.kafka_group_id,
            "messages_consumed": stats.get("messages_consumed", 0),
            "batches_processed": stats.get("batches_processed", 0),
        },
        "mongodb": {
            "database": settings.mongodb_database,
            "documents_written": stats.get("documents_written", 0),
            "timelines_updated": stats.get("timelines_updated", 0),
            "explanations_written": stats.get("explanations_written", 0),
            "total_clinical_docs": stats.get("total_clinical_docs", 0),
            "total_patient_timelines": stats.get("total_patient_timelines", 0),
            "total_ml_explanations": stats.get("total_ml_explanations", 0),
        },
        "processing": {
            "batch_size": settings.batch_size,
            "batch_timeout_seconds": settings.batch_timeout_seconds,
            "errors": stats.get("errors", 0),
        },
    }


@app.get("/collections/stats")
async def get_collection_stats() -> Dict[str, Any]:
    """Get MongoDB collection statistics."""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    try:
        stats = {}

        # Clinical documents stats
        clinical_stats = projector.db.command("collStats", "clinical_documents")
        stats["clinical_documents"] = {
            "count": clinical_stats.get("count", 0),
            "size": clinical_stats.get("size", 0),
            "avg_obj_size": clinical_stats.get("avgObjSize", 0),
            "storage_size": clinical_stats.get("storageSize", 0),
            "indexes": clinical_stats.get("nindexes", 0),
        }

        # Patient timelines stats
        timeline_stats = projector.db.command("collStats", "patient_timelines")
        stats["patient_timelines"] = {
            "count": timeline_stats.get("count", 0),
            "size": timeline_stats.get("size", 0),
            "avg_obj_size": timeline_stats.get("avgObjSize", 0),
            "storage_size": timeline_stats.get("storageSize", 0),
            "indexes": timeline_stats.get("nindexes", 0),
        }

        # ML explanations stats
        explanation_stats = projector.db.command("collStats", "ml_explanations")
        stats["ml_explanations"] = {
            "count": explanation_stats.get("count", 0),
            "size": explanation_stats.get("size", 0),
            "avg_obj_size": explanation_stats.get("avgObjSize", 0),
            "storage_size": explanation_stats.get("storageSize", 0),
            "indexes": explanation_stats.get("nindexes", 0),
        }

        return stats

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error getting collection stats: {str(e)}")


@app.post("/control/pause")
async def pause_projector():
    """Pause the projector."""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    # Note: KafkaConsumerBase doesn't have pause/resume methods yet
    # This is a placeholder for future implementation
    return {"status": "pause not yet implemented"}


@app.post("/control/resume")
async def resume_projector():
    """Resume the projector."""
    if not projector:
        raise HTTPException(status_code=503, detail="Projector not initialized")

    # Note: KafkaConsumerBase doesn't have pause/resume methods yet
    # This is a placeholder for future implementation
    return {"status": "resume not yet implemented"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=settings.service_port,
        reload=False,
    )
