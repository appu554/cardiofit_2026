from fastapi import APIRouter, Request, Response, HTTPException, status
import httpx
import logging
import re
from app.config import settings
from app.auth import require_permissions, require_role
from urllib.parse import urljoin

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create router
router = APIRouter()

# Service route configuration
SERVICE_ROUTES = {
    # Auth service routes
    "auth": {
        "prefix": "/api/auth",
        "target": settings.AUTH_SERVICE_URL,
        "strip_prefix": True,
        "public_paths": [
            "/login",
            "/token",
            "/authorize",
            "/callback",
            "/verify"
        ]
    },
    # Apollo Federation GraphQL Gateway
    "apollo_federation": {
        "prefix": "/api/graphql",
        "target": settings.APOLLO_FEDERATION_URL.replace("/graphql", ""),
        "strip_prefix": False,
        "public_paths": ["/playground"]
    },
    # FHIR service routes - MUST be before other microservice routes to ensure proper routing
    "fhir": {
        "prefix": "/api/fhir",
        "target": settings.FHIR_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Patient service routes (direct access, not through FHIR)
    "patient": {
        "prefix": "/api/patients",
        "target": settings.PATIENT_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Notes service routes (direct access, not through FHIR)
    "notes": {
        "prefix": "/api/notes",
        "target": settings.NOTES_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Labs service routes (direct access, not through FHIR)
    "labs": {
        "prefix": "/api/labs",
        "target": settings.LABS_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Medication service routes (direct access, not through FHIR)
    "medication": {
        "prefix": "/api/medications",
        "target": settings.MEDICATION_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Observation service routes (direct access, not through FHIR)
    "observation": {
        "prefix": "/api/observations",
        "target": settings.OBSERVATION_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Condition service routes (direct access, not through FHIR)
    "condition": {
        "prefix": "/api/conditions",
        "target": settings.CONDITION_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Encounter service routes (direct access, not through FHIR)
    "encounter": {
        "prefix": "/api/encounters",
        "target": settings.ENCOUNTER_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Timeline service routes (direct access, not through FHIR)
    "timeline": {
        "prefix": "/api/timeline",
        "target": settings.TIMELINE_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Ingestion Service — strips /api/v1/ingest prefix, forwards remainder to :8140
    # e.g. /api/v1/ingest/fhir/Observation → /fhir/Observation
    #      /api/v1/ingest/devices          → /devices
    "ingestion": {
        "prefix": "/api/v1/ingest",
        "target": settings.INGESTION_SERVICE_URL,
        "strip_prefix": True,
        "public_paths": []
    },
    # Intake-Onboarding Service — strips /api/v1/intake prefix, forwards remainder to :8141
    # e.g. /api/v1/intake/fhir/Patient/$enroll → /fhir/Patient/$enroll
    "intake_onboarding": {
        "prefix": "/api/v1/intake",
        "target": settings.INTAKE_SERVICE_URL,
        "strip_prefix": True,
        "public_paths": []
    }
}

async def forward_request(
    request: Request,
    target_url: str,
    path: str,
    strip_prefix: bool = False,
    service_prefix: str = ""
) -> Response:
    """
    Forward the request to the target service and return the response.

    Args:
        request: The incoming request
        target_url: The base URL of the target service
        path: The path to forward to
        strip_prefix: Whether to strip the service prefix from the path
        service_prefix: The prefix to strip if strip_prefix is True

    Returns:
        The response from the target service
    """
    # Log the original path for debugging
    logger.info(f"Original path: {request.url.path}")
    logger.info(f"Path to forward: {path}")

    # Determine the target path
    target_path = path
    if strip_prefix and path.startswith(service_prefix):
        target_path = path[len(service_prefix):]
        # Ensure the path starts with a slash
        if not target_path.startswith('/'):
            target_path = '/' + target_path

    # Build the full URL
    url = urljoin(target_url, target_path)

    # Log the full URL for debugging
    logger.info(f"Forwarding to URL: {url}")

    # Get request details
    method = request.method
    headers = dict(request.headers)

    # Remove host header to avoid conflicts
    headers.pop('host', None)

    # Add user information to headers for downstream services
    if hasattr(request.state, 'user'):
        user = request.state.user

        # Add user ID
        headers['X-User-ID'] = str(user.get('id', ''))

        # Add user email
        headers['X-User-Email'] = str(user.get('email', ''))

        # Add user name if available
        if user.get('name'):
            headers['X-User-Name'] = str(user.get('name', ''))
        elif user.get('full_name'):
            headers['X-User-Name'] = str(user.get('full_name', ''))

        # Add user role
        if hasattr(request.state, 'user_role'):
            headers['X-User-Role'] = request.state.user_role
        else:
            headers['X-User-Role'] = str(user.get('role', 'authenticated'))

        # Add user roles as a comma-separated list
        if hasattr(request.state, 'user_roles') and request.state.user_roles:
            headers['X-User-Roles'] = ','.join(request.state.user_roles)
        elif user.get('roles'):
            headers['X-User-Roles'] = ','.join(user.get('roles', []))

        # Add user permissions as a comma-separated list
        if hasattr(request.state, 'user_permissions') and request.state.user_permissions:
            headers['X-User-Permissions'] = ','.join(request.state.user_permissions)
        elif user.get('permissions'):
            headers['X-User-Permissions'] = ','.join(user.get('permissions', []))

        # Add original token for services that might need it
        # This is optional and can be removed if not needed
        auth_header = request.headers.get("Authorization")
        if auth_header and auth_header.startswith("Bearer "):
            headers['X-Original-Token'] = auth_header.replace("Bearer ", "")

    # Get query parameters
    params = dict(request.query_params)

    # Get request body
    body = await request.body()

    # Log detailed information about the forwarded request
    logger.info(f"=== API GATEWAY FORWARDING REQUEST ===")
    logger.info(f"Path: {request.url.path}")
    logger.info(f"Method: {request.method}")
    logger.info(f"Target URL: {url}")
    logger.info(f"User ID: {headers.get('X-User-ID')}")
    logger.info(f"User Email: {headers.get('X-User-Email')}")
    logger.info(f"User Role: {headers.get('X-User-Role')}")
    logger.info(f"User Roles: {headers.get('X-User-Roles')}")
    logger.info(f"User Permissions: {headers.get('X-User-Permissions')}")
    logger.info(f"Service Prefix: {service_prefix}")
    logger.info(f"Strip Prefix: {strip_prefix}")
    logger.info(f"Headers: {headers}")
    logger.info(f"Query Params: {params}")

    # Log original token information
    auth_header = request.headers.get("Authorization", "")
    if auth_header and auth_header.startswith("Bearer "):
        token_prefix = auth_header[7:20] + "..." if len(auth_header) > 27 else auth_header[7:]
        logger.info(f"Original Token: {token_prefix}")

    # Log request body (truncated for privacy)
    if body:
        body_str = body.decode('utf-8', errors='replace')
        logger.info(f"Request body (truncated): {body_str[:100]}...")

    # Log all user-related headers for debugging
    logger.debug("All user headers being forwarded:")
    for header_name, header_value in headers.items():
        if header_name.startswith('X-User'):
            logger.debug(f"  {header_name}: {header_value}")
    logger.info(f"=== END API GATEWAY FORWARDING ===")

    # Forward the request
    async with httpx.AsyncClient(follow_redirects=True) as client:
        try:
            # Log that we're sending the request (details already logged above)
            logger.info(f"=== API GATEWAY SENDING REQUEST ===")
            logger.info(f"Sending {method} request to {url}")
            logger.info(f"=== END API GATEWAY SENDING ===")

            # Use a longer timeout for GraphQL requests
            timeout = 60.0 if path.startswith("/api/graphql") else 30.0
            logger.info(f"Using timeout of {timeout} seconds for {path}")

            response = await client.request(
                method=method,
                url=url,
                headers=headers,
                params=params,
                content=body,
                timeout=timeout
            )

            # Log the response details
            logger.info(f"=== API GATEWAY RECEIVED RESPONSE ===")
            logger.info(f"Status Code: {response.status_code}")
            logger.info(f"Headers: {response.headers}")
            logger.info(f"Content (truncated): {response.content[:100]}...")
            logger.info(f"=== END API GATEWAY RESPONSE ===")

            # Create FastAPI response
            return Response(
                content=response.content,
                status_code=response.status_code,
                headers=dict(response.headers),
                media_type=response.headers.get('content-type')
            )
        except httpx.RequestError as e:
            logger.error(f"Error forwarding request to {url}: {str(e)}")
            logger.error(f"Request details: method={method}, headers={headers}, params={params}")
            logger.error(f"Exception type: {type(e).__name__}")
            logger.error(f"Exception traceback: {e.__traceback__}")

            # Return a more detailed error response
            raise HTTPException(
                status_code=status.HTTP_502_BAD_GATEWAY,
                detail={
                    "error": "Error forwarding request",
                    "message": str(e),
                    "url": url,
                    "method": method
                }
            )
        except Exception as e:
            logger.error(f"Unexpected error forwarding request to {url}: {str(e)}")
            logger.error(f"Request details: method={method}, headers={headers}, params={params}")
            logger.error(f"Exception type: {type(e).__name__}")
            logger.error(f"Exception traceback: {e.__traceback__}")

            # Return a more detailed error response
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail={
                    "error": "Unexpected error forwarding request",
                    "message": str(e),
                    "url": url,
                    "method": method
                }
            )

async def check_permissions_with_auth_service(request: Request, path: str) -> tuple:
    """
    Check if the user has the required permissions for the requested path by asking the Auth Service.

    Args:
        request: The incoming request
        path: The path to check

    Returns:
        A tuple of (has_permission, status_code, detail)
        - has_permission: True if the user has the required permissions, False otherwise
        - status_code: HTTP status code to return if permission is denied
        - detail: Error message to return if permission is denied
    """
    # If no user is authenticated, deny access
    if not hasattr(request.state, 'user'):
        return False, status.HTTP_401_UNAUTHORIZED, "Authentication required"

    # Get the user permissions from the request state
    user_permissions = getattr(request.state, "user_permissions", [])
    user_roles = getattr(request.state, "user_roles", [])
    user_role = getattr(request.state, "user_role", "")

    # Admin users bypass permission checks
    if "admin" in user_roles or user_role == "admin" or "doctor" in user_roles or user_role == "doctor":
        logger.info(f"User has admin/doctor role, bypassing permission check")
        return True, status.HTTP_200_OK, ""

    # Define basic permission mappings
    if path.startswith("/api/fhir/Patient") or path.startswith("/api/patients") or path.startswith("/api/graphql"):
        if request.method == "GET" and "patient:read" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method in ["POST", "PUT"] and "patient:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "DELETE" and ("patient:delete" in user_permissions or "patient:write" in user_permissions):
            return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/fhir/Observation") or path.startswith("/api/observations"):
        if request.method == "GET" and "observation:read" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method in ["POST", "PUT"] and "observation:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "DELETE" and ("observation:delete" in user_permissions or "observation:write" in user_permissions):
            return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/fhir/Condition") or path.startswith("/api/conditions"):
        if request.method == "GET" and "condition:read" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method in ["POST", "PUT"] and "condition:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "DELETE" and ("condition:delete" in user_permissions or "condition:write" in user_permissions):
            return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/fhir/Medication") or path.startswith("/api/medications"):
        if request.method == "GET" and "medication:read" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method in ["POST", "PUT"] and "medication:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "DELETE" and ("medication:delete" in user_permissions or "medication:write" in user_permissions):
            return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/fhir/Encounter") or path.startswith("/api/encounters"):
        if request.method == "GET" and "encounter:read" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method in ["POST", "PUT"] and "encounter:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "DELETE" and ("encounter:delete" in user_permissions or "encounter:write" in user_permissions):
            return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/timeline"):
        if request.method == "GET" and "timeline:read" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method in ["POST", "PUT"] and "timeline:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "DELETE" and ("timeline:delete" in user_permissions or "timeline:write" in user_permissions):
            return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/v1/intake"):
        if "$approve" in path or "$escalate" in path or "$request-clarification" in path:
            if request.method == "POST" and "intake:review" in user_permissions:
                return True, status.HTTP_200_OK, ""
        elif "DetectedIssue" in path:
            if request.method == "GET" and "safety:read" in user_permissions:
                return True, status.HTTP_200_OK, ""
        elif "$enroll" in path or "$verify" in path or "$link-abha" in path:
            if request.method == "POST" and "intake:enroll" in user_permissions:
                return True, status.HTTP_200_OK, ""
        elif "$checkin" in path:
            if request.method == "POST" and "intake:checkin" in user_permissions:
                return True, status.HTTP_200_OK, ""
        else:
            if request.method == "GET" and "intake:read" in user_permissions:
                return True, status.HTTP_200_OK, ""
            elif request.method in ["POST", "PUT"] and "intake:write" in user_permissions:
                return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/v1/ingest"):
        if "$source-status" in path or "OperationOutcome" in path:
            if request.method == "GET" and "ingest:admin" in user_permissions:
                return True, status.HTTP_200_OK, ""
        elif "/labs" in path and request.method == "POST" and "ingest:lab" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif "/ehr" in path and request.method == "POST" and "ingest:ehr" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif "/abdm" in path and request.method == "POST" and "ingest:abdm" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif ("/devices" in path or "/wearables" in path) and request.method == "POST" and "ingest:device" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif request.method == "POST" and "ingest:write" in user_permissions:
            return True, status.HTTP_200_OK, ""

    # If no specific rule matches, deny access
    logger.warning(f"Permission denied for {request.method} {path} - no matching rule")
    return False, status.HTTP_403_FORBIDDEN, "Insufficient permissions"

# This function is no longer used, but kept for reference
# def check_permissions_locally(request: Request, path: str) -> bool:
#     """
#     Fallback method to check permissions locally if the Auth Service is unavailable.
#
#     Args:
#         request: The incoming request
#         path: The path to check
#
#     Returns:
#         True if the user has the required permissions, False otherwise
#     """
#     # Get the user permissions from the request state
#     user_permissions = getattr(request.state, "user_permissions", [])
#     user_roles = getattr(request.state, "user_roles", [])
#     user_role = getattr(request.state, "user_role", "")
#
#     # Admin users bypass permission checks
#     if "admin" in user_roles or user_role == "admin" or "doctor" in user_roles or user_role == "doctor":
#         logger.info(f"User has admin/doctor role, bypassing permission check")
#         return True
#
#     # Define basic permission mappings for fallback
#     if path.startswith("/api/fhir/Patient") or path.startswith("/api/patients") or path.startswith("/api/graphql"):
#         if request.method == "GET" and "patient:read" in user_permissions:
#             return True
#         elif request.method in ["POST", "PUT"] and "patient:write" in user_permissions:
#             return True
#         elif request.method == "DELETE" and ("patient:delete" in user_permissions or "patient:write" in user_permissions):
#             return True
#     elif path.startswith("/api/fhir/Observation") or path.startswith("/api/observations"):
#         if request.method == "GET" and "observation:read" in user_permissions:
#             return True
#         elif request.method in ["POST", "PUT"] and "observation:write" in user_permissions:
#             return True
#         elif request.method == "DELETE" and ("observation:delete" in user_permissions or "observation:write" in user_permissions):
#             return True
#     elif path.startswith("/api/fhir/Condition") or path.startswith("/api/conditions"):
#         if request.method == "GET" and "condition:read" in user_permissions:
#             return True
#         elif request.method in ["POST", "PUT"] and "condition:write" in user_permissions:
#             return True
#         elif request.method == "DELETE" and ("condition:delete" in user_permissions or "condition:write" in user_permissions):
#             return True
#     elif path.startswith("/api/fhir/Medication") or path.startswith("/api/medications"):
#         if request.method == "GET" and "medication:read" in user_permissions:
#             return True
#         elif request.method in ["POST", "PUT"] and "medication:write" in user_permissions:
#             return True
#         elif request.method == "DELETE" and ("medication:delete" in user_permissions or "medication:write" in user_permissions):
#             return True
#     elif path.startswith("/api/fhir/Encounter") or path.startswith("/api/encounters"):
#         if request.method == "GET" and "encounter:read" in user_permissions:
#             return True
#         elif request.method in ["POST", "PUT"] and "encounter:write" in user_permissions:
#             return True
#         elif request.method == "DELETE" and ("encounter:delete" in user_permissions or "encounter:write" in user_permissions):
#             return True
#     elif path.startswith("/api/timeline"):
#         if request.method == "GET" and "timeline:read" in user_permissions:
#             return True
#         elif request.method in ["POST", "PUT"] and "timeline:write" in user_permissions:
#             return True
#         elif request.method == "DELETE" and ("timeline:delete" in user_permissions or "timeline:write" in user_permissions):
#             return True
#
#     # If no specific rule matches, deny access
#     logger.warning(f"Permission denied for {request.method} {path} - no matching rule")
#     return False

@router.api_route("/{path:path}", methods=["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"])
async def proxy_endpoint(request: Request, path: str):
    """
    Main proxy endpoint that forwards requests to the appropriate service.

    Args:
        request: The incoming request
        path: The path to forward

    Returns:
        The response from the target service
    """
    # Get the full path including query parameters
    full_path = request.url.path

    # Log the incoming request
    logger.info(f"=== API GATEWAY RECEIVED REQUEST ===")
    logger.info(f"Method: {request.method}")
    logger.info(f"Path: {full_path}")
    logger.info(f"Headers: {request.headers}")
    logger.info(f"=== END API GATEWAY REQUEST ===")

    # Special handling for FHIR routes - route directly to microservices
    if full_path.startswith("/api/fhir"):
        # Determine which microservice to route to based on the FHIR resource type
        resource_type = full_path.split("/")[3] if len(full_path.split("/")) > 3 else ""

        if resource_type == "Patient":
            service_name = "patient"
            route_config = SERVICE_ROUTES["patient"]
            # Convert /api/fhir/Patient to /api/patients
            full_path = full_path.replace("/api/fhir/Patient", "/api/patients")
        elif resource_type == "Observation":
            service_name = "observation"
            route_config = SERVICE_ROUTES["observation"]
            # Convert /api/fhir/Observation to /api/observations
            full_path = full_path.replace("/api/fhir/Observation", "/api/observations")
        elif resource_type == "Condition":
            service_name = "condition"
            route_config = SERVICE_ROUTES["condition"]
            # Convert /api/fhir/Condition to /api/conditions
            full_path = full_path.replace("/api/fhir/Condition", "/api/conditions")
        elif resource_type == "Medication":
            service_name = "medication"
            route_config = SERVICE_ROUTES["medication"]
            # Convert /api/fhir/Medication to /api/medications
            full_path = full_path.replace("/api/fhir/Medication", "/api/medications")
        elif resource_type == "Encounter":
            service_name = "encounter"
            route_config = SERVICE_ROUTES["encounter"]
            # Convert /api/fhir/Encounter to /api/encounters
            full_path = full_path.replace("/api/fhir/Encounter", "/api/encounters")
        else:
            # Default to FHIR service for unknown resource types
            service_name = "fhir"
            route_config = SERVICE_ROUTES["fhir"]

        prefix = route_config["prefix"]

        # Add very visible logging
        print(f"\n\n==== API GATEWAY ROUTING FHIR REQUEST DIRECTLY TO MICROSERVICE ====")
        print(f"Path: {full_path}")
        print(f"Method: {request.method}")
        print(f"Resource Type: {resource_type}")
        print(f"Target Service: {service_name}")
        print(f"Target URL: {route_config['target']}")
        print(f"==== END API GATEWAY ROUTING ====\n\n")
        logger.info(f"FHIR path detected: {request.url.path} - routing directly to {service_name} service as {full_path}")

    # Special handling for timeline routes - route directly to timeline service
    elif full_path.startswith("/api/timeline/patient/"):
        service_name = "timeline"
        route_config = SERVICE_ROUTES["timeline"]
        prefix = route_config["prefix"]

        # Add very visible logging
        print(f"\n\n==== API GATEWAY ROUTING TIMELINE REQUEST ====")
        print(f"Path: {full_path}")
        print(f"Method: {request.method}")
        print(f"Target: {route_config['target']}")
        print(f"==== END API GATEWAY ROUTING ====\n\n")
        logger.info(f"Timeline path detected: {full_path} - routing to timeline service")
    else:
        # For non-FHIR routes, find the matching service
        matched = False
        for service_name, route_config in SERVICE_ROUTES.items():
            prefix = route_config["prefix"]
            if full_path.startswith(prefix):
                matched = True
                break

        if not matched:
            # If no service route matches, return 404
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"No service route found for path: {full_path}"
            )

    # Process the matched route
    # Log the matched service route
    logger.info(f"Matched service route: {service_name} with prefix {prefix}")

    # Check if this is a public path that doesn't require authentication
    is_public = any(full_path.endswith(public_path) for public_path in route_config["public_paths"])

    # Log if this is a public path
    if is_public:
        logger.info(f"Path {full_path} is public, skipping authentication")

    # Check if the user is authenticated
    if not is_public and not hasattr(request.state, 'user'):
        # Get the authorization header
        auth_header = request.headers.get("Authorization")
        if not auth_header or not auth_header.startswith("Bearer "):
            logger.warning(f"Unauthenticated request to non-public path: {full_path}")
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Authentication required",
                headers={"WWW-Authenticate": "Bearer"}
            )

        # Extract the token
        token = auth_header.replace("Bearer ", "")

        # Call the Auth Service to validate the token
        try:
            async with httpx.AsyncClient() as client:
                auth_response = await client.post(
                    f"{settings.AUTH_SERVICE_URL}/api/auth/verify",
                    json={"token": token}
                )

                if auth_response.status_code == 200:
                    # Token is valid, get the user information
                    user_info = auth_response.json()

                    # Set the user information in the request state
                    request.state.user = user_info
                    request.state.user_role = user_info.get("role", "authenticated")
                    request.state.user_roles = user_info.get("roles", [])
                    request.state.user_permissions = user_info.get("permissions", [])

                    logger.info(f"Authenticated user: {user_info.get('id')} with role: {request.state.user_role}")
                else:
                    # Token is invalid
                    logger.warning(f"Invalid token for path: {full_path}")
                    raise HTTPException(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        detail="Invalid token",
                        headers={"WWW-Authenticate": "Bearer"}
                    )
        except Exception as e:
            logger.error(f"Error validating token: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Error validating token: {str(e)}"
            )

    # Check if the user has the required permissions
    if not is_public:
        has_permission, status_code, detail = await check_permissions_with_auth_service(request, full_path)
        if not has_permission:
            logger.warning(f"Permission denied for path: {full_path} - {detail}")
            raise HTTPException(
                status_code=status_code,
                detail=detail
            )

    # Log the service we're routing to
    logger.info(f"=== API GATEWAY ROUTING TO SERVICE ===")
    logger.info(f"Service: {service_name}")
    logger.info(f"Target URL: {route_config['target']}")
    logger.info(f"Path: {full_path}")
    logger.info(f"Strip Prefix: {route_config['strip_prefix']}")
    logger.info(f"Service Prefix: {prefix}")
    logger.info(f"=== END API GATEWAY ROUTING ===")

    try:
        # Forward the request to the target service
        return await forward_request(
            request=request,
            target_url=route_config["target"],
            path=full_path,
            strip_prefix=route_config["strip_prefix"],
            service_prefix=prefix
        )
    except Exception as e:
        # Log the error
        logger.error(f"=== API GATEWAY ERROR ===")
        logger.error(f"Error forwarding request to {route_config['target']}: {str(e)}")
        logger.error(f"Service: {service_name}")
        logger.error(f"Path: {full_path}")
        logger.error(f"=== END API GATEWAY ERROR ===")

        # Re-raise the exception
        raise
