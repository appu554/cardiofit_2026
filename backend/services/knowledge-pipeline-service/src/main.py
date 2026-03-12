"""
Knowledge Pipeline Service - Main Application
Real data ingestion pipeline for clinical knowledge into GraphDB
"""

import asyncio
import logging
from contextlib import asynccontextmanager
from typing import Dict, Any

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
import structlog

from core.config import settings
from core.pipeline_orchestrator import PipelineOrchestrator
from core.graphdb_client import GraphDBClient
from api.routes import pipeline_router, set_orchestrator

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)

# Global pipeline orchestrator
pipeline_orchestrator: PipelineOrchestrator = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan management"""
    global pipeline_orchestrator
    
    logger.info("Starting Knowledge Pipeline Service", version=settings.VERSION)
    
    try:
        # Initialize GraphDB client
        graphdb_client = GraphDBClient(
            endpoint=settings.GRAPHDB_ENDPOINT,
            repository=settings.GRAPHDB_REPOSITORY,
            username=settings.GRAPHDB_USERNAME,
            password=settings.GRAPHDB_PASSWORD
        )
        
        # Test GraphDB connection
        await graphdb_client.connect()
        logger.info("GraphDB connection established", 
                   endpoint=settings.GRAPHDB_ENDPOINT,
                   repository=settings.GRAPHDB_REPOSITORY)
        
        # Initialize pipeline orchestrator
        pipeline_orchestrator = PipelineOrchestrator(graphdb_client)
        await pipeline_orchestrator.initialize()

        # Set orchestrator in API routes
        set_orchestrator(pipeline_orchestrator)

        logger.info("Knowledge Pipeline Service started successfully")
        
        yield
        
    except Exception as e:
        logger.error("Failed to start Knowledge Pipeline Service", error=str(e))
        raise
    finally:
        # Cleanup
        if pipeline_orchestrator:
            await pipeline_orchestrator.cleanup()
        if graphdb_client:
            await graphdb_client.disconnect()
        logger.info("Knowledge Pipeline Service stopped")


# Create FastAPI application
app = FastAPI(
    title="Knowledge Pipeline Service",
    description="Real clinical knowledge ingestion pipeline for GraphDB",
    version=settings.VERSION,
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include API routes
app.include_router(pipeline_router, prefix="/api/v1")


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    try:
        if pipeline_orchestrator:
            status = await pipeline_orchestrator.get_status()
            return {
                "status": "healthy",
                "service": "knowledge-pipeline-service",
                "version": settings.VERSION,
                "graphdb_connected": status.get("graphdb_connected", False),
                "ingesters_available": status.get("ingesters_available", [])
            }
        else:
            return {
                "status": "starting",
                "service": "knowledge-pipeline-service",
                "version": settings.VERSION
            }
    except Exception as e:
        logger.error("Health check failed", error=str(e))
        raise HTTPException(status_code=503, detail="Service unhealthy")


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "Knowledge Pipeline Service",
        "version": settings.VERSION,
        "description": "Real clinical knowledge ingestion pipeline for GraphDB",
        "endpoints": {
            "health": "/health",
            "api": "/api/v1",
            "docs": "/docs"
        }
    }


if __name__ == "__main__":
    import uvicorn
    
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level="info"
    )
