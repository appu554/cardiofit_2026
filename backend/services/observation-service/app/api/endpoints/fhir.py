from fastapi import APIRouter, Depends, Request, HTTPException
from typing import Dict, List, Any, Optional
import logging
import sys
import os

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR router
try:
    from shared.fhir.router import create_fhir_router, FHIRRouterConfig
except ImportError as e:
    print(f"Error importing FHIR router: {e}")
    print(f"Current sys.path: {sys.path}")
    print(f"Backend directory: {backend_dir}")
    if os.path.exists(backend_dir):
        print(f"Shared directory exists: {os.path.exists(os.path.join(backend_dir, 'shared'))}")
        if os.path.exists(os.path.join(backend_dir, 'shared')):
            print(f"Shared directory contents: {os.listdir(os.path.join(backend_dir, 'shared'))}")
    raise

# Import the FHIR service
try:
    from app.services.fhir_service import ObservationFHIRService
    from app.core.auth import get_token_payload
except ImportError as e:
    print(f"Error importing local modules: {e}")
    raise

logger = logging.getLogger(__name__)

# Create the FHIR router configuration
try:
    config = FHIRRouterConfig(
        resource_type="Observation",
        service_class=ObservationFHIRService,
        get_token_payload=get_token_payload,
        prefix="",  # Empty prefix because we're already at /api/fhir
        tags=["FHIR Observation"]
    )

    # Create the router using the factory function
    router = create_fhir_router(config)

except Exception as e:
    print(f"Error creating FHIR router: {e}")
    # Create a minimal router that will return an error
    router = APIRouter()
    
    @router.get("/{path:path}")
    @router.post("/{path:path}")
    @router.put("/{path:path}")
    @router.delete("/{path:path}")
    async def fhir_error():
        raise HTTPException(
            status_code=500,
            detail="FHIR router initialization failed. Please check the server logs."
        )

# Add very visible logging to show that the router was created
logger.info("=== OBSERVATION SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Resource Type: {config.resource_type}")
logger.info(f"Prefix: {config.prefix}")
logger.info(f"Tags: {config.tags}")
logger.info("=== END OBSERVATION SERVICE FHIR ROUTER ===")

# The router now has the following endpoints:
# POST /Observation - Create a new observation
# GET /Observation/{id} - Get an observation by ID
# PUT /Observation/{id} - Update an observation
# DELETE /Observation/{id} - Delete an observation
# GET /Observation - Search for observations
