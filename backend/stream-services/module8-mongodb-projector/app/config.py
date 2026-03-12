"""Configuration for MongoDB Projector service."""

from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    """Application settings."""

    # Service
    service_name: str = "mongodb-projector"
    service_port: int = 8051

    # Kafka Configuration
    kafka_bootstrap_servers: str = "localhost:9092"
    kafka_topic: str = "prod.ehr.events.enriched"
    kafka_group_id: str = "mongodb-projector-group"
    kafka_auto_offset_reset: str = "earliest"
    kafka_enable_auto_commit: bool = False
    kafka_max_poll_records: int = 100
    kafka_session_timeout_ms: int = 30000

    # MongoDB Configuration
    mongodb_uri: str = "mongodb://localhost:27017"
    mongodb_database: str = "module8_clinical"
    mongodb_max_pool_size: int = 50
    mongodb_min_pool_size: int = 10
    mongodb_connect_timeout_ms: int = 5000
    mongodb_server_selection_timeout_ms: int = 5000

    # Batch Processing
    batch_size: int = 50
    batch_timeout_seconds: int = 10
    max_retries: int = 3
    retry_delay_seconds: int = 5

    # Patient Timeline Settings
    max_events_per_patient: int = 1000

    class Config:
        env_file = ".env"
        case_sensitive = False


@lru_cache()
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()
