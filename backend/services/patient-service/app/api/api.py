from fastapi import APIRouter, Request, Depends, Body, HTTPException, status
from app.api.endpoints import patient, hl7, fhir, graphql, graphql_explorer, federation, webhooks, context
from typing import Dict, Any, List, Optional
from app.core.auth import get_token_payload
import logging
from fastapi.responses import HTMLResponse

# Import the patient service
from app.services.patient_service import get_patient_service

# Configure logging
logger = logging.getLogger(__name__)

api_router = APIRouter()

api_router.include_router(patient.router, prefix="/patients", tags=["Patients"])
api_router.include_router(hl7.router, prefix="/hl7", tags=["HL7"])
api_router.include_router(fhir.router, prefix="/fhir", tags=["FHIR"])
api_router.include_router(graphql.router, prefix="/graphql", tags=["GraphQL"])
api_router.include_router(graphql_explorer.router, prefix="/graphql/explorer", tags=["GraphQL Explorer"])
api_router.include_router(federation.router, prefix="/federation", tags=["Federation"])
api_router.include_router(webhooks.router, prefix="/webhooks", tags=["Webhooks"])
api_router.include_router(context.router, prefix="/context", tags=["Context Service"])

# Add direct handlers for FHIR endpoints
@api_router.post("/fhir/Patient", response_model=Dict[str, Any], tags=["FHIR"])
async def create_patient_fhir(
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Create a new patient using FHIR format.
    This endpoint handles requests from the FHIR service.
    """
    try:
        logger.info(f"Creating Patient resource")

        # Get the patient service
        patient_service = await get_patient_service()

        # Create the patient using the FHIR service
        created_patient = await patient_service.fhir_service.create_resource(resource)

        if not created_patient:
            logger.error("Failed to create Patient resource")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail="Failed to create Patient resource"
            )

        logger.info(f"Created Patient resource with ID {created_patient.get('id')}")
        return created_patient
    except Exception as e:
        logger.error(f"Error creating Patient resource: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error creating Patient resource: {str(e)}"
        )

@api_router.get("/fhir/Patient/{patient_id}", response_model=Dict[str, Any], tags=["FHIR"])
async def get_patient_fhir(
    patient_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Get a patient by ID using FHIR format.
    This endpoint handles requests from the FHIR service.
    """
    try:
        logger.info(f"Getting Patient resource with ID {patient_id}")

        # Get the patient service
        patient_service = await get_patient_service()

        # Get the patient using the FHIR service
        patient = await patient_service.fhir_service.get_resource(patient_id)

        if not patient:
            logger.error(f"Patient resource with ID {patient_id} not found")
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Patient resource with ID {patient_id} not found"
            )

        logger.info(f"Retrieved Patient resource with ID {patient_id}")
        return patient
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting Patient resource with ID {patient_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting Patient resource with ID {patient_id}: {str(e)}"
        )

@api_router.put("/fhir/Patient/{patient_id}", response_model=Dict[str, Any], tags=["FHIR"])
async def update_patient_fhir(
    patient_id: str,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Update a patient using FHIR format.
    This endpoint handles requests from the FHIR service.
    """
    try:
        logger.info(f"Updating Patient resource with ID {patient_id}")

        # Ensure the resource has the correct ID
        resource["id"] = patient_id
        resource["resourceType"] = "Patient"

        # Get the patient service
        patient_service = await get_patient_service()

        # Update the patient using the FHIR service
        updated_patient = await patient_service.fhir_service.update_resource(patient_id, resource)

        if not updated_patient:
            logger.error(f"Patient resource with ID {patient_id} not found or could not be updated")
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Patient resource with ID {patient_id} not found or could not be updated"
            )

        logger.info(f"Updated Patient resource with ID {patient_id}")
        return updated_patient
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating Patient resource with ID {patient_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error updating Patient resource with ID {patient_id}: {str(e)}"
        )

@api_router.delete("/fhir/Patient/{patient_id}", response_model=Dict[str, Any], tags=["FHIR"])
async def delete_patient_fhir(
    patient_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Delete a patient using FHIR format.
    This endpoint handles requests from the FHIR service.
    """
    try:
        logger.info(f"Deleting Patient resource with ID {patient_id}")

        # Get the patient service
        patient_service = await get_patient_service()

        # Delete the patient using the FHIR service
        deleted = await patient_service.fhir_service.delete_resource(patient_id)

        if not deleted:
            logger.error(f"Patient resource with ID {patient_id} not found or could not be deleted")
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Patient resource with ID {patient_id} not found or could not be deleted"
            )

        logger.info(f"Deleted Patient resource with ID {patient_id}")
        return {"message": f"Patient with ID {patient_id} deleted successfully"}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting Patient resource with ID {patient_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error deleting Patient resource with ID {patient_id}: {str(e)}"
        )

@api_router.get("/fhir/Patient", response_model=List[Dict[str, Any]], tags=["FHIR"])
async def search_patients_fhir(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Search for patients using FHIR format.
    This endpoint handles requests from the FHIR service.
    """
    try:
        # Get query parameters
        params = dict(request.query_params)

        logger.info(f"Searching Patient resources with params: {params}")

        # Get the patient service
        patient_service = await get_patient_service()

        # Search for patients using the FHIR service
        patients = await patient_service.fhir_service.search_resources(params)

        logger.info(f"Found {len(patients)} Patient resources")
        return patients
    except Exception as e:
        logger.error(f"Error searching Patient resources: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error searching Patient resources: {str(e)}"
        )
