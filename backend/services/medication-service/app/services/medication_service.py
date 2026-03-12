from typing import Dict, List, Optional, Any, Union
import logging
import httpx
from bson import ObjectId
from datetime import datetime
from app.db.mongodb import db
from app.core.config import settings
from app.models.medication import (
    MedicationCreate, MedicationRequestCreate, MedicationAdministrationCreate, MedicationStatementCreate,
    MedicationUpdate, MedicationRequestUpdate, MedicationAdministrationUpdate, MedicationStatementUpdate
)
from shared.models import (
    Medication, MedicationRequest, MedicationAdministration, MedicationStatement
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Singleton instance
_medication_service_instance = None

def get_medication_service():
    """Get or create a singleton instance of the medication service."""
    global _medication_service_instance
    if _medication_service_instance is None:
        _medication_service_instance = MedicationService()
    return _medication_service_instance

class MedicationService:
    """Service for managing medication-related resources."""

    async def create_medication(self, medication: MedicationCreate, auth_header: str) -> Dict[str, Any]:
        """
        Create a new medication.

        Args:
            medication: The medication to create
            auth_header: The authorization header for API calls

        Returns:
            The created medication
        """
        try:
            # Convert to FHIR Medication
            fhir_medication = medication.to_fhir_medication()

            # Convert to dict for API
            medication_dict = fhir_medication.model_dump(exclude_none=True)

            # Create in FHIR server
            return await self.create_resource_in_fhir_server("Medication", medication_dict, auth_header)
        except Exception as e:
            logger.error(f"Error creating medication: {str(e)}")
            raise

    async def get_medication(self, medication_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Get a medication by ID.

        Args:
            medication_id: The medication ID
            auth_header: The authorization header for API calls

        Returns:
            The medication
        """
        try:
            # Get from FHIR server
            return await self.get_resource_from_fhir_server("Medication", medication_id, auth_header)
        except Exception as e:
            logger.error(f"Error getting medication: {str(e)}")
            raise

    async def update_medication(self, medication_id: str, medication: MedicationUpdate, auth_header: str) -> Dict[str, Any]:
        """
        Update a medication.

        Args:
            medication_id: The medication ID
            medication: The medication updates
            auth_header: The authorization header for API calls

        Returns:
            The updated medication
        """
        try:
            # Get current medication
            current_medication = await self.get_medication(medication_id, auth_header)

            # Convert to FHIR update dict
            update_dict = medication.to_fhir_medication_update()

            # Update fields
            for key, value in update_dict.items():
                current_medication[key] = value

            # Update in FHIR server
            return await self.update_resource_in_fhir_server("Medication", medication_id, current_medication, auth_header)
        except Exception as e:
            logger.error(f"Error updating medication: {str(e)}")
            raise

    async def delete_medication(self, medication_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Delete a medication.

        Args:
            medication_id: The medication ID
            auth_header: The authorization header for API calls

        Returns:
            The deletion result
        """
        try:
            # Delete from FHIR server
            return await self.delete_resource_from_fhir_server("Medication", medication_id, auth_header)
        except Exception as e:
            logger.error(f"Error deleting medication: {str(e)}")
            raise

    async def search_medications(self, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for medications.

        Args:
            params: The search parameters
            auth_header: The authorization header for API calls

        Returns:
            The matching medications
        """
        try:
            # Search in FHIR server
            return await self.search_resources_in_fhir_server("Medication", params, auth_header)
        except Exception as e:
            logger.error(f"Error searching medications: {str(e)}")
            raise

    # MedicationRequest methods

    async def create_medication_request(self, medication_request: MedicationRequestCreate, auth_header: str) -> Dict[str, Any]:
        """
        Create a new medication request.

        Args:
            medication_request: The medication request to create
            auth_header: The authorization header for API calls

        Returns:
            The created medication request
        """
        try:
            # Convert to FHIR MedicationRequest
            fhir_medication_request = medication_request.to_fhir_medication_request()

            # Convert to dict for API
            medication_request_dict = fhir_medication_request.model_dump(exclude_none=True)

            # Create in FHIR server
            return await self.create_resource_in_fhir_server("MedicationRequest", medication_request_dict, auth_header)
        except Exception as e:
            logger.error(f"Error creating medication request: {str(e)}")
            raise

    async def get_medication_request(self, medication_request_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Get a medication request by ID.

        Args:
            medication_request_id: The medication request ID
            auth_header: The authorization header for API calls

        Returns:
            The medication request
        """
        try:
            # Get from FHIR server
            return await self.get_resource_from_fhir_server("MedicationRequest", medication_request_id, auth_header)
        except Exception as e:
            logger.error(f"Error getting medication request: {str(e)}")
            raise

    async def update_medication_request(self, medication_request_id: str, medication_request: MedicationRequestUpdate, auth_header: str) -> Dict[str, Any]:
        """
        Update a medication request.

        Args:
            medication_request_id: The medication request ID
            medication_request: The medication request updates
            auth_header: The authorization header for API calls

        Returns:
            The updated medication request
        """
        try:
            # Get current medication request
            current_medication_request = await self.get_medication_request(medication_request_id, auth_header)

            # Convert to FHIR update dict
            update_dict = medication_request.to_fhir_medication_request_update()

            # Update fields
            for key, value in update_dict.items():
                current_medication_request[key] = value

            # Update in FHIR server
            return await self.update_resource_in_fhir_server("MedicationRequest", medication_request_id, current_medication_request, auth_header)
        except Exception as e:
            logger.error(f"Error updating medication request: {str(e)}")
            raise

    async def delete_medication_request(self, medication_request_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Delete a medication request.

        Args:
            medication_request_id: The medication request ID
            auth_header: The authorization header for API calls

        Returns:
            The deletion result
        """
        try:
            # Delete from FHIR server
            return await self.delete_resource_from_fhir_server("MedicationRequest", medication_request_id, auth_header)
        except Exception as e:
            logger.error(f"Error deleting medication request: {str(e)}")
            raise

    async def search_medication_requests(self, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for medication requests.

        Args:
            params: The search parameters
            auth_header: The authorization header for API calls

        Returns:
            The matching medication requests
        """
        try:
            # Search in FHIR server
            return await self.search_resources_in_fhir_server("MedicationRequest", params, auth_header)
        except Exception as e:
            logger.error(f"Error searching medication requests: {str(e)}")
            raise

    async def get_patient_medication_requests(self, patient_id: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Get medication requests for a patient.

        Args:
            patient_id: The patient ID
            params: Additional search parameters
            auth_header: The authorization header for API calls

        Returns:
            The patient's medication requests
        """
        try:
            # Add patient parameter
            params["subject"] = f"Patient/{patient_id}"

            # Search in FHIR server
            return await self.search_resources_in_fhir_server("MedicationRequest", params, auth_header)
        except Exception as e:
            logger.error(f"Error getting patient medication requests: {str(e)}")
            raise

    # MedicationAdministration methods

    async def create_medication_administration(self, medication_administration: MedicationAdministrationCreate, auth_header: str) -> Dict[str, Any]:
        """
        Create a new medication administration.

        Args:
            medication_administration: The medication administration to create
            auth_header: The authorization header for API calls

        Returns:
            The created medication administration
        """
        try:
            # Convert to FHIR MedicationAdministration
            fhir_medication_administration = medication_administration.to_fhir_medication_administration()

            # Convert to dict for API
            medication_administration_dict = fhir_medication_administration.model_dump(exclude_none=True)

            # Create in FHIR server
            return await self.create_resource_in_fhir_server("MedicationAdministration", medication_administration_dict, auth_header)
        except Exception as e:
            logger.error(f"Error creating medication administration: {str(e)}")
            raise

    async def get_medication_administration(self, medication_administration_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Get a medication administration by ID.

        Args:
            medication_administration_id: The medication administration ID
            auth_header: The authorization header for API calls

        Returns:
            The medication administration
        """
        try:
            # Get from FHIR server
            return await self.get_resource_from_fhir_server("MedicationAdministration", medication_administration_id, auth_header)
        except Exception as e:
            logger.error(f"Error getting medication administration: {str(e)}")
            raise

    async def update_medication_administration(self, medication_administration_id: str, medication_administration: MedicationAdministrationUpdate, auth_header: str) -> Dict[str, Any]:
        """
        Update a medication administration.

        Args:
            medication_administration_id: The medication administration ID
            medication_administration: The medication administration updates
            auth_header: The authorization header for API calls

        Returns:
            The updated medication administration
        """
        try:
            # Get current medication administration
            current_medication_administration = await self.get_medication_administration(medication_administration_id, auth_header)

            # Convert to FHIR update dict
            update_dict = medication_administration.to_fhir_medication_administration_update()

            # Update fields
            for key, value in update_dict.items():
                current_medication_administration[key] = value

            # Update in FHIR server
            return await self.update_resource_in_fhir_server("MedicationAdministration", medication_administration_id, current_medication_administration, auth_header)
        except Exception as e:
            logger.error(f"Error updating medication administration: {str(e)}")
            raise

    async def delete_medication_administration(self, medication_administration_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Delete a medication administration.

        Args:
            medication_administration_id: The medication administration ID
            auth_header: The authorization header for API calls

        Returns:
            The deletion result
        """
        try:
            # Delete from FHIR server
            return await self.delete_resource_from_fhir_server("MedicationAdministration", medication_administration_id, auth_header)
        except Exception as e:
            logger.error(f"Error deleting medication administration: {str(e)}")
            raise

    async def search_medication_administrations(self, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for medication administrations.

        Args:
            params: The search parameters
            auth_header: The authorization header for API calls

        Returns:
            The matching medication administrations
        """
        try:
            # Search in FHIR server
            return await self.search_resources_in_fhir_server("MedicationAdministration", params, auth_header)
        except Exception as e:
            logger.error(f"Error searching medication administrations: {str(e)}")
            raise

    async def get_patient_medication_administrations(self, patient_id: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Get medication administrations for a patient.

        Args:
            patient_id: The patient ID
            params: Additional search parameters
            auth_header: The authorization header for API calls

        Returns:
            The patient's medication administrations
        """
        try:
            # Add patient parameter
            params["subject"] = f"Patient/{patient_id}"

            # Search in FHIR server
            return await self.search_resources_in_fhir_server("MedicationAdministration", params, auth_header)
        except Exception as e:
            logger.error(f"Error getting patient medication administrations: {str(e)}")
            raise

    # MedicationStatement methods

    async def create_medication_statement(self, medication_statement: MedicationStatementCreate, auth_header: str) -> Dict[str, Any]:
        """
        Create a new medication statement.

        Args:
            medication_statement: The medication statement to create
            auth_header: The authorization header for API calls

        Returns:
            The created medication statement
        """
        try:
            # Convert to FHIR MedicationStatement
            fhir_medication_statement = medication_statement.to_fhir_medication_statement()

            # Convert to dict for API
            medication_statement_dict = fhir_medication_statement.model_dump(exclude_none=True)

            # Create in FHIR server
            return await self.create_resource_in_fhir_server("MedicationStatement", medication_statement_dict, auth_header)
        except Exception as e:
            logger.error(f"Error creating medication statement: {str(e)}")
            raise

    async def get_medication_statement(self, medication_statement_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Get a medication statement by ID.

        Args:
            medication_statement_id: The medication statement ID
            auth_header: The authorization header for API calls

        Returns:
            The medication statement
        """
        try:
            # Get from FHIR server
            return await self.get_resource_from_fhir_server("MedicationStatement", medication_statement_id, auth_header)
        except Exception as e:
            logger.error(f"Error getting medication statement: {str(e)}")
            raise

    async def update_medication_statement(self, medication_statement_id: str, medication_statement: MedicationStatementUpdate, auth_header: str) -> Dict[str, Any]:
        """
        Update a medication statement.

        Args:
            medication_statement_id: The medication statement ID
            medication_statement: The medication statement updates
            auth_header: The authorization header for API calls

        Returns:
            The updated medication statement
        """
        try:
            # Get current medication statement
            current_medication_statement = await self.get_medication_statement(medication_statement_id, auth_header)

            # Convert to FHIR update dict
            update_dict = medication_statement.to_fhir_medication_statement_update()

            # Update fields
            for key, value in update_dict.items():
                current_medication_statement[key] = value

            # Update in FHIR server
            return await self.update_resource_in_fhir_server("MedicationStatement", medication_statement_id, current_medication_statement, auth_header)
        except Exception as e:
            logger.error(f"Error updating medication statement: {str(e)}")
            raise

    async def delete_medication_statement(self, medication_statement_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Delete a medication statement.

        Args:
            medication_statement_id: The medication statement ID
            auth_header: The authorization header for API calls

        Returns:
            The deletion result
        """
        try:
            # Delete from FHIR server
            return await self.delete_resource_from_fhir_server("MedicationStatement", medication_statement_id, auth_header)
        except Exception as e:
            logger.error(f"Error deleting medication statement: {str(e)}")
            raise

    async def search_medication_statements(self, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for medication statements.

        Args:
            params: The search parameters
            auth_header: The authorization header for API calls

        Returns:
            The matching medication statements
        """
        try:
            # Search in FHIR server
            return await self.search_resources_in_fhir_server("MedicationStatement", params, auth_header)
        except Exception as e:
            logger.error(f"Error searching medication statements: {str(e)}")
            raise

    async def get_patient_medication_statements(self, patient_id: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Get medication statements for a patient.

        Args:
            patient_id: The patient ID
            params: Additional search parameters
            auth_header: The authorization header for API calls

        Returns:
            The patient's medication statements
        """
        try:
            # Add patient parameter
            params["subject"] = f"Patient/{patient_id}"

            # Search in FHIR server
            return await self.search_resources_in_fhir_server("MedicationStatement", params, auth_header)
        except Exception as e:
            logger.error(f"Error getting patient medication statements: {str(e)}")
            raise

    # FHIR server interaction methods

    async def create_resource_in_fhir_server(self, resource_type: str, resource: Dict[str, Any], auth_header: str) -> Dict[str, Any]:
        """Create a resource in the FHIR server."""
        try:
            # Try different FHIR server URLs
            fhir_urls = [
                f"{settings.FHIR_SERVICE_URL}/fhir/{resource_type}",  # Standard path
                f"{settings.FHIR_SERVICE_URL}/api/fhir/{resource_type}",  # With /api prefix
                f"http://localhost:8004/fhir/{resource_type}",  # Direct localhost
                f"http://localhost:8004/api/fhir/{resource_type}"  # Direct localhost with /api
            ]

            logger.info(f"Creating {resource_type} in FHIR server")
            logger.info(f"Resource: {resource}")

            last_error = None

            for url in fhir_urls:
                try:
                    logger.info(f"Trying FHIR server URL: {url}")

                    async with httpx.AsyncClient() as client:
                        response = await client.post(
                            url,
                            json=resource,
                            headers={"Authorization": auth_header},
                            timeout=10.0  # 10 second timeout
                        )

                        logger.info(f"FHIR server response: {response.status_code}")

                        if response.status_code == 201:
                            return response.json()
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error creating {resource_type} in FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            error_msg = last_error or f"All FHIR server URLs failed for {resource_type}"
            logger.error(error_msg)
            raise Exception(error_msg)
        except Exception as e:
            logger.error(f"Exception during FHIR server request: {str(e)}")
            raise

    async def get_resource_from_fhir_server(self, resource_type: str, resource_id: str, auth_header: str) -> Dict[str, Any]:
        """Get a resource from the FHIR server."""
        try:
            # Try different FHIR server URLs
            fhir_urls = [
                f"{settings.FHIR_SERVICE_URL}/fhir/{resource_type}/{resource_id}",  # Standard path
                f"{settings.FHIR_SERVICE_URL}/api/fhir/{resource_type}/{resource_id}",  # With /api prefix
                f"http://localhost:8004/fhir/{resource_type}/{resource_id}",  # Direct localhost
                f"http://localhost:8004/api/fhir/{resource_type}/{resource_id}"  # Direct localhost with /api
            ]

            logger.info(f"Getting {resource_type}/{resource_id} from FHIR server")

            last_error = None

            for url in fhir_urls:
                try:
                    logger.info(f"Trying FHIR server URL: {url}")

                    async with httpx.AsyncClient() as client:
                        response = await client.get(
                            url,
                            headers={"Authorization": auth_header},
                            timeout=10.0  # 10 second timeout
                        )

                        logger.info(f"FHIR server response: {response.status_code}")

                        if response.status_code == 200:
                            return response.json()
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error getting {resource_type} from FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            error_msg = last_error or f"All FHIR server URLs failed for {resource_type}/{resource_id}"
            logger.error(error_msg)
            raise Exception(error_msg)
        except Exception as e:
            logger.error(f"Exception during FHIR server request: {str(e)}")
            raise

    async def update_resource_in_fhir_server(self, resource_type: str, resource_id: str, resource: Dict[str, Any], auth_header: str) -> Dict[str, Any]:
        """Update a resource in the FHIR server."""
        try:
            # Try different FHIR server URLs
            fhir_urls = [
                f"{settings.FHIR_SERVICE_URL}/fhir/{resource_type}/{resource_id}",  # Standard path
                f"{settings.FHIR_SERVICE_URL}/api/fhir/{resource_type}/{resource_id}",  # With /api prefix
                f"http://localhost:8004/fhir/{resource_type}/{resource_id}",  # Direct localhost
                f"http://localhost:8004/api/fhir/{resource_type}/{resource_id}"  # Direct localhost with /api
            ]

            logger.info(f"Updating {resource_type}/{resource_id} in FHIR server")
            logger.info(f"Resource: {resource}")

            last_error = None

            for url in fhir_urls:
                try:
                    logger.info(f"Trying FHIR server URL: {url}")

                    async with httpx.AsyncClient() as client:
                        response = await client.put(
                            url,
                            json=resource,
                            headers={"Authorization": auth_header},
                            timeout=10.0  # 10 second timeout
                        )

                        logger.info(f"FHIR server response: {response.status_code}")

                        if response.status_code == 200:
                            return response.json()
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error updating {resource_type} in FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            error_msg = last_error or f"All FHIR server URLs failed for updating {resource_type}/{resource_id}"
            logger.error(error_msg)
            raise Exception(error_msg)
        except Exception as e:
            logger.error(f"Exception during FHIR server request: {str(e)}")
            raise

    async def delete_resource_from_fhir_server(self, resource_type: str, resource_id: str, auth_header: str) -> Dict[str, Any]:
        """Delete a resource from the FHIR server."""
        try:
            # Try different FHIR server URLs
            fhir_urls = [
                f"{settings.FHIR_SERVICE_URL}/fhir/{resource_type}/{resource_id}",  # Standard path
                f"{settings.FHIR_SERVICE_URL}/api/fhir/{resource_type}/{resource_id}",  # With /api prefix
                f"http://localhost:8004/fhir/{resource_type}/{resource_id}",  # Direct localhost
                f"http://localhost:8004/api/fhir/{resource_type}/{resource_id}"  # Direct localhost with /api
            ]

            logger.info(f"Deleting {resource_type}/{resource_id} from FHIR server")

            last_error = None

            for url in fhir_urls:
                try:
                    logger.info(f"Trying FHIR server URL: {url}")

                    async with httpx.AsyncClient() as client:
                        response = await client.delete(
                            url,
                            headers={"Authorization": auth_header},
                            timeout=10.0  # 10 second timeout
                        )

                        logger.info(f"FHIR server response: {response.status_code}")

                        if response.status_code == 200:
                            return response.json()
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error deleting {resource_type} from FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            error_msg = last_error or f"All FHIR server URLs failed for deleting {resource_type}/{resource_id}"
            logger.error(error_msg)
            raise Exception(error_msg)
        except Exception as e:
            logger.error(f"Exception during FHIR server request: {str(e)}")
            raise

    async def search_resources_in_fhir_server(self, resource_type: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """Search for resources in the FHIR server."""
        try:
            # Try different FHIR server URLs
            fhir_urls = [
                f"{settings.FHIR_SERVICE_URL}/fhir/{resource_type}",  # Standard path
                f"{settings.FHIR_SERVICE_URL}/api/fhir/{resource_type}",  # With /api prefix
                f"http://localhost:8004/fhir/{resource_type}",  # Direct localhost
                f"http://localhost:8004/api/fhir/{resource_type}"  # Direct localhost with /api
            ]

            logger.info(f"Searching {resource_type} in FHIR server with params: {params}")

            last_error = None

            for url in fhir_urls:
                try:
                    logger.info(f"Trying FHIR server URL: {url}")

                    async with httpx.AsyncClient() as client:
                        response = await client.get(
                            url,
                            params=params,
                            headers={"Authorization": auth_header},
                            timeout=10.0  # 10 second timeout
                        )

                        logger.info(f"FHIR server response: {response.status_code}")

                        if response.status_code == 200:
                            return response.json()
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error searching {resource_type} in FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            error_msg = last_error or f"All FHIR server URLs failed for searching {resource_type}"
            logger.error(error_msg)
            raise Exception(error_msg)
        except Exception as e:
            logger.error(f"Exception during FHIR server request: {str(e)}")
            raise
