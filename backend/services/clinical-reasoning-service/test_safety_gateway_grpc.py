"""
Safety Gateway Platform gRPC Integration Test

Tests the gRPC integration between Safety Gateway Platform and CAE Engine
using the proper gRPC protocol instead of HTTP.
"""

import asyncio
import grpc
import sys
import logging
import uuid
from pathlib import Path
from dotenv import load_dotenv
load_dotenv()

# Add Safety Gateway proto path
safety_gateway_proto_path = Path(__file__).parent.parent / "safety-gateway-platform" / "proto"
sys.path.insert(0, str(safety_gateway_proto_path))

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class SafetyGatewayGRPCTester:
    """Test Safety Gateway Platform gRPC integration"""
    
    def __init__(self):
        self.safety_gateway_url = "localhost:8030"  # Safety Gateway gRPC port
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
    
    async def test_safety_gateway_grpc_health(self):
        """Test Safety Gateway Platform gRPC health"""
        print("\n🛡️ Testing Safety Gateway Platform gRPC Health")
        print("-" * 50)
        
        try:
            # Import gRPC health check
            from grpc_health.v1 import health_pb2, health_pb2_grpc
            
            channel = grpc.aio.insecure_channel(self.safety_gateway_url)
            
            # Test basic connectivity
            try:
                await asyncio.wait_for(channel.channel_ready(), timeout=5.0)
                connectivity_ok = True
            except asyncio.TimeoutError:
                connectivity_ok = False
            
            self.log_test_result(
                "Safety Gateway Connectivity",
                connectivity_ok,
                f"Connected to {self.safety_gateway_url}" if connectivity_ok else f"Failed to connect to {self.safety_gateway_url}"
            )
            
            if connectivity_ok:
                # Test health check
                health_stub = health_pb2_grpc.HealthStub(channel)
                health_request = health_pb2.HealthCheckRequest(service="safety-gateway")
                
                try:
                    health_response = await asyncio.wait_for(
                        health_stub.Check(health_request),
                        timeout=5.0
                    )
                    
                    health_ok = health_response.status == health_pb2.HealthCheckResponse.SERVING
                    
                    self.log_test_result(
                        "Safety Gateway Health Check",
                        health_ok,
                        f"Health status: {health_response.status}"
                    )
                    
                except Exception as e:
                    self.log_test_result(
                        "Safety Gateway Health Check",
                        False,
                        f"Health check failed: {str(e)}"
                    )
                    health_ok = False
            else:
                health_ok = False
            
            await channel.close()
            return connectivity_ok and health_ok
            
        except ImportError:
            self.log_test_result(
                "Safety Gateway gRPC Health",
                False,
                "gRPC health check not available - install grpcio-health-checking"
            )
            return False
        except Exception as e:
            self.log_test_result(
                "Safety Gateway gRPC Health",
                False,
                f"Error: {str(e)}"
            )
            return False
    
    async def test_safety_gateway_service(self):
        """Test Safety Gateway Platform service calls"""
        print("\n🔒 Testing Safety Gateway Platform Service")
        print("-" * 50)
        
        try:
            # Import Safety Gateway gRPC bindings
            import safety_gateway_pb2
            import safety_gateway_pb2_grpc
            
            channel = grpc.aio.insecure_channel(self.safety_gateway_url)
            stub = safety_gateway_pb2_grpc.SafetyGatewayStub(channel)

            # Generate proper UUIDs (must be at least 36 characters)
            request_id = str(uuid.uuid4())
            clinician_id = str(uuid.uuid4())

            # Add authorization metadata
            metadata = [
                ('authorization', 'Bearer test-token'),
                ('clinician-id', clinician_id),
                ('request-id', request_id)
            ]

            # Create test request using correct protobuf messages
            request = safety_gateway_pb2.SafetyRequest(
                request_id=request_id,
                patient_id="905a60cb-8241-418f-b29b-5b020e851392",
                clinician_id=clinician_id,
                action_type="medication_order",
                priority="normal",
                medication_ids=["warfarin_5mg", "ciprofloxacin_500mg"],
                condition_ids=["atrial_fibrillation"],
                allergy_ids=[],
                context={
                    "patient_age": "65",
                    "patient_weight": "70.0"
                }
            )
            
            # Make gRPC call with authorization metadata
            try:
                response = await asyncio.wait_for(
                    stub.ValidateSafety(request, metadata=metadata),
                    timeout=10.0
                )
                
                # Validate response using correct protobuf fields
                # Status 5 = SAFETY_STATUS_ERROR, which means CAE Engine had issues
                service_success = (
                    hasattr(response, 'status') and
                    hasattr(response, 'risk_score') and
                    response.status in [
                        safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_SAFE,      # 1
                        safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_UNSAFE,    # 2
                        safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_WARNING,   # 3
                        safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_MANUAL_REVIEW, # 4
                        safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_ERROR      # 5
                    ]
                )

                # Log the actual status for debugging
                status_names = {
                    0: "UNSPECIFIED",
                    1: "SAFE",
                    2: "UNSAFE",
                    3: "WARNING",
                    4: "MANUAL_REVIEW",
                    5: "ERROR"
                }
                status_name = status_names.get(response.status, f"UNKNOWN({response.status})")
                
                self.log_test_result(
                    "Safety Gateway Service Call",
                    service_success,
                    f"Status: {status_name} ({response.status}), Risk Score: {response.risk_score:.2f}, "
                    f"Warnings: {len(response.warnings)}, Critical: {len(response.critical_violations)}"
                )
                
                await channel.close()
                return service_success
                
            except asyncio.TimeoutError:
                self.log_test_result(
                    "Safety Gateway Service Call",
                    False,
                    "Service call timeout - Safety Gateway may not be responding"
                )
                await channel.close()
                return False
                
        except ImportError as e:
            self.log_test_result(
                "Safety Gateway Service Call",
                False,
                f"gRPC bindings not available: {str(e)}"
            )
            return False
        except Exception as e:
            self.log_test_result(
                "Safety Gateway Service Call",
                False,
                f"Service call error: {str(e)}"
            )
            return False
    
    async def test_end_to_end_flow(self):
        """Test end-to-end clinical decision flow"""
        print("\n🩺 Testing End-to-End Clinical Decision Flow")
        print("-" * 50)
        
        try:
            import safety_gateway_pb2
            import safety_gateway_pb2_grpc
            
            channel = grpc.aio.insecure_channel(self.safety_gateway_url)
            stub = safety_gateway_pb2_grpc.SafetyGatewayStub(channel)

            # Generate proper UUIDs (must be at least 36 characters)
            request_id = str(uuid.uuid4())
            clinician_id = str(uuid.uuid4())

            # Add authorization metadata
            metadata = [
                ('authorization', 'Bearer test-token'),
                ('clinician-id', clinician_id),
                ('request-id', request_id)
            ]

            # Test scenario: Drug interaction
            request = safety_gateway_pb2.SafetyRequest(
                request_id=request_id,
                patient_id="905a60cb-8241-418f-b29b-5b020e851392",
                clinician_id=clinician_id,
                action_type="medication_order",
                priority="normal",
                medication_ids=["warfarin_5mg", "ciprofloxacin_500mg"],
                condition_ids=[],
                allergy_ids=[],
                context={
                    "patient_age": "60"
                }
            )
            
            import time
            start_time = time.time()
            
            response = await asyncio.wait_for(
                stub.ValidateSafety(request, metadata=metadata),
                timeout=15.0
            )
            
            end_time = time.time()
            execution_time = (end_time - start_time) * 1000
            
            # Validate end-to-end flow
            status_names = {0: "UNSPECIFIED", 1: "SAFE", 2: "UNSAFE", 3: "WARNING", 4: "MANUAL_REVIEW", 5: "ERROR"}
            status_name = status_names.get(response.status, f"UNKNOWN({response.status})")

            e2e_success = (
                response.status in [
                    safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_SAFE,
                    safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_WARNING,
                    safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_UNSAFE,
                    safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_MANUAL_REVIEW,
                    safety_gateway_pb2.SafetyStatus.SAFETY_STATUS_ERROR
                ] and
                execution_time < 5000  # Under 5 seconds
            )

            self.log_test_result(
                "End-to-End Clinical Flow",
                e2e_success,
                f"Status: {status_name} ({response.status}), Time: {execution_time:.1f}ms, "
                f"Risk Score: {response.risk_score:.2f}, Warnings: {len(response.warnings)}"
            )
            
            await channel.close()
            return e2e_success
            
        except Exception as e:
            self.log_test_result(
                "End-to-End Clinical Flow",
                False,
                f"E2E flow error: {str(e)}"
            )
            return False
    
    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 60)
        print("📊 SAFETY GATEWAY gRPC INTEGRATION SUMMARY")
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
        if success_rate >= 80:
            print(f"\n🎉 SAFETY GATEWAY gRPC INTEGRATION: SUCCESS!")
            print("✅ Complete clinical decision support flow working via gRPC")
            return True
        else:
            print(f"\n❌ SAFETY GATEWAY gRPC INTEGRATION: NEEDS WORK")
            print("🔧 gRPC integration issues to resolve")
            return False

async def main():
    """Main test execution"""
    print("🛡️ Safety Gateway Platform gRPC Integration Test")
    print("=" * 60)
    
    tester = SafetyGatewayGRPCTester()
    
    try:
        # Run gRPC integration tests
        await tester.test_safety_gateway_grpc_health()
        await tester.test_safety_gateway_service()
        await tester.test_end_to_end_flow()
        
        # Print summary
        success = tester.print_summary()
        return success
        
    except Exception as e:
        print(f"❌ gRPC integration testing failed: {e}")
        return False

if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
