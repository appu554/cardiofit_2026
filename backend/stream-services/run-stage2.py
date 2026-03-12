#!/usr/bin/env python3
"""
Python Script to Run Stage 2: Storage Fan-Out Service
Windows-compatible launcher with your exact PySpark ETL configuration
"""

import os
import sys
import subprocess
import time
import requests
from pathlib import Path

def print_status(message):
    print(f"✅ {message}")

def print_error(message):
    print(f"❌ {message}")

def print_info(message):
    print(f"ℹ️  {message}")

def check_prerequisites():
    """Check if Python and pip are available"""
    print_info("Checking prerequisites...")
    
    # Check Python
    try:
        result = subprocess.run([sys.executable, '--version'], capture_output=True, text=True)
        if result.returncode == 0:
            print_status(f"Python is available: {result.stdout.strip()}")
        else:
            print_error("Python not found")
            return False
    except FileNotFoundError:
        print_error("Python not found")
        return False
    
    # Check pip
    try:
        result = subprocess.run([sys.executable, '-m', 'pip', '--version'], capture_output=True, text=True)
        if result.returncode == 0:
            print_status("pip is available")
        else:
            print_error("pip not found")
            return False
    except FileNotFoundError:
        print_error("pip not found")
        return False
    
    return True

def setup_environment():
    """Set up environment variables for Stage 2 with your exact PySpark configuration"""
    print_info("Setting up environment variables with your PySpark ETL configuration...")
    
    # Service Configuration
    os.environ['PORT'] = '8042'
    os.environ['DEBUG'] = 'true'
    os.environ['SERVICE_NAME'] = 'stage2-storage-fanout'
    
    # Kafka Configuration (your actual credentials)
    os.environ['KAFKA_BOOTSTRAP_SERVERS'] = 'pkc-619z3.us-east1.gcp.confluent.cloud:9092'
    os.environ['KAFKA_API_KEY'] = 'LGJ3AQ2L6VRPW4S2'
    os.environ['KAFKA_API_SECRET'] = '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl'
    os.environ['KAFKA_SECURITY_PROTOCOL'] = 'SASL_SSL'
    os.environ['KAFKA_SASL_MECHANISM'] = 'PLAIN'
    
    # Kafka Topics
    os.environ['KAFKA_INPUT_TOPIC'] = 'validated-device-data.v1'
    os.environ['KAFKA_DLQ_TOPIC'] = 'sink-write-failures.v1'
    os.environ['KAFKA_CONSUMER_GROUP'] = 'stage2-storage-fanout'
    
    # Multi-Sink Configuration
    os.environ['PARALLEL_WRITES'] = 'true'
    os.environ['THREAD_POOL_SIZE'] = '6'
    os.environ['SINK_TIMEOUT_SECONDS'] = '30'
    os.environ['BATCH_SIZE'] = '100'
    
    # FHIR Store Configuration (EXACT same as your PySpark ETL)
    os.environ['FHIR_STORE_ENABLED'] = 'true'
    os.environ['GOOGLE_CLOUD_PROJECT'] = 'cardiofit-905a8'
    os.environ['GOOGLE_CLOUD_LOCATION'] = 'asia-south1'
    os.environ['GOOGLE_CLOUD_DATASET'] = 'clinical-synthesis-hub'
    os.environ['GOOGLE_CLOUD_FHIR_STORE'] = 'fhir-store'
    # Note: Set GOOGLE_APPLICATION_CREDENTIALS to your actual credentials file path
    
    # Elasticsearch Configuration (EXACT same as your PySpark ETL)
    os.environ['ELASTICSEARCH_ENABLED'] = 'true'
    os.environ['ELASTICSEARCH_URL'] = 'https://my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud:443'
    os.environ['ELASTICSEARCH_API_KEY'] = 'd0gyTG5aY0JGajhWTVBOTzkzeDk6VGxoNENEd29DZEtERXBxRXpRUXBEUQ=='
    os.environ['ELASTICSEARCH_INDEX_PREFIX'] = 'patient-readings'
    os.environ['ELASTICSEARCH_TIMEOUT'] = '30'
    
    # MongoDB Configuration (EXACT same as your PySpark ETL)
    os.environ['MONGODB_ENABLED'] = 'true'
    os.environ['MONGODB_URI'] = 'mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0'
    os.environ['MONGODB_DATABASE'] = 'clinical_synthesis_hub'
    os.environ['MONGODB_COLLECTION'] = 'device_readings'
    os.environ['MONGODB_TIMEOUT'] = '30'
    
    # Redis Configuration
    os.environ['REDIS_HOST'] = 'localhost'
    os.environ['REDIS_PORT'] = '6379'
    os.environ['REDIS_DB'] = '1'
    
    # Circuit Breaker Configuration
    os.environ['CIRCUIT_BREAKER_ENABLED'] = 'true'
    os.environ['CIRCUIT_BREAKER_FAILURE_THRESHOLD'] = '10'
    os.environ['CIRCUIT_BREAKER_RECOVERY_TIMEOUT'] = '60'
    
    # Retry Configuration
    os.environ['RETRY_ENABLED'] = 'true'
    os.environ['RETRY_MAX_ATTEMPTS'] = '3'
    os.environ['RETRY_BACKOFF_FACTOR'] = '2.0'
    os.environ['RETRY_MAX_WAIT'] = '60'
    
    # Monitoring Configuration
    os.environ['PROMETHEUS_ENABLED'] = 'true'
    os.environ['METRICS_ENABLED'] = 'true'
    
    # Logging Configuration
    os.environ['LOG_LEVEL'] = 'DEBUG'
    os.environ['LOG_FORMAT'] = 'json'
    
    print_status("Environment variables configured with your PySpark ETL settings")

def setup_stage2():
    """Install Stage 2 dependencies"""
    print_info("Setting up Stage 2 dependencies...")
    
    # Change to stage2 directory
    stage2_dir = Path('stage2-storage-fanout')
    if not stage2_dir.exists():
        print_error("Stage 2 directory not found. Please run from backend/stream-services")
        return False
    
    os.chdir(stage2_dir)
    
    try:
        # Install requirements
        print_info("Installing Python dependencies...")
        result = subprocess.run([sys.executable, '-m', 'pip', 'install', '-r', 'requirements.txt'], 
                              capture_output=True, text=True)
        
        if result.returncode == 0:
            print_status("Stage 2 dependencies installed successfully")
            return True
        else:
            print_error("Failed to install dependencies:")
            print(result.stderr)
            return False
            
    except Exception as e:
        print_error(f"Setup failed: {e}")
        return False
    finally:
        # Return to parent directory
        os.chdir('..')

def start_stage2():
    """Start Stage 2 service"""
    print_info("Starting Stage 2: Storage Fan-Out Service on port 8042...")
    
    # Change to stage2 directory
    stage2_dir = Path('stage2-storage-fanout')
    if not stage2_dir.exists():
        print_error("Stage 2 directory not found")
        return False
    
    os.chdir(stage2_dir)
    
    try:
        print_info("Starting FastAPI application with your PySpark ETL configuration...")
        print_info("This will run in the foreground. Press Ctrl+C to stop.")
        print_info("Multi-sink writes enabled: FHIR Store + Elasticsearch + MongoDB")
        
        # Start the FastAPI application
        subprocess.run([sys.executable, '-m', 'uvicorn', 'app.main:app', 
                       '--host', '0.0.0.0', '--port', '8042', '--reload'])
        
    except KeyboardInterrupt:
        print_info("Stage 2 service stopped by user")
    except Exception as e:
        print_error(f"Failed to start Stage 2: {e}")
    finally:
        os.chdir('..')

def check_health():
    """Check if Stage 2 is running and healthy"""
    print_info("Checking Stage 2 health...")
    
    try:
        response = requests.get('http://localhost:8042/api/v1/health', timeout=5)
        if response.status_code == 200:
            health_data = response.json()
            if health_data.get('status') == 'UP':
                print_status("Stage 2 is healthy and running!")
                
                # Check individual components
                components = health_data.get('components', {})
                for component, status in components.items():
                    if isinstance(status, dict) and status.get('status') == 'UP':
                        print_status(f"  {component}: Healthy")
                    else:
                        print_error(f"  {component}: Not healthy")
                
                return True
            else:
                print_error("Stage 2 is not healthy")
                return False
        else:
            print_error(f"Health check failed with status: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print_error(f"Cannot connect to Stage 2: {e}")
        return False

def check_sinks():
    """Check individual sink health"""
    print_info("Checking sink health...")
    
    try:
        response = requests.get('http://localhost:8042/api/v1/health/sinks', timeout=5)
        if response.status_code == 200:
            sink_data = response.json()
            print_status("Sink health status:")
            
            sink_health = sink_data.get('sink_health', {})
            for sink_name, health in sink_health.items():
                status = health.get('status', 'UNKNOWN')
                if status == 'UP':
                    print_status(f"  {sink_name}: Healthy")
                else:
                    print_error(f"  {sink_name}: {status}")
            
            return True
        else:
            print_error(f"Sink health check failed with status: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print_error(f"Cannot check sink health: {e}")
        return False

def main():
    """Main function"""
    print("💾 Stage 2: Storage Fan-Out Service Launcher")
    print("=" * 50)
    print("🎯 Configured with your exact PySpark ETL settings:")
    print("   - FHIR Store: projects/cardiofit-905a8/.../fhir-store")
    print("   - Elasticsearch: my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud")
    print("   - MongoDB: cluster0.yqdzbvb.mongodb.net/clinical_synthesis_hub")
    print("   - Patient ID: 905a60cb-8241-418f-b29b-5b020e851392")
    print("=" * 50)
    
    # Check if we're in the right directory
    if not Path('stage2-storage-fanout').exists():
        print_error("Please run this script from backend/stream-services directory")
        sys.exit(1)
    
    # Menu
    while True:
        print("\nSelect an option:")
        print("1. Check prerequisites")
        print("2. Setup Stage 2 dependencies")
        print("3. Start Stage 2 (will run in foreground)")
        print("4. Check health")
        print("5. Check sink health")
        print("6. Full setup and start (1-3)")
        print("0. Exit")
        
        choice = input("\nEnter your choice: ").strip()
        
        if choice == '1':
            if check_prerequisites():
                print_status("All prerequisites are met!")
            else:
                print_error("Please install missing prerequisites")
                
        elif choice == '2':
            setup_environment()
            if setup_stage2():
                print_status("Stage 2 is ready to run!")
            else:
                print_error("Setup failed. Please check the errors above.")
                
        elif choice == '3':
            setup_environment()
            start_stage2()
            
        elif choice == '4':
            check_health()
            
        elif choice == '5':
            check_sinks()
            
        elif choice == '6':
            if not check_prerequisites():
                print_error("Prerequisites check failed")
                continue
                
            setup_environment()
            
            if not setup_stage2():
                print_error("Setup failed")
                continue
                
            print_status("Setup complete! Starting Stage 2...")
            time.sleep(2)
            start_stage2()
            
        elif choice == '0':
            print_info("Goodbye!")
            break
            
        else:
            print_error("Invalid choice. Please try again.")

if __name__ == "__main__":
    main()
