@echo off
echo ================================================================================
echo ========================= Starting FHIR Service ===============================
echo ================================================================================

REM Set the PYTHONPATH environment variable to include the backend directory
set PYTHONPATH=%~dp0..\..

echo Starting FHIR Service with the following configuration:
echo   PYTHONPATH: %PYTHONPATH%
echo   Running on port: 8014

REM Run the service using uvicorn
python -m uvicorn app.main:app --host 0.0.0.0 --port 8014 --reload
