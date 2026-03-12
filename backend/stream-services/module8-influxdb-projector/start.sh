#!/bin/bash
# Quick start script for InfluxDB Projector Service

echo "🚀 Starting InfluxDB Projector Service..."
echo ""

# Check if InfluxDB is running
if ! docker ps | grep -q "cardiofit-influxdb"; then
    echo "❌ InfluxDB container is not running!"
    echo "   Start it with: docker start cardiofit-influxdb"
    exit 1
fi

echo "✅ InfluxDB is running"

# Check Python dependencies
if ! python3 -c "import influxdb_client" 2>/dev/null; then
    echo "📦 Installing dependencies..."
    python3 -m pip install -r requirements.txt -q
fi

echo "✅ Dependencies installed"

# Verify environment variables
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    echo "   Copy .env.example to .env and configure"
    exit 1
fi

echo "✅ Configuration loaded"

# Start the service
echo ""
echo "🎯 Starting service on port 8054..."
echo "   Health check: http://localhost:8054/health"
echo "   Statistics: http://localhost:8054/stats"
echo ""

python3 run_service.py
