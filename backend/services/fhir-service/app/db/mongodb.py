from motor.motor_asyncio import AsyncIOMotorClient
from app.core.config import settings
import asyncio
import logging
import traceback

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

class MongoDB:
    client: AsyncIOMotorClient = None
    db = None

db = MongoDB()

async def connect_to_mongo():
    """Connect to MongoDB."""
    try:
        # Log the MongoDB URI (with password masked)
        uri_parts = settings.MONGODB_URI.split('@')
        if len(uri_parts) > 1:
            masked_uri = f"mongodb+srv://***:***@{uri_parts[1]}"
        else:
            masked_uri = "mongodb+srv://***:***@example.mongodb.net"
        logger.info(f"Connecting to MongoDB Atlas: {masked_uri}")

        # Connect to MongoDB Atlas with a longer timeout
        db.client = AsyncIOMotorClient(
            settings.MONGODB_URI,
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

        # Verify the connection by getting server info
        logger.info("Verifying connection...")
        server_info = await db.client.admin.command('ismaster')
        logger.debug(f"Server info: {server_info}")

        # Get the default database from the connection string
        # or use 'fhirdb' as a fallback
        db.db = db.client.get_database()
        logger.info(f"Using database: {db.db.name}")

        # Test the database connection by inserting and removing a test document
        logger.info("Testing database write access...")
        test_result = await db.db.connection_test.insert_one({"test": "connection", "timestamp": asyncio.get_event_loop().time()})
        logger.debug(f"Test document inserted with ID: {test_result.inserted_id}")
        delete_result = await db.db.connection_test.delete_one({"_id": test_result.inserted_id})
        logger.debug(f"Test document deleted: {delete_result.deleted_count} document(s)")

        logger.info("Successfully connected to MongoDB Atlas")
    except Exception as e:
        logger.error(f"Failed to connect to MongoDB: {str(e)}")
        logger.error(traceback.format_exc())
        raise

async def close_mongo_connection():
    """Close MongoDB connection."""
    if db.client:
        db.client.close()
        logger.info("Closed MongoDB connection")

# Add a helper function to check connection status
async def check_connection():
    """Check if the MongoDB connection is still alive."""
    try:
        if db.client and db.db:
            # Try to ping the database
            await db.client.admin.command('ping')
            logger.info("MongoDB connection is alive")
            return True
        else:
            logger.error("MongoDB client or database is None")
            return False
    except Exception as e:
        logger.error(f"MongoDB connection check failed: {str(e)}")
        logger.error(traceback.format_exc())
        return False
