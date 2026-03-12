"""
Patient repository for the Patient Service.

This module provides a repository for accessing patient data in MongoDB.
It abstracts the database operations and provides a clean interface for the service layer.
"""

from typing import Dict, List, Optional, Any
import logging
import uuid
from datetime import datetime, timezone

# Import MongoDB connection
from app.db.mongodb import get_patients_collection

# Configure logging
logger = logging.getLogger(__name__)

class PatientRepository:
    """
    Repository for accessing patient data in MongoDB.

    This class provides methods for CRUD operations on patient data,
    abstracting the underlying database implementation.
    """

    def __init__(self):
        """Initialize the repository."""
        self.collection = None
        self._initialized = False

    async def initialize(self) -> bool:
        """
        Initialize the repository.

        This method ensures the MongoDB collection is available
        and initializes any necessary resources.

        Returns:
            True if initialization was successful, False otherwise
        """
        if self._initialized:
            return True

        logger.info("Initializing PatientRepository...")

        # Get the MongoDB collection
        self.collection = get_patients_collection()

        if self.collection is None:
            logger.error("Failed to get patients collection")
            return False

        self._initialized = True
        logger.info("PatientRepository initialized successfully")
        return True

    async def find_by_id(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """
        Find a patient by ID.

        Args:
            patient_id: The patient ID

        Returns:
            The patient if found, None otherwise
        """
        if not await self.initialize():
            logger.error("Repository not initialized")
            return None

        try:
            # Query MongoDB for the patient
            patient = await self.collection.find_one({"id": patient_id})

            if patient:
                # Convert MongoDB _id to string and remove it
                patient["_id"] = str(patient["_id"])

            return patient
        except Exception as e:
            logger.error(f"Error finding patient with ID {patient_id}: {str(e)}")
            return None

    async def find_many(self, query: Dict[str, Any], skip: int = 0, limit: int = 100) -> List[Dict[str, Any]]:
        """
        Find patients matching the query.

        Args:
            query: The query to match
            skip: Number of documents to skip
            limit: Maximum number of documents to return

        Returns:
            A list of matching patients
        """
        if not await self.initialize():
            logger.error("Repository not initialized")
            return []

        try:
            # Build MongoDB query
            mongo_query = self._build_mongo_query(query)

            # Query MongoDB for patients
            cursor = self.collection.find(mongo_query).skip(skip).limit(limit)

            # Convert cursor to list
            patients = []
            async for patient in cursor:
                # Convert MongoDB _id to string
                patient["_id"] = str(patient["_id"])
                patients.append(patient)

            return patients
        except Exception as e:
            logger.error(f"Error finding patients with query {query}: {str(e)}")
            return []

    async def count(self, query: Dict[str, Any]) -> int:
        """
        Count patients matching the query.

        Args:
            query: The query to match

        Returns:
            The number of matching patients
        """
        if not await self.initialize():
            logger.error("Repository not initialized")
            return 0

        try:
            # Build MongoDB query
            mongo_query = self._build_mongo_query(query)

            # Count matching documents
            count = await self.collection.count_documents(mongo_query)
            return count
        except Exception as e:
            logger.error(f"Error counting patients with query {query}: {str(e)}")
            return 0

    async def create(self, patient: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a new patient.

        Args:
            patient: The patient data

        Returns:
            The created patient if successful, None otherwise
        """
        if not await self.initialize():
            logger.error("Repository not initialized")
            return None

        try:
            # Ensure patient has an ID
            if "id" not in patient or not patient["id"]:
                patient["id"] = str(uuid.uuid4())

            # Ensure patient has a resource type
            if "resourceType" not in patient:
                patient["resourceType"] = "Patient"

            # Add metadata
            if "meta" not in patient:
                patient["meta"] = {}

            # Add creation timestamp
            patient["meta"]["lastUpdated"] = datetime.now(timezone.utc).isoformat()

            # Insert patient into MongoDB
            result = await self.collection.insert_one(patient)

            if result.inserted_id:
                # Get the inserted patient
                inserted_patient = await self.find_by_id(patient["id"])
                return inserted_patient
            else:
                logger.error("Failed to insert patient")
                return None
        except Exception as e:
            logger.error(f"Error creating patient: {str(e)}")
            return None

    async def update(self, patient_id: str, patient: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update a patient.

        Args:
            patient_id: The patient ID
            patient: The updated patient data

        Returns:
            The updated patient if successful, None otherwise
        """
        if not await self.initialize():
            logger.error("Repository not initialized")
            return None

        try:
            # Ensure patient has the correct ID
            patient["id"] = patient_id

            # Ensure patient has a resource type
            if "resourceType" not in patient:
                patient["resourceType"] = "Patient"

            # Update metadata
            if "meta" not in patient:
                patient["meta"] = {}

            # Update timestamp
            patient["meta"]["lastUpdated"] = datetime.now(timezone.utc).isoformat()

            # Update patient in MongoDB
            result = await self.collection.replace_one({"id": patient_id}, patient)

            if result.modified_count > 0:
                # Get the updated patient
                updated_patient = await self.find_by_id(patient_id)
                return updated_patient
            else:
                logger.error(f"Failed to update patient with ID {patient_id}")
                return None
        except Exception as e:
            logger.error(f"Error updating patient with ID {patient_id}: {str(e)}")
            return None

    async def delete(self, patient_id: str) -> bool:
        """
        Delete a patient.

        Args:
            patient_id: The patient ID

        Returns:
            True if the patient was deleted, False otherwise
        """
        if not await self.initialize():
            logger.error("Repository not initialized")
            return False

        try:
            # Delete patient from MongoDB
            result = await self.collection.delete_one({"id": patient_id})

            if result.deleted_count > 0:
                logger.info(f"Deleted patient with ID {patient_id}")
                return True
            else:
                logger.error(f"Failed to delete patient with ID {patient_id}")
                return False
        except Exception as e:
            logger.error(f"Error deleting patient with ID {patient_id}: {str(e)}")
            return False

    def _build_mongo_query(self, query: Dict[str, Any]) -> Dict[str, Any]:
        """
        Build a MongoDB query from a FHIR search query.

        Args:
            query: The FHIR search query

        Returns:
            A MongoDB query
        """
        mongo_query = {}

        # Process each search parameter
        for key, value in query.items():
            # Handle special parameters
            if key.startswith("_"):
                continue

            # Handle name search
            if key == "name":
                mongo_query["$or"] = [
                    {"name.family": {"$regex": value, "$options": "i"}},
                    {"name.given": {"$regex": value, "$options": "i"}}
                ]
            # Handle identifier search
            elif key == "identifier":
                mongo_query["identifier.value"] = {"$regex": value, "$options": "i"}
            # Handle gender search
            elif key == "gender":
                mongo_query["gender"] = value
            # Handle birthDate search
            elif key == "birthDate":
                mongo_query["birthDate"] = value
            # Handle active search
            elif key == "active":
                mongo_query["active"] = value.lower() == "true"
            # Handle generalPractitioner search
            elif key == "generalPractitioner":
                mongo_query["generalPractitioner.reference"] = value
            # Handle other parameters
            else:
                mongo_query[key] = value

        return mongo_query

# Singleton instance
_patient_repository_instance = None

async def get_patient_repository():
    """
    Get or create a singleton instance of the PatientRepository.

    Returns:
        A singleton instance of the PatientRepository
    """
    global _patient_repository_instance
    if _patient_repository_instance is None:
        logger.info("Creating new PatientRepository instance")
        _patient_repository_instance = PatientRepository()
        await _patient_repository_instance.initialize()

    return _patient_repository_instance
