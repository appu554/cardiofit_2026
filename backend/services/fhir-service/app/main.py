from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
import os
import sys
import logging

# Ensure shared module is importable
# Need to go up three levels: app -> fhir-service -> services -> backend
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
from app.db.mongodb import connect_to_mongo, close_mongo_connection

app = FastAPI(
    title=settings.PROJECT_NAME,
    description="FHIR Service API for Clinical Synthesis Hub",
    version="0.1.0",
    openapi_url=f"{settings.API_PREFIX}/openapi.json",
    docs_url=f"{settings.API_PREFIX}/docs",
    redoc_url=f"{settings.API_PREFIX}/redoc",
)

# Set up CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add authentication middleware
app.add_middleware(
    HeaderAuthMiddleware,
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health", "/"]
)
logger.info("Added HeaderAuthMiddleware to extract user information from request headers")

# API Gateway now handles permissions, so we don't need to check permissions here
logger.info("Permission checking is handled by the API Gateway")

# Include API router
app.include_router(api_router, prefix=settings.API_PREFIX)

# Connect to MongoDB on startup
@app.on_event("startup")
async def startup_db_client():
    await connect_to_mongo()

# Close MongoDB connection on shutdown
@app.on_event("shutdown")
async def shutdown_db_client():
    await close_mongo_connection()

@app.get("/")
async def root():
    return {"message": "Welcome to the FHIR Service API"}

@app.get("/health")
async def health_check():
    return {"status": "healthy"}
