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
    _initialized = False
    _status = "not_connected"

    @classmethod
    def is_connected(cls) -> bool:
        """Check if the database is connected."""
        # More robust check that verifies both client and db are available
        is_connected = cls._initialized and cls.client is not None and cls.db is not None
        if not is_connected and cls._status == "connected":
            # Fix inconsistent state
            logger.warning("Database state is inconsistent. Status is 'connected' but client or db is None.")
            if cls.client is None:
                logger.warning("Database client is None")
            if cls.db is None:
                logger.warning("Database db is None")
            cls._status = "not_connected"
        return is_connected

    @classmethod
    def get_status(cls) -> str:
        """Get the database connection status."""
        # Verify status is consistent with actual state
        if cls._status == "connected" and (cls.client is None or cls.db is None):
            cls._status = "not_connected"
        return cls._status

db = Database()

# Connection retry decorator
def with_retry(max_retries=3, retry_delay=2):
    """Decorator to retry async functions with exponential backoff"""
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            for attempt in range(max_retries):
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    if attempt < max_retries - 1:
                        wait_time = retry_delay * (2 ** attempt)  # Exponential backoff
                        logger.warning(f"Operation failed: {str(e)}. Retrying in {wait_time}s...")
                        await asyncio.sleep(wait_time)
                    else:
                        logger.error(f"Operation failed after {max_retries} attempts: {str(e)}")
                        raise
        return wrapper
    return decorator

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

            # Ensure the observations collection exists
            await ensure_collection_exists("observations")

            return True
        except Exception as e:
            logger.warning(f"Failed to connect to MongoDB on attempt {attempt + 1}: {str(e)}")

            if attempt < max_retries - 1:
                wait_time = retry_delay * (2 ** attempt)  # Exponential backoff
                logger.info(f"Retrying in {wait_time} seconds...")
                await asyncio.sleep(wait_time)
            else:
                logger.error(f"Failed to connect to MongoDB after {max_retries} attempts")
                db._status = f"connection_failed: {str(e)}"
                return False

    return False

async def close_mongo_connection():
    """Close MongoDB connection."""
    logger.info("Closing MongoDB connection...")
    if db.client:
        db.client.close()
        db._initialized = False
        db._status = "disconnected"
    logger.info("MongoDB connection closed")

@with_retry(max_retries=2, retry_delay=1)
async def ensure_collection_exists(collection_name: str) -> bool:
    """
    Ensure a collection exists in the database.

    Args:
        collection_name: Name of the collection to check/create

    Returns:
        bool: True if the collection exists or was created, False otherwise
    """
    if not db.is_connected():
        logger.warning(f"Database not connected, cannot ensure {collection_name} collection exists")
        return False

    try:
        # Check if collection exists
        collections = await db.db.list_collection_names()

        if collection_name not in collections:
            logger.info(f"Creating {collection_name} collection...")
            await db.db.create_collection(collection_name)
            logger.info(f"{collection_name} collection created")

        return True
    except Exception as e:
        logger.error(f"Error ensuring {collection_name} collection exists: {str(e)}")
        return False

def get_observations_collection():
    """Get the observations collection."""
    # Check connection status
    if not db.is_connected():
        logger.warning("Database not connected, cannot get observations collection")
        return None

    try:
        # Get the collection
        if db.db is None:
            logger.error("Database object is None, cannot get observations collection")
            return None

        collection = db.db.observations
        return collection
    except Exception as e:
        logger.error(f"Error getting observations collection: {str(e)}")
        return None
