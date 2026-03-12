from fastapi import APIRouter, Depends, Request
from typing import Dict, List, Any, Optional
import logging
import sys
import os

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR router
from services.shared.fhir import create_fhir_router, FHIRRouterConfig

# Import the FHIR service
from app.services.fhir_service import ConditionFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create the FHIR router configuration
config = FHIRRouterConfig(
    resource_type="Condition",
    service_class=ConditionFHIRService,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR Condition"]
)

# Create the router using the factory function
router = create_fhir_router(config)

# Add very visible logging to show that the router was created
logger.info("=== CONDITION SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Resource Type: {config.resource_type}")
logger.info(f"Prefix: {config.prefix}")
logger.info(f"Tags: {config.tags}")
logger.info("=== END CONDITION SERVICE FHIR ROUTER ===")

# The router now has the following endpoints:
# POST /Condition - Create a new condition
# GET /Condition/{id} - Get a condition by ID
# PUT /Condition/{id} - Update a condition
# DELETE /Condition/{id} - Delete a condition
# GET /Condition - Search for conditions
