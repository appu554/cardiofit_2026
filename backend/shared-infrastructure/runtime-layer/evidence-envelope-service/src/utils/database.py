"""
Database utilities for Evidence Envelope Service
Provides MongoDB and Redis connection management
"""

import asyncio
from typing import Optional, Dict, Any
import motor.motor_asyncio
import redis.asyncio as aioredis
import structlog
from .config import settings

logger = structlog.get_logger()


class MongoDB:
    """Async MongoDB connection manager"""

    def __init__(self):
        self.client: Optional[motor.motor_asyncio.AsyncIOMotorClient] = None
        self.database: Optional[motor.motor_asyncio.AsyncIOMotorDatabase] = None

    async def connect(self) -> None:
        """Establish connection to MongoDB"""
        try:
            self.client = motor.motor_asyncio.AsyncIOMotorClient(
                settings.MONGODB_CONNECTION_STRING,
                maxPoolSize=settings.MONGODB_MAX_CONNECTIONS,
                minPoolSize=settings.MONGODB_MIN_CONNECTIONS,
                serverSelectionTimeoutMS=5000,
                connectTimeoutMS=10000,
                socketTimeoutMS=20000
            )

            self.database = self.client[settings.MONGODB_DATABASE_NAME]

            # Test connection
            await self.client.admin.command('ping')

            # Create indexes for optimal query performance
            await self._create_indexes()

            logger.info(
                "mongodb_connected",
                database=settings.MONGODB_DATABASE_NAME,
                connection_string=settings.MONGODB_CONNECTION_STRING.split('@')[-1]
            )

        except Exception as e:
            logger.error("mongodb_connection_failed", error=str(e))
            raise

    async def _create_indexes(self) -> None:
        """Create necessary indexes for envelope collection"""
        envelopes_collection = self.database.envelopes

        # Compound indexes for common queries
        await envelopes_collection.create_index([
            ("proposal_id", 1),
            ("created_at", -1)
        ])

        await envelopes_collection.create_index([
            ("patient_id", 1),
            ("workflow_type", 1),
            ("status", 1)
        ])

        await envelopes_collection.create_index([
            ("envelope_id", 1)
        ], unique=True)

        await envelopes_collection.create_index([
            ("status", 1),
            ("created_at", -1)
        ])

        # TTL index for automated cleanup (based on retention policy)
        await envelopes_collection.create_index([
            ("created_at", 1)
        ], expireAfterSeconds=settings.AUDIT_RETENTION_DAYS * 86400)

        logger.info("mongodb_indexes_created")

    async def close(self) -> None:
        """Close MongoDB connection"""
        if self.client:
            self.client.close()
            logger.info("mongodb_disconnected")

    def get_collection(self, collection_name: str):
        """Get a MongoDB collection"""
        if not self.database:
            raise RuntimeError("Database not connected")
        return self.database[collection_name]


class RedisClient:
    """Async Redis connection manager with connection pooling"""

    def __init__(self):
        self.redis: Optional[aioredis.Redis] = None
        self.connection_pool: Optional[aioredis.ConnectionPool] = None

    async def connect(self) -> None:
        """Establish connection to Redis"""
        try:
            self.connection_pool = aioredis.ConnectionPool(
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                password=settings.REDIS_PASSWORD if settings.REDIS_PASSWORD else None,
                db=settings.REDIS_DB,
                max_connections=settings.REDIS_MAX_CONNECTIONS,
                retry_on_timeout=settings.REDIS_RETRY_ON_TIMEOUT,
                decode_responses=True
            )

            self.redis = aioredis.Redis(connection_pool=self.connection_pool)

            # Test connection
            await self.redis.ping()

            logger.info(
                "redis_connected",
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                db=settings.REDIS_DB
            )

        except Exception as e:
            logger.error("redis_connection_failed", error=str(e))
            raise

    async def close(self) -> None:
        """Close Redis connection"""
        if self.redis:
            await self.redis.close()
        if self.connection_pool:
            await self.connection_pool.disconnect()
        logger.info("redis_disconnected")

    async def get(self, key: str) -> Optional[str]:
        """Get value from Redis"""
        if not self.redis:
            raise RuntimeError("Redis not connected")
        return await self.redis.get(key)

    async def set(
        self,
        key: str,
        value: str,
        ttl: Optional[int] = None
    ) -> bool:
        """Set value in Redis with optional TTL"""
        if not self.redis:
            raise RuntimeError("Redis not connected")
        return await self.redis.set(key, value, ex=ttl)

    async def delete(self, key: str) -> int:
        """Delete key from Redis"""
        if not self.redis:
            raise RuntimeError("Redis not connected")
        return await self.redis.delete(key)

    async def exists(self, key: str) -> bool:
        """Check if key exists in Redis"""
        if not self.redis:
            raise RuntimeError("Redis not connected")
        return await self.redis.exists(key) > 0

    async def ping(self) -> bool:
        """Test Redis connectivity"""
        try:
            if not self.redis:
                return False
            await self.redis.ping()
            return True
        except Exception:
            return False


# Global database instances
mongodb = MongoDB()
redis_client = RedisClient()


async def init_databases() -> None:
    """Initialize all database connections"""
    await mongodb.connect()
    await redis_client.connect()


async def close_databases() -> None:
    """Close all database connections"""
    await mongodb.close()
    await redis_client.close()