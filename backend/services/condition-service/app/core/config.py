import os
from typing import List
from pydantic import AnyHttpUrl
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "Condition Service"
    API_PREFIX: str = "/api"
    PORT: int = int(os.getenv("PORT", "8010"))

    # MongoDB settings
    MONGODB_URL: str = os.getenv("MONGODB_URL", "mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0")
    MONGODB_DB_NAME: str = os.getenv("MONGODB_DB_NAME", "clinical_synthesis_hub")

    # FHIR Service URL
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8004")

    # Auth Service URL
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api")

    # CORS settings
    BACKEND_CORS_ORIGINS: List[AnyHttpUrl] = []

settings = Settings()
