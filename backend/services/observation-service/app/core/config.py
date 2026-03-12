import os
from pathlib import Path # Import Path
from typing import Any, Dict, List, Optional, Union
from pydantic import AnyHttpUrl, model_validator, validator, Field # Ensure both model_validator and validator are imported
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    PROJECT_NAME: str = "Observation Service"
    API_PREFIX: str = "/api"
    PORT: int = int(os.getenv("PORT", "8007"))

    # Define project root
    PROJECT_ROOT_DIR: Path = Path(__file__).resolve().parent.parent.parent

    # Google Healthcare API Settings
    USE_GOOGLE_HEALTHCARE_API: bool = os.getenv("USE_GOOGLE_HEALTHCARE_API", "false").lower() == "true"
    GOOGLE_CLOUD_PROJECT_ID: str = os.getenv("GOOGLE_CLOUD_PROJECT_ID", os.getenv("GOOGLE_CLOUD_PROJECT", ""))
    GOOGLE_CLOUD_LOCATION: str = os.getenv("GOOGLE_CLOUD_LOCATION", os.getenv("HEALTHCARE_LOCATION", "us-central1"))
    GOOGLE_CLOUD_DATASET_ID: str = os.getenv("GOOGLE_CLOUD_DATASET_ID", os.getenv("HEALTHCARE_DATASET", "clinical_synthesis_hub"))
    GOOGLE_CLOUD_FHIR_STORE_ID: str = os.getenv("GOOGLE_CLOUD_FHIR_STORE_ID", os.getenv("FHIR_STORE", "observations"))
    
    # Path to service account credentials (optional if using application default credentials)
    GOOGLE_CLOUD_CREDENTIALS_PATH: Optional[str] = os.getenv("GOOGLE_CLOUD_CREDENTIALS_PATH") # Aligned with patient-service
    
    # For backward compatibility
    GOOGLE_CLOUD_PROJECT: str = GOOGLE_CLOUD_PROJECT_ID
    HEALTHCARE_LOCATION: str = GOOGLE_CLOUD_LOCATION
    HEALTHCARE_DATASET: str = GOOGLE_CLOUD_DATASET_ID
    FHIR_STORE: str = GOOGLE_CLOUD_FHIR_STORE_ID
    
    # Auth Service URL for user authentication
    AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api")

    # CORS settings
    BACKEND_CORS_ORIGINS: List[AnyHttpUrl] = []

    @validator("BACKEND_CORS_ORIGINS", pre=True)
    def assemble_cors_origins(cls, v: Union[str, List[str]]) -> Union[List[str], str]:
        if isinstance(v, str) and not v.startswith("["):
            return [i.strip() for i in v.split(",")]
        elif isinstance(v, (list, str)):
            return v
        raise ValueError(v)
    
    @property
    def fhir_store_name(self) -> str:
        """Get the full FHIR store name in the format projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{fhirStore}"""
        return f"projects/{self.GOOGLE_CLOUD_PROJECT_ID}/locations/{self.GOOGLE_CLOUD_LOCATION}/datasets/{self.GOOGLE_CLOUD_DATASET_ID}/fhirStores/{self.GOOGLE_CLOUD_FHIR_STORE_ID}"

settings = Settings()
