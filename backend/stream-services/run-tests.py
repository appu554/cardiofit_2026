#!/usr/bin/env python3
"""
Comprehensive Test Runner for Stage 1 & Stage 2
Windows-compatible testing with your exact configuration
"""

import os
import sys
import time
import requests
import subprocess
from pathlib import Path

def print_status(message):
    print(f"✅ {message}")

def print_error(message):
    print(f"❌ {message}")

def print_info(message):
    print(f"ℹ️  {message}")

def print_warning(message):
    print(f"⚠️  {message}")

def check_service_health(service_name, port, endpoint="/api/v1/health"):
    """Check if a service is healthy"""
    try:
        url = f"http://localhost:{port}{endpoint}"
        response = requests.get(url, timeout=5)
        
        if response.status_code == 200:
            health_data = response.json()
            if health_data.get('status') == 'UP':
                print_status(f"{service_name} is healthy on port {port}")
                return True
            else:
                print_error(f"{service_name} is not healthy: {health_data}")
                return False
        else:
            print_error(f"{service_name} health check failed: HTTP {response.status_code}")
            return False
            
    except requests.exceptions.RequestException as e:
        print_error(f"Cannot connect to {service_name} on port {port}: {e}")
        return False

def send_test_data():
    """Send test data using the test data generator"""
    print_info("Sending test data with your patient ID: 905a60cb-8241-418f-b29b-5b020e851392")
    
    try:
        # Run the test data generator
        result = subprocess.run([sys.executable, 'test-data-generator.py'], 
                              input="5\n10\n", text=True, capture_output=True)
        
        if result.returncode == 0:
            print_status("Test data sent successfully!")
            print("Output:", result.stdout)
        else:
            print_error("Failed to send test data:")
            print("Error:", result.stderr)
            
    except Exception as e:
        print_error(f"Failed to run test data generator: {e}")

def monitor_kafka_topics():
    """Show commands to monitor Kafka topics"""
    print_info("Kafka Topic Monitoring Commands:")
    print()
    print("📊 Monitor validated data (Stage 1 → Stage 2):")
    print("   confluent kafka topic consume validated-device-data.v1 --from-beginning --max-messages 5")
    print()
    print("🚨 Monitor validation failures (Stage 1 DLQ):")
    print("   confluent kafka topic consume failed-validation.v1 --from-beginning --max-messages 5")
    print()
    print("💾 Monitor sink failures (Stage 2 DLQ):")
    print("   confluent kafka topic consume sink-write-failures.v1 --from-beginning --max-messages 5")
    print()
    print("🔍 Check consumer group lag:")
    print("   confluent kafka consumer group describe stage1-validator-enricher")
    print("   confluent kafka consumer group describe stage2-storage-fanout")

def show_metrics():
    """Show metrics endpoints"""
    print_info("Metrics Endpoints:")
    print()
    print("📈 Stage 1 Metrics:")
    print("   http://localhost:8041/actuator/metrics")
    print("   http://localhost:8041/actuator/kafka-streams")
    print("   http://localhost:8041/api/v1/health/validation")
    print()
    print("📊 Stage 2 Metrics:")
    print("   http://localhost:8042/api/v1/metrics")
    print("   http://localhost:8042/api/v1/metrics/kafka")
    print("   http://localhost:8042/api/v1/metrics/sinks")
    print("   http://localhost:8042/api/v1/metrics/dlq")
    print()
    
    # Try to fetch some basic metrics
    try:
        print_info("Fetching current metrics...")
        
        # Stage 1 health
        response = requests.get('http://localhost:8041/api/v1/health', timeout=3)
        if response.status_code == 200:
            print_status("Stage 1 is responding")
        
        # Stage 2 metrics
        response = requests.get('http://localhost:8042/api/v1/metrics/summary', timeout=3)
        if response.status_code == 200:
            metrics = response.json()
            print_status(f"Stage 2 Summary:")
            print(f"   Messages processed: {metrics.get('messages_processed_total', 0)}")
            print(f"   Sink writes: {metrics.get('sink_writes_total', 0)}")
            print(f"   Success rate: {metrics.get('sink_success_rate', 0):.2%}")
            
    except Exception as e:
        print_warning(f"Could not fetch metrics: {e}")

def run_end_to_end_test():
    """Run a complete end-to-end test"""
    print_info("Running End-to-End Test...")
    print("=" * 50)
    
    # Step 1: Check both services are running
    print_info("Step 1: Checking service health...")
    stage1_healthy = check_service_health("Stage 1", 8041)
    stage2_healthy = check_service_health("Stage 2", 8042)
    
    if not (stage1_healthy and stage2_healthy):
        print_error("Both services must be running for end-to-end test")
        print_info("Start Stage 1: python run-stage1.py")
        print_info("Start Stage 2: python run-stage2.py")
        return False
    
    # Step 2: Check sink health
    print_info("Step 2: Checking sink health...")
    try:
        response = requests.get('http://localhost:8042/api/v1/health/sinks', timeout=5)
        if response.status_code == 200:
            sink_data = response.json()
            enabled_sinks = sink_data.get('enabled_sinks', {})
            print_status(f"Enabled sinks: {enabled_sinks}")
        else:
            print_warning("Could not check sink health")
    except Exception as e:
        print_warning(f"Sink health check failed: {e}")
    
    # Step 3: Send test data
    print_info("Step 3: Sending test data...")
    send_test_data()
    
    # Step 4: Wait and check results
    print_info("Step 4: Waiting for processing...")
    time.sleep(10)  # Wait for processing
    
    # Step 5: Check metrics
    print_info("Step 5: Checking processing results...")
    show_metrics()
    
    print_status("End-to-end test completed!")
    return True

def show_troubleshooting():
    """Show troubleshooting information"""
    print_info("Troubleshooting Guide:")
    print()
    print("🔧 Common Issues:")
    print()
    print("1. Stage 1 won't start:")
    print("   - Check Java version: java -version (need 17+)")
    print("   - Check Maven: mvn -version")
    print("   - Check Kafka connectivity")
    print()
    print("2. Stage 2 won't start:")
    print("   - Check Python version: python --version (need 3.11+)")
    print("   - Install dependencies: pip install -r stage2-storage-fanout/requirements.txt")
    print("   - Check MongoDB/Elasticsearch connectivity")
    print()
    print("3. No data flowing:")
    print("   - Check Kafka topics exist")
    print("   - Check consumer group lag")
    print("   - Look for errors in service logs")
    print()
    print("4. Sink failures:")
    print("   - Check Google Cloud credentials")
    print("   - Check Elasticsearch API key")
    print("   - Check MongoDB connection string")
    print()
    print("📞 Debug Commands:")
    print("   curl http://localhost:8041/api/v1/health")
    print("   curl http://localhost:8042/api/v1/health")
    print("   curl http://localhost:8042/api/v1/health/sinks")

def setup_kafka_topics():
    """Setup Kafka topics using the Python script"""
    print_info("Setting up Kafka topics...")

    try:
        result = subprocess.run([sys.executable, 'setup-kafka-topics.py'],
                              input="5\n", text=True, capture_output=True)

        if result.returncode == 0:
            print_status("Kafka topics setup completed!")
            print("Output:", result.stdout)
        else:
            print_error("Failed to setup Kafka topics:")
            print("Error:", result.stderr)

    except Exception as e:
        print_error(f"Failed to run Kafka setup: {e}")

def main():
    """Main function"""
    print("🧪 Stage 1 & Stage 2 Test Runner")
    print("=" * 50)
    print("🎯 Your Configuration:")
    print("   Patient ID: 905a60cb-8241-418f-b29b-5b020e851392")
    print("   FHIR Store: cardiofit-905a8/clinical-synthesis-hub/fhir-store")
    print("   Elasticsearch: my-elasticsearch-project-ba1a02.es.us-east-1.aws.elastic.cloud")
    print("   MongoDB: cluster0.yqdzbvb.mongodb.net/clinical_synthesis_hub")
    print("=" * 50)
    
    # Check if we're in the right directory
    if not (Path('stage1-validator-enricher').exists() and Path('stage2-storage-fanout').exists()):
        print_error("Please run this script from backend/stream-services directory")
        sys.exit(1)
    
    # Menu
    while True:
        print("\nSelect an option:")
        print("1. Setup Kafka topics")
        print("2. Check service health")
        print("3. Send test data")
        print("4. Monitor Kafka topics (show commands)")
        print("5. Show metrics")
        print("6. Run end-to-end test")
        print("7. Troubleshooting guide")
        print("0. Exit")
        
        choice = input("\nEnter your choice: ").strip()

        if choice == '1':
            setup_kafka_topics()

        elif choice == '2':
            print_info("Checking service health...")
            check_service_health("Stage 1", 8041)
            check_service_health("Stage 2", 8042)

        elif choice == '3':
            send_test_data()

        elif choice == '4':
            monitor_kafka_topics()

        elif choice == '5':
            show_metrics()

        elif choice == '6':
            run_end_to_end_test()

        elif choice == '7':
            show_troubleshooting()
            
        elif choice == '0':
            print_info("Goodbye!")
            break
            
        else:
            print_error("Invalid choice. Please try again.")

if __name__ == "__main__":
    main()
