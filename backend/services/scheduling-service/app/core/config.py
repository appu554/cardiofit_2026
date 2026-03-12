import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "Scheduling Service"
    API_PREFIX: str = "/api"
    PORT: int = int(os.getenv("PORT", "8014"))

    # Database settings - Using Google Healthcare API
    USE_GOOGLE_HEALTHCARE_API: bool = True

    # Google Healthcare API settings (matching other services)
    GOOGLE_CLOUD_PROJECT: str = os.getenv("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
    GOOGLE_CLOUD_LOCATION: str = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
    GOOGLE_CLOUD_DATASET: str = os.getenv("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
    GOOGLE_CLOUD_FHIR_STORE: str = os.getenv("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")
    GOOGLE_APPLICATION_CREDENTIALS: str = os.getenv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")

    # Auth Service Configuration
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")

    # CORS Configuration
    BACKEND_CORS_ORIGINS: str = os.getenv("BACKEND_CORS_ORIGINS", "http://localhost:3000,http://localhost:8000,http://localhost:8005")

    class Config:
        env_file = ".env"
        case_sensitive = True

settings = Settings()
