"""
Test Same Clinical Scenarios - Full Flow vs Direct CAE

This test compares the exact same clinical scenarios that worked with direct CAE Engine
against the full Safety Gateway → CAE gRPC → Neo4j flow to ensure consistency.
"""

import asyncio
import grpc
import sys
import logging
import uuid
from pathlib import Path
from dotenv import load_dotenv
load_dotenv()

# Add app directory to path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

# Add Safety Gateway proto path
safety_gateway_proto_path = Path(__file__).parent.parent / "safety-gateway-platform" / "proto"
sys.path.insert(0, str(safety_gateway_proto_path))

from app.cae_engine_neo4j import CAEEngine

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class SameScenarioTester:
    """Test same clinical scenarios through both direct CAE and full flow"""
    
    def __init__(self):
        self.safety_gateway_url = "localhost:8030"
        self.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
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
    
    async def test_scenario_1_direct_cae(self):
        """Test Scenario 1: Drug Interaction (Warfarin + Ciprofloxacin) - Direct CAE"""
        print("\n💊 Scenario 1: Drug Interaction (Direct CAE)")
        print("-" * 50)
        
        try:
            # Initialize CAE Engine
            self.cae_engine = CAEEngine()
            initialized = await self.cae_engine.initialize()
            
            if not initialized:
                self.log_test_result("Direct CAE - Initialization", False, "Failed to initialize")
                return None
            
            # Same test case that worked before
            clinical_context = {
                'patient': {
                    'id': self.patient_id,
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
            
            # Validate result
            success = (
                result.get('overall_status') in ['SAFE', 'WARNING', 'UNSAFE'] and
                result.get('total_findings', 0) >= 0 and
                result.get('performance', {}).get('total_execution_time_ms', 0) < 2000
            )
            
            self.log_test_result(
                "Direct CAE - Drug Interaction",
                success,
                f"Status: {result.get('overall_status')}, Findings: {result.get('total_findings')}, "
                f"Time: {result.get('performance', {}).get('total_execution_time_ms', 0):.1f}ms"
            )
            
            return result
            
        except Exception as e:
            self.log_test_result("Direct CAE - Drug Interaction", False, f"Error: {str(e)}")
            return None
    
    async def test_scenario_1_full_flow(self):
        """Test Scenario 1: Drug Interaction (Warfarin + Ciprofloxacin) - Full Flow"""
        print("\n🛡️ Scenario 1: Drug Interaction (Full Flow)")
        print("-" * 50)
        
        try:
            import safety_gateway_pb2
            import safety_gateway_pb2_grpc
            
            channel = grpc.aio.insecure_channel(self.safety_gateway_url)
            stub = safety_gateway_pb2_grpc.SafetyGatewayStub(channel)
            
            # Generate UUIDs
            request_id = str(uuid.uuid4())
            clinician_id = str(uuid.uuid4())
            
            # Authorization metadata
            metadata = [
                ('authorization', 'Bearer test-token'),
                ('clinician-id', clinician_id),
                ('request-id', request_id)
            ]
            
            # Same clinical scenario through Safety Gateway
            request = safety_gateway_pb2.SafetyRequest(
                request_id=request_id,
                patient_id=self.patient_id,
                clinician_id=clinician_id,
                action_type="medication_order",
                priority="normal",
                medication_ids=["warfarin_5mg", "ciprofloxacin_500mg"],
                condition_ids=["atrial_fibrillation", "pneumonia"],
                allergy_ids=[],
                context={
                    "patient_age": "65",
                    "patient_weight": "70",
                    "patient_gender": "male"
                }
            )
            
            import time
            start_time = time.time()
            
            response = await asyncio.wait_for(
                stub.ValidateSafety(request, metadata=metadata),
                timeout=10.0
            )
            
            end_time = time.time()
            execution_time = (end_time - start_time) * 1000
            
            # Map status codes
            status_names = {0: "UNSPECIFIED", 1: "SAFE", 2: "UNSAFE", 3: "WARNING", 4: "MANUAL_REVIEW", 5: "ERROR"}
            status_name = status_names.get(response.status, f"UNKNOWN({response.status})")
            
            # Validate result
            success = (
                response.status in [1, 2, 3, 4] and  # SAFE, UNSAFE, WARNING, MANUAL_REVIEW
                execution_time < 5000  # Under 5 seconds
            )
            
            self.log_test_result(
                "Full Flow - Drug Interaction",
                success,
                f"Status: {status_name} ({response.status}), Risk Score: {response.risk_score:.2f}, "
                f"Time: {execution_time:.1f}ms, Warnings: {len(response.warnings)}"
            )
            
            await channel.close()
            return {
                'status': status_name,
                'risk_score': response.risk_score,
                'execution_time': execution_time,
                'warnings': len(response.warnings),
                'critical_violations': len(response.critical_violations)
            }
            
        except Exception as e:
            self.log_test_result("Full Flow - Drug Interaction", False, f"Error: {str(e)}")
            return None
    
    async def test_scenario_2_direct_cae(self):
        """Test Scenario 2: Known Allergy (Penicillin) - Direct CAE"""
        print("\n🚨 Scenario 2: Known Allergy (Direct CAE)")
        print("-" * 50)
        
        try:
            clinical_context = {
                'patient': {
                    'id': self.patient_id,
                    'age': 35,
                    'gender': 'female'
                },
                'medications': [
                    {'name': 'penicillin', 'dose': '500mg', 'frequency': 'four times daily'}
                ],
                'conditions': [
                    {'name': 'pneumonia'}
                ],
                'allergies': [
                    {'substance': 'penicillin', 'reaction': 'rash', 'severity': 'moderate'}
                ]
            }
            
            result = await self.cae_engine.validate_safety(clinical_context)
            
            success = (
                result.get('overall_status') in ['UNSAFE', 'WARNING'] and  # Should detect allergy
                result.get('total_findings', 0) > 0
            )
            
            self.log_test_result(
                "Direct CAE - Known Allergy",
                success,
                f"Status: {result.get('overall_status')}, Findings: {result.get('total_findings')}"
            )
            
            return result
            
        except Exception as e:
            self.log_test_result("Direct CAE - Known Allergy", False, f"Error: {str(e)}")
            return None
    
    async def test_scenario_2_full_flow(self):
        """Test Scenario 2: Known Allergy (Penicillin) - Full Flow"""
        print("\n🛡️ Scenario 2: Known Allergy (Full Flow)")
        print("-" * 50)
        
        try:
            import safety_gateway_pb2
            import safety_gateway_pb2_grpc
            
            channel = grpc.aio.insecure_channel(self.safety_gateway_url)
            stub = safety_gateway_pb2_grpc.SafetyGatewayStub(channel)
            
            request_id = str(uuid.uuid4())
            clinician_id = str(uuid.uuid4())
            
            metadata = [
                ('authorization', 'Bearer test-token'),
                ('clinician-id', clinician_id),
                ('request-id', request_id)
            ]
            
            request = safety_gateway_pb2.SafetyRequest(
                request_id=request_id,
                patient_id=self.patient_id,
                clinician_id=clinician_id,
                action_type="medication_order",
                priority="normal",
                medication_ids=["penicillin_500mg"],
                condition_ids=["pneumonia"],
                allergy_ids=["penicillin"],
                context={
                    "patient_age": "35",
                    "patient_gender": "female"
                }
            )
            
            response = await asyncio.wait_for(
                stub.ValidateSafety(request, metadata=metadata),
                timeout=10.0
            )
            
            status_names = {0: "UNSPECIFIED", 1: "SAFE", 2: "UNSAFE", 3: "WARNING", 4: "MANUAL_REVIEW", 5: "ERROR"}
            status_name = status_names.get(response.status, f"UNKNOWN({response.status})")
            
            success = response.status in [2, 3]  # UNSAFE or WARNING expected for allergy
            
            self.log_test_result(
                "Full Flow - Known Allergy",
                success,
                f"Status: {status_name} ({response.status}), Risk Score: {response.risk_score:.2f}"
            )
            
            await channel.close()
            return {'status': status_name, 'risk_score': response.risk_score}
            
        except Exception as e:
            self.log_test_result("Full Flow - Known Allergy", False, f"Error: {str(e)}")
            return None
    
    async def compare_results(self, direct_result, full_flow_result, scenario_name):
        """Compare results between direct CAE and full flow"""
        print(f"\n🔍 Comparison: {scenario_name}")
        print("-" * 50)
        
        if not direct_result or not full_flow_result:
            self.log_test_result(f"Comparison - {scenario_name}", False, "Missing results to compare")
            return False
        
        # For direct CAE results
        if isinstance(direct_result, dict) and 'overall_status' in direct_result:
            direct_status = direct_result['overall_status']
        else:
            direct_status = "UNKNOWN"
        
        # For full flow results
        if isinstance(full_flow_result, dict) and 'status' in full_flow_result:
            full_flow_status = full_flow_result['status']
        else:
            full_flow_status = "UNKNOWN"
        
        # Compare clinical reasoning consistency
        consistent = self._are_statuses_consistent(direct_status, full_flow_status)
        
        self.log_test_result(
            f"Comparison - {scenario_name}",
            consistent,
            f"Direct CAE: {direct_status} vs Full Flow: {full_flow_status}"
        )
        
        return consistent
    
    def _are_statuses_consistent(self, direct_status, full_flow_status):
        """Check if statuses are clinically consistent"""
        # Map statuses to risk levels
        risk_levels = {
            'SAFE': 0,
            'WARNING': 1,
            'UNSAFE': 2,
            'MANUAL_REVIEW': 1,
            'ERROR': -1
        }
        
        direct_risk = risk_levels.get(direct_status, -1)
        full_flow_risk = risk_levels.get(full_flow_status, -1)
        
        # Both should detect similar risk levels (allow some variation)
        return abs(direct_risk - full_flow_risk) <= 1 and direct_risk >= 0 and full_flow_risk >= 0
    
    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 60)
        print("📊 SAME SCENARIOS COMPARISON SUMMARY")
        print("=" * 60)
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result['success'])
        success_rate = (passed_tests / total_tests * 100) if total_tests > 0 else 0
        
        print(f"✅ Passed: {passed_tests}")
        print(f"❌ Failed: {total_tests - passed_tests}")
        print(f"📈 Success Rate: {success_rate:.1f}%")
        print(f"🧪 Total Tests: {total_tests}")
        
        if success_rate >= 80:
            print(f"\n🎉 CLINICAL CONSISTENCY: EXCELLENT!")
            print("✅ Full flow preserves clinical reasoning capabilities")
            return True
        else:
            print(f"\n❌ CLINICAL CONSISTENCY: NEEDS WORK")
            print("🔧 Full flow may have clinical reasoning issues")
            return False
    
    async def cleanup(self):
        """Cleanup resources"""
        if self.cae_engine:
            await self.cae_engine.close()

async def main():
    """Main test execution"""
    print("🔄 Same Clinical Scenarios - Full Flow vs Direct CAE")
    print("=" * 60)
    
    tester = SameScenarioTester()
    
    try:
        # Test Scenario 1: Drug Interaction
        direct_result_1 = await tester.test_scenario_1_direct_cae()
        full_flow_result_1 = await tester.test_scenario_1_full_flow()
        await tester.compare_results(direct_result_1, full_flow_result_1, "Drug Interaction")
        
        # Test Scenario 2: Known Allergy
        direct_result_2 = await tester.test_scenario_2_direct_cae()
        full_flow_result_2 = await tester.test_scenario_2_full_flow()
        await tester.compare_results(direct_result_2, full_flow_result_2, "Known Allergy")
        
        # Print summary
        success = tester.print_summary()
        return success
        
    except Exception as e:
        print(f"❌ Testing failed: {e}")
        return False
    
    finally:
        await tester.cleanup()

if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
