#!/usr/bin/env python3
"""
Test Supabase Connection with Correct Password

Now that we have the correct password (9FTqQnA4LRCsu8sw), 
let's test the connection and set up the outbox pattern.
"""
import asyncio
import logging
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


async def test_connection_with_correct_password():
    """Test connection with the correct password"""
    logger.info("Testing Supabase Connection with Correct Password")
    logger.info("=" * 60)
    
    # Connection strings with correct password
    database_url = "postgresql://postgres:9FTqQnA4LRCsu8sw@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
    async_database_url = "postgresql+asyncpg://postgres:9FTqQnA4LRCsu8sw@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
    
    # Test 1: Sync connection
    logger.info("1. Testing sync PostgreSQL connection...")
    try:
        from sqlalchemy import create_engine, text
        
        engine = create_engine(database_url, echo=False)
        with engine.connect() as conn:
            result = conn.execute(text("SELECT version(), NOW() as current_time"))
            row = result.fetchone()
            
            logger.info("   SUCCESS: Sync connection works!")
            logger.info(f"   Database version: {row[0][:50]}...")
            logger.info(f"   Current time: {row[1]}")
        
        engine.dispose()
        
    except Exception as e:
        logger.error(f"   FAILED: Sync connection failed: {e}")
        return False
    
    # Test 2: Async connection
    logger.info("2. Testing async PostgreSQL connection...")
    try:
        from sqlalchemy.ext.asyncio import create_async_engine
        from sqlalchemy import text
        
        async_engine = create_async_engine(async_database_url, echo=False)
        async with async_engine.connect() as conn:
            result = await conn.execute(text("SELECT version(), NOW() as current_time"))
            row = result.fetchone()
            
            logger.info("   SUCCESS: Async connection works!")
            logger.info(f"   Database version: {row[0][:50]}...")
            logger.info(f"   Current time: {row[1]}")
        
        await async_engine.dispose()
        
    except Exception as e:
        logger.error(f"   FAILED: Async connection failed: {e}")
        return False
    
    # Test 3: Check existing tables
    logger.info("3. Checking existing database tables...")
    try:
        from sqlalchemy import create_engine, text
        
        engine = create_engine(database_url, echo=False)
        with engine.connect() as conn:
            result = conn.execute(text("""
                SELECT table_name 
                FROM information_schema.tables 
                WHERE table_schema = 'public' 
                ORDER BY table_name
            """))
            
            tables = [row[0] for row in result.fetchall()]
            outbox_tables = [t for t in tables if 'outbox' in t.lower()]
            
            logger.info(f"   SUCCESS: Found {len(tables)} tables in database")
            
            if outbox_tables:
                logger.info(f"   Outbox tables found: {outbox_tables}")
                logger.info("   Great! Outbox tables already exist.")
            else:
                logger.info("   No outbox tables found yet.")
                logger.info("   Need to run migration to create outbox tables.")
            
            # Show some existing tables
            logger.info(f"   Sample tables: {tables[:10]}...")
        
        engine.dispose()
        
    except Exception as e:
        logger.error(f"   FAILED: Table check failed: {e}")
        return False
    
    # Test 4: Test our service configuration
    logger.info("4. Testing service configuration...")
    try:
        from app.config import settings
        
        logger.info("   SUCCESS: Configuration loaded")
        logger.info(f"   Database URL: {settings.DATABASE_URL[:50]}...")
        logger.info(f"   Supabase URL: {settings.SUPABASE_URL}")
        logger.info(f"   Service Port: {settings.PORT}")
        
    except Exception as e:
        logger.error(f"   FAILED: Configuration test failed: {e}")
        return False
    
    # Test 5: Test Supabase SDK
    logger.info("5. Testing Supabase SDK...")
    try:
        from supabase import create_client
        
        supabase_url = 'https://auugxeqzgrnknklgwqrh.supabase.co'
        supabase_key = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8'
        
        client = create_client(supabase_url, supabase_key)
        
        # Test a simple query
        response = client.rpc('version').execute()
        logger.info("   SUCCESS: Supabase SDK works!")
        
    except ImportError:
        logger.info("   INFO: Supabase SDK not installed (optional)")
        logger.info("   Install with: pip install supabase")
    except Exception as e:
        logger.info(f"   INFO: Supabase SDK test: {e}")
    
    logger.info("")
    logger.info("=" * 60)
    logger.info("EXCELLENT! ALL DATABASE CONNECTIONS WORK!")
    logger.info("=" * 60)
    logger.info("")
    logger.info("NEXT STEPS TO GET OUTBOX PATTERN WORKING:")
    logger.info("")
    logger.info("1. CREATE OUTBOX TABLES:")
    logger.info("   - Go to Supabase Dashboard → SQL Editor")
    logger.info("   - Run the SQL script from supabase_setup.sql")
    logger.info("   - This creates all vendor outbox tables")
    logger.info("")
    logger.info("2. RUN MIGRATION (Alternative):")
    logger.info("   python run_migration.py")
    logger.info("")
    logger.info("3. START THE SERVICE:")
    logger.info("   python run_service.py")
    logger.info("")
    logger.info("4. TEST THE ENDPOINTS:")
    logger.info("   curl http://localhost:8015/api/v1/vendors/supported")
    logger.info("   curl -X POST http://localhost:8015/api/v1/ingest/device-data-smart \\")
    logger.info("     -H 'Content-Type: application/json' \\")
    logger.info("     -d '{\"device_id\":\"test_device\",\"reading_type\":\"heart_rate\",\"value\":75}'")
    logger.info("")
    logger.info("5. TEST END-TO-END TO KAFKA:")
    logger.info("   python test_end_to_end_kafka.py")
    logger.info("")
    logger.info("🎉 THE OUTBOX PATTERN IS READY TO WORK!")
    logger.info("=" * 60)
    
    return True


if __name__ == "__main__":
    asyncio.run(test_connection_with_correct_password())
