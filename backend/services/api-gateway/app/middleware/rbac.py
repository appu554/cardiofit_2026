from fastapi import Request, Response, HTTPException, status
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Callable, Dict, List, Optional, Any
import logging
import re
import os

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Define route permission mappings
ROUTE_PERMISSIONS = {
    # GraphQL permissions
    r"^/api/graphql": {
        "GET": ["patient:read"],
        "POST": ["patient:read", "patient:write"]
    },
    # Patient service permissions
    r"^/api/patients": {
        "GET": ["patient:read"],
        "POST": ["patient:write"],
        "PUT": ["patient:write"],
        "DELETE": ["patient:delete"]
    },
    # FHIR service permissions
    r"^/api/fhir/Patient": {
        "GET": ["patient:read"],
        "POST": ["patient:write"],
        "PUT": ["patient:write"],
        "DELETE": ["patient:delete"]
    },
    # Observation service permissions
    r"^/api/observations": {
        "GET": ["observation:read"],
        "POST": ["observation:write"],
        "PUT": ["observation:write"],
        "DELETE": ["observation:delete"]
    },
    r"^/api/fhir/Observation": {
        "GET": ["observation:read"],
        "POST": ["observation:write"],
        "PUT": ["observation:write"],
        "DELETE": ["observation:delete"]
    },
    # Condition service permissions
    r"^/api/conditions": {
        "GET": ["condition:read"],
        "POST": ["condition:write"],
        "PUT": ["condition:write"],
        "DELETE": ["condition:delete"]
    },
    r"^/api/fhir/Condition": {
        "GET": ["condition:read"],
        "POST": ["condition:write"],
        "PUT": ["condition:write"],
        "DELETE": ["condition:delete"]
    },
    # Medication service permissions
    r"^/api/medications": {
        "GET": ["medication:read"],
        "POST": ["medication:write"],
        "PUT": ["medication:write"],
        "DELETE": ["medication:delete"]
    },
    r"^/api/fhir/Medication": {
        "GET": ["medication:read"],
        "POST": ["medication:write"],
        "PUT": ["medication:write"],
        "DELETE": ["medication:delete"]
    },
    # Encounter service permissions
    r"^/api/encounters": {
        "GET": ["encounter:read"],
        "POST": ["encounter:write"],
        "PUT": ["encounter:write"],
        "DELETE": ["encounter:delete"]
    },
    r"^/api/fhir/Encounter": {
        "GET": ["encounter:read"],
        "POST": ["encounter:write"],
        "PUT": ["encounter:write"],
        "DELETE": ["encounter:delete"]
    },
    # Admin routes
    r"^/api/admin": {
        "GET": ["admin:access"],
        "POST": ["admin:access"],
        "PUT": ["admin:access"],
        "DELETE": ["admin:access"]
    },
    # === Intake-Onboarding Service ===
    r"^/api/v1/intake/fhir/Patient/\$enroll": {
        "POST": ["intake:enroll"]
    },
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$verify": {
        "POST": ["intake:enroll"]
    },
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$checkin": {
        "POST": ["intake:checkin"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$fill-slot": {
        "POST": ["intake:write"]
    },
    r"^/api/v1/intake/fhir/Observation": {
        "POST": ["intake:write"],
        "GET": ["intake:read"]
    },
    r"^/api/v1/intake/fhir/Patient": {
        "GET": ["patient:read"],
        "POST": ["patient:write"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$approve": {
        "POST": ["intake:review"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$escalate": {
        "POST": ["intake:review"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$request-clarification": {
        "POST": ["intake:review"]
    },
    r"^/api/v1/intake/fhir/DetectedIssue": {
        "GET": ["safety:read"]
    },
    # === Ingestion Service ===
    r"^/api/v1/ingest/fhir/Observation": {
        "POST": ["ingest:write"]
    },
    r"^/api/v1/ingest/devices": {
        "POST": ["ingest:device"]
    },
    r"^/api/v1/ingest/wearables": {
        "POST": ["ingest:device"]
    },
    r"^/api/v1/ingest/fhir/OperationOutcome": {
        "GET": ["ingest:admin"]
    },
    r"^/api/v1/ingest/\$source-status": {
        "GET": ["ingest:admin"]
    },
    r"^/api/v1/ingest/labs": {
        "POST": ["ingest:lab"]
    },
    r"^/api/v1/ingest/ehr": {
        "POST": ["ingest:ehr"]
    },
    r"^/api/v1/ingest/abdm": {
        "POST": ["ingest:abdm"]
    }
}

# Define role-based route restrictions
ROLE_ROUTE_RESTRICTIONS = {
    # Routes that only doctors can access
    r"^/api/patients/\d+/medical-history": ["doctor"],

    # Routes that only admins can access
    r"^/api/admin": ["admin"],

    # Routes that require specific roles
    r"^/api/prescriptions": ["doctor", "pharmacist"],
    # === Intake-Onboarding ===
    r"^/api/v1/intake/fhir/Patient/\$enroll":
        ["patient", "pharmacist", "physician", "asha"],
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$fill-slot":
        ["patient", "pharmacist", "physician", "asha"],
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$checkin":
        ["patient"],
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$(approve|escalate|request-clarification)":
        ["pharmacist", "physician"],
    r"^/api/v1/intake/fhir/DetectedIssue":
        ["pharmacist", "physician"],
    r"^/api/v1/intake/fhir/Encounter":
        ["pharmacist", "physician"],
    # === Ingestion ===
    r"^/api/v1/ingest/fhir/Observation":
        ["patient", "asha", "physician"],
    r"^/api/v1/ingest/devices":
        ["patient"],
    r"^/api/v1/ingest/wearables":
        ["patient"],
    r"^/api/v1/ingest/\$source-status":
        ["admin", "pharmacist", "physician"],
    r"^/api/v1/ingest/fhir/OperationOutcome":
        ["admin", "physician"],
    r"^/api/v1/ingest/labs":
        ["system", "physician"],
    r"^/api/v1/ingest/ehr":
        ["system", "physician"],
    r"^/api/v1/ingest/abdm":
        ["system"]
}

class RBACMiddleware(BaseHTTPMiddleware):
    """
    Middleware for enforcing Role-Based Access Control (RBAC).

    This middleware checks if the authenticated user has the required permissions
    for the requested route based on predefined permission mappings.
    """

    def __init__(
        self,
        app,
        exclude_paths: Optional[list] = None
    ):
        super().__init__(app)
        self.exclude_paths = exclude_paths or ["/docs", "/openapi.json", "/redoc", "/health", "/graphql"]
        logger.info(f"Initialized RBACMiddleware with excluded paths: {self.exclude_paths}")

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process the request, check permissions, and pass it to the next middleware.

        Args:
            request: The incoming request
            call_next: The next middleware to call

        Returns:
            The response from the next middleware
        """
        # Skip RBAC for excluded paths
        path = request.url.path
        if any(path.startswith(excluded) for excluded in self.exclude_paths):
            return await call_next(request)

        # Skip RBAC for OPTIONS requests (CORS preflight)
        if request.method == "OPTIONS":
            return await call_next(request)

        # Skip RBAC if user is not authenticated
        if not hasattr(request.state, 'user'):
            # Authentication middleware should have handled this already
            return await call_next(request)

        # Get user permissions and roles
        user_permissions = getattr(request.state, 'user_permissions', [])
        user_role = getattr(request.state, 'user_role', None)
        user_roles = getattr(request.state, 'user_roles', [])

        # Add the primary role to the roles list if it's not already there
        if user_role and user_role not in user_roles:
            user_roles.append(user_role)

        # Check role-based route restrictions
        for route_pattern, allowed_roles in ROLE_ROUTE_RESTRICTIONS.items():
            if re.match(route_pattern, path):
                # Check if user has any of the allowed roles
                if not any(role in allowed_roles for role in user_roles):
                    logger.warning(f"Role authorization failed for {path}. User roles: {user_roles}, Required roles: {allowed_roles}")
                    return Response(
                        status_code=status.HTTP_403_FORBIDDEN,
                        content=f"Insufficient role permissions. Required roles: {', '.join(allowed_roles)}",
                        media_type="text/plain"
                    )

        # Check permission-based route restrictions
        for route_pattern, method_permissions in ROUTE_PERMISSIONS.items():
            if re.match(route_pattern, path) and request.method in method_permissions:
                required_permissions = method_permissions[request.method]

                # Check if user has all required permissions
                if not all(perm in user_permissions for perm in required_permissions):
                    logger.warning(f"Permission authorization failed for {path}. User permissions: {user_permissions}, Required permissions: {required_permissions}")
                    return Response(
                        status_code=status.HTTP_403_FORBIDDEN,
                        content=f"Insufficient permissions. Required: {', '.join(required_permissions)}",
                        media_type="text/plain"
                    )

        # If all checks pass, continue with the request
        return await call_next(request)
