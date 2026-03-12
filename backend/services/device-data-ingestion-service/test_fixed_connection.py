#!/usr/bin/env python3
"""
Test Fixed Database Connection

Test the corrected database connection string with URL-encoded password.
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


async def test_fixed_connection():
    """Test the fixed database connection"""
    logger.info("Testing Fixed Database Connection")
    logger.info("=" * 50)
    
    # Fixed connection strings with URL-encoded password
    database_url = "postgresql://postgres:Cardiofit%40123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
    async_database_url = "postgresql+asyncpg://postgres:Cardiofit%40123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
    
    # Test 1: Sync connection
    logger.info("Testing sync connection...")
    try:
        from sqlalchemy import create_engine, text
        
        engine = create_engine(database_url, echo=False)
        with engine.connect() as conn:
            result = conn.execute(text("SELECT version(), NOW() as current_time"))
            row = result.fetchone()
            
            logger.info("SUCCESS: Sync connection works!")
            logger.info(f"  Database version: {row[0][:50]}...")
            logger.info(f"  Current time: {row[1]}")
        
        engine.dispose()
        
    except Exception as e:
        logger.error(f"FAILED: Sync connection failed: {e}")
        return False
    
    # Test 2: Async connection
    logger.info("Testing async connection...")
    try:
        from sqlalchemy.ext.asyncio import create_async_engine
        from sqlalchemy import text
        
        async_engine = create_async_engine(async_database_url, echo=False)
        async with async_engine.connect() as conn:
            result = await conn.execute(text("SELECT version(), NOW() as current_time"))
            row = result.fetchone()
            
            logger.info("SUCCESS: Async connection works!")
            logger.info(f"  Database version: {row[0][:50]}...")
            logger.info(f"  Current time: {row[1]}")
        
        await async_engine.dispose()
        
    except Exception as e:
        logger.error(f"FAILED: Async connection failed: {e}")
        return False
    
    # Test 3: Check existing tables
    logger.info("Checking existing tables...")
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
            
            logger.info(f"SUCCESS: Found {len(tables)} tables")
            if outbox_tables:
                logger.info(f"  Outbox tables: {outbox_tables}")
            else:
                logger.info("  No outbox tables found - need to run migration")
            
            logger.info(f"  Sample tables: {tables[:5]}...")
        
        engine.dispose()
        
    except Exception as e:
        logger.error(f"FAILED: Table check failed: {e}")
        return False
    
    # Test 4: Test our config
    logger.info("Testing service configuration...")
    try:
        from app.config import settings
        
        logger.info("SUCCESS: Configuration loaded")
        logger.info(f"  Database URL: {settings.DATABASE_URL[:50]}...")
        logger.info(f"  Supabase URL: {settings.SUPABASE_URL}")
        logger.info(f"  Service Port: {settings.PORT}")
        
    except Exception as e:
        logger.error(f"FAILED: Configuration test failed: {e}")
        return False
    
    logger.info("")
    logger.info("=" * 50)
    logger.info("ALL TESTS PASSED!")
    logger.info("The outbox pattern is ready to work!")
    logger.info("")
    logger.info("NEXT STEPS:")
    logger.info("1. Run migration: python run_migration.py")
    logger.info("2. Start service: python run_service.py")
    logger.info("3. Test endpoints: curl http://localhost:8015/api/v1/vendors/supported")
    logger.info("=" * 50)
    
    return True


if __name__ == "__main__":
    asyncio.run(test_fixed_connection())
