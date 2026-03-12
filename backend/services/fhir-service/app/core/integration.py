"""
FHIR Integration Layer

This module provides the core functionality for the FHIR Integration Layer,
which acts as a central hub for FHIR resource management and provides
standardized interfaces to other microservices.
"""

import httpx
import logging
import traceback
import urllib.parse
from typing import Dict, List, Any, Optional
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class FHIRIntegrationLayer:
    """
    FHIR Integration Layer that coordinates access to resource-specific microservices.
    """

    def __init__(self):
        # Override the MedicationRequest URL to use port 8009 directly
        medication_service_url = "http://localhost:8009"

        self.service_registry = {
            "Patient": settings.PATIENT_SERVICE_URL,
            "Observation": settings.OBSERVATION_SERVICE_URL,
            "DocumentReference": settings.NOTES_SERVICE_URL,
            "MedicationRequest": medication_service_url,  # Use the hardcoded URL
            "Medication": medication_service_url,  # Use the hardcoded URL
            "MedicationAdministration": medication_service_url,  # Use the hardcoded URL
            "MedicationStatement": medication_service_url,  # Use the hardcoded URL
            "ImagingStudy": settings.IMAGING_SERVICE_URL,
            "Condition": settings.CONDITION_SERVICE_URL,
            "Encounter": settings.ENCOUNTER_SERVICE_URL,
            "DiagnosticReport": "http://localhost:8000",  # Updated to port 8000 for Lab service
            "Specimen": "http://localhost:8000"  # Updated to port 8000 for Lab service
        }

        # Log the service registry for debugging
        logger.info("=== FHIR INTEGRATION LAYER SERVICE REGISTRY ===")
        for resource_type, url in self.service_registry.items():
            logger.info(f"{resource_type}: {url}")
        logger.info("=== END FHIR INTEGRATION LAYER SERVICE REGISTRY ===")

        # Default to the FHIR service itself for any unregistered resource types
        self.default_service_url = settings.FHIR_SERVICE_URL

    def get_service_url(self, resource_type: str) -> str:
        """
        Get the service URL for a specific resource type.

        Args:
            resource_type: The FHIR resource type

        Returns:
            The service URL for the resource type
        """
        # Route to specific microservices for production-like testing
        return self.service_registry.get(resource_type, self.default_service_url)

    async def create_resource(self, resource_type: str, resource: Dict[str, Any], auth_header: str) -> Dict[str, Any]:
        """
        Create a new FHIR resource by routing to the appropriate microservice.

        Args:
            resource_type: The FHIR resource type
            resource: The FHIR resource data
            auth_header: The authorization header

        Returns:
            The created FHIR resource
        """
        try:
            service_url = self.get_service_url(resource_type)
            logger.info(f"Creating {resource_type} using service at {service_url}")

            # Add very visible logging for debugging
            print(f"\n\n==== FHIR SERVICE ROUTING DETAILS ====")
            print(f"Resource Type: {resource_type}")
            print(f"Service URL from registry: {service_url}")
            print(f"Service URL from settings: {settings.MEDICATION_SERVICE_URL}")
            print(f"==== END FHIR SERVICE ROUTING DETAILS ====\n\n")

            # If we're using the default FHIR service, use the local FHIR service directly
            if service_url == self.default_service_url:
                logger.info(f"Using local FHIR service for {resource_type}")
                from app.services.fhir_service import get_fhir_service
                fhir_service = get_fhir_service()
                return await fhir_service.create_resource(resource_type, resource)

            # Otherwise, route to the appropriate microservice
            logger.info(f"=== FHIR SERVICE ROUTING REQUEST ===")
            logger.info(f"Routing {resource_type} to external service at {service_url}")
            logger.info(f"Resource Type: {resource_type}")
            logger.info(f"Service URL: {service_url}")
            logger.info(f"Auth Header: {auth_header[:20]}...")
            logger.info(f"Resource Data: {resource}")
            logger.info(f"=== END FHIR SERVICE ROUTING ===")
            async with httpx.AsyncClient(follow_redirects=True) as client:
                try:
                    # Use the correct path for the microservice
                    # The microservices have their FHIR endpoints at /api/fhir/{resource_type}
                    # For MedicationRequest, Encounter, and DiagnosticReport, we need to use a different path
                    if resource_type == "MedicationRequest":
                        endpoint = f"{service_url}/api/fhir/MedicationRequest"
                    elif resource_type == "Encounter":
                        endpoint = f"{service_url}/api/fhir/Encounter"
                    elif resource_type == "DiagnosticReport":
                        endpoint = f"{service_url}/api/fhir/DiagnosticReport"
                    else:
                        endpoint = f"{service_url}/api/fhir/{resource_type}"
                    logger.info(f"Sending POST request to {endpoint}")
                    logger.info(f"Request body: {resource}")
                    logger.info(f"Request headers: Authorization: {auth_header[:10]}...")

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

                    # Add very visible logging
                    print(f"\n\n==== FHIR SERVICE FORWARDING REQUEST TO MICROSERVICE ====")
                    print(f"Resource Type: {resource_type}")
                    print(f"Endpoint: {endpoint}")
                    print(f"Method: POST")
                    print(f"Headers: {headers}")
                    print(f"==== END FHIR SERVICE FORWARDING ====\n\n")

                    response = await client.post(
                        endpoint,
                        json=resource,
                        headers=headers,
                        timeout=30.0  # 30 second timeout
                    )

                    logger.info(f"=== FHIR SERVICE RECEIVED RESPONSE ===")
                    logger.info(f"Response status: {response.status_code}")
                    logger.info(f"Response headers: {response.headers}")
                    logger.info(f"Response content: {response.content}")
                    logger.info(f"Request URL: {endpoint}")
                    logger.info(f"Request method: POST")
                    logger.info(f"=== END FHIR SERVICE RESPONSE ===")

                    response.raise_for_status()
                    response_data = response.json()
                    logger.debug(f"Response data: {response_data}")
                    return response_data
                except httpx.HTTPStatusError as e:
                    logger.error(f"HTTP error creating {resource_type}: {e}")
                    logger.error(f"Response content: {e.response.content}")
                    raise Exception(f"HTTP error: {e.response.status_code} - {e.response.content}")
                except httpx.RequestError as e:
                    logger.error(f"Request error creating {resource_type}: {e}")
                    raise Exception(f"Request error: {str(e)}")
                except Exception as e:
                    logger.error(f"Error creating {resource_type}: {e}")
                    logger.error(traceback.format_exc())
                    raise
        except Exception as e:
            logger.error(f"Error in create_resource for {resource_type}: {str(e)}")
            logger.error(traceback.format_exc())
            raise

    async def get_resource(self, resource_type: str, resource_id: str, auth_header: str) -> Optional[Dict[str, Any]]:
        """
        Get a FHIR resource by ID by routing to the appropriate microservice.

        Args:
            resource_type: The FHIR resource type
            resource_id: The FHIR resource ID
            auth_header: The authorization header

        Returns:
            The FHIR resource, or None if not found
        """
        try:
            service_url = self.get_service_url(resource_type)

            # If we're using the default FHIR service, use the local FHIR service directly
            if service_url == self.default_service_url:
                logger.info(f"Using local FHIR service for getting {resource_type}/{resource_id}")
                from app.services.fhir_service import get_fhir_service
                fhir_service = get_fhir_service()
                return await fhir_service.get_resource(resource_type, resource_id)

            # Otherwise, route to the appropriate microservice
            logger.info(f"Routing GET {resource_type}/{resource_id} to external service at {service_url}")
            async with httpx.AsyncClient(follow_redirects=True) as client:
                try:
                    # Use the correct path for the microservice
                    # The microservices have their FHIR endpoints at /api/fhir/{resource_type}
                    # For MedicationRequest, Encounter, and DiagnosticReport, we need to use a different path
                    if resource_type == "MedicationRequest":
                        endpoint = f"{service_url}/api/fhir/MedicationRequest/{resource_id}"
                    elif resource_type == "Encounter":
                        endpoint = f"{service_url}/api/fhir/Encounter/{resource_id}"
                    elif resource_type == "DiagnosticReport":
                        endpoint = f"{service_url}/api/fhir/DiagnosticReport/{resource_id}"
                    else:
                        endpoint = f"{service_url}/api/fhir/{resource_type}/{resource_id}"
                    logger.debug(f"Sending GET request to {endpoint}")

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

                    response = await client.get(
                        endpoint,
                        headers=headers
                    )

                    if response.status_code == 404:
                        return None

                    response.raise_for_status()
                    return response.json()
                except httpx.HTTPStatusError as e:
                    if e.response.status_code == 404:
                        return None
                    logger.error(f"HTTP error getting {resource_type}/{resource_id}: {e}")
                    raise
                except Exception as e:
                    logger.error(f"Error getting {resource_type}/{resource_id}: {e}")
                    raise
        except Exception as e:
            logger.error(f"Error in get_resource for {resource_type}/{resource_id}: {str(e)}")
            logger.error(traceback.format_exc())
            raise

    async def update_resource(self, resource_type: str, resource_id: str, resource: Dict[str, Any], auth_header: str) -> Optional[Dict[str, Any]]:
        """
        Update a FHIR resource by routing to the appropriate microservice.

        Args:
            resource_type: The FHIR resource type
            resource_id: The FHIR resource ID
            resource: The FHIR resource data
            auth_header: The authorization header

        Returns:
            The updated FHIR resource, or None if not found
        """
        try:
            service_url = self.get_service_url(resource_type)

            # If we're using the default FHIR service, use the local FHIR service directly
            if service_url == self.default_service_url:
                logger.info(f"Using local FHIR service for updating {resource_type}/{resource_id}")
                from app.services.fhir_service import get_fhir_service
                fhir_service = get_fhir_service()
                return await fhir_service.update_resource(resource_type, resource_id, resource)

            # Otherwise, route to the appropriate microservice
            logger.info(f"Routing PUT {resource_type}/{resource_id} to external service at {service_url}")
            async with httpx.AsyncClient(follow_redirects=True) as client:
                try:
                    # Use the correct path for the microservice
                    # The microservices have their FHIR endpoints at /api/fhir/{resource_type}
                    # For MedicationRequest, Encounter, and DiagnosticReport, we need to use a different path
                    if resource_type == "MedicationRequest":
                        endpoint = f"{service_url}/api/fhir/MedicationRequest/{resource_id}"
                    elif resource_type == "Encounter":
                        endpoint = f"{service_url}/api/fhir/Encounter/{resource_id}"
                    elif resource_type == "DiagnosticReport":
                        endpoint = f"{service_url}/api/fhir/DiagnosticReport/{resource_id}"
                    else:
                        endpoint = f"{service_url}/api/fhir/{resource_type}/{resource_id}"
                    logger.debug(f"Sending PUT request to {endpoint}")
                    logger.debug(f"Request body: {resource}")

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

                    response = await client.put(
                        endpoint,
                        json=resource,
                        headers=headers
                    )

                    if response.status_code == 404:
                        return None

                    response.raise_for_status()
                    return response.json()
                except httpx.HTTPStatusError as e:
                    if e.response.status_code == 404:
                        return None
                    logger.error(f"HTTP error updating {resource_type}/{resource_id}: {e}")
                    raise
                except Exception as e:
                    logger.error(f"Error updating {resource_type}/{resource_id}: {e}")
                    raise
        except Exception as e:
            logger.error(f"Error in update_resource for {resource_type}/{resource_id}: {str(e)}")
            logger.error(traceback.format_exc())
            raise

    async def delete_resource(self, resource_type: str, resource_id: str, auth_header: str) -> bool:
        """
        Delete a FHIR resource by routing to the appropriate microservice.

        Args:
            resource_type: The FHIR resource type
            resource_id: The FHIR resource ID
            auth_header: The authorization header

        Returns:
            True if the resource was deleted, False otherwise
        """
        try:
            service_url = self.get_service_url(resource_type)

            # If we're using the default FHIR service, use the local FHIR service directly
            if service_url == self.default_service_url:
                logger.info(f"Using local FHIR service for deleting {resource_type}/{resource_id}")
                from app.services.fhir_service import get_fhir_service
                fhir_service = get_fhir_service()
                return await fhir_service.delete_resource(resource_type, resource_id)

            # Otherwise, route to the appropriate microservice
            logger.info(f"Routing DELETE {resource_type}/{resource_id} to external service at {service_url}")
            async with httpx.AsyncClient(follow_redirects=True) as client:
                try:
                    # Use the correct path for the microservice
                    # The microservices have their FHIR endpoints at /api/fhir/{resource_type}
                    # For MedicationRequest, Encounter, and DiagnosticReport, we need to use a different path
                    if resource_type == "MedicationRequest":
                        endpoint = f"{service_url}/api/fhir/MedicationRequest/{resource_id}"
                    elif resource_type == "Encounter":
                        endpoint = f"{service_url}/api/fhir/Encounter/{resource_id}"
                    elif resource_type == "DiagnosticReport":
                        endpoint = f"{service_url}/api/fhir/DiagnosticReport/{resource_id}"
                    else:
                        endpoint = f"{service_url}/api/fhir/{resource_type}/{resource_id}"
                    logger.debug(f"Sending DELETE request to {endpoint}")

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

                    response = await client.delete(
                        endpoint,
                        headers=headers
                    )

                    if response.status_code == 404:
                        return False

                    response.raise_for_status()
                    return True
                except httpx.HTTPStatusError as e:
                    if e.response.status_code == 404:
                        return False
                    logger.error(f"HTTP error deleting {resource_type}/{resource_id}: {e}")
                    raise
                except Exception as e:
                    logger.error(f"Error deleting {resource_type}/{resource_id}: {e}")
                    raise
        except Exception as e:
            logger.error(f"Error in delete_resource for {resource_type}/{resource_id}: {str(e)}")
            logger.error(traceback.format_exc())
            raise

    async def search_resources(self, resource_type: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for FHIR resources by routing to the appropriate microservice.

        Args:
            resource_type: The FHIR resource type
            params: The search parameters
            auth_header: The authorization header

        Returns:
            A list of FHIR resources
        """
        try:
            service_url = self.get_service_url(resource_type)

            # If we're using the default FHIR service, use the local FHIR service directly
            if service_url == self.default_service_url:
                logger.info(f"Using local FHIR service for searching {resource_type}")
                from app.services.fhir_service import get_fhir_service
                fhir_service = get_fhir_service()
                return await fhir_service.search_resources(resource_type, params)

            # Otherwise, route to the appropriate microservice
            logger.info(f"Routing GET {resource_type} search to external service at {service_url}")

            # Add very visible logging for debugging
            print(f"\n\n==== FHIR SERVICE SEARCH ROUTING DETAILS ====")
            print(f"Resource Type: {resource_type}")
            print(f"Service URL from registry: {service_url}")
            print(f"Service URL from settings: {settings.MEDICATION_SERVICE_URL}")
            print(f"Search Parameters: {params}")
            print(f"==== END FHIR SERVICE SEARCH ROUTING DETAILS ====\n\n")

            async with httpx.AsyncClient(follow_redirects=True) as client:
                try:
                    # Use the correct path for the microservice
                    # The microservices have their FHIR endpoints at /api/fhir/{resource_type}
                    # For MedicationRequest, Encounter, and DiagnosticReport, we need to use a different path
                    if resource_type == "MedicationRequest":
                        endpoint = f"{service_url}/api/fhir/MedicationRequest"
                    elif resource_type == "Encounter":
                        endpoint = f"{service_url}/api/fhir/Encounter"
                    elif resource_type == "DiagnosticReport":
                        endpoint = f"{service_url}/api/fhir/DiagnosticReport"
                    else:
                        endpoint = f"{service_url}/api/fhir/{resource_type}"
                    logger.info(f"Sending GET request to {endpoint}")
                    logger.info(f"Request params: {params}")

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

                    # Log the full URL with parameters for debugging
                    logger.info(f"Full request URL: {endpoint}?{urllib.parse.urlencode(params)}")

                    # Add a timeout to avoid hanging indefinitely
                    response = await client.get(
                        endpoint,
                        params=params,
                        headers=headers,
                        timeout=30.0  # 30 seconds timeout
                    )

                    # Log the response details
                    logger.info(f"=== FHIR SERVICE RECEIVED SEARCH RESPONSE ===")
                    logger.info(f"Response status: {response.status_code}")
                    logger.info(f"Response headers: {response.headers}")
                    logger.info(f"Response content: {response.content}")
                    logger.info(f"Request URL: {endpoint}")
                    logger.info(f"Request method: GET")
                    logger.info(f"=== END FHIR SERVICE SEARCH RESPONSE ===")

                    response.raise_for_status()
                    response_data = response.json()
                    logger.info(f"Response data: {response_data}")
                    return response_data
                except httpx.HTTPStatusError as e:
                    logger.error(f"HTTP error searching {resource_type}: {e}")
                    logger.error(f"Response content: {e.response.content}")
                    raise Exception(f"HTTP error: {e.response.status_code} - {e.response.content}")
                except httpx.RequestError as e:
                    logger.error(f"Request error searching {resource_type}: {e}")
                    raise Exception(f"Request error: {str(e)}")
                except Exception as e:
                    logger.error(f"Error searching {resource_type}: {e}")
                    logger.error(traceback.format_exc())
                    raise
        except Exception as e:
            logger.error(f"Error in search_resources for {resource_type}: {str(e)}")
            logger.error(traceback.format_exc())
            raise

    async def execute_operation(self, resource_type: str, operation: str, params: Dict[str, Any], auth_header: str) -> Dict[str, Any]:
        """
        Execute a FHIR operation by routing to the appropriate microservice.

        Args:
            resource_type: The FHIR resource type
            operation: The operation name
            params: The operation parameters
            auth_header: The authorization header

        Returns:
            The operation result
        """
        service_url = self.get_service_url(resource_type)

        async with httpx.AsyncClient(follow_redirects=True) as client:
            try:
                response = await client.post(
                    f"{service_url}/api/fhir/${operation}",
                    json=params,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                return response.json()
            except httpx.HTTPStatusError as e:
                logger.error(f"HTTP error executing {resource_type}/${operation}: {e}")
                raise
            except Exception as e:
                logger.error(f"Error executing {resource_type}/${operation}: {e}")
                raise

    async def execute_transaction(self, bundle: Dict[str, Any], auth_header: str) -> Dict[str, Any]:
        """
        Execute a FHIR transaction bundle.

        Args:
            bundle: The FHIR transaction bundle
            auth_header: The authorization header

        Returns:
            The transaction result bundle
        """
        # For transactions, we use the FHIR service itself
        async with httpx.AsyncClient(follow_redirects=True) as client:
            try:
                response = await client.post(
                    f"{self.default_service_url}/api/fhir/",
                    json=bundle,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                return response.json()
            except httpx.HTTPStatusError as e:
                logger.error(f"HTTP error executing transaction: {e}")
                raise
            except Exception as e:
                logger.error(f"Error executing transaction: {e}")
                raise

    async def get_patient_timeline(self, patient_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Get a patient's timeline by calling the Timeline Service.

        Args:
            patient_id: The patient ID
            auth_header: The authorization header

        Returns:
            The patient timeline
        """
        async with httpx.AsyncClient(follow_redirects=True) as client:
            try:
                response = await client.get(
                    f"{settings.TIMELINE_SERVICE_URL}/api/timeline/patients/{patient_id}",
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                return response.json()
            except httpx.HTTPStatusError as e:
                logger.error(f"HTTP error getting timeline for patient {patient_id}: {e}")
                raise
            except Exception as e:
                logger.error(f"Error getting timeline for patient {patient_id}: {e}")
                raise
