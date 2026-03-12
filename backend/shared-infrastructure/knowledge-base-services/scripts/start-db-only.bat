@echo off
REM Start only database services for KB-Drug-Rules
REM Run the KB service locally with Go

setlocal enabledelayedexpansion

echo 🗄️  Starting KB Database Services Only
echo ====================================
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

REM Stop any existing containers
echo [INFO] Stopping any existing KB containers...
docker-compose -f docker-compose.db-only.yml down >nul 2>&1

REM Start only database services
echo [INFO] Starting database services...
echo This will start:
echo   - PostgreSQL on port 5433
echo   - Redis on port 6380  
echo   - Adminer (database UI) on port 8082
echo.

docker-compose -f docker-compose.db-only.yml up -d

if %errorlevel% neq 0 (
    echo [ERROR] Failed to start database services
    echo Check Docker logs for details
    pause
    exit /b 1
)

echo [SUCCESS] Database services are starting...
echo.

REM Wait for services to be ready
echo [INFO] Waiting for services to be ready...
timeout /t 15 /nobreak >nul

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

echo.
echo 🎉 Database Services Started Successfully!
echo =========================================
echo.
echo Services available at:
echo   🗄️  PostgreSQL:          localhost:5433
echo   🗄️  Redis:               localhost:6380
echo   🗄️  Database UI (Adminer): http://localhost:8082
echo.
echo Database connection details:
echo   Host:     localhost
echo   Port:     5433
echo   Database: kb_drug_rules
echo   Username: kb_drug_rules_user
echo   Password: kb_password
echo.
echo Next steps:
echo   1. Navigate to kb-drug-rules directory
echo   2. Initialize Go module: go mod init kb-drug-rules
echo   3. Download dependencies: go mod tidy
echo   4. Set environment variables and run service
echo.
echo Environment variables to set:
echo   set DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules
echo   set REDIS_URL=redis://localhost:6380/0
echo   set PORT=8081
echo   set DEBUG=true
echo.
echo Then run: go run cmd/server/main.go
echo.

REM Test database connection
echo [INFO] Testing database connection...
timeout /t 5 /nobreak >nul

docker exec kb-postgres psql -U kb_drug_rules_user -d kb_drug_rules -c "SELECT COUNT(*) as drug_count FROM drug_rule_packs;" >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] Database connection and sample data verified!
    docker exec kb-postgres psql -U kb_drug_rules_user -d kb_drug_rules -c "SELECT drug_id, version FROM drug_rule_packs;"
) else (
    echo [INFO] Database is still initializing, sample data will be available shortly
)

echo.
echo 🎯 Database Setup Complete!
echo.
echo Your database services are ready. Now run the KB service:
echo.
echo   cd kb-drug-rules
echo   go mod init kb-drug-rules
echo   go mod tidy
echo   set DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules
echo   set REDIS_URL=redis://localhost:6380/0
echo   set PORT=8081
echo   set DEBUG=true
echo   go run cmd/server/main.go
echo.

pause
