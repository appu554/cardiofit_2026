from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> order-management-service -> services -> backend
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
    logger.info("Successfully imported HeaderAuthMiddleware from shared.auth")
except ImportError as e:
    logger.error(f"Error importing HeaderAuthMiddleware from shared.auth: {e}")
    # If that fails, try the direct import module
    from app.direct_import import HeaderAuthMiddleware
    logger.info("Using HeaderAuthMiddleware from direct_import")

from app.api.api import api_router
from app.core.config import settings

# Lifespan event handler
async def lifespan(app: FastAPI):
    # Startup
    logger.info("Using Google Healthcare API for order management data storage")

    # Initialize FHIR service
    try:
        from app.services.fhir_service_factory import initialize_fhir_service
        fhir_service = await initialize_fhir_service()
        logger.info(f"FHIR service initialized: {type(fhir_service).__name__}")
    except Exception as e:
        logger.error(f"Failed to initialize FHIR service: {e}")

    yield

    # Shutdown
    logger.info("Order Management service shutdown complete")

app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Order Management Service API for Clinical Synthesis Hub - CPOE Core",
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

    # Also mount at /graphql for direct access
    app.include_router(GraphQLRouter(schema), prefix="/graphql")
    logger.info("GraphQL endpoint mounted at /graphql")

except ImportError as e:
    logger.warning(f"Could not mount GraphQL endpoints: {e}")
except Exception as e:
    logger.error(f"Error mounting GraphQL endpoints: {e}")

@app.get("/")
async def root():
    return {"message": "Welcome to the Order Management Service API - CPOE Core"}

@app.get("/health")
async def health_check():
    return {"status": "healthy", "service": "order-management-service", "port": settings.PORT}

# Run the app
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=settings.PORT)
