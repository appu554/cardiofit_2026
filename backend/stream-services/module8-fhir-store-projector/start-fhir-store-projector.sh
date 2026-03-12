#!/bin/bash
# FHIR Store Projector Startup Script

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Starting FHIR Store Projector${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if .env exists
if [ ! -f ".env" ]; then
    echo -e "${RED}ERROR: .env file not found${NC}"
    echo "Please create .env from .env.example"
    exit 1
fi

# Check if module8-shared is available
if [ ! -d "../module8-shared" ]; then
    echo -e "${RED}ERROR: module8-shared not found${NC}"
    exit 1
fi

# Create log directory
mkdir -p logs

# Set log file
LOG_FILE="logs/fhir-store-projector-$(date +%Y%m%d-%H%M%S).log"

echo -e "${YELLOW}Configuration:${NC}"
echo "  - Kafka: localhost:9092 (PLAINTEXT)"
echo "  - Consumer Group: module8-fhir-store-projector"
echo "  - Topic: prod.ehr.fhir.upsert"
echo "  - Service Port: 8056"
echo "  - Log File: $LOG_FILE"
echo ""

# Export Python path
export PYTHONPATH="${SCRIPT_DIR}:${SCRIPT_DIR}/../module8-shared:${PYTHONPATH}"

echo -e "${GREEN}Starting service...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Start the service
python3 run.py 2>&1 | tee "$LOG_FILE"
