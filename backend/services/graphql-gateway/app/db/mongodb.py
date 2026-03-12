from motor.motor_asyncio import AsyncIOMotorClient
import logging
import traceback
from typing import Optional, Dict, Any, List

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class MongoDB:
    client: AsyncIOMotorClient = None
    db = None

db = MongoDB()

async def connect_to_mongo(mongodb_uri: str):
    """Connect to MongoDB."""
    try:
        # Log the MongoDB URI (with password masked)
        uri_parts = mongodb_uri.split('@')
        if len(uri_parts) > 1:
            masked_uri = f"mongodb+srv://***:***@{uri_parts[1]}"
        else:
            masked_uri = "mongodb+srv://***:***@example.mongodb.net"
        logger.info(f"Connecting to MongoDB Atlas: {masked_uri}")

        # Connect to MongoDB Atlas with a longer timeout
        db.client = AsyncIOMotorClient(
            mongodb_uri,
            serverSelectionTimeoutMS=30000,  # 30 seconds
            connectTimeoutMS=30000,          # 30 seconds
            socketTimeoutMS=30000,           # 30 seconds
            maxPoolSize=10,                  # Maximum connection pool size
            minPoolSize=1,                   # Minimum connection pool size
            maxIdleTimeMS=60000,            # Maximum idle time for a connection
            waitQueueTimeoutMS=10000,        # How long a thread will wait for a connection
            retryWrites=True,                # Retry writes if they fail
            w="majority"                     # Write concern
        )

        # Get the default database from the connection string
        # or use 'fhirdb' as a fallback
        db.db = db.client.get_database()
        logger.info(f"Using database: {db.db.name}")

        logger.info("Successfully connected to MongoDB Atlas")
        return True
    except Exception as e:
        logger.error(f"Failed to connect to MongoDB: {str(e)}")
        logger.error(traceback.format_exc())
        return False

async def close_mongo_connection():
    """Close MongoDB connection."""
    if db.client:
        db.client.close()
        logger.info("Closed MongoDB connection")

# Helper functions for FHIR resources
async def get_patients() -> List[Dict[str, Any]]:
    """Get all patients from the database."""
    try:
        cursor = db.db.Patient.find({})
        patients = await cursor.to_list(length=100)

        # Convert MongoDB _id to id for each patient
        for patient in patients:
            if '_id' in patient and 'id' not in patient:
                # Convert ObjectId to string
                patient['id'] = str(patient['_id'])

        return patients
    except Exception as e:
        logger.error(f"Error getting patients: {str(e)}")
        return []

async def get_patient(patient_id: str) -> Optional[Dict[str, Any]]:
    """Get a patient by ID."""
    try:
        # Try to find by id field first
        patient = await db.db.Patient.find_one({"id": patient_id})

        # If not found, try to find by _id field
        if not patient:
            from bson.objectid import ObjectId
            try:
                # Try to convert to ObjectId (if it's a valid ObjectId string)
                patient = await db.db.Patient.find_one({"_id": ObjectId(patient_id)})
            except:
                # If conversion fails, it's not a valid ObjectId
                pass

        # Add id field if only _id exists
        if patient and '_id' in patient and 'id' not in patient:
            patient['id'] = str(patient['_id'])

        return patient
    except Exception as e:
        logger.error(f"Error getting patient {patient_id}: {str(e)}")
        return None

async def create_patient(patient_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
    """Create a new patient."""
    try:
        # Make sure we have an id field
        if 'id' not in patient_data:
            import uuid
            patient_data['id'] = str(uuid.uuid4())

        result = await db.db.Patient.insert_one(patient_data)
        if result.inserted_id:
            # Use the id field for retrieval
            return await get_patient(patient_data["id"])
        return None
    except Exception as e:
        logger.error(f"Error creating patient: {str(e)}")
        return None

async def update_patient(patient_id: str, patient_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
    """Update a patient."""
    try:
        # First get the patient to determine which ID field to use
        patient = await get_patient(patient_id)
        if not patient:
            return None

        # Determine which field to use for the query
        from bson.objectid import ObjectId
        if '_id' in patient:
            # If the patient has an _id field, use that for the update
            try:
                # Try to use ObjectId
                query = {"_id": ObjectId(str(patient['_id']))}
            except:
                # Fall back to string ID
                query = {"_id": patient['_id']}
        else:
            # Otherwise use the id field
            query = {"id": patient_id}

        result = await db.db.Patient.update_one(
            query,
            {"$set": patient_data}
        )

        if result.modified_count > 0:
            return await get_patient(patient_id)
        return None
    except Exception as e:
        logger.error(f"Error updating patient {patient_id}: {str(e)}")
        return None

async def delete_patient(patient_id: str) -> bool:
    """Delete a patient."""
    try:
        # First get the patient to determine which ID field to use
        patient = await get_patient(patient_id)
        if not patient:
            return False

        # Determine which field to use for the query
        from bson.objectid import ObjectId
        if '_id' in patient:
            # If the patient has an _id field, use that for the delete
            try:
                # Try to use ObjectId
                query = {"_id": ObjectId(str(patient['_id']))}
            except:
                # Fall back to string ID
                query = {"_id": patient['_id']}
        else:
            # Otherwise use the id field
            query = {"id": patient_id}

        result = await db.db.Patient.delete_one(query)
        return result.deleted_count > 0
    except Exception as e:
        logger.error(f"Error deleting patient {patient_id}: {str(e)}")
        return False
