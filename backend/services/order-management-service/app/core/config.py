import os
from typing import Any, Dict, List, Optional, Union
from pydantic import AnyHttpUrl, field_validator
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "Order Management Service"
    API_PREFIX: str = "/api"
    PORT: int = int(os.getenv("PORT", "8013"))

    # Database settings - Using Google Healthcare API
    USE_GOOGLE_HEALTHCARE_API: bool = True

    # Google Healthcare API settings
    GOOGLE_CLOUD_PROJECT: str = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
    GOOGLE_CLOUD_LOCATION: str = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
    GOOGLE_CLOUD_DATASET: str = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
    GOOGLE_CLOUD_FHIR_STORE: str = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")
    GOOGLE_APPLICATION_CREDENTIALS: str = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")

    # Service URLs
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8004")
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api")

    # CORS settings
    BACKEND_CORS_ORIGINS: List[AnyHttpUrl] = []

    @field_validator("BACKEND_CORS_ORIGINS", mode="before")
    def assemble_cors_origins(cls, v: Union[str, List[str]]) -> Union[List[str], str]:
        if isinstance(v, str) and not v.startswith("["):
            return [i.strip() for i in v.split(",")]
        elif isinstance(v, (list, str)):
            return v
        raise ValueError(v)

settings = Settings()
