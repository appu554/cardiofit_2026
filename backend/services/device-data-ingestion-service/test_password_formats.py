#!/usr/bin/env python3
"""
Test Different Password Formats for Supabase Connection

This script helps you find the correct password format for your Supabase database.
"""
import asyncio
import logging
import sys
import urllib.parse

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


async def test_password_formats():
    """Test different password formats"""
    logger.info("Testing Different Password Formats for Supabase")
    logger.info("=" * 60)
    
    # Common password formats to try
    password_candidates = [
        "Cardiofit@123",           # Original
        "Cardiofit%40123",         # URL encoded @
        "Cardiofit123",            # Without @
        "[YOUR-PASSWORD]",         # Placeholder
    ]
    
    base_url = "postgresql://postgres:{password}@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres"
    
    logger.info("IMPORTANT: Please check your Supabase dashboard for the exact password!")
    logger.info("Go to: Supabase Dashboard → Settings → Database → Connection string")
    logger.info("")
    
    for i, password in enumerate(password_candidates, 1):
        logger.info(f"Format {i}: {password}")
        
        # Show URL-encoded version
        encoded_password = urllib.parse.quote(password, safe='')
        connection_url = base_url.format(password=encoded_password)
        
        logger.info(f"  Connection URL: {connection_url}")
        logger.info(f"  URL-encoded: {encoded_password}")
        
        # Test the connection
        try:
            from sqlalchemy import create_engine, text
            
            engine = create_engine(connection_url, echo=False)
            with engine.connect() as conn:
                result = conn.execute(text("SELECT 1 as test"))
                test_value = result.fetchone()[0]
                
                if test_value == 1:
                    logger.info(f"  ✅ SUCCESS: This password format works!")
                    logger.info(f"  Use this in your config: {encoded_password}")
                    logger.info("")
                    return encoded_password
                else:
                    logger.info(f"  ❌ FAILED: Unexpected result")
            
            engine.dispose()
            
        except Exception as e:
            logger.info(f"  ❌ FAILED: {str(e)[:100]}...")
        
        logger.info("")
    
    logger.info("=" * 60)
    logger.info("NONE OF THE COMMON FORMATS WORKED")
    logger.info("")
    logger.info("NEXT STEPS:")
    logger.info("1. Go to Supabase Dashboard → Settings → Database")
    logger.info("2. Look for 'Connection string' section")
    logger.info("3. Copy the exact password from there")
    logger.info("4. If needed, reset the database password")
    logger.info("5. Update the DATABASE_URL in config.py")
    logger.info("=" * 60)
    
    return None


async def test_supabase_sdk_fallback():
    """Test Supabase SDK as fallback"""
    logger.info("Testing Supabase SDK Fallback...")
    
    try:
        from supabase import create_client
        
        supabase_url = 'https://auugxeqzgrnknklgwqrh.supabase.co'
        supabase_key = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8'
        
        client = create_client(supabase_url, supabase_key)
        
        # Test a simple query
        response = client.table('vendor_outbox_registry').select('vendor_id').limit(1).execute()
        
        logger.info("✅ SUCCESS: Supabase SDK works!")
        logger.info("We can use SDK-based outbox pattern as fallback")
        return True
        
    except ImportError:
        logger.info("❌ Supabase SDK not installed")
        logger.info("Install with: pip install supabase")
        return False
    except Exception as e:
        logger.info(f"❌ Supabase SDK failed: {e}")
        return False


async def main():
    """Main test runner"""
    logger.info("🔍 Supabase Connection Troubleshooting")
    logger.info("")
    
    # Test password formats
    working_password = await test_password_formats()
    
    if working_password:
        logger.info("🎉 GREAT! Direct PostgreSQL connection works!")
        logger.info(f"Use this password format: {working_password}")
        logger.info("")
        logger.info("NEXT STEPS:")
        logger.info("1. Update DATABASE_URL in config.py with the working password")
        logger.info("2. Run: python run_migration.py")
        logger.info("3. Start service: python run_service.py")
    else:
        logger.info("🔄 Direct PostgreSQL failed, testing SDK fallback...")
        logger.info("")
        
        sdk_works = await test_supabase_sdk_fallback()
        
        if sdk_works:
            logger.info("")
            logger.info("✅ GOOD NEWS: Supabase SDK works!")
            logger.info("We can use the SDK-based outbox pattern instead")
            logger.info("")
            logger.info("NEXT STEPS:")
            logger.info("1. Install Supabase SDK: pip install supabase")
            logger.info("2. Run SQL setup in Supabase dashboard")
            logger.info("3. Use SDK-based outbox service")
        else:
            logger.info("")
            logger.info("❌ Both direct PostgreSQL and SDK failed")
            logger.info("Please check your Supabase project settings")


if __name__ == "__main__":
    asyncio.run(main())
