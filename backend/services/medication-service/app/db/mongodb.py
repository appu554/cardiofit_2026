from motor.motor_asyncio import AsyncIOMotorClient, AsyncIOMotorDatabase, AsyncIOMotorCollection
from app.core.config import settings
import logging

# Configure logging
logger = logging.getLogger(__name__)

# MongoDB client
client = None
db = None
_initialized = False

class MongoDB:
    def __init__(self):
        self.client = None
        self.db = None
        self._initialized = False

    async def connect(self):
        """Connect to MongoDB."""
        try:
            logger.info(f"Connecting to MongoDB at {settings.MONGODB_URL}...")
            self.client = AsyncIOMotorClient(settings.MONGODB_URL)
            self.db = self.client[settings.MONGODB_DB_NAME]
            self._initialized = True
            logger.info(f"Successfully connected to MongoDB")
            return True
        except Exception as e:
            logger.error(f"Error connecting to MongoDB: {str(e)}")
            return False

    async def close(self):
        """Close MongoDB connection."""
        if self.client:
            self.client.close()
            logger.info("Closed MongoDB connection")

    def is_connected(self):
        """Check if MongoDB is connected."""
        return self._initialized and self.client is not None and self.db is not None

    def get_status(self):
        """Get MongoDB connection status."""
        if not self._initialized:
            return "not_initialized"
        if not self.client:
            return "no_client"
        if not self.db:
            return "no_database"
        return "connected"

    async def ensure_collection(self, collection_name):
        """Ensure a collection exists."""
        if not self.is_connected():
            logger.warning(f"Database not connected, cannot ensure {collection_name} collection exists")
            return False

        try:
            collections = await self.db.list_collection_names()
            if collection_name not in collections:
                # Create the collection
                await self.db.create_collection(collection_name)
                logger.info(f"Created {collection_name} collection")
            return True
        except Exception as e:
            logger.error(f"Error ensuring {collection_name} collection exists: {str(e)}")
            return False

# Create a global instance of MongoDB
db = MongoDB()

async def connect_to_mongo():
    """Connect to MongoDB."""
    global db
    return await db.connect()

async def close_mongo_connection():
    """Close MongoDB connection."""
    global db
    await db.close()

def get_medication_requests_collection() -> AsyncIOMotorCollection:
    """Get the medication_requests collection."""
    if not db.is_connected():
        logger.warning("Database not connected, cannot get medication_requests collection")
        return None
    return db.db.medication_requests
