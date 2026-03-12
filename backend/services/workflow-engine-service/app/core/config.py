"""
Configuration settings for the Workflow Engine Service.
"""
import os
from typing import Optional, List
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings."""

    # Service Configuration
    SERVICE_NAME: str = "workflow-engine-service"
    SERVICE_VERSION: str = "1.0.0"
    SERVICE_PORT: int = int(os.getenv("SERVICE_PORT", "8017"))
    DEBUG: bool = True

    # Supabase Configuration
    SUPABASE_URL: str = os.getenv("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
    SUPABASE_KEY: str = os.getenv("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8")
    SUPABASE_JWT_SECRET: str = os.getenv("SUPABASE_JWT_SECRET", "nXwqv86rPXO5HqJ1R1xeQnhy9JbeLeLypwUZmMoJ1prMGG6io5lU88nD6lG8MmvpN7Z2pZJvfuF33Z1x2PwCoA==")
    SUPABASE_ALGORITHMS: List[str] = ["HS256"]

    # Database Configuration (Supabase PostgreSQL for workflow state)
    # Using direct database connection format (not pooler)
    DATABASE_URL: str = os.getenv(
        "DATABASE_URL",
        "postgresql://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
    )
    
    # Google Cloud Healthcare API Configuration
    USE_GOOGLE_HEALTHCARE_API: bool = True
    GOOGLE_CLOUD_PROJECT: str = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
    GOOGLE_CLOUD_LOCATION: str = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
    GOOGLE_CLOUD_DATASET: str = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
    GOOGLE_CLOUD_FHIR_STORE: str = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")
    GOOGLE_APPLICATION_CREDENTIALS: str = os.getenv(
        "GOOGLE_APPLICATION_CREDENTIALS", 
        "credentials/google-credentials.json"
    )
    
    # Authentication Configuration
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")
    
    # External Service URLs
    PATIENT_SERVICE_URL: str = os.getenv("PATIENT_SERVICE_URL", "http://localhost:8003")
    MEDICATION_SERVICE_URL: str = os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8009")
    ORDER_SERVICE_URL: str = os.getenv("ORDER_SERVICE_URL", "http://localhost:8013")
    SCHEDULING_SERVICE_URL: str = os.getenv("SCHEDULING_SERVICE_URL", "http://localhost:8014")
    ENCOUNTER_SERVICE_URL: str = os.getenv("ENCOUNTER_SERVICE_URL", "http://localhost:8020")
    
    # Workflow Engine Configuration
    # Local Camunda Configuration
    CAMUNDA_ENGINE_URL: str = os.getenv("CAMUNDA_ENGINE_URL", "http://localhost:8080/engine-rest")

    # Camunda Cloud Configuration
    USE_CAMUNDA_CLOUD: bool = os.getenv("USE_CAMUNDA_CLOUD", "false").lower() == "true"
    CAMUNDA_CLOUD_CLIENT_ID: str = os.getenv("CAMUNDA_CLOUD_CLIENT_ID", "")
    CAMUNDA_CLOUD_CLIENT_SECRET: str = os.getenv("CAMUNDA_CLOUD_CLIENT_SECRET", "")
    CAMUNDA_CLOUD_CLUSTER_ID: str = os.getenv("CAMUNDA_CLOUD_CLUSTER_ID", "")
    CAMUNDA_CLOUD_REGION: str = os.getenv("CAMUNDA_CLOUD_REGION", "")
    CAMUNDA_CLOUD_AUTHORIZATION_SERVER_URL: str = os.getenv("CAMUNDA_CLOUD_AUTHORIZATION_SERVER_URL", "https://login.cloud.camunda.io/oauth/token")

    WORKFLOW_EXECUTION_TIMEOUT: int = 3600  # 1 hour default timeout
    TASK_ASSIGNMENT_TIMEOUT: int = 86400    # 24 hours default task timeout
    
    # Event Configuration
    EVENT_POLLING_INTERVAL: int = 30  # seconds
    TASK_POLLING_INTERVAL: int = 10   # seconds

    # Phase 4 Integration Configuration
    WORKFLOW_MOCK_MODE: bool = os.getenv("WORKFLOW_MOCK_MODE", "false").lower() == "true"
    WORKFLOW_ENABLE_WEBHOOKS: bool = os.getenv("WORKFLOW_ENABLE_WEBHOOKS", "true").lower() == "true"
    WORKFLOW_ENABLE_FHIR_MONITORING: bool = os.getenv("WORKFLOW_ENABLE_FHIR_MONITORING", "true").lower() == "true"
    WORKFLOW_ENABLE_EVENT_STORE: bool = os.getenv("WORKFLOW_ENABLE_EVENT_STORE", "false").lower() == "true"

    # Logging Configuration
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")
    
    class Config:
        env_file = ".env"
        case_sensitive = True


# Global settings instance
settings = Settings()
