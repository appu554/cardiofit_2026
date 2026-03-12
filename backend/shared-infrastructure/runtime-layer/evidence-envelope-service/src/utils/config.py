"""
Configuration management for Evidence Envelope Service
"""

import os
from typing import List
from pydantic import BaseSettings, validator


class Settings(BaseSettings):
    """Application settings with environment variable support"""

    # Service Configuration
    SERVICE_NAME: str = "evidence-envelope-service"
    VERSION: str = "1.0.0"
    DEBUG: bool = False
    PORT: int = 8020

    # Authentication
    JWT_SECRET_KEY: str = "your-jwt-secret-key-change-in-production"
    JWT_ALGORITHM: str = "HS256"
    JWT_EXPIRATION_HOURS: int = 24

    # CORS Configuration
    ALLOWED_ORIGINS: List[str] = ["http://localhost:3000", "http://localhost:4000"]

    @validator('ALLOWED_ORIGINS', pre=True)
    def parse_allowed_origins(cls, v):
        if isinstance(v, str):
            return [origin.strip() for origin in v.split(',')]
        return v

    # Redis Configuration
    REDIS_HOST: str = "localhost"
    REDIS_PORT: int = 6379
    REDIS_PASSWORD: str = ""
    REDIS_DB: int = 0
    REDIS_MAX_CONNECTIONS: int = 20
    REDIS_RETRY_ON_TIMEOUT: bool = True

    # Kafka Configuration
    KAFKA_BOOTSTRAP_SERVERS: str = "localhost:9092"
    KAFKA_TOPIC_AUDIT_EVENTS: str = "audit-events"
    KAFKA_TOPIC_ENVELOPE_EVENTS: str = "envelope-events"
    KAFKA_CLIENT_ID: str = "evidence-envelope-service"
    KAFKA_COMPRESSION_TYPE: str = "gzip"
    KAFKA_BATCH_SIZE: int = 16384
    KAFKA_LINGER_MS: int = 10
    KAFKA_ACKS: str = "all"

    # Database Configuration
    MONGODB_CONNECTION_STRING: str = "mongodb://localhost:27017"
    MONGODB_DATABASE_NAME: str = "evidence_envelopes"
    MONGODB_MAX_CONNECTIONS: int = 100
    MONGODB_MIN_CONNECTIONS: int = 10

    # Performance Configuration
    ENVELOPE_CACHE_SIZE: int = 1000
    ENVELOPE_CACHE_TTL: int = 300  # 5 minutes
    MAX_INFERENCE_STEPS: int = 50
    MAX_ENVELOPE_SIZE_MB: int = 10

    # Regulatory Compliance
    AUDIT_RETENTION_DAYS: int = 2555  # 7 years for HIPAA
    INTEGRITY_CHECK_INTERVAL: int = 86400  # Daily
    CHECKSUM_ALGORITHM: str = "sha256"

    # Monitoring
    PROMETHEUS_METRICS_PORT: int = 8021
    LOG_LEVEL: str = "INFO"
    STRUCTURED_LOGGING: bool = True

    class Config:
        env_file = ".env"
        case_sensitive = True


# Global settings instance
settings = Settings()