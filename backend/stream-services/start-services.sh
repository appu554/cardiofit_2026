#!/bin/bash

# Quick Start Script for Stage 1 & Stage 2 with Your Kafka Credentials
# This script starts both services with the correct configuration

set -e

echo "🚀 Starting Stage 1 & Stage 2 Services"
echo "======================================"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if we're in the right directory
if [ ! -d "stage1-validator-enricher" ] || [ ! -d "stage2-storage-fanout" ]; then
    echo "❌ Please run this script from the backend/stream-services directory"
    exit 1
fi

# Function to start Stage 1
start_stage1() {
    print_step "Starting Stage 1 (Validator & Enricher) on port 8041..."
    
    cd stage1-validator-enricher
    
    # Set Kafka environment variables
    export KAFKA_BOOTSTRAP_SERVERS="pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    export KAFKA_API_KEY="LGJ3AQ2L6VRPW4S2"
    export KAFKA_API_SECRET="2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"
    export KAFKA_SECURITY_PROTOCOL="SASL_SSL"
    export KAFKA_SASL_MECHANISM="PLAIN"
    
    # Set other environment variables
    export REDIS_HOST="localhost"
    export REDIS_PORT="6379"
    export PATIENT_SERVICE_URL="http://localhost:8003/api/v1/patient"
    
    print_status "Environment variables set for Stage 1"
    print_status "Starting Spring Boot application..."
    
    mvn spring-boot:run -Dspring-boot.run.profiles=dev
}

# Function to start Stage 2
start_stage2() {
    print_step "Starting Stage 2 (Storage Fan-Out) on port 8042..."
    
    cd stage2-storage-fanout
    
    # Load environment from .env.dev (already has your credentials)
    if [ -f ".env.dev" ]; then
        export $(cat .env.dev | grep -v '^#' | xargs)
        print_status "Environment variables loaded from .env.dev"
    fi
    
    # Override with your specific credentials (just to be sure)
    export KAFKA_BOOTSTRAP_SERVERS="pkc-619z3.us-east1.gcp.confluent.cloud:9092"
    export KAFKA_API_KEY="LGJ3AQ2L6VRPW4S2"
    export KAFKA_API_SECRET="2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl"
    
    print_status "Starting FastAPI application..."
    
    python -m uvicorn app.main:app --host 0.0.0.0 --port 8042 --reload
}

# Function to check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check Java
    if ! command -v java &> /dev/null; then
        echo "❌ Java not found. Please install Java 17+"
        exit 1
    fi
    
    # Check Maven
    if ! command -v mvn &> /dev/null; then
        echo "❌ Maven not found. Please install Maven"
        exit 1
    fi
    
    # Check Python
    if ! command -v python3 &> /dev/null; then
        echo "❌ Python 3 not found. Please install Python 3.11+"
        exit 1
    fi
    
    print_status "Prerequisites check passed ✅"
}

# Function to build Stage 1
build_stage1() {
    print_step "Building Stage 1..."
    
    cd stage1-validator-enricher
    mvn clean package -DskipTests
    cd ..
    
    print_status "Stage 1 built successfully ✅"
}

# Function to setup Stage 2
setup_stage2() {
    print_step "Setting up Stage 2..."
    
    cd stage2-storage-fanout
    pip install -r requirements.txt
    cd ..
    
    print_status "Stage 2 setup completed ✅"
}

# Main menu
show_menu() {
    echo ""
    echo "Select an option:"
    echo "1. Check prerequisites"
    echo "2. Build Stage 1"
    echo "3. Setup Stage 2"
    echo "4. Start Stage 1 (Validator & Enricher)"
    echo "5. Start Stage 2 (Storage Fan-Out)"
    echo "6. Full setup (1-3)"
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
            2) build_stage1 ;;
            3) setup_stage2 ;;
            4) start_stage1 ;;
            5) start_stage2 ;;
            6) 
                check_prerequisites
                build_stage1
                setup_stage2
                print_status "Full setup completed!"
                echo ""
                echo "🎯 Next steps:"
                echo "1. Open Terminal 1: ./start-services.sh → Select option 4 (Start Stage 1)"
                echo "2. Open Terminal 2: ./start-services.sh → Select option 5 (Start Stage 2)"
                echo "3. Open Terminal 3: python3 test-data-generator.py (to send test data)"
                ;;
            0) 
                print_status "Goodbye!"
                exit 0
                ;;
            *) 
                echo "❌ Invalid option. Please try again."
                ;;
        esac
        
        echo ""
        read -p "Press Enter to continue..."
    done
}

# Run main function
main
