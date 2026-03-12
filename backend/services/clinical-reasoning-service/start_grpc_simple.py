"""
Simple CAE Engine gRPC Server Startup

Bypasses the complex config system and starts the gRPC server directly
with the CAE Engine Neo4j integration.
"""

import asyncio
import grpc
import logging
import sys
from pathlib import Path
from concurrent import futures

# Add app directory to path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

# Load environment variables
try:
    from dotenv import load_dotenv
    load_dotenv()
    print("✅ Loaded environment variables from .env file")
except ImportError:
    print("⚠️  python-dotenv not installed")

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

# Import gRPC components
try:
    from app.grpc_generated import clinical_reasoning_pb2_grpc
    from app.cae_engine_neo4j import CAEEngine
    GRPC_AVAILABLE = True
except ImportError as e:
    print(f"❌ gRPC components not available: {e}")
    GRPC_AVAILABLE = False

class SimpleCAEServicer(clinical_reasoning_pb2_grpc.ClinicalReasoningServiceServicer):
    """Simple CAE Engine gRPC servicer"""
    
    def __init__(self):
        self.cae_engine = CAEEngine()
        self.initialized = False
    
    async def initialize(self):
        """Initialize the CAE Engine"""
        try:
            self.initialized = await self.cae_engine.initialize()
            if self.initialized:
                logger.info("✅ CAE Engine initialized successfully")
            else:
                logger.error("❌ CAE Engine initialization failed")
            return self.initialized
        except Exception as e:
            logger.error(f"❌ CAE Engine initialization error: {e}")
            return False
    
    async def ValidateSafety(self, request, context):
        """Validate clinical safety using CAE Engine"""
        if not self.initialized:
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details("CAE Engine not initialized")
            return clinical_reasoning_pb2.SafetyValidationResponse()
        
        try:
            # Convert gRPC request to CAE Engine format
            clinical_context = {
                'patient': {
                    'id': request.patient_id,
                    'age': getattr(request, 'patient_age', 0),
                    'weight': getattr(request, 'patient_weight', 0),
                    'gender': getattr(request, 'patient_gender', 'unknown')
                },
                'medications': [
                    {
                        'name': med.name,
                        'dose': med.dose,
                        'frequency': getattr(med, 'frequency', 'unknown')
                    }
                    for med in request.medications
                ],
                'conditions': [
                    {'name': cond.name}
                    for cond in request.conditions
                ],
                'allergies': [
                    {
                        'substance': allergy.substance,
                        'reaction': getattr(allergy, 'reaction', 'unknown'),
                        'severity': getattr(allergy, 'severity', 'unknown')
                    }
                    for allergy in request.allergies
                ]
            }
            
            # Get safety validation from CAE Engine
            result = await self.cae_engine.validate_safety(clinical_context)
            
            # Convert result to gRPC response
            response = clinical_reasoning_pb2.SafetyValidationResponse()
            response.overall_status = result.get('overall_status', 'UNKNOWN')
            response.total_findings = result.get('total_findings', 0)
            response.execution_time_ms = result.get('performance', {}).get('total_execution_time_ms', 0)
            
            # Add findings
            for finding in result.get('findings', []):
                grpc_finding = response.findings.add()
                grpc_finding.finding_type = finding.get('finding_type', 'UNKNOWN')
                grpc_finding.severity = finding.get('severity', 'MEDIUM')
                grpc_finding.message = finding.get('message', 'No message')
                grpc_finding.recommendation = finding.get('recommendation', 'No recommendation')
            
            logger.info(f"✅ Safety validation completed for patient {request.patient_id}: {response.overall_status}")
            return response
            
        except Exception as e:
            logger.error(f"❌ Safety validation error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Internal error: {str(e)}")
            return clinical_reasoning_pb2.SafetyValidationResponse()
    
    async def close(self):
        """Close the CAE Engine"""
        if self.cae_engine:
            await self.cae_engine.close()

async def serve():
    """Start the gRPC server"""
    if not GRPC_AVAILABLE:
        print("❌ gRPC components not available")
        return False
    
    print("🚀 Starting Simple CAE Engine gRPC Server")
    print("=" * 50)
    
    # Create servicer
    servicer = SimpleCAEServicer()
    
    # Initialize CAE Engine
    print("🔧 Initializing CAE Engine with Neo4j...")
    initialized = await servicer.initialize()
    
    if not initialized:
        print("❌ Failed to initialize CAE Engine")
        return False
    
    # Create gRPC server
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    clinical_reasoning_pb2_grpc.add_ClinicalReasoningServiceServicer_to_server(servicer, server)
    
    # Configure server address
    listen_addr = '[::]:50051'
    server.add_insecure_port(listen_addr)
    
    # Start server
    await server.start()
    print("🌐 gRPC server listening on port 50051")
    print("📡 Ready to receive clinical reasoning requests")
    print("=" * 50)
    print("Press Ctrl+C to stop the server")
    
    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        print("\n👋 Shutting down gRPC server...")
        await servicer.close()
        await server.stop(grace=5)
        print("✅ Server stopped gracefully")
    
    return True

async def main():
    """Main function"""
    try:
        success = await serve()
        return 0 if success else 1
    except Exception as e:
        print(f"❌ Server failed: {e}")
        logger.error(f"Server error: {e}", exc_info=True)
        return 1

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
