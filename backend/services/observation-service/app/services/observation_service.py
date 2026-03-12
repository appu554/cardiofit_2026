import logging
from typing import List, Optional, Dict, Any, Union
from datetime import datetime
import uuid
import json
from app.core.config import settings

# Import FHIR service factory
from app.services.fhir_service_factory import get_fhir_service, initialize_fhir_service

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Singleton instance
_observation_service_instance = None

async def get_observation_service():
    """
    Get or create a singleton instance of the Observation service.
    
    Returns:
        ObservationService: The singleton instance of the Observation service
    """
    global _observation_service_instance
    if _observation_service_instance is None:
        _observation_service_instance = ObservationService()
        await _observation_service_instance.initialize()
    return _observation_service_instance

class ObservationService:
    """
    Service for managing Observation resources.
    
    This service provides a higher-level API for managing observations,
    using the appropriate FHIR service implementation based on configuration.
    """
    
    def __init__(self):
        """Initialize the Observation service."""
        self.fhir_service = None
        logger.info("Initialized ObservationService")
    
    async def initialize(self):
        """Initialize the FHIR service."""
        # Initialize the FHIR service
        self.fhir_service = await initialize_fhir_service()
        logger.info(f"Initialized FHIR service: {type(self.fhir_service).__name__}")
        
        # Initialize in-memory storage for testing if needed
        self.observations = {}  # In-memory storage for testing
        self.collection = None  # Will be initialized when needed
        logger.info("ObservationService initialization complete")

    def _format_observation(self, observation: Dict[str, Any]) -> Dict[str, Any]:
        """Format observation data for the API response."""
        return observation

    async def create_observation(self, observation_data: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new observation.

        Args:
            observation_data: The observation data
            token_payload: The decoded JWT token payload

        Returns:
            The created observation
        """
        try:
            # Generate a new ID if not provided
            if "id" not in observation_data:
                observation_data["id"] = str(uuid.uuid4())

            # Set the resource type
            observation_data["resourceType"] = "Observation"

            # Set the timestamp if not provided
            if "effectiveDateTime" not in observation_data:
                observation_data["effectiveDateTime"] = datetime.utcnow().isoformat() + "Z"

            # Create the observation using the FHIR service
            created_observation = await self.fhir_service.create(observation_data, token_payload)

            logger.info(f"Created observation with ID: {created_observation['id']}")
            return self._format_observation(created_observation)

        except Exception as e:
            logger.error(f"Error creating observation: {str(e)}")
            raise

    async def get_observation_by_id(self, observation_id: str, token_payload: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Get an observation by ID.

        Args:
            observation_id: The observation ID
            token_payload: The decoded JWT token payload

        Returns:
            The observation if found, None otherwise
        """
        try:
            # Get the observation using the FHIR service
            observation = await self.fhir_service.read(observation_id, token_payload)

            if observation:
                logger.info(f"Retrieved observation with ID: {observation_id}")
                return self._format_observation(observation)
            else:
                logger.warning(f"Observation with ID {observation_id} not found")
                return None

        except Exception as e:
            logger.error(f"Error retrieving observation with ID {observation_id}: {str(e)}")
            raise

    async def update_observation(
        self, observation_id: str, observation_data: Dict[str, Any], token_payload: Dict[str, Any]
    ) -> Optional[Dict[str, Any]]:
        """
        Update an observation.

        Args:
            observation_id: The observation ID
            observation_data: The updated observation data
            token_payload: The decoded JWT token payload

        Returns:
            The updated observation if found, None otherwise
        """
        try:
            # Check if the observation exists
            existing_observation = await self.get_observation_by_id(observation_id, token_payload)
            if not existing_observation:
                return None

            # Update the observation data
            updated_data = {**existing_observation, **observation_data}
            updated_data["id"] = observation_id
            updated_data["resourceType"] = "Observation"

            # Update the observation using the FHIR service
            updated_observation = await self.fhir_service.update(observation_id, updated_data, token_payload)

            logger.info(f"Updated observation with ID: {observation_id}")
            return self._format_observation(updated_observation)

        except Exception as e:
            logger.error(f"Error updating observation with ID {observation_id}: {str(e)}")
            raise

    async def delete_observation(self, observation_id: str, token_payload: Dict[str, Any]) -> bool:
        """
        Delete an observation.

        Args:
            observation_id: The observation ID
            token_payload: The decoded JWT token payload

        Returns:
            True if the observation was deleted, False otherwise
        """
        try:
            # Check if the observation exists
            existing_observation = await self.get_observation_by_id(observation_id, token_payload)
            if not existing_observation:
                return False

            # Delete the observation using the FHIR service
            await self.fhir_service.delete(observation_id, token_payload)

            logger.info(f"Deleted observation with ID: {observation_id}")
            return True

        except Exception as e:
            logger.error(f"Error deleting observation with ID {observation_id}: {str(e)}")
            return False

    async def search_observations(self, search_params: Dict[str, Any], token_payload: Dict[str, Any]) -> tuple[List[Dict[str, Any]], int]:
        """
        Search for observations.

        Args:
            search_params: The search parameters, including FHIR search parameters like _count and _offset.
            token_payload: The decoded JWT token payload.

        Returns:
            A tuple containing a list of observation resources (dictionaries) and the total count of matching resources.
        """
        try:
            # Search for observations using the FHIR service
            search_results = await self.fhir_service.search(search_params, token_payload)

            # Extract resources from the bundle
            resources = [
                self._format_observation(entry["resource"])
                for entry in search_results.get("entry", [])
                if "resource" in entry
            ]

            # Extract total count from the bundle
            total_count = search_results.get("total", 0)

            logger.info(f"Search returned {len(resources)} observations out of total {total_count}.")
            return resources, total_count

        except Exception as e:
            logger.error(f"Error searching for observations: {str(e)}")
            raise

    async def get_patient_observations(
        self, patient_id: str, category: Optional[str] = None, code: Optional[str] = None,
        token_payload: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """
        Get all observations for a specific patient with optional filtering.
        
        Args:
            patient_id: The ID of the patient
            category: Optional category to filter by
            code: Optional code to filter by
            token_payload: The decoded JWT token payload
            
        Returns:
            List of observation dictionaries
        """
        try:
            # Build search parameters
            search_params = {'patient': f'Patient/{patient_id}'}
            if category:
                search_params['category'] = category
            if code:
                search_params['code'] = code
                
            # Search observations using the FHIR service
            search_results = await self.fhir_service.search(search_params, token_payload or {})
            
            # Format the response
            observations = [
                self._format_observation(entry["resource"])
                for entry in search_results.get("entry", [])
                if "resource" in entry
            ]
            
            logger.info(f"Found {len(observations)} observations for patient {patient_id}")
            return observations
            
        except Exception as e:
            logger.error(f"Error getting observations for patient {patient_id}: {str(e)}")
            raise

    async def get_observations_by_category(
        self, 
        category: str, 
        patient_id: Optional[str] = None,
        date: Optional[str] = None,
        status: Optional[str] = None,
        page: int = 1,
        count: int = 10,
        token_payload: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        Get observations by category with optional filters.
        
        Args:
            category: The observation category (e.g., 'vital-signs', 'laboratory')
            patient_id: Optional patient ID to filter by
            date: Filter by date (YYYY-MM-DD format)
            status: Filter by status
            page: Page number (1-based)
            count: Number of items per page
            token_payload: The decoded JWT token payload
            
        Returns:
            Dict containing the search results and pagination info
        """
        try:
            # Build search parameters
            search_params = {
                'category': category,
                '_count': str(count),
                '_page': str(page)
            }
            
            if patient_id:
                search_params['patient'] = f'Patient/{patient_id}'
            if date:
                search_params['date'] = date
            if status:
                search_params['status'] = status
                
            # Search observations using the FHIR service
            search_results = await self.fhir_service.search(search_params, token_payload or {})
            
            # Format the response
            observations = [
                self._format_observation(entry["resource"])
                for entry in search_results.get("entry", [])
                if "resource" in entry
            ]
            
            logger.info(f"Found {len(observations)} observations in category '{category}'")
            return {
                "resourceType": "Bundle",
                "type": "searchset",
                "total": search_results.get("total", len(observations)),
                "entry": [{"resource": obs} for obs in observations]
            }
            
        except Exception as e:
            logger.error(f"Error getting observations by category '{category}': {str(e)}")
            raise
        
    async def get_observations_by_code(
        self, 
        code: str, 
        patient_id: Optional[str] = None,
        date: Optional[str] = None,
        status: Optional[str] = None,
        page: int = 1,
        count: int = 10,
        token_payload: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        Get observations by code with optional filters.
        
        Args:
            code: The observation code (LOINC or other coding system)
            patient_id: Optional patient ID to filter by
            date: Filter by date (YYYY-MM-DD format)
            status: Filter by status
            page: Page number (1-based)
            count: Number of items per page
            token_payload: The decoded JWT token payload
            
        Returns:
            Dict containing the search results and pagination info
        """
        try:
            # Build search parameters
            search_params = {
                'code': code,
                '_count': str(count),
                '_page': str(page)
            }
            
            if patient_id:
                search_params['patient'] = f'Patient/{patient_id}'
            if date:
                search_params['date'] = date
            if status:
                search_params['status'] = status
                
            # Search observations using the FHIR service
            search_results = await self.fhir_service.search(search_params, token_payload or {})
            
            # Format the response
            observations = [
                self._format_observation(entry["resource"])
                for entry in search_results.get("entry", [])
                if "resource" in entry
            ]
            
            logger.info(f"Found {len(observations)} observations with code '{code}'")
            return {
                "resourceType": "Bundle",
                "type": "searchset",
                "total": search_results.get("total", len(observations)),
                "entry": [{"resource": obs} for obs in observations]
            }
            
        except Exception as e:
            logger.error(f"Error getting observations with code '{code}': {str(e)}")
            raise

    def to_fhir_observation(self, observation: Dict[str, Any]) -> Dict[str, Any]:
        """
        Convert an observation to a standardized format for the API response.
        
        Args:
            observation: The observation data in FHIR format
            
        Returns:
            The formatted observation data
        """
        try:
            if not observation:
                return {}
                
            # Create a copy of the observation to avoid modifying the original
            formatted = observation.copy()
            
            # Ensure required fields are present
            formatted.setdefault('resourceType', 'Observation')
            formatted.setdefault('status', 'unknown')
            
            # Format the code if present
            if 'code' in formatted and isinstance(formatted['code'], dict):
                if 'coding' in formatted['code'] and isinstance(formatted['code']['coding'], list):
                    # Ensure each coding has the required fields
                    for coding in formatted['code']['coding']:
                        coding.setdefault('system', 'http://loinc.org')
                        coding.setdefault('code', '')
                        coding.setdefault('display', '')
            
            # Format the subject reference if present
            if 'subject' in formatted and isinstance(formatted['subject'], dict):
                if 'reference' in formatted['subject'] and not formatted['subject']['reference'].startswith('Patient/'):
                    formatted['subject']['reference'] = f"Patient/{formatted['subject']['reference']}"
            
            # Format the effective date/time if present
            if 'effectiveDateTime' in formatted and not formatted['effectiveDateTime'].endswith('Z'):
                formatted['effectiveDateTime'] += 'Z'
            
            return formatted
            
        except Exception as e:
            logger.error(f"Error formatting observation: {str(e)}")
            return observation or {}
            
    async def close(self):
        """
        Clean up resources used by the service.
        """
        # No resources to clean up in this implementation
        pass
