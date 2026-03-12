from fastapi import Request, Response, status
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Callable, Dict, List, Optional, Any
import logging
import json

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Define GraphQL operation permission mappings
GRAPHQL_PERMISSIONS = {
    # Patient operations
    "patient": {
        "Query": ["patient:read"],
        "Mutation": ["patient:write"]
    },
    "searchPatients": {
        "Query": ["patient:read"]
    },
    "createPatient": {
        "Mutation": ["patient:write"]
    },
    "updatePatient": {
        "Mutation": ["patient:write"]
    },
    "deletePatient": {
        "Mutation": ["patient:delete"]
    },
    
    # Observation operations
    "observations": {
        "Query": ["observation:read"]
    },
    "patientObservations": {
        "Query": ["observation:read"]
    },
    "createObservation": {
        "Mutation": ["observation:write"]
    },
    "updateObservation": {
        "Mutation": ["observation:write"]
    },
    "deleteObservation": {
        "Mutation": ["observation:delete"]
    },
    
    # Condition operations
    "conditions": {
        "Query": ["condition:read"]
    },
    "patientConditions": {
        "Query": ["condition:read"]
    },
    "createCondition": {
        "Mutation": ["condition:write"]
    },
    "updateCondition": {
        "Mutation": ["condition:write"]
    },
    "deleteCondition": {
        "Mutation": ["condition:delete"]
    },
    
    # Medication operations
    "medications": {
        "Query": ["medication:read"]
    },
    "patientMedications": {
        "Query": ["medication:read"]
    },
    "createMedication": {
        "Mutation": ["medication:write"]
    },
    "updateMedication": {
        "Mutation": ["medication:write"]
    },
    "deleteMedication": {
        "Mutation": ["medication:delete"]
    },
    
    # Encounter operations
    "encounters": {
        "Query": ["encounter:read"]
    },
    "patientEncounters": {
        "Query": ["encounter:read"]
    },
    "createEncounter": {
        "Mutation": ["encounter:write"]
    },
    "updateEncounter": {
        "Mutation": ["encounter:write"]
    },
    "deleteEncounter": {
        "Mutation": ["encounter:delete"]
    }
}

class GraphQLRBACMiddleware(BaseHTTPMiddleware):
    """
    Middleware for enforcing Role-Based Access Control (RBAC) for GraphQL operations.
    
    This middleware checks if the authenticated user has the required permissions
    for the requested GraphQL operations based on predefined permission mappings.
    """
    
    def __init__(self, app):
        super().__init__(app)
        logger.info("Initialized GraphQLRBACMiddleware")
    
    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process the request, check permissions for GraphQL operations, and pass it to the next middleware.
        
        Args:
            request: The incoming request
            call_next: The next middleware to call
            
        Returns:
            The response from the next middleware
        """
        # Only process GraphQL POST requests
        if request.url.path.endswith("/graphql") and request.method == "POST":
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
            
            # Admin role bypasses permission checks
            if "admin" in user_roles or user_role == "admin":
                logger.info("Admin role detected, bypassing RBAC checks for GraphQL")
                return await call_next(request)
            
            # Parse GraphQL request body
            try:
                body = await request.body()
                body_dict = json.loads(body)
                query = body_dict.get("query", "")
                operation_name = body_dict.get("operationName")
                
                # Determine if this is a query or mutation
                operation_type = "Query" if query.strip().startswith("query") else "Mutation"
                
                logger.info(f"GraphQL operation: {operation_name} ({operation_type})")
                
                # Check permissions based on operation name and type
                if operation_name and operation_name in GRAPHQL_PERMISSIONS:
                    required_permissions = GRAPHQL_PERMISSIONS[operation_name].get(operation_type, [])
                    
                    logger.info(f"Required permissions for {operation_name}: {required_permissions}")
                    logger.info(f"User permissions: {user_permissions}")
                    
                    # Check if user has all required permissions
                    if required_permissions and not all(perm in user_permissions for perm in required_permissions):
                        logger.warning(f"Permission authorization failed for GraphQL operation: {operation_name}. User permissions: {user_permissions}, Required permissions: {required_permissions}")
                        return Response(
                            status_code=status.HTTP_403_FORBIDDEN,
                            content=json.dumps({
                                "errors": [{
                                    "message": f"Insufficient permissions for GraphQL operation: {operation_name}. Required: {', '.join(required_permissions)}",
                                    "extensions": {
                                        "code": "FORBIDDEN",
                                        "required_permissions": required_permissions
                                    }
                                }]
                            }),
                            media_type="application/json"
                        )
                    
                    logger.info(f"Permission check passed for GraphQL operation: {operation_name}")
            except Exception as e:
                logger.error(f"Error processing GraphQL request for RBAC: {str(e)}")
                # Continue with the request if there's an error parsing
        
        # Continue with the request
        return await call_next(request)
