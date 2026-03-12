#!/usr/bin/env python3
"""
Test script to verify the Rust Engine is running and healthy
"""

import requests
import json

def test_health():
    """Test the health endpoint"""
    try:
        response = requests.get("http://localhost:8080/health", timeout=5)
        print(f"Health Check Status: {response.status_code}")
        print(f"Response: {response.text}")
        return response.status_code == 200
    except Exception as e:
        print(f"Health check failed: {e}")
        return False

def test_detailed_health():
    """Test the detailed health endpoint"""
    try:
        response = requests.get("http://localhost:8080/health/detailed", timeout=5)
        print(f"Detailed Health Status: {response.status_code}")
        print(f"Response: {json.dumps(response.json(), indent=2)}")
        return response.status_code == 200
    except Exception as e:
        print(f"Detailed health check failed: {e}")
        return False

def test_status():
    """Test the status endpoint"""
    try:
        response = requests.get("http://localhost:8080/status", timeout=5)
        print(f"Status Check: {response.status_code}")
        print(f"Response: {json.dumps(response.json(), indent=2)}")
        return response.status_code == 200
    except Exception as e:
        print(f"Status check failed: {e}")
        return False

if __name__ == "__main__":
    print("🧪 Testing Rust Engine Health...")
    print("=" * 50)
    
    print("\n1. Basic Health Check:")
    health_ok = test_health()
    
    print("\n2. Detailed Health Check:")
    detailed_ok = test_detailed_health()
    
    print("\n3. Status Check:")
    status_ok = test_status()
    
    print("\n" + "=" * 50)
    if health_ok and detailed_ok and status_ok:
        print("✅ All health checks passed! Engine is running properly.")
    else:
        print("❌ Some health checks failed.")
