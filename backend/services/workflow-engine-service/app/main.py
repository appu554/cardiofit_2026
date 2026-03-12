"""
Main FastAPI application for Workflow Engine Service.
"""
import logging
import os
import sys
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from strawberry.fastapi import GraphQLRouter

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Add the backend directory to Python path for shared imports
backend_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))
sys.path.insert(0, os.path.join(backend_dir, "services"))

from app.core.config import settings
from app.db.database import init_db, close_db
from app.google_fhir_service import google_fhir_service
from app.supabase_service import supabase_service
from app.workflow_engine_service import workflow_engine_service


@asynccontextmanager
async def lifespan(_: FastAPI):
    """
    Lifespan context manager for FastAPI.
    Handles startup and shutdown events.
    """
    # Startup
    logger.info("Starting Workflow Engine Service...")
    
    # Initialize Supabase service
    logger.info("Initializing Supabase service...")
    supabase_success = await supabase_service.initialize()
    if supabase_success:
        logger.info("Supabase service initialized successfully")
    else:
        logger.warning("Supabase service initialization failed")

    # Initialize database
    logger.info("Initializing database...")
    db_success = await init_db()
    if db_success:
        logger.info("Database initialized successfully")
    else:
        logger.warning("Database initialization failed")

    # Initialize Google Healthcare API client
    if settings.USE_GOOGLE_HEALTHCARE_API:
        logger.info("Initializing Google Healthcare API client...")
        fhir_success = await google_fhir_service.initialize()
        if fhir_success:
            logger.info("Google Healthcare API client initialized successfully")
        else:
            logger.warning("Google Healthcare API client initialization failed")
    else:
        logger.info("Google Healthcare API integration disabled")

    # Initialize workflow engine service
    logger.info("Initializing Workflow Engine Service...")
    engine_success = await workflow_engine_service.initialize()
    if engine_success:
        logger.info("Workflow Engine Service initialized successfully")
        # Start monitoring in background
        import asyncio
        asyncio.create_task(workflow_engine_service.start_monitoring())
    else:
        logger.warning("Workflow Engine Service initialization failed")

    logger.info("Workflow Engine Service startup complete")
    
    yield
    
    # Shutdown
    logger.info("Shutting down Workflow Engine Service...")

    # Stop workflow engine monitoring
    await workflow_engine_service.stop_monitoring()

    # Close database connections
    await close_db()

    logger.info("Workflow Engine Service shutdown complete")


# Create FastAPI app
app = FastAPI(
    title="Workflow Engine Service",
    description="Workflow management service for Clinical Synthesis Hub",
    version=settings.SERVICE_VERSION,
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

# Add authentication middleware
try:
    from app.security.authentication.auth import AuthenticationMiddleware

    app.add_middleware(
        AuthenticationMiddleware,
        auth_service_url=settings.AUTH_SERVICE_URL,
        exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/api/federation", "/"]
    )
    logger.info("Authentication middleware added successfully")
except ImportError as e:
    logger.warning(f"Could not import authentication middleware: {e}")
except Exception as e:
    logger.error(f"Error adding authentication middleware: {e}")

# Add federation endpoint (no authentication required for schema introspection)
try:
    from app.gql_schema.federation_schema import schema
    
    # Create GraphQL router for federation
    graphql_router = GraphQLRouter(schema)
    
    # Mount the federation endpoint
    app.include_router(graphql_router, prefix="/api/federation")
    logger.info("Federation endpoint mounted at /api/federation")
except ImportError as e:
    logger.warning(f"Could not mount federation endpoint: {e}")
except Exception as e:
    logger.error(f"Error mounting federation endpoint: {e}")

# Add strategic orchestration endpoints
try:
    from app.api.strategic_orchestration import router as orchestration_router
    
    # Mount the strategic orchestration endpoints
    app.include_router(
        orchestration_router, 
        prefix="/api/v1", 
        tags=["Strategic Orchestration"]
    )
    logger.info("Strategic orchestration endpoints mounted at /api/v1/orchestrate")
except ImportError as e:
    logger.warning(f"Could not mount strategic orchestration endpoints: {e}")
except Exception as e:
    logger.error(f"Error mounting strategic orchestration endpoints: {e}")

# Add monitoring and observability endpoints
try:
    from app.api.monitoring import router as monitoring_router
    from app.monitoring.workflow_metrics import init_metrics_collector
    
    # Initialize monitoring system
    enable_prometheus = getattr(settings, 'ENABLE_PROMETHEUS_METRICS', True)
    init_metrics_collector(enable_prometheus=enable_prometheus)
    
    # Mount the monitoring endpoints
    app.include_router(
        monitoring_router,
        tags=["Monitoring & Observability"]
    )
    logger.info("Monitoring endpoints mounted at /monitoring")
except ImportError as e:
    logger.warning(f"Could not mount monitoring endpoints: {e}")
except Exception as e:
    logger.error(f"Error mounting monitoring endpoints: {e}")


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "message": "Welcome to the Workflow Engine Service API",
        "service": settings.SERVICE_NAME,
        "version": settings.SERVICE_VERSION
    }


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": settings.SERVICE_NAME,
        "version": settings.SERVICE_VERSION,
        "supabase_initialized": supabase_service.initialized,
        "database_models_loaded": True,
        "google_fhir_initialized": google_fhir_service.initialized,
        "google_fhir_mock_mode": getattr(google_fhir_service, 'mock_mode', False),
        "workflow_engine_initialized": workflow_engine_service.initialized,
        "workflow_engine_running": workflow_engine_service.running,
        "use_camunda_cloud": getattr(workflow_engine_service, 'use_camunda_cloud', False),
        "camunda_initialized": getattr(workflow_engine_service.camunda_service, 'initialized', False),
        "camunda_cloud_initialized": getattr(workflow_engine_service.camunda_cloud_service, 'initialized', False),
        "federation_endpoint": "/api/federation",
        "database_backend": "Supabase PostgreSQL"
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=settings.SERVICE_PORT,
        reload=settings.DEBUG
    )
