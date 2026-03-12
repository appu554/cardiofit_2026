"""
FHIR Service for Condition resources.

This module implements the FHIR service for Condition resources using MongoDB for data persistence.
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
from app.db.mongodb import get_conditions_collection, connect_to_mongo, db

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
_condition_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service.

    This function creates a global instance of the ConditionFHIRService
    and ensures it's properly connected to the database.

    Returns:
        The initialized ConditionFHIRService instance
    """
    global _condition_fhir_service

    # Log initialization start
    logger.info("Starting FHIR service initialization...")

    # Create a new service instance if needed
    if _condition_fhir_service is None:
        _condition_fhir_service = ConditionFHIRService()

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
    await _condition_fhir_service.initialize()

    # Log initialization completion
    logger.info("FHIR service initialization complete.")

    return _condition_fhir_service

def get_fhir_service():
    """
    Get the global FHIR service instance.

    Returns:
        The global ConditionFHIRService instance
    """
    global _condition_fhir_service

    if _condition_fhir_service is None:
        logger.warning("FHIR service not initialized yet. Creating a new instance.")
        _condition_fhir_service = ConditionFHIRService()

    return _condition_fhir_service

class ConditionFHIRService(FHIRServiceBase):
    """
    FHIR service for Condition resources.

    This service implements the FHIR operations for Condition resources
    using MongoDB for data persistence.
    """

    def __init__(self):
        """Initialize the Condition FHIR service."""
        super().__init__("Condition")
        # Use MongoDB for storing conditions
        self.collection = None
        # In-memory fallback storage for when MongoDB is not available
        self.conditions = {}
        # Flag to track if the service has been initialized
        self._initialized = False

    async def initialize(self):
        """
        Initialize the service and ensure database connection.

        This method ensures the MongoDB collection is available
        and initializes any necessary resources.
        """
        if self._initialized:
            logger.info("Condition FHIR service already initialized, skipping initialization")
            return

        logger.info("Initializing Condition FHIR service...")

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
            logger.info("Condition FHIR service initialized with in-memory storage only.")
            return

        # Try to get the MongoDB collection
        self.collection = get_conditions_collection()

        if self.collection is None:
            logger.warning("MongoDB collection not available. Trying to create it directly...")

            try:
                # Check if collection already exists
                collections = await db.db.list_collection_names()
                logger.info(f"Available collections: {collections}")

                if "conditions" not in collections:
                    logger.info("Creating conditions collection...")
                    await db.db.create_collection("conditions")
                    logger.info("Successfully created conditions collection")
                else:
                    logger.info("Conditions collection already exists")

                # Try to get the collection directly
                self.collection = db.db.conditions
                if self.collection is not None:
                    logger.info("Successfully retrieved conditions collection directly")
                else:
                    logger.error("Failed to retrieve conditions collection directly")
            except Exception as e:
                logger.error(f"Error creating/accessing conditions collection: {str(e)}")
                import traceback
                logger.error(traceback.format_exc())
                logger.warning("Will use in-memory storage.")
        else:
            logger.info("MongoDB collection available.")

            # Load existing conditions into memory for faster access
            try:
                logger.info("Loading conditions from MongoDB into memory cache...")
                cursor = self.collection.find({"resourceType": "Condition"})
                count = 0

                async for condition in cursor:
                    # Convert ObjectId to string
                    condition = _convert_objectid(condition)

                    # Cache the condition in memory
                    if "id" in condition:
                        self.conditions[condition["id"]] = condition
                        count += 1

                logger.info(f"Loaded {count} conditions from MongoDB into memory cache.")
            except Exception as e:
                logger.error(f"Error loading conditions from MongoDB: {str(e)}")
                import traceback
                logger.error(traceback.format_exc())

                # Try one more time with a direct approach
                try:
                    logger.info("Trying direct approach to load conditions...")
                    # Get all documents in the collection
                    cursor = db.db.conditions.find({})
                    count = 0

                    async for doc in cursor:
                        # Convert ObjectId to string
                        doc = _convert_objectid(doc)

                        # Cache the condition in memory
                        if "id" in doc:
                            self.conditions[doc["id"]] = doc
                            count += 1

                    logger.info(f"Loaded {count} conditions using direct approach.")
                except Exception as direct_error:
                    logger.error(f"Error with direct approach: {str(direct_error)}")
                    logger.error(traceback.format_exc())

        self._initialized = True
        logger.info("Condition FHIR service initialized.")
        logger.info(f"Using MongoDB collection: {self.collection is not None}")
        logger.info(f"In-memory cache size: {len(self.conditions)}")

    async def _ensure_collection(self):
        """
        Ensure the MongoDB collection is available.

        This method tries to get the MongoDB collection and reconnects
        to MongoDB if necessary.

        Returns:
            bool: True if the collection is available, False otherwise
        """
        if not self._initialized:
            logger.warning("Condition FHIR service not initialized. Call initialize() first.")
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
            self.collection = get_conditions_collection()

            # If still None, try to create the collection directly
            if self.collection is None and db.is_connected() and db.db is not None:
                try:
                    logger.info("Attempting to create conditions collection directly...")
                    # Check if collection already exists
                    collections = await db.db.list_collection_names()
                    logger.info(f"Available collections: {collections}")

                    if "conditions" not in collections:
                        await db.db.create_collection("conditions")
                        logger.info("Successfully created conditions collection")
                    else:
                        logger.info("Conditions collection already exists")

                    # Try to get the collection directly
                    self.collection = db.db.conditions
                    if self.collection is not None:
                        logger.info("Successfully retrieved conditions collection directly")
                    else:
                        logger.error("Failed to retrieve conditions collection directly")
                except Exception as e:
                    logger.error(f"Error creating/accessing conditions collection: {str(e)}")
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
        Create a new Condition resource.

        Args:
            resource: The FHIR Condition resource to create
            auth_header: Optional authorization header

        Returns:
            The created Condition resource

        Raises:
            HTTPException: If there is an error creating the resource
        """
        try:
            logger.info(f"Creating Condition resource with ID: {resource.get('id', 'new')}")

            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = str(uuid.uuid4())
            else:
                # Check if a condition with this ID already exists
                if await self._ensure_collection():
                    try:
                        existing_condition = await self.collection.find_one({"id": resource["id"]})
                        if existing_condition:
                            logger.warning(f"Condition with ID {resource['id']} already exists. Generating a new ID.")
                            resource["id"] = str(uuid.uuid4())
                    except Exception as e:
                        logger.error(f"Error checking for existing condition: {str(e)}")

            # Ensure the resource has the correct resource type
            resource["resourceType"] = "Condition"

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Make a copy of the resource to avoid modifying the original
                db_resource = dict(resource)

                # Store the condition in MongoDB
                try:
                    result = await self.collection.insert_one(db_resource)

                    # Update the resource with the MongoDB ID
                    if not resource.get("id"):
                        resource["id"] = str(result.inserted_id)

                    logger.info(f"Created Condition resource with ID {resource['id']} in MongoDB")
                except Exception as e:
                    # Check if it's a duplicate key error
                    if "duplicate key error" in str(e).lower():
                        logger.warning(f"Duplicate key error for Condition with ID {resource['id']}. Generating a new ID.")
                        # Generate a new ID
                        resource["id"] = str(uuid.uuid4())
                        db_resource["id"] = resource["id"]

                        try:
                            # Try again with the new ID
                            result = await self.collection.insert_one(db_resource)
                            logger.info(f"Created Condition resource with new ID {resource['id']} in MongoDB")
                        except Exception as retry_error:
                            logger.error(f"Error storing Condition resource with new ID in MongoDB: {str(retry_error)}")
                            # Fallback to in-memory storage
                            self.conditions[resource["id"]] = resource
                            logger.info(f"Created Condition resource with ID {resource['id']} in memory (after MongoDB error)")
                    else:
                        logger.error(f"Error storing Condition resource in MongoDB: {str(e)}")
                        # Fallback to in-memory storage
                        self.conditions[resource["id"]] = resource
                        logger.info(f"Created Condition resource with ID {resource['id']} in memory (after MongoDB error)")
            else:
                # Fallback to in-memory storage
                self.conditions[resource["id"]] = resource
                logger.info(f"Created Condition resource with ID {resource['id']} in memory")

            # Ensure all ObjectId instances are converted to strings
            return _convert_objectid(resource)
        except Exception as e:
            logger.error(f"Error creating Condition resource: {str(e)}")
            # Just raise the error
            raise

    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Get a Condition resource by ID.

        Args:
            resource_id: The ID of the Condition resource to retrieve
            auth_header: Optional authorization header

        Returns:
            The Condition resource if found, None otherwise

        Raises:
            HTTPException: If there is an error retrieving the resource
        """
        try:
            logger.info(f"Getting Condition resource with ID {resource_id}")

            # Check if the condition exists in memory first
            if resource_id in self.conditions:
                logger.info(f"Retrieved Condition resource with ID {resource_id} from memory")
                return self.conditions[resource_id]

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                # Try to find the condition in MongoDB
                try:
                    condition = await self.collection.find_one({"id": resource_id})
                except Exception as e:
                    logger.error(f"Error retrieving Condition resource from MongoDB: {str(e)}")
                    return None

                # If not found by id, try to find by MongoDB _id
                if not condition:
                    try:
                        # Try to convert the resource_id to ObjectId
                        if ObjectId.is_valid(resource_id):
                            condition = await self.collection.find_one({"_id": ObjectId(resource_id)})
                    except Exception as e:
                        logger.error(f"Error converting resource_id to ObjectId: {str(e)}")

                # If condition is found, return it
                if condition:
                    # Convert ObjectId to string
                    condition = _convert_objectid(condition)

                    # Cache the condition in memory
                    self.conditions[resource_id] = condition

                    logger.info(f"Retrieved Condition resource with ID {resource_id} from MongoDB")
                    return condition

            # If no condition is found, return None
            logger.info(f"No condition found with ID {resource_id}")
            return None
        except Exception as e:
            logger.error(f"Error getting Condition resource: {str(e)}")
            # Just raise the error
            raise

    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Update a Condition resource.

        Args:
            resource_id: The ID of the Condition resource to update
            resource: The updated Condition resource
            auth_header: Optional authorization header

        Returns:
            The updated Condition resource

        Raises:
            HTTPException: If there is an error updating the resource
        """
        try:
            logger.info(f"Updating Condition resource with ID {resource_id}")

            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "Condition"

            # Always update the in-memory cache
            self.conditions[resource_id] = resource

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                try:
                    # Check if the condition exists
                    existing_condition = await self.collection.find_one({"id": resource_id})

                    if existing_condition:
                        # Update the existing condition
                        result = await self.collection.replace_one({"id": resource_id}, resource)
                        logger.info(f"Updated Condition resource with ID {resource_id} in MongoDB, matched count: {result.matched_count}")
                    else:
                        # Create a new condition
                        result = await self.collection.insert_one(resource)
                        logger.info(f"Created new Condition resource with ID {resource_id} in MongoDB")
                except Exception as e:
                    logger.error(f"Error updating Condition resource in MongoDB: {str(e)}")
                    logger.info(f"Updated Condition resource with ID {resource_id} in memory only")
            else:
                logger.info(f"Updated Condition resource with ID {resource_id} in memory only")

            # Ensure all ObjectId instances are converted to strings
            return _convert_objectid(resource)
        except Exception as e:
            logger.error(f"Error updating Condition resource: {str(e)}")
            # Just raise the error
            raise

    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """
        Delete a Condition resource.

        Args:
            resource_id: The ID of the Condition resource to delete
            auth_header: Optional authorization header

        Returns:
            True if the resource was deleted, False otherwise

        Raises:
            HTTPException: If there is an error deleting the resource
        """
        try:
            # Add very visible logging
            print(f"\n\n==== CONDITION FHIR SERVICE DELETING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END CONDITION FHIR SERVICE ====\n\n")

            logger.info(f"Deleting Condition resource with ID {resource_id}")

            # Delete from in-memory cache
            in_memory = resource_id in self.conditions
            if in_memory:
                del self.conditions[resource_id]
                logger.info(f"Deleted Condition resource with ID {resource_id} from memory")

            # Check if MongoDB collection is available
            if await self._ensure_collection():
                try:
                    # Delete the condition from MongoDB
                    result = await self.collection.delete_one({"id": resource_id})

                    # Check if the condition was deleted
                    if result.deleted_count > 0:
                        logger.info(f"Deleted Condition resource with ID {resource_id} from MongoDB")
                        return True
                    else:
                        logger.info(f"No Condition resource found with ID {resource_id} in MongoDB")
                        # Return True if it was in memory
                        return in_memory
                except Exception as e:
                    logger.error(f"Error deleting Condition resource from MongoDB: {str(e)}")
                    # Return True if it was in memory
                    return in_memory
            else:
                # Return True if it was in memory
                return in_memory
        except Exception as e:
            logger.error(f"Error deleting Condition resource: {str(e)}")
            # Just raise the error
            raise

    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Search for Condition resources.

        Args:
            params: Search parameters
            auth_header: Optional authorization header

        Returns:
            A list of Condition resources matching the search criteria

        Raises:
            HTTPException: If there is an error searching for resources
        """
        try:
            # Add very visible logging
            print(f"\n\n==== CONDITION FHIR SERVICE SEARCHING RESOURCES ====")
            print(f"Search Parameters: {params}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END CONDITION FHIR SERVICE ====\n\n")

            logger.info(f"Searching for Condition resources with parameters: {params}")

            # First, try to search in MongoDB if available
            if await self._ensure_collection():
                # Build the query based on search parameters
                query = {"resourceType": "Condition"}

                # Handle subject search parameter
                if "subject" in params:
                    subject = params["subject"]
                    if subject.startswith("Patient/"):
                        patient_id = subject.split("/")[1]
                        query["subject.reference"] = f"Patient/{patient_id}"

                # Handle code search parameter
                if "code" in params:
                    code = params["code"]
                    query["code.coding.code"] = code

                # Handle clinical status search parameter
                if "clinical-status" in params:
                    clinical_status = params["clinical-status"]
                    query["clinicalStatus.coding.code"] = clinical_status

                # Handle verification status search parameter
                if "verification-status" in params:
                    verification_status = params["verification-status"]
                    query["verificationStatus.coding.code"] = verification_status

                # Handle onset date search parameter
                if "onset-date" in params:
                    onset_date = params["onset-date"]
                    query["onsetDateTime"] = {"$regex": f"^{onset_date}"}

                try:
                    # Get pagination parameters
                    count = int(params.get("_count", 100))
                    page = int(params.get("_page", 1))
                    skip = (page - 1) * count

                    # Execute the query
                    cursor = self.collection.find(query).skip(skip).limit(count)

                    # Convert cursor to list
                    conditions = []
                    async for condition in cursor:
                        # Convert MongoDB ObjectId to string
                        condition = _convert_objectid(condition)
                        conditions.append(condition)

                    # If conditions found in MongoDB, return them
                    if conditions:
                        logger.info(f"Returning {len(conditions)} Condition resources from MongoDB")
                        return conditions
                except Exception as db_error:
                    logger.error(f"Error searching MongoDB: {str(db_error)}")

            # If MongoDB search failed or no conditions found, search in memory
            memory_conditions = list(self.conditions.values())

            # Filter by search parameters
            filtered_conditions = memory_conditions

            # Handle subject search parameter
            if "subject" in params:
                subject = params["subject"]
                if subject.startswith("Patient/"):
                    patient_id = subject.split("/")[1]
                    filtered_conditions = [
                        c for c in filtered_conditions
                        if c.get("subject", {}).get("reference") == f"Patient/{patient_id}"
                    ]

            # Handle code search parameter
            if "code" in params:
                code = params["code"]
                filtered_conditions = [
                    c for c in filtered_conditions
                    if any(
                        coding.get("code") == code
                        for coding in c.get("code", {}).get("coding", [])
                    )
                ]

            # Handle clinical status search parameter
            if "clinical-status" in params:
                clinical_status = params["clinical-status"]
                filtered_conditions = [
                    c for c in filtered_conditions
                    if any(
                        coding.get("code") == clinical_status
                        for coding in c.get("clinicalStatus", {}).get("coding", [])
                    )
                ]

            # Handle verification status search parameter
            if "verification-status" in params:
                verification_status = params["verification-status"]
                filtered_conditions = [
                    c for c in filtered_conditions
                    if any(
                        coding.get("code") == verification_status
                        for coding in c.get("verificationStatus", {}).get("coding", [])
                    )
                ]

            # Handle onset date search parameter
            if "onset-date" in params:
                onset_date = params["onset-date"]
                filtered_conditions = [
                    c for c in filtered_conditions
                    if c.get("onsetDateTime", "").startswith(onset_date)
                ]

            # Get pagination parameters
            count = int(params.get("_count", 100))
            page = int(params.get("_page", 1))
            start = (page - 1) * count
            end = start + count

            # Apply pagination
            paginated_conditions = filtered_conditions[start:end]

            # If conditions found in memory, return them
            if paginated_conditions:
                logger.info(f"Returning {len(paginated_conditions)} Condition resources from memory")
                return paginated_conditions

            # If no conditions found anywhere, return an empty list
            logger.info("No conditions found, returning empty list")
            return []
        except Exception as e:
            logger.error(f"Error searching Condition resources: {str(e)}")
            # Just raise the error
            raise
