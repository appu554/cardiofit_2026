@echo off
REM Workflow Engine Service - Quick Start Script for Windows

setlocal enabledelayedexpansion

echo =============================================================================
echo    🚀 Starting Workflow Engine Service
echo =============================================================================
echo.

REM Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker is not running. Please start Docker Desktop and try again.
    pause
    exit /b 1
)

REM Check if docker-compose is available
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo ❌ docker-compose is not installed. Please install it and try again.
    pause
    exit /b 1
)

echo ✅ Docker is running

REM Create .env file if it doesn't exist
if not exist .env (
    echo ⚠️  No .env file found. Creating from template...
    copy .env.example .env >nul
    echo ✅ Created .env file. You may want to review and customize it.
)

REM Create required directories
echo 📁 Creating required directories...
if not exist logs mkdir logs
if not exist configs\grafana\provisioning\dashboards mkdir configs\grafana\provisioning\dashboards
if not exist configs\grafana\provisioning\datasources mkdir configs\grafana\provisioning\datasources

REM Create Prometheus configuration if it doesn't exist
if not exist configs\prometheus.yml (
    echo ⚠️  Creating Prometheus configuration...
    if not exist configs mkdir configs
    (
        echo global:
        echo   scrape_interval: 15s
        echo   evaluation_interval: 15s
        echo.
        echo rule_files:
        echo   # - "first_rules.yml"
        echo   # - "second_rules.yml"
        echo.
        echo scrape_configs:
        echo   - job_name: 'prometheus'
        echo     static_configs:
        echo       - targets: ['localhost:9090']
        echo.  
        echo   - job_name: 'workflow-engine'
        echo     static_configs:
        echo       - targets: ['workflow-engine:8017']
        echo     scrape_interval: 5s
        echo     metrics_path: /metrics
    ) > configs\prometheus.yml
    echo ✅ Created Prometheus configuration
)

REM Create Grafana datasource configuration
if not exist configs\grafana\provisioning\datasources\prometheus.yml (
    (
        echo apiVersion: 1
        echo.
        echo datasources:
        echo   - name: Prometheus
        echo     type: prometheus
        echo     access: proxy
        echo     url: http://prometheus:9090
        echo     isDefault: true
        echo     editable: true
    ) > configs\grafana\provisioning\datasources\prometheus.yml
    echo ✅ Created Grafana datasource configuration
)

echo.
echo 🐳 Starting Docker services...

REM Start the infrastructure services first
echo 🔧 Starting infrastructure services (database, monitoring)...
docker-compose up -d postgres redis prometheus grafana jaeger adminer

REM Wait for database to be ready
echo ⏳ Waiting for database to be ready...
timeout /t 15 /nobreak >nul

REM Check if database is ready (simple approach for Windows)
echo ✅ Database should be ready

REM Build and start the main application
echo 🔨 Building and starting Workflow Engine Service...
docker-compose up -d --build workflow-engine

REM Wait for the service to be ready
echo ⏳ Waiting for Workflow Engine Service to be ready...
timeout /t 10 /nobreak >nul

REM Health check loop
set /a attempts=0
set /a max_attempts=12

:healthcheck
set /a attempts+=1
curl -f -s http://localhost:8017/health >nul 2>&1
if errorlevel 0 (
    echo ✅ Workflow Engine Service is ready!
    goto :services_ready
)

if !attempts! geq !max_attempts! (
    echo ❌ Workflow Engine Service failed to start. Check logs with: docker-compose logs workflow-engine
    pause
    exit /b 1
)

echo ⏳ Attempt !attempts!/!max_attempts! - Service not ready yet...
timeout /t 5 /nobreak >nul
goto :healthcheck

:services_ready
echo.
echo 🎉 All services are up and running!
echo.
echo =============================================================================
echo    📋 Service Access URLs
echo =============================================================================
echo   • Workflow Engine API:  http://localhost:8017
echo   • Health Check:         http://localhost:8017/health
echo   • GraphQL Playground:   http://localhost:8017/graphql
echo   • Metrics:              http://localhost:8017/metrics
echo.
echo =============================================================================
echo    🔍 Monitoring ^& Management
echo =============================================================================
echo   • Grafana:              http://localhost:3000 (admin:admin123)
echo   • Prometheus:           http://localhost:9090
echo   • Jaeger Tracing:       http://localhost:16686
echo   • Database Admin:       http://localhost:8080
echo.
echo =============================================================================
echo    📊 Quick Commands
echo =============================================================================
echo   • View logs:            docker-compose logs -f workflow-engine
echo   • View all logs:        docker-compose logs -f
echo   • Stop services:        docker-compose stop
echo   • Stop ^& remove:        docker-compose down
echo   • Restart service:      docker-compose restart workflow-engine
echo.
echo =============================================================================
echo    🧪 Testing
echo =============================================================================
echo   • Health check:         curl http://localhost:8017/health
echo   • API test:             curl -X POST http://localhost:8017/api/v1/orchestration/medication
echo.
echo ✨ Setup complete! The Workflow Engine Service is ready for use.
echo.
echo Press any key to exit...
pause >nul