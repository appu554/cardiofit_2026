#!/usr/bin/env python3
"""
KB Services Integration Test Suite

This script tests all Knowledge Base services for health, API endpoints,
and cross-service integration scenarios.
"""

import requests
import json
import sys
import time
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from datetime import datetime

# Colors for terminal output
class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    PURPLE = '\033[0;35m'
    CYAN = '\033[0;36m'
    NC = '\033[0m'  # No Color

@dataclass
class TestResult:
    name: str
    passed: bool
    details: Optional[str] = None
    duration: Optional[float] = None

class KBIntegrationTester:
    def __init__(self):
        self.services = {
            'KB-1': 'http://localhost:8081',
            'KB-3': 'http://localhost:8083',
            'KB-4': 'http://localhost:8084',
            'KB-5': 'http://localhost:8085',
            'KB-7': 'http://localhost:8087'
        }
        
        self.results: List[TestResult] = []
        self.session = requests.Session()
        self.session.timeout = 10
        
    def log(self, message: str, color: str = Colors.NC):
        """Log a colored message"""
        print(f"{color}{message}{Colors.NC}")
        
    def test_result(self, name: str, passed: bool, details: str = None, duration: float = None):
        """Record a test result"""
        result = TestResult(name, passed, details, duration)
        self.results.append(result)
        
        status = f"{Colors.GREEN}✅ PASS" if passed else f"{Colors.RED}❌ FAIL"
        duration_str = f" ({duration:.2f}s)" if duration else ""
        
        print(f"{status}{Colors.NC}: {name}{duration_str}")
        if details:
            print(f"   {Colors.YELLOW}Details: {details}{Colors.NC}")
    
    def make_request(self, method: str, url: str, **kwargs) -> Tuple[bool, Optional[Dict], str]:
        """Make HTTP request and return success status, response data, and error message"""
        try:
            start_time = time.time()
            
            if method.upper() == 'GET':
                response = self.session.get(url, **kwargs)
            elif method.upper() == 'POST':
                response = self.session.post(url, **kwargs)
            elif method.upper() == 'PUT':
                response = self.session.put(url, **kwargs)
            else:
                return False, None, f"Unsupported method: {method}"
                
            duration = time.time() - start_time
            
            if response.status_code == 200:
                try:
                    data = response.json()
                    return True, data, ""
                except json.JSONDecodeError:
                    return True, {"raw": response.text}, ""
            else:
                return False, None, f"HTTP {response.status_code}: {response.text[:100]}"
                
        except requests.exceptions.ConnectionError:
            return False, None, "Connection refused - service not running"
        except requests.exceptions.Timeout:
            return False, None, "Request timed out"
        except Exception as e:
            return False, None, f"Request failed: {str(e)}"
    
    def test_service_health(self, service_name: str, base_url: str):
        """Test basic service health and metrics endpoints"""
        self.log(f"Testing {service_name} Health...", Colors.BLUE)
        
        # Health endpoint
        start_time = time.time()
        success, data, error = self.make_request('GET', f"{base_url}/health")
        duration = time.time() - start_time
        
        self.test_result(
            f"{service_name} Health Endpoint",
            success,
            error if not success else None,
            duration
        )
        
        if success and data:
            # Check health status in response
            is_healthy = (
                isinstance(data, dict) and 
                (data.get('status') == 'healthy' or 
                 data.get('success') == True or
                 'healthy' in str(data).lower())
            )
            self.test_result(
                f"{service_name} Health Status",
                is_healthy,
                "Service reports healthy status" if is_healthy else "Service reports unhealthy status"
            )
        
        # Metrics endpoint
        success, _, error = self.make_request('GET', f"{base_url}/metrics")
        self.test_result(
            f"{service_name} Metrics Endpoint",
            success,
            error if not success else "Prometheus metrics available"
        )
    
    def test_kb1_endpoints(self):
        """Test KB-1 Drug Rules specific endpoints"""
        self.log("Testing KB-1 Drug Rules Specific Endpoints...", Colors.BLUE)
        base_url = self.services['KB-1']
        
        # Test drug rules retrieval
        success, data, error = self.make_request('GET', f"{base_url}/v1/items/metformin")
        self.test_result(
            "KB-1 Drug Rules Query (metformin)",
            success,
            error if not success else "Drug rules retrieved successfully"
        )
        
        # Test rules validation
        validation_payload = {
            "content": '[meta]\\ndrug_name="Test"\\ntherapeutic_class=["Test"]\\n[dose_calculation]\\nbase_formula="100mg"\\nmax_daily_dose=200.0\\nmin_daily_dose=50.0\\n[safety_verification]\\ncontraindications=[]\\nwarnings=[]\\nprecautions=[]\\ninteraction_checks=[]\\nlab_monitoring=[]\\nmonitoring_requirements=[]\\nregional_variations={}',
            "regions": ["US"]
        }
        
        success, _, error = self.make_request(
            'POST', 
            f"{base_url}/v1/validate",
            json=validation_payload,
            headers={'Content-Type': 'application/json'}
        )
        self.test_result(
            "KB-1 Rules Validation",
            success,
            error if not success else "Rules validation successful"
        )
    
    def test_kb3_endpoints(self):
        """Test KB-3 Guidelines specific endpoints"""
        self.log("Testing KB-3 Guidelines Specific Endpoints...", Colors.BLUE)
        base_url = self.services['KB-3']
        
        # Test guidelines list
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/guidelines")
        self.test_result(
            "KB-3 Guidelines List",
            success,
            error if not success else "Guidelines retrieved successfully"
        )
        
        # Test evidence search
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/evidence/search?query=diabetes")
        self.test_result(
            "KB-3 Evidence Search",
            success,
            error if not success else "Evidence search successful"
        )
        
        # Test recommendations
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/recommendations")
        self.test_result(
            "KB-3 Clinical Recommendations",
            success,
            error if not success else "Recommendations retrieved successfully"
        )
    
    def test_kb4_endpoints(self):
        """Test KB-4 Patient Safety specific endpoints"""
        self.log("Testing KB-4 Patient Safety Specific Endpoints...", Colors.BLUE)
        base_url = self.services['KB-4']
        
        # Test alerts list
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/alerts")
        self.test_result(
            "KB-4 Safety Alerts List",
            success,
            error if not success else "Safety alerts retrieved successfully"
        )
        
        # Test risk assessment
        risk_payload = {
            "patient_id": "test-patient-123",
            "risk_factors": ["diabetes", "hypertension"],
            "medications": ["metformin", "lisinopril"]
        }
        
        success, _, error = self.make_request(
            'POST',
            f"{base_url}/api/v1/risk-assessment",
            json=risk_payload,
            headers={'Content-Type': 'application/json'}
        )
        self.test_result(
            "KB-4 Risk Assessment",
            success,
            error if not success else "Risk assessment completed successfully"
        )
        
        # Test monitoring rules
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/monitoring/rules")
        self.test_result(
            "KB-4 Monitoring Rules",
            success,
            error if not success else "Monitoring rules retrieved successfully"
        )
    
    def test_kb5_endpoints(self):
        """Test KB-5 Drug Interactions specific endpoints"""
        self.log("Testing KB-5 Drug Interactions Specific Endpoints...", Colors.BLUE)
        base_url = self.services['KB-5']
        
        # Test interaction check
        interaction_payload = {
            "drug_codes": ["warfarin", "aspirin"],
            "check_type": "comprehensive",
            "patient_id": "test-patient-123"
        }
        
        success, _, error = self.make_request(
            'POST',
            f"{base_url}/api/v1/interactions/check",
            json=interaction_payload,
            headers={'Content-Type': 'application/json'}
        )
        self.test_result(
            "KB-5 Interaction Check",
            success,
            error if not success else "Interaction check completed successfully"
        )
        
        # Test quick check
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/interactions/quick-check?drugs=warfarin,aspirin")
        self.test_result(
            "KB-5 Quick Interaction Check",
            success,
            error if not success else "Quick check completed successfully"
        )
        
        # Test drug interactions lookup
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/drugs/warfarin/interactions")
        self.test_result(
            "KB-5 Drug Interactions Lookup",
            success,
            error if not success else "Drug lookup completed successfully"
        )
        
        # Test interaction statistics
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/interactions/statistics")
        self.test_result(
            "KB-5 Interaction Statistics",
            success,
            error if not success else "Statistics retrieved successfully"
        )
    
    def test_kb7_endpoints(self):
        """Test KB-7 Terminology specific endpoints"""
        self.log("Testing KB-7 Terminology Specific Endpoints...", Colors.BLUE)
        base_url = self.services['KB-7']
        
        # Test terminology search
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/terminology/search?query=diabetes&system=snomed")
        self.test_result(
            "KB-7 Terminology Search",
            success,
            error if not success else "Terminology search successful"
        )
        
        # Test code validation
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/terminology/validate/snomed/73211009")
        self.test_result(
            "KB-7 Code Validation",
            success,
            error if not success else "Code validation successful"
        )
        
        # Test mappings
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/terminology/mappings?from_system=icd10&to_system=snomed")
        self.test_result(
            "KB-7 Terminology Mappings",
            success,
            error if not success else "Mappings retrieved successfully"
        )
        
        # Test value sets
        success, _, error = self.make_request('GET', f"{base_url}/api/v1/terminology/valuesets")
        self.test_result(
            "KB-7 Value Sets",
            success,
            error if not success else "Value sets retrieved successfully"
        )
    
    def test_integration_scenarios(self):
        """Test cross-service integration scenarios"""
        self.log("Testing Cross-Service Integration Scenarios...", Colors.BLUE)
        
        # Scenario 1: Drug Rules + Interaction Check
        self.log("Scenario 1: Drug Rules + Interaction Check", Colors.YELLOW)
        
        # Get drug rules from KB-1
        kb1_success, _, _ = self.make_request('GET', f"{self.services['KB-1']}/v1/items/warfarin")
        
        # Check interactions with KB-5
        kb5_success, _, _ = self.make_request('GET', f"{self.services['KB-5']}/api/v1/interactions/quick-check?drugs=warfarin,aspirin")
        
        integration_success = kb1_success and kb5_success
        self.test_result(
            "Integration: Drug Rules + Interactions",
            integration_success,
            "Both services working together" if integration_success else "One or both services failed"
        )
        
        # Scenario 2: Guidelines + Safety Monitoring
        self.log("Scenario 2: Guidelines + Safety Monitoring", Colors.YELLOW)
        
        kb3_success, _, _ = self.make_request('GET', f"{self.services['KB-3']}/api/v1/guidelines")
        kb4_success, _, _ = self.make_request('GET', f"{self.services['KB-4']}/api/v1/alerts")
        
        integration_success = kb3_success and kb4_success
        self.test_result(
            "Integration: Guidelines + Safety Monitoring",
            integration_success,
            "Both services working together" if integration_success else "One or both services failed"
        )
        
        # Scenario 3: Terminology Validation
        self.log("Scenario 3: Cross-Service Terminology Validation", Colors.YELLOW)
        
        kb7_success, _, _ = self.make_request('GET', f"{self.services['KB-7']}/api/v1/terminology/validate/snomed/73211009")
        
        self.test_result(
            "Integration: Terminology Validation",
            kb7_success,
            "Terminology validation working" if kb7_success else "Terminology validation failed"
        )
    
    def check_prerequisites(self) -> bool:
        """Check if prerequisites are met"""
        self.log("Checking prerequisites...", Colors.BLUE)
        
        # Check if any services are running
        running_services = []
        for service, url in self.services.items():
            success, _, _ = self.make_request('GET', f"{url}/health")
            if success:
                running_services.append(service)
        
        if not running_services:
            self.log("Error: No KB services are running. Please start the services first.", Colors.RED)
            self.log("Run: make run-kb-docker", Colors.YELLOW)
            return False
        
        self.log(f"Prerequisites check passed. Found {len(running_services)} running services: {', '.join(running_services)}", Colors.GREEN)
        return True
    
    def print_summary(self):
        """Print test results summary"""
        passed = sum(1 for result in self.results if result.passed)
        failed = len(self.results) - passed
        
        self.log("=" * 50, Colors.BLUE)
        self.log(" Test Results Summary", Colors.BLUE)
        self.log("=" * 50, Colors.BLUE)
        
        print(f"Total Tests: {len(self.results)}")
        print(f"{Colors.GREEN}Passed: {passed}{Colors.NC}")
        print(f"{Colors.RED}Failed: {failed}{Colors.NC}")
        print()
        
        if failed == 0:
            self.log("🎉 All tests passed! KB services are running correctly.", Colors.GREEN)
            return True
        else:
            self.log("⚠️  Some tests failed. Please check the services and try again.", Colors.RED)
            
            # Show failed tests
            failed_tests = [result for result in self.results if not result.passed]
            if failed_tests:
                self.log("\\nFailed tests:", Colors.RED)
                for test in failed_tests:
                    print(f"  - {test.name}")
                    if test.details:
                        print(f"    {test.details}")
            return False
    
    def run_all_tests(self):
        """Run all integration tests"""
        self.log("=" * 50, Colors.BLUE)
        self.log(" KB Services Integration Test Suite", Colors.BLUE)
        self.log("=" * 50, Colors.BLUE)
        print()
        
        if not self.check_prerequisites():
            return False
        
        print()
        
        # Test individual service health
        for service, url in self.services.items():
            self.test_service_health(service, url)
            print()
        
        # Test service-specific endpoints
        self.test_kb1_endpoints()
        print()
        self.test_kb3_endpoints()
        print()
        self.test_kb4_endpoints()
        print()
        self.test_kb5_endpoints()
        print()
        self.test_kb7_endpoints()
        print()
        
        # Test integration scenarios
        self.test_integration_scenarios()
        print()
        
        # Print summary
        return self.print_summary()

def main():
    """Main function"""
    tester = KBIntegrationTester()
    
    try:
        success = tester.run_all_tests()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print(f"\\n{Colors.YELLOW}Test interrupted by user{Colors.NC}")
        sys.exit(1)
    except Exception as e:
        print(f"\\n{Colors.RED}Test suite failed with error: {str(e)}{Colors.NC}")
        sys.exit(1)

if __name__ == "__main__":
    main()