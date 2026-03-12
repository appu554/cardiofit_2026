"""
Global Outbox Service - Main Application

FastAPI application with gRPC server for centralized event publishing.
Provides REST endpoints for monitoring and gRPC endpoints for event publishing.
"""

import asyncio
import logging
import signal
import sys
from contextlib import asynccontextmanager
from typing import Dict, Any

from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
import uvicorn

from app.core.config import settings
from app.core.database import db_manager
from app.services.medical_circuit_breaker import medical_circuit_breaker

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL.upper()),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('global-outbox-service.log')
    ]
)

logger = logging.getLogger(__name__)

# Global gRPC server reference
grpc_server = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager"""
    global grpc_server
    
    # Startup
    logger.info(f"🌐 Starting {settings.PROJECT_NAME} v{settings.VERSION}")
    logger.info(f"   Environment: {settings.ENVIRONMENT}")
    logger.info(f"   HTTP Port: {settings.PORT}")
    logger.info(f"   gRPC Port: {settings.GRPC_PORT}")
    
    try:
        # Initialize database
        logger.info("Initializing database connection...")
        db_connected = await db_manager.connect()
        
        if not db_connected:
            logger.error("❌ Failed to connect to database")
            raise RuntimeError("Database connection failed")
        
        # Run database migration
        logger.info("Running database migration...")
        migration_success = await db_manager.execute_migration()
        
        if not migration_success:
            logger.warning("⚠️  Database migration failed, but continuing...")
        
        # Start gRPC server
        logger.info("Starting gRPC server...")
        try:
            grpc_server = await start_grpc_server()
            logger.info("✅ gRPC server started successfully")
        except Exception as grpc_error:
            logger.warning(f"⚠️  gRPC server failed to start: {grpc_error}")
            grpc_server = None

        # Start background publisher
        logger.info("Starting background publisher...")
        try:
            from app.services.publisher import start_background_publisher
            publisher_task = asyncio.create_task(start_background_publisher())
            logger.info("✅ Background publisher started successfully")
        except Exception as publisher_error:
            logger.warning(f"⚠️  Background publisher failed to start: {publisher_error}")
            publisher_task = None

        if grpc_server or publisher_task:
            logger.info("✅ Global Outbox Service started successfully")
        else:
            logger.error("❌ Failed to start core services")
            raise RuntimeError("Core services failed to start")
        
    except Exception as e:
        logger.error(f"❌ Failed to start service: {e}")
        raise
    
    yield
    
    # Shutdown
    logger.info("🛑 Shutting down Global Outbox Service...")
    
    try:
        # Stop background publisher
        if 'publisher_task' in locals() and publisher_task:
            logger.info("Stopping background publisher...")
            from app.services.publisher import stop_background_publisher
            await stop_background_publisher()
            publisher_task.cancel()
            try:
                await publisher_task
            except asyncio.CancelledError:
                pass

        # Stop gRPC server
        if grpc_server:
            logger.info("Stopping gRPC server...")
            grpc_server.cancel()
            try:
                await grpc_server
            except asyncio.CancelledError:
                pass
        
        # Close database connections
        logger.info("Closing database connections...")
        await db_manager.disconnect()
        
        logger.info("✅ Global Outbox Service shutdown complete")
        
    except Exception as e:
        logger.error(f"❌ Error during shutdown: {e}")

# Create FastAPI application
app = FastAPI(
    title=settings.PROJECT_NAME,
    version=settings.VERSION,
    description="Centralized event publishing service for Clinical Synthesis Hub microservices",
    lifespan=lifespan,
    docs_url="/docs" if settings.DEBUG else None,
    redoc_url="/redoc" if settings.DEBUG else None
)

async def start_grpc_server():
    """Start the gRPC server"""
    try:
        # Import here to avoid circular imports
        from app.grpc_server import serve_grpc

        # Start gRPC server in background task
        grpc_task = asyncio.create_task(serve_grpc())

        # Give it a moment to start
        await asyncio.sleep(0.1)

        logger.info(f"✅ gRPC server started on port {settings.GRPC_PORT}")
        return grpc_task

    except Exception as e:
        logger.error(f"❌ Failed to start gRPC server: {e}")
        raise

# =====================================================================
# REST API Endpoints
# =====================================================================

@app.get("/")
async def root():
    """Root endpoint with service information"""
    return {
        "service": settings.PROJECT_NAME,
        "version": settings.VERSION,
        "status": "running",
        "environment": settings.ENVIRONMENT,
        "endpoints": {
            "health": "/health",
            "metrics": "/metrics",
            "stats": "/stats",
            "grpc": f"localhost:{settings.GRPC_PORT}"
        }
    }

@app.get("/health")
async def health_check():
    """
    Comprehensive health check endpoint
    
    Returns detailed health information including:
    - Overall service status
    - Database connectivity
    - Component health
    """
    try:
        # Check database health
        db_health = await db_manager.health_check()
        
        # Determine overall health
        overall_healthy = (
            db_health.get("status") == "healthy" and
            db_manager.is_connected
        )
        
        health_response = {
            "service": settings.PROJECT_NAME,
            "version": settings.VERSION,
            "status": "healthy" if overall_healthy else "unhealthy",
            "timestamp": asyncio.get_event_loop().time(),
            "environment": settings.ENVIRONMENT,
            "components": {
                "database": db_health,
                "grpc_server": {
                    "status": "healthy" if 'grpc_server' in locals() and grpc_server else "unhealthy",
                    "port": settings.GRPC_PORT
                }
            }
        }
        
        # Return appropriate HTTP status
        status_code = 200 if overall_healthy else 503
        
        return JSONResponse(
            content=health_response,
            status_code=status_code
        )
        
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return JSONResponse(
            content={
                "service": settings.PROJECT_NAME,
                "status": "unhealthy",
                "error": str(e),
                "timestamp": asyncio.get_event_loop().time()
            },
            status_code=503
        )

@app.get("/stats")
async def get_outbox_stats():
    """
    Get outbox statistics
    
    Returns:
    - Queue depths by service
    - Success rates
    - Processing metrics
    - Dead letter queue status
    """
    try:
        if not db_manager.is_connected:
            raise HTTPException(status_code=503, detail="Database not connected")
        
        stats = await db_manager.get_outbox_stats()
        
        return {
            "service": settings.PROJECT_NAME,
            "timestamp": asyncio.get_event_loop().time(),
            "statistics": stats
        }
        
    except Exception as e:
        logger.error(f"Failed to get outbox stats: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/metrics")
async def get_metrics():
    """
    Prometheus-compatible metrics endpoint
    
    Returns metrics in Prometheus format for monitoring
    """
    try:
        if not db_manager.is_connected:
            raise HTTPException(status_code=503, detail="Database not connected")
        
        stats = await db_manager.get_outbox_stats()
        
        # Generate Prometheus-style metrics
        metrics = []
        
        # Queue depth metrics
        for service, depth in stats.get("queue_depths", {}).items():
            metrics.append(f'outbox_queue_depth{{service="{service}"}} {depth}')
        
        # Total metrics
        metrics.append(f'outbox_total_processed_24h {stats.get("total_processed_24h", 0)}')
        metrics.append(f'outbox_dead_letter_count {stats.get("dead_letter_count", 0)}')
        
        # Service health
        metrics.append(f'outbox_service_healthy{{component="database"}} {1 if db_manager.is_healthy else 0}')
        metrics.append(f'outbox_service_healthy{{component="grpc"}} {1 if "grpc_server" in locals() and grpc_server else 0}')
        
        return "\n".join(metrics)

    except Exception as e:
        logger.error(f"Failed to generate metrics: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/circuit-breaker")
async def get_circuit_breaker_status():
    """
    Get medical circuit breaker status

    Returns detailed information about the medical-aware circuit breaker
    including priority lane states, load metrics, and safety statistics.
    """
    try:
        if not settings.MEDICAL_CIRCUIT_BREAKER_ENABLED:
            return {
                "enabled": False,
                "message": "Medical circuit breaker is disabled"
            }

        status = medical_circuit_breaker.get_circuit_breaker_status()

        return {
            "service": settings.PROJECT_NAME,
            "timestamp": asyncio.get_event_loop().time(),
            "medical_circuit_breaker": {
                "enabled": True,
                **status
            }
        }

    except Exception as e:
        logger.error(f"Failed to get circuit breaker status: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# Event publishing is handled via gRPC - see app/grpc_server.py

@app.get("/debug/config")
async def get_debug_config():
    """Debug endpoint to view configuration (development only)"""
    if not settings.DEBUG:
        raise HTTPException(status_code=404, detail="Not found")
    
    return {
        "project_name": settings.PROJECT_NAME,
        "version": settings.VERSION,
        "environment": settings.ENVIRONMENT,
        "ports": {
            "http": settings.PORT,
            "grpc": settings.GRPC_PORT,
            "metrics": settings.METRICS_PORT
        },
        "database": {
            "connected": db_manager.is_connected,
            "healthy": db_manager.is_healthy,
            "pool_size": settings.DATABASE_POOL_SIZE
        },
        "publisher": {
            "enabled": settings.PUBLISHER_ENABLED,
            "poll_interval": settings.PUBLISHER_POLL_INTERVAL,
            "batch_size": settings.PUBLISHER_BATCH_SIZE
        }
    }

# =====================================================================
# Error Handlers
# =====================================================================

@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """Global exception handler"""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    
    return JSONResponse(
        status_code=500,
        content={
            "service": settings.PROJECT_NAME,
            "error": "Internal server error",
            "detail": str(exc) if settings.DEBUG else "An unexpected error occurred"
        }
    )

# =====================================================================
# Signal Handlers
# =====================================================================

def setup_signal_handlers():
    """Setup signal handlers for graceful shutdown"""
    def signal_handler(signum, frame):
        logger.info(f"Received signal {signum}, initiating graceful shutdown...")
        # The lifespan context manager will handle the actual shutdown
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

if __name__ == "__main__":
    setup_signal_handlers()
    
    logger.info(f"Starting {settings.PROJECT_NAME} on {settings.HOST}:{settings.PORT}")
    
    uvicorn.run(
        "app.main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level=settings.LOG_LEVEL.lower(),
        access_log=True
    )
