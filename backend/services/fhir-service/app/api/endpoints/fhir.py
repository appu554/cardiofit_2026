from typing import Dict, List, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status, Request
import httpx
from app.core.auth import get_token_payload
from app.services.fhir_service import FHIRService
from app.core.integration import FHIRIntegrationLayer
from app.core.config import settings
from shared.models import (
    Patient, Observation, Condition, Encounter,
    MedicationRequest, DiagnosticReport
)

router = APIRouter()
fhir_service = FHIRService()
fhir_integration = FHIRIntegrationLayer()

# Generic FHIR Resource endpoints
@router.post("/{resource_type}", status_code=status.HTTP_201_CREATED)
async def create_resource(
    resource_type: str = Path(..., description="FHIR resource type"),
    resource: Dict[str, Any] = Body(..., description="FHIR resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new FHIR resource."""
    try:
        # Add very visible logging to show that FHIR service received the request
        print(f"\n\n==== FHIR SERVICE RECEIVED CREATE REQUEST FOR {resource_type} ====")
        print(f"Resource: {resource}")
        print(f"==== END FHIR SERVICE REQUEST ====\n\n")

        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.create_resource(resource_type, resource, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating resource: {str(e)}"
        )

# Resource-specific POST endpoints
@router.post("/Condition", status_code=status.HTTP_201_CREATED)
async def create_condition(
    resource: Dict[str, Any] = Body(..., description="FHIR Condition resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new Condition resource."""
    try:
        # Add very visible logging to show that FHIR service received the request
        print(f"\n\n==== FHIR SERVICE RECEIVED CREATE REQUEST FOR Condition ====")
        print(f"Resource: {resource}")
        print(f"==== END FHIR SERVICE REQUEST ====\n\n")

        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.create_resource("Condition", resource, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating Condition resource: {str(e)}"
        )

@router.post("/Observation", status_code=status.HTTP_201_CREATED)
async def create_observation(
    resource: Dict[str, Any] = Body(..., description="FHIR Observation resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new Observation resource."""
    try:
        # Add very visible logging to show that FHIR service received the request
        print(f"\n\n==== FHIR SERVICE RECEIVED CREATE REQUEST FOR Observation ====")
        print(f"Resource: {resource}")
        print(f"==== END FHIR SERVICE REQUEST ====\n\n")

        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.create_resource("Observation", resource, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating Observation resource: {str(e)}"
        )

@router.post("/MedicationRequest", status_code=status.HTTP_201_CREATED)
async def create_medication_request(
    resource: Dict[str, Any] = Body(..., description="FHIR MedicationRequest resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new MedicationRequest resource."""
    try:
        # Add very visible logging to show that FHIR service received the request
        print(f"\n\n==== FHIR SERVICE RECEIVED CREATE REQUEST FOR MedicationRequest ====")
        print(f"Resource: {resource}")
        print(f"==== END FHIR SERVICE REQUEST ====\n\n")

        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.create_resource("MedicationRequest", resource, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating MedicationRequest resource: {str(e)}"
        )

@router.post("/Encounter", status_code=status.HTTP_201_CREATED)
async def create_encounter(
    resource: Dict[str, Any] = Body(..., description="FHIR Encounter resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new Encounter resource."""
    try:
        # Add very visible logging to show that FHIR service received the request
        print(f"\n\n==== FHIR SERVICE RECEIVED CREATE REQUEST FOR Encounter ====")
        print(f"Resource: {resource}")
        print(f"==== END FHIR SERVICE REQUEST ====\n\n")

        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.create_resource("Encounter", resource, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating Encounter resource: {str(e)}"
        )

@router.post("/DiagnosticReport", status_code=status.HTTP_201_CREATED)
async def create_diagnostic_report(
    resource: Dict[str, Any] = Body(..., description="FHIR DiagnosticReport resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Create a new DiagnosticReport resource."""
    try:
        # Add very visible logging to show that FHIR service received the request
        print(f"\n\n==== FHIR SERVICE RECEIVED CREATE REQUEST FOR DiagnosticReport ====")
        print(f"Resource: {resource}")
        print(f"==== END FHIR SERVICE REQUEST ====\n\n")

        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.create_resource("DiagnosticReport", resource, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating DiagnosticReport resource: {str(e)}"
        )

@router.get("/{resource_type}/{id}")
async def get_resource(
    resource_type: str = Path(..., description="FHIR resource type"),
    id: str = Path(..., description="Resource ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Get a FHIR resource by ID."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        resource = await fhir_integration.get_resource(resource_type, id, auth_header)
        if not resource:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"{resource_type} with ID {id} not found"
            )
        return resource
    except Exception as e:
        if "404" in str(e):
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"{resource_type} with ID {id} not found"
            )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting resource: {str(e)}"
        )

@router.put("/{resource_type}/{id}")
async def update_resource(
    resource_type: str = Path(..., description="FHIR resource type"),
    id: str = Path(..., description="Resource ID"),
    resource: Dict[str, Any] = Body(..., description="FHIR resource"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Update a FHIR resource."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        updated_resource = await fhir_integration.update_resource(resource_type, id, resource, auth_header)
        if not updated_resource:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"{resource_type} with ID {id} not found"
            )
        return updated_resource
    except Exception as e:
        if "404" in str(e):
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"{resource_type} with ID {id} not found"
            )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error updating resource: {str(e)}"
        )

@router.delete("/{resource_type}/{id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_resource(
    resource_type: str = Path(..., description="FHIR resource type"),
    id: str = Path(..., description="Resource ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> None:
    """Delete a FHIR resource."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        success = await fhir_integration.delete_resource(resource_type, id, auth_header)
        if not success:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"{resource_type} with ID {id} not found"
            )
    except Exception as e:
        if "404" in str(e):
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"{resource_type} with ID {id} not found"
            )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error deleting resource: {str(e)}"
        )

@router.get("/{resource_type}")
async def search_resources(
    request: Request,
    resource_type: str = Path(..., description="FHIR resource type"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    # Common search parameters
    _id: Optional[str] = Query(None, description="Resource ID"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    # Other parameters will be extracted from the request query params
) -> List[Dict[str, Any]]:
    """Search for FHIR resources."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters directly from the request
        params = dict(request.query_params)

        # Add the explicitly defined parameters if they're not already in the query params
        if _id is not None and "_id" not in params:
            params["_id"] = str(_id)
        if "_count" not in params:
            params["_count"] = str(_count)
        if "_page" not in params:
            params["_page"] = str(_page)

        # Log the parameters for debugging
        print(f"\n\n==== FHIR SERVICE SEARCH PARAMETERS ====")
        print(f"Resource Type: {resource_type}")
        print(f"Query Parameters: {params}")
        print(f"==== END FHIR SERVICE SEARCH PARAMETERS ====\n\n")

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources(resource_type, params, auth_header)
    except Exception as e:
        print(f"Error searching resources: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching resources: {str(e)}"
        )

from app.core.permissions import require_permissions

# Patient-specific endpoints
@router.get("/Patient/{id}")
async def get_patient(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """Get a patient by ID."""
    # Permission checking is now handled by the API Gateway
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to route to the appropriate microservice
        patient = await fhir_integration.get_resource("Patient", id, auth_header)
        if not patient:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Patient with ID {id} not found"
            )
        return patient
    except HTTPException as he:
        # Re-raise HTTP exceptions
        raise he
    except Exception as e:
        if "404" in str(e):
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Patient with ID {id} not found"
            )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient: {str(e)}"
        )

@router.get("/Patient")
async def search_patients(
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    name: Optional[str] = Query(None, description="Patient name"),
    identifier: Optional[str] = Query(None, description="Patient identifier"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Search for patients."""
    # Permission checking is now handled by the API Gateway
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("Patient", params, auth_header)
    except HTTPException as he:
        # Re-raise HTTP exceptions
        raise he
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching patients: {str(e)}"
        )

@router.get("/Patient/{id}/Observation")
async def get_patient_observations(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    code: Optional[str] = Query(None, description="Observation code"),
    category: Optional[str] = Query(None, description="Observation category"),
    date: Optional[str] = Query(None, description="Observation date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get observations for a patient."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["id", "token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Add subject parameter for the patient
        params["subject"] = f"Patient/{id}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("Observation", params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting observations: {str(e)}"
        )

@router.get("/Patient/{id}/LabResults")
async def get_patient_lab_results(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    code: Optional[str] = Query(None, description="Lab test code"),
    date: Optional[str] = Query(None, description="Lab test date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get lab results for a patient.

    This endpoint is now a proxy to the Observation Microservice.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Call the Observation Microservice
        async with httpx.AsyncClient() as client:
            params = {}
            if code:
                params["code"] = code
            if date:
                params["date"] = date
            if _count:
                params["_count"] = _count
            if _page:
                params["_page"] = _page

            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/laboratory/patient/{id}",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                raise HTTPException(
                    status_code=response.status_code,
                    detail=f"Error from Observation Microservice: {response.text}"
                )

            # Return the observations directly
            return response.json()
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting lab results: {str(e)}"
        )

@router.get("/Patient/{id}/Condition")
async def get_patient_conditions(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    code: Optional[str] = Query(None, description="Condition code"),
    clinical_status: Optional[str] = Query(None, description="Clinical status"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get conditions for a patient."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["id", "token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Add subject parameter for the patient
        params["subject"] = f"Patient/{id}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("Condition", params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting conditions: {str(e)}"
        )

@router.get("/Patient/{id}/MedicationRequest")
async def get_patient_medications(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    status: Optional[str] = Query(None, description="Medication request status"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get medication requests for a patient."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["id", "token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Add subject parameter for the patient
        params["subject"] = f"Patient/{id}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("MedicationRequest", params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting medication requests: {str(e)}"
        )

@router.get("/Patient/{id}/DiagnosticReport")
async def get_patient_diagnostic_reports(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    code: Optional[str] = Query(None, description="Report code"),
    date: Optional[str] = Query(None, description="Report date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get diagnostic reports for a patient."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["id", "token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Add subject parameter for the patient
        params["subject"] = f"Patient/{id}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("DiagnosticReport", params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting diagnostic reports: {str(e)}"
        )

@router.get("/Patient/{id}/Encounter")
async def get_patient_encounters(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    status: Optional[str] = Query(None, description="Encounter status"),
    date: Optional[str] = Query(None, description="Encounter date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get encounters for a patient."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["id", "token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Add subject parameter for the patient
        params["subject"] = f"Patient/{id}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("Encounter", params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting encounters: {str(e)}"
        )

@router.get("/Patient/{id}/DocumentReference")
async def get_patient_documents(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    type: Optional[str] = Query(None, description="Document type"),
    date: Optional[str] = Query(None, description="Document date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
) -> List[Dict[str, Any]]:
    """Get document references for a patient."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["id", "token_payload", "fhir_service", "fhir_integration", "auth_header"] and v is not None}

        # Add subject parameter for the patient
        params["subject"] = f"Patient/{id}"

        # Use the integration layer to route to the appropriate microservice
        return await fhir_integration.search_resources("DocumentReference", params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting document references: {str(e)}"
        )

@router.get("/Patient/{id}/timeline")
async def get_patient_timeline(
    id: str = Path(..., description="Patient ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload),
) -> Dict[str, Any]:
    """Get a patient's timeline."""
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Use the integration layer to get the patient timeline
        return await fhir_integration.get_patient_timeline(id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting patient timeline: {str(e)}"
        )
