from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import logging
import os
import sys
from contextlib import asynccontextmanager

# Ensure shared module is importable
# Need to go up three levels: app -> timeline-service -> services -> backend
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
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Lifespan context manager for the FastAPI application.
    Handles startup and shutdown events.
    """
    # Startup: Log service configuration
    logger.info(f"Starting Timeline Service on port {settings.PORT}")
    logger.info(f"FHIR Service URL: {settings.FHIR_SERVICE_URL}")
    logger.info(f"Observation Service URL: {settings.OBSERVATION_SERVICE_URL}")
    logger.info(f"Condition Service URL: {settings.CONDITION_SERVICE_URL}")
    logger.info(f"Medication Service URL: {settings.MEDICATION_SERVICE_URL}")
    logger.info(f"Encounter Service URL: {settings.ENCOUNTER_SERVICE_URL}")
    logger.info(f"Document Service URL: {settings.DOCUMENT_SERVICE_URL}")

    # Yield control back to FastAPI
    yield

    # Shutdown: Log shutdown
    logger.info("Shutting down Timeline Service")

# Create the FastAPI app with lifespan
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Timeline Service API for Clinical Synthesis Hub",
    version="1.0.0",  # Updated version for production
    openapi_url=f"{settings.API_PREFIX}/openapi.json",
    docs_url=f"{settings.API_PREFIX}/docs",
    redoc_url=f"{settings.API_PREFIX}/redoc",
    lifespan=lifespan,
)

# Set up CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Replace with specific origins in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add authentication middleware
app.add_middleware(
    HeaderAuthMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/"]
)

# Log middleware configuration
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

@app.get("/")
async def root():
    return {"message": "Welcome to the Timeline Service API"}

@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "service": "timeline-service",
        "version": "1.0.0",
        "port": settings.PORT,
        "endpoints": {
            "fhir": settings.FHIR_SERVICE_URL,
            "observation": settings.OBSERVATION_SERVICE_URL,
            "condition": settings.CONDITION_SERVICE_URL,
            "medication": settings.MEDICATION_SERVICE_URL,
            "encounter": settings.ENCOUNTER_SERVICE_URL,
            "document": settings.DOCUMENT_SERVICE_URL,
            "lab": settings.LAB_SERVICE_URL
        },
        "api_routes": {
            "timeline": "/api/timeline",
            "fhir": "/api/fhir"
        }
    }
