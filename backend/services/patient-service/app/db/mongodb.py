"""
MongoDB connection module for the Patient Service.

This module provides functions for connecting to MongoDB and accessing the database.
It handles connection pooling, retries, and graceful error handling.
"""

import logging
import time
from typing import Optional
from motor.motor_asyncio import AsyncIOMotorClient, AsyncIOMotorDatabase
from app.core.config import settings

# Configure logging
logger = logging.getLogger(__name__)

class Database:
    """Database connection manager for MongoDB."""

    client: Optional[AsyncIOMotorClient] = None
    db: Optional[AsyncIOMotorDatabase] = None
    _initialized: bool = False

    @classmethod
    def get_collection(cls, collection_name: str):
        """
        Get a collection from the database.

        Args:
            collection_name: Name of the collection

        Returns:
            The collection or None if the database is not connected
        """
        if not cls._initialized or cls.db is None:
            logger.warning(f"Database not connected, cannot get collection {collection_name}")
            return None

        try:
            # Try to access the collection
            collection = cls.db[collection_name]
            logger.info(f"Successfully accessed collection {collection_name}")
            return collection
        except Exception as e:
            logger.error(f"Error accessing collection {collection_name}: {str(e)}")
            return None

    @classmethod
    def is_connected(cls) -> bool:
        """
        Check if the database is connected.

        Returns:
            True if connected, False otherwise
        """
        return cls._initialized and cls.client is not None and cls.db is not None

    @classmethod
    def set_initialized(cls, value: bool = True):
        """
        Set the initialized state of the database.

        Args:
            value: The value to set the initialized state to
        """
        cls._initialized = value
        logger.info(f"Database initialized state set to {value}")

    @classmethod
    def get_status(cls) -> str:
        """
        Get the status of the database connection.

        Returns:
            A string describing the status of the database connection
        """
        if not cls._initialized:
            return "Not initialized"
        if cls.client is None:
            return "No client"
        if cls.db is None:
            return "No database"
        return "Connected"

db = Database()

async def connect_to_mongo(max_retries: int = 3, retry_delay: int = 5) -> bool:
    """
    Connect to MongoDB with retry logic.

    Args:
        max_retries: Maximum number of connection attempts
        retry_delay: Delay between retries in seconds

    Returns:
        True if connection was successful, False otherwise
    """
    # Log MongoDB connection details (with password masked)
    mongo_url = settings.MONGODB_URL
    masked_url = mongo_url
    if "@" in mongo_url:
        # Mask the password in the URL for logging
        parts = mongo_url.split("@")
        auth_parts = parts[0].split(":")
        if len(auth_parts) > 2:
            masked_url = f"{auth_parts[0]}:****@{parts[1]}"

    logger.info(f"MongoDB URL: {masked_url}")
    logger.info(f"MongoDB Database: {settings.MONGODB_DB_NAME}")

    for attempt in range(1, max_retries + 1):
        try:
            logger.info(f"Connecting to MongoDB (attempt {attempt}/{max_retries})...")

            # Connect to MongoDB with a timeout
            db.client = AsyncIOMotorClient(
                settings.MONGODB_URL,
                serverSelectionTimeoutMS=10000,  # 10 seconds
                connectTimeoutMS=10000,          # 10 seconds
                socketTimeoutMS=10000,           # 10 seconds
                maxPoolSize=10,                  # Maximum connection pool size
                minPoolSize=1,                   # Minimum connection pool size
                maxIdleTimeMS=60000,             # Maximum idle time for a connection
                waitQueueTimeoutMS=10000,        # How long a thread will wait for a connection
                retryWrites=True,                # Retry writes if they fail
                w="majority",                    # Write concern
                appname="patient-service"        # Application name for monitoring
            )

            # Get the database
            db.db = db.client[settings.MONGODB_DB_NAME]

            # Test the connection with a timeout
            logger.info("Testing MongoDB connection with ping command...")
            await db.client.admin.command('ping')

            # List available databases to verify connection
            database_names = await db.client.list_database_names()
            logger.info(f"Available MongoDB databases: {database_names}")

            # Check if our database exists
            if settings.MONGODB_DB_NAME in database_names:
                logger.info(f"Database '{settings.MONGODB_DB_NAME}' exists")
            else:
                logger.warning(f"Database '{settings.MONGODB_DB_NAME}' does not exist yet, it will be created on first write")

            # Create the patients collection if it doesn't exist
            if settings.MONGODB_DB_NAME in database_names:
                collection_names = await db.db.list_collection_names()
                logger.info(f"Collections in {settings.MONGODB_DB_NAME}: {collection_names}")

                if "patients" not in collection_names:
                    logger.info("Creating 'patients' collection...")
                    await db.db.create_collection("patients")
                    logger.info("'patients' collection created successfully")

                    # Create a unique index on the id field
                    logger.info("Creating unique index on 'id' field...")
                    await db.db.patients.create_index("id", unique=True)
                    logger.info("Unique index created successfully")
                else:
                    logger.info("'patients' collection already exists")

                    # Check if the unique index exists
                    indexes = await db.db.patients.index_information()
                    has_unique_id_index = any(
                        index_info.get("unique", False) and "id_1" in index_name
                        for index_name, index_info in indexes.items()
                    )

                    if not has_unique_id_index:
                        logger.info("Creating unique index on 'id' field...")
                        await db.db.patients.create_index("id", unique=True)
                        logger.info("Unique index created successfully")
                    else:
                        logger.info("Unique index on 'id' field already exists")

            # Mark the database as initialized
            db.set_initialized(True)

            logger.info("Connected to MongoDB successfully")
            return True
        except Exception as e:
            logger.error(f"Failed to connect to MongoDB (attempt {attempt}/{max_retries}): {str(e)}")

            # Make sure db.db is None to indicate connection failure
            db.db = None

            if db.client:
                try:
                    db.client.close()
                except Exception:
                    pass
                db.client = None

            # If this is not the last attempt, wait before retrying
            if attempt < max_retries:
                logger.info(f"Retrying in {retry_delay} seconds...")
                time.sleep(retry_delay)

    logger.error(f"Failed to connect to MongoDB after {max_retries} attempts")
    return False

async def close_mongo_connection() -> None:
    """
    Close MongoDB connection safely.

    This function ensures that the MongoDB connection is properly closed
    to prevent resource leaks.
    """
    logger.info("Closing MongoDB connection...")
    if db.client:
        try:
            db.client.close()
            logger.info("MongoDB connection closed successfully")
        except Exception as e:
            logger.error(f"Error closing MongoDB connection: {str(e)}")
    else:
        logger.info("No MongoDB connection to close")

    # Reset the database connection objects
    db.client = None
    db.db = None
    db.set_initialized(False)

# Helper function to get the patients collection
def get_patients_collection():
    """
    Get the patients collection from MongoDB.

    Returns:
        The patients collection or None if the database is not connected
    """
    # Check if the database is connected
    if not db._initialized:
        logger.warning(f"Database not initialized (status: {db.get_status()}), attempting to reconnect...")
        # We can't reconnect here because this is a synchronous function
        # Just return None and let the caller handle it
        return None

    if db.db is None:
        logger.warning("Database object is None, cannot get collection")
        return None

    # Get the collection directly from the db object
    try:
        collection = db.db["patients"]
        logger.info("Successfully got patients collection")
        return collection
    except Exception as e:
        logger.error(f"Error getting patients collection: {str(e)}")
        return None
