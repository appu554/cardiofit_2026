"""
Configuration management for L1 Cache and Prefetcher Service
"""

import os
from typing import List
from pydantic import BaseSettings, validator


class Settings(BaseSettings):
    """Application settings with environment variable support"""

    # Service Configuration
    SERVICE_NAME: str = "l1-cache-prefetcher-service"
    VERSION: str = "1.0.0"
    DEBUG: bool = False
    PORT: int = 8030

    # L1 Cache Configuration
    L1_CACHE_SIZE_MB: int = 512
    L1_CACHE_DEFAULT_TTL: int = 10  # seconds
    L1_CACHE_MAX_ENTRIES: int = 50000
    MEMORY_PRESSURE_THRESHOLD: float = 0.85

    # ML Prefetch Configuration
    ML_MODEL_UPDATE_INTERVAL_HOURS: int = 6
    ML_MIN_TRAINING_SAMPLES: int = 1000
    ML_PREDICTION_HORIZON_HOURS: int = 1

    # Prefetch Configuration
    MAX_CONCURRENT_FETCHES: int = 20
    PREFETCH_BUDGET_MB: int = 100
    MIN_CONFIDENCE_THRESHOLD: float = 0.6

    # Authentication
    JWT_SECRET_KEY: str = "l1-cache-jwt-secret-key-change-in-production"
    JWT_ALGORITHM: str = "HS256"
    JWT_EXPIRATION_HOURS: int = 24

    # CORS Configuration
    ALLOWED_ORIGINS: List[str] = ["http://localhost:3000", "http://localhost:4000"]

    @validator('ALLOWED_ORIGINS', pre=True)
    def parse_allowed_origins(cls, v):
        if isinstance(v, str):
            return [origin.strip() for origin in v.split(',')]
        return v

    # External Data Sources
    PATIENT_SERVICE_URL: str = "http://localhost:8003"
    CLINICAL_SERVICE_URL: str = "http://localhost:8010"
    MEDICATION_SERVICE_URL: str = "http://localhost:8004"
    GUIDELINE_SERVICE_URL: str = "http://localhost:8081"
    SEMANTIC_SERVICE_URL: str = "http://localhost:8090"

    # Redis Configuration (for L2 cache if needed)
    REDIS_HOST: str = "localhost"
    REDIS_PORT: int = 6379
    REDIS_PASSWORD: str = ""
    REDIS_DB: int = 1  # Different DB from evidence envelope service

    # Performance Configuration
    REQUEST_TIMEOUT_MS: int = 5000
    MAX_REQUEST_SIZE_MB: int = 10
    WORKER_THREADS: int = 4

    # Monitoring
    PROMETHEUS_METRICS_PORT: int = 8031
    LOG_LEVEL: str = "INFO"
    STRUCTURED_LOGGING: bool = True

    # ML Model Storage
    MODEL_STORAGE_PATH: str = "/app/models"
    MODEL_BACKUP_ENABLED: bool = True

    class Config:
        env_file = ".env"
        case_sensitive = True


# Global settings instance
settings = Settings()