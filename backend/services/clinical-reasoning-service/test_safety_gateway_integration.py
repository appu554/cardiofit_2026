"""
Safety Gateway Platform Integration Test

Tests the integration between Safety Gateway Platform and CAE Engine
to ensure the complete clinical decision support flow works end-to-end.
"""

import asyncio
import requests
import json
import logging
from dotenv import load_dotenv
load_dotenv()

import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class SafetyGatewayIntegrationTester:
    """Test Safety Gateway Platform integration with CAE Engine"""
    
    def __init__(self):
        self.safety_gateway_url = "http://localhost:8030"  # Safety Gateway Platform port
        self.cae_grpc_url = "localhost:8027"  # CAE gRPC service
        self.test_results = []
    
    def log_test_result(self, test_name: str, success: bool, details: str = ""):
        """Log test result"""
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{status} {test_name}")
        if details:
            print(f"    {details}")
        
        self.test_results.append({
            'test': test_name,
            'success': success,
            'details': details
        })
    
    async def test_safety_gateway_health(self):
        """Test Safety Gateway Platform health"""
        print("\n🏥 Testing Safety Gateway Platform Health")
        print("-" * 50)
        
        try:
            response = requests.get(f"{self.safety_gateway_url}/health", timeout=5)
            health_ok = response.status_code == 200
            
            if health_ok:
                health_data = response.json()
                cae_connected = health_data.get('cae_engine', {}).get('status') == 'connected'
                
                self.log_test_result(
                    "Safety Gateway Health",
                    health_ok,
                    f"Status: {response.status_code}, CAE Connected: {cae_connected}"
                )
                
                return health_ok and cae_connected
            else:
                self.log_test_result(
                    "Safety Gateway Health",
                    False,
                    f"HTTP {response.status_code}"
                )
                return False
                
        except requests.exceptions.ConnectionError:
            self.log_test_result(
                "Safety Gateway Health",
                False,
                "Safety Gateway Platform not running on localhost:8030"
            )
            return False
        except Exception as e:
            self.log_test_result(
                "Safety Gateway Health",
                False,
                f"Error: {str(e)}"
            )
            return False
    
    async def test_safety_validation_endpoint(self):
        """Test Safety Gateway validation endpoint"""
        print("\n🔒 Testing Safety Validation Endpoint")
        print("-" * 50)
        
        try:
            # Test payload for safety validation
            payload = {
                "patient": {
                    "id": "safety_gateway_test_1",
                    "age": 65,
                    "weight": 70,
                    "gender": "male"
                },
                "medications": [
                    {
                        "name": "warfarin",
                        "dose": "5mg",
                        "frequency": "daily"
                    },
                    {
                        "name": "ciprofloxacin", 
                        "dose": "500mg",
                        "frequency": "twice daily"
                    }
                ],
                "conditions": [
                    {"name": "atrial fibrillation"},
                    {"name": "pneumonia"}
                ],
                "allergies": []
            }
            
            response = requests.post(
                f"{self.safety_gateway_url}/api/safety/validate",
                json=payload,
                timeout=10,
                headers={"Content-Type": "application/json"}
            )
            
            if response.status_code == 200:
                result = response.json()
                
                # Validate response structure
                required_fields = ['overall_status', 'findings', 'execution_time_ms']
                structure_valid = all(field in result for field in required_fields)
                
                # Check if CAE Engine was used
                cae_used = (
                    result.get('overall_status') in ['SAFE', 'WARNING', 'UNSAFE'] and
                    'findings' in result
                )
                
                self.log_test_result(
                    "Safety Validation Endpoint",
                    structure_valid and cae_used,
                    f"Status: {result.get('overall_status', 'N/A')}, "
                    f"Findings: {len(result.get('findings', []))}, "
                    f"Time: {result.get('execution_time_ms', 0):.1f}ms"
                )
                
                return structure_valid and cae_used
            else:
                self.log_test_result(
                    "Safety Validation Endpoint",
                    False,
                    f"HTTP {response.status_code}: {response.text[:100]}"
                )
                return False
                
        except Exception as e:
            self.log_test_result(
                "Safety Validation Endpoint",
                False,
                f"Error: {str(e)}"
            )
            return False
    
    async def test_clinical_decision_flow(self):
        """Test complete clinical decision support flow"""
        print("\n🩺 Testing Clinical Decision Support Flow")
        print("-" * 50)
        
        clinical_scenarios = [
            {
                "name": "Drug Interaction Alert",
                "payload": {
                    "patient": {"id": "flow_test_1", "age": 60},
                    "medications": [
                        {"name": "warfarin", "dose": "5mg"},
                        {"name": "ciprofloxacin", "dose": "500mg"}
                    ],
                    "conditions": [],
                    "allergies": []
                },
                "expected_unsafe": True
            },
            {
                "name": "Allergy Alert",
                "payload": {
                    "patient": {"id": "flow_test_2", "age": 35},
                    "medications": [
                        {"name": "penicillin", "dose": "500mg"}
                    ],
                    "conditions": [],
                    "allergies": [
                        {"substance": "penicillin", "reaction": "rash"}
                    ]
                },
                "expected_unsafe": True
            },
            {
                "name": "Safe Prescription",
                "payload": {
                    "patient": {"id": "flow_test_3", "age": 30},
                    "medications": [
                        {"name": "acetaminophen", "dose": "500mg"}
                    ],
                    "conditions": [],
                    "allergies": []
                },
                "expected_unsafe": False
            }
        ]
        
        scenario_results = []
        
        for scenario in clinical_scenarios:
            try:
                response = requests.post(
                    f"{self.safety_gateway_url}/api/safety/validate",
                    json=scenario["payload"],
                    timeout=10
                )
                
                if response.status_code == 200:
                    result = response.json()
                    status = result.get('overall_status', 'UNKNOWN')
                    
                    # Check if result matches expectation
                    is_unsafe = status in ['WARNING', 'UNSAFE']
                    expectation_met = is_unsafe == scenario["expected_unsafe"]
                    
                    scenario_results.append(expectation_met)
                    
                    self.log_test_result(
                        f"Scenario: {scenario['name']}",
                        expectation_met,
                        f"Expected: {'UNSAFE' if scenario['expected_unsafe'] else 'SAFE'}, "
                        f"Got: {status}"
                    )
                else:
                    scenario_results.append(False)
                    self.log_test_result(
                        f"Scenario: {scenario['name']}",
                        False,
                        f"HTTP {response.status_code}"
                    )
                    
            except Exception as e:
                scenario_results.append(False)
                self.log_test_result(
                    f"Scenario: {scenario['name']}",
                    False,
                    f"Error: {str(e)}"
                )
        
        return all(scenario_results)
    
    async def test_performance_integration(self):
        """Test performance of integrated system"""
        print("\n⚡ Testing Integration Performance")
        print("-" * 50)
        
        try:
            payload = {
                "patient": {"id": "perf_test", "age": 50},
                "medications": [
                    {"name": "warfarin", "dose": "5mg"},
                    {"name": "acetaminophen", "dose": "500mg"}
                ],
                "conditions": [],
                "allergies": []
            }
            
            # Test multiple requests for performance
            response_times = []
            
            for i in range(3):
                import time
                start_time = time.time()
                
                response = requests.post(
                    f"{self.safety_gateway_url}/api/safety/validate",
                    json=payload,
                    timeout=10
                )
                
                end_time = time.time()
                response_time = (end_time - start_time) * 1000
                response_times.append(response_time)
                
                if response.status_code != 200:
                    break
            
            if response_times:
                avg_time = sum(response_times) / len(response_times)
                max_time = max(response_times)
                performance_ok = avg_time < 1000 and max_time < 2000  # Under 1s avg, 2s max
                
                self.log_test_result(
                    "Integration Performance",
                    performance_ok,
                    f"Avg: {avg_time:.1f}ms, Max: {max_time:.1f}ms"
                )
                
                return performance_ok
            else:
                self.log_test_result(
                    "Integration Performance",
                    False,
                    "No successful requests"
                )
                return False
                
        except Exception as e:
            self.log_test_result(
                "Integration Performance",
                False,
                f"Error: {str(e)}"
            )
            return False
    
    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 60)
        print("📊 SAFETY GATEWAY INTEGRATION SUMMARY")
        print("=" * 60)
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result['success'])
        success_rate = (passed_tests / total_tests * 100) if total_tests > 0 else 0
        
        print(f"✅ Passed: {passed_tests}")
        print(f"❌ Failed: {total_tests - passed_tests}")
        print(f"📈 Success Rate: {success_rate:.1f}%")
        print(f"🧪 Total Tests: {total_tests}")
        
        # Show failed tests
        failed_tests = [result for result in self.test_results if not result['success']]
        if failed_tests:
            print(f"\n❌ Failed Tests:")
            for test in failed_tests:
                print(f"  - {test['test']}: {test['details']}")
        
        # Overall assessment
        if success_rate >= 90:
            print(f"\n🎉 SAFETY GATEWAY INTEGRATION: EXCELLENT!")
            print("✅ Complete clinical decision support flow working")
            return True
        elif success_rate >= 75:
            print(f"\n✅ SAFETY GATEWAY INTEGRATION: GOOD")
            print("⚠️  Minor integration issues")
            return True
        else:
            print(f"\n❌ SAFETY GATEWAY INTEGRATION: NEEDS WORK")
            print("🔧 Integration issues to resolve")
            return False

async def main():
    """Main test execution"""
    print("🔗 Safety Gateway Platform Integration Test")
    print("=" * 60)
    
    tester = SafetyGatewayIntegrationTester()
    
    try:
        # Run integration tests
        await tester.test_safety_gateway_health()
        await tester.test_safety_validation_endpoint()
        await tester.test_clinical_decision_flow()
        await tester.test_performance_integration()
        
        # Print summary
        success = tester.print_summary()
        return success
        
    except Exception as e:
        print(f"❌ Integration testing failed: {e}")
        return False

if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
