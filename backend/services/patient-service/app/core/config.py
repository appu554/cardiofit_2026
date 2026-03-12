import os
from typing import List, Optional
from pydantic import Field
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "Patient Service"
    API_PREFIX: str = "/api"
    PORT: int = int(os.getenv("PORT", "8003"))

    # MongoDB settings (legacy, will be removed)
    MONGODB_URL: str = Field(default=os.getenv("MONGODB_URL", "mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"))
    MONGODB_DB_NAME: str = Field(default=os.getenv("MONGODB_DB_NAME", "clinical_synthesis_hub"))

    # Google Cloud Healthcare API settings
    USE_GOOGLE_HEALTHCARE_API: bool = Field(default=os.getenv("USE_GOOGLE_HEALTHCARE_API", "true").lower() == "true")
    GOOGLE_CLOUD_PROJECT_ID: str = Field(default=os.getenv("GOOGLE_CLOUD_PROJECT_ID", ""))
    GOOGLE_CLOUD_LOCATION: str = Field(default=os.getenv("GOOGLE_CLOUD_LOCATION", "us-central1"))
    GOOGLE_CLOUD_DATASET_ID: str = Field(default=os.getenv("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub"))
    GOOGLE_CLOUD_FHIR_STORE_ID: str = Field(default=os.getenv("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store"))
    GOOGLE_CLOUD_CREDENTIALS_PATH: Optional[str] = Field(default=os.getenv("GOOGLE_CLOUD_CREDENTIALS_PATH"))

    # FHIR Service URL
    FHIR_SERVICE_URL: str = Field(default=os.getenv("FHIR_SERVICE_URL", "http://localhost:8004"))

    # Auth Service URL
    AUTH_SERVICE_URL: str = Field(default=os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api"))

    # CORS settings
    BACKEND_CORS_ORIGINS: List[str] = Field(default=["*"])

settings = Settings()
