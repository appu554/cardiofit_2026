from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> organization-service -> services -> backend
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Print the backend directory for debugging
print(f"Backend directory: {backend_dir}")
print(f"Checking if shared module exists: {os.path.exists(os.path.join(backend_dir, 'shared'))}")
if os.path.exists(os.path.join(backend_dir, 'shared')):
    print(f"Contents of shared directory:")
    for item in os.listdir(os.path.join(backend_dir, 'shared')):
        print(f"  {item}")

    # Check if auth directory exists
    auth_dir = os.path.join(backend_dir, 'shared', 'auth')
    if os.path.exists(auth_dir):
        print(f"Contents of auth directory:")
        for item in os.listdir(auth_dir):
            print(f"  {item}")

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

from app.api.api import api_router
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Lifespan event handler
async def lifespan(app: FastAPI):
    # Startup
    logger.info("Using Google Healthcare API for organization data storage")

    # Initialize FHIR service
    try:
        from app.services.organization_management_service import get_management_service
        management_service = get_management_service()
        success = await management_service.initialize()
        logger.info(f"Organization management service initialized: {success}")
    except Exception as e:
        logger.error(f"Failed to initialize organization management service: {e}")

    yield

    # Shutdown
    logger.info("Organization service shutdown complete")

app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Organization Service API for Clinical Synthesis Hub",
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

# Log middleware configuration
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

# Add GraphQL endpoints
try:
    import strawberry
    from strawberry.fastapi import GraphQLRouter
    from app.graphql.federation_schema import schema

    # Create GraphQL router for federation
    graphql_router = GraphQLRouter(schema)

    # Mount the federation endpoint (no authentication required for schema introspection)
    app.include_router(graphql_router, prefix="/api/federation")
    logger.info("Federation endpoint mounted at /api/federation")

    # Mount the regular GraphQL endpoint (with authentication)
    graphql_router_auth = GraphQLRouter(schema)
    app.include_router(graphql_router_auth, prefix="/api/graphql")
    logger.info("GraphQL endpoint mounted at /api/graphql")

    # Also mount at /graphql for direct access
    app.include_router(GraphQLRouter(schema), prefix="/graphql")
    logger.info("GraphQL endpoint mounted at /graphql")

except ImportError as e:
    logger.warning(f"Could not mount GraphQL endpoints: {e}")
except Exception as e:
    logger.error(f"Error mounting GraphQL endpoints: {e}")

@app.get("/")
async def root():
    return {"message": "Welcome to the Organization Service API"}

@app.get("/health")
async def health_check():
    return {"status": "healthy"}
