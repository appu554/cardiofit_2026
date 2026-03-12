"""
Main FastAPI application for the Scheduling Service.

This service provides comprehensive appointment scheduling functionality
with FHIR compliance and Apollo Federation support.
"""

import os
import sys
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Try to import the shared module using a more direct approach
try:
    # First try the normal import
    from shared.auth import HeaderAuthMiddleware
    print("Successfully imported HeaderAuthMiddleware from shared.auth")
except ImportError as e:
    print(f"Error importing HeaderAuthMiddleware from shared.auth: {e}")
    # If that fails, try the direct import module
    from app.direct_import import HeaderAuthMiddleware
    print("Using HeaderAuthMiddleware from direct_import")

from app.core.config import settings

# Lifespan event handler
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info("Using Google Healthcare API for scheduling data storage")

    # Initialize FHIR service
    try:
        from app.services.fhir_service_factory import initialize_fhir_service
        fhir_service = await initialize_fhir_service()
        logger.info(f"FHIR service initialized: {type(fhir_service).__name__}")
    except Exception as e:
        logger.error(f"Failed to initialize FHIR service: {e}")

    yield

    # Shutdown
    logger.info("Scheduling service shutdown complete")

app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Scheduling Service API for Clinical Synthesis Hub",
    version="1.0.0",
    openapi_url=f"{settings.API_PREFIX}/openapi.json",
    docs_url=f"{settings.API_PREFIX}/docs",
    redoc_url=f"{settings.API_PREFIX}/redoc",
    lifespan=lifespan
)

# Set up CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add header-based authentication middleware
# This middleware extracts user information from headers set by the API Gateway
app.add_middleware(
    HeaderAuthMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/", "/api/federation", "/api/webhooks"]
)

# Include API routes
try:
    from app.api.api import api_router
    app.include_router(api_router, prefix=settings.API_PREFIX)
    logger.info("API routes mounted")
except ImportError as e:
    logger.warning(f"Could not mount API routes: {e}")

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
    return {"message": "Welcome to the Scheduling Service API"}

@app.get("/health")
async def health_check():
    return {"status": "healthy", "service": "scheduling-service", "port": settings.PORT}

# Run the app
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=settings.PORT)
