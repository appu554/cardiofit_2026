#!/usr/bin/env python3
"""
Test Database Connection Using Same Pattern as Other Services

This test uses the exact same connection pattern as:
- workflow-engine-service
- auth-service  
- other microservices in the project

This will help us verify if the connection issue is specific to our service
or a general network/configuration problem.
"""
import asyncio
import logging
import sys
import os
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging for Windows compatibility
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


class ConnectionTest:
    """Test database connection using patterns from other services"""
    
    def __init__(self):
        # Use the same connection string as other services
        self.database_url = "postgresql://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
        self.async_database_url = "postgresql+asyncpg://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
        
        # Supabase configuration (same as other services)
        self.supabase_url = "https://auugxeqzgrnknklgwqrh.supabase.co"
        self.supabase_key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8"
        
        self.test_results = {}
    
    async def run_all_tests(self):
        """Run all connection tests"""
        logger.info("Starting Connection Tests (Using Same Pattern as Other Services)")
        logger.info("=" * 70)
        
        # Test 1: Basic configuration
        await self.test_configuration()
        
        # Test 2: SQLAlchemy sync connection (like workflow-engine-service)
        await self.test_sqlalchemy_sync()
        
        # Test 3: SQLAlchemy async connection (like our outbox service)
        await self.test_sqlalchemy_async()
        
        # Test 4: Supabase SDK connection (like auth-service)
        await self.test_supabase_sdk()
        
        # Test 5: Network connectivity
        await self.test_network_connectivity()
        
        # Print results
        self.print_results()
    
    async def test_configuration(self):
        """Test configuration values"""
        logger.info("Testing configuration...")
        
        try:
            from app.config import settings
            
            logger.info(f"SUCCESS: Config loaded")
            logger.info(f"  Database URL: {settings.DATABASE_URL[:50]}...")
            logger.info(f"  Supabase URL: {settings.SUPABASE_URL}")
            logger.info(f"  Service Port: {settings.PORT}")
            
            self.test_results["configuration"] = True
            
        except Exception as e:
            logger.error(f"FAILED: Configuration test failed: {e}")
            self.test_results["configuration"] = False
    
    async def test_sqlalchemy_sync(self):
        """Test SQLAlchemy sync connection (workflow-engine-service pattern)"""
        logger.info("Testing SQLAlchemy sync connection...")
        
        try:
            from sqlalchemy import create_engine, text
            from sqlalchemy.orm import sessionmaker
            
            # Create engine exactly like workflow-engine-service
            engine = create_engine(
                self.database_url,
                pool_pre_ping=True,
                pool_recycle=300,
                echo=False
            )
            
            SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
            
            # Test connection
            with SessionLocal() as session:
                result = session.execute(text("SELECT 1 as test, NOW() as current_time"))
                row = result.fetchone()
                
                logger.info(f"SUCCESS: SQLAlchemy sync connection")
                logger.info(f"  Test value: {row.test}")
                logger.info(f"  Database time: {row.current_time}")
                
                self.test_results["sqlalchemy_sync"] = True
            
            engine.dispose()
            
        except Exception as e:
            logger.error(f"FAILED: SQLAlchemy sync connection failed: {e}")
            self.test_results["sqlalchemy_sync"] = False
    
    async def test_sqlalchemy_async(self):
        """Test SQLAlchemy async connection (our outbox pattern)"""
        logger.info("Testing SQLAlchemy async connection...")
        
        try:
            from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
            from sqlalchemy import text
            
            # Create async engine like our outbox service
            async_engine = create_async_engine(
                self.async_database_url,
                pool_size=5,
                max_overflow=10,
                pool_pre_ping=True,
                pool_recycle=300,
                echo=False
            )
            
            # Test async connection
            async with AsyncSession(async_engine) as session:
                result = await session.execute(text("SELECT 1 as test, NOW() as current_time"))
                row = result.fetchone()
                
                logger.info(f"SUCCESS: SQLAlchemy async connection")
                logger.info(f"  Test value: {row.test}")
                logger.info(f"  Database time: {row.current_time}")
                
                self.test_results["sqlalchemy_async"] = True
            
            await async_engine.dispose()
            
        except Exception as e:
            logger.error(f"FAILED: SQLAlchemy async connection failed: {e}")
            self.test_results["sqlalchemy_async"] = False
    
    async def test_supabase_sdk(self):
        """Test Supabase SDK connection (auth-service pattern)"""
        logger.info("Testing Supabase SDK connection...")
        
        try:
            # Try to import supabase
            try:
                from supabase import create_client
                logger.info("SUCCESS: Supabase SDK imported")
            except ImportError:
                logger.warning("WARNING: Supabase SDK not installed - skipping SDK test")
                self.test_results["supabase_sdk"] = False
                return
            
            # Create client like auth-service
            client = create_client(self.supabase_url, self.supabase_key)
            
            # Test a simple query
            response = client.table('vendor_outbox_registry').select('vendor_id').limit(1).execute()
            
            logger.info(f"SUCCESS: Supabase SDK connection")
            logger.info(f"  Response data: {response.data}")
            
            self.test_results["supabase_sdk"] = True
            
        except Exception as e:
            logger.error(f"FAILED: Supabase SDK connection failed: {e}")
            self.test_results["supabase_sdk"] = False
    
    async def test_network_connectivity(self):
        """Test basic network connectivity"""
        logger.info("Testing network connectivity...")
        
        try:
            import socket
            import urllib.request
            
            # Test 1: DNS resolution
            try:
                host = "db.auugxeqzgrnknklgwqrh.supabase.co"
                ip = socket.gethostbyname(host)
                logger.info(f"SUCCESS: DNS resolution - {host} -> {ip}")
                self.test_results["dns_resolution"] = True
            except Exception as e:
                logger.error(f"FAILED: DNS resolution failed: {e}")
                self.test_results["dns_resolution"] = False
            
            # Test 2: HTTP connectivity to Supabase
            try:
                response = urllib.request.urlopen(self.supabase_url, timeout=10)
                logger.info(f"SUCCESS: HTTP connectivity - Status: {response.getcode()}")
                self.test_results["http_connectivity"] = True
            except Exception as e:
                logger.error(f"FAILED: HTTP connectivity failed: {e}")
                self.test_results["http_connectivity"] = False
            
            # Test 3: PostgreSQL port connectivity
            try:
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.settimeout(10)
                result = sock.connect_ex(("db.auugxeqzgrnknklgwqrh.supabase.co", 5432))
                sock.close()
                
                if result == 0:
                    logger.info("SUCCESS: PostgreSQL port 5432 is reachable")
                    self.test_results["postgres_port"] = True
                else:
                    logger.error(f"FAILED: PostgreSQL port 5432 not reachable - Error: {result}")
                    self.test_results["postgres_port"] = False
            except Exception as e:
                logger.error(f"FAILED: PostgreSQL port test failed: {e}")
                self.test_results["postgres_port"] = False
            
        except Exception as e:
            logger.error(f"FAILED: Network connectivity test failed: {e}")
            self.test_results["network_connectivity"] = False
    
    def print_results(self):
        """Print comprehensive test results"""
        logger.info("")
        logger.info("=" * 70)
        logger.info("CONNECTION TEST RESULTS")
        logger.info("=" * 70)
        
        # Configuration
        logger.info("")
        logger.info("CONFIGURATION:")
        config_status = "PASS" if self.test_results.get("configuration") else "FAIL"
        logger.info(f"Configuration Load: {config_status}")
        
        # Database connections
        logger.info("")
        logger.info("DATABASE CONNECTIONS:")
        sync_status = "PASS" if self.test_results.get("sqlalchemy_sync") else "FAIL"
        async_status = "PASS" if self.test_results.get("sqlalchemy_async") else "FAIL"
        sdk_status = "PASS" if self.test_results.get("supabase_sdk") else "FAIL"
        
        logger.info(f"SQLAlchemy Sync (like workflow-engine): {sync_status}")
        logger.info(f"SQLAlchemy Async (like outbox): {async_status}")
        logger.info(f"Supabase SDK (like auth-service): {sdk_status}")
        
        # Network connectivity
        logger.info("")
        logger.info("NETWORK CONNECTIVITY:")
        dns_status = "PASS" if self.test_results.get("dns_resolution") else "FAIL"
        http_status = "PASS" if self.test_results.get("http_connectivity") else "FAIL"
        postgres_status = "PASS" if self.test_results.get("postgres_port") else "FAIL"
        
        logger.info(f"DNS Resolution: {dns_status}")
        logger.info(f"HTTP Connectivity: {http_status}")
        logger.info(f"PostgreSQL Port 5432: {postgres_status}")
        
        # Summary
        logger.info("")
        logger.info("=" * 70)
        logger.info("SUMMARY:")
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results.values() if result)
        
        logger.info(f"Tests Passed: {passed_tests}/{total_tests}")
        logger.info(f"Success Rate: {(passed_tests/total_tests)*100:.1f}%")
        
        # Diagnosis
        if self.test_results.get("dns_resolution") and self.test_results.get("postgres_port"):
            if self.test_results.get("sqlalchemy_sync") or self.test_results.get("sqlalchemy_async"):
                logger.info("DIAGNOSIS: Database connection is working!")
                logger.info("The outbox pattern should work properly.")
            else:
                logger.info("DIAGNOSIS: Network is fine, but database authentication may be failing.")
                logger.info("Check database credentials and permissions.")
        elif not self.test_results.get("dns_resolution"):
            logger.info("DIAGNOSIS: DNS resolution is failing.")
            logger.info("This is likely a network/firewall issue.")
        elif not self.test_results.get("postgres_port"):
            logger.info("DIAGNOSIS: Cannot reach PostgreSQL port 5432.")
            logger.info("This is likely a firewall or network routing issue.")
        else:
            logger.info("DIAGNOSIS: Mixed results - check individual test failures above.")
        
        logger.info("=" * 70)


async def main():
    """Main test runner"""
    test = ConnectionTest()
    await test.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())
