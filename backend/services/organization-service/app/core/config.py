import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "Organization Service"
    API_PREFIX: str = "/api"

    # Service Configuration
    SERVICE_PORT: int = int(os.getenv("ORGANIZATION_SERVICE_PORT", "8012"))
    SERVICE_HOST: str = os.getenv("ORGANIZATION_SERVICE_HOST", "0.0.0.0")
    DEBUG: bool = os.getenv("DEBUG", "false").lower() == "true"

    # Google Healthcare API Configuration (matching your actual setup)
    GOOGLE_CLOUD_PROJECT_ID: str = os.getenv("GOOGLE_CLOUD_PROJECT_ID", "cardiofit-905a8")
    GOOGLE_CLOUD_LOCATION: str = os.getenv("GOOGLE_CLOUD_LOCATION", "asia-south1")
    GOOGLE_CLOUD_DATASET_ID: str = os.getenv("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub")
    GOOGLE_CLOUD_FHIR_STORE_ID: str = os.getenv("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store")
    GOOGLE_CLOUD_CREDENTIALS_PATH: str = os.getenv("GOOGLE_CLOUD_CREDENTIALS_PATH", "credentials/service-account-key.json")

    # Auth Service Configuration
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")

    class Config:
        env_file = ".env"
        case_sensitive = True

settings = Settings()
