"""
Template for implementing the shared FHIR router in a microservice.

This template can be used as a starting point for implementing the shared FHIR router
in a microservice. Copy this file to your microservice's app/api/endpoints/fhir.py
and customize it for your resource type.
"""

from fastapi import Depends, Request
import logging
import sys
import os

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR router
from services.shared.fhir import create_fhir_router, FHIRRouterConfig, MockFHIRService

# Import your FHIR service implementation
# from app.services.fhir_service import YourResourceFHIRService

# Import your token payload function
# from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create the FHIR router configuration
config = FHIRRouterConfig(
    resource_type="YourResource",  # Replace with your resource type (e.g., "Patient", "Observation")
    service_class=MockFHIRService,  # Replace with your service class
    # get_token_payload=get_token_payload,  # Uncomment and replace with your token payload function
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR YourResource"]  # Replace with your resource type
)

# Create the router using the factory function
router = create_fhir_router(config)

# Add very visible logging to show that the router was created
logger.info(f"=== {config.resource_type.upper()} SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Resource Type: {config.resource_type}")
logger.info(f"Prefix: {config.prefix}")
logger.info(f"Tags: {config.tags}")
logger.info(f"=== END {config.resource_type.upper()} SERVICE FHIR ROUTER ===")

# The router now has the following endpoints:
# POST /{resource_type} - Create a new resource
# GET /{resource_type}/{id} - Get a resource by ID
# PUT /{resource_type}/{id} - Update a resource
# DELETE /{resource_type}/{id} - Delete a resource
# GET /{resource_type} - Search for resources
