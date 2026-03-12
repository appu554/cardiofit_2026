"""
Configuration settings for Device Data Ingestion Service
"""
import os
from typing import Optional

try:
    # Try new pydantic-settings package first
    from pydantic_settings import BaseSettings
except ImportError:
    # Fallback to old pydantic BaseSettings
    try:
        from pydantic import BaseSettings
    except ImportError:
        # If neither works, create a simple alternative
        class BaseSettings:
            def __init__(self, **kwargs):
                for key, value in kwargs.items():
                    setattr(self, key, value)


class Settings(BaseSettings):
    """Application settings"""
    
    # Service Configuration
    PROJECT_NAME: str = "Device Data Ingestion Service"
    VERSION: str = "1.0.0"
    API_PREFIX: str = "/api/v1"
    
    # Server Configuration
    HOST: str = "0.0.0.0"
    PORT: int = 8030
    DEBUG: bool = False
    
    # Kafka Configuration
    KAFKA_BOOTSTRAP_SERVERS: str = "pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    KAFKA_API_KEY: str = "LGJ3AQ2L6VRPW4S2"
    KAFKA_API_SECRET: str = os.getenv("KAFKA_API_SECRET", "2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl")
    KAFKA_TOPIC_DEVICE_DATA: str = "raw-device-data.v1"
    
    # Authentication Configuration
    AUTH_SERVICE_URL: str = "http://localhost:8001"
    API_KEY_HEADER: str = "X-API-Key"
    
    # Rate Limiting Configuration
    RATE_LIMIT_PER_MINUTE: int = 1000
    RATE_LIMIT_PER_DEVICE_PER_MINUTE: int = 100
    
    # Monitoring Configuration
    ENABLE_METRICS: bool = True
    METRICS_PORT: int = 9090

    # Supabase Configuration (matching other services)
    SUPABASE_URL: str = os.getenv("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
    SUPABASE_KEY: str = os.getenv("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8")
    SUPABASE_JWT_SECRET: str = os.getenv("SUPABASE_JWT_SECRET", "nXwqv86rPXO5HqJ1R1xeQnhy9JbeLeLypwUZmMoJ1prMGG6io5lU88nD6lG8MmvpN7Z2pZJvfuF33Z1x2PwCoA==")

    # Database Configuration (Supabase PostgreSQL for transactional outbox)
    # Using Supabase pooled connection for free tier
    # Password: 9FTqQnA4LRCsu8sw (no special characters, no encoding needed)
    DATABASE_URL: str = os.getenv(
        "DATABASE_URL",
        "postgresql://postgres.auugxeqzgrnknklgwqrh:9FTqQnA4LRCsu8sw@aws-0-ap-south-1.pooler.supabase.com:5432/postgres"
    )

    # Async Database URL for asyncpg (using pooled connection)
    ASYNC_DATABASE_URL: str = os.getenv(
        "ASYNC_DATABASE_URL",
        "postgresql+asyncpg://postgres.auugxeqzgrnknklgwqrh:9FTqQnA4LRCsu8sw@aws-0-ap-south-1.pooler.supabase.com:5432/postgres"
    )

    # Fallback mode for network issues
    ENABLE_FALLBACK_MODE: bool = os.getenv("ENABLE_FALLBACK_MODE", "true").lower() == "true"
    FALLBACK_STORAGE_PATH: str = os.getenv("FALLBACK_STORAGE_PATH", "./fallback_outbox")

    # Outbox Configuration
    OUTBOX_BATCH_SIZE: int = 50
    OUTBOX_POLL_INTERVAL: int = 5  # seconds
    MAX_CONCURRENT_VENDORS: int = 10
    OUTBOX_RETRY_BACKOFF_SECONDS: int = 60

    # Google Cloud Monitoring (for cloud-native metrics)
    GCP_PROJECT_ID: str = os.getenv("GCP_PROJECT_ID", "cardiofit-905a8")
    ENABLE_CLOUD_METRICS: bool = True

    class Config:
        env_file = ".env"
        case_sensitive = True


# Global settings instance
settings = Settings()
