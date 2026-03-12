#!/usr/bin/env python3
"""
Test script for Device Data Ingestion Service
Tests all endpoints and functionality
"""
import asyncio
import json
import time
from typing import Dict, Any
import httpx
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Test configuration
BASE_URL = "http://localhost:8015"
API_KEY_1 = "dv1_test_key_12345"  # Heart rate, blood pressure, blood glucose
API_KEY_2 = "dv2_test_key_67890"  # Temperature, oxygen saturation, weight

# Test data
SAMPLE_DEVICE_READINGS = [
    {
        "device_id": "hr-monitor-001",
        "timestamp": int(time.time()),
        "reading_type": "heart_rate",
        "value": 72.5,
        "unit": "bpm",
        "patient_id": "patient-12345",
        "metadata": {
            "battery_level": 85,
            "signal_quality": "good"
        }
    },
    {
        "device_id": "bp-monitor-002", 
        "timestamp": int(time.time()),
        "reading_type": "blood_pressure_systolic",
        "value": 120.0,
        "unit": "mmHg",
        "patient_id": "patient-12345"
    },
    {
        "device_id": "glucose-meter-003",
        "timestamp": int(time.time()),
        "reading_type": "blood_glucose",
        "value": 95.0,
        "unit": "mg/dL",
        "patient_id": "patient-67890"
    },
    {
        "device_id": "temp-sensor-004",
        "timestamp": int(time.time()),
        "reading_type": "temperature",
        "value": 98.6,
        "unit": "°F",
        "patient_id": "patient-67890"
    }
]


async def test_health_endpoint():
    """Test the health check endpoint"""
    logger.info("Testing health endpoint...")
    
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(f"{BASE_URL}/api/v1/health")
            
            if response.status_code == 200:
                data = response.json()
                logger.info(f"✓ Health check passed: {data['status']}")
                logger.info(f"  Kafka connected: {data.get('kafka_connected', 'unknown')}")
                return True
            else:
                logger.error(f"✗ Health check failed: {response.status_code}")
                return False
                
    except Exception as e:
        logger.error(f"✗ Health check error: {e}")
        return False


async def test_single_ingestion():
    """Test single device data ingestion"""
    logger.info("Testing single device data ingestion...")
    
    try:
        async with httpx.AsyncClient() as client:
            # Test with valid API key and allowed device type
            headers = {
                "X-API-Key": API_KEY_1,
                "Content-Type": "application/json"
            }
            
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/device-data",
                headers=headers,
                json=SAMPLE_DEVICE_READINGS[0]
            )
            
            if response.status_code == 200:
                data = response.json()
                logger.info(f"✓ Single ingestion successful: {data['status']}")
                logger.info(f"  Ingestion ID: {data.get('ingestion_id')}")
                return True
            else:
                logger.error(f"✗ Single ingestion failed: {response.status_code}")
                logger.error(f"  Response: {response.text}")
                return False
                
    except Exception as e:
        logger.error(f"✗ Single ingestion error: {e}")
        return False


async def test_batch_ingestion():
    """Test batch device data ingestion"""
    logger.info("Testing batch device data ingestion...")
    
    try:
        async with httpx.AsyncClient() as client:
            # Test batch with mixed API keys (should use first valid one)
            headers = {
                "X-API-Key": API_KEY_1,
                "Content-Type": "application/json"
            }
            
            # Use only readings allowed for API_KEY_1
            allowed_readings = [r for r in SAMPLE_DEVICE_READINGS 
                              if r['reading_type'] in ['heart_rate', 'blood_pressure_systolic', 'blood_glucose']]
            
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/batch-device-data",
                headers=headers,
                json=allowed_readings
            )
            
            if response.status_code == 200:
                data = response.json()
                logger.info(f"✓ Batch ingestion successful: {data['status']}")
                logger.info(f"  Total: {data['total_readings']}, Successful: {data['successful']}, Failed: {data['failed']}")
                return True
            else:
                logger.error(f"✗ Batch ingestion failed: {response.status_code}")
                logger.error(f"  Response: {response.text}")
                return False
                
    except Exception as e:
        logger.error(f"✗ Batch ingestion error: {e}")
        return False


async def test_authentication():
    """Test authentication and authorization"""
    logger.info("Testing authentication and authorization...")
    
    try:
        async with httpx.AsyncClient() as client:
            # Test 1: No API key
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/device-data",
                json=SAMPLE_DEVICE_READINGS[0]
            )
            
            if response.status_code == 401:
                logger.info("✓ No API key correctly rejected")
            else:
                logger.error(f"✗ No API key should be rejected, got: {response.status_code}")
                return False
            
            # Test 2: Invalid API key
            headers = {"X-API-Key": "invalid-key"}
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/device-data",
                headers=headers,
                json=SAMPLE_DEVICE_READINGS[0]
            )
            
            if response.status_code == 401:
                logger.info("✓ Invalid API key correctly rejected")
            else:
                logger.error(f"✗ Invalid API key should be rejected, got: {response.status_code}")
                return False
            
            # Test 3: Valid API key but unauthorized device type
            headers = {"X-API-Key": API_KEY_1}  # Only allows heart_rate, blood_pressure, blood_glucose
            temp_reading = {
                "device_id": "temp-001",
                "timestamp": int(time.time()),
                "reading_type": "temperature",  # Not allowed for API_KEY_1
                "value": 98.6,
                "unit": "°F"
            }
            
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/device-data",
                headers=headers,
                json=temp_reading
            )
            
            if response.status_code == 403:
                logger.info("✓ Unauthorized device type correctly rejected")
            else:
                logger.error(f"✗ Unauthorized device type should be rejected, got: {response.status_code}")
                return False
            
            return True
            
    except Exception as e:
        logger.error(f"✗ Authentication test error: {e}")
        return False


async def test_validation():
    """Test input validation"""
    logger.info("Testing input validation...")
    
    try:
        async with httpx.AsyncClient() as client:
            headers = {"X-API-Key": API_KEY_1}
            
            # Test 1: Missing required field
            invalid_reading = {
                "device_id": "test-001",
                # Missing timestamp
                "reading_type": "heart_rate",
                "value": 72.0,
                "unit": "bpm"
            }
            
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/device-data",
                headers=headers,
                json=invalid_reading
            )
            
            if response.status_code == 422:
                logger.info("✓ Missing field correctly rejected")
            else:
                logger.error(f"✗ Missing field should be rejected, got: {response.status_code}")
                return False
            
            # Test 2: Invalid reading type
            invalid_reading = {
                "device_id": "test-001",
                "timestamp": int(time.time()),
                "reading_type": "invalid_type",
                "value": 72.0,
                "unit": "bpm"
            }
            
            response = await client.post(
                f"{BASE_URL}/api/v1/ingest/device-data",
                headers=headers,
                json=invalid_reading
            )
            
            if response.status_code == 422:
                logger.info("✓ Invalid reading type correctly rejected")
            else:
                logger.error(f"✗ Invalid reading type should be rejected, got: {response.status_code}")
                return False
            
            return True
            
    except Exception as e:
        logger.error(f"✗ Validation test error: {e}")
        return False


async def test_metrics_endpoint():
    """Test the metrics endpoint"""
    logger.info("Testing metrics endpoint...")
    
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(f"{BASE_URL}/api/v1/metrics")
            
            if response.status_code == 200:
                data = response.json()
                logger.info(f"✓ Metrics endpoint working: {data['service']}")
                return True
            else:
                logger.error(f"✗ Metrics endpoint failed: {response.status_code}")
                return False
                
    except Exception as e:
        logger.error(f"✗ Metrics endpoint error: {e}")
        return False


async def run_all_tests():
    """Run all tests"""
    logger.info("🚀 Starting Device Data Ingestion Service tests...")
    
    tests = [
        ("Health Check", test_health_endpoint),
        ("Single Ingestion", test_single_ingestion),
        ("Batch Ingestion", test_batch_ingestion),
        ("Authentication", test_authentication),
        ("Validation", test_validation),
        ("Metrics", test_metrics_endpoint)
    ]
    
    results = []
    
    for test_name, test_func in tests:
        logger.info(f"\n--- Running {test_name} Test ---")
        try:
            result = await test_func()
            results.append((test_name, result))
        except Exception as e:
            logger.error(f"Test {test_name} failed with exception: {e}")
            results.append((test_name, False))
    
    # Summary
    logger.info("\n" + "="*50)
    logger.info("TEST RESULTS SUMMARY")
    logger.info("="*50)
    
    passed = 0
    total = len(results)
    
    for test_name, result in results:
        status = "✓ PASS" if result else "✗ FAIL"
        logger.info(f"{test_name:20} {status}")
        if result:
            passed += 1
    
    logger.info("="*50)
    logger.info(f"TOTAL: {passed}/{total} tests passed")
    
    if passed == total:
        logger.info("🎉 All tests passed!")
        return True
    else:
        logger.error("❌ Some tests failed!")
        return False


if __name__ == "__main__":
    success = asyncio.run(run_all_tests())
    exit(0 if success else 1)
