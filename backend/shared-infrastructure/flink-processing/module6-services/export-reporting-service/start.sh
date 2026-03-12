#!/bin/bash

# Export & Reporting Service - Quick Start Script

echo "=================================="
echo "Export & Reporting Service"
echo "Module 6 - Components 6F and 6G"
echo "=================================="
echo ""

# Check if Maven is installed
if ! command -v mvn &> /dev/null; then
    echo "Error: Maven is not installed"
    exit 1
fi

# Check if PostgreSQL is running
echo "Checking PostgreSQL connection..."
if ! nc -z localhost 5433 2>/dev/null; then
    echo "Warning: PostgreSQL not detected on port 5433"
    echo "Please ensure the analytics database is running"
    echo ""
fi

# Build the project
echo "Building project..."
mvn clean install -DskipTests

if [ $? -ne 0 ]; then
    echo "Build failed"
    exit 1
fi

echo ""
echo "Build successful!"
echo ""

# Run the application
echo "Starting Export & Reporting Service on port 8050..."
echo ""
echo "Available endpoints:"
echo "  - Health: http://localhost:8050/api/export/health"
echo "  - Export Patients CSV: http://localhost:8050/api/export/patients/csv"
echo "  - Export Alerts CSV: http://localhost:8050/api/export/alerts/csv"
echo "  - Export Predictions JSON: http://localhost:8050/api/export/predictions/json"
echo "  - Export FHIR: http://localhost:8050/api/export/patients/fhir"
echo "  - Quality Report PDF: http://localhost:8050/api/export/reports/quality-metrics"
echo ""

mvn spring-boot:run
