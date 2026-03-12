@echo off
REM Test script for KB-Drug-Rules Docker setup

setlocal enabledelayedexpansion

echo 🧪 Testing KB-Drug-Rules Docker Setup
echo ====================================
echo.

REM Check if services are running
echo [INFO] Checking if KB services are running...

docker ps --filter "name=kb-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not running or KB services are not started
    echo Please run: make run-kb-docker
    pause
    exit /b 1
)

echo.
echo [INFO] Running comprehensive tests...
echo.

REM Test 1: Health Check
echo ========================================
echo Test 1: Health Check
echo ========================================
curl -s http://localhost:8081/health
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] Health check passed!
) else (
    echo [ERROR] Health check failed
    goto :test_failed
)

echo.
echo.

REM Test 2: Get Metformin Rules
echo ========================================
echo Test 2: Get Metformin Rules
echo ========================================
curl -s http://localhost:8081/v1/items/metformin
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] Metformin rules retrieved!
) else (
    echo [ERROR] Failed to get metformin rules
    goto :test_failed
)

echo.
echo.

REM Test 3: Get Lisinopril Rules
echo ========================================
echo Test 3: Get Lisinopril Rules
echo ========================================
curl -s http://localhost:8081/v1/items/lisinopril
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] Lisinopril rules retrieved!
) else (
    echo [ERROR] Failed to get lisinopril rules
    goto :test_failed
)

echo.
echo.

REM Test 4: Get Warfarin Rules
echo ========================================
echo Test 4: Get Warfarin Rules
echo ========================================
curl -s http://localhost:8081/v1/items/warfarin
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] Warfarin rules retrieved!
) else (
    echo [ERROR] Failed to get warfarin rules
    goto :test_failed
)

echo.
echo.

REM Test 5: Validate TOML Rules
echo ========================================
echo Test 5: Validate TOML Rules
echo ========================================
curl -s -X POST http://localhost:8081/v1/validate ^
  -H "Content-Type: application/json" ^
  -d "{\"content\":\"[meta]\\ndrug_name=\\\"Test Drug\\\"\\ntherapeutic_class=[\\\"Test\\\"]\\n[dose_calculation]\\nbase_formula=\\\"100mg daily\\\"\\nmax_daily_dose=200.0\\nmin_daily_dose=50.0\\n[safety_verification]\\ncontraindications=[]\\nwarnings=[]\\nprecautions=[]\\ninteraction_checks=[]\\nlab_monitoring=[]\\nmonitoring_requirements=[]\\nregional_variations={}\",\"regions\":[\"US\"]}"
if %errorlevel% equ 0 (
    echo.
    echo [SUCCESS] TOML validation passed!
) else (
    echo [ERROR] TOML validation failed
    goto :test_failed
)

echo.
echo.

REM Test 6: Metrics Endpoint
echo ========================================
echo Test 6: Metrics Endpoint
echo ========================================
curl -s http://localhost:8081/metrics | findstr "kb_" >nul
if %errorlevel% equ 0 (
    echo [SUCCESS] Metrics endpoint working!
) else (
    echo [ERROR] Metrics endpoint failed
    goto :test_failed
)

echo.
echo.

REM Test 7: Database Connection
echo ========================================
echo Test 7: Database Connection
echo ========================================
docker exec kb-postgres pg_isready -U postgres >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] PostgreSQL is ready!
) else (
    echo [ERROR] PostgreSQL connection failed
    goto :test_failed
)

REM Test 8: Redis Connection
echo ========================================
echo Test 8: Redis Connection
echo ========================================
docker exec kb-redis redis-cli ping >nul 2>&1
if %errorlevel% equ 0 (
    echo [SUCCESS] Redis is ready!
) else (
    echo [ERROR] Redis connection failed
    goto :test_failed
)

echo.
echo.

REM All tests passed
echo ========================================
echo 🎉 ALL TESTS PASSED!
echo ========================================
echo.
echo Your KB-Drug-Rules Docker setup is working perfectly!
echo.
echo Services running:
echo   ✅ KB-Drug-Rules API:    http://localhost:8081
echo   ✅ PostgreSQL:          localhost:5433
echo   ✅ Redis:               localhost:6380
echo   ✅ Adminer:             http://localhost:8082
echo.
echo Sample data available:
echo   ✅ Metformin (diabetes medication)
echo   ✅ Lisinopril (blood pressure medication)
echo   ✅ Warfarin (anticoagulant)
echo.
echo Ready for Flow2 integration! 🚀
echo.
echo Next steps:
echo   1. Integrate with your Flow2 orchestrator
echo   2. Add more drug rules via API
echo   3. Monitor performance with /metrics
echo   4. Use Adminer for database management
echo.
goto :end

:test_failed
echo.
echo ========================================
echo ❌ TESTS FAILED
echo ========================================
echo.
echo Some tests failed. Please check:
echo   1. Are all services running? docker ps
echo   2. Check service logs: make logs-kb
echo   3. Restart services: make stop-kb && make run-kb-docker
echo.
echo For troubleshooting, see: README-DOCKER-KB.md
echo.

:end
pause
