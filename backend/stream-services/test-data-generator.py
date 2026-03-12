#!/usr/bin/env python3
"""
Test Data Generator for Stage 1 & Stage 2 Testing

Generates realistic device reading data and sends it to the raw-device-data.v1 topic
to test the complete pipeline: Stage 1 (Validation) → Stage 2 (Storage Fan-Out)
"""

import json
import time
import random
from datetime import datetime
from typing import Dict, Any
from kafka import KafkaProducer

# Kafka configuration
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

# Test data templates
DEVICE_TYPES = [
    {
        'reading_type': 'heart_rate',
        'unit': 'bpm',
        'normal_range': (60, 100),
        'critical_range': (40, 150)
    },
    {
        'reading_type': 'blood_pressure_systolic',
        'unit': 'mmHg',
        'normal_range': (90, 140),
        'critical_range': (70, 180)
    },
    {
        'reading_type': 'blood_pressure_diastolic',
        'unit': 'mmHg',
        'normal_range': (60, 90),
        'critical_range': (40, 110)
    },
    {
        'reading_type': 'blood_glucose',
        'unit': 'mg/dL',
        'normal_range': (70, 140),
        'critical_range': (50, 200)
    },
    {
        'reading_type': 'temperature',
        'unit': 'F',
        'normal_range': (97.0, 99.5),
        'critical_range': (95.0, 104.0)
    },
    {
        'reading_type': 'oxygen_saturation',
        'unit': '%',
        'normal_range': (95, 100),
        'critical_range': (85, 100)
    },
    {
        'reading_type': 'weight',
        'unit': 'kg',
        'normal_range': (50, 120),
        'critical_range': (30, 200)
    }
]

PATIENTS = [
    '905a60cb-8241-418f-b29b-5b020e851392',  # Your specific patient ID
    'patient-002', 'patient-003', 'patient-004', 'patient-005'
]

DEVICES = [
    'device-heart-monitor-001', 'device-bp-cuff-002', 'device-glucose-meter-003',
    'device-thermometer-004', 'device-pulse-ox-005', 'device-scale-006'
]

VENDORS = [
    {'vendor_id': 'philips', 'vendor_name': 'Philips Healthcare'},
    {'vendor_id': 'ge', 'vendor_name': 'GE Healthcare'},
    {'vendor_id': 'medtronic', 'vendor_name': 'Medtronic'},
    {'vendor_id': 'abbott', 'vendor_name': 'Abbott'}
]


def generate_device_reading(scenario: str = 'normal') -> Dict[str, Any]:
    """Generate a realistic device reading"""
    
    device_type = random.choice(DEVICE_TYPES)
    device_id = random.choice(DEVICES)
    patient_id = random.choice(PATIENTS)
    vendor = random.choice(VENDORS)
    
    # Generate value based on scenario
    if scenario == 'normal':
        value = random.uniform(*device_type['normal_range'])
    elif scenario == 'critical':
        # Generate critical values (outside normal but within critical range)
        if random.choice([True, False]):
            value = random.uniform(device_type['critical_range'][0], device_type['normal_range'][0])
        else:
            value = random.uniform(device_type['normal_range'][1], device_type['critical_range'][1])
    elif scenario == 'emergency':
        # Generate emergency values (outside critical range)
        if random.choice([True, False]):
            value = device_type['critical_range'][0] - random.uniform(5, 20)
        else:
            value = device_type['critical_range'][1] + random.uniform(5, 20)
    elif scenario == 'invalid':
        # Generate invalid data for testing validation
        value = random.choice([None, -999, float('inf'), float('nan')])
    else:
        value = random.uniform(*device_type['normal_range'])
    
    # Round to appropriate precision
    if device_type['reading_type'] in ['temperature']:
        value = round(value, 1) if value is not None and not (isinstance(value, float) and (value != value or value == float('inf'))) else value
    else:
        value = round(value) if value is not None and not (isinstance(value, float) and (value != value or value == float('inf'))) else value
    
    reading = {
        'device_id': device_id,
        'timestamp': int(time.time()),
        'reading_type': device_type['reading_type'],
        'value': value,
        'unit': device_type['unit'],
        'patient_id': patient_id,
        'metadata': {
            'battery_level': random.randint(20, 100),
            'signal_quality': random.choice(['excellent', 'good', 'fair', 'poor']),
            'device_model': f"{device_id.split('-')[1]}-v{random.randint(1, 3)}"
        },
        'vendor_info': vendor
    }
    
    return reading


def generate_invalid_reading() -> Dict[str, Any]:
    """Generate invalid reading for testing validation failures"""
    scenarios = [
        # Missing required fields
        {'device_id': None, 'timestamp': int(time.time()), 'reading_type': 'heart_rate', 'value': 75, 'unit': 'bpm'},
        # Invalid timestamp
        {'device_id': 'device-001', 'timestamp': -1, 'reading_type': 'heart_rate', 'value': 75, 'unit': 'bpm'},
        # Invalid value
        {'device_id': 'device-001', 'timestamp': int(time.time()), 'reading_type': 'heart_rate', 'value': None, 'unit': 'bpm'},
        # Missing unit
        {'device_id': 'device-001', 'timestamp': int(time.time()), 'reading_type': 'heart_rate', 'value': 75, 'unit': None},
    ]
    
    return random.choice(scenarios)


def send_test_data(producer: KafkaProducer, count: int = 10, scenario: str = 'mixed'):
    """Send test data to Kafka"""
    
    topic = 'raw-device-data.v1'
    
    print(f"🚀 Sending {count} test messages to {topic} (scenario: {scenario})")
    
    for i in range(count):
        try:
            if scenario == 'mixed':
                # Mix of different scenarios
                test_scenario = random.choices(
                    ['normal', 'critical', 'emergency', 'invalid'],
                    weights=[70, 20, 5, 5]
                )[0]
            else:
                test_scenario = scenario
            
            if test_scenario == 'invalid':
                reading = generate_invalid_reading()
            else:
                reading = generate_device_reading(test_scenario)
            
            key = reading.get('device_id', f'test-{i}')
            
            # Send to Kafka
            future = producer.send(topic, key=key, value=reading)
            result = future.get(timeout=10)
            
            print(f"✅ Sent message {i+1}/{count}: {reading['reading_type']}={reading['value']} ({test_scenario}) to partition {result.partition}")
            
            # Small delay between messages
            time.sleep(0.1)
            
        except Exception as e:
            print(f"❌ Failed to send message {i+1}: {e}")
    
    print(f"🎉 Completed sending {count} test messages")


def main():
    """Main test function"""
    print("🧪 Stage 1 & Stage 2 Test Data Generator")
    print("=" * 50)
    
    # Initialize Kafka producer
    try:
        producer = KafkaProducer(**KAFKA_CONFIG)
        print("✅ Kafka producer initialized")
    except Exception as e:
        print(f"❌ Failed to initialize Kafka producer: {e}")
        print("💡 Make sure to update KAFKA_CONFIG with your actual API secret")
        return
    
    try:
        # Test scenarios
        print("\n📊 Test Scenarios:")
        print("1. Normal readings (should pass validation)")
        print("2. Critical readings (should pass validation but flagged)")
        print("3. Emergency readings (should pass validation but high priority)")
        print("4. Invalid readings (should fail validation → DLQ)")
        print("5. Mixed scenario (realistic mix)")
        
        scenario = input("\nSelect scenario (1-5) or press Enter for mixed: ").strip()
        
        scenario_map = {
            '1': 'normal',
            '2': 'critical', 
            '3': 'emergency',
            '4': 'invalid',
            '5': 'mixed',
            '': 'mixed'
        }
        
        selected_scenario = scenario_map.get(scenario, 'mixed')
        
        count = input("Number of messages to send (default 10): ").strip()
        count = int(count) if count.isdigit() else 10
        
        # Send test data
        send_test_data(producer, count, selected_scenario)
        
    except KeyboardInterrupt:
        print("\n⏹️ Test interrupted by user")
    except Exception as e:
        print(f"❌ Test failed: {e}")
    finally:
        producer.close()
        print("🔒 Kafka producer closed")


if __name__ == "__main__":
    main()
