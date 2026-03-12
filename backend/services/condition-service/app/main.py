from fastapi import FastAPI, Request
from contextlib import asynccontextmanager
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> condition-service -> services -> backend
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
from app.db.mongodb import connect_to_mongo, close_mongo_connection
from app.services.fhir_service import initialize_fhir_service

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Define lifespan context manager for startup and shutdown events
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Connect to MongoDB and initialize FHIR service
    await connect_to_mongo()
    logger.info("Initializing FHIR service...")
    await initialize_fhir_service()
    logger.info("FHIR service initialized")

    yield

    # Shutdown: Close MongoDB connection
    await close_mongo_connection()

# Initialize FastAPI app with lifespan
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Condition Service API for Clinical Synthesis Hub",
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
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/"]
)

# Log middleware configuration
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# Add production-ready request logging middleware
@app.middleware("http")
async def log_requests(request: Request, call_next):
    """Log important request information without excessive details."""
    # Only log essential information
    logger.info(f"Request: {request.method} {request.url.path}")

    # Process the request
    response = await call_next(request)

    # Log response status
    logger.info(f"Response: {response.status_code}")

    return response

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

@app.get("/")
async def root():
    return {"message": "Welcome to the Condition Service API"}

@app.get("/health")
async def health_check():
    """
    Health check endpoint that verifies the service is running and connected to MongoDB.
    Returns status information about the service.
    """
    from app.db.mongodb import db

    # Check MongoDB connection
    db_status = "connected" if db.is_connected() else "disconnected"

    # Get FHIR service status
    from app.services.fhir_service import get_fhir_service
    fhir_service = get_fhir_service()
    fhir_status = "initialized" if fhir_service and fhir_service._initialized else "not_initialized"

    return {
        "status": "healthy",
        "service": settings.PROJECT_NAME,
        "version": "1.0.0",
        "database": {
            "status": db_status,
            "connection_details": db.get_status()
        },
        "fhir_service": {
            "status": fhir_status,
            "using_mongodb": fhir_service and fhir_service.collection is not None if fhir_service else False
        }
    }
