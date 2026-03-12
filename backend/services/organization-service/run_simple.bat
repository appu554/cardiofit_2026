@echo off
echo Setting up environment for Google Cloud Healthcare API...

REM Set environment variables (matching Patient Service configuration)
set USE_GOOGLE_HEALTHCARE_API=true
set GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
set GOOGLE_CLOUD_LOCATION=asia-south1
set GOOGLE_CLOUD_DATASET_ID=clinical-synthesis-hub
set GOOGLE_CLOUD_FHIR_STORE_ID=fhir-store
set GOOGLE_CLOUD_CREDENTIALS_PATH=credentials/service-account-key.json

REM Run the Organization service
echo Starting Organization service with Google Cloud Healthcare API...
uvicorn app.main:app --reload --port 8012
