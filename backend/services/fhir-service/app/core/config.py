import os
from pydantic_settings import BaseSettings
from dotenv import load_dotenv

load_dotenv()

class Settings(BaseSettings):
    # API Configuration
    API_PREFIX: str = "/api"
    PROJECT_NAME: str = "FHIR Integration Layer"
    DEBUG: bool = True

    # MongoDB Configuration
    MONGODB_URI: str = os.getenv("MONGODB_URI", "mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0")

    # Auth0 Configuration
    AUTH0_DOMAIN: str = os.getenv("AUTH0_DOMAIN", "")
    AUTH0_API_AUDIENCE: str = os.getenv("AUTH0_API_AUDIENCE", "")

    # FHIR Configuration
    FHIR_VERSION: str = "R4"  # FHIR Release 4 (the most widely used version)

    # HL7 Configuration
    HL7_VERSION: str = "2.5"  # Default HL7 version
    HL7_PROCESSING_ID: str = "P"  # P for Production, T for Testing, D for Development
    HL7_RECEIVING_APPLICATION: str = "Clinical-Synthesis-Hub"
    HL7_RECEIVING_FACILITY: str = "CSH"

    # Microservice URLs - Updated to match the correct ports
    FHIR_SERVICE_URL: str = os.getenv("FHIR_SERVICE_URL", "http://localhost:8014")  # Updated to match API Gateway config
    PATIENT_SERVICE_URL: str = os.getenv("PATIENT_SERVICE_URL", "http://localhost:8003")
    OBSERVATION_SERVICE_URL: str = os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8007")
    NOTES_SERVICE_URL: str = os.getenv("NOTES_SERVICE_URL", "http://localhost:8006")
    MEDICATION_SERVICE_URL: str = os.getenv("MEDICATION_SERVICE_URL", "http://localhost:8009")
    IMAGING_SERVICE_URL: str = os.getenv("IMAGING_SERVICE_URL", "http://localhost:8007")
    PROBLEM_LIST_SERVICE_URL: str = os.getenv("PROBLEM_LIST_SERVICE_URL", "http://localhost:8008")
    CONDITION_SERVICE_URL: str = os.getenv("CONDITION_SERVICE_URL", "http://localhost:8010")
    ENCOUNTER_SERVICE_URL: str = os.getenv("ENCOUNTER_SERVICE_URL", "http://localhost:8011")
    LAB_SERVICE_URL: str = os.getenv("LAB_SERVICE_URL", "http://localhost:8000")
    TIMELINE_SERVICE_URL: str = os.getenv("TIMELINE_SERVICE_URL", "http://localhost:8012")  # Updated to port 8012 to match Timeline service

settings = Settings()
