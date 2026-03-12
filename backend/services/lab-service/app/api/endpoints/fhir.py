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
from app.services.fhir_service import DiagnosticReportFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create the FHIR router configuration for DiagnosticReport resources
diagnostic_report_config = FHIRRouterConfig(
    resource_type="DiagnosticReport",
    service_class=DiagnosticReportFHIRService,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR DiagnosticReport"]
)

# Create the router using the factory function
router = create_fhir_router(diagnostic_report_config)

# Add very visible logging to show that the router was created
logger.info("=== LAB SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Resource Type: {diagnostic_report_config.resource_type}")
logger.info(f"Prefix: {diagnostic_report_config.prefix}")
logger.info(f"Tags: {diagnostic_report_config.tags}")
logger.info("=== END LAB SERVICE FHIR ROUTER ===")

# The router now has the following endpoints:
# POST /DiagnosticReport - Create a new diagnostic report
# GET /DiagnosticReport/{id} - Get a diagnostic report by ID
# PUT /DiagnosticReport/{id} - Update a diagnostic report
# DELETE /DiagnosticReport/{id} - Delete a diagnostic report
# GET /DiagnosticReport - Search for diagnostic reports
