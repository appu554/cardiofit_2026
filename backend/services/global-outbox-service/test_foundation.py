#!/usr/bin/env python3
"""
Foundation Test Script for Global Outbox Service

Tests the basic components of the Global Outbox Service to ensure
the foundation is working correctly before proceeding to Phase 2.
"""

import asyncio
import logging
import sys
from pathlib import Path

# Add the backend directory to Python path
backend_dir = Path(__file__).parent.parent.parent
if str(backend_dir) not in sys.path:
    sys.path.insert(0, str(backend_dir))

from app.core.config import settings
from app.core.database import db_manager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class FoundationTester:
    """Test suite for Global Outbox Service foundation"""
    
    def __init__(self):
        self.tests_passed = 0
        self.tests_failed = 0
        self.test_results = []
    
    def log_test_result(self, test_name: str, passed: bool, message: str = ""):
        """Log test result"""
        status = "✅ PASS" if passed else "❌ FAIL"
        full_message = f"{status} {test_name}"
        if message:
            full_message += f" - {message}"
        
        logger.info(full_message)
        
        if passed:
            self.tests_passed += 1
        else:
            self.tests_failed += 1
        
        self.test_results.append({
            "test": test_name,
            "passed": passed,
            "message": message
        })
    
    def test_configuration(self):
        """Test configuration loading"""
        try:
            # Test basic configuration
            assert settings.PROJECT_NAME == "Global Outbox Service"
            assert settings.PORT == 8040
            assert settings.GRPC_PORT == 50051
            assert settings.DATABASE_URL is not None
            
            self.log_test_result("Configuration Loading", True, "All settings loaded correctly")
            
        except Exception as e:
            self.log_test_result("Configuration Loading", False, str(e))
    
    def test_imports(self):
        """Test that all required modules can be imported"""
        try:
            # Test core imports
            from app.core.config import settings
            from app.core.database import db_manager
            
            # Test that FastAPI can be imported
            import fastapi
            import uvicorn
            
            # Test that gRPC can be imported
            import grpc
            
            # Test that database driver can be imported
            import asyncpg
            
            self.log_test_result("Module Imports", True, "All required modules available")
            
        except ImportError as e:
            self.log_test_result("Module Imports", False, f"Missing module: {e}")
        except Exception as e:
            self.log_test_result("Module Imports", False, str(e))
    
    def test_protocol_buffers(self):
        """Test Protocol Buffer compilation"""
        try:
            # Try to import the generated proto files
            from app.proto import outbox_pb2, outbox_pb2_grpc
            
            # Test that we can create a request object
            request = outbox_pb2.PublishEventRequest(
                idempotency_key="test-key",
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b"test-payload"
            )
            
            assert request.idempotency_key == "test-key"
            assert request.origin_service == "test-service"
            
            self.log_test_result("Protocol Buffers", True, "Proto files compiled and working")
            
        except ImportError:
            self.log_test_result("Protocol Buffers", False, "Proto files not found - run 'python compile_proto.py'")
        except Exception as e:
            self.log_test_result("Protocol Buffers", False, str(e))
    
    async def test_database_connection(self):
        """Test database connectivity"""
        try:
            # Test connection
            connected = await db_manager.connect()
            if not connected:
                self.log_test_result("Database Connection", False, "Failed to connect")
                return
            
            # Test health check
            health = await db_manager.health_check()
            if health.get("status") != "healthy":
                self.log_test_result("Database Connection", False, f"Health check failed: {health}")
                return
            
            # Test basic query
            result = await db_manager.execute_query("SELECT 1 as test")
            if result != 1:
                self.log_test_result("Database Connection", False, "Basic query failed")
                return
            
            # Clean up
            await db_manager.disconnect()
            
            self.log_test_result("Database Connection", True, "Connection and queries working")
            
        except Exception as e:
            self.log_test_result("Database Connection", False, str(e))
    
    async def test_database_schema(self):
        """Test database schema creation"""
        try:
            # Connect to database
            connected = await db_manager.connect()
            if not connected:
                self.log_test_result("Database Schema", False, "Cannot connect to database")
                return
            
            # Run migration
            migration_success = await db_manager.execute_migration()
            if not migration_success:
                self.log_test_result("Database Schema", False, "Migration failed")
                return
            
            # Check if main table exists
            table_exists = await db_manager.execute_query("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_name = 'global_event_outbox'
                )
            """)
            
            if not table_exists:
                self.log_test_result("Database Schema", False, "Main outbox table not created")
                return
            
            # Check if partitions exist
            partition_count = await db_manager.execute_query("""
                SELECT COUNT(*) FROM information_schema.tables 
                WHERE table_name LIKE 'outbox_%' AND table_type = 'BASE TABLE'
            """)
            
            if partition_count < 5:  # Should have at least 5 partitions
                self.log_test_result("Database Schema", False, f"Only {partition_count} partitions created")
                return
            
            # Check if dead letter table exists
            dlq_exists = await db_manager.execute_query("""
                SELECT EXISTS (
                    SELECT FROM information_schema.tables 
                    WHERE table_name = 'global_dead_letter_queue'
                )
            """)
            
            if not dlq_exists:
                self.log_test_result("Database Schema", False, "Dead letter queue table not created")
                return
            
            # Clean up
            await db_manager.disconnect()
            
            self.log_test_result("Database Schema", True, f"Schema created with {partition_count} partitions")
            
        except Exception as e:
            self.log_test_result("Database Schema", False, str(e))
    
    def test_file_structure(self):
        """Test that all required files exist"""
        try:
            base_path = Path(__file__).parent
            
            required_files = [
                "app/__init__.py",
                "app/main.py",
                "app/grpc_server.py",
                "app/core/__init__.py",
                "app/core/config.py",
                "app/core/database.py",
                "app/services/__init__.py",
                "app/models/__init__.py",
                "app/proto/__init__.py",
                "app/proto/outbox.proto",
                "migrations/001_create_outbox_tables.sql",
                "requirements.txt",
                "run_service.py",
                "compile_proto.py",
                ".env.example"
            ]
            
            missing_files = []
            for file_path in required_files:
                if not (base_path / file_path).exists():
                    missing_files.append(file_path)
            
            if missing_files:
                self.log_test_result("File Structure", False, f"Missing files: {', '.join(missing_files)}")
            else:
                self.log_test_result("File Structure", True, "All required files present")
                
        except Exception as e:
            self.log_test_result("File Structure", False, str(e))
    
    async def run_all_tests(self):
        """Run all foundation tests"""
        logger.info("🧪 Running Global Outbox Service Foundation Tests")
        logger.info("=" * 60)
        
        # Run synchronous tests
        self.test_configuration()
        self.test_imports()
        self.test_protocol_buffers()
        self.test_file_structure()
        
        # Run asynchronous tests
        await self.test_database_connection()
        await self.test_database_schema()
        
        # Print summary
        logger.info("=" * 60)
        logger.info(f"📊 Test Results: {self.tests_passed} passed, {self.tests_failed} failed")
        
        if self.tests_failed == 0:
            logger.info("🎉 All foundation tests passed! Ready for Phase 2.")
            return True
        else:
            logger.error("❌ Some tests failed. Please fix issues before proceeding.")
            
            # Show failed tests
            logger.error("\nFailed tests:")
            for result in self.test_results:
                if not result["passed"]:
                    logger.error(f"  - {result['test']}: {result['message']}")
            
            return False

async def main():
    """Main test runner"""
    tester = FoundationTester()
    success = await tester.run_all_tests()
    
    if success:
        print("\n✅ Foundation is solid! You can proceed with Phase 2: Core Outbox Logic")
        print("\nNext steps:")
        print("1. Start Phase 2 implementation")
        print("2. Implement OutboxManager service")
        print("3. Create background publisher")
        print("4. Add retry logic and error handling")
    else:
        print("\n❌ Foundation needs work before proceeding")
        print("\nRecommended actions:")
        print("1. Fix the failed tests above")
        print("2. Run 'python compile_proto.py' if Protocol Buffer test failed")
        print("3. Check database credentials if database tests failed")
        print("4. Install missing dependencies if import tests failed")
    
    return success

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
