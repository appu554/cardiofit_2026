@echo off
REM KB-7 Terminology Service - Data Loading Script for Windows
REM Loads SNOMED CT, RxNorm, and LOINC datasets into the terminology database

setlocal enabledelayedexpansion

REM Configuration
set "SCRIPT_DIR=%~dp0"
set "DATA_DIR=%SCRIPT_DIR%data"
set "ETL_CMD=%SCRIPT_DIR%cmd\etl\main.go"

REM Default values
set "SYSTEMS=snomed,rxnorm,loinc"
set "BATCH_SIZE=10000"
set "WORKERS=4"
set "VALIDATE_ONLY=false"
set "DEBUG=false"
set "FORCE=false"

REM Parse command line arguments
:parse_args
if "%~1"=="" goto args_parsed
if "%~1"=="--systems" (
    set "SYSTEMS=%~2"
    shift & shift
    goto parse_args
)
if "%~1"=="--batch-size" (
    set "BATCH_SIZE=%~2"
    shift & shift
    goto parse_args
)
if "%~1"=="--workers" (
    set "WORKERS=%~2"
    shift & shift
    goto parse_args
)
if "%~1"=="--validate-only" (
    set "VALIDATE_ONLY=true"
    shift
    goto parse_args
)
if "%~1"=="--debug" (
    set "DEBUG=true"
    shift
    goto parse_args
)
if "%~1"=="--force" (
    set "FORCE=true"
    shift
    goto parse_args
)
if "%~1"=="--help" (
    goto show_usage
)
echo Unknown option: %~1
goto show_usage

:args_parsed

echo.
echo KB-7 Terminology Data Loading Script
echo ======================================

REM Check prerequisites
echo [INFO] Checking prerequisites...

REM Check if Go is installed
go version >nul 2>&1
if !errorlevel! neq 0 (
    echo [ERROR] Go is not installed or not in PATH
    exit /b 1
)

REM Check if data directory exists
if not exist "%DATA_DIR%" (
    echo [ERROR] Data directory not found: %DATA_DIR%
    exit /b 1
)

REM Check if ETL tool exists
if not exist "%ETL_CMD%" (
    echo [ERROR] ETL tool not found: %ETL_CMD%
    exit /b 1
)

echo [SUCCESS] Prerequisites check passed

REM Validate data directories
echo [INFO] Validating data directories...

REM Parse systems
set "SYSTEM_LIST=%SYSTEMS:,= %"
for %%s in (%SYSTEM_LIST%) do (
    if "%%s"=="snomed" (
        if not exist "%DATA_DIR%\snomed" (
            echo [ERROR] SNOMED CT data directory not found: %DATA_DIR%\snomed
            exit /b 1
        )
    ) else if "%%s"=="rxnorm" (
        if not exist "%DATA_DIR%\rxnorm" (
            echo [ERROR] RxNorm data directory not found: %DATA_DIR%\rxnorm
            exit /b 1
        )
    ) else if "%%s"=="loinc" (
        if not exist "%DATA_DIR%\loinc" (
            echo [ERROR] LOINC data directory not found: %DATA_DIR%\loinc
            exit /b 1
        )
    ) else (
        echo [ERROR] Unsupported system: %%s
        exit /b 1
    )
)

echo [SUCCESS] Data directories validation passed

REM Show data summary
echo.
echo [INFO] Data Summary:
echo ==============
if "%SYSTEMS%" == "*snomed*" (
    echo SNOMED CT:
    echo   - Location: %DATA_DIR%\snomed
)
if "%SYSTEMS%" == "*rxnorm*" (
    echo RxNorm:
    echo   - Location: %DATA_DIR%\rxnorm
)
if "%SYSTEMS%" == "*loinc*" (
    echo LOINC:
    echo   - Location: %DATA_DIR%\loinc
)
echo.

if "%VALIDATE_ONLY%"=="true" (
    echo [INFO] Validation mode - no data will be loaded
)

REM Load each system
set "SUCCESS_COUNT=0"
set "FAILED_SYSTEMS="

for %%s in (%SYSTEM_LIST%) do (
    echo [INFO] Processing system: %%s
    call :load_system %%s
    if !errorlevel! equ 0 (
        set /a SUCCESS_COUNT+=1
    ) else (
        set "FAILED_SYSTEMS=!FAILED_SYSTEMS! %%s"
    )
    echo.
)

REM Summary
echo [INFO] Loading Summary:
echo ================
echo [SUCCESS] Successfully loaded: %SUCCESS_COUNT% systems

if not "%FAILED_SYSTEMS%"=="" (
    echo [ERROR] Failed systems: %FAILED_SYSTEMS%
    exit /b 1
) else (
    echo [SUCCESS] All systems loaded successfully!
)

goto :eof

:load_system
set "system=%~1"
set "data_path="

if "%system%"=="snomed" (
    REM Try snapshot first, then extracted
    if exist "%DATA_DIR%\snomed\snapshot\sct2_Concept_Snapshot_INT.txt" (
        set "data_path=%DATA_DIR%\snomed\snapshot"
    ) else if exist "%DATA_DIR%\snomed\extracted" (
        REM Find the extracted SNOMED directory
        for /d %%d in ("%DATA_DIR%\snomed\extracted\SnomedCT_*") do (
            set "data_path=%%d\Snapshot\Terminology"
            goto snomed_found
        )
        echo [ERROR] Could not find SNOMED extracted data
        exit /b 1
        :snomed_found
    ) else (
        echo [ERROR] Could not find SNOMED data files
        exit /b 1
    )
) else if "%system%"=="rxnorm" (
    REM Try rrf first, then extracted
    if exist "%DATA_DIR%\rxnorm\rrf\RXNCONSO.RRF" (
        set "data_path=%DATA_DIR%\rxnorm\rrf"
    ) else if exist "%DATA_DIR%\rxnorm\extracted\rrf" (
        set "data_path=%DATA_DIR%\rxnorm\extracted\rrf"
    ) else (
        echo [ERROR] Could not find RxNorm data files
        exit /b 1
    )
) else if "%system%"=="loinc" (
    if exist "%DATA_DIR%\loinc\snapshot" (
        set "data_path=%DATA_DIR%\loinc\snapshot"
    ) else (
        echo [ERROR] Could not find LOINC data files
        exit /b 1
    )
)

echo [INFO] Loading %system% from: !data_path!

REM Build ETL command
set "etl_args=--data=!data_path! --system=%system% --batch-size=%BATCH_SIZE% --workers=%WORKERS%"

if "%VALIDATE_ONLY%"=="true" (
    set "etl_args=!etl_args! --validate-only"
)

if "%DEBUG%"=="true" (
    set "etl_args=!etl_args! --debug"
)

if "%FORCE%"=="true" (
    set "etl_args=!etl_args! --force"
)

REM Execute ETL command
echo [INFO] Running: go run "%ETL_CMD%" !etl_args!

go run "%ETL_CMD%" !etl_args!
if !errorlevel! equ 0 (
    echo [SUCCESS] Successfully loaded %system%
) else (
    echo [ERROR] Failed to load %system%
    exit /b 1
)

goto :eof

:show_usage
echo.
echo KB-7 Terminology Data Loading Script
echo.
echo Usage: %~n0 [OPTIONS]
echo.
echo Options:
echo     --systems SYSTEMS     Comma-separated list of systems to load (default: snomed,rxnorm,loinc)
echo     --batch-size SIZE     Number of records per batch (default: 10000)
echo     --workers NUM         Number of concurrent workers (default: 4)
echo     --validate-only       Only validate data files, don't load
echo     --debug               Enable debug logging
echo     --force               Force reload even if data exists
echo     --help                Show this help message
echo.
echo Examples:
echo     # Load all systems with default settings
echo     %~n0
echo.
echo     # Load only SNOMED CT with debug logging
echo     %~n0 --systems snomed --debug
echo.
echo     # Validate all data files without loading
echo     %~n0 --validate-only
echo.
echo     # Force reload with custom batch size
echo     %~n0 --force --batch-size 5000
echo.
echo Supported Systems:
echo     - snomed   : SNOMED CT International Edition
echo     - rxnorm   : RxNorm drug terminology
echo     - loinc    : LOINC laboratory codes
echo.
exit /b 0