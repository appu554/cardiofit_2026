"""
FHIR Endpoint for MedicationRequest resources.

This module provides a FHIR-compliant API for MedicationRequest resources.
"""

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
from app.services.medication_request_service import MedicationRequestFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create the FHIR router configuration for MedicationRequest resources
medication_request_config = FHIRRouterConfig(
    resource_type="MedicationRequest",
    service_class=MedicationRequestFHIRService,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR MedicationRequest"]
)

# Create the router using the factory function
router = create_fhir_router(medication_request_config)

# Add very visible logging to show that the router was created
logger.info("=== MEDICATION SERVICE FHIR MEDICATIONREQUEST ROUTER CREATED ===")
logger.info(f"Resource Type: {medication_request_config.resource_type}")
logger.info(f"Prefix: {medication_request_config.prefix}")
logger.info(f"Tags: {medication_request_config.tags}")
logger.info("=== END MEDICATION SERVICE FHIR MEDICATIONREQUEST ROUTER ===")

# The router now has the following endpoints:
# POST /MedicationRequest - Create a new medication request
# GET /MedicationRequest/{id} - Get a medication request by ID
# PUT /MedicationRequest/{id} - Update a medication request
# DELETE /MedicationRequest/{id} - Delete a medication request
# GET /MedicationRequest - Search for medication requests
