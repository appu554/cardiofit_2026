import os
from pydantic_settings import BaseSettings
from dotenv import load_dotenv

load_dotenv()

class Settings(BaseSettings):
    # Service URLs
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8000")
    USER_SERVICE_URL: str = os.getenv("USER_SERVICE_URL", "http://localhost:8001")
    PATIENT_SERVICE_URL: str = os.getenv("PATIENT_SERVICE_URL", "http://localhost:8002")
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8004")
    OBSERVATION_SERVICE_URL: str = os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8007")
    NOTES_SERVICE_URL: str = os.getenv("NOTES_SERVICE_URL", "http://localhost:8008")
    LABS_SERVICE_URL: str = os.getenv("LABS_SERVICE_URL", "http://localhost:8009")
    MEDICATION_SERVICE_URL: str = os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8018")
    CONDITION_SERVICE_URL: str = os.getenv("CONDITION_SERVICE_URL", "http://localhost:8019")
    ENCOUNTER_SERVICE_URL: str = os.getenv("ENCOUNTER_SERVICE_URL", "http://localhost:8020")
    TIMELINE_SERVICE_URL: str = os.getenv("TIMELINE_SERVICE_URL", "http://localhost:8010")

    # API Configuration
    API_PREFIX: str = "/api"
    PROJECT_NAME: str = "Clinical Synthesis Hub"
    DEBUG: bool = True

settings = Settings()
