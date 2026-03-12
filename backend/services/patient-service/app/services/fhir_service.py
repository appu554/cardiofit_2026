"""
FHIR Service for Patient resources.

This module implements the FHIR service for Patient resources using the shared FHIR models
and MongoDB for data persistence.
"""

import logging
from typing import Dict, List, Any, Optional
import sys
import os
import uuid
try:
    from bson import ObjectId
    BSON_INSTALLED = True
except ImportError:
    BSON_INSTALLED = False
    # Create a placeholder ObjectId class for type checking
    class ObjectId:
        @staticmethod
        def is_valid(oid: str) -> bool:
            """Check if a string is a valid ObjectId."""
            return False

# Import shared FHIR models
from shared.models import Patient as PatientModel
from shared.models import validate_fhir_resource

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR service base class
from services.shared.fhir.service import FHIRServiceBase

# Import the MongoDB connection
from app.db.mongodb import get_patients_collection, Database, connect_to_mongo, db

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def _convert_objectid(obj):
    """
    Convert MongoDB ObjectId to string in a document or list of documents.

    This function recursively traverses dictionaries and lists to convert
    all ObjectId instances to strings, making the document serializable.

    Args:
        obj: The object to convert (can be a dict, list, ObjectId, or other type)

    Returns:
        The converted object with all ObjectId instances replaced with strings
    """
    if obj is None:
        return None

    if isinstance(obj, list):
        return [_convert_objectid(item) for item in obj]

    if isinstance(obj, dict):
        result = {}
        for key, value in obj.items():
            # Convert _id to string if it's an ObjectId
            if key == "_id" and BSON_INSTALLED and isinstance(value, ObjectId):
                result[key] = str(value)
            # Recursively convert nested documents
            elif isinstance(value, (dict, list)):
                result[key] = _convert_objectid(value)
            else:
                result[key] = value
        return result

    # If it's an ObjectId, convert to string
    if BSON_INSTALLED and isinstance(obj, ObjectId):
        return str(obj)

    return obj


# Global instance of the FHIR service
_patient_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service.

    This function creates a global instance of the PatientFHIRService
    and ensures it's properly connected to the database.

    Returns:
        The initialized PatientFHIRService instance
    """
    global _patient_fhir_service

    if _patient_fhir_service is None:
        logger.info("Creating new PatientFHIRService instance...")
        _patient_fhir_service = PatientFHIRService()

    # Log the database status
    logger.info(f"Database status before initialization: {db.get_status()}")

    # Ensure the collection is available
    await _patient_fhir_service.initialize()

    # Log the database status again
    logger.info(f"Database status after initialization: {db.get_status()}")

    # Log the FHIR service status
    if _patient_fhir_service.collection is not None:
        logger.info("FHIR service has a valid MongoDB collection")
    else:
        logger.warning("FHIR service does not have a valid MongoDB collection")

    return _patient_fhir_service

def get_fhir_service():
    """
    Get the global FHIR service instance.

    Returns:
        The global PatientFHIRService instance
    """
    global _patient_fhir_service

    if _patient_fhir_service is None:
        logger.warning("FHIR service not initialized yet. Creating a new instance.")
        _patient_fhir_service = PatientFHIRService()

    return _patient_fhir_service

class PatientFHIRService(FHIRServiceBase):
    """
    FHIR service for Patient resources.

    This service implements the FHIR operations for Patient resources
    using MongoDB for data persistence.
    """

    def __init__(self):
        """Initialize the Patient FHIR service."""
        super().__init__("Patient")
        # Use MongoDB for storing patients
        self.collection = None
        # In-memory fallback storage for when MongoDB is not available
        self.patients = {}
        # Flag to track if the service has been initialized
        self._initialized = False

    async def validate_resource(self, resource: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Validate a Patient resource against the FHIR schema.

        Args:
            resource: The FHIR Patient resource to validate

        Returns:
            The validated resource if validation succeeds, None otherwise
        """
        if not resource:
            return None

        try:
            # Fix common validation issues before validation
            resource = self._fix_validation_issues(resource)

            # Create a Patient model instance from the resource
            patient_model = PatientModel.from_fhir(resource)

            # Validate the resource
            validate_fhir_resource(patient_model)

            # Convert back to dictionary
            validated_resource = patient_model.to_fhir()
            logger.info("Successfully validated Patient resource using shared model")
            return validated_resource
        except Exception as validation_error:
            logger.error(f"Patient validation error: {str(validation_error)}")
            # Return the original resource instead of None to avoid data loss
            return resource

    def _fix_validation_issues(self, resource: Dict[str, Any]) -> Dict[str, Any]:
        """
        Fix common validation issues in a Patient resource.

        Args:
            resource: The FHIR Patient resource to fix

        Returns:
            The fixed resource
        """
        # Make a copy to avoid modifying the original
        fixed_resource = dict(resource)

        # Fix identifier.assigner.reference if missing
        if "identifier" in fixed_resource and fixed_resource["identifier"]:
            for i, identifier in enumerate(fixed_resource["identifier"]):
                if isinstance(identifier, dict) and "assigner" in identifier:
                    assigner = identifier["assigner"]
                    if isinstance(assigner, dict) and "display" in assigner and "reference" not in assigner:
                        # Add a reference field to the assigner
                        fixed_resource["identifier"][i]["assigner"]["reference"] = f"Organization/{uuid.uuid4()}"
                        logger.info(f"Added missing reference to identifier.assigner: {fixed_resource['identifier'][i]['assigner']}")

        return fixed_resource

    async def notify_resource_deleted(self, resource_type: str, resource_id: str) -> bool:
        """
        Notify the FHIR service that a resource has been deleted.

        Args:
            resource_type: The type of resource that was deleted
            resource_id: The ID of the resource that was deleted

        Returns:
            True if the notification was successful, False otherwise
        """
        logger.info(f"Received notification that {resource_type} resource with ID {resource_id} was deleted")
        return True

    async def initialize(self):
        """
        Initialize the service and ensure database connection.

        This method ensures the MongoDB collection is available
        and initializes any necessary resources.
        """
        if self._initialized:
            return

        logger.info("Initializing Patient FHIR service...")

        # Try to connect to MongoDB if not already connected
        if not Database.is_connected():
            logger.info("MongoDB not connected. Attempting to connect...")
            connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

            if connection_success:
                logger.info("Successfully connected to MongoDB during initialization")
                logger.info(f"Database status: {Database.get_status()}")
                logger.info(f"Database initialized: {Database._initialized}")
            else:
                logger.warning("Failed to connect to MongoDB during initialization. Will use in-memory storage.")

        # Try to get the MongoDB collection
        self.collection = get_patients_collection()

        if self.collection is None:
            logger.warning("MongoDB collection not available. Using in-memory storage.")
        else:
            logger.info("MongoDB collection available.")

            # Fix duplicate IDs in the database
            await self._fix_duplicate_ids()

            # Load existing patients into memory for faster access
            try:
                logger.info("Loading patients from MongoDB into memory cache...")
                cursor = self.collection.find({"resourceType": "Patient"})
                count = 0

                async for patient in cursor:
                    # Convert ObjectId to string
                    patient = _convert_objectid(patient)

                    # Cache the patient in memory
                    if "id" in patient:
                        self.patients[patient["id"]] = patient
                        count += 1

                logger.info(f"Loaded {count} patients from MongoDB into memory cache.")
            except Exception as e:
                logger.error(f"Error loading patients from MongoDB: {str(e)}")
                # Try to reconnect to MongoDB
                logger.info("Attempting to reconnect to MongoDB...")
                connection_success = await connect_to_mongo(max_retries=2, retry_delay=1)

                if connection_success:
                    logger.info("Successfully reconnected to MongoDB")
                    # Try to get the collection again
                    self.collection = get_patients_collection()
                    if self.collection is None:
                        logger.warning("MongoDB collection still not available after reconnection")
                else:
                    logger.warning("Failed to reconnect to MongoDB")

        self._initialized = True
        logger.info("Patient FHIR service initialized.")

    async def _fix_duplicate_ids(self):
        """
        Fix duplicate IDs in the database.

        This method finds all patients with duplicate IDs and assigns new IDs to them.
        """
        if self.collection is None:
            logger.warning("Cannot fix duplicate IDs: MongoDB collection not available")
            return

        try:
            logger.info("Checking for duplicate Patient IDs in the database...")

            # Find all patient IDs
            pipeline = [
                {"$match": {"resourceType": "Patient"}},
                {"$group": {"_id": "$id", "count": {"$sum": 1}, "docs": {"$push": "$$ROOT"}}},
                {"$match": {"count": {"$gt": 1}}}
            ]

            cursor = self.collection.aggregate(pipeline)
            duplicate_groups = []
            async for group in cursor:
                duplicate_groups.append(group)

            if not duplicate_groups:
                logger.info("No duplicate Patient IDs found in the database")
                return

            logger.warning(f"Found {len(duplicate_groups)} groups of duplicate Patient IDs")

            # Fix each group of duplicates
            for group in duplicate_groups:
                duplicate_id = group["_id"]
                docs = group["docs"]
                logger.warning(f"Fixing {len(docs)} patients with duplicate ID '{duplicate_id}'")

                # Keep the first document as is, update the rest with new IDs
                for i, doc in enumerate(docs[1:], 1):
                    old_id = doc["id"]
                    new_id = str(uuid.uuid4())

                    # Update the document with a new ID
                    doc["id"] = new_id

                    try:
                        # Remove the old document
                        await self.collection.delete_one({"_id": doc["_id"]})

                        # Insert the updated document
                        doc_copy = dict(doc)
                        if "_id" in doc_copy:
                            del doc_copy["_id"]  # Let MongoDB generate a new _id

                        await self.collection.insert_one(doc_copy)
                        logger.info(f"Updated Patient ID from '{old_id}' to '{new_id}'")
                    except Exception as e:
                        logger.error(f"Error updating Patient with duplicate ID: {str(e)}")

            logger.info("Finished fixing duplicate Patient IDs")
        except Exception as e:
            logger.error(f"Error fixing duplicate Patient IDs: {str(e)}")

    async def _ensure_collection(self):
        """
        Ensure the MongoDB collection is available.

        This method tries to get the MongoDB collection and reconnects
        to MongoDB if necessary.

        Returns:
            bool: True if the collection is available, False otherwise
        """
        if not self._initialized:
            logger.warning("Patient FHIR service not initialized. Call initialize() first.")
            await self.initialize()

        # If collection is None, try to get it again
        if self.collection is None:
            # Try to connect to MongoDB if not already connected
            if not Database.is_connected():
                logger.info("MongoDB not connected. Attempting to connect...")
                connection_success = await connect_to_mongo(max_retries=3, retry_delay=1)

                if connection_success:
                    logger.info("Successfully connected to MongoDB")
                else:
                    logger.warning("Failed to connect to MongoDB")
                    return False

            # Try to get the collection
            self.collection = get_patients_collection()

        # Return True if collection is available, False otherwise
        has_collection = self.collection is not None

        if has_collection:
            logger.info("MongoDB collection is available")
        else:
            logger.warning("MongoDB collection is not available")

        return has_collection

    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """
        Create a new Patient resource.

        Args:
            resource: The FHIR Patient resource to create
            auth_header: Optional authorization header

        Returns:
            The created Patient resource

        Raises:
            HTTPException: If there is an error creating the resource
        """
        try:
            logger.info(f"Creating Patient resource")

            # Generate a resource ID if not provided or ensure it's unique
            if "id" not in resource:
                resource["id"] = str(uuid.uuid4())
            else:
                # Check if a patient with this ID already exists
                if await self._ensure_collection():
                    try:
                        existing_patient = await self.collection.find_one({"id": resource["id"]})
                        if existing_patient:
                            logger.warning(f"Patient with ID {resource['id']} already exists. Generating a new ID.")
                            resource["id"] = str(uuid.uuid4())
                    except Exception as e:
                        logger.error(f"Error checking for existing patient: {str(e)}")

            # Ensure the resource has the correct resource type
            resource["resourceType"] = "Patient"

            # Validate the resource using the shared Patient model
            try:
                # Create a Patient model instance from the resource
                patient_model = PatientModel.from_fhir(resource)

                # Validate the resource
                validate_fhir_resource(patient_model)

                # Convert back to dictionary
                validated_resource = patient_model.to_fhir()
                logger.info("Successfully validated Patient resource using shared model")
            except Exception as validation_error:
                logger.error(f"Patient validation error: {str(validation_error)}")
                # Continue with the original resource if validation fails
                validated_resource = resource
                logger.warning("Using unvalidated Patient resource")

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Make a copy of the resource to avoid modifying the original
                db_resource = dict(validated_resource)

                # Store the patient in MongoDB
                try:
                    result = await self.collection.insert_one(db_resource)

                    # Update the resource with the MongoDB ID
                    if not validated_resource.get("id"):
                        validated_resource["id"] = str(result.inserted_id)

                    logger.info(f"Created Patient resource with ID {validated_resource['id']} in MongoDB")
                except Exception as e:
                    # Check if it's a duplicate key error
                    if "duplicate key error" in str(e).lower():
                        logger.warning(f"Duplicate key error for Patient with ID {validated_resource['id']}. Generating a new ID.")
                        # Generate a new ID
                        validated_resource["id"] = str(uuid.uuid4())
                        db_resource["id"] = validated_resource["id"]

                        try:
                            # Try again with the new ID
                            result = await self.collection.insert_one(db_resource)
                            logger.info(f"Created Patient resource with new ID {validated_resource['id']} in MongoDB")
                        except Exception as retry_error:
                            logger.error(f"Error storing Patient resource with new ID in MongoDB: {str(retry_error)}")
                            # Fallback to in-memory storage
                            self.patients[validated_resource["id"]] = validated_resource
                            logger.info(f"Created Patient resource with ID {validated_resource['id']} in memory (after MongoDB error)")
                    else:
                        logger.error(f"Error storing Patient resource in MongoDB: {str(e)}")
                        # Fallback to in-memory storage
                        self.patients[validated_resource["id"]] = validated_resource
                        logger.info(f"Created Patient resource with ID {validated_resource['id']} in memory (after MongoDB error)")
            else:
                # Fallback to in-memory storage
                self.patients[validated_resource["id"]] = validated_resource
                logger.info(f"Created Patient resource with ID {validated_resource['id']} in memory")

            # Ensure all ObjectId instances are converted to strings
            return _convert_objectid(validated_resource)
        except Exception as e:
            logger.error(f"Error creating Patient resource: {str(e)}")
            # Just raise the error
            raise

    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Get a Patient resource by ID.

        Args:
            resource_id: The ID of the Patient resource to retrieve
            auth_header: Optional authorization header

        Returns:
            The Patient resource if found, None otherwise

        Raises:
            HTTPException: If there is an error retrieving the resource
        """
        try:
            logger.info(f"Getting Patient resource with ID {resource_id}")

            # Check if the patient exists in memory first
            if resource_id in self.patients:
                logger.info(f"Retrieved Patient resource with ID {resource_id} from memory")
                return self.patients[resource_id]

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Try to find the patient in MongoDB
                try:
                    patient = await self.collection.find_one({"id": resource_id})
                except Exception as e:
                    logger.error(f"Error retrieving Patient resource from MongoDB: {str(e)}")
                    return None

                # If not found by id, try to find by MongoDB _id
                if not patient:
                    try:
                        # Try to convert the resource_id to ObjectId
                        if ObjectId.is_valid(resource_id):
                            patient = await self.collection.find_one({"_id": ObjectId(resource_id)})
                    except Exception as e:
                        logger.error(f"Error converting resource_id to ObjectId: {str(e)}")

                # If patient is found, return it
                if patient:
                    # Convert ObjectId to string
                    patient = _convert_objectid(patient)

                    # Try to validate with the shared Patient model
                    try:
                        # Create a Patient model instance from the resource
                        patient_model = PatientModel.from_fhir(patient)

                        # Validate the resource
                        validate_fhir_resource(patient_model)

                        # Convert back to dictionary
                        validated_patient = patient_model.to_fhir()
                        logger.info("Successfully validated Patient resource using shared model")

                        # Cache the validated patient in memory
                        self.patients[resource_id] = validated_patient

                        logger.info(f"Retrieved Patient resource with ID {resource_id} from MongoDB")
                        return validated_patient
                    except Exception as validation_error:
                        logger.error(f"Patient validation error: {str(validation_error)}")
                        # Continue with the original patient if validation fails
                        logger.warning("Using unvalidated Patient resource")

                        # Cache the patient in memory
                        self.patients[resource_id] = patient

                        logger.info(f"Retrieved Patient resource with ID {resource_id} from MongoDB")
                        return patient

            # If no patient is found, return None
            logger.info(f"No patient found with ID {resource_id}")
            return None
        except Exception as e:
            logger.error(f"Error getting Patient resource: {str(e)}")
            # Just raise the error
            raise

    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Update a Patient resource.

        Args:
            resource_id: The ID of the Patient resource to update
            resource: The updated Patient resource
            auth_header: Optional authorization header

        Returns:
            The updated Patient resource

        Raises:
            HTTPException: If there is an error updating the resource
        """
        try:
            logger.info(f"Updating Patient resource with ID {resource_id}")

            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "Patient"

            # Validate the resource using the shared Patient model
            try:
                # Create a Patient model instance from the resource
                patient_model = PatientModel.from_fhir(resource)

                # Validate the resource
                validate_fhir_resource(patient_model)

                # Convert back to dictionary
                validated_resource = patient_model.to_fhir()
                logger.info("Successfully validated Patient resource using shared model")
            except Exception as validation_error:
                logger.error(f"Patient validation error: {str(validation_error)}")
                # Continue with the original resource if validation fails
                validated_resource = resource
                logger.warning("Using unvalidated Patient resource")

            # Always update the in-memory cache
            self.patients[resource_id] = validated_resource

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                try:
                    # Check if the patient exists
                    existing_patient = await self.collection.find_one({"id": resource_id})

                    if existing_patient:
                        # Update the existing patient
                        result = await self.collection.replace_one({"id": resource_id}, validated_resource)
                        logger.info(f"Updated Patient resource with ID {resource_id} in MongoDB, matched count: {result.matched_count}")
                    else:
                        # Create a new patient
                        result = await self.collection.insert_one(validated_resource)
                        logger.info(f"Created new Patient resource with ID {resource_id} in MongoDB")
                except Exception as e:
                    logger.error(f"Error updating Patient resource in MongoDB: {str(e)}")
                    logger.info(f"Updated Patient resource with ID {resource_id} in memory only")
            else:
                logger.info(f"Updated Patient resource with ID {resource_id} in memory only")

            # Ensure all ObjectId instances are converted to strings
            return _convert_objectid(validated_resource)
        except Exception as e:
            logger.error(f"Error updating Patient resource: {str(e)}")
            # Just raise the error
            raise

    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """
        Delete a Patient resource.

        Args:
            resource_id: The ID of the Patient resource to delete
            auth_header: Optional authorization header

        Returns:
            True if the resource was deleted, False otherwise

        Raises:
            HTTPException: If there is an error deleting the resource
        """
        try:
            logger.info(f"Deleting Patient resource with ID {resource_id}")

            # Delete from in-memory cache
            in_memory = resource_id in self.patients
            if in_memory:
                del self.patients[resource_id]
                logger.info(f"Deleted Patient resource with ID {resource_id} from memory")

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                try:
                    # Delete the patient from MongoDB
                    result = await self.collection.delete_one({"id": resource_id})

                    # Check if the patient was deleted
                    if result.deleted_count > 0:
                        logger.info(f"Deleted Patient resource with ID {resource_id} from MongoDB")
                        return True
                    else:
                        logger.info(f"No Patient resource found with ID {resource_id} in MongoDB")
                        # Return True if it was in memory
                        return in_memory
                except Exception as e:
                    logger.error(f"Error deleting Patient resource from MongoDB: {str(e)}")
                    # Return True if it was in memory
                    return in_memory
            else:
                # Return True if it was in memory
                return in_memory
        except Exception as e:
            logger.error(f"Error deleting Patient resource: {str(e)}")
            # Just raise the error
            raise

    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Search for Patient resources.

        Args:
            params: Search parameters
            auth_header: Optional authorization header

        Returns:
            A list of Patient resources matching the search criteria

        Raises:
            HTTPException: If there is an error searching for resources
        """
        try:
            logger.info(f"Searching for Patient resources with parameters: {params}")

            # First, try to search in MongoDB if available
            if await self._ensure_collection():
                # Build the query based on search parameters
                query = {"resourceType": "Patient"}

                # Handle name search parameter
                if "name" in params:
                    name = params["name"]
                    query["$or"] = [
                        {"name.family": {"$regex": name, "$options": "i"}},
                        {"name.given": {"$regex": name, "$options": "i"}}
                    ]

                # Handle gender search parameter
                if "gender" in params:
                    query["gender"] = params["gender"]

                # Handle birthDate search parameter
                if "birthDate" in params:
                    query["birthDate"] = params["birthDate"]

                try:
                    # Execute the query
                    cursor = self.collection.find(query)

                    # Convert cursor to list
                    patients = []
                    async for patient in cursor:
                        # Convert MongoDB ObjectId to string
                        patient = _convert_objectid(patient)

                        # Try to validate with the shared Patient model
                        try:
                            # Create a Patient model instance from the resource
                            patient_model = PatientModel.from_fhir(patient)

                            # Validate the resource
                            validate_fhir_resource(patient_model)

                            # Convert back to dictionary
                            validated_patient = patient_model.to_fhir()
                            patients.append(validated_patient)
                        except Exception as validation_error:
                            logger.error(f"Patient validation error: {str(validation_error)}")
                            # Continue with the original patient if validation fails
                            logger.warning("Using unvalidated Patient resource")
                            patients.append(patient)

                    # If patients found in MongoDB, return them
                    if patients:
                        logger.info(f"Returning {len(patients)} Patient resources from MongoDB")
                        return patients
                except Exception as db_error:
                    logger.error(f"Error searching MongoDB: {str(db_error)}")

            # If MongoDB search failed or no patients found, search in memory
            memory_patients = list(self.patients.values())

            # Filter by search parameters
            filtered_patients = memory_patients

            # Handle name search parameter
            if "name" in params and params["name"]:
                name = params["name"].lower()
                filtered_patients = [
                    p for p in filtered_patients
                    if any(
                        name in n.get("family", "").lower() or
                        any(name in given.lower() for given in n.get("given", []))
                        for n in p.get("name", [])
                    )
                ]

            # Handle gender search parameter
            if "gender" in params and params["gender"]:
                filtered_patients = [
                    p for p in filtered_patients
                    if p.get("gender") == params["gender"]
                ]

            # Handle birthDate search parameter
            if "birthDate" in params and params["birthDate"]:
                filtered_patients = [
                    p for p in filtered_patients
                    if p.get("birthDate") == params["birthDate"]
                ]

            # If patients found in memory, return them
            if filtered_patients:
                logger.info(f"Returning {len(filtered_patients)} Patient resources from memory")
                return filtered_patients

            # If no patients found anywhere, return an empty list
            logger.info("No patients found, returning empty list")
            return []
        except Exception as e:
            logger.error(f"Error searching Patient resources: {str(e)}")
            # Just raise the error
            raise
