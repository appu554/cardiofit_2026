import motor.motor_asyncio
from app.core.config import settings
import logging
import time
import asyncio
from typing import Optional
from functools import wraps

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Connection pool settings
MAX_POOL_SIZE = 10
MIN_POOL_SIZE = 1
MAX_IDLE_TIME_MS = 30000  # 30 seconds

class Database:
    client: motor.motor_asyncio.AsyncIOMotorClient = None
    db = None
    _initialized: bool = False
    _status: str = "not_initialized"

    def is_connected(self) -> bool:
        """Check if MongoDB is connected."""
        return self._initialized and self.client is not None and self.db is not None

    def get_status(self) -> str:
        """Get MongoDB connection status."""
        if hasattr(self, '_status'):
            return self._status
        if not self._initialized:
            return "not_initialized"
        if not self.client:
            return "no_client"
        if not self.db:
            return "no_database"
        return "connected"

    async def ensure_collection(self, collection_name: str) -> bool:
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

db = Database()

async def ensure_collection_exists(collection_name: str) -> bool:
    """Ensure a collection exists in the database."""
    return await db.ensure_collection(collection_name)

async def connect_to_mongo(max_retries: int = 3, retry_delay: int = 2) -> bool:
    """
    Connect to MongoDB with retry logic and connection pooling.

    Args:
        max_retries: Maximum number of connection attempts
        retry_delay: Delay between retries in seconds

    Returns:
        bool: True if connection was successful, False otherwise
    """
    logger.info(f"Connecting to MongoDB at {settings.MONGODB_URL}...")

    # Reset the database state
    db._initialized = False
    db._status = "connecting"

    # Only close the client if it exists
    if db.client:
        logger.info("Closing existing MongoDB connection...")
        db.client.close()

    db.client = None
    db.db = None

    for attempt in range(max_retries):
        try:
            # Create the client with connection pooling
            db.client = motor.motor_asyncio.AsyncIOMotorClient(
                settings.MONGODB_URL,
                serverSelectionTimeoutMS=5000,  # 5 second timeout
                maxPoolSize=MAX_POOL_SIZE,
                minPoolSize=MIN_POOL_SIZE,
                maxIdleTimeMS=MAX_IDLE_TIME_MS,
                retryWrites=True,
                retryReads=True,
                connectTimeoutMS=5000,
                socketTimeoutMS=10000,
                waitQueueTimeoutMS=5000
            )

            # Test the connection with a timeout
            await db.client.admin.command('ping')

            # If we get here, the connection is successful
            db.db = db.client[settings.MONGODB_DB_NAME]
            db._initialized = True
            db._status = "connected"

            # Log the database state
            logger.info(f"Successfully connected to MongoDB on attempt {attempt + 1}")

            # Ensure the diagnostic_reports collection exists
            await ensure_collection_exists("diagnostic_reports")

            return True
        except Exception as e:
            logger.warning(f"Failed to connect to MongoDB on attempt {attempt + 1}: {str(e)}")

            if attempt < max_retries - 1:
                # Wait before retrying
                logger.info(f"Retrying in {retry_delay} seconds...")
                await asyncio.sleep(retry_delay)
            else:
                # Last attempt failed
                logger.error(f"Failed to connect to MongoDB after {max_retries} attempts")
                db._status = "connection_failed"
                return False

    return False

async def close_mongo_connection():
    """Close MongoDB connection."""
    try:
        logger.info("Closing MongoDB connection...")
        if db.client:
            db.client.close()
        db._initialized = False
        db._status = "disconnected"
        logger.info("MongoDB connection closed")
    except Exception as e:
        logger.error(f"Error closing MongoDB connection: {str(e)}")

def get_diagnostic_reports_collection() -> Optional[motor.motor_asyncio.AsyncIOMotorCollection]:
    """Get the diagnostic_reports collection."""
    if not db.is_connected():
        logger.warning("Database not connected, cannot get diagnostic_reports collection")
        return None
    return db.db.diagnostic_reports
