#!/bin/bash
# Start RxNav-in-a-Box for CardioFit Phase 3
#
# First run will download ~3GB of RxNorm data (takes 30-60 minutes)
# Subsequent starts are fast (<30 seconds)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "  CardioFit RxNav-in-a-Box Startup"
echo "=========================================="

# Create network if it doesn't exist
docker network create cardiofit-kb-network 2>/dev/null || true

# Check if this is first run
if ! docker volume inspect cardiofit-rxnav-data >/dev/null 2>&1; then
    echo ""
    echo "⚠️  FIRST RUN DETECTED"
    echo "   RxNorm data download will begin (~3GB)"
    echo "   This may take 30-60 minutes..."
    echo ""
fi

# Start RxNav
echo "Starting RxNav-in-a-Box..."
docker-compose up -d

echo ""
echo "Waiting for RxNav to be ready..."
echo "(This may take several minutes on first run)"
echo ""

# Wait for health check
MAX_WAIT=900  # 15 minutes
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -sf http://localhost:4000/REST/version >/dev/null 2>&1; then
        echo ""
        echo "✅ RxNav-in-a-Box is ready!"
        echo ""
        echo "API Base URL: http://localhost:4000"
        echo ""
        echo "Quick Test:"
        echo "  curl 'http://localhost:4000/REST/rxcui.json?name=metformin'"
        echo ""

        # Show version
        echo "Version Info:"
        curl -s 'http://localhost:4000/REST/version' | head -20
        exit 0
    fi

    sleep 10
    WAITED=$((WAITED + 10))
    echo "  ... waiting ($WAITED seconds elapsed)"
done

echo "❌ RxNav failed to start within $MAX_WAIT seconds"
echo "   Check logs: docker-compose logs rxnav"
exit 1
