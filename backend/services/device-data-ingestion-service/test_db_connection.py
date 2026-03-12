#!/usr/bin/env python3
"""
Test Database Connection

Simple test to verify database connectivity and configuration.
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


async def test_database_connection():
    """Test database connection with current configuration"""
    logger.info("🔧 Testing Database Connection Configuration")
    logger.info("=" * 60)
    
    try:
        # Test configuration loading
        logger.info("📋 Step 1: Testing configuration...")
        from app.config import settings
        
        logger.info(f"✅ Configuration loaded successfully")
        logger.info(f"   Database URL: {settings.DATABASE_URL[:50]}...")
        logger.info(f"   Async URL: {settings.ASYNC_DATABASE_URL[:50]}...")
        logger.info(f"   Service Port: {settings.PORT}")
        
        # Test sync connection
        logger.info("🔄 Step 2: Testing sync database connection...")
        try:
            from sqlalchemy import create_engine, text
            
            engine = create_engine(
                settings.DATABASE_URL,
                pool_pre_ping=True,
                pool_recycle=300,
                echo=False,
                connect_args={"connect_timeout": 10}
            )
            
            with engine.connect() as conn:
                result = conn.execute(text("SELECT 1 as test, NOW() as current_time"))
                row = result.fetchone()
                
                logger.info(f"✅ Sync connection successful")
                logger.info(f"   Test value: {row.test}")
                logger.info(f"   Database time: {row.current_time}")
            
            engine.dispose()
            
        except Exception as e:
            logger.error(f"❌ Sync connection failed: {e}")
            logger.info("🔧 This might be a network connectivity issue")
            return False
        
        # Test async connection
        logger.info("🔄 Step 3: Testing async database connection...")
        try:
            from sqlalchemy.ext.asyncio import create_async_engine
            from sqlalchemy import text
            
            async_engine = create_async_engine(
                settings.ASYNC_DATABASE_URL,
                pool_size=5,
                max_overflow=10,
                pool_pre_ping=True,
                pool_recycle=300,
                echo=False,
                connect_args={
                    "command_timeout": 10,
                    "server_settings": {
                        "application_name": "device-data-ingestion-test",
                    },
                }
            )
            
            async with async_engine.connect() as conn:
                result = await conn.execute(text("SELECT 1 as test, NOW() as current_time"))
                row = result.fetchone()
                
                logger.info(f"✅ Async connection successful")
                logger.info(f"   Test value: {row.test}")
                logger.info(f"   Database time: {row.current_time}")
            
            await async_engine.dispose()
            
        except Exception as e:
            logger.error(f"❌ Async connection failed: {e}")
            logger.info("🔧 This might be a network connectivity issue")
            return False
        
        # Test outbox tables
        logger.info("📊 Step 4: Testing outbox tables...")
        try:
            from sqlalchemy import create_engine, text
            
            engine = create_engine(settings.DATABASE_URL, echo=False)
            with engine.connect() as conn:
                result = conn.execute(text("""
                    SELECT table_name 
                    FROM information_schema.tables 
                    WHERE table_schema = 'public' 
                    AND table_name LIKE '%outbox%'
                    ORDER BY table_name
                """))
                
                outbox_tables = [row[0] for row in result.fetchall()]
                
                if outbox_tables:
                    logger.info(f"✅ Found outbox tables: {outbox_tables}")
                else:
                    logger.warning("⚠️ No outbox tables found - run migration first")
            
            engine.dispose()
            
        except Exception as e:
            logger.error(f"❌ Outbox table check failed: {e}")
        
        logger.info("")
        logger.info("=" * 60)
        logger.info("🎉 DATABASE CONNECTION TEST SUCCESSFUL!")
        logger.info("=" * 60)
        logger.info("")
        logger.info("✅ Configuration is correct")
        logger.info("✅ Database connectivity works")
        logger.info("✅ Ready to start the service")
        logger.info("")
        logger.info("🚀 NEXT STEPS:")
        logger.info("1. Start service: python run_service.py")
        logger.info("2. Test endpoints in Postman")
        logger.info("3. Verify background publisher processing")
        logger.info("=" * 60)
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Database connection test failed: {e}", exc_info=True)
        logger.info("")
        logger.info("🔧 TROUBLESHOOTING:")
        logger.info("1. Check internet connection")
        logger.info("2. Verify Supabase database is running")
        logger.info("3. Check firewall settings")
        logger.info("4. Try different network (mobile hotspot)")
        logger.info("5. Check DNS settings")
        return False


if __name__ == "__main__":
    asyncio.run(test_database_connection())
