from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> encounter-service -> services -> backend
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the HeaderAuthMiddleware
try:
    from shared.auth import HeaderAuthMiddleware
except ImportError:
    # Fallback to direct import if shared module is not available
    from app.direct_import import HeaderAuthMiddleware

from app.api.api import api_router
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Lifespan event handler
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info("Using Google Healthcare API for encounter data storage")

    # Initialize FHIR service
    try:
        from app.services.fhir_service_factory import initialize_fhir_service
        fhir_service = await initialize_fhir_service()
        logger.info(f"FHIR service initialized: {type(fhir_service).__name__}")
    except Exception as e:
        logger.error(f"Failed to initialize FHIR service: {e}")

    yield

    # Shutdown
    logger.info("Encounter Management service shutdown complete")

# Initialize FastAPI app with lifespan
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Encounter Service API for Clinical Synthesis Hub",
    version="1.0.0",  # Updated version for production
    openapi_url=f"{settings.API_PREFIX}/openapi.json",
    docs_url=f"{settings.API_PREFIX}/docs",
    redoc_url=f"{settings.API_PREFIX}/redoc",
    lifespan=lifespan,
)

# Set up CORS with more restrictive settings for production
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "http://localhost:4200",  # Angular dev server
        "https://clinical-synthesis-hub.com",  # Production domain (example)
        "https://api.clinical-synthesis-hub.com",  # API domain (example)
    ],
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"],
    allow_headers=["Authorization", "Content-Type", "Accept", "Origin", "User-Agent"],
)

# Add authentication middleware
app.add_middleware(
    HeaderAuthMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/", "/api/webhooks"]
)

# Log middleware configuration
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

# Add federation endpoint (no authentication required for schema introspection)
try:
    import strawberry
    from strawberry.fastapi import GraphQLRouter
    from app.graphql.federation_schema import schema

    # Create GraphQL router for federation
    graphql_router = GraphQLRouter(schema)

    # Mount the federation endpoint
    app.include_router(graphql_router, prefix="/api/federation")

    logger.info("Federation endpoint mounted at /api/federation")
except ImportError as e:
    logger.warning(f"Could not mount federation endpoint: {e}")
except Exception as e:
    logger.error(f"Error mounting federation endpoint: {e}")

@app.get("/")
async def root():
    return {"message": "Welcome to the Encounter Management Service API"}

@app.get("/health")
async def health_check():
    """
    Health check endpoint that verifies the service is running.
    Returns status information about the service.
    """
    # Get FHIR service status
    try:
        from app.services.fhir_service_factory import get_fhir_service
        fhir_service = await get_fhir_service()
        fhir_status = "initialized" if fhir_service and fhir_service._initialized else "not_initialized"
        using_google_healthcare = True
    except Exception as e:
        fhir_status = f"error: {str(e)}"
        using_google_healthcare = False

    return {
        "status": "healthy",
        "service": settings.PROJECT_NAME,
        "version": "1.0.0",
        "fhir_service": {
            "status": fhir_status,
            "using_google_healthcare_api": using_google_healthcare,
            "fhir_store_path": f"projects/{settings.GOOGLE_CLOUD_PROJECT}/locations/{settings.GOOGLE_CLOUD_LOCATION}/datasets/{settings.GOOGLE_CLOUD_DATASET}/fhirStores/{settings.GOOGLE_CLOUD_FHIR_STORE}"
        },
        "federation": {
            "endpoint": "/api/federation",
            "schema_available": True
        }
    }
