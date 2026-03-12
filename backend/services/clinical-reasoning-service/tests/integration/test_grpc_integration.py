import asyncio
import grpc
import logging
import os
import subprocess
import sys
import time
import unittest

# Add project root to the Python path to allow for absolute imports
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
sys.path.insert(0, PROJECT_ROOT)

from app.proto import clinical_reasoning_pb2
from app.proto import clinical_reasoning_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class TestGrpcIntegration(unittest.IsolatedAsyncioTestCase):
    """
    Integration test for the Clinical Reasoning Service gRPC server.
    It starts the server, sends a request, and validates the response.
    """
    server_process = None
    SERVER_ADDRESS = 'localhost:8027'

    @classmethod
    def setUpClass(cls):
        """Starts the gRPC server in a separate process."""
        command = [sys.executable, '-m', 'app.grpc_server']
        logger.info(f"Starting server with command: {' '.join(command)}")
        try:
            cls.server_process = subprocess.Popen(
                command,
                cwd=PROJECT_ROOT,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            # Give the server a moment to start
            time.sleep(5) 
            logger.info(f"Server started with PID: {cls.server_process.pid}")
        except FileNotFoundError:
            logger.error(f"Could not find python executable at {sys.executable}")
            raise
        except Exception as e:
            logger.error(f"Failed to start server: {e}")
            raise

    @classmethod
    def tearDownClass(cls):
        """Stops the gRPC server."""
        if cls.server_process:
            logger.info(f"Stopping server with PID: {cls.server_process.pid}")
            cls.server_process.terminate()
            try:
                stdout, stderr = cls.server_process.communicate(timeout=5)
                logger.info("Server stdout:")
                logger.info(stdout)
                logger.error("Server stderr:")
                logger.error(stderr)
            except subprocess.TimeoutExpired:
                cls.server_process.kill()
                logger.warning("Server did not terminate gracefully, had to kill it.")
            logger.info("Server stopped.")

    async def test_generate_clinical_assertions(self):
        """Tests the GenerateClinicalAssertions RPC endpoint."""
        self.assertIsNotNone(self.server_process, "Server process should be running")
        self.assertIsNone(self.server_process.poll(), f"Server process terminated unexpectedly with code {self.server_process.returncode}")

        try:
            async with grpc.aio.insecure_channel(self.SERVER_ADDRESS) as channel:
                stub = clinical_reasoning_pb2_grpc.ClinicalReasoningServiceStub(channel)
                
                # Use the elderly cardiovascular patient data from comprehensive tests
                from google.protobuf.struct_pb2 import Struct
                
                patient_context = Struct()
                patient_context.update({
                    "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
                    "demographics": {
                        "age": 67,
                        "gender": "male",
                        "weight_kg": 78.5
                    },
                    "allergy_ids": [],
                    "metadata": {},
                    "context_version": "",
                    "assembly_time": "0001-01-01T00:00:00Z"
                })
                
                request = clinical_reasoning_pb2.ClinicalAssertionRequest(
                    patient_id="905a60cb-8241-418f-b29b-5b020e851392",
                    correlation_id="test_correlation_123",
                    medication_ids=["warfarin", "aspirin", "lisinopril", "metoprolol", "metformin", "atorvastatin"],
                    condition_ids=["atrial_fibrillation", "hypertension", "diabetes", "coronary_artery_disease"],
                    patient_context=patient_context,
                    priority=clinical_reasoning_pb2.PRIORITY_STANDARD,
                    reasoner_types=["interaction", "contraindication", "duplicate_therapy", "dosing"]
                )

                logger.info("Sending request to GenerateAssertions...")
                response = await stub.GenerateAssertions(request)

                logger.info("Received response from server.")
                self.assertIsNotNone(response)
                self.assertIsInstance(response, clinical_reasoning_pb2.ClinicalResponse)
                
                # Check that we received some assertions
                self.assertGreater(len(response.assertions), 0, "Expected at least one clinical assertion")

                # Validate the content of the first assertion
                first_assertion = response.assertions[0]
                self.assertEqual(first_assertion.patient_id, 'patient-test-001')
                self.assertIn(first_assertion.severity, clinical_reasoning_pb2.AssertionSeverity.values(), "Assertion severity is not a valid enum value")
                logger.info(f"Successfully validated {len(response.assertions)} assertions.")

        except grpc.aio.AioRpcError as e:
            self.fail(f"gRPC call failed: {e.code()} - {e.details()}")

if __name__ == '__main__':
    unittest.main()
