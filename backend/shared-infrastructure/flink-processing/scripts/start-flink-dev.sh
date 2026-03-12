#!/bin/bash

# CardioFit Flink EHR Intelligence Engine - Development Startup Script

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
KAFKA_DIR="${PROJECT_ROOT}/../../kafka"

echo "=========================================="
echo "CardioFit Flink EHR Intelligence Engine"
echo "Development Environment Setup"
echo "=========================================="

# Function to check if a service is running
check_service() {
    local service=$1
    local port=$2
    nc -z localhost $port 2>/dev/null
    return $?
}

# Step 1: Check if Kafka is running
echo ""
echo "Step 1: Checking Kafka Infrastructure..."
if ! check_service "Kafka" 9092; then
    echo "⚠️  Kafka is not running. Starting Kafka infrastructure..."
    cd "$KAFKA_DIR"
    if [ -f "./start-kafka.sh" ]; then
        ./start-kafka.sh
        echo "⏳ Waiting for Kafka to be ready..."
        sleep 20
    else
        echo "❌ Error: start-kafka.sh not found in $KAFKA_DIR"
        echo "Please start Kafka manually first."
        exit 1
    fi
else
    echo "✅ Kafka is already running on port 9092"
fi

# Step 2: Build the Flink application
echo ""
echo "Step 2: Building Flink Application..."
cd "$PROJECT_ROOT"

if [ ! -f "pom.xml" ]; then
    echo "❌ Error: pom.xml not found. Please ensure you're in the correct directory."
    exit 1
fi

# Check if Maven is installed
if ! command -v mvn &> /dev/null; then
    echo "❌ Error: Maven is not installed. Please install Maven first."
    exit 1
fi

echo "📦 Running Maven build..."
mvn clean package -DskipTests

if [ $? -ne 0 ]; then
    echo "❌ Error: Maven build failed"
    exit 1
fi

echo "✅ Build successful!"

# Step 3: Create necessary directories
echo ""
echo "Step 3: Creating necessary directories..."
mkdir -p "$PROJECT_ROOT/checkpoints"
mkdir -p "$PROJECT_ROOT/savepoints"
mkdir -p "$PROJECT_ROOT/logs"
mkdir -p "$PROJECT_ROOT/config/grafana/dashboards"
mkdir -p "$PROJECT_ROOT/config/grafana/datasources"

# Step 4: Create Grafana datasource configuration
echo ""
echo "Step 4: Configuring Grafana datasources..."
cat > "$PROJECT_ROOT/config/grafana/datasources/prometheus.yml" <<EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
EOF

echo "✅ Grafana datasource configured"

# Step 5: Start Flink cluster
echo ""
echo "Step 5: Starting Flink Cluster..."
cd "$PROJECT_ROOT"

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed. Please install Docker first."
    exit 1
fi

if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker daemon is not running. Please start Docker."
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: docker-compose is not installed. Please install docker-compose first."
    exit 1
fi

echo "🚀 Starting Flink containers..."
docker-compose down 2>/dev/null || true
docker-compose up -d

# Wait for services to start
echo ""
echo "⏳ Waiting for services to start..."
sleep 10

# Step 6: Health check
echo ""
echo "Step 6: Performing health checks..."

# Check Flink JobManager
if check_service "Flink JobManager" 8081; then
    echo "✅ Flink JobManager is running (Web UI: http://localhost:8081)"
else
    echo "⚠️  Flink JobManager may still be starting..."
fi

# Check Prometheus
if check_service "Prometheus" 9090; then
    echo "✅ Prometheus is running (UI: http://localhost:9090)"
else
    echo "⚠️  Prometheus may still be starting..."
fi

# Check Grafana
if check_service "Grafana" 3001; then
    echo "✅ Grafana is running (UI: http://localhost:3001)"
    echo "   Username: admin, Password: admin"
else
    echo "⚠️  Grafana may still be starting..."
fi

# Step 7: Display service URLs
echo ""
echo "=========================================="
echo "🎉 Flink EHR Intelligence Engine Started!"
echo "=========================================="
echo ""
echo "Service URLs:"
echo "  📊 Flink Web UI:      http://localhost:8081"
echo "  📈 Prometheus:        http://localhost:9090"
echo "  📉 Grafana:          http://localhost:3001"
echo "  🔧 Kafka UI:         http://localhost:8080"
echo ""
echo "Useful Commands:"
echo "  View logs:           docker-compose logs -f [service-name]"
echo "  Stop all services:  ./scripts/stop-flink-dev.sh"
echo "  Submit a job:       ./scripts/submit-job.sh"
echo "  Monitor status:     ./scripts/check-status.sh"
echo ""
echo "Next Steps:"
echo "  1. Open Flink Web UI to monitor cluster status"
echo "  2. Submit Flink jobs using the submit-job.sh script"
echo "  3. Monitor metrics in Grafana dashboard"
echo ""