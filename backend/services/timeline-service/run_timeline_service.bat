@echo off
REM Run the Timeline Service

REM Check if Python is installed
where python >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Python is not installed or not in PATH
    exit /b 1
)

REM Check if the virtual environment exists, if not create it
if not exist venv (
    echo Creating virtual environment...
    python -m venv venv
)

REM Set environment variables
set AUTH_SERVICE_URL=http://localhost:8001
set FHIR_SERVICE_URL=http://localhost:8014/api
set OBSERVATION_SERVICE_URL=http://localhost:8007/api
set CONDITION_SERVICE_URL=http://localhost:8010/api
set MEDICATION_SERVICE_URL=http://localhost:8009/api
set ENCOUNTER_SERVICE_URL=http://localhost:8011/api
set DOCUMENT_SERVICE_URL=http://localhost:8008/api
set LAB_SERVICE_URL=http://localhost:8000/api
set PORT=8012

REM Activate the virtual environment and install dependencies
call venv\Scripts\activate.bat
pip install -r requirements.txt

REM Run the service
echo Starting Timeline Service on port %PORT%...
uvicorn app.main:app --host 0.0.0.0 --port %PORT% --reload

REM Deactivate the virtual environment when done
call venv\Scripts\deactivate.bat
