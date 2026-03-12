from fastapi import APIRouter, Depends, HTTPException, Query, Path, Request, Body
from typing import Dict, List, Any, Optional
import logging
import json
import sys
import os

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR router
from services.shared.fhir import create_fhir_router, FHIRRouterConfig

# Import the FHIR service factory and other services
from app.services.fhir_service_factory import get_fhir_service
from app.services.patient_service import get_patient_service
from app.core.auth import get_token_payload
from app.core.config import settings

logger = logging.getLogger(__name__)

# Determine which FHIR service class to use based on configuration
if settings.USE_GOOGLE_HEALTHCARE_API:
    from app.services.google_fhir_service import GooglePatientFHIRService as FHIRServiceClass
else:
    from app.services.fhir_service import PatientFHIRService as FHIRServiceClass

# Create the FHIR router configuration for Patient resources
patient_config = FHIRRouterConfig(
    resource_type="Patient",
    service_class=FHIRServiceClass,
    get_token_payload=get_token_payload,
    prefix="",  # Empty prefix because we're already at /api/fhir
    tags=["FHIR Patient"]
)

# Create the Patient router using the factory function
patient_router = create_fhir_router(patient_config)

# Create a router for generic FHIR resources
generic_router = APIRouter()
# We'll use get_patient_service() function to get the properly initialized service
# This will be called in the route handlers as needed

# Combine the routers
router = APIRouter()
router.include_router(patient_router)
router.include_router(generic_router)

# Add very visible logging to show that the router was created
logger.info("=== PATIENT SERVICE FHIR ROUTER CREATED ===")
logger.info(f"Patient Resource Type: {patient_config.resource_type}")
logger.info(f"Patient Prefix: {patient_config.prefix}")
logger.info(f"Patient Tags: {patient_config.tags}")
logger.info("=== END PATIENT SERVICE FHIR ROUTER ===")

# The Patient router now has the following endpoints:
# POST /Patient - Create a new patient
# GET /Patient/{id} - Get a patient by ID
# PUT /Patient/{id} - Update a patient
# DELETE /Patient/{id} - Delete a patient
# GET /Patient - Search for patients

# Generic FHIR resource handler - this will catch all requests for non-Patient resources
@generic_router.post("/{resource_type}", response_model=Dict[str, Any])
async def create_fhir_resource(
    resource_type: str,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Generic handler for creating any FHIR resource except Patient.
    This endpoint handles requests from the FHIR service for any resource type.
    """
    try:
        # Skip Patient resources as they are handled by the Patient router
        if resource_type == "Patient":
            raise HTTPException(status_code=404, detail=f"Resource type {resource_type} not found")

        # Add very visible logging
        print(f"\n\n==== PATIENT SERVICE RECEIVED CREATE REQUEST FOR {resource_type} ====")
        print(f"Resource: {resource}")
        print(f"Headers: {dict(token_payload)}")
        print(f"==== END PATIENT SERVICE REQUEST ====\n\n")

        logger.info(f"Creating {resource_type}: {resource}")

        # For other resource types, just echo back with an ID
        resource["id"] = f"test-{resource_type.lower()}-id"
        return resource
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error creating {resource_type}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

# Generic GET handler for any FHIR resource except Patient
@generic_router.get("/{resource_type}/{resource_id}", response_model=Dict[str, Any])
async def get_fhir_resource(
    resource_type: str,
    resource_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Generic handler for getting any FHIR resource by ID except Patient.
    """
    try:
        # Skip Patient resources as they are handled by the Patient router
        if resource_type == "Patient":
            raise HTTPException(status_code=404, detail=f"Resource type {resource_type} not found")

        # Add very visible logging
        print(f"\n\n==== PATIENT SERVICE RECEIVED GET REQUEST FOR {resource_type}/{resource_id} ====")
        print(f"Headers: {dict(token_payload)}")
        print(f"==== END PATIENT SERVICE REQUEST ====\n\n")

        logger.info(f"Getting {resource_type}/{resource_id}")

        # For other resource types, return a generic response
        return {
            "resourceType": resource_type,
            "id": resource_id,
            "status": "active",
            "subject": {"reference": "Patient/test-patient-id"}
        }
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting {resource_type}/{resource_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

# Add generic PUT handler for any FHIR resource except Patient
@generic_router.put("/{resource_type}/{resource_id}", response_model=Dict[str, Any])
async def update_fhir_resource(
    resource_type: str,
    resource_id: str,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Generic handler for updating any FHIR resource except Patient.
    """
    try:
        # Skip Patient resources as they are handled by the Patient router
        if resource_type == "Patient":
            raise HTTPException(status_code=404, detail=f"Resource type {resource_type} not found")

        # Add very visible logging
        print(f"\n\n==== PATIENT SERVICE RECEIVED PUT REQUEST FOR {resource_type}/{resource_id} ====")
        print(f"Resource: {resource}")
        print(f"Headers: {dict(token_payload)}")
        print(f"==== END PATIENT SERVICE REQUEST ====\n\n")

        logger.info(f"Updating {resource_type}/{resource_id}: {resource}")

        # Ensure the resource has the correct ID
        resource["id"] = resource_id

        # Return the updated resource
        return resource
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating {resource_type}/{resource_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

# Add generic DELETE handler for any FHIR resource except Patient
@generic_router.delete("/{resource_type}/{resource_id}", response_model=Dict[str, Any])
async def delete_fhir_resource(
    resource_type: str,
    resource_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Generic handler for deleting any FHIR resource except Patient.
    """
    try:
        # Skip Patient resources as they are handled by the Patient router
        if resource_type == "Patient":
            raise HTTPException(status_code=404, detail=f"Resource type {resource_type} not found")

        # Add very visible logging
        print(f"\n\n==== PATIENT SERVICE RECEIVED DELETE REQUEST FOR {resource_type}/{resource_id} ====")
        print(f"Headers: {dict(token_payload)}")
        print(f"==== END PATIENT SERVICE REQUEST ====\n\n")

        logger.info(f"Deleting {resource_type}/{resource_id}")

        # Return a success message
        return {"message": f"{resource_type} with ID {resource_id} deleted successfully"}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting {resource_type}/{resource_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

# Add generic search handler for any FHIR resource except Patient
@generic_router.get("/{resource_type}", response_model=List[Dict[str, Any]])
async def search_fhir_resources(
    resource_type: str,
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Generic handler for searching any FHIR resources except Patient.
    """
    try:
        # Skip Patient resources as they are handled by the Patient router
        if resource_type == "Patient":
            raise HTTPException(status_code=404, detail=f"Resource type {resource_type} not found")

        # Get query parameters
        params = dict(request.query_params)

        # Add very visible logging
        print(f"\n\n==== PATIENT SERVICE RECEIVED SEARCH REQUEST FOR {resource_type} ====")
        print(f"Query params: {params}")
        print(f"Headers: {dict(token_payload)}")
        print(f"==== END PATIENT SERVICE REQUEST ====\n\n")

        logger.info(f"Searching {resource_type} with params: {params}")

        # For other resource types, return a generic response
        return [{
            "resourceType": resource_type,
            "id": f"test-{resource_type.lower()}-id",
            "status": "active",
            "subject": {"reference": "Patient/test-patient-id"}
        }]
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error searching {resource_type}: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))
