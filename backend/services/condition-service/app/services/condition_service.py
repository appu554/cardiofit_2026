import logging
import httpx
from typing import Dict, List, Any, Optional
from app.core.config import settings
from app.models.condition import ConditionCreate, ConditionUpdate
from shared.models import Condition

logger = logging.getLogger(__name__)

class ConditionService:
    """Service for managing condition resources."""

    async def create_condition(self, condition: ConditionCreate, auth_header: str) -> Dict[str, Any]:
        """
        Create a new condition.

        Args:
            condition: The condition to create
            auth_header: The authorization header for API calls

        Returns:
            The created condition
        """
        try:
            # Convert to FHIR Condition
            fhir_condition = condition.to_fhir_condition()

            # Convert to dict for API
            condition_dict = fhir_condition.model_dump(exclude_none=True)

            # Create in FHIR server
            return await self.create_resource_in_fhir_server("Condition", condition_dict, auth_header)
        except Exception as e:
            logger.error(f"Error creating condition: {str(e)}")
            raise

    async def get_condition(self, condition_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Get a condition by ID.

        Args:
            condition_id: The condition ID
            auth_header: The authorization header for API calls

        Returns:
            The condition
        """
        try:
            # Get from FHIR server
            return await self.get_resource_from_fhir_server("Condition", condition_id, auth_header)
        except Exception as e:
            logger.error(f"Error getting condition: {str(e)}")
            raise

    async def update_condition(self, condition_id: str, condition: ConditionUpdate, auth_header: str) -> Dict[str, Any]:
        """
        Update a condition.

        Args:
            condition_id: The condition ID
            condition: The condition updates
            auth_header: The authorization header for API calls

        Returns:
            The updated condition
        """
        try:
            # Get existing condition
            existing_condition = await self.get_condition(condition_id, auth_header)

            # Convert update to dict
            update_dict = condition.to_fhir_condition_update()

            # Update existing condition
            for key, value in update_dict.items():
                existing_condition[key] = value

            # Update in FHIR server
            return await self.update_resource_in_fhir_server("Condition", condition_id, existing_condition, auth_header)
        except Exception as e:
            logger.error(f"Error updating condition: {str(e)}")
            raise

    async def delete_condition(self, condition_id: str, auth_header: str) -> Dict[str, Any]:
        """
        Delete a condition.

        Args:
            condition_id: The condition ID
            auth_header: The authorization header for API calls

        Returns:
            The deletion response
        """
        try:
            # Delete from FHIR server
            return await self.delete_resource_from_fhir_server("Condition", condition_id, auth_header)
        except Exception as e:
            logger.error(f"Error deleting condition: {str(e)}")
            raise

    async def search_conditions(self, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for conditions.

        Args:
            params: Search parameters
            auth_header: The authorization header for API calls

        Returns:
            The matching conditions
        """
        try:
            # Search in FHIR server
            return await self.search_resources_in_fhir_server("Condition", params, auth_header)
        except Exception as e:
            logger.error(f"Error searching conditions: {str(e)}")
            raise

    async def get_patient_conditions(self, patient_id: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Get conditions for a patient.

        Args:
            patient_id: The patient ID
            params: Additional search parameters
            auth_header: The authorization header for API calls

        Returns:
            The patient's conditions
        """
        try:
            # Add patient parameter
            params["subject"] = f"Patient/{patient_id}"

            # Search in FHIR server
            conditions = await self.search_resources_in_fhir_server("Condition", params, auth_header)

            # Additional filtering to ensure we only get conditions for this patient
            # This is a safeguard in case the FHIR server's filtering isn't working correctly
            filtered_conditions = []
            for condition in conditions:
                if 'subject' in condition and isinstance(condition['subject'], dict):
                    subject_ref = condition['subject'].get('reference', '')
                    # Check if the reference matches the patient ID exactly or ends with the patient ID
                    if subject_ref == f"Patient/{patient_id}" or subject_ref.endswith(f"/{patient_id}"):
                        filtered_conditions.append(condition)

            logger.info(f"Filtered {len(conditions)} conditions to {len(filtered_conditions)} for patient {patient_id}")
            return filtered_conditions
        except Exception as e:
            logger.error(f"Error getting patient conditions: {str(e)}")
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
            raise Exception(last_error or f"Failed to create {resource_type} in FHIR server")
        except Exception as e:
            logger.error(f"Error creating resource in FHIR server: {str(e)}")
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
            raise Exception(last_error or f"Failed to get {resource_type}/{resource_id} from FHIR server")
        except Exception as e:
            logger.error(f"Error getting resource from FHIR server: {str(e)}")
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
            raise Exception(last_error or f"Failed to update {resource_type}/{resource_id} in FHIR server")
        except Exception as e:
            logger.error(f"Error updating resource in FHIR server: {str(e)}")
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

                        if response.status_code == 200 or response.status_code == 204:
                            return {"status": "success", "message": f"{resource_type} {resource_id} deleted"}
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error deleting {resource_type} from FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            raise Exception(last_error or f"Failed to delete {resource_type}/{resource_id} from FHIR server")
        except Exception as e:
            logger.error(f"Error deleting resource from FHIR server: {str(e)}")
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

            logger.info(f"Searching for {resource_type} in FHIR server")
            logger.info(f"Search parameters: {params}")

            # Special handling for patient filtering
            patient_filter = None
            if 'subject' in params and params['subject'].startswith('Patient/'):
                patient_id = params['subject'].split('/', 1)[1]
                patient_filter = patient_id
                logger.info(f"Detected patient filter for ID: {patient_id}")

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
                            results = response.json()

                            # Additional filtering for patient if needed
                            if patient_filter and resource_type == 'Condition':
                                logger.info(f"Performing additional patient filtering for {patient_filter}")
                                filtered_results = []
                                for item in results:
                                    if 'subject' in item and isinstance(item['subject'], dict):
                                        subject_ref = item['subject'].get('reference', '')
                                        if subject_ref == f"Patient/{patient_filter}" or subject_ref.endswith(f"/{patient_filter}"):
                                            filtered_results.append(item)

                                logger.info(f"Filtered {len(results)} results to {len(filtered_results)} for patient {patient_filter}")
                                return filtered_results

                            return results
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error searching for {resource_type} in FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            raise Exception(last_error or f"Failed to search for {resource_type} in FHIR server")
        except Exception as e:
            logger.error(f"Error searching for resources in FHIR server: {str(e)}")
            raise
