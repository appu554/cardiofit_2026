"""
MongoDB Sink Implementation

Handles writing raw device data to MongoDB for backup and historical analysis
with comprehensive error handling and DLQ integration.
"""

from datetime import datetime
from typing import Dict, Any, Optional, List

import structlog
from motor.motor_asyncio import AsyncIOMotorClient
from pymongo.errors import PyMongoError, ConnectionFailure, ServerSelectionTimeoutError

from app.config import settings, get_mongodb_config

logger = structlog.get_logger(__name__)


class MongoDBSink:
    """
    MongoDB Sink for raw device data backup
    
    Stores raw device readings in MongoDB for historical analysis,
    backup purposes, and data recovery scenarios.
    """
    
    def __init__(self):
        self.sink_name = "mongodb"
        self.client = None
        self.database = None
        self.collection = None
        self.total_writes = 0
        self.successful_writes = 0
        self.failed_writes = 0
        
        mongo_config = get_mongodb_config()
        self.database_name = mongo_config["database"]
        self.collection_name = mongo_config["collection"]
        
        logger.info("MongoDB Sink initialized",
                   database=self.database_name,
                   collection=self.collection_name)
    
    async def initialize(self):
        """Initialize MongoDB client and database connection"""
        try:
            mongo_config = get_mongodb_config()
            
            # Initialize async MongoDB client
            self.client = AsyncIOMotorClient(
                mongo_config["uri"],
                serverSelectionTimeoutMS=mongo_config["timeout"] * 1000
            )
            
            # Get database and collection references
            self.database = self.client[self.database_name]
            self.collection = self.database[self.collection_name]
            
            # Test connection
            await self.client.admin.command('ping')
            
            # Ensure indexes for better performance
            await self._ensure_indexes()
            
            logger.info("MongoDB client initialized successfully",
                       database=self.database_name,
                       collection=self.collection_name)
            
        except ConnectionFailure as e:
            logger.error("Failed to connect to MongoDB", error=str(e))
            raise
        except Exception as e:
            logger.error("Failed to initialize MongoDB client", error=str(e))
            raise
    
    async def write_raw_data(self, raw_data: Dict[str, Any], device_id: str) -> bool:
        """
        Write raw device data to MongoDB
        
        Args:
            raw_data: Raw device reading data
            device_id: Device ID for logging
            
        Returns:
            True if successful, False otherwise
        """
        try:
            self.total_writes += 1
            
            # Prepare document for MongoDB
            document = raw_data.copy()
            document.update({
                "_id": self._generate_document_id(raw_data, device_id),
                "stored_at": datetime.utcnow(),
                "sink_name": self.sink_name,
                "document_type": "raw_device_reading"
            })
            
            # Insert document
            result = await self.collection.insert_one(document)
            
            if result.inserted_id:
                self.successful_writes += 1
                
                logger.debug("Raw data written to MongoDB",
                            device_id=device_id,
                            document_id=result.inserted_id,
                            collection=self.collection_name)
                
                return True
            else:
                self.failed_writes += 1
                logger.error("MongoDB insert failed - no inserted_id", device_id=device_id)
                return False
            
        except PyMongoError as e:
            self.failed_writes += 1
            logger.error("MongoDB write error", device_id=device_id, error=str(e))
            raise
            
        except Exception as e:
            self.failed_writes += 1
            logger.error("MongoDB write failed", device_id=device_id, error=str(e))
            raise
    
    def write_fhir_observation(self, fhir_data: str, device_id: str) -> bool:
        """
        MongoDB doesn't handle FHIR observations in this implementation - this is a no-op
        
        Args:
            fhir_data: FHIR Observation JSON string
            device_id: Device ID for logging
            
        Returns:
            True (no-op)
        """
        logger.debug("FHIR observation write skipped for MongoDB", device_id=device_id)
        return True
    
    def write_ui_document(self, ui_data: str, device_id: str) -> bool:
        """
        MongoDB doesn't handle UI documents in this implementation - this is a no-op
        
        Args:
            ui_data: UI document JSON string
            device_id: Device ID for logging
            
        Returns:
            True (no-op)
        """
        logger.debug("UI document write skipped for MongoDB", device_id=device_id)
        return True
    
    def _generate_document_id(self, raw_data: Dict[str, Any], device_id: str) -> str:
        """Generate unique document ID for MongoDB"""
        timestamp = raw_data.get("timestamp", int(datetime.utcnow().timestamp()))
        reading_type = raw_data.get("reading_type", "unknown")
        return f"{device_id}_{timestamp}_{reading_type}_raw"
    
    async def _ensure_indexes(self):
        """Ensure proper indexes exist for better query performance"""
        try:
            # Create indexes for common query patterns
            indexes = [
                [("device_id", 1), ("timestamp", -1)],  # Device timeline queries
                [("patient_id", 1), ("timestamp", -1)],  # Patient timeline queries
                [("reading_type", 1), ("timestamp", -1)],  # Reading type queries
                [("stored_at", -1)],  # Recent data queries
                [("alert_level", 1), ("timestamp", -1)]  # Alert queries
            ]
            
            for index_spec in indexes:
                try:
                    await self.collection.create_index(index_spec, background=True)
                except Exception as e:
                    logger.warning("Failed to create MongoDB index", 
                                 index=index_spec, error=str(e))
            
            logger.info("MongoDB indexes ensured", collection=self.collection_name)
            
        except Exception as e:
            logger.warning("Failed to ensure MongoDB indexes", error=str(e))
    
    async def get_recent_readings(self, device_id: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Get recent readings for a device (utility method)"""
        try:
            cursor = self.collection.find(
                {"device_id": device_id}
            ).sort("timestamp", -1).limit(limit)
            
            return await cursor.to_list(length=limit)
            
        except Exception as e:
            logger.error("Failed to get recent readings", device_id=device_id, error=str(e))
            return []
    
    async def get_readings_by_patient(self, patient_id: str, 
                                    start_time: Optional[int] = None,
                                    end_time: Optional[int] = None,
                                    limit: int = 100) -> List[Dict[str, Any]]:
        """Get readings for a patient within time range (utility method)"""
        try:
            query = {"patient_id": patient_id}
            
            if start_time or end_time:
                time_query = {}
                if start_time:
                    time_query["$gte"] = start_time
                if end_time:
                    time_query["$lte"] = end_time
                query["timestamp"] = time_query
            
            cursor = self.collection.find(query).sort("timestamp", -1).limit(limit)
            
            return await cursor.to_list(length=limit)
            
        except Exception as e:
            logger.error("Failed to get patient readings", patient_id=patient_id, error=str(e))
            return []
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get MongoDB sink metrics"""
        return {
            "sink_name": self.sink_name,
            "total_writes": self.total_writes,
            "successful_writes": self.successful_writes,
            "failed_writes": self.failed_writes,
            "success_rate": self.successful_writes / max(self.total_writes, 1),
            "database": self.database_name,
            "collection": self.collection_name
        }
    
    def is_healthy(self) -> bool:
        """Check if MongoDB sink is healthy"""
        try:
            if not self.client:
                return False

            # Simple health check - verify client and database are available
            return self.database is not None and self.collection is not None

        except Exception as e:
            logger.warning("MongoDB health check failed", error=str(e))
            return False
    
    async def close(self):
        """Close MongoDB sink and cleanup resources"""
        if self.client:
            self.client.close()
            self.client = None
        
        logger.info("MongoDB sink closed",
                   total_writes=self.total_writes,
                   successful_writes=self.successful_writes,
                   failed_writes=self.failed_writes)
