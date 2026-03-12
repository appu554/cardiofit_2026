"""
Direct FHIR endpoint for the API Gateway.

This module provides endpoints for direct FHIR operations without any validation.
It implements the flow: API Gateway > Auth > Microservice > Google Healthcare API.
"""

from fastapi import APIRouter, HTTPException, Header, Request
from typing import Dict, Any, Optional
import httpx
import logging
import json
from app.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create router
router = APIRouter(prefix="/direct-fhir", tags=["Direct FHIR"])

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

@router.post("/Patient")
async def create_patient(
    request: Request,
    authorization: Optional[str] = Header(None)
):
    """
    Create a patient directly without any validation.

    This endpoint forwards the request directly to the Patient service's FHIR endpoint,
    which then forwards it to Google Healthcare API.
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

        # Forward the request directly to the Patient service's FHIR endpoint
        async with httpx.AsyncClient(follow_redirects=True) as client:
            response = await client.post(
                f"{settings.PATIENT_SERVICE_URL}/api/fhir/Patient",
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

@router.get("/Patient")
async def search_patients(
    request: Request,
    authorization: Optional[str] = Header(None)
):
    """
    Search for patients directly without any validation.

    This endpoint forwards the request directly to the Patient service's FHIR endpoint,
    which then forwards it to Google Healthcare API.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Extract token and get user information
    token = auth_header.replace("Bearer ", "") if auth_header.startswith("Bearer ") else auth_header
    user_info = await get_user_info(token)

    try:
        # Get query parameters
        query_params = dict(request.query_params)

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

        # Forward the request directly to the Patient service's FHIR endpoint
        async with httpx.AsyncClient(follow_redirects=True) as client:
            response = await client.get(
                f"{settings.PATIENT_SERVICE_URL}/api/fhir/Patient",
                params=query_params,
                headers=headers
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



@router.get("/Patient/{patient_id}")
async def get_patient(
    patient_id: str,
    authorization: Optional[str] = Header(None)
):
    """
    Get a patient by ID directly without any validation.

    This endpoint forwards the request directly to the Patient service's FHIR endpoint,
    which then forwards it to Google Healthcare API.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Extract token and get user information
    token = auth_header.replace("Bearer ", "") if auth_header.startswith("Bearer ") else auth_header
    user_info = await get_user_info(token)

    try:
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

        # Forward the request directly to the Patient service's FHIR endpoint
        async with httpx.AsyncClient(follow_redirects=True) as client:
            response = await client.get(
                f"{settings.PATIENT_SERVICE_URL}/api/fhir/Patient/{patient_id}",
                headers=headers
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error getting patient: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error getting patient: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/Patient/{patient_id}")
async def update_patient(
    patient_id: str,
    request: Request,
    authorization: Optional[str] = Header(None)
):
    """
    Update a patient directly without any validation.

    This endpoint forwards the request directly to the Patient service's FHIR endpoint,
    which then forwards it to Google Healthcare API.
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

        # Forward the request directly to the Patient service's FHIR endpoint
        async with httpx.AsyncClient(follow_redirects=True) as client:
            response = await client.put(
                f"{settings.PATIENT_SERVICE_URL}/api/fhir/Patient/{patient_id}",
                content=body,
                headers=headers
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
