#!/usr/bin/env python3
"""
Test script for local PostgreSQL connection
Tests the KB-Drug-Rules service with local PostgreSQL database
"""

import requests
import json
import time
import sys

# Configuration
KB_SERVICE_URL = "http://localhost:8081"
HEALTH_ENDPOINT = f"{KB_SERVICE_URL}/health"
DRUG_RULES_ENDPOINT = f"{KB_SERVICE_URL}/v1/items"
VALIDATE_ENDPOINT = f"{KB_SERVICE_URL}/v1/validate"

def test_health_check():
    """Test the health check endpoint"""
    print("🔍 Testing health check...")
    try:
        response = requests.get(HEALTH_ENDPOINT, timeout=10)
        if response.status_code == 200:
            health_data = response.json()
            print(f"✅ Health check passed: {health_data['status']}")
            
            # Check database connection
            if 'checks' in health_data and 'database' in health_data['checks']:
                db_status = health_data['checks']['database']
                print(f"   📊 Database: {db_status}")
            
            # Check cache connection
            if 'checks' in health_data and 'cache' in health_data['checks']:
                cache_status = health_data['checks']['cache']
                print(f"   🗄️  Cache: {cache_status}")
                
            return True
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Health check error: {e}")
        return False

def test_get_drug_rules():
    """Test getting drug rules"""
    print("\n🔍 Testing drug rules retrieval...")
    
    # Test drugs from sample data
    test_drugs = ["metformin", "lisinopril"]
    
    for drug in test_drugs:
        try:
            response = requests.get(f"{DRUG_RULES_ENDPOINT}/{drug}", timeout=10)
            if response.status_code == 200:
                drug_data = response.json()
                print(f"✅ Retrieved {drug} rules:")
                print(f"   📋 Drug: {drug_data.get('drug_id')}")
                print(f"   🔢 Version: {drug_data.get('version')}")
                print(f"   🌍 Regions: {drug_data.get('regions')}")
                print(f"   ✅ Signature Valid: {drug_data.get('signature_valid')}")
                
                # Check content structure
                content = drug_data.get('content', {})
                if 'meta' in content:
                    print(f"   💊 Drug Name: {content['meta'].get('drug_name')}")
                    print(f"   🏥 Class: {content['meta'].get('therapeutic_class')}")
                
                if 'dose_calculation' in content:
                    dose_calc = content['dose_calculation']
                    print(f"   💉 Base Formula: {dose_calc.get('base_formula')}")
                    print(f"   📊 Max Daily Dose: {dose_calc.get('max_daily_dose')}")
                
            elif response.status_code == 404:
                print(f"⚠️  {drug} not found (expected for new database)")
            else:
                print(f"❌ Failed to get {drug}: {response.status_code}")
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Error getting {drug}: {e}")

def test_validate_rules():
    """Test rule validation"""
    print("\n🔍 Testing rule validation...")
    
    # Sample TOML content for validation
    sample_toml = """
[meta]
drug_name = "Test Drug"
therapeutic_class = ["Test Class"]
evidence_sources = ["Test Guidelines 2024"]
last_major_update = "2024-01-01T00:00:00Z"
update_rationale = "Test validation"

[dose_calculation]
base_formula = "100mg daily"
max_daily_dose = 200.0
min_daily_dose = 50.0

[[dose_calculation.adjustment_factors]]
factor = "age"
condition = "age > 65"
multiplier = 0.8

[safety_verification]
contraindications = []
warnings = []
precautions = []
interaction_checks = []
lab_monitoring = []

monitoring_requirements = []
regional_variations = {}
    """
    
    validation_request = {
        "content": sample_toml.strip(),
        "regions": ["US"]
    }
    
    try:
        response = requests.post(
            VALIDATE_ENDPOINT, 
            json=validation_request,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 200:
            validation_result = response.json()
            print(f"✅ Validation successful:")
            print(f"   ✅ Valid: {validation_result.get('valid')}")
            print(f"   ❌ Errors: {len(validation_result.get('errors', []))}")
            print(f"   ⚠️  Warnings: {len(validation_result.get('warnings', []))}")
            print(f"   ℹ️  Info: {len(validation_result.get('info', []))}")
            
            if validation_result.get('errors'):
                print("   Error details:")
                for error in validation_result['errors']:
                    print(f"     - {error}")
                    
        else:
            print(f"❌ Validation failed: {response.status_code}")
            print(f"   Response: {response.text}")
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Validation error: {e}")

def test_metrics():
    """Test metrics endpoint"""
    print("\n🔍 Testing metrics...")
    try:
        response = requests.get(f"{KB_SERVICE_URL}/metrics", timeout=10)
        if response.status_code == 200:
            metrics_text = response.text
            # Count metrics
            metric_lines = [line for line in metrics_text.split('\n') if line and not line.startswith('#')]
            print(f"✅ Metrics endpoint working: {len(metric_lines)} metrics available")
        else:
            print(f"❌ Metrics failed: {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"❌ Metrics error: {e}")

def main():
    """Main test function"""
    print("🧪 KB-Drug-Rules Local PostgreSQL Test")
    print("=" * 50)
    
    # Wait for service to be ready
    print("⏳ Waiting for service to be ready...")
    max_retries = 30
    for i in range(max_retries):
        try:
            response = requests.get(HEALTH_ENDPOINT, timeout=5)
            if response.status_code == 200:
                print("✅ Service is ready!")
                break
        except:
            pass
        
        if i < max_retries - 1:
            print(f"   Retry {i+1}/{max_retries}...")
            time.sleep(2)
    else:
        print("❌ Service not ready after 60 seconds")
        sys.exit(1)
    
    # Run tests
    success = True
    
    success &= test_health_check()
    test_get_drug_rules()
    test_validate_rules()
    test_metrics()
    
    print("\n" + "=" * 50)
    if success:
        print("🎉 All critical tests passed!")
        print("\n📋 Next steps:")
        print("   1. Check Adminer: http://localhost:8080")
        print("   2. Check Grafana: http://localhost:3000 (admin/admin)")
        print("   3. Check Prometheus: http://localhost:9090")
        print("   4. Test API manually with curl or Postman")
    else:
        print("❌ Some tests failed. Check the service logs.")
        sys.exit(1)

if __name__ == "__main__":
    main()
