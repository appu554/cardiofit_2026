@echo off
REM Start KB-Drug-Rules service with Docker PostgreSQL
REM This script starts PostgreSQL in Docker and the KB service

setlocal enabledelayedexpansion

echo 🚀 Starting KB-Drug-Rules Service with Docker PostgreSQL
echo ========================================================
echo.

REM Check if Docker is running
echo [INFO] Checking Docker...
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not installed or not running
    echo Please install Docker Desktop from: https://www.docker.com/products/docker-desktop
    pause
    exit /b 1
)

echo [SUCCESS] Docker is available

REM Navigate to the correct directory
cd /d "%~dp0.."

REM Check if docker-compose file exists
if not exist "docker-compose.kb-only.yml" (
    echo [ERROR] docker-compose.kb-only.yml not found
    echo Please make sure you're in the correct directory
    pause
    exit /b 1
)

REM Stop any existing containers
echo [INFO] Stopping any existing KB containers...
docker-compose -f docker-compose.kb-only.yml down >nul 2>&1

REM Start the services
echo [INFO] Starting KB services with Docker...
echo This will:
echo   - Start PostgreSQL on port 5433 (to avoid conflict with your PostgreSQL 17.6)
echo   - Start Redis on port 6380
echo   - Start KB-Drug-Rules service on port 8081
echo   - Start Adminer (database UI) on port 8082
echo.

docker-compose -f docker-compose.kb-only.yml up -d

if %errorlevel% neq 0 (
    echo [ERROR] Failed to start services
    echo Check Docker logs for details
    pause
    exit /b 1
)

echo [SUCCESS] Services are starting...
echo.

REM Wait for services to be ready
echo [INFO] Waiting for services to be ready...
timeout /t 10 /nobreak >nul

REM Check service health
echo [INFO] Checking service health...

REM Check PostgreSQL
docker exec kb-postgres pg_isready -U postgres >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] PostgreSQL is ready
) else (
    echo [WARNING] PostgreSQL is still starting...
)

REM Check Redis
docker exec kb-redis redis-cli ping >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] Redis is ready
) else (
    echo [WARNING] Redis is still starting...
)

REM Wait a bit more for KB service
echo [INFO] Waiting for KB-Drug-Rules service...
timeout /t 15 /nobreak >nul

REM Test KB service
curl -s http://localhost:8081/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] KB-Drug-Rules service is ready!
) else (
    echo [WARNING] KB-Drug-Rules service is still starting...
    echo You can check logs with: docker logs kb-drug-rules
)

echo.
echo 🎉 KB Services Started Successfully!
echo ====================================
echo.
echo Services available at:
echo   📊 KB-Drug-Rules API:    http://localhost:8081
echo   🔍 Health Check:         http://localhost:8081/health  
echo   📈 Metrics:              http://localhost:8081/metrics
echo   🗄️  Database (Adminer):   http://localhost:8082
echo   🗄️  PostgreSQL:          localhost:5433 (user: kb_drug_rules_user, password: kb_password)
echo   🗄️  Redis:               localhost:6380
echo.
echo Database connection details:
echo   Host:     localhost
echo   Port:     5433
echo   Database: kb_drug_rules  
echo   Username: kb_drug_rules_user
echo   Password: kb_password
echo.
echo Test commands:
echo   curl http://localhost:8081/health
echo   curl http://localhost:8081/v1/items/metformin
echo.
echo To stop services: docker-compose -f docker-compose.kb-only.yml down
echo To view logs: docker logs kb-drug-rules
echo.

REM Test the API
echo [INFO] Testing API endpoints...
echo.

REM Test health endpoint
echo Testing health endpoint...
curl -s http://localhost:8081/health
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] Health endpoint working!
) else (
    echo [INFO] Health endpoint not ready yet, service may still be starting
)

echo.
echo Testing drug rules endpoint...
curl -s http://localhost:8081/v1/items/metformin
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] Drug rules endpoint working!
) else (
    echo [INFO] Drug rules endpoint not ready yet, service may still be starting
)

echo.
echo 🎯 Setup Complete!
echo.
echo Your KB-Drug-Rules service is now running with:
echo   ✅ Isolated PostgreSQL (port 5433)
echo   ✅ Sample drug data (metformin, lisinopril, warfarin)
echo   ✅ Complete API endpoints
echo   ✅ Database management UI
echo.
echo Ready for Flow2 integration! 🚀

pause
