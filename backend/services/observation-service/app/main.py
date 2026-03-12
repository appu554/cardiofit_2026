"""
FastAPI application for the Observation Service with GraphQL support.

This module initializes the FastAPI application and sets up the GraphQL endpoint
using Strawberry GraphQL.
"""

import os
import sys
import logging
from contextlib import asynccontextmanager
from typing import Optional, List, Dict, Any
from fastapi import FastAPI, Request, Depends, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from starlette.middleware import Middleware
from starlette.middleware.cors import CORSMiddleware as CORSMiddlewareStarlette
import strawberry
from strawberry.fastapi import GraphQLRouter

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import GraphQL schema
from app.graphql import schema # Use the schema from app/graphql/__init__.py (which points to app/graphql/schema.py)

# Create GraphQL app
graphql_app = GraphQLRouter(schema, graphiql=True)

# Import local modules
try:
    from app.core.config import settings
    # from app.db.mongodb import connect_to_mongo, close_mongo_connection # MongoDB not used
    from app.services.fhir_service import initialize_fhir_service
    from app.api.api import api_router
except ImportError as e:
    print(f"Error importing local modules: {e}")
    raise

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Define lifespan context manager for startup and shutdown events
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Initialize FHIR service
    # await connect_to_mongo() # MongoDB not used
    logger.info("Initializing FHIR service...")
    await initialize_fhir_service()
    logger.info("FHIR service initialized")

    yield

    # Shutdown: 
    # await close_mongo_connection() # MongoDB not used

# Configure CORS middleware
middleware = [
    Middleware(
        CORSMiddlewareStarlette,
        allow_origins=["*"],  # In production, replace with specific origins
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
]

# Initialize FastAPI app with lifespan and settings
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Observation Service API with Apollo Federation",
    version="1.0.0",
    middleware=middleware,
    lifespan=lifespan
)

# Add GraphQL endpoint
app.include_router(graphql_app, prefix="/graphql")

# Add API router (includes federation endpoint)
app.include_router(api_router, prefix="/api")

# Add health check endpoint
@app.get("/health")
async def health_check():
    """Health check endpoint for the service."""
    try:
        # Check FHIR service
        from app.services.fhir_service_factory import get_fhir_service
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise Exception("FHIR service not available")

        return {
            "status": "healthy",
            "version": "1.0.0",
            "fhir_service": "available"
        }
    except Exception as e:
        raise HTTPException(
            status_code=503,
            detail={
                "status": "unhealthy",
                "error": str(e)
            }
        )

# Root endpoint
@app.get("/")
async def root():
    """Root endpoint that returns service information."""
    return {
        "service": "Observation Service",
        "version": "1.0.0",
        "graphql_endpoint": "/graphql",
        "federation_endpoint": "/api/federation",
        "health_check": "/health"
    }
