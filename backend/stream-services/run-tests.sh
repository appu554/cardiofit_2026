#!/bin/bash

# Comprehensive Testing Script for Stage 1 & Stage 2
# This script helps you run and test both services

set -e

echo "🧪 Stage 1 & Stage 2 Testing Suite"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check Java for Stage 1
    if ! command -v java &> /dev/null; then
        print_error "Java not found. Please install Java 17+ for Stage 1"
        exit 1
    fi
    
    # Check Maven for Stage 1
    if ! command -v mvn &> /dev/null; then
        print_error "Maven not found. Please install Maven for Stage 1"
        exit 1
    fi
    
    # Check Python for Stage 2
    if ! command -v python3 &> /dev/null; then
        print_error "Python 3 not found. Please install Python 3.11+ for Stage 2"
        exit 1
    fi
    
    # Check pip
    if ! command -v pip &> /dev/null; then
        print_error "pip not found. Please install pip for Stage 2"
        exit 1
    fi
    
    print_status "Prerequisites check passed ✅"
}

# Setup Kafka topics
setup_kafka_topics() {
    print_step "Setting up Kafka topics..."
    
    if [ -f "setup-kafka-topics.sh" ]; then
        chmod +x setup-kafka-topics.sh
        print_status "Kafka credentials already configured, running topic setup..."
        ./setup-kafka-topics.sh
    else
        print_error "setup-kafka-topics.sh not found"
        exit 1
    fi
}

# Build Stage 1
build_stage1() {
    print_step "Building Stage 1 (Validator & Enricher)..."
    
    cd stage1-validator-enricher
    
    print_status "Compiling Java application..."
    mvn clean compile
    
    print_status "Running tests..."
    mvn test || print_warning "Some tests failed, continuing..."
    
    print_status "Packaging application..."
    mvn package -DskipTests
    
    cd ..
    print_status "Stage 1 build completed ✅"
}

# Setup Stage 2
setup_stage2() {
    print_step "Setting up Stage 2 (Storage Fan-Out)..."
    
    cd stage2-storage-fanout
    
    print_status "Installing Python dependencies..."
    pip install -r requirements.txt
    
    cd ..
    print_status "Stage 2 setup completed ✅"
}

# Start Stage 1
start_stage1() {
    print_step "Starting Stage 1 (Validator & Enricher) on port 8041..."
    
    cd stage1-validator-enricher
    
    print_status "Starting Spring Boot application..."
    print_warning "This will run in the foreground. Open a new terminal for Stage 2."
    
    # Set environment variables
    export KAFKA_BOOTSTRAP_SERVERS="pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    export KAFKA_API_KEY="LGJ3AQ2L6VRPW4S2"
    export KAFKA_API_SECRET="2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"
    export REDIS_HOST="localhost"
    export PATIENT_SERVICE_URL="http://localhost:8003/api/v1/patient"
    
    mvn spring-boot:run -Dspring-boot.run.profiles=dev
}

# Start Stage 2
start_stage2() {
    print_step "Starting Stage 2 (Storage Fan-Out) on port 8042..."
    
    cd stage2-storage-fanout
    
    print_status "Starting FastAPI application..."
    
    # Set environment variables
    export KAFKA_BOOTSTRAP_SERVERS="pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    export KAFKA_API_KEY="LGJ3AQ2L6VRPW4S2"
    export KAFKA_API_SECRET="2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"
    export GOOGLE_APPLICATION_CREDENTIALS="/path/to/gcp-credentials.json"
    export ELASTICSEARCH_URL="https://your-elasticsearch-url:443"
    export ELASTICSEARCH_API_KEY="your-elasticsearch-api-key"
    export MONGODB_URI="mongodb://localhost:27017"
    
    python -m uvicorn app.main:app --host 0.0.0.0 --port 8042 --reload
}

# Health checks
check_health() {
    print_step "Performing health checks..."
    
    # Check Stage 1
    print_status "Checking Stage 1 health..."
    if curl -f http://localhost:8041/api/v1/health > /dev/null 2>&1; then
        print_status "Stage 1 is healthy ✅"
    else
        print_error "Stage 1 health check failed ❌"
    fi
    
    # Check Stage 2
    print_status "Checking Stage 2 health..."
    if curl -f http://localhost:8042/api/v1/health > /dev/null 2>&1; then
        print_status "Stage 2 is healthy ✅"
    else
        print_error "Stage 2 health check failed ❌"
    fi
}

# Send test data
send_test_data() {
    print_step "Sending test data..."
    
    if [ -f "test-data-generator.py" ]; then
        print_status "Kafka credentials already configured, running test data generator..."
        python3 test-data-generator.py
    else
        print_error "test-data-generator.py not found"
    fi
}

# Monitor topics
monitor_topics() {
    print_step "Monitoring Kafka topics..."
    
    echo "Available monitoring commands:"
    echo "1. Monitor validated data: confluent kafka topic consume validated-device-data.v1 --from-beginning"
    echo "2. Monitor validation failures: confluent kafka topic consume failed-validation.v1 --from-beginning"
    echo "3. Monitor sink failures: confluent kafka topic consume sink-write-failures.v1 --from-beginning"
    echo ""
    echo "Health check endpoints:"
    echo "- Stage 1: http://localhost:8041/api/v1/health"
    echo "- Stage 2: http://localhost:8042/api/v1/health"
    echo ""
    echo "Metrics endpoints:"
    echo "- Stage 1: http://localhost:8041/actuator/metrics"
    echo "- Stage 2: http://localhost:8042/api/v1/metrics"
}

# Main menu
show_menu() {
    echo ""
    echo "Select an option:"
    echo "1. Check prerequisites"
    echo "2. Setup Kafka topics"
    echo "3. Build Stage 1"
    echo "4. Setup Stage 2"
    echo "5. Start Stage 1 (Validator & Enricher)"
    echo "6. Start Stage 2 (Storage Fan-Out)"
    echo "7. Check health"
    echo "8. Send test data"
    echo "9. Monitor topics"
    echo "10. Full setup (1-4)"
    echo "0. Exit"
    echo ""
}

# Main execution
main() {
    while true; do
        show_menu
        read -p "Enter your choice: " choice
        
        case $choice in
            1) check_prerequisites ;;
            2) setup_kafka_topics ;;
            3) build_stage1 ;;
            4) setup_stage2 ;;
            5) start_stage1 ;;
            6) start_stage2 ;;
            7) check_health ;;
            8) send_test_data ;;
            9) monitor_topics ;;
            10) 
                check_prerequisites
                setup_kafka_topics
                build_stage1
                setup_stage2
                print_status "Full setup completed! Now start Stage 1 and Stage 2 in separate terminals."
                ;;
            0) 
                print_status "Goodbye!"
                exit 0
                ;;
            *) 
                print_error "Invalid option. Please try again."
                ;;
        esac
        
        echo ""
        read -p "Press Enter to continue..."
    done
}

# Run main function
main
