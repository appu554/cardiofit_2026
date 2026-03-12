"""
Module 8 Storage Clients

Unified client library for accessing all Module 8 storage infrastructure:
- MongoDB (document storage)
- Elasticsearch (search and analytics)
- ClickHouse (time-series analytics)
- Redis (caching)

Usage:
    from module8_storage_clients import Module8Storage

    storage = Module8Storage()

    # MongoDB operations
    storage.mongo.patients.insert_one(patient_data)

    # Elasticsearch operations
    storage.es.index(index="clinical_events", document=event_data)

    # ClickHouse operations
    storage.clickhouse.execute("SELECT * FROM patient_events")

    # Redis operations
    storage.redis.set("patient:123", patient_json)
"""

from typing import Optional, Dict, Any, List
import logging
from dataclasses import dataclass
from datetime import datetime
import json

# MongoDB
from pymongo import MongoClient
from pymongo.database import Database
from pymongo.collection import Collection

# Elasticsearch
from elasticsearch import Elasticsearch
from elasticsearch.helpers import bulk

# ClickHouse
from clickhouse_driver import Client as ClickHouseClient

# Redis
import redis
from redis import Redis

logger = logging.getLogger(__name__)


@dataclass
class StorageConfig:
    """Configuration for all storage services"""

    # MongoDB
    mongo_url: str = "mongodb://localhost:27017"
    mongo_database: str = "module8_clinical"

    # Elasticsearch
    es_hosts: List[str] = None
    es_index_prefix: str = "clinical_events"

    # ClickHouse
    clickhouse_host: str = "localhost"
    clickhouse_port: int = 9000
    clickhouse_user: str = "module8_user"
    clickhouse_password: str = "module8_password"
    clickhouse_database: str = "module8_analytics"

    # Redis
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_db: int = 0
    redis_decode_responses: bool = True

    def __post_init__(self):
        if self.es_hosts is None:
            self.es_hosts = ["http://localhost:9200"]


class MongoStorage:
    """MongoDB storage client for document operations"""

    def __init__(self, config: StorageConfig):
        self.config = config
        self.client: Optional[MongoClient] = None
        self.db: Optional[Database] = None

    def connect(self) -> None:
        """Establish MongoDB connection"""
        try:
            self.client = MongoClient(self.config.mongo_url)
            self.db = self.client[self.config.mongo_database]
            # Test connection
            self.client.admin.command('ping')
            logger.info(f"Connected to MongoDB: {self.config.mongo_database}")
        except Exception as e:
            logger.error(f"MongoDB connection failed: {e}")
            raise

    def close(self) -> None:
        """Close MongoDB connection"""
        if self.client:
            self.client.close()
            logger.info("MongoDB connection closed")

    @property
    def patients(self) -> Collection:
        """Get patients collection"""
        return self.db.patients

    @property
    def observations(self) -> Collection:
        """Get observations collection"""
        return self.db.observations

    @property
    def encounters(self) -> Collection:
        """Get encounters collection"""
        return self.db.encounters

    @property
    def medications(self) -> Collection:
        """Get medications collection"""
        return self.db.medications

    @property
    def alerts(self) -> Collection:
        """Get alerts collection"""
        return self.db.alerts

    @property
    def clinical_events(self) -> Collection:
        """Get clinical_events collection"""
        return self.db.clinical_events

    def get_collection(self, name: str) -> Collection:
        """Get any collection by name"""
        return self.db[name]

    def health_check(self) -> bool:
        """Check MongoDB health"""
        try:
            self.client.admin.command('ping')
            return True
        except Exception as e:
            logger.error(f"MongoDB health check failed: {e}")
            return False


class ElasticsearchStorage:
    """Elasticsearch storage client for search and analytics"""

    def __init__(self, config: StorageConfig):
        self.config = config
        self.client: Optional[Elasticsearch] = None

    def connect(self) -> None:
        """Establish Elasticsearch connection"""
        try:
            self.client = Elasticsearch(self.config.es_hosts)
            # Test connection
            if not self.client.ping():
                raise ConnectionError("Elasticsearch ping failed")
            logger.info(f"Connected to Elasticsearch: {self.config.es_hosts}")
        except Exception as e:
            logger.error(f"Elasticsearch connection failed: {e}")
            raise

    def close(self) -> None:
        """Close Elasticsearch connection"""
        if self.client:
            self.client.close()
            logger.info("Elasticsearch connection closed")

    def index(self, index: str, document: Dict[str, Any], doc_id: Optional[str] = None) -> Dict:
        """Index a single document"""
        return self.client.index(index=index, document=document, id=doc_id)

    def bulk_index(self, index: str, documents: List[Dict[str, Any]]) -> tuple:
        """Bulk index documents"""
        actions = [
            {
                "_index": index,
                "_source": doc
            }
            for doc in documents
        ]
        return bulk(self.client, actions)

    def search(self, index: str, query: Dict[str, Any], size: int = 100) -> Dict:
        """Search documents"""
        return self.client.search(index=index, query=query, size=size)

    def get(self, index: str, doc_id: str) -> Dict:
        """Get document by ID"""
        return self.client.get(index=index, id=doc_id)

    def delete(self, index: str, doc_id: str) -> Dict:
        """Delete document by ID"""
        return self.client.delete(index=index, id=doc_id)

    def create_index(self, index: str, mappings: Optional[Dict] = None) -> Dict:
        """Create index with optional mappings"""
        body = {}
        if mappings:
            body["mappings"] = mappings
        return self.client.indices.create(index=index, body=body)

    def health_check(self) -> bool:
        """Check Elasticsearch health"""
        try:
            return self.client.ping()
        except Exception as e:
            logger.error(f"Elasticsearch health check failed: {e}")
            return False


class ClickHouseStorage:
    """ClickHouse storage client for analytics and time-series data"""

    def __init__(self, config: StorageConfig):
        self.config = config
        self.client: Optional[ClickHouseClient] = None

    def connect(self) -> None:
        """Establish ClickHouse connection"""
        try:
            self.client = ClickHouseClient(
                host=self.config.clickhouse_host,
                port=self.config.clickhouse_port,
                user=self.config.clickhouse_user,
                password=self.config.clickhouse_password,
                database=self.config.clickhouse_database
            )
            # Test connection
            self.client.execute("SELECT 1")
            logger.info(f"Connected to ClickHouse: {self.config.clickhouse_database}")
        except Exception as e:
            logger.error(f"ClickHouse connection failed: {e}")
            raise

    def close(self) -> None:
        """Close ClickHouse connection"""
        if self.client:
            self.client.disconnect()
            logger.info("ClickHouse connection closed")

    def execute(self, query: str, params: Optional[Dict] = None) -> List:
        """Execute query and return results"""
        return self.client.execute(query, params or {})

    def insert_patient_event(
        self,
        event_id: str,
        patient_id: str,
        event_type: str,
        event_time: datetime,
        event_data: Dict[str, Any]
    ) -> None:
        """Insert patient event"""
        query = """
            INSERT INTO patient_events
            (event_id, patient_id, event_type, event_time, event_data)
            VALUES
        """
        self.client.execute(query, [{
            'event_id': event_id,
            'patient_id': patient_id,
            'event_type': event_type,
            'event_time': event_time,
            'event_data': json.dumps(event_data)
        }])

    def insert_vital_sign(
        self,
        measurement_id: str,
        patient_id: str,
        vital_type: str,
        value: float,
        unit: str,
        measured_at: datetime
    ) -> None:
        """Insert vital sign measurement"""
        query = """
            INSERT INTO vital_signs
            (measurement_id, patient_id, vital_type, value, unit, measured_at)
            VALUES
        """
        self.client.execute(query, [{
            'measurement_id': measurement_id,
            'patient_id': patient_id,
            'vital_type': vital_type,
            'value': value,
            'unit': unit,
            'measured_at': measured_at
        }])

    def get_patient_events(
        self,
        patient_id: str,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None
    ) -> List[Dict]:
        """Get patient events within time range"""
        query = "SELECT * FROM patient_events WHERE patient_id = %(patient_id)s"
        params = {'patient_id': patient_id}

        if start_time:
            query += " AND event_time >= %(start_time)s"
            params['start_time'] = start_time

        if end_time:
            query += " AND event_time <= %(end_time)s"
            params['end_time'] = end_time

        query += " ORDER BY event_time DESC"

        return self.client.execute(query, params)

    def get_vital_signs(
        self,
        patient_id: str,
        vital_type: Optional[str] = None,
        start_time: Optional[datetime] = None
    ) -> List[Dict]:
        """Get vital signs for patient"""
        query = "SELECT * FROM vital_signs WHERE patient_id = %(patient_id)s"
        params = {'patient_id': patient_id}

        if vital_type:
            query += " AND vital_type = %(vital_type)s"
            params['vital_type'] = vital_type

        if start_time:
            query += " AND measured_at >= %(start_time)s"
            params['start_time'] = start_time

        query += " ORDER BY measured_at DESC"

        return self.client.execute(query, params)

    def health_check(self) -> bool:
        """Check ClickHouse health"""
        try:
            self.client.execute("SELECT 1")
            return True
        except Exception as e:
            logger.error(f"ClickHouse health check failed: {e}")
            return False


class RedisStorage:
    """Redis storage client for caching and real-time data"""

    def __init__(self, config: StorageConfig):
        self.config = config
        self.client: Optional[Redis] = None

    def connect(self) -> None:
        """Establish Redis connection"""
        try:
            self.client = redis.Redis(
                host=self.config.redis_host,
                port=self.config.redis_port,
                db=self.config.redis_db,
                decode_responses=self.config.redis_decode_responses
            )
            # Test connection
            self.client.ping()
            logger.info(f"Connected to Redis: {self.config.redis_host}:{self.config.redis_port}")
        except Exception as e:
            logger.error(f"Redis connection failed: {e}")
            raise

    def close(self) -> None:
        """Close Redis connection"""
        if self.client:
            self.client.close()
            logger.info("Redis connection closed")

    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        """Set key-value with optional expiration (seconds)"""
        if isinstance(value, (dict, list)):
            value = json.dumps(value)
        return self.client.set(key, value, ex=ex)

    def get(self, key: str, as_json: bool = False) -> Any:
        """Get value by key"""
        value = self.client.get(key)
        if value and as_json:
            return json.loads(value)
        return value

    def delete(self, *keys: str) -> int:
        """Delete one or more keys"""
        return self.client.delete(*keys)

    def exists(self, key: str) -> bool:
        """Check if key exists"""
        return bool(self.client.exists(key))

    def incr(self, key: str) -> int:
        """Increment counter"""
        return self.client.incr(key)

    def expire(self, key: str, seconds: int) -> bool:
        """Set expiration time"""
        return self.client.expire(key, seconds)

    def hset(self, name: str, key: str, value: Any) -> int:
        """Set hash field"""
        if isinstance(value, (dict, list)):
            value = json.dumps(value)
        return self.client.hset(name, key, value)

    def hget(self, name: str, key: str, as_json: bool = False) -> Any:
        """Get hash field"""
        value = self.client.hget(name, key)
        if value and as_json:
            return json.loads(value)
        return value

    def hgetall(self, name: str) -> Dict:
        """Get all hash fields"""
        return self.client.hgetall(name)

    def cache_patient(self, patient_id: str, patient_data: Dict, ttl: int = 3600) -> bool:
        """Cache patient data with TTL"""
        return self.set(f"patient:{patient_id}", patient_data, ex=ttl)

    def get_cached_patient(self, patient_id: str) -> Optional[Dict]:
        """Get cached patient data"""
        return self.get(f"patient:{patient_id}", as_json=True)

    def health_check(self) -> bool:
        """Check Redis health"""
        try:
            return self.client.ping()
        except Exception as e:
            logger.error(f"Redis health check failed: {e}")
            return False


class Module8Storage:
    """
    Unified storage client for all Module 8 infrastructure

    Provides access to:
    - MongoDB (document storage)
    - Elasticsearch (search and analytics)
    - ClickHouse (time-series analytics)
    - Redis (caching)

    Usage:
        storage = Module8Storage()
        storage.connect_all()

        # MongoDB
        storage.mongo.patients.insert_one(patient)

        # Elasticsearch
        storage.es.index(index="events", document=event)

        # ClickHouse
        storage.clickhouse.execute("SELECT * FROM patient_events")

        # Redis
        storage.redis.set("key", "value")

        storage.close_all()
    """

    def __init__(self, config: Optional[StorageConfig] = None):
        self.config = config or StorageConfig()

        # Initialize storage clients
        self.mongo = MongoStorage(self.config)
        self.es = ElasticsearchStorage(self.config)
        self.clickhouse = ClickHouseStorage(self.config)
        self.redis = RedisStorage(self.config)

    def connect_all(self) -> None:
        """Connect to all storage services"""
        logger.info("Connecting to all Module 8 storage services...")

        try:
            self.mongo.connect()
            self.es.connect()
            self.clickhouse.connect()
            self.redis.connect()
            logger.info("All storage services connected successfully")
        except Exception as e:
            logger.error(f"Failed to connect to all services: {e}")
            raise

    def close_all(self) -> None:
        """Close all storage connections"""
        logger.info("Closing all Module 8 storage connections...")

        self.mongo.close()
        self.es.close()
        self.clickhouse.close()
        self.redis.close()

        logger.info("All storage connections closed")

    def health_check_all(self) -> Dict[str, bool]:
        """Check health of all storage services"""
        return {
            "mongodb": self.mongo.health_check(),
            "elasticsearch": self.es.health_check(),
            "clickhouse": self.clickhouse.health_check(),
            "redis": self.redis.health_check()
        }

    def __enter__(self):
        """Context manager entry"""
        self.connect_all()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit"""
        self.close_all()


# Convenience function
def get_storage(config: Optional[StorageConfig] = None) -> Module8Storage:
    """
    Get configured Module8Storage instance

    Args:
        config: Optional custom configuration

    Returns:
        Module8Storage instance
    """
    return Module8Storage(config)


if __name__ == "__main__":
    # Example usage
    logging.basicConfig(level=logging.INFO)

    # Using context manager
    with Module8Storage() as storage:
        # Health check
        health = storage.health_check_all()
        print(f"Health Status: {health}")

        # MongoDB example
        patient = {
            "patient_id": "P123",
            "name": "John Doe",
            "age": 45,
            "created_at": datetime.utcnow()
        }
        # storage.mongo.patients.insert_one(patient)

        # Elasticsearch example
        event = {
            "event_id": "E123",
            "event_type": "admission",
            "timestamp": datetime.utcnow().isoformat()
        }
        # storage.es.index(index="clinical_events", document=event)

        # ClickHouse example
        # results = storage.clickhouse.execute("SELECT count() FROM patient_events")
        # print(f"Patient events count: {results}")

        # Redis example
        storage.redis.set("test_key", "test_value", ex=60)
        value = storage.redis.get("test_key")
        print(f"Redis test: {value}")
