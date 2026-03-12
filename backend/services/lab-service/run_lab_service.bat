@echo off
echo ================================================================================
echo ========================= Starting Lab Service ================================
echo ================================================================================

REM Set the PYTHONPATH environment variable to include the backend directory
set PYTHONPATH=%~dp0..\..

echo Starting Lab Service with the following configuration:
echo   PYTHONPATH: %PYTHONPATH%
echo   Running on port: 8000

REM Set environment variables
set AUTH_SERVICE_URL=http://localhost:8001
set FHIR_SERVICE_URL=http://localhost:8004/api
set MONGODB_URL=mongodb+srv://admin:Apoorva@554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0
set MONGODB_DB_NAME=clinical_synthesis_hub

REM Run the service using uvicorn
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
