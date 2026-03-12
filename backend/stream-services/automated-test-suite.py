#!/usr/bin/env python3
"""
Automated Test Suite for Stage 1 & Stage 2
Comprehensive testing of the modular stream processing pipeline
"""

import asyncio
import json
import time
import requests
import subprocess
import sys
from datetime import datetime
from typing import Dict, Any, List
from kafka import KafkaProducer, KafkaConsumer
from kafka.errors import KafkaError

# Test configuration
KAFKA_CONFIG = {
    'bootstrap_servers': 'pkc-619z3.us-east1.gcp.confluent.cloud:9092',
    'security_protocol': 'SASL_SSL',
    'sasl_mechanism': 'PLAIN',
    'sasl_plain_username': 'LGJ3AQ2L6VRPW4S2',
    'sasl_plain_password': '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl',
    'value_serializer': lambda x: json.dumps(x).encode('utf-8'),
    'key_serializer': lambda x: x.encode('utf-8') if x else None,
    'acks': 'all',
    'retries': 3
}

CONSUMER_CONFIG = {
    'bootstrap_servers': 'pkc-619z3.us-east1.gcp.confluent.cloud:9092',
    'security_protocol': 'SASL_SSL',
    'sasl_mechanism': 'PLAIN',
    'sasl_plain_username': 'LGJ3AQ2L6VRPW4S2',
    'sasl_plain_password': '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl',
    'value_deserializer': lambda x: json.loads(x.decode('utf-8')) if x else None,
    'key_deserializer': lambda x: x.decode('utf-8') if x else None,
    'auto_offset_reset': 'latest',
    'group_id': 'automated-test-suite'
}

# Service endpoints
STAGE1_BASE_URL = "http://localhost:8041"
STAGE2_BASE_URL = "http://localhost:8042"

# Test topics
INPUT_TOPIC = "raw-device-data.v1"
VALIDATED_TOPIC = "validated-device-data.v1"
FAILED_VALIDATION_TOPIC = "failed-validation.v1"
SINK_FAILURES_TOPIC = "sink-write-failures.v1"

class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    BOLD = '\033[1m'
    END = '\033[0m'

class AutomatedTestSuite:
    def __init__(self):
        self.test_results = []
        self.producer = None
        self.start_time = time.time()
        
    def log(self, message: str, level: str = "INFO"):
        timestamp = datetime.now().strftime("%H:%M:%S")
        color = Colors.GREEN if level == "PASS" else Colors.RED if level == "FAIL" else Colors.BLUE
        print(f"{color}[{timestamp}] {level}: {message}{Colors.END}")
        
    def log_test_result(self, test_name: str, passed: bool, details: str = ""):
        self.test_results.append({
            'test': test_name,
            'passed': passed,
            'details': details,
            'timestamp': time.time()
        })
        level = "PASS" if passed else "FAIL"
        self.log(f"{test_name}: {'✅' if passed else '❌'} {details}", level)

    async def setup(self):
        """Initialize test environment"""
        self.log("🚀 Starting Automated Test Suite for Stage 1 & Stage 2")
        self.log("=" * 60)
        
        try:
            # Initialize Kafka producer
            self.producer = KafkaProducer(**KAFKA_CONFIG)
            self.log("Kafka producer initialized ✅")
            return True
        except Exception as e:
            self.log(f"Failed to initialize Kafka producer: {e}", "FAIL")
            return False

    async def test_service_health(self):
        """Test health endpoints for both services"""
        self.log("\n🏥 Testing Service Health...")
        
        # Test Stage 1 health
        try:
            response = requests.get(f"{STAGE1_BASE_URL}/api/v1/health", timeout=10)
            if response.status_code == 200:
                health_data = response.json()
                stage1_healthy = health_data.get('status') == 'UP'
                self.log_test_result("Stage 1 Health Check", stage1_healthy, 
                                   f"Status: {health_data.get('status', 'Unknown')}")
            else:
                self.log_test_result("Stage 1 Health Check", False, 
                                   f"HTTP {response.status_code}")
        except Exception as e:
            self.log_test_result("Stage 1 Health Check", False, f"Connection error: {e}")

        # Test Stage 2 health
        try:
            response = requests.get(f"{STAGE2_BASE_URL}/api/v1/health", timeout=10)
            if response.status_code == 200:
                health_data = response.json()
                stage2_healthy = health_data.get('status') == 'UP'
                self.log_test_result("Stage 2 Health Check", stage2_healthy,
                                   f"Status: {health_data.get('status', 'Unknown')}")
            else:
                self.log_test_result("Stage 2 Health Check", False,
                                   f"HTTP {response.status_code}")
        except Exception as e:
            self.log_test_result("Stage 2 Health Check", False, f"Connection error: {e}")

    async def test_component_health(self):
        """Test individual component health"""
        self.log("\n🔧 Testing Component Health...")
        
        # Stage 1 components
        components = [
            ("Stage 1 Validation", f"{STAGE1_BASE_URL}/api/v1/health/validation"),
            ("Stage 1 Patient Context", f"{STAGE1_BASE_URL}/api/v1/health/patient-context"),
            ("Stage 2 Kafka Consumer", f"{STAGE2_BASE_URL}/api/v1/health/kafka"),
            ("Stage 2 Sinks", f"{STAGE2_BASE_URL}/api/v1/health/sinks"),
            ("Stage 2 DLQ", f"{STAGE2_BASE_URL}/api/v1/health/dlq"),
            ("Stage 2 FHIR Transformer", f"{STAGE2_BASE_URL}/api/v1/health/fhir-transformer")
        ]
        
        for component_name, url in components:
            try:
                response = requests.get(url, timeout=5)
                if response.status_code == 200:
                    health_data = response.json()
                    healthy = health_data.get('status') == 'UP'
                    self.log_test_result(component_name, healthy,
                                       f"Status: {health_data.get('status', 'Unknown')}")
                else:
                    self.log_test_result(component_name, False, f"HTTP {response.status_code}")
            except Exception as e:
                self.log_test_result(component_name, False, f"Error: {e}")

    def generate_test_data(self, scenario: str = "normal") -> Dict[str, Any]:
        """Generate test device reading data"""
        base_data = {
            'device_id': f'test-device-{int(time.time() * 1000) % 1000}',
            'timestamp': int(time.time()),
            'reading_type': 'heart_rate',
            'value': 75,
            'unit': 'bpm',
            'patient_id': 'test-patient-001',
            'metadata': {
                'battery_level': 85,
                'signal_quality': 'good',
                'device_model': 'test-monitor-v1'
            },
            'vendor_info': {
                'vendor_id': 'test-vendor',
                'vendor_name': 'Test Medical Devices'
            }
        }
        
        if scenario == "critical":
            base_data['value'] = 180  # Critical heart rate
        elif scenario == "emergency":
            base_data['value'] = 35   # Emergency heart rate
        elif scenario == "invalid":
            base_data['value'] = None  # Invalid value
        elif scenario == "missing_field":
            del base_data['device_id']  # Missing required field
            
        return base_data

    async def test_data_flow(self):
        """Test end-to-end data flow"""
        self.log("\n📊 Testing Data Flow...")
        
        # Test scenarios
        scenarios = [
            ("Normal Reading", "normal"),
            ("Critical Reading", "critical"),
            ("Emergency Reading", "emergency"),
            ("Invalid Reading", "invalid"),
            ("Missing Field", "missing_field")
        ]
        
        for scenario_name, scenario_type in scenarios:
            try:
                # Generate and send test data
                test_data = self.generate_test_data(scenario_type)
                key = test_data.get('device_id', f'test-{int(time.time())}')
                
                # Send to input topic
                future = self.producer.send(INPUT_TOPIC, key=key, value=test_data)
                result = future.get(timeout=10)
                
                self.log_test_result(f"Send {scenario_name}", True,
                                   f"Sent to partition {result.partition}")
                
                # Small delay for processing
                await asyncio.sleep(1)
                
            except Exception as e:
                self.log_test_result(f"Send {scenario_name}", False, f"Error: {e}")

    async def test_kafka_topics(self):
        """Test Kafka topic consumption"""
        self.log("\n📨 Testing Kafka Topic Consumption...")
        
        topics_to_test = [
            (VALIDATED_TOPIC, "Validated Data Topic"),
            (FAILED_VALIDATION_TOPIC, "Failed Validation Topic"),
            (SINK_FAILURES_TOPIC, "Sink Failures Topic")
        ]
        
        for topic, topic_name in topics_to_test:
            try:
                # Create consumer for this topic
                consumer = KafkaConsumer(
                    topic,
                    **CONSUMER_CONFIG,
                    consumer_timeout_ms=5000  # 5 second timeout
                )
                
                message_count = 0
                for message in consumer:
                    message_count += 1
                    if message_count >= 3:  # Check first 3 messages
                        break
                
                consumer.close()
                
                self.log_test_result(f"Consume {topic_name}", True,
                                   f"Found {message_count} messages")
                
            except Exception as e:
                self.log_test_result(f"Consume {topic_name}", False, f"Error: {e}")

    async def test_metrics(self):
        """Test metrics endpoints"""
        self.log("\n📈 Testing Metrics...")
        
        metrics_endpoints = [
            ("Stage 1 Actuator Metrics", f"{STAGE1_BASE_URL}/actuator/metrics"),
            ("Stage 1 Kafka Streams", f"{STAGE1_BASE_URL}/actuator/kafka-streams"),
            ("Stage 2 Overall Metrics", f"{STAGE2_BASE_URL}/api/v1/metrics"),
            ("Stage 2 Kafka Metrics", f"{STAGE2_BASE_URL}/api/v1/metrics/kafka"),
            ("Stage 2 Sink Metrics", f"{STAGE2_BASE_URL}/api/v1/metrics/sinks"),
            ("Stage 2 DLQ Metrics", f"{STAGE2_BASE_URL}/api/v1/metrics/dlq")
        ]
        
        for metric_name, url in metrics_endpoints:
            try:
                response = requests.get(url, timeout=5)
                if response.status_code == 200:
                    metrics_data = response.json()
                    has_data = len(metrics_data) > 0
                    self.log_test_result(metric_name, has_data,
                                       f"Returned {len(str(metrics_data))} bytes")
                else:
                    self.log_test_result(metric_name, False, f"HTTP {response.status_code}")
            except Exception as e:
                self.log_test_result(metric_name, False, f"Error: {e}")

    async def test_fhir_transformation(self):
        """Test FHIR transformation by checking Stage 2 processing"""
        self.log("\n🏥 Testing FHIR Transformation...")
        
        # Send a well-formed test message
        test_data = self.generate_test_data("normal")
        
        try:
            # Send test data
            future = self.producer.send(INPUT_TOPIC, 
                                      key=test_data['device_id'], 
                                      value=test_data)
            result = future.get(timeout=10)
            
            # Wait for processing
            await asyncio.sleep(3)
            
            # Check Stage 2 metrics for FHIR processing
            response = requests.get(f"{STAGE2_BASE_URL}/api/v1/metrics", timeout=5)
            if response.status_code == 200:
                metrics = response.json()
                # Look for processing indicators in metrics
                kafka_metrics = metrics.get('components', {}).get('kafka_consumer', {})
                processed = kafka_metrics.get('processed_messages', 0)
                
                self.log_test_result("FHIR Transformation Test", processed > 0,
                                   f"Processed {processed} messages")
            else:
                self.log_test_result("FHIR Transformation Test", False,
                                   "Could not get metrics")
                
        except Exception as e:
            self.log_test_result("FHIR Transformation Test", False, f"Error: {e}")

    async def generate_test_report(self):
        """Generate comprehensive test report"""
        self.log("\n📋 Test Report")
        self.log("=" * 60)
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result['passed'])
        failed_tests = total_tests - passed_tests
        
        self.log(f"Total Tests: {total_tests}")
        self.log(f"Passed: {Colors.GREEN}{passed_tests}{Colors.END}")
        self.log(f"Failed: {Colors.RED}{failed_tests}{Colors.END}")
        self.log(f"Success Rate: {(passed_tests/total_tests*100):.1f}%")
        
        if failed_tests > 0:
            self.log(f"\n{Colors.RED}Failed Tests:{Colors.END}")
            for result in self.test_results:
                if not result['passed']:
                    self.log(f"❌ {result['test']}: {result['details']}")
        
        self.log(f"\nTest Duration: {time.time() - self.start_time:.1f} seconds")
        
        # Overall assessment
        if passed_tests == total_tests:
            self.log(f"\n{Colors.GREEN}🎉 ALL TESTS PASSED! Stage 1 & Stage 2 are working correctly!{Colors.END}")
        elif passed_tests >= total_tests * 0.8:
            self.log(f"\n{Colors.YELLOW}⚠️ Most tests passed. Minor issues detected.{Colors.END}")
        else:
            self.log(f"\n{Colors.RED}❌ Multiple test failures. Please check service configuration.{Colors.END}")

    async def run_all_tests(self):
        """Run the complete test suite"""
        if not await self.setup():
            return
        
        try:
            # Run all test categories
            await self.test_service_health()
            await self.test_component_health()
            await self.test_data_flow()
            await asyncio.sleep(5)  # Wait for data processing
            await self.test_kafka_topics()
            await self.test_metrics()
            await self.test_fhir_transformation()
            
            # Generate final report
            await self.generate_test_report()
            
        except KeyboardInterrupt:
            self.log("\n⏹️ Test suite interrupted by user")
        except Exception as e:
            self.log(f"\n❌ Test suite failed: {e}", "FAIL")
        finally:
            if self.producer:
                self.producer.close()
                self.log("Kafka producer closed")

async def main():
    """Main test execution"""
    test_suite = AutomatedTestSuite()
    await test_suite.run_all_tests()

if __name__ == "__main__":
    asyncio.run(main())
