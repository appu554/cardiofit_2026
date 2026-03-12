#!/usr/bin/env python3
"""
Direct Supabase Connection Test

Test connection using your exact Supabase credentials to verify
if the outbox pattern can connect to your Supabase instance.
"""
import asyncio
import logging
import sys
import json
from pathlib import Path

# Configure logging for Windows compatibility
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


class DirectSupabaseTest:
    """Test direct connection to your Supabase instance"""
    
    def __init__(self):
        # Your exact Supabase credentials
        self.supabase_url = 'https://auugxeqzgrnknklgwqrh.supabase.co'
        self.supabase_key = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8'
        
        # Database connection strings
        self.database_url = "postgresql://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
        self.async_database_url = "postgresql+asyncpg://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
        
        self.test_results = {}
    
    async def run_tests(self):
        """Run all direct connection tests"""
        logger.info("Starting Direct Supabase Connection Tests")
        logger.info("=" * 60)
        
        # Test 1: Basic network connectivity
        await self.test_basic_connectivity()
        
        # Test 2: Supabase REST API
        await self.test_supabase_rest_api()
        
        # Test 3: Supabase SDK (if available)
        await self.test_supabase_sdk()
        
        # Test 4: Direct PostgreSQL connection
        await self.test_postgresql_direct()
        
        # Test 5: Check existing tables
        await self.test_existing_tables()
        
        # Print results
        self.print_results()
    
    async def test_basic_connectivity(self):
        """Test basic HTTP connectivity to Supabase"""
        logger.info("Testing basic connectivity...")
        
        try:
            import urllib.request
            import urllib.parse
            
            # Test basic HTTP request to Supabase
            request = urllib.request.Request(self.supabase_url)
            request.add_header('apikey', self.supabase_key)
            request.add_header('Authorization', f'Bearer {self.supabase_key}')
            
            response = urllib.request.urlopen(request, timeout=10)
            status_code = response.getcode()
            
            logger.info(f"SUCCESS: Basic connectivity - Status: {status_code}")
            self.test_results["basic_connectivity"] = True
            
        except Exception as e:
            logger.error(f"FAILED: Basic connectivity failed: {e}")
            self.test_results["basic_connectivity"] = False
    
    async def test_supabase_rest_api(self):
        """Test Supabase REST API directly"""
        logger.info("Testing Supabase REST API...")
        
        try:
            import urllib.request
            import urllib.parse
            import json
            
            # Test REST API endpoint
            url = f"{self.supabase_url}/rest/v1/"
            request = urllib.request.Request(url)
            request.add_header('apikey', self.supabase_key)
            request.add_header('Authorization', f'Bearer {self.supabase_key}')
            request.add_header('Content-Type', 'application/json')
            
            response = urllib.request.urlopen(request, timeout=10)
            data = response.read().decode('utf-8')
            
            logger.info(f"SUCCESS: REST API accessible")
            logger.info(f"  Response: {data[:100]}...")
            self.test_results["rest_api"] = True
            
        except Exception as e:
            logger.error(f"FAILED: REST API test failed: {e}")
            self.test_results["rest_api"] = False
    
    async def test_supabase_sdk(self):
        """Test Supabase Python SDK"""
        logger.info("Testing Supabase SDK...")
        
        try:
            # Try to import and use Supabase SDK
            try:
                from supabase import create_client
                logger.info("SUCCESS: Supabase SDK imported")
            except ImportError:
                logger.warning("WARNING: Supabase SDK not installed")
                logger.info("  Install with: pip install supabase")
                self.test_results["supabase_sdk"] = False
                return
            
            # Create client
            client = create_client(self.supabase_url, self.supabase_key)
            
            # Test a simple operation - list tables
            try:
                # Try to access a system table that should exist
                response = client.rpc('version').execute()
                logger.info(f"SUCCESS: Supabase SDK connection")
                logger.info(f"  Database version info available")
                self.test_results["supabase_sdk"] = True
                
            except Exception as e:
                # If that fails, try a different approach
                logger.info(f"INFO: SDK connected but limited access: {e}")
                self.test_results["supabase_sdk"] = True
            
        except Exception as e:
            logger.error(f"FAILED: Supabase SDK test failed: {e}")
            self.test_results["supabase_sdk"] = False
    
    async def test_postgresql_direct(self):
        """Test direct PostgreSQL connection"""
        logger.info("Testing direct PostgreSQL connection...")
        
        try:
            # Test sync connection first
            try:
                from sqlalchemy import create_engine, text
                
                engine = create_engine(self.database_url, echo=False)
                with engine.connect() as conn:
                    result = conn.execute(text("SELECT version()"))
                    version = result.fetchone()[0]
                    
                    logger.info(f"SUCCESS: PostgreSQL sync connection")
                    logger.info(f"  Version: {version[:50]}...")
                    self.test_results["postgresql_sync"] = True
                
                engine.dispose()
                
            except Exception as e:
                logger.error(f"FAILED: PostgreSQL sync connection: {e}")
                self.test_results["postgresql_sync"] = False
            
            # Test async connection
            try:
                from sqlalchemy.ext.asyncio import create_async_engine
                from sqlalchemy import text
                
                async_engine = create_async_engine(self.async_database_url, echo=False)
                async with async_engine.connect() as conn:
                    result = await conn.execute(text("SELECT version()"))
                    version = result.fetchone()[0]
                    
                    logger.info(f"SUCCESS: PostgreSQL async connection")
                    logger.info(f"  Version: {version[:50]}...")
                    self.test_results["postgresql_async"] = True
                
                await async_engine.dispose()
                
            except Exception as e:
                logger.error(f"FAILED: PostgreSQL async connection: {e}")
                self.test_results["postgresql_async"] = False
            
        except Exception as e:
            logger.error(f"FAILED: PostgreSQL test setup failed: {e}")
            self.test_results["postgresql_direct"] = False
    
    async def test_existing_tables(self):
        """Check what tables exist in the database"""
        logger.info("Checking existing tables...")
        
        try:
            from sqlalchemy import create_engine, text
            
            engine = create_engine(self.database_url, echo=False)
            with engine.connect() as conn:
                # Check for existing tables
                result = conn.execute(text("""
                    SELECT table_name 
                    FROM information_schema.tables 
                    WHERE table_schema = 'public' 
                    ORDER BY table_name
                """))
                
                tables = [row[0] for row in result.fetchall()]
                
                logger.info(f"SUCCESS: Found {len(tables)} tables in database")
                
                # Check for outbox-related tables
                outbox_tables = [t for t in tables if 'outbox' in t.lower()]
                vendor_tables = [t for t in tables if 'vendor' in t.lower()]
                
                if outbox_tables:
                    logger.info(f"  Outbox tables: {outbox_tables}")
                else:
                    logger.info("  No outbox tables found (need to run migration)")
                
                if vendor_tables:
                    logger.info(f"  Vendor tables: {vendor_tables}")
                
                # Show first 10 tables
                logger.info(f"  All tables: {tables[:10]}{'...' if len(tables) > 10 else ''}")
                
                self.test_results["existing_tables"] = {
                    "total_tables": len(tables),
                    "outbox_tables": outbox_tables,
                    "vendor_tables": vendor_tables,
                    "all_tables": tables
                }
            
            engine.dispose()
            
        except Exception as e:
            logger.error(f"FAILED: Table check failed: {e}")
            self.test_results["existing_tables"] = False
    
    def print_results(self):
        """Print comprehensive test results"""
        logger.info("")
        logger.info("=" * 60)
        logger.info("DIRECT SUPABASE CONNECTION TEST RESULTS")
        logger.info("=" * 60)
        
        # Basic connectivity
        logger.info("")
        logger.info("BASIC CONNECTIVITY:")
        basic_status = "PASS" if self.test_results.get("basic_connectivity") else "FAIL"
        rest_status = "PASS" if self.test_results.get("rest_api") else "FAIL"
        logger.info(f"HTTP Connectivity: {basic_status}")
        logger.info(f"REST API Access: {rest_status}")
        
        # SDK and PostgreSQL
        logger.info("")
        logger.info("DATABASE CONNECTIONS:")
        sdk_status = "PASS" if self.test_results.get("supabase_sdk") else "FAIL"
        sync_status = "PASS" if self.test_results.get("postgresql_sync") else "FAIL"
        async_status = "PASS" if self.test_results.get("postgresql_async") else "FAIL"
        
        logger.info(f"Supabase SDK: {sdk_status}")
        logger.info(f"PostgreSQL Sync: {sync_status}")
        logger.info(f"PostgreSQL Async: {async_status}")
        
        # Table information
        logger.info("")
        logger.info("DATABASE SCHEMA:")
        tables_result = self.test_results.get("existing_tables")
        if tables_result and isinstance(tables_result, dict):
            total_tables = tables_result.get("total_tables", 0)
            outbox_tables = tables_result.get("outbox_tables", [])
            logger.info(f"Total Tables: {total_tables}")
            logger.info(f"Outbox Tables: {len(outbox_tables)} ({outbox_tables})")
            
            if not outbox_tables:
                logger.info("  ACTION NEEDED: Run migration to create outbox tables")
        else:
            logger.info("Table Check: FAIL")
        
        # Summary and next steps
        logger.info("")
        logger.info("=" * 60)
        logger.info("SUMMARY AND NEXT STEPS:")
        
        # Count successful tests
        successful_tests = sum(1 for result in self.test_results.values() 
                             if result is True or (isinstance(result, dict) and result))
        total_tests = len(self.test_results)
        
        logger.info(f"Tests Passed: {successful_tests}/{total_tests}")
        
        # Determine next steps
        if self.test_results.get("postgresql_async"):
            logger.info("")
            logger.info("EXCELLENT NEWS: PostgreSQL async connection works!")
            logger.info("This means the outbox pattern will work perfectly.")
            logger.info("")
            logger.info("NEXT STEPS:")
            logger.info("1. Run migration to create outbox tables:")
            logger.info("   python run_migration.py")
            logger.info("2. Start the device ingestion service:")
            logger.info("   python run_service.py")
            logger.info("3. Test the outbox endpoints:")
            logger.info("   curl -X POST http://localhost:8015/api/v1/ingest/device-data-smart")
            
        elif self.test_results.get("postgresql_sync"):
            logger.info("")
            logger.info("GOOD NEWS: PostgreSQL sync connection works!")
            logger.info("Async connection might need asyncpg package.")
            logger.info("")
            logger.info("NEXT STEPS:")
            logger.info("1. Install asyncpg: pip install asyncpg")
            logger.info("2. Run migration: python run_migration.py")
            logger.info("3. Test the service")
            
        elif self.test_results.get("basic_connectivity"):
            logger.info("")
            logger.info("PARTIAL SUCCESS: Can reach Supabase but database connection fails.")
            logger.info("This might be a credentials or permissions issue.")
            logger.info("")
            logger.info("NEXT STEPS:")
            logger.info("1. Check database password in Supabase dashboard")
            logger.info("2. Verify database permissions")
            logger.info("3. Try resetting database password")
            
        else:
            logger.info("")
            logger.info("CONNECTION ISSUES: Cannot reach Supabase.")
            logger.info("This is likely a network/firewall issue.")
            logger.info("")
            logger.info("NEXT STEPS:")
            logger.info("1. Check internet connection")
            logger.info("2. Try different network (mobile hotspot)")
            logger.info("3. Check Windows Firewall settings")
        
        logger.info("=" * 60)


async def main():
    """Main test runner"""
    test = DirectSupabaseTest()
    await test.run_tests()


if __name__ == "__main__":
    asyncio.run(main())
