from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging
from contextlib import asynccontextmanager

# Ensure shared module is importable
# Need to go up three levels: app -> lab-service -> services -> backend
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
from app.api.fhir_api import fhir_router
from app.core.config import settings
from app.db.mongodb import connect_to_mongo, close_mongo_connection

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Lifespan context manager for the FastAPI application.
    Handles startup and shutdown events.
    """
    # Startup: Connect to MongoDB
    logger.info("Starting up Lab Service...")
    logger.info("Connecting to MongoDB...")

    # Use connect_to_mongo with retry logic
    connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

    if connection_success:
        logger.info("MongoDB connection established successfully")
        from app.db.mongodb import db
        logger.info(f"MongoDB status: {db.get_status()}")
    else:
        logger.error("Failed to connect to MongoDB. Service cannot start without MongoDB.")
        # Raise an exception to prevent the service from starting
        raise Exception("MongoDB connection failed. Service cannot start without MongoDB.")

    # Yield control back to FastAPI
    yield

    # Shutdown: Close MongoDB connection
    logger.info("Shutting down Lab Service...")
    try:
        logger.info("Closing MongoDB connection...")
        await close_mongo_connection()
        logger.info("MongoDB connection closed")
    except Exception as e:
        logger.error(f"Error closing MongoDB connection: {str(e)}")

# Create the FastAPI app with lifespan
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Lab Service API for Clinical Synthesis Hub",
    version="0.1.0",
    openapi_url=f"{settings.API_PREFIX}/openapi.json",
    docs_url=f"{settings.API_PREFIX}/docs",
    redoc_url=f"{settings.API_PREFIX}/redoc",
    lifespan=lifespan,
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
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/"]
)

# Log middleware configuration
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

# Include FHIR router
app.include_router(fhir_router, prefix=f"{settings.API_PREFIX}/fhir")

@app.get("/")
async def root():
    return {"message": "Welcome to the Lab Service API"}

@app.get("/health")
async def health_check():
    """Health check endpoint."""
    from app.db.mongodb import db

    # Check MongoDB connection
    mongodb_status = "connected" if db.is_connected() else db.get_status()

    return {
        "status": "healthy",
        "mongodb_status": mongodb_status,
        "service": "lab-service",
        "version": "1.0.0"
    }
