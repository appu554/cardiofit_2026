import os
from pydantic_settings import BaseSettings
from dotenv import load_dotenv

load_dotenv()

class Settings(BaseSettings):
    # Service URLs
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")
    USER_SERVICE_URL: str = os.getenv("USER_SERVICE_URL", "http://localhost:8002")
    PATIENT_SERVICE_URL: str = os.getenv("PATIENT_SERVICE_URL", "http://localhost:8003")
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8014")
    NOTES_SERVICE_URL: str = os.getenv("NOTES_SERVICE_URL", "http://localhost:8006")
    LABS_SERVICE_URL: str = os.getenv("LABS_SERVICE_URL", "http://localhost:8007")
    OBSERVATION_SERVICE_URL: str = os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8008")
    MEDICATION_SERVICE_URL: str = os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8009")
    CONDITION_SERVICE_URL: str = os.getenv("CONDITION_SERVICE_URL", "http://localhost:8010")
    ENCOUNTER_SERVICE_URL: str = os.getenv("ENCOUNTER_SERVICE_URL", "http://localhost:8020")
    TIMELINE_SERVICE_URL: str = os.getenv("TIMELINE_SERVICE_URL", "http://localhost:8012")
    WORKFLOW_ENGINE_SERVICE_URL: str = os.getenv("WORKFLOW_ENGINE_SERVICE_URL", "http://localhost:8015")
    GRAPHQL_SERVICE_URL: str = os.getenv("GRAPHQL_SERVICE_URL", "http://localhost:8005")

    # Vaidshala Clinical Runtime Services
    INGESTION_SERVICE_URL: str = os.getenv("INGESTION_SERVICE_URL", "http://localhost:8140")
    INTAKE_SERVICE_URL: str = os.getenv("INTAKE_SERVICE_URL", "http://localhost:8141")

    # Apollo Federation Gateway URL
    APOLLO_FEDERATION_URL: str = os.getenv("APOLLO_FEDERATION_URL", "http://localhost:4000/graphql")

    # API Configuration
    API_PREFIX: str = "/api"
    PROJECT_NAME: str = "Clinical Synthesis Hub API Gateway"
    DEBUG: bool = True

    # Gateway Configuration
    ENABLE_REQUEST_LOGGING: bool = os.getenv("ENABLE_REQUEST_LOGGING", "True").lower() in ("true", "1", "t")
    LOG_REQUEST_BODY: bool = os.getenv("LOG_REQUEST_BODY", "False").lower() in ("true", "1", "t")
    LOG_RESPONSE_BODY: bool = os.getenv("LOG_RESPONSE_BODY", "False").lower() in ("true", "1", "t")

    # Rate Limiting
    RATE_LIMIT_ENABLED: bool = os.getenv("RATE_LIMIT_ENABLED", "False").lower() in ("true", "1", "t")
    RATE_LIMIT_REQUESTS: int = int(os.getenv("RATE_LIMIT_REQUESTS", "100"))
    RATE_LIMIT_WINDOW: int = int(os.getenv("RATE_LIMIT_WINDOW", "60"))  # seconds

settings = Settings()