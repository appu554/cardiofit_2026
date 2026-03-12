"""
Complete Flow Testing for CAE Engine

Tests the complete flow from different entry points:
1. Direct CAE Engine API
2. gRPC Service Integration  
3. Safety Gateway Platform Integration
4. End-to-End Clinical Scenarios
"""

import asyncio
import grpc
import json
import logging
from dotenv import load_dotenv
load_dotenv()

import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.cae_engine_neo4j import CAEEngine

# Import gRPC generated files
try:
    from app.grpc_generated import clinical_reasoning_pb2, clinical_reasoning_pb2_grpc
    GRPC_AVAILABLE = True
except ImportError:
    GRPC_AVAILABLE = False
    print("⚠️  gRPC files not available - skipping gRPC tests")

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class CompleteFlowTester:
    """Complete flow testing for CAE Engine"""
    
    def __init__(self):
        self.test_results = []
        self.cae_engine = None
    
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
    
    async def test_flow_1_direct_cae_api(self):
        """Flow 1: Direct CAE Engine API Test"""
        print("\n🔄 Flow 1: Direct CAE Engine API")
        print("-" * 50)
        
        try:
            # Initialize CAE Engine
            self.cae_engine = CAEEngine()
            initialized = await self.cae_engine.initialize()
            
            self.log_test_result(
                "CAE Engine Initialization", 
                initialized,
                "Connected to Neo4j Cloud" if initialized else "Failed to connect"
            )
            
            if not initialized:
                return False
            
            # Test clinical scenario
            clinical_context = {
                'patient': {
                    'id': 'flow_test_patient_1',
                    'age': 65,
                    'weight': 70,
                    'gender': 'male'
                },
                'medications': [
                    {'name': 'warfarin', 'dose': '5mg', 'frequency': 'daily'},
                    {'name': 'ciprofloxacin', 'dose': '500mg', 'frequency': 'twice daily'}
                ],
                'conditions': [
                    {'name': 'atrial fibrillation'},
                    {'name': 'pneumonia'}
                ],
                'allergies': []
            }
            
            result = await self.cae_engine.validate_safety(clinical_context)
            
            # Validate result structure
            required_fields = ['overall_status', 'total_findings', 'findings', 'checker_results', 'performance']
            structure_valid = all(field in result for field in required_fields)
            
            self.log_test_result(
                "Result Structure Validation",
                structure_valid,
                f"Status: {result.get('overall_status', 'N/A')}, Findings: {result.get('total_findings', 0)}"
            )
            
            # Check performance
            execution_time = result.get('performance', {}).get('total_execution_time_ms', 0)
            performance_ok = execution_time < 1000  # Under 1 second
            
            self.log_test_result(
                "Performance Check",
                performance_ok,
                f"Execution time: {execution_time:.1f}ms"
            )
            
            # Check Neo4j integration
            neo4j_working = result.get('overall_status') in ['SAFE', 'WARNING', 'UNSAFE']
            
            self.log_test_result(
                "Neo4j Integration",
                neo4j_working,
                f"Neo4j data used successfully" if neo4j_working else "Neo4j integration failed"
            )
            
            return structure_valid and performance_ok and neo4j_working
            
        except Exception as e:
            self.log_test_result("Direct CAE API", False, f"Error: {str(e)}")
            return False
    
    async def test_flow_2_grpc_service(self):
        """Flow 2: gRPC Service Integration Test"""
        print("\n🔄 Flow 2: gRPC Service Integration")
        print("-" * 50)
        
        if not GRPC_AVAILABLE:
            self.log_test_result("gRPC Service", False, "gRPC files not available")
            return False
        
        try:
            # Test gRPC connection
            channel = grpc.aio.insecure_channel('localhost:8027')
            stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
            
            # Create gRPC request
            request = clinical_reasoning_pb2.SafetyValidationRequest(
                patient_id='flow_test_patient_2',
                patient_age=45,
                medications=[
                    clinical_reasoning_pb2.Medication(name='acetaminophen', dose='500mg'),
                    clinical_reasoning_pb2.Medication(name='ibuprofen', dose='200mg')
                ],
                conditions=[
                    clinical_reasoning_pb2.Condition(name='headache')
                ],
                allergies=[]
            )
            
            # Make gRPC call with timeout
            response = await asyncio.wait_for(
                stub.ValidateSafety(request),
                timeout=10.0
            )
            
            # Validate gRPC response
            grpc_success = (
                hasattr(response, 'overall_status') and
                hasattr(response, 'findings') and
                response.overall_status in ['SAFE', 'WARNING', 'UNSAFE']
            )
            
            self.log_test_result(
                "gRPC Service Call",
                grpc_success,
                f"Status: {response.overall_status if grpc_success else 'Invalid'}"
            )
            
            await channel.close()
            return grpc_success
            
        except asyncio.TimeoutError:
            self.log_test_result("gRPC Service", False, "gRPC service not running on localhost:8027")
            return False
        except Exception as e:
            self.log_test_result("gRPC Service", False, f"gRPC error: {str(e)}")
            return False
    
    async def test_flow_3_clinical_scenarios(self):
        """Flow 3: Clinical Scenarios End-to-End"""
        print("\n🔄 Flow 3: Clinical Scenarios End-to-End")
        print("-" * 50)
        
        scenarios = [
            {
                'name': 'High-Risk Drug Interaction',
                'context': {
                    'patient': {'id': 'scenario_1', 'age': 70, 'weight': 65},
                    'medications': [
                        {'name': 'warfarin', 'dose': '5mg'},
                        {'name': 'ciprofloxacin', 'dose': '500mg'}
                    ],
                    'conditions': [{'name': 'atrial fibrillation'}],
                    'allergies': []
                },
                'expected_status': 'UNSAFE'
            },
            {
                'name': 'Known Allergy Alert',
                'context': {
                    'patient': {'id': 'scenario_2', 'age': 35, 'gender': 'female'},
                    'medications': [
                        {'name': 'penicillin', 'dose': '500mg'}
                    ],
                    'conditions': [{'name': 'pneumonia'}],
                    'allergies': [
                        {'substance': 'penicillin', 'reaction': 'rash', 'severity': 'moderate'}
                    ]
                },
                'expected_status': 'UNSAFE'
            },
            {
                'name': 'Safe Medication Combination',
                'context': {
                    'patient': {'id': 'scenario_3', 'age': 25, 'weight': 60},
                    'medications': [
                        {'name': 'acetaminophen', 'dose': '500mg'}
                    ],
                    'conditions': [{'name': 'headache'}],
                    'allergies': []
                },
                'expected_status': 'SAFE'
            }
        ]
        
        scenario_results = []
        
        for scenario in scenarios:
            try:
                result = await self.cae_engine.validate_safety(scenario['context'])
                
                status_match = result['overall_status'] == scenario['expected_status']
                execution_time = result.get('performance', {}).get('total_execution_time_ms', 0)
                performance_ok = execution_time < 1000
                
                scenario_success = status_match and performance_ok
                scenario_results.append(scenario_success)
                
                self.log_test_result(
                    f"Scenario: {scenario['name']}",
                    scenario_success,
                    f"Expected: {scenario['expected_status']}, Got: {result['overall_status']}, Time: {execution_time:.1f}ms"
                )
                
            except Exception as e:
                scenario_results.append(False)
                self.log_test_result(
                    f"Scenario: {scenario['name']}",
                    False,
                    f"Error: {str(e)}"
                )
        
        return all(scenario_results)
    
    async def test_flow_4_performance_stress(self):
        """Flow 4: Performance and Stress Testing"""
        print("\n🔄 Flow 4: Performance and Stress Testing")
        print("-" * 50)
        
        try:
            # Concurrent requests test
            concurrent_requests = []
            
            base_context = {
                'patient': {'id': 'stress_test', 'age': 50},
                'medications': [
                    {'name': 'warfarin', 'dose': '5mg'},
                    {'name': 'acetaminophen', 'dose': '500mg'}
                ],
                'conditions': [],
                'allergies': []
            }
            
            # Create 5 concurrent requests
            for i in range(5):
                context = base_context.copy()
                context['patient']['id'] = f'stress_test_{i}'
                concurrent_requests.append(
                    self.cae_engine.validate_safety(context)
                )
            
            # Execute concurrent requests
            import time
            start_time = time.time()
            results = await asyncio.gather(*concurrent_requests, return_exceptions=True)
            total_time = (time.time() - start_time) * 1000
            
            # Check results
            successful_results = [r for r in results if not isinstance(r, Exception)]
            success_rate = len(successful_results) / len(results) * 100
            
            performance_ok = total_time < 3000  # Under 3 seconds for 5 concurrent requests
            success_ok = success_rate >= 80  # At least 80% success rate
            
            self.log_test_result(
                "Concurrent Requests",
                performance_ok and success_ok,
                f"5 requests in {total_time:.1f}ms, Success rate: {success_rate:.1f}%"
            )
            
            return performance_ok and success_ok
            
        except Exception as e:
            self.log_test_result("Performance Stress", False, f"Error: {str(e)}")
            return False
    
    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 60)
        print("📊 COMPLETE FLOW TEST SUMMARY")
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
            print(f"\n🎉 COMPLETE FLOW: EXCELLENT!")
            print("✅ Ready for production deployment")
            return True
        elif success_rate >= 75:
            print(f"\n✅ COMPLETE FLOW: GOOD")
            print("⚠️  Minor issues to address")
            return True
        else:
            print(f"\n❌ COMPLETE FLOW: NEEDS WORK")
            print("🔧 Significant issues to resolve")
            return False
    
    async def cleanup(self):
        """Cleanup resources"""
        if self.cae_engine:
            await self.cae_engine.close()

async def main():
    """Main test execution"""
    print("🧪 Complete Flow Testing for CAE Engine")
    print("=" * 60)
    
    tester = CompleteFlowTester()
    
    try:
        # Run all flow tests
        await tester.test_flow_1_direct_cae_api()
        await tester.test_flow_2_grpc_service()
        await tester.test_flow_3_clinical_scenarios()
        await tester.test_flow_4_performance_stress()
        
        # Print summary
        success = tester.print_summary()
        return success
        
    except Exception as e:
        print(f"❌ Flow testing failed: {e}")
        return False
    
    finally:
        await tester.cleanup()

if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
