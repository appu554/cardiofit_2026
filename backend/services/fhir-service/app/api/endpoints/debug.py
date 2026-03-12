"""
Debug endpoints for the FHIR service.
"""
from fastapi import APIRouter, Request, Depends, Body, HTTPException
from typing import Dict, Any, Optional
from app.core.auth import get_token_payload
import httpx
import logging
from app.core.config import settings
from app.core.integration import FHIRIntegrationLayer

# Configure logging
logger = logging.getLogger(__name__)

router = APIRouter()
fhir_integration = FHIRIntegrationLayer()

@router.get("/debug/auth")
async def debug_auth(request: Request, token_payload: Dict[str, Any] = Depends(get_token_payload)):
    """
    Debug endpoint to check authentication and permissions.
    """
    # Get all attributes from the request state
    state_attrs = {}
    for attr in dir(request.state):
        if not attr.startswith('_'):
            try:
                value = getattr(request.state, attr)
                if not callable(value):
                    state_attrs[attr] = value
            except Exception as e:
                state_attrs[attr] = f"Error getting attribute: {str(e)}"

    # Return debug information
    return {
        "token_payload": token_payload,
        "request_state": state_attrs,
        "headers": dict(request.headers),
        "user_permissions": getattr(request.state, "user_permissions", None),
        "user_roles": getattr(request.state, "user_roles", None),
        "user_role": getattr(request.state, "user_role", None),
        "user": getattr(request.state, "user", None),
    }

@router.post("/debug/route-test")
async def debug_route_test(
    request: Request,
    resource_type: str = Body(..., description="FHIR resource type"),
    service_url: str = Body(..., description="Service URL to test"),
    endpoint: str = Body(..., description="Endpoint to test"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Debug endpoint to test routing to microservices.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Log the request details
        logger.info(f"=== DEBUG ROUTE TEST ===")
        logger.info(f"Resource Type: {resource_type}")
        logger.info(f"Service URL: {service_url}")
        logger.info(f"Endpoint: {endpoint}")
        logger.info(f"Auth Header: {auth_header[:20]}...")

        # Make the request to the microservice
        async with httpx.AsyncClient(follow_redirects=True) as client:
            # Forward all headers including RBAC headers
            headers = {
                "Authorization": auth_header,
                "Content-Type": "application/json",
                "X-User-ID": "test-user-id",
                "X-User-Email": "doctor@example.com",
                "X-User-Role": "doctor",
                "X-User-Roles": "doctor",
                "X-User-Permissions": "patient:read,patient:write,observation:read,observation:write,condition:read,condition:write,medication:read,medication:write,encounter:read,encounter:write"
            }

            # Make a GET request to the endpoint
            full_url = f"{service_url}{endpoint}"
            logger.info(f"Making GET request to: {full_url}")

            response = await client.get(
                full_url,
                headers=headers,
                timeout=30.0
            )

            logger.info(f"Response status: {response.status_code}")
            logger.info(f"Response headers: {response.headers}")
            logger.info(f"Response content: {response.content}")
            logger.info(f"=== END DEBUG ROUTE TEST ===")

            # Return the response details
            return {
                "request": {
                    "url": full_url,
                    "method": "GET",
                    "headers": headers
                },
                "response": {
                    "status_code": response.status_code,
                    "headers": dict(response.headers),
                    "content": response.content.decode('utf-8', errors='replace')
                }
            }
    except Exception as e:
        logger.error(f"Error in debug route test: {str(e)}")
        return {
            "error": str(e),
            "request": {
                "resource_type": resource_type,
                "service_url": service_url,
                "endpoint": endpoint
            }
        }

@router.get("/debug/service-registry")
async def debug_service_registry(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Debug endpoint to check the FHIR service registry.
    """
    try:
        # Get the FHIR integration layer
        from app.core.integration import fhir_integration

        # Get the service registry
        registry = fhir_integration.service_registry

        # Return the registry
        return {
            "service_registry": registry,
            "medication_service_url": registry.get("MedicationRequest", "Not found"),
            "medication_service_url_from_settings": settings.MEDICATION_SERVICE_URL
        }
    except Exception as e:
        logger.error(f"Error getting service registry: {str(e)}")
        return {
            "error": str(e)
        }

@router.get("/debug/mongodb")
async def debug_mongodb(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    subject: Optional[str] = None,
    show_all: Optional[bool] = False
):
    """
    Debug endpoint to check MongoDB connection and collection status.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Log the request details
        logger.info(f"=== DEBUG MONGODB ===")

        # Forward the request to the Medication service
        medication_url = settings.MEDICATION_SERVICE_URL
        debug_endpoint = "/api/fhir/debug/mongodb"

        # Add query parameters if provided
        query_params = []
        if subject:
            query_params.append(f"subject={subject}")
        if show_all:
            query_params.append(f"show_all={str(show_all).lower()}")

        if query_params:
            debug_endpoint += f"?{'&'.join(query_params)}"

        logger.info(f"Forwarding request to: {medication_url}{debug_endpoint}")

        # Make the request to the Medication service
        async with httpx.AsyncClient(follow_redirects=True) as client:
            # Forward all headers including RBAC headers
            headers = {
                "Authorization": auth_header,
                "Content-Type": "application/json",
                "X-User-ID": "test-user-id",
                "X-User-Email": "doctor@example.com",
                "X-User-Role": "doctor",
                "X-User-Roles": "doctor",
                "X-User-Permissions": "patient:read,patient:write,observation:read,observation:write,condition:read,condition:write,medication:read,medication:write,encounter:read,encounter:write,timeline:read"
            }

            try:
                response = await client.get(
                    f"{medication_url}{debug_endpoint}",
                    headers=headers,
                    timeout=10.0
                )

                logger.info(f"Response status: {response.status_code}")
                logger.info(f"Response content: {response.content}")

                # Return the response from the Medication service
                if response.status_code == 200:
                    return response.json()
                else:
                    return {
                        "error": f"Medication service returned status code {response.status_code}",
                        "content": response.content.decode('utf-8', errors='replace')
                    }
            except Exception as e:
                logger.error(f"Error forwarding request to Medication service: {str(e)}")
                return {
                    "error": f"Error forwarding request to Medication service: {str(e)}",
                    "medication_service_url": medication_url,
                    "debug_endpoint": debug_endpoint
                }
    except Exception as e:
        logger.error(f"Error in debug mongodb: {str(e)}")
        return {
            "error": str(e)
        }

@router.get("/debug/medication-requests")
async def debug_medication_requests(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Debug endpoint to check the Medication service status.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Log the request details
        logger.info(f"=== DEBUG MEDICATION REQUESTS ===")

        # Forward the request to the Medication service
        medication_url = settings.MEDICATION_SERVICE_URL
        debug_endpoint = "/api/fhir/debug/medication-requests"

        logger.info(f"Forwarding request to: {medication_url}{debug_endpoint}")

        # Make the request to the Medication service
        async with httpx.AsyncClient(follow_redirects=True) as client:
            # Forward all headers including RBAC headers
            headers = {
                "Authorization": auth_header,
                "Content-Type": "application/json",
                "X-User-ID": "test-user-id",
                "X-User-Email": "doctor@example.com",
                "X-User-Role": "doctor",
                "X-User-Roles": "doctor",
                "X-User-Permissions": "patient:read,patient:write,observation:read,observation:write,condition:read,condition:write,medication:read,medication:write,encounter:read,encounter:write,timeline:read"
            }

            try:
                response = await client.get(
                    f"{medication_url}{debug_endpoint}",
                    headers=headers,
                    timeout=10.0
                )

                logger.info(f"Response status: {response.status_code}")
                logger.info(f"Response content: {response.content}")

                # Return the response from the Medication service
                if response.status_code == 200:
                    return response.json()
                else:
                    return {
                        "error": f"Medication service returned status code {response.status_code}",
                        "content": response.content.decode('utf-8', errors='replace')
                    }
            except Exception as e:
                logger.error(f"Error forwarding request to Medication service: {str(e)}")
                return {
                    "error": f"Error forwarding request to Medication service: {str(e)}",
                    "medication_service_url": medication_url,
                    "debug_endpoint": debug_endpoint
                }
    except Exception as e:
        logger.error(f"Error in debug medication requests: {str(e)}")
        return {
            "error": str(e)
        }

@router.post("/debug/test-condition-service")
async def debug_test_condition_service(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """
    Debug endpoint to test routing to the Condition service.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Log the request details
        logger.info(f"=== DEBUG TEST CONDITION SERVICE ===")

        # Test direct connection to Condition service
        condition_url = settings.CONDITION_SERVICE_URL
        logger.info(f"Testing direct connection to Condition service at: {condition_url}")

        # Make a request to the health endpoint
        async with httpx.AsyncClient(follow_redirects=True) as client:
            health_url = f"{condition_url}/health"
            logger.info(f"Making GET request to: {health_url}")

            try:
                health_response = await client.get(
                    health_url,
                    timeout=5.0
                )

                logger.info(f"Health response status: {health_response.status_code}")
                logger.info(f"Health response content: {health_response.content}")
            except Exception as health_e:
                logger.error(f"Error checking Condition service health: {str(health_e)}")

        # Test routing through FHIR integration layer
        logger.info(f"Testing routing through FHIR integration layer")

        # Create a test condition
        test_condition = {
            "resourceType": "Condition",
            "subject": {
                "reference": "Patient/test-patient"
            },
            "code": {
                "coding": [
                    {
                        "system": "http://snomed.info/sct",
                        "code": "73211009",
                        "display": "Diabetes mellitus"
                    }
                ],
                "text": "Diabetes"
            },
            "clinicalStatus": {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
                        "code": "active",
                        "display": "Active"
                    }
                ]
            },
            "verificationStatus": {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
                        "code": "confirmed",
                        "display": "Confirmed"
                    }
                ]
            },
            "onsetDateTime": "2023-01-01"
        }

        try:
            # Use the integration layer to route to the Condition service
            result = await fhir_integration.create_resource("Condition", test_condition, auth_header)
            logger.info(f"Integration layer result: {result}")
        except Exception as integration_e:
            logger.error(f"Error using integration layer: {str(integration_e)}")

        logger.info(f"=== END DEBUG TEST CONDITION SERVICE ===")

        # Return the test results
        return {
            "condition_service_url": condition_url,
            "direct_connection_test": {
                "url": health_url,
                "success": True if 'health_response' in locals() and health_response.status_code == 200 else False,
                "status_code": health_response.status_code if 'health_response' in locals() else None,
                "content": health_response.content.decode('utf-8', errors='replace') if 'health_response' in locals() else None
            },
            "integration_layer_test": {
                "success": True if 'result' in locals() else False,
                "result": result if 'result' in locals() else None,
                "error": str(integration_e) if 'integration_e' in locals() else None
            }
        }
    except Exception as e:
        logger.error(f"Error in debug test condition service: {str(e)}")
        return {
            "error": str(e)
        }
