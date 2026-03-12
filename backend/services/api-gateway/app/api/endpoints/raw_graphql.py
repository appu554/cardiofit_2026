"""
Raw GraphQL endpoint for the API Gateway.

This module provides an endpoint for direct GraphQL operations without schema validation.
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
router = APIRouter(prefix="/raw-graphql", tags=["Raw GraphQL"])

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

@router.post("")
async def raw_graphql(
    request: Request,
    authorization: Optional[str] = Header(None)
):
    """
    Process GraphQL requests directly without schema validation.

    This endpoint accepts any GraphQL query or mutation and forwards it to the appropriate microservice.
    """
    # Get authorization header
    auth_header = authorization
    if not auth_header:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # Extract token and get user information
    token = auth_header.replace("Bearer ", "") if auth_header.startswith("Bearer ") else auth_header
    user_info = await get_user_info(token)

    # Get the request body
    body = await request.body()
    try:
        # Parse the request body
        data = json.loads(body)

        # Extract the GraphQL query and variables
        query = data.get("query", "")
        variables = data.get("variables", {})
        operation_name = data.get("operationName")

        logger.info(f"Raw GraphQL request: {operation_name}")
        logger.info(f"Query: {query[:100]}...")

        # Route all GraphQL requests to the Apollo Federation Gateway
        target_url = settings.APOLLO_FEDERATION_URL
        logger.info(f"Routing to Apollo Federation Gateway at {target_url}")

        # Forward the request to the Apollo Federation Gateway
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

            # Send request to the Apollo Federation Gateway
            response = await client.post(
                target_url,
                json=data,
                headers=headers
            )

            # Raise exception for HTTP errors
            response.raise_for_status()

            # Return the response
            return response.json()
    except json.JSONDecodeError:
        logger.error("Invalid JSON in request body")
        raise HTTPException(status_code=400, detail="Invalid JSON in request body")
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error processing GraphQL request: {str(e)}")
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)
    except Exception as e:
        logger.error(f"Error processing GraphQL request: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

def determine_target_service(query: str, operation_name: Optional[str] = None) -> str:
    """
    Determine which microservice to route to based on the GraphQL query.

    Args:
        query: The GraphQL query
        operation_name: The operation name

    Returns:
        The name of the target service
    """
    query = query.lower()

    # Check operation name first
    if operation_name:
        operation_name = operation_name.lower()
        if "patient" in operation_name:
            return "patient"
        elif "observation" in operation_name:
            return "observation"
        elif "condition" in operation_name:
            return "condition"
        elif "medication" in operation_name:
            return "medication"
        elif "encounter" in operation_name:
            return "encounter"
        elif "timeline" in operation_name:
            return "timeline"

    # Check query content
    if "patient" in query:
        return "patient"
    elif "observation" in query:
        return "observation"
    elif "condition" in query:
        return "condition"
    elif "medication" in query:
        return "medication"
    elif "encounter" in query:
        return "encounter"
    elif "timeline" in query:
        return "timeline"

    # Default to patient service
    return "patient"

def get_service_url(service_name: str) -> str:
    """
    Get the URL for a service.

    Args:
        service_name: The name of the service

    Returns:
        The URL for the service
    """
    if service_name == "patient":
        return settings.PATIENT_SERVICE_URL
    elif service_name == "observation":
        return settings.OBSERVATION_SERVICE_URL
    elif service_name == "condition":
        return settings.CONDITION_SERVICE_URL
    elif service_name == "medication":
        return settings.MEDICATION_SERVICE_URL
    elif service_name == "encounter":
        return settings.ENCOUNTER_SERVICE_URL
    elif service_name == "timeline":
        return settings.TIMELINE_SERVICE_URL
    else:
        return settings.PATIENT_SERVICE_URL
