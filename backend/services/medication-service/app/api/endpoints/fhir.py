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
from app.services.fhir_service import MedicationFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create the FHIR router configuration for Medication resources
medication_config = FHIRRouterConfig(
    resource_type="Medication",
    service_class=MedicationFHIRService,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR Medication"]
)

# Create the router using the factory function
router = create_fhir_router(medication_config)

# Add very visible logging to show that the router was created
logger.info("=== MEDICATION SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Resource Type: {medication_config.resource_type}")
logger.info(f"Prefix: {medication_config.prefix}")
logger.info(f"Tags: {medication_config.tags}")
logger.info("=== END MEDICATION SERVICE FHIR ROUTER ===")

# The router now has the following endpoints:
# POST /Medication - Create a new medication
# GET /Medication/{id} - Get a medication by ID
# PUT /Medication/{id} - Update a medication
# DELETE /Medication/{id} - Delete a medication
# GET /Medication - Search for medications
