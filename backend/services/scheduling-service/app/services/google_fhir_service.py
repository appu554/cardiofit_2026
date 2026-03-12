"""
Google Healthcare API FHIR service for scheduling operations.

This service handles CRUD operations for scheduling-related FHIR resources:
- Appointment (appointment bookings)
- Schedule (provider schedules)
- Slot (available appointment slots)
- AppointmentResponse (appointment confirmations)
"""

import os
import sys
import json
import logging
from typing import Dict, List, Optional, Any
from google.auth.transport.requests import Request
from google.oauth2 import service_account
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', '..', '..'))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

from services.shared.google_healthcare import GoogleHealthcareClient

logger = logging.getLogger(__name__)

class SchedulingFHIRService:
    """
    FHIR service for scheduling operations using Google Healthcare API.
    
    This service handles CRUD operations for scheduling-related FHIR resources:
    - Appointment (appointment bookings)
    - Schedule (provider schedules)  
    - Slot (available appointment slots)
    - AppointmentResponse (appointment confirmations)
    """
    
    def __init__(self):
        """Initialize the Scheduling FHIR service."""
        self.client = None
        self.shared_client = None
        self.resource_types = [
            "Appointment",
            "Schedule",
            "Slot",
            "AppointmentResponse"
        ]
        self._initialized = False
        
    async def initialize(self) -> bool:
        """
        Initialize the Google Healthcare API client.

        Returns:
            bool: True if initialization successful, False otherwise
        """
        try:
            # Get configuration from environment
            project_id = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
            location = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
            dataset_id = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
            fhir_store_id = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")
            credentials_path = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")

            # Initialize the shared Google Healthcare client
            self.shared_client = GoogleHealthcareClient(
                project_id=project_id,
                location=location,
                dataset_id=dataset_id,
                fhir_store_id=fhir_store_id,
                credentials_path=credentials_path
            )

            # Initialize the shared client
            if not self.shared_client.initialize():
                logger.error("Failed to initialize shared Google Healthcare client")
                return False

            # Also keep the old client for operations that need it
            # Get credentials path from environment
            credentials_path = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")

            if not os.path.exists(credentials_path):
                logger.error(f"Google credentials file not found at: {credentials_path}")
                return False

            # Load service account credentials
            credentials = service_account.Credentials.from_service_account_file(
                credentials_path,
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
    
    async def create_appointment(self, appointment_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a new appointment in the FHIR store.
        
        Args:
            appointment_data: FHIR Appointment resource data
            
        Returns:
            Created appointment resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()
            
            # Ensure resource type is set
            appointment_data["resourceType"] = "Appointment"
            
            # Create the appointment
            request = self.client.projects().locations().datasets().fhirStores().fhir().create(
                parent=self.fhir_store_path,
                type="Appointment",
                body=appointment_data
            )
            
            response = request.execute()
            logger.info(f"Created appointment with ID: {response.get('id')}")
            return response
            
        except HttpError as e:
            logger.error(f"HTTP error creating appointment: {e}")
            return None
        except Exception as e:
            logger.error(f"Error creating appointment: {e}")
            return None
    
    async def get_appointment(self, appointment_id: str) -> Optional[Dict[str, Any]]:
        """
        Get an appointment by ID.
        
        Args:
            appointment_id: The appointment ID
            
        Returns:
            Appointment resource or None if not found
        """
        try:
            if not self._initialized:
                await self.initialize()
            
            request = self.client.projects().locations().datasets().fhirStores().fhir().read(
                name=f"{self.fhir_store_path}/fhir/Appointment/{appointment_id}"
            )
            
            response = request.execute()
            return response
            
        except HttpError as e:
            if e.resp.status == 404:
                logger.warning(f"Appointment not found: {appointment_id}")
                return None
            logger.error(f"HTTP error getting appointment: {e}")
            return None
        except Exception as e:
            logger.error(f"Error getting appointment: {e}")
            return None
    
    async def search_appointments(self, search_params: Dict[str, str]) -> List[Dict[str, Any]]:
        """
        Search for appointments using FHIR search parameters.

        Args:
            search_params: FHIR search parameters

        Returns:
            List of appointment resources
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Debug logging
            logger.info(f"Original search_params: {search_params}")

            # Use the shared client for search operations
            resources = await self.shared_client.search_resources("Appointment", search_params)

            logger.info(f"Found {len(resources)} appointments")
            return resources

        except Exception as e:
            logger.error(f"Error searching appointments: {e}")
            return []
    
    async def update_appointment(self, appointment_id: str, appointment_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update an existing appointment.

        Args:
            appointment_id: The appointment ID
            appointment_data: Updated appointment data

        Returns:
            Updated appointment resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Debug logging
            logger.info(f"Updating appointment {appointment_id} with data: {appointment_data}")

            # First, get the existing appointment to merge with updates
            existing_appointment = await self.get_appointment(appointment_id)
            if not existing_appointment:
                logger.error(f"Appointment {appointment_id} not found for update")
                return None

            # Merge the update data with existing appointment
            updated_appointment = existing_appointment.copy()
            updated_appointment.update(appointment_data)

            # Ensure resource type and ID are set
            updated_appointment["resourceType"] = "Appointment"
            updated_appointment["id"] = appointment_id

            logger.info(f"Final appointment data for update: {updated_appointment}")

            request = self.client.projects().locations().datasets().fhirStores().fhir().update(
                name=f"{self.fhir_store_path}/fhir/Appointment/{appointment_id}",
                body=updated_appointment
            )

            response = request.execute()
            logger.info(f"Updated appointment: {appointment_id}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error updating appointment: {e}")
            logger.error(f"Error details: {e.content if hasattr(e, 'content') else 'No details available'}")
            return None
        except Exception as e:
            logger.error(f"Error updating appointment: {e}")
            return None
    
    async def delete_appointment(self, appointment_id: str) -> bool:
        """
        Delete an appointment.
        
        Args:
            appointment_id: The appointment ID
            
        Returns:
            True if deleted successfully, False otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()
            
            request = self.client.projects().locations().datasets().fhirStores().fhir().delete(
                name=f"{self.fhir_store_path}/fhir/Appointment/{appointment_id}"
            )
            
            request.execute()
            logger.info(f"Deleted appointment: {appointment_id}")
            return True
            
        except HttpError as e:
            logger.error(f"HTTP error deleting appointment: {e}")
            return False
        except Exception as e:
            logger.error(f"Error deleting appointment: {e}")
            return False

    # Schedule operations
    async def create_schedule(self, schedule_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a new schedule in the FHIR store.

        Args:
            schedule_data: FHIR Schedule resource data

        Returns:
            Created schedule resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type is set
            schedule_data["resourceType"] = "Schedule"

            # Create the schedule
            request = self.client.projects().locations().datasets().fhirStores().fhir().create(
                parent=self.fhir_store_path,
                type="Schedule",
                body=schedule_data
            )

            response = request.execute()
            logger.info(f"Created schedule with ID: {response.get('id')}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error creating schedule: {e}")
            return None
        except Exception as e:
            logger.error(f"Error creating schedule: {e}")
            return None

    async def get_schedule(self, schedule_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a schedule by ID.

        Args:
            schedule_id: The schedule ID

        Returns:
            Schedule resource or None if not found
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().read(
                name=f"{self.fhir_store_path}/fhir/Schedule/{schedule_id}"
            )

            response = request.execute()
            return response

        except HttpError as e:
            if e.resp.status == 404:
                logger.warning(f"Schedule not found: {schedule_id}")
                return None
            logger.error(f"HTTP error getting schedule: {e}")
            return None
        except Exception as e:
            logger.error(f"Error getting schedule: {e}")
            return None

    async def search_schedules(self, search_params: Dict[str, str]) -> List[Dict[str, Any]]:
        """
        Search for schedules using FHIR search parameters.

        Args:
            search_params: FHIR search parameters

        Returns:
            List of schedule resources
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Use the shared client for search operations
            resources = await self.shared_client.search_resources("Schedule", search_params)

            logger.info(f"Found {len(resources)} schedules")
            return resources

        except Exception as e:
            logger.error(f"Error searching schedules: {e}")
            return []

    # Slot operations
    async def create_slot(self, slot_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a new slot in the FHIR store.

        Args:
            slot_data: FHIR Slot resource data

        Returns:
            Created slot resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type is set
            slot_data["resourceType"] = "Slot"

            # Create the slot
            request = self.client.projects().locations().datasets().fhirStores().fhir().create(
                parent=self.fhir_store_path,
                type="Slot",
                body=slot_data
            )

            response = request.execute()
            logger.info(f"Created slot with ID: {response.get('id')}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error creating slot: {e}")
            return None
        except Exception as e:
            logger.error(f"Error creating slot: {e}")
            return None

    async def get_slot(self, slot_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a slot by ID.

        Args:
            slot_id: The slot ID

        Returns:
            Slot resource or None if not found
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().read(
                name=f"{self.fhir_store_path}/fhir/Slot/{slot_id}"
            )

            response = request.execute()
            return response

        except HttpError as e:
            if e.resp.status == 404:
                logger.warning(f"Slot not found: {slot_id}")
                return None
            logger.error(f"HTTP error getting slot: {e}")
            return None
        except Exception as e:
            logger.error(f"Error getting slot: {e}")
            return None

    async def search_slots(self, search_params: Dict[str, str]) -> List[Dict[str, Any]]:
        """
        Search for slots using FHIR search parameters.

        Args:
            search_params: FHIR search parameters

        Returns:
            List of slot resources
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Use the shared client for search operations
            resources = await self.shared_client.search_resources("Slot", search_params)

            logger.info(f"Found {len(resources)} slots")
            return resources

        except Exception as e:
            logger.error(f"Error searching slots: {e}")
            return []

    async def update_schedule(self, schedule_id: str, schedule_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update an existing schedule.

        Args:
            schedule_id: The schedule ID
            schedule_data: Updated schedule data

        Returns:
            Updated schedule resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type and ID are set
            schedule_data["resourceType"] = "Schedule"
            schedule_data["id"] = schedule_id

            request = self.client.projects().locations().datasets().fhirStores().fhir().update(
                name=f"{self.fhir_store_path}/fhir/Schedule/{schedule_id}",
                body=schedule_data
            )

            response = request.execute()
            logger.info(f"Updated schedule: {schedule_id}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error updating schedule: {e}")
            return None
        except Exception as e:
            logger.error(f"Error updating schedule: {e}")
            return None

    async def delete_schedule(self, schedule_id: str) -> bool:
        """
        Delete a schedule.

        Args:
            schedule_id: The schedule ID

        Returns:
            True if deleted successfully, False otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().delete(
                name=f"{self.fhir_store_path}/fhir/Schedule/{schedule_id}"
            )

            request.execute()
            logger.info(f"Deleted schedule: {schedule_id}")
            return True

        except HttpError as e:
            logger.error(f"HTTP error deleting schedule: {e}")
            return False
        except Exception as e:
            logger.error(f"Error deleting schedule: {e}")
            return False

    async def update_slot(self, slot_id: str, slot_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update an existing slot.

        Args:
            slot_id: The slot ID
            slot_data: Updated slot data

        Returns:
            Updated slot resource or None if failed
        """
        try:
            if not self._initialized:
                await self.initialize()

            # Ensure resource type and ID are set
            slot_data["resourceType"] = "Slot"
            slot_data["id"] = slot_id

            request = self.client.projects().locations().datasets().fhirStores().fhir().update(
                name=f"{self.fhir_store_path}/fhir/Slot/{slot_id}",
                body=slot_data
            )

            response = request.execute()
            logger.info(f"Updated slot: {slot_id}")
            return response

        except HttpError as e:
            logger.error(f"HTTP error updating slot: {e}")
            return None
        except Exception as e:
            logger.error(f"Error updating slot: {e}")
            return None

    async def delete_slot(self, slot_id: str) -> bool:
        """
        Delete a slot.

        Args:
            slot_id: The slot ID

        Returns:
            True if deleted successfully, False otherwise
        """
        try:
            if not self._initialized:
                await self.initialize()

            request = self.client.projects().locations().datasets().fhirStores().fhir().delete(
                name=f"{self.fhir_store_path}/fhir/Slot/{slot_id}"
            )

            request.execute()
            logger.info(f"Deleted slot: {slot_id}")
            return True

        except HttpError as e:
            logger.error(f"HTTP error deleting slot: {e}")
            return False
        except Exception as e:
            logger.error(f"Error deleting slot: {e}")
            return False
