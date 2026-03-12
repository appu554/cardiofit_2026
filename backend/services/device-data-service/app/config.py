"""
Configuration for Device Data Service
"""
import os
from pydantic import BaseSettings


class Settings(BaseSettings):
    """Application settings"""
    
    # Service Configuration
    PROJECT_NAME: str = "Device Data Service"
    VERSION: str = "1.0.0"
    API_PREFIX: str = "/api/v1"
    
    # Server Configuration
    HOST: str = "0.0.0.0"
    PORT: int = 8016
    DEBUG: bool = False
    
    # Elasticsearch Configuration
    ELASTICSEARCH_URL: str = "http://localhost:9200"
    ELASTICSEARCH_INDEX: str = "patient-readings"
    
    # Google Healthcare API Configuration
    GOOGLE_CLOUD_PROJECT: str = "cardiofit-905a8"
    GOOGLE_CLOUD_LOCATION: str = "asia-south1"
    GOOGLE_CLOUD_DATASET: str = "clinical-synthesis-hub"
    GOOGLE_CLOUD_FHIR_STORE: str = "fhir-store"
    
    # Authentication
    AUTH_SERVICE_URL: str = "http://localhost:8001"
    
    class Config:
        env_file = ".env"
        case_sensitive = True


# Global settings instance
settings = Settings()
