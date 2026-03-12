@echo off
echo Starting Python Environment Validation...
echo.

REM Try different Python commands
echo Trying 'py' command...
py --version
if %errorlevel% == 0 (
    echo Success with 'py' command
    py validate_setup.py
    goto :end
)

echo Trying 'python' command...
python --version
if %errorlevel% == 0 (
    echo Success with 'python' command
    python validate_setup.py
    goto :end
)

echo Trying direct path...
"C:\Users\apoor\AppData\Local\Microsoft\WindowsApps\PythonSoftwareFoundation.Python.3.12_qbz5n2kfra8p0\python.exe" --version
if %errorlevel% == 0 (
    echo Success with direct path
    "C:\Users\apoor\AppData\Local\Microsoft\WindowsApps\PythonSoftwareFoundation.Python.3.12_qbz5n2kfra8p0\python.exe" validate_setup.py
    goto :end
)

echo Could not find Python executable
echo Please check your Python installation

:end
pause
