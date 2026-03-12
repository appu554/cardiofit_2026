"""
Google Healthcare API FHIR service for Encounter Management.

This service handles CRUD operations for encounter-related FHIR resources
using Google Cloud Healthcare API.
"""

import os
import sys
import json
import logging
from typing import Dict, List, Optional, Any
from google.oauth2 import service_account
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

# Configure logging
logger = logging.getLogger(__name__)

class EncounterFHIRService:
    """
    FHIR service for encounter operations using Google Healthcare API.
    
    This service handles CRUD operations for encounter-related FHIR resources:
    - Encounter (patient encounters and visits)
    - Location (physical locations within healthcare facilities)
    - EpisodeOfCare (care episodes spanning multiple encounters)
    """
    
    def __init__(self):
        """Initialize the Encounter FHIR service."""
        self.client = None
        self.shared_client = None
        self.resource_types = [
            "Encounter",
            "Location", 
            "EpisodeOfCare"
        ]
        self._initialized = False
        
    async def initialize(self) -> bool:
        """
        Initialize the Google Healthcare API client.
        
        Returns:
            bool: True if initialization successful, False otherwise
        """
        try:
            if self._initialized:
                return True

            # Get credentials path
            credentials_path = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")
            
            # Make path relative to the service directory
            if not os.path.isabs(credentials_path):
                service_dir = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
                credentials_path = os.path.join(service_dir, credentials_path)
            
            if not os.path.exists(credentials_path):
                logger.error(f"Google credentials file not found at: {credentials_path}")
                return False

            # Load credentials
            with open(credentials_path, 'r') as f:
                credentials_info = json.load(f)

            credentials = service_account.Credentials.from_service_account_info(
                credentials_info,
                scopes=['https://www.googleapis.com/auth/cloud-healthcare']
            )

            # Build the Healthcare API client
            self.client = build('healthcare', 'v1', credentials=credentials)

            # Set up the FHIR store path
            project_id = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
            location = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
            dataset_id = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
            fhir_store_id = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")

            self.fhir_store_path = f"projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores/{fhir_store_id}"

            logger.info(f"Initialized Google Healthcare API client for FHIR store: {self.fhir_store_path}")
            self._initialized = True
            return True

        except Exception as e:
            logger.error(f"Failed to initialize Google Healthcare API client: {e}")
            return False

    # Encounter operations
    async def create_encounter(self, encounter_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a new encounter in the FHIR store.

        Args:
            encounter_data: FHIR Encounter resource data

        Returns:
            Created encounter resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type is set
            encounter_data["resourceType"] = "Encounter"

            # Create the encounter
            request = self.client.projects().locations().datasets().fhirStores().fhir().create(
                parent=self.fhir_store_path,
                type="Encounter",
                body=encounter_data
            )

            response = request.execute()
            logger.info(f"Created encounter with ID: {response.get('id')}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error creating encounter: {e}")
            logger.error(f"HTTP error details: Status: {e.resp.status}, Content: {e.content}")
            logger.error(f"Encounter data being sent: {encounter_data}")
            return None
        except Exception as e:
            logger.error(f"Error creating encounter: {e}")
            return None

    async def get_encounter(self, encounter_id: str) -> Optional[Dict[str, Any]]:
        """
        Get an encounter by ID.

        Args:
            encounter_id: The encounter ID

        Returns:
            Encounter resource or None if not found
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().read(
                name=f"{self.fhir_store_path}/fhir/Encounter/{encounter_id}"
            )

            response = request.execute()
            logger.info(f"Retrieved encounter: {encounter_id}")
            return response

        except HttpError as e:
            if e.resp.status == 404:
                logger.warning(f"Encounter not found: {encounter_id}")
                return None
            logger.error(f"HTTP error retrieving encounter {encounter_id}: {e}")
            return None
        except Exception as e:
            logger.error(f"Error retrieving encounter {encounter_id}: {e}")
            return None

    async def update_encounter(self, encounter_id: str, encounter_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update an encounter.

        Args:
            encounter_id: The encounter ID
            encounter_data: Updated encounter data

        Returns:
            Updated encounter resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type and ID are set
            encounter_data["resourceType"] = "Encounter"
            encounter_data["id"] = encounter_id

            request = self.client.projects().locations().datasets().fhirStores().fhir().update(
                name=f"{self.fhir_store_path}/fhir/Encounter/{encounter_id}",
                body=encounter_data
            )

            response = request.execute()
            logger.info(f"Updated encounter: {encounter_id}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error updating encounter {encounter_id}: {e}")
            logger.error(f"HTTP error details: Status: {e.resp.status}, Content: {e.content}")
            logger.error(f"Encounter data being sent: {encounter_data}")
            return None
        except Exception as e:
            logger.error(f"Error updating encounter {encounter_id}: {e}")
            return None

    async def delete_encounter(self, encounter_id: str) -> bool:
        """
        Delete an encounter.

        Args:
            encounter_id: The encounter ID

        Returns:
            True if deleted successfully, False otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().delete(
                name=f"{self.fhir_store_path}/fhir/Encounter/{encounter_id}"
            )

            request.execute()
            logger.info(f"Deleted encounter: {encounter_id}")
            return True

        except HttpError as e:
            logger.error(f"HTTP error deleting encounter {encounter_id}: {e}")
            return False
        except Exception as e:
            logger.error(f"Error deleting encounter {encounter_id}: {e}")
            return False

    async def search_encounters(self, search_params: Dict[str, str]) -> Optional[Dict[str, Any]]:
        """
        Search for encounters using FHIR search parameters.

        Args:
            search_params: Dictionary of search parameters

        Returns:
            Bundle of encounter resources or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Use the shared client for search operations
            try:
                from services.shared.google_healthcare.client import GoogleHealthcareClient
            except ImportError:
                # Try alternative import path
                import sys
                backend_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))
                sys.path.insert(0, os.path.join(backend_dir, "services"))
                from services.shared.google_healthcare.client import GoogleHealthcareClient

            # Initialize shared client if needed
            if not hasattr(self, 'shared_client') or self.shared_client is None:
                credentials_path = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")
                if not os.path.isabs(credentials_path):
                    service_dir = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
                    credentials_path = os.path.join(service_dir, credentials_path)

                self.shared_client = GoogleHealthcareClient(
                    project_id=os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8"),
                    location=os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1"),
                    dataset_id=os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub"),
                    fhir_store_id=os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store"),
                    credentials_path=credentials_path
                )

                # Initialize the shared client
                if not self.shared_client.initialize():
                    logger.error("Failed to initialize shared Google Healthcare client")
                    return None

            # Search using the shared client
            resources = await self.shared_client.search_resources("Encounter", search_params)

            # Convert to bundle format for compatibility
            bundle = {
                "resourceType": "Bundle",
                "type": "searchset",
                "total": len(resources),
                "entry": [{"resource": resource} for resource in resources]
            }

            logger.info(f"Search encounters returned {len(resources)} results")
            return bundle

        except Exception as e:
            logger.error(f"Error searching encounters: {e}")
            return None

    async def get_encounters_by_patient(self, patient_id: str) -> List[Dict[str, Any]]:
        """
        Get all encounters for a specific patient.

        Args:
            patient_id: The patient ID

        Returns:
            List of encounter resources
        """
        try:
            search_params = {"subject": f"Patient/{patient_id}"}
            bundle = await self.search_encounters(search_params)
            
            if bundle and "entry" in bundle:
                return [entry["resource"] for entry in bundle["entry"]]
            return []

        except Exception as e:
            logger.error(f"Error getting encounters for patient {patient_id}: {e}")
            return []

    async def get_active_inpatient_encounters(self, organization_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Get all active inpatient encounters.

        Args:
            organization_id: Optional organization ID to filter by

        Returns:
            List of active inpatient encounter resources
        """
        try:
            search_params = {
                "status": "in-progress",
                "class": "IMP"  # Inpatient class
            }
            
            if organization_id:
                search_params["service-provider"] = f"Organization/{organization_id}"
            
            bundle = await self.search_encounters(search_params)
            
            if bundle and "entry" in bundle:
                return [entry["resource"] for entry in bundle["entry"]]
            return []

        except Exception as e:
            logger.error(f"Error getting active inpatient encounters: {e}")
            return []

    # Location operations
    async def create_location(self, location_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a new location in the FHIR store.

        Args:
            location_data: FHIR Location resource data

        Returns:
            Created location resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type is set
            location_data["resourceType"] = "Location"

            request = self.client.projects().locations().datasets().fhirStores().fhir().create(
                parent=self.fhir_store_path,
                type="Location",
                body=location_data
            )

            response = request.execute()
            logger.info(f"Created location with ID: {response.get('id')}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error creating location: {e}")
            return None
        except Exception as e:
            logger.error(f"Error creating location: {e}")
            return None

    async def get_location(self, location_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a location by ID.

        Args:
            location_id: The location ID

        Returns:
            Location resource or None if not found
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().read(
                name=f"{self.fhir_store_path}/fhir/Location/{location_id}"
            )

            response = request.execute()
            logger.info(f"Retrieved location: {location_id}")
            return response

        except HttpError as e:
            if e.resp.status == 404:
                logger.warning(f"Location not found: {location_id}")
                return None
            logger.error(f"HTTP error retrieving location {location_id}: {e}")
            return None
        except Exception as e:
            logger.error(f"Error retrieving location {location_id}: {e}")
            return None
