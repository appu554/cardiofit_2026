"""
Configuration for Stage 2: Storage Fan-Out Service
"""

import os
from typing import Optional

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings"""
    
    # Service Configuration
    PORT: int = 8042
    DEBUG: bool = False
    SERVICE_NAME: str = "stage2-storage-fanout"
    
    # Kafka Configuration
    KAFKA_BOOTSTRAP_SERVERS: str = "pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    KAFKA_API_KEY: str = "LGJ3AQ2L6VRPW4S2"
    KAFKA_API_SECRET: str = ""
    KAFKA_SECURITY_PROTOCOL: str = "SASL_SSL"
    KAFKA_SASL_MECHANISM: str = "PLAIN"
    
    # Kafka Topics
    KAFKA_INPUT_TOPIC: str = "validated-device-data.v1"
    KAFKA_DLQ_TOPIC: str = "sink-write-failures.v1"
    KAFKA_CONSUMER_GROUP: str = "stage2-storage-fanout"
    
    # Kafka Consumer Settings
    KAFKA_AUTO_OFFSET_RESET: str = "latest"
    KAFKA_ENABLE_AUTO_COMMIT: bool = True
    KAFKA_AUTO_COMMIT_INTERVAL_MS: int = 1000
    KAFKA_MAX_POLL_RECORDS: int = 100
    KAFKA_SESSION_TIMEOUT_MS: int = 30000
    KAFKA_HEARTBEAT_INTERVAL_MS: int = 3000
    
    # Multi-Sink Configuration
    PARALLEL_WRITES: bool = True
    THREAD_POOL_SIZE: int = 6
    SINK_TIMEOUT_SECONDS: int = 30
    BATCH_SIZE: int = 50
    
    # FHIR Store Configuration (EXACT same as PySpark ETL)
    FHIR_STORE_ENABLED: bool = True
    GOOGLE_CLOUD_PROJECT: str = "cardiofit-905a8"
    GOOGLE_CLOUD_LOCATION: str = "asia-south1"
    GOOGLE_CLOUD_DATASET: str = "clinical-synthesis-hub"
    GOOGLE_CLOUD_FHIR_STORE: str = "fhir-store"
    GOOGLE_APPLICATION_CREDENTIALS: Optional[str] = None

    # Elasticsearch Configuration (EXACT same as PySpark ETL)
    ELASTICSEARCH_ENABLED: bool = True
    ELASTICSEARCH_URL: str = "https://my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud:443"
    ELASTICSEARCH_API_KEY: str = "d0gyTG5aY0JGajhWTVBOTzkzeDk6VGxoNENEd29DZEtERXBxRXpRUXBEUQ=="
    ELASTICSEARCH_INDEX_PREFIX: str = "patient-readings"  # Same as PySpark
    ELASTICSEARCH_TIMEOUT: int = 30

    # MongoDB Configuration (EXACT same as PySpark ETL)
    MONGODB_ENABLED: bool = True
    MONGODB_URI: str = "mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
    MONGODB_DATABASE: str = "clinical_synthesis_hub"
    MONGODB_COLLECTION: str = "device_readings"  # Same as PySpark
    MONGODB_TIMEOUT: int = 30
    
    # Redis Configuration (for coordination and caching)
    REDIS_HOST: str = "localhost"
    REDIS_PORT: int = 6379
    REDIS_PASSWORD: Optional[str] = None
    REDIS_DB: int = 1  # Different DB from Stage 1
    
    # Circuit Breaker Configuration
    CIRCUIT_BREAKER_ENABLED: bool = True
    CIRCUIT_BREAKER_FAILURE_THRESHOLD: int = 10
    CIRCUIT_BREAKER_RECOVERY_TIMEOUT: int = 60
    
    # Retry Configuration
    RETRY_ENABLED: bool = True
    RETRY_MAX_ATTEMPTS: int = 3
    RETRY_BACKOFF_FACTOR: float = 2.0
    RETRY_MAX_WAIT: int = 60
    
    # Monitoring Configuration
    PROMETHEUS_ENABLED: bool = True
    PROMETHEUS_PORT: int = 9042
    METRICS_ENABLED: bool = True
    
    # Health Check Configuration
    HEALTH_CHECK_INTERVAL: int = 30
    HEALTH_CHECK_TIMEOUT: int = 5
    
    # Logging Configuration
    LOG_LEVEL: str = "INFO"
    LOG_FORMAT: str = "json"
    
    # Performance Configuration
    MAX_CONCURRENT_REQUESTS: int = 100
    REQUEST_TIMEOUT: int = 30
    CONNECTION_POOL_SIZE: int = 20
    
    class Config:
        env_file = ".env"
        case_sensitive = True


# Create global settings instance
settings = Settings()


def get_kafka_config() -> dict:
    """Get Kafka configuration dictionary"""
    return {
        'bootstrap_servers': settings.KAFKA_BOOTSTRAP_SERVERS,
        'security_protocol': settings.KAFKA_SECURITY_PROTOCOL,
        'sasl_mechanism': settings.KAFKA_SASL_MECHANISM,
        'sasl_plain_username': settings.KAFKA_API_KEY,
        'sasl_plain_password': settings.KAFKA_API_SECRET,
        'group_id': settings.KAFKA_CONSUMER_GROUP,
        'auto_offset_reset': settings.KAFKA_AUTO_OFFSET_RESET,
        'enable_auto_commit': settings.KAFKA_ENABLE_AUTO_COMMIT,
        'auto_commit_interval_ms': settings.KAFKA_AUTO_COMMIT_INTERVAL_MS,
        'max_poll_records': settings.KAFKA_MAX_POLL_RECORDS,
        'session_timeout_ms': settings.KAFKA_SESSION_TIMEOUT_MS,
        'heartbeat_interval_ms': settings.KAFKA_HEARTBEAT_INTERVAL_MS,
        'value_deserializer': lambda x: x.decode('utf-8') if x else None,
        'key_deserializer': lambda x: x.decode('utf-8') if x else None
    }


def get_fhir_store_path() -> str:
    """Get Google Healthcare API FHIR store path"""
    return (f"projects/{settings.GOOGLE_CLOUD_PROJECT}/"
            f"locations/{settings.GOOGLE_CLOUD_LOCATION}/"
            f"datasets/{settings.GOOGLE_CLOUD_DATASET}/"
            f"fhirStores/{settings.GOOGLE_CLOUD_FHIR_STORE}")


def get_elasticsearch_config() -> dict:
    """Get Elasticsearch configuration dictionary - EXACT same as your PySpark implementation"""
    return {
        'hosts': [settings.ELASTICSEARCH_URL],
        'api_key': settings.ELASTICSEARCH_API_KEY,
        'request_timeout': settings.ELASTICSEARCH_TIMEOUT,  # Fixed: use request_timeout not timeout
        'max_retries': 3,
        'retry_on_timeout': True,
        'verify_certs': True,
        'ssl_show_warn': False
    }


def get_mongodb_config() -> dict:
    """Get MongoDB configuration dictionary"""
    return {
        'uri': settings.MONGODB_URI,
        'database': settings.MONGODB_DATABASE,
        'collection': settings.MONGODB_COLLECTION,
        'timeout': settings.MONGODB_TIMEOUT
    }
