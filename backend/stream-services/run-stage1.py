#!/usr/bin/env python3
"""
Python Script to Run Stage 1: Validator & Enricher Service
Windows-compatible launcher for the Java Spring Boot service
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
    """Check if Java and Maven are available"""
    print_info("Checking prerequisites...")
    
    # Check Java
    try:
        result = subprocess.run(['java', '-version'], capture_output=True, text=True)
        if result.returncode == 0:
            print_status("Java is available")
        else:
            print_error("Java not found. Please install Java 17+")
            return False
    except FileNotFoundError:
        print_error("Java not found. Please install Java 17+")
        return False
    
    # Check Maven
    try:
        result = subprocess.run(['mvn', '-version'], capture_output=True, text=True)
        if result.returncode == 0:
            print_status("Maven is available")
        else:
            print_error("Maven not found. Please install Maven")
            return False
    except FileNotFoundError:
        print_error("Maven not found. Please install Maven")
        return False
    
    return True

def setup_environment():
    """Set up environment variables for Stage 1"""
    print_info("Setting up environment variables...")
    
    # Kafka Configuration (your actual credentials)
    os.environ['KAFKA_BOOTSTRAP_SERVERS'] = 'pkc-619z3.us-east1.gcp.confluent.cloud:9092'
    os.environ['KAFKA_API_KEY'] = 'LGJ3AQ2L6VRPW4S2'
    os.environ['KAFKA_API_SECRET'] = '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl'
    os.environ['KAFKA_SECURITY_PROTOCOL'] = 'SASL_SSL'
    os.environ['KAFKA_SASL_MECHANISM'] = 'PLAIN'
    
    # Redis Configuration
    os.environ['REDIS_HOST'] = 'localhost'
    os.environ['REDIS_PORT'] = '6379'
    os.environ['REDIS_PASSWORD'] = ''
    
    # Patient Service Configuration
    os.environ['PATIENT_SERVICE_URL'] = 'http://localhost:8003/api/v1/patient'
    
    # Kafka Topics
    os.environ['KAFKA_INPUT_TOPIC'] = 'raw-device-data.v1'
    os.environ['KAFKA_OUTPUT_TOPIC'] = 'validated-device-data.v1'
    os.environ['KAFKA_DLQ_TOPIC'] = 'failed-validation.v1'
    
    print_status("Environment variables configured")

def build_stage1():
    """Build Stage 1 service"""
    print_info("Building Stage 1 service...")
    
    # Change to stage1 directory
    stage1_dir = Path('stage1-validator-enricher')
    if not stage1_dir.exists():
        print_error("Stage 1 directory not found. Please run from backend/stream-services")
        return False
    
    os.chdir(stage1_dir)
    
    try:
        # Clean and package
        print_info("Running Maven clean package...")
        result = subprocess.run(['mvn', 'clean', 'package', '-DskipTests'], 
                              capture_output=True, text=True)
        
        if result.returncode == 0:
            print_status("Stage 1 built successfully")
            return True
        else:
            print_error("Maven build failed:")
            print(result.stderr)
            return False
            
    except Exception as e:
        print_error(f"Build failed: {e}")
        return False
    finally:
        # Return to parent directory
        os.chdir('..')

def start_stage1():
    """Start Stage 1 service"""
    print_info("Starting Stage 1: Validator & Enricher Service on port 8041...")
    
    # Change to stage1 directory
    stage1_dir = Path('stage1-validator-enricher')
    if not stage1_dir.exists():
        print_error("Stage 1 directory not found")
        return False
    
    os.chdir(stage1_dir)
    
    try:
        print_info("Starting Spring Boot application...")
        print_info("This will run in the foreground. Press Ctrl+C to stop.")
        print_info("Open another terminal to run Stage 2.")
        
        # Start the Spring Boot application
        subprocess.run(['mvn', 'spring-boot:run', '-Dspring-boot.run.profiles=dev'])
        
    except KeyboardInterrupt:
        print_info("Stage 1 service stopped by user")
    except Exception as e:
        print_error(f"Failed to start Stage 1: {e}")
    finally:
        os.chdir('..')

def check_health():
    """Check if Stage 1 is running and healthy"""
    print_info("Checking Stage 1 health...")
    
    try:
        response = requests.get('http://localhost:8041/api/v1/health', timeout=5)
        if response.status_code == 200:
            health_data = response.json()
            if health_data.get('status') == 'UP':
                print_status("Stage 1 is healthy and running!")
                return True
            else:
                print_error("Stage 1 is not healthy")
                return False
        else:
            print_error(f"Health check failed with status: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print_error(f"Cannot connect to Stage 1: {e}")
        return False

def main():
    """Main function"""
    print("🚀 Stage 1: Validator & Enricher Service Launcher")
    print("=" * 50)
    
    # Check if we're in the right directory
    if not Path('stage1-validator-enricher').exists():
        print_error("Please run this script from backend/stream-services directory")
        sys.exit(1)
    
    # Menu
    while True:
        print("\nSelect an option:")
        print("1. Check prerequisites")
        print("2. Build Stage 1")
        print("3. Start Stage 1 (will run in foreground)")
        print("4. Check health")
        print("5. Full setup and start (1-3)")
        print("0. Exit")
        
        choice = input("\nEnter your choice: ").strip()
        
        if choice == '1':
            if check_prerequisites():
                print_status("All prerequisites are met!")
            else:
                print_error("Please install missing prerequisites")
                
        elif choice == '2':
            setup_environment()
            if build_stage1():
                print_status("Stage 1 is ready to run!")
            else:
                print_error("Build failed. Please check the errors above.")
                
        elif choice == '3':
            setup_environment()
            start_stage1()
            
        elif choice == '4':
            check_health()
            
        elif choice == '5':
            if not check_prerequisites():
                print_error("Prerequisites check failed")
                continue
                
            setup_environment()
            
            if not build_stage1():
                print_error("Build failed")
                continue
                
            print_status("Setup complete! Starting Stage 1...")
            time.sleep(2)
            start_stage1()
            
        elif choice == '0':
            print_info("Goodbye!")
            break
            
        else:
            print_error("Invalid choice. Please try again.")

if __name__ == "__main__":
    main()
