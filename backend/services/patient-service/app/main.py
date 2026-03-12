from fastapi import FastAPI, Depends
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> patient-service -> services -> backend
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Import shared modules
try:
    from shared.auth import HeaderAuthMiddleware, get_current_user
    logger.info("Successfully imported HeaderAuthMiddleware from shared.auth")
except ImportError as e:
    logger.error(f"Error importing HeaderAuthMiddleware from shared.auth: {e}")
    # If that fails, try the direct import module
    from app.direct_import import HeaderAuthMiddleware, get_current_user
    logger.info("Using HeaderAuthMiddleware from direct_import")

# Import database connection
from app.db.mongodb import connect_to_mongo, close_mongo_connection

# Import API router
from app.api.api import api_router
logger = logging.getLogger(__name__)

# Use lifespan context manager for startup/shutdown events
from contextlib import asynccontextmanager

@asynccontextmanager
async def lifespan(_: FastAPI):
    """
    Lifespan context manager for FastAPI.

    Handles startup and shutdown events.
    """
    # Import settings
    from app.core.config import settings

    if not settings.USE_GOOGLE_HEALTHCARE_API:
        # Connect to MongoDB only if not using Google Healthcare API
        logger.info("Connecting to MongoDB on startup...")
        connection_success = await connect_to_mongo(max_retries=5, retry_delay=3)

        if connection_success:
            from app.db.mongodb import db
            logger.info(f"MongoDB connection established successfully. Database status: {db.get_status()}")
        else:
            logger.warning("Failed to connect to MongoDB. Service will use fallback storage.")
    else:
        logger.info("Using Google Cloud Healthcare API. Skipping MongoDB connection.")

    # Initialize the FHIR service
    from app.services.fhir_service_factory import initialize_fhir_service
    fhir_service = await initialize_fhir_service()
    logger.info(f"FHIR service initialized successfully: {type(fhir_service).__name__}")

    yield

    # Shutdown: Close MongoDB connection if not using Google Healthcare API
    if not settings.USE_GOOGLE_HEALTHCARE_API:
        logger.info("Closing MongoDB connection on shutdown...")
        await close_mongo_connection()
        logger.info("MongoDB connection closed")
    else:
        logger.info("Using Google Cloud Healthcare API. No MongoDB connection to close.")

# Create FastAPI app
app = FastAPI(
    title="Patient Service",
    description="Patient management service for Clinical Synthesis Hub",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
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
    exclude_paths=[
        "/docs",
        "/openapi.json",
        "/redoc",
        "/health",
        "/api/federation",
        "/api/federation/playground",
        "/api/webhooks",  # Exclude webhook endpoints from authentication
        "/api/context"   # Exclude context service endpoints from authentication
    ]
)

# Include API router
app.include_router(api_router, prefix="/api")

# Define routes
@app.get("/health")
async def health():
    """
    Health check endpoint that doesn't require authentication.
    """
    return {"status": "healthy"}

# These endpoints are now handled by the API router

@app.get("/me")
async def get_me(user=Depends(get_current_user)):
    """
    Get the authenticated user's information.
    """
    return user

# Run the app
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
