"""
UPS Projector Main Application

FastAPI service wrapping the UPS projector with health and metrics endpoints.
"""

import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.responses import JSONResponse

from module8_shared.config import StreamConfig
from projector import UPSProjector

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global projector instance
projector: UPSProjector = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    global projector

    # Startup
    logger.info("Starting UPS Projector service...")
    config = StreamConfig()

    projector = UPSProjector(config)

    # Start projector in background task
    asyncio.create_task(projector.start())

    yield

    # Shutdown
    logger.info("Shutting down UPS Projector service...")
    if projector:
        projector.shutdown()


app = FastAPI(
    title="UPS Projector Service",
    description="Unified Patient Summary Read Model Projector",
    version="1.0.0",
    lifespan=lifespan
)


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    if not projector:
        return JSONResponse(
            status_code=503,
            content={"status": "unhealthy", "reason": "Projector not initialized"}
        )

    health = projector.get_health()
    status_code = 200 if health["status"] == "healthy" else 503

    return JSONResponse(status_code=status_code, content=health)


@app.get("/metrics")
async def get_metrics():
    """Get projector metrics."""
    if not projector:
        return JSONResponse(
            status_code=503,
            content={"error": "Projector not initialized"}
        )

    return projector.get_health()


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "service": "UPS Projector",
        "description": "Unified Patient Summary Read Model Projector",
        "version": "1.0.0",
        "endpoints": {
            "health": "/health",
            "metrics": "/metrics"
        }
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8055)
