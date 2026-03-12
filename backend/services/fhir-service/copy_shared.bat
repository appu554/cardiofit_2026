@echo off
REM Copy the shared directory to the FHIR service

echo Copying shared directory to FHIR service...
xcopy /E /I /Y ..\..\shared app\shared

echo Done!
pause
