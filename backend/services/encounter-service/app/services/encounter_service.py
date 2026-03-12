from typing import Dict, List, Optional, Any
from bson import ObjectId
import httpx
import logging
from app.db.mongodb import db
from app.core.config import settings
from app.models.encounter import EncounterCreate, EncounterUpdate
from shared.models import Encounter

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class EncounterService:
    """Service for managing encounter resources."""

    async def create_encounter(self, encounter: EncounterCreate, auth_header: str) -> Dict[str, Any]:
        """
        Create a new encounter.

        Args:
            encounter: The encounter to create
            auth_header: The authorization header for API calls

        Returns:
            The created encounter
        """
        try:
            # Convert to FHIR Encounter
            fhir_encounter = encounter.to_fhir_encounter()

            # Convert to dict for API
            encounter_dict = fhir_encounter.model_dump(exclude_none=True)

            # Create in FHIR server
            return await self.create_resource_in_fhir_server("Encounter", encounter_dict, auth_header)
        except Exception as e:
            logger.error(f"Error creating encounter: {str(e)}")
            raise

    async def get_encounter(self, encounter_id: str, auth_header: str) -> Optional[Dict[str, Any]]:
        """
        Get an encounter by ID.

        Args:
            encounter_id: The encounter ID
            auth_header: The authorization header for API calls

        Returns:
            The encounter
        """
        try:
            # Get from FHIR server
            return await self.get_resource_from_fhir_server("Encounter", encounter_id, auth_header)
        except Exception as e:
            logger.error(f"Error getting encounter: {str(e)}")
            raise

    async def update_encounter(self, encounter_id: str, encounter: EncounterUpdate, auth_header: str) -> Optional[Dict[str, Any]]:
        """
        Update an encounter.

        Args:
            encounter_id: The encounter ID
            encounter: The encounter updates
            auth_header: The authorization header for API calls

        Returns:
            The updated encounter
        """
        try:
            # First, get the existing encounter
            existing_encounter = await self.get_encounter(encounter_id, auth_header)
            if not existing_encounter:
                return None

            # Convert update to FHIR update dict
            update_dict = encounter.to_fhir_encounter_update()

            # Merge with existing encounter
            for key, value in update_dict.items():
                existing_encounter[key] = value

            # Update in FHIR server
            return await self.update_resource_in_fhir_server("Encounter", encounter_id, existing_encounter, auth_header)
        except Exception as e:
            logger.error(f"Error updating encounter: {str(e)}")
            raise

    async def delete_encounter(self, encounter_id: str, auth_header: str) -> bool:
        """
        Delete an encounter.

        Args:
            encounter_id: The encounter ID
            auth_header: The authorization header for API calls

        Returns:
            True if deleted, False otherwise
        """
        try:
            # Delete from FHIR server
            return await self.delete_resource_from_fhir_server("Encounter", encounter_id, auth_header)
        except Exception as e:
            logger.error(f"Error deleting encounter: {str(e)}")
            raise

    async def search_encounters(self, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Search for encounters.

        Args:
            params: Search parameters
            auth_header: The authorization header for API calls

        Returns:
            List of encounters matching the search criteria
        """
        try:
            # Search in FHIR server
            return await self.search_resources_in_fhir_server("Encounter", params, auth_header)
        except Exception as e:
            logger.error(f"Error searching encounters: {str(e)}")
            raise

    async def get_patient_encounters(self, patient_id: str, params: Dict[str, Any], auth_header: str) -> List[Dict[str, Any]]:
        """
        Get encounters for a patient.

        Args:
            patient_id: The patient ID
            params: Additional search parameters
            auth_header: The authorization header for API calls

        Returns:
            The patient's encounters
        """
        try:
            # Add patient parameter
            params["subject"] = f"Patient/{patient_id}"

            # Search in FHIR server
            return await self.search_resources_in_fhir_server("Encounter", params, auth_header)
        except Exception as e:
            logger.error(f"Error getting patient encounters: {str(e)}")
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

    async def get_resource_from_fhir_server(self, resource_type: str, resource_id: str, auth_header: str) -> Optional[Dict[str, Any]]:
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
                        elif response.status_code == 404:
                            return None
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error getting {resource_type}/{resource_id} from FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            raise Exception(last_error or f"Failed to get {resource_type}/{resource_id} from FHIR server")
        except Exception as e:
            logger.error(f"Error getting resource from FHIR server: {str(e)}")
            raise

    async def update_resource_in_fhir_server(self, resource_type: str, resource_id: str, resource: Dict[str, Any], auth_header: str) -> Optional[Dict[str, Any]]:
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
                            last_error = f"Error updating {resource_type}/{resource_id} in FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            raise Exception(last_error or f"Failed to update {resource_type}/{resource_id} in FHIR server")
        except Exception as e:
            logger.error(f"Error updating resource in FHIR server: {str(e)}")
            raise

    async def delete_resource_from_fhir_server(self, resource_type: str, resource_id: str, auth_header: str) -> bool:
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

                        if response.status_code == 204:
                            return True
                        else:
                            logger.warning(f"Error response from {url}: {response.status_code} - {response.text}")
                            last_error = f"Error deleting {resource_type}/{resource_id} from FHIR server at {url}: {response.text}"
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
            logger.info(f"Params: {params}")

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
                            last_error = f"Error searching for {resource_type} in FHIR server at {url}: {response.text}"
                except Exception as e:
                    logger.warning(f"Exception trying {url}: {str(e)}")
                    last_error = f"Exception during FHIR server request to {url}: {str(e)}"

            # If we get here, all URLs failed
            raise Exception(last_error or f"Failed to search for {resource_type} in FHIR server")
        except Exception as e:
            logger.error(f"Error searching for resources in FHIR server: {str(e)}")
            raise

# Create a singleton instance
encounter_service = EncounterService()
