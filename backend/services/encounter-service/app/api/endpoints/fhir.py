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
from app.services.fhir_service import EncounterFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create the FHIR router configuration for Encounter resources
encounter_config = FHIRRouterConfig(
    resource_type="Encounter",
    service_class=EncounterFHIRService,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR Encounter"]
)

# Create the router using the factory function
router = create_fhir_router(encounter_config)

# Add very visible logging to show that the router was created
logger.info("=== ENCOUNTER SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Resource Type: {encounter_config.resource_type}")
logger.info(f"Prefix: {encounter_config.prefix}")
logger.info(f"Tags: {encounter_config.tags}")
logger.info("=== END ENCOUNTER SERVICE FHIR ROUTER ===")

# The router now has the following endpoints:
# POST /Encounter - Create a new encounter
# GET /Encounter/{id} - Get an encounter by ID
# PUT /Encounter/{id} - Update an encounter
# DELETE /Encounter/{id} - Delete an encounter
# GET /Encounter - Search for encounters
