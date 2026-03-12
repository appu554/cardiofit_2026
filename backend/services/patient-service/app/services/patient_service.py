"""
Patient service for the Patient Service.

This module provides services for managing patients in the Patient Service.
It implements the service layer pattern, using the repository for data access
and the FHIR service for FHIR validation and integration.
"""

from typing import Dict, Optional, Any, List, Union
import logging
import uuid
import os
import sys
from datetime import datetime, timezone

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import event publishing mixin
try:
    from services.shared.event_publishing.event_publisher_mixin import EventPublisherMixin
except ImportError:
    # Fallback if event publishing not available
    class EventPublisherMixin:
        def initialize_event_publisher(self, *args, **kwargs):
            pass
        async def publish_patient_event(self, *args, **kwargs):
            return None

# Import models
from app.models.patient import PatientCreate, PatientUpdate, PatientResponse, PatientList

# Import FHIR service factory
from app.services.fhir_service_factory import get_fhir_service

# Import repository
from app.repositories.patient_repository import get_patient_repository

# Import settings
from app.core.config import settings

# Configure logging
logger = logging.getLogger(__name__)

# Singleton instance
_patient_service_instance = None

async def get_patient_service():
    """
    Get or create a singleton instance of the Patient service.

    This function ensures that the FHIR service and repository are properly initialized
    before returning the Patient service instance.

    Returns:
        A singleton instance of the PatientService
    """
    global _patient_service_instance
    if _patient_service_instance is None:
        logger.info("Creating new PatientService instance")

        # Initialize the FHIR service
        fhir_service = get_fhir_service()
        await fhir_service.initialize()

        # Initialize the repository only if not using Google Healthcare API
        if settings.USE_GOOGLE_HEALTHCARE_API:
            logger.info("Using Google Healthcare API - skipping MongoDB repository initialization")
            repository = None
        else:
            logger.info("Using MongoDB - initializing repository")
            repository = await get_patient_repository()

        # Create the service instance
        _patient_service_instance = PatientService(fhir_service, repository)
        logger.info(f"PatientService instance created with FHIR service type: {type(fhir_service).__name__}")

    return _patient_service_instance

class PatientService(EventPublisherMixin):
    """
    Service for managing patients.

    This service provides a higher-level API for managing patients,
    using the repository for data access and the FHIR service for validation.
    It implements the service layer pattern to separate business logic from data access.
    Includes event publishing for patient state changes.
    """

    def __init__(self, fhir_service, repository):
        """
        Initialize the service with the FHIR service and repository.

        Args:
            fhir_service: The FHIR service for validation
            repository: The repository for data access (can be None when using Google Healthcare API)
        """
        super().__init__()
        self.fhir_service = fhir_service
        self.repository = repository

        # Initialize event publishing
        self.initialize_event_publisher("patient-service", enabled=True)

        if repository is None and settings.USE_GOOGLE_HEALTHCARE_API:
            logger.info("PatientService initialized with FHIR service only (Google Healthcare API mode)")
        else:
            logger.info("PatientService initialized with FHIR service and repository")

    async def get_patient(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a patient by ID.

        Args:
            patient_id: The patient ID

        Returns:
            The patient if found, None otherwise
        """
        try:
            # If using Google Healthcare API, use FHIR service directly
            if settings.USE_GOOGLE_HEALTHCARE_API:
                logger.info(f"Using Google Healthcare API to get patient {patient_id}")
                return await self.fhir_service.get_resource(patient_id)

            # Otherwise use the repository
            patient = await self.repository.find_by_id(patient_id)

            if patient is None:
                logger.info(f"Patient with ID {patient_id} not found")
                return None

            # Validate the patient against FHIR schema
            validated_patient = await self.fhir_service.validate_resource(patient)

            logger.info(f"Retrieved patient with ID {patient_id}")
            return validated_patient or patient
        except Exception as e:
            logger.error(f"Error getting patient with ID {patient_id}: {str(e)}")
            return None

    async def search_patients(self, params: Dict[str, Any]) -> PatientList:
        """
        Search for patients.

        Args:
            params: Search parameters

        Returns:
            A list of matching patients
        """
        try:
            # Get pagination parameters
            count = int(params.get("_count", 10))
            page = int(params.get("_page", 1))

            # If using Google Healthcare API, use FHIR service directly
            if settings.USE_GOOGLE_HEALTHCARE_API:
                logger.info(f"Using Google Healthcare API to search patients with params: {params}")
                try:
                    # Search for patients using the FHIR service
                    patients = await self.fhir_service.search_resources(params)

                    # For Google Healthcare API, we don't have an easy way to get the total count
                    # So we'll just use the length of the returned list
                    total = len(patients)
                    logger.info(f"Found {total} patients using Google Healthcare API")

                    # Convert to response model
                    patient_responses = []
                    for p in patients:
                        try:
                            # Fix common validation issues in the response model
                            p_data = self._fix_response_model_issues(p)

                            patient_response = PatientResponse(
                                id=p_data.get("id", ""),
                                resourceType=p_data.get("resourceType", "Patient"),
                                identifier=p_data.get("identifier", []),
                                active=p_data.get("active", True),
                                name=p_data.get("name", []),
                                telecom=p_data.get("telecom", []),
                                gender=p_data.get("gender", None),
                                birthDate=p_data.get("birthDate", None),
                                address=p_data.get("address", []),
                                maritalStatus=p_data.get("maritalStatus", None),
                                multipleBirthBoolean=p_data.get("multipleBirthBoolean", None),
                                communication=p_data.get("communication", []),
                                generalPractitioner=p_data.get("generalPractitioner", []),
                                managingOrganization=p_data.get("managingOrganization", None)
                            )
                            patient_responses.append(patient_response)
                        except Exception as e:
                            logger.error(f"Error converting patient to response model: {str(e)}")

                    return PatientList(
                        patients=patient_responses,
                        total=total,
                        page=page,
                        count=count
                    )
                except Exception as e:
                    logger.error(f"Error searching patients with Google Healthcare API: {str(e)}")
                    return PatientList(
                        patients=[],
                        total=0,
                        page=page,
                        count=count
                    )

            # Otherwise use the repository
            skip = (page - 1) * count

            # Create a copy of the params for repository search
            search_params = dict(params)

            # Remove pagination parameters from search
            if "_count" in search_params:
                del search_params["_count"]
            if "_page" in search_params:
                del search_params["_page"]

            # Search for patients using the repository
            patients = await self.repository.find_many(search_params, skip=skip, limit=count)

            # Get total count
            total = await self.repository.count(search_params)

            # Convert to response model
            patient_responses = []
            for p in patients:
                try:
                    # Validate the patient against FHIR schema
                    validated_patient = await self.fhir_service.validate_resource(p)
                    p_data = validated_patient or p

                    # Fix common validation issues in the response model
                    p_data = self._fix_response_model_issues(p_data)

                    patient_response = PatientResponse(
                        id=p_data.get("id", ""),
                        resourceType=p_data.get("resourceType", "Patient"),
                        identifier=p_data.get("identifier", []),
                        active=p_data.get("active", True),
                        name=p_data.get("name", []),
                        telecom=p_data.get("telecom", []),
                        gender=p_data.get("gender", None),
                        birthDate=p_data.get("birthDate", None),
                        address=p_data.get("address", []),
                        maritalStatus=p_data.get("maritalStatus", None),
                        multipleBirthBoolean=p_data.get("multipleBirthBoolean", None),
                        communication=p_data.get("communication", []),
                        generalPractitioner=p_data.get("generalPractitioner", []),
                        managingOrganization=p_data.get("managingOrganization", None)
                    )
                    patient_responses.append(patient_response)
                except Exception as e:
                    logger.error(f"Error converting patient to response model: {str(e)}")
                    # Try a simplified version without validation
                    try:
                        patient_response = PatientResponse(
                            id=p.get("id", ""),
                            resourceType=p.get("resourceType", "Patient"),
                            name=p.get("name", []),
                            gender=p.get("gender", None),
                            birthDate=p.get("birthDate", None),
                            active=p.get("active", True),
                            generalPractitioner=p.get("generalPractitioner", []),
                            managingOrganization=p.get("managingOrganization", None),
                            communication=p.get("communication", [])
                        )
                        patient_responses.append(patient_response)
                        logger.info(f"Added simplified patient response for ID {p.get('id', 'unknown')}")
                    except Exception as e2:
                        logger.error(f"Error creating simplified patient response: {str(e2)}")

            logger.info(f"Found {total} patients, returning {len(patient_responses)} (page {page}, count {count})")

            return PatientList(
                patients=patient_responses,
                total=total,
                page=page,
                count=count
            )
        except Exception as e:
            logger.error(f"Error searching for patients: {str(e)}")
            # Return an empty list in case of error
            return PatientList(
                patients=[],
                total=0,
                page=page,
                count=count
            )

    async def create_patient(self, patient_data: Union[PatientCreate, Dict[str, Any]]) -> Dict[str, Any]:
        """
        Create a new patient.

        Args:
            patient_data: The patient data (either a PatientCreate model or a dictionary)

        Returns:
            The created patient
        """
        try:
            # Convert input to a dictionary
            if hasattr(patient_data, 'model_dump'):
                # It's a Pydantic model
                patient_dict = patient_data.model_dump(exclude_unset=True)
            else:
                # It's already a dictionary
                patient_dict = dict(patient_data)

            # Add required FHIR fields
            patient_dict["resourceType"] = "Patient"

            # Generate a UUID if not provided
            if "id" not in patient_dict or not patient_dict["id"]:
                patient_dict["id"] = str(uuid.uuid4())

            # Add metadata with timestamp
            if "meta" not in patient_dict:
                patient_dict["meta"] = {}
            patient_dict["meta"]["lastUpdated"] = datetime.now(timezone.utc).isoformat()

            # Convert snake_case keys to camelCase for FHIR
            patient_dict = self._convert_keys_to_camel_case(patient_dict)

            # Validate the patient against FHIR schema
            validated_patient = await self.fhir_service.validate_resource(patient_dict)

            if validated_patient:
                # Use the validated patient data
                patient_dict = validated_patient

            # If using Google Healthcare API, use FHIR service directly
            if settings.USE_GOOGLE_HEALTHCARE_API:
                logger.info(f"Using Google Healthcare API to create patient")
                created_patient = await self.fhir_service.create_resource(patient_dict)
                if created_patient:
                    # Publish patient created event
                    await self.publish_patient_event(
                        patient_id=created_patient.get('id'),
                        operation="created",
                        patient_data=created_patient
                    )

                    logger.info(f"Created patient with ID {created_patient.get('id')} using Google Healthcare API")
                    return created_patient
                else:
                    logger.error("Failed to create patient using Google Healthcare API")
                    raise ValueError("Failed to create patient using Google Healthcare API")
            else:
                # Create the patient using the repository
                created_patient = await self.repository.create(patient_dict)

                if created_patient:
                    # Publish patient created event
                    await self.publish_patient_event(
                        patient_id=created_patient.get('id'),
                        operation="created",
                        patient_data=created_patient
                    )

                    logger.info(f"Created patient with ID {created_patient.get('id')}")
                    return created_patient
                else:
                    logger.error("Failed to create patient")
                    raise ValueError("Failed to create patient")
        except Exception as e:
            logger.error(f"Error creating patient: {str(e)}")
            raise

    async def update_patient(self, patient_id: str, patient_data: Union[PatientUpdate, Dict[str, Any]]) -> Optional[Dict[str, Any]]:
        """
        Update a patient.

        Args:
            patient_id: The patient ID
            patient_data: The patient data to update (either a PatientUpdate model or a dictionary)

        Returns:
            The updated patient if found, None otherwise
        """
        try:
            # If using Google Healthcare API, get the patient directly from the API
            if settings.USE_GOOGLE_HEALTHCARE_API:
                logger.info(f"Using Google Healthcare API to update patient {patient_id}")

                # Get existing patient from Google Healthcare API
                existing_patient = await self.fhir_service.get_resource(patient_id)
                if not existing_patient:
                    logger.warning(f"Patient with ID {patient_id} not found in Google Healthcare API")
                    return None

                # Convert input to a dictionary
                if hasattr(patient_data, 'model_dump'):
                    # It's a Pydantic model
                    update_data = patient_data.model_dump(exclude_unset=True)
                else:
                    # It's already a dictionary
                    update_data = dict(patient_data)

                # Merge with existing patient data
                merged_patient = {**existing_patient, **update_data}

                # Ensure required FHIR fields
                merged_patient["id"] = patient_id
                merged_patient["resourceType"] = "Patient"

                # Update the timestamp
                if "meta" not in merged_patient:
                    merged_patient["meta"] = {}
                merged_patient["meta"]["lastUpdated"] = datetime.now(timezone.utc).isoformat()

                # Convert snake_case keys to camelCase for FHIR
                merged_patient = self._convert_keys_to_camel_case(merged_patient)

                # Validate the patient against FHIR schema
                validated_patient = await self.fhir_service.validate_resource(merged_patient)

                if validated_patient:
                    # Use the validated patient data
                    merged_patient = validated_patient

                # Update the patient using the FHIR service
                updated_patient = await self.fhir_service.update_resource(patient_id, merged_patient)

                if updated_patient:
                    # Publish patient updated event
                    await self.publish_patient_event(
                        patient_id=patient_id,
                        operation="updated",
                        patient_data=updated_patient
                    )

                    logger.info(f"Updated patient with ID {patient_id} using Google Healthcare API")
                    return updated_patient
                else:
                    logger.error(f"Failed to update patient with ID {patient_id} using Google Healthcare API")
                    return None
            else:
                # Check if patient exists in repository
                existing_patient = await self.repository.find_by_id(patient_id)
                if not existing_patient:
                    logger.warning(f"Patient with ID {patient_id} not found for update")
                    return None

                # Convert input to a dictionary
                if hasattr(patient_data, 'model_dump'):
                    # It's a Pydantic model
                    update_data = patient_data.model_dump(exclude_unset=True)
                else:
                    # It's already a dictionary
                    update_data = dict(patient_data)

                # Merge with existing patient data
                merged_patient = {**existing_patient, **update_data}

                # Ensure required FHIR fields
                merged_patient["id"] = patient_id
                merged_patient["resourceType"] = "Patient"

                # Update the timestamp
                if "meta" not in merged_patient:
                    merged_patient["meta"] = {}
                merged_patient["meta"]["lastUpdated"] = datetime.now(timezone.utc).isoformat()

                # Convert snake_case keys to camelCase for FHIR
                merged_patient = self._convert_keys_to_camel_case(merged_patient)

                # Validate the patient against FHIR schema
                validated_patient = await self.fhir_service.validate_resource(merged_patient)

                if validated_patient:
                    # Use the validated patient data
                    merged_patient = validated_patient

                # Update the patient using the repository
                updated_patient = await self.repository.update(patient_id, merged_patient)

                if updated_patient:
                    # Publish patient updated event
                    await self.publish_patient_event(
                        patient_id=patient_id,
                        operation="updated",
                        patient_data=updated_patient
                    )

                    logger.info(f"Updated patient with ID {patient_id}")
                    return updated_patient
                else:
                    logger.error(f"Failed to update patient with ID {patient_id}")
                    return None
        except Exception as e:
            logger.error(f"Error updating patient with ID {patient_id}: {str(e)}")
            return None

    def _fix_response_model_issues(self, patient_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Fix common issues in patient data before converting to response model.

        Args:
            patient_data: The patient data to fix

        Returns:
            The fixed patient data
        """
        # Make a copy to avoid modifying the original
        fixed_data = dict(patient_data)

        # Fix identifier.assigner.reference if missing
        if "identifier" in fixed_data and fixed_data["identifier"]:
            for i, identifier in enumerate(fixed_data["identifier"]):
                if isinstance(identifier, dict) and "assigner" in identifier:
                    assigner = identifier["assigner"]
                    if isinstance(assigner, dict) and "display" in assigner and "reference" not in assigner:
                        # Add a reference field to the assigner
                        fixed_data["identifier"][i]["assigner"]["reference"] = f"Organization/{uuid.uuid4()}"
                        logger.info(f"Added missing reference to identifier.assigner in response model")

        # Ensure name is a list
        if "name" in fixed_data and not isinstance(fixed_data["name"], list):
            fixed_data["name"] = [fixed_data["name"]] if fixed_data["name"] else []

        # Ensure telecom is a list
        if "telecom" in fixed_data and not isinstance(fixed_data["telecom"], list):
            fixed_data["telecom"] = [fixed_data["telecom"]] if fixed_data["telecom"] else []

        # Ensure address is a list
        if "address" in fixed_data and not isinstance(fixed_data["address"], list):
            fixed_data["address"] = [fixed_data["address"]] if fixed_data["address"] else []

        # Ensure identifier is a list
        if "identifier" in fixed_data and not isinstance(fixed_data["identifier"], list):
            fixed_data["identifier"] = [fixed_data["identifier"]] if fixed_data["identifier"] else []

        # Ensure communication is a list
        if "communication" in fixed_data and not isinstance(fixed_data["communication"], list):
            fixed_data["communication"] = [fixed_data["communication"]] if fixed_data["communication"] else []

        # Ensure generalPractitioner is a list
        if "generalPractitioner" in fixed_data and not isinstance(fixed_data["generalPractitioner"], list):
            fixed_data["generalPractitioner"] = [fixed_data["generalPractitioner"]] if fixed_data["generalPractitioner"] else []

        # Ensure managingOrganization has reference if display is present
        if "managingOrganization" in fixed_data and fixed_data["managingOrganization"]:
            org = fixed_data["managingOrganization"]
            if isinstance(org, dict) and "display" in org and "reference" not in org:
                fixed_data["managingOrganization"]["reference"] = f"Organization/{uuid.uuid4()}"
                logger.info(f"Added missing reference to managingOrganization in response model")

        return fixed_data

    def _convert_keys_to_camel_case(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Convert dictionary keys from snake_case to camelCase for FHIR compatibility.

        Args:
            data: The dictionary with snake_case keys

        Returns:
            A new dictionary with camelCase keys
        """
        if not isinstance(data, dict):
            return data

        result = {}
        for key, value in data.items():
            # Skip keys that are already camelCase or should not be converted
            if key in ["resourceType", "id", "meta"]:
                new_key = key
            else:
                # Convert snake_case to camelCase
                parts = key.split('_')
                new_key = parts[0] + ''.join(x.title() for x in parts[1:])

            # Recursively convert nested dictionaries and lists
            if isinstance(value, dict):
                result[new_key] = self._convert_keys_to_camel_case(value)
            elif isinstance(value, list):
                result[new_key] = [
                    self._convert_keys_to_camel_case(item) if isinstance(item, dict) else item
                    for item in value
                ]
            else:
                result[new_key] = value

        return result

    async def delete_patient(self, patient_id: str) -> bool:
        """
        Delete a patient.

        Args:
            patient_id: The patient ID

        Returns:
            True if the patient was deleted, False otherwise
        """
        try:
            # If using Google Healthcare API, delete directly from the API
            if settings.USE_GOOGLE_HEALTHCARE_API:
                logger.info(f"Using Google Healthcare API to delete patient {patient_id}")

                # Check if patient exists in Google Healthcare API
                existing_patient = await self.fhir_service.get_resource(patient_id)
                if not existing_patient:
                    logger.warning(f"Patient with ID {patient_id} not found in Google Healthcare API")
                    return False

                # Delete the patient using the FHIR service
                deleted = await self.fhir_service.delete_resource(patient_id)

                if deleted:
                    logger.info(f"Deleted patient with ID {patient_id} using Google Healthcare API")
                    return True
                else:
                    logger.error(f"Failed to delete patient with ID {patient_id} using Google Healthcare API")
                    return False
            else:
                # Check if patient exists in repository
                existing_patient = await self.repository.find_by_id(patient_id)
                if not existing_patient:
                    logger.warning(f"Patient with ID {patient_id} not found for deletion")
                    return False

                # Delete the patient using the repository
                deleted = await self.repository.delete(patient_id)

                if deleted:
                    logger.info(f"Deleted patient with ID {patient_id}")
                    # Notify the FHIR service about the deletion
                    await self.fhir_service.notify_resource_deleted("Patient", patient_id)
                    return True
                else:
                    logger.error(f"Failed to delete patient with ID {patient_id}")
                    return False
        except Exception as e:
            logger.error(f"Error deleting patient with ID {patient_id}: {str(e)}")
            return False
