"""
Raw FHIR endpoints for the API Gateway.

This module provides endpoints for direct FHIR operations without GraphQL schema validation.
"""

from fastapi import APIRouter, HTTPException, Body, Header, Request
from typing import Dict, Any, Optional
import httpx
import logging
import json
from app.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create router
router = APIRouter(prefix="/raw-fhir", tags=["Raw FHIR"])

async def get_user_info(token: str) -> Dict[str, Any]:
    """
    Get user information from the Auth Service.

    Args:
        token: The JWT token

    Returns:
        The user information
    """
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{settings.AUTH_SERVICE_URL}/api/auth/verify",
                headers={"Authorization": f"Bearer {token}"}
            )

            if response.status_code != 200:
                logger.error(f"Token validation failed: {response.status_code} - {response.text}")
                raise HTTPException(
                    status_code=401,
                    detail="Invalid authentication credentials",
                    headers={"WWW-Authenticate": "Bearer"}
                )

            result = response.json()

            if not result.get("valid", False):
                logger.error(f"Token validation failed: {result.get('error', 'Unknown error')}")
                raise HTTPException(
                    status_code=401,
                    detail=result.get("error", "Invalid token"),
                    headers={"WWW-Authenticate": "Bearer"}
                )

            # Get the user info from the response
            user_info = result.get("user", {})

            # Extract RBAC information
            user_info["roles"] = user_info.get("roles", [])
            user_info["role"] = user_info.get("role", "authenticated")
            user_info["permissions"] = user_info.get("permissions", [])

            return user_info
    except httpx.RequestError as e:
        logger.error(f"Error calling Auth Service: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Error calling Auth Service: {str(e)}"
        )

@router.post("/Patient", status_code=201)
async def create_patient(
    patient_data: Dict[str, Any] = Body(...),
    authorization: Optional[str] = Header(None)
):
    """
    Create a patient directly without GraphQL schema validation.

    This endpoint accepts any valid FHIR Patient resource and forwards it to the Patient service.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Extract token and get user information
    token = auth_header.replace("Bearer ", "") if auth_header.startswith("Bearer ") else auth_header
    user_info = await get_user_info(token)

    # Ensure resourceType is set
    if "resourceType" not in patient_data:
        patient_data["resourceType"] = "Patient"

    # Forward the request directly to the Patient service
    try:
        async with httpx.AsyncClient(follow_redirects=True) as client:
            # Create headers with user information
            headers = {
                "Authorization": auth_header,
                "Content-Type": "application/json",
                "X-User-ID": str(user_info.get("id", "")),
                "X-User-Email": str(user_info.get("email", "")),
                "X-User-Role": str(user_info.get("role", "authenticated")),
                "X-User-Roles": ",".join(user_info.get("roles", [])),
                "X-User-Permissions": ",".join(user_info.get("permissions", []))
            }

            # Add user name if available
            if user_info.get("name"):
                headers["X-User-Name"] = str(user_info.get("name", ""))
            elif user_info.get("full_name"):
                headers["X-User-Name"] = str(user_info.get("full_name", ""))

            response = await client.post(
                f"{settings.PATIENT_SERVICE_URL}/api/patients",
                json=patient_data,
                headers=headers
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error creating patient: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error creating patient: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/direct-patient", status_code=201)
async def create_direct_patient(
    request: Request,
    authorization: Optional[str] = Header(None)
):
    """
    Create a patient with absolutely no validation.

    This endpoint accepts any JSON and forwards it directly to the Patient service.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Extract token and get user information
    token = auth_header.replace("Bearer ", "") if auth_header.startswith("Bearer ") else auth_header
    user_info = await get_user_info(token)

    try:
        # Get the raw request body
        body = await request.body()

        # Forward the raw request directly to the Patient service
        async with httpx.AsyncClient(follow_redirects=True) as client:
            # Create headers with user information
            headers = {
                "Authorization": auth_header,
                "Content-Type": "application/json",
                "X-User-ID": str(user_info.get("id", "")),
                "X-User-Email": str(user_info.get("email", "")),
                "X-User-Role": str(user_info.get("role", "authenticated")),
                "X-User-Roles": ",".join(user_info.get("roles", [])),
                "X-User-Permissions": ",".join(user_info.get("permissions", []))
            }

            # Add user name if available
            if user_info.get("name"):
                headers["X-User-Name"] = str(user_info.get("name", ""))
            elif user_info.get("full_name"):
                headers["X-User-Name"] = str(user_info.get("full_name", ""))

            response = await client.post(
                f"{settings.PATIENT_SERVICE_URL}/api/patients",
                content=body,
                headers=headers
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error creating patient: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error creating patient: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/Patient/{patient_id}")
async def get_patient(
    patient_id: str,
    authorization: Optional[str] = Header(None)
):
    """
    Get a patient by ID directly without GraphQL schema validation.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Extract token and get user information
    token = auth_header.replace("Bearer ", "") if auth_header.startswith("Bearer ") else auth_header
    user_info = await get_user_info(token)

    # Forward the request directly to the Patient service
    try:
        async with httpx.AsyncClient(follow_redirects=True) as client:
            # Create headers with user information
            headers = {
                "Authorization": auth_header,
                "X-User-ID": str(user_info.get("id", "")),
                "X-User-Email": str(user_info.get("email", "")),
                "X-User-Role": str(user_info.get("role", "authenticated")),
                "X-User-Roles": ",".join(user_info.get("roles", [])),
                "X-User-Permissions": ",".join(user_info.get("permissions", []))
            }

            # Add user name if available
            if user_info.get("name"):
                headers["X-User-Name"] = str(user_info.get("name", ""))
            elif user_info.get("full_name"):
                headers["X-User-Name"] = str(user_info.get("full_name", ""))

            response = await client.get(
                f"{settings.PATIENT_SERVICE_URL}/api/patients/{patient_id}",
                headers=headers
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except httpx.HTTPStatusError as e:
        if e.response.status_code == 404:
            raise HTTPException(status_code=404, detail="Patient not found")
        logger.error(f"HTTP error getting patient: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error getting patient: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/Patient/{patient_id}")
async def update_patient(
    patient_id: str,
    patient_data: Dict[str, Any] = Body(...),
    authorization: Optional[str] = Header(None)
):
    """
    Update a patient directly without GraphQL schema validation.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Ensure resourceType and id are set
    if "resourceType" not in patient_data:
        patient_data["resourceType"] = "Patient"
    patient_data["id"] = patient_id

    # Forward the request directly to the Patient service
    try:
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{settings.PATIENT_SERVICE_URL}/api/patients/{patient_id}",
                json=patient_data,
                headers={"Authorization": auth_header}
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error updating patient: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error updating patient: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/Patient")
async def search_patients(
    authorization: Optional[str] = Header(None)
):
    """
    Search for patients directly without GraphQL schema validation.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Forward the request directly to the Patient service
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.PATIENT_SERVICE_URL}/api/patients",
                headers={"Authorization": auth_header}
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error searching patients: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error searching patients: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))
