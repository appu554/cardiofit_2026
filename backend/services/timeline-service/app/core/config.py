import os
from typing import List
from pydantic import AnyHttpUrl
from pydantic_settings import BaseSettings
from dotenv import load_dotenv

load_dotenv()

class Settings(BaseSettings):
    PROJECT_NAME: str = "Timeline Service"
    API_PREFIX: str = "/api"
    PORT: int = int(os.getenv("PORT", "8012"))  # Updated to port 8012 to match API Gateway config

    # MongoDB settings
    MONGODB_URL: str = os.getenv("MONGODB_URL", "mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0")
    MONGODB_DB_NAME: str = os.getenv("MONGODB_DB_NAME", "clinical_synthesis_hub")

    # Service URLs - Updated to match the correct ports
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8014")  # Updated to port 8014 to match API Gateway config
    OBSERVATION_SERVICE_URL: str = os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8007")
    CONDITION_SERVICE_URL: str = os.getenv("CONDITION_SERVICE_URL", "http://localhost:8010")
    MEDICATION_SERVICE_URL: str = os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8009")
    ENCOUNTER_SERVICE_URL: str = os.getenv("ENCOUNTER_SERVICE_URL", "http://localhost:8011")  # Updated to port 8011
    DOCUMENT_SERVICE_URL: str = os.getenv("DOCUMENT_SERVICE_URL", "http://localhost:8008")
    LAB_SERVICE_URL: str = os.getenv("LAB_SERVICE_URL", "http://localhost:8000")

    # Auth Service URL
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api")

    # CORS settings
    BACKEND_CORS_ORIGINS: List[AnyHttpUrl] = []

settings = Settings()
