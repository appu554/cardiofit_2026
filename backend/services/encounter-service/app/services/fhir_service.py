"""
FHIR Service for Encounter resources.

This module implements the FHIR service for Encounter resources using MongoDB for data persistence.
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

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR service base class
from services.shared.fhir.service import FHIRServiceBase

# Import the MongoDB connection
from app.db.mongodb import get_encounters_collection, connect_to_mongo, db

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
_encounter_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service.

    This function creates a global instance of the EncounterFHIRService
    and ensures it's properly connected to the database.

    Returns:
        The initialized EncounterFHIRService instance
    """
    global _encounter_fhir_service

    # Log initialization start
    logger.info("Starting FHIR service initialization...")

    # Create a new service instance if needed
    if _encounter_fhir_service is None:
        _encounter_fhir_service = EncounterFHIRService()

    # Force a fresh connection to MongoDB
    db._initialized = False
    db._status = "not_connected"
    db.client = None
    db.db = None

    # Connect with increased retries
    connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

    if not connection_success:
        logger.error("Failed to connect to MongoDB. Will use in-memory storage.")

    # Initialize the service
    await _encounter_fhir_service.initialize()

    # Log initialization completion
    logger.info("FHIR service initialization complete.")

    return _encounter_fhir_service

def get_fhir_service():
    """
    Get the global FHIR service instance.

    Returns:
        The global EncounterFHIRService instance
    """
    global _encounter_fhir_service

    if _encounter_fhir_service is None:
        logger.warning("FHIR service not initialized yet. Creating a new instance.")
        _encounter_fhir_service = EncounterFHIRService()

    return _encounter_fhir_service

class EncounterFHIRService(FHIRServiceBase):
    """
    FHIR service for Encounter resources.

    This service implements the FHIR operations for Encounter resources
    using MongoDB for data persistence.
    """

    def __init__(self):
        """Initialize the Encounter FHIR service."""
        super().__init__("Encounter")
        # Use MongoDB for storing encounters
        self.collection = None
        # In-memory fallback storage for when MongoDB is not available
        self.encounters = {}
        # Flag to track if the service has been initialized
        self._initialized = False

    async def initialize(self):
        """
        Initialize the service and ensure database connection.

        This method ensures the MongoDB collection is available
        and initializes any necessary resources.
        """
        if self._initialized:
            logger.info("Encounter FHIR service already initialized, skipping initialization")
            return

        logger.info("Initializing Encounter FHIR service...")

        # Force a fresh connection to MongoDB
        logger.info("Forcing a fresh connection to MongoDB...")
        # Reset the database state
        db._initialized = False
        db._status = "not_connected"
        db.client = None
        db.db = None

        # Connect with increased retries
        connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

        if connection_success:
            logger.info("Successfully connected to MongoDB during initialization")
            logger.info(f"Database status: {db.get_status()}")
            logger.info(f"Database initialized: {db._initialized}")
            logger.info(f"Database client exists: {db.client is not None}")
            logger.info(f"Database object exists: {db.db is not None}")
        else:
            logger.warning("Failed to connect to MongoDB during initialization. Will use in-memory storage.")
            self._initialized = True
            logger.info("Encounter FHIR service initialized with in-memory storage only.")
            return

        # Try to get the MongoDB collection
        self.collection = get_encounters_collection()

        if self.collection is None:
            logger.warning("MongoDB collection not available. Trying to create it directly...")

            try:
                # Check if collection already exists
                collections = await db.db.list_collection_names()
                logger.info(f"Available collections: {collections}")

                if "encounters" not in collections:
                    logger.info("Creating encounters collection...")
                    await db.db.create_collection("encounters")
                    logger.info("Successfully created encounters collection")
                else:
                    logger.info("Encounters collection already exists")

                # Try to get the collection directly
                self.collection = db.db.encounters
                if self.collection is not None:
                    logger.info("Successfully retrieved encounters collection directly")
                else:
                    logger.error("Failed to retrieve encounters collection directly")
            except Exception as e:
                logger.error(f"Error creating/accessing encounters collection: {str(e)}")
                import traceback
                logger.error(traceback.format_exc())
                logger.warning("Will use in-memory storage.")
        else:
            logger.info("MongoDB collection available.")

            # Load existing encounters into memory for faster access
            try:
                logger.info("Loading encounters from MongoDB into memory cache...")
                cursor = self.collection.find({"resourceType": "Encounter"})
                count = 0

                async for encounter in cursor:
                    # Convert ObjectId to string
                    encounter = _convert_objectid(encounter)

                    # Cache the encounter in memory
                    if "id" in encounter:
                        self.encounters[encounter["id"]] = encounter
                        count += 1

                logger.info(f"Loaded {count} encounters from MongoDB into memory cache.")
            except Exception as e:
                logger.error(f"Error loading encounters from MongoDB: {str(e)}")
                import traceback
                logger.error(traceback.format_exc())

                # Try one more time with a direct approach
                try:
                    logger.info("Trying direct approach to load encounters...")
                    # Get all documents in the collection
                    cursor = db.db.encounters.find({})
                    count = 0

                    async for doc in cursor:
                        # Convert ObjectId to string
                        doc = _convert_objectid(doc)

                        # Cache the encounter in memory
                        if "id" in doc:
                            self.encounters[doc["id"]] = doc
                            count += 1

                    logger.info(f"Loaded {count} encounters using direct approach.")
                except Exception as direct_error:
                    logger.error(f"Error with direct approach: {str(direct_error)}")
                    logger.error(traceback.format_exc())

        self._initialized = True
        logger.info("Encounter FHIR service initialized.")
        logger.info(f"Using MongoDB collection: {self.collection is not None}")
        logger.info(f"In-memory cache size: {len(self.encounters)}")

    async def _ensure_collection(self):
        """
        Ensure the MongoDB collection is available.

        This method tries to get the MongoDB collection and reconnects
        to MongoDB if necessary.

        Returns:
            bool: True if the collection is available, False otherwise
        """
        if not self._initialized:
            logger.warning("Encounter FHIR service not initialized. Call initialize() first.")
            await self.initialize()
            # After initialization, check if we have a collection
            return self.collection is not None

        # If collection is None, try to get it again
        if self.collection is None:
            logger.warning("Collection is None, attempting to reconnect to MongoDB")

            # Force a fresh connection to MongoDB
            logger.info("Forcing a fresh connection to MongoDB...")
            # Reset the database state
            db._initialized = False
            db._status = "not_connected"
            db.client = None
            db.db = None

            # Connect with increased retries
            connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

            if connection_success:
                logger.info("Successfully connected to MongoDB with fresh connection")
                logger.info(f"Database status: {db.get_status()}")
                logger.info(f"Database initialized: {db._initialized}")
                logger.info(f"Database client exists: {db.client is not None}")
                logger.info(f"Database object exists: {db.db is not None}")
            else:
                logger.error("Failed to connect to MongoDB with fresh connection")
                return False

            # Try to get the collection
            self.collection = get_encounters_collection()

            # If still None, try to create the collection directly
            if self.collection is None and db.is_connected() and db.db is not None:
                try:
                    logger.info("Attempting to create encounters collection directly...")
                    # Check if collection already exists
                    collections = await db.db.list_collection_names()
                    logger.info(f"Available collections: {collections}")

                    if "encounters" not in collections:
                        await db.db.create_collection("encounters")
                        logger.info("Successfully created encounters collection")
                    else:
                        logger.info("Encounters collection already exists")

                    # Try to get the collection directly
                    self.collection = db.db.encounters
                    if self.collection is not None:
                        logger.info("Successfully retrieved encounters collection directly")
                    else:
                        logger.error("Failed to retrieve encounters collection directly")
                except Exception as e:
                    logger.error(f"Error creating/accessing encounters collection: {str(e)}")
                    import traceback
                    logger.error(traceback.format_exc())

        # Return True if collection is available, False otherwise
        has_collection = self.collection is not None

        if has_collection:
            logger.info("MongoDB collection is available")
        else:
            logger.warning("MongoDB collection is not available")

        return has_collection

    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """
        Create a new Encounter resource.

        Args:
            resource: The FHIR Encounter resource to create
            auth_header: Optional authorization header

        Returns:
            The created Encounter resource

        Raises:
            HTTPException: If there is an error creating the resource
        """
        try:
            logger.info(f"Creating Encounter resource with ID: {resource.get('id', 'new')}")

            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = str(uuid.uuid4())
            else:
                # Check if an encounter with this ID already exists
                if await self._ensure_collection():
                    try:
                        existing_encounter = await self.collection.find_one({"id": resource["id"]})
                        if existing_encounter:
                            logger.warning(f"Encounter with ID {resource['id']} already exists. Generating a new ID.")
                            resource["id"] = str(uuid.uuid4())
                    except Exception as e:
                        logger.error(f"Error checking for existing encounter: {str(e)}")

            # Ensure the resource has the correct resource type
            resource["resourceType"] = "Encounter"

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Make a copy of the resource to avoid modifying the original
                db_resource = dict(resource)

                # Store the encounter in MongoDB
                try:
                    result = await self.collection.insert_one(db_resource)

                    # Update the resource with the MongoDB ID
                    if not resource.get("id"):
                        resource["id"] = str(result.inserted_id)

                    logger.info(f"Created Encounter resource with ID {resource['id']} in MongoDB")
                except Exception as e:
                    # Check if it's a duplicate key error
                    if "duplicate key error" in str(e).lower():
                        logger.warning(f"Duplicate key error for Encounter with ID {resource['id']}. Generating a new ID.")
                        # Generate a new ID
                        resource["id"] = str(uuid.uuid4())
                        db_resource["id"] = resource["id"]

                        try:
                            # Try again with the new ID
                            result = await self.collection.insert_one(db_resource)
                            logger.info(f"Created Encounter resource with new ID {resource['id']} in MongoDB")
                        except Exception as retry_error:
                            logger.error(f"Error storing Encounter resource with new ID in MongoDB: {str(retry_error)}")
                            # Fallback to in-memory storage
                            self.encounters[resource["id"]] = resource
                            logger.info(f"Created Encounter resource with ID {resource['id']} in memory (after MongoDB error)")
                    else:
                        logger.error(f"Error storing Encounter resource in MongoDB: {str(e)}")
                        # Fallback to in-memory storage
                        self.encounters[resource["id"]] = resource
                        logger.info(f"Created Encounter resource with ID {resource['id']} in memory (after MongoDB error)")
            else:
                # Fallback to in-memory storage
                self.encounters[resource["id"]] = resource
                logger.info(f"Created Encounter resource with ID {resource['id']} in memory")

            # Ensure all ObjectId instances are converted to strings
            return _convert_objectid(resource)
        except Exception as e:
            logger.error(f"Error creating Encounter resource: {str(e)}")
            # Just raise the error
            raise

    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Get an Encounter resource by ID.

        Args:
            resource_id: The ID of the Encounter resource to retrieve
            auth_header: Optional authorization header

        Returns:
            The Encounter resource if found, None otherwise

        Raises:
            HTTPException: If there is an error retrieving the resource
        """
        try:
            logger.info(f"Getting Encounter resource with ID {resource_id}")

            # Check if the encounter exists in memory first
            if resource_id in self.encounters:
                logger.info(f"Retrieved Encounter resource with ID {resource_id} from memory")
                return self.encounters[resource_id]

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Try to find the encounter in MongoDB
                try:
                    encounter = await self.collection.find_one({"id": resource_id})
                except Exception as e:
                    logger.error(f"Error retrieving Encounter resource from MongoDB: {str(e)}")
                    return None

                # If not found by id, try to find by MongoDB _id
                if not encounter:
                    try:
                        # Try to convert the resource_id to ObjectId
                        if ObjectId.is_valid(resource_id):
                            encounter = await self.collection.find_one({"_id": ObjectId(resource_id)})
                    except Exception as e:
                        logger.error(f"Error converting resource_id to ObjectId: {str(e)}")

                # If encounter is found, return it
                if encounter:
                    # Convert ObjectId to string
                    encounter = _convert_objectid(encounter)

                    # Cache the encounter in memory
                    self.encounters[resource_id] = encounter

                    logger.info(f"Retrieved Encounter resource with ID {resource_id} from MongoDB")
                    return encounter

            # If no encounter is found, return None
            logger.info(f"No encounter found with ID {resource_id}")
            return None
        except Exception as e:
            logger.error(f"Error getting Encounter resource: {str(e)}")
            # Just raise the error
            raise

    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Update an Encounter resource.

        Args:
            resource_id: The ID of the Encounter resource to update
            resource: The updated Encounter resource
            auth_header: Optional authorization header

        Returns:
            The updated Encounter resource

        Raises:
            HTTPException: If there is an error updating the resource
        """
        try:
            logger.info(f"Updating Encounter resource with ID {resource_id}")

            # Ensure the resource has the correct resource type
            resource["resourceType"] = "Encounter"

            # Ensure the resource ID matches the requested ID
            resource["id"] = resource_id

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Make a copy of the resource to avoid modifying the original
                db_resource = dict(resource)

                # Update the encounter in MongoDB
                try:
                    # Try to update by id field
                    result = await self.collection.replace_one(
                        {"id": resource_id},
                        db_resource,
                        upsert=True
                    )

                    if result.modified_count > 0 or result.upserted_id:
                        logger.info(f"Updated Encounter resource with ID {resource_id} in MongoDB")
                    else:
                        logger.warning(f"No Encounter resource with ID {resource_id} found in MongoDB to update")
                except Exception as e:
                    logger.error(f"Error updating Encounter resource in MongoDB: {str(e)}")
                    # Fallback to in-memory storage
                    self.encounters[resource_id] = resource
                    logger.info(f"Updated Encounter resource with ID {resource_id} in memory (after MongoDB error)")
            else:
                # Fallback to in-memory storage
                self.encounters[resource_id] = resource
                logger.info(f"Updated Encounter resource with ID {resource_id} in memory")

            # Update the in-memory cache
            self.encounters[resource_id] = resource

            # Ensure all ObjectId instances are converted to strings
            return _convert_objectid(resource)
        except Exception as e:
            logger.error(f"Error updating Encounter resource: {str(e)}")
            # Just raise the error
            raise

    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """
        Delete an Encounter resource.

        Args:
            resource_id: The ID of the Encounter resource to delete
            auth_header: Optional authorization header

        Returns:
            True if the resource was deleted, False otherwise

        Raises:
            HTTPException: If there is an error deleting the resource
        """
        try:
            logger.info(f"Deleting Encounter resource with ID {resource_id}")

            # Track if the resource was deleted
            deleted = False

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Try to delete from MongoDB
                try:
                    # Try to delete by id field
                    result = await self.collection.delete_one({"id": resource_id})

                    if result.deleted_count > 0:
                        logger.info(f"Deleted Encounter resource with ID {resource_id} from MongoDB")
                        deleted = True
                    else:
                        logger.warning(f"No Encounter resource with ID {resource_id} found in MongoDB to delete")
                except Exception as e:
                    logger.error(f"Error deleting Encounter resource from MongoDB: {str(e)}")

            # Check if the encounter exists in memory
            if resource_id in self.encounters:
                # Delete from memory
                del self.encounters[resource_id]
                logger.info(f"Deleted Encounter resource with ID {resource_id} from memory")
                deleted = True

            return deleted
        except Exception as e:
            logger.error(f"Error deleting Encounter resource: {str(e)}")
            # Just raise the error
            raise

    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Search for Encounter resources.

        Args:
            params: The search parameters
            auth_header: Optional authorization header

        Returns:
            A list of Encounter resources matching the search parameters

        Raises:
            HTTPException: If there is an error searching for resources
        """
        try:
            logger.info(f"Searching for Encounter resources with params: {params}")

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                try:
                    # Build the query based on the search parameters
                    query = {"resourceType": "Encounter"}

                    # Extract pagination parameters
                    pagination_params = {}
                    if "_count" in params:
                        try:
                            pagination_params["limit"] = int(params["_count"])
                        except (ValueError, TypeError):
                            pagination_params["limit"] = 100
                    else:
                        pagination_params["limit"] = 100

                    if "_page" in params:
                        try:
                            pagination_params["skip"] = (int(params["_page"]) - 1) * pagination_params["limit"]
                        except (ValueError, TypeError):
                            pagination_params["skip"] = 0
                    else:
                        pagination_params["skip"] = 0

                    # Add search parameters to the query (excluding pagination)
                    for key, value in params.items():
                        # Skip pagination parameters
                        if key in ["_count", "_page"]:
                            continue

                        # Handle special parameters
                        if key == "_id":
                            query["id"] = value
                        elif key == "patient":
                            # Handle patient reference search
                            query["subject.reference"] = f"Patient/{value}"
                        elif key == "date":
                            # Handle date search
                            query["period.start"] = {"$lte": value}
                            query["period.end"] = {"$gte": value}
                        elif key == "subject":
                            # Handle subject reference
                            query["subject.reference"] = value
                            # Log for debugging
                            logger.info(f"Adding subject search parameter: {value}")
                        else:
                            # Handle other parameters
                            query[key] = value

                    # Log the final query for debugging
                    logger.info(f"MongoDB query: {query}")
                    logger.info(f"Pagination: limit={pagination_params['limit']}, skip={pagination_params['skip']}")

                    # Execute the query with pagination
                    cursor = self.collection.find(query).skip(pagination_params["skip"]).limit(pagination_params["limit"])

                    # Convert the results to a list
                    results = []
                    async for encounter in cursor:
                        # Convert ObjectId to string
                        encounter = _convert_objectid(encounter)

                        # Add to results
                        results.append(encounter)

                        # Cache in memory
                        if "id" in encounter:
                            self.encounters[encounter["id"]] = encounter

                    # Log the results
                    logger.info(f"Found {len(results)} Encounter resources in MongoDB")

                    # If no results found, log the document structure for debugging
                    if len(results) == 0:
                        try:
                            # Get a sample document to check structure
                            sample = await self.collection.find_one({"resourceType": "Encounter"})
                            if sample:
                                sample = _convert_objectid(sample)
                                logger.info(f"Sample encounter document structure: {sample}")

                                # Check if the subject reference format matches what we're searching for
                                if "subject" in sample and "reference" in sample["subject"]:
                                    logger.info(f"Sample subject reference: {sample['subject']['reference']}")

                                    # If we're searching by subject, check if the format matches
                                    if "subject" in params:
                                        logger.info(f"Search subject: {params['subject']}")
                                        logger.info(f"Match: {sample['subject']['reference'] == params['subject']}")
                        except Exception as e:
                            logger.error(f"Error getting sample document: {str(e)}")

                    return results
                except Exception as e:
                    logger.error(f"Error searching for Encounter resources in MongoDB: {str(e)}")
                    # Fallback to in-memory search
                    logger.info("Falling back to in-memory search")

            # Fallback to in-memory search
            results = []

            # Log the search parameters for debugging
            logger.info(f"In-memory search parameters: {params}")

            # Filter encounters based on search parameters
            for encounter in self.encounters.values():
                # Check if the encounter matches all search parameters
                match = True

                for key, value in params.items():
                    # Skip pagination parameters
                    if key in ["_count", "_page"]:
                        continue

                    # Handle special parameters
                    if key == "_id" and encounter.get("id") != value:
                        match = False
                        logger.debug(f"Encounter {encounter.get('id')} doesn't match _id={value}")
                        break
                    elif key == "patient" and not encounter.get("subject", {}).get("reference", "").endswith(f"/{value}"):
                        match = False
                        logger.debug(f"Encounter {encounter.get('id')} doesn't match patient={value}")
                        break
                    elif key == "subject":
                        subject_ref = encounter.get("subject", {}).get("reference", "")
                        if subject_ref != value:
                            logger.debug(f"Encounter {encounter.get('id')} subject={subject_ref} doesn't match {value}")
                            match = False
                            break
                        else:
                            logger.debug(f"Encounter {encounter.get('id')} matches subject={value}")
                    elif key == "date":
                        # Check if the encounter period includes the date
                        period = encounter.get("period", {})
                        start = period.get("start")
                        end = period.get("end")

                        if not start or not end or start > value or end < value:
                            match = False
                            logger.debug(f"Encounter {encounter.get('id')} period doesn't include date={value}")
                            break
                    elif key not in encounter or encounter[key] != value:
                        match = False
                        logger.debug(f"Encounter {encounter.get('id')} doesn't match {key}={value}")
                        break

                # If the encounter matches all parameters, add it to the results
                if match:
                    results.append(encounter)

            logger.info(f"Found {len(results)} Encounter resources in memory")
            return results
        except Exception as e:
            logger.error(f"Error searching for Encounter resources: {str(e)}")
            # Just raise the error
            raise
