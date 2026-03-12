#!/usr/bin/env python3
"""
Simple Pipeline Test with Enhanced Error Handling
Tests the pipeline with a single source to isolate issues
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

# Setup logging first
from core.logging_config import setup_pipeline_logging, get_logger

async def test_single_source():
    """Test pipeline with a single source"""
    
    # Setup logging
    pipeline_logger = setup_pipeline_logging()
    logger = get_logger(__name__)
    
    print("🧪 TESTING SINGLE SOURCE PIPELINE")
    print("=" * 50)
    print(f"📝 Logs: {pipeline_logger.log_dir}")
    print("=" * 50)
    
    try:
        # Test database connection first
        logger.info("🔌 Testing database connection...")
        
        from core.database_factory import create_database_client
        
        database_client = await create_database_client()
        
        # Test connection
        if hasattr(database_client, 'test_connection'):
            connection_ok = await database_client.test_connection()
            if connection_ok:
                logger.info("✅ Database connection successful")
                print("✅ Database connection successful")
            else:
                logger.error("❌ Database connection failed")
                print("❌ Database connection failed")
                return False
        
        # Create index for faster lookups
        logger.info("🚀 Creating index for rxcui to optimize performance...")
        try:
            # The node label is 'cae_Drug' based on our RDF parser
            await database_client.execute_cypher("CREATE INDEX rxcui_index IF NOT EXISTS FOR (n:cae_Drug) ON (n.rxcui)")
            logger.info("✅ Index creation/verification successful.")
            print("✅ Index creation/verification successful.")
        except Exception as e:
            logger.error("❌ Failed to create index", error=str(e))
            print(f"❌ Failed to create index: {e}")
            # Stop if index creation fails, as it predicts performance issues
            return False

        # Test with RxNorm only (smallest dataset)
        logger.info("🧬 Testing RxNorm ingester...")
        
        from ingesters.rxnorm_ingester import RxNormIngester
        
        # Create ingester
        ingester = RxNormIngester(database_client)
        
        # Test ingestion
        result = await ingester.ingest(force_download=False)
        
        if result.success:
            logger.info("✅ RxNorm ingestion successful", 
                       entities=result.total_records_processed,
                       time=result.duration)
            print(f"✅ RxNorm ingestion successful!")
            print(f"   📊 Entities processed: {result.total_records_processed}")
            print(f"   ⏱️  Time: {result.duration:.2f}s")
            return True
        else:
            logger.error("❌ RxNorm ingestion failed", errors=result.errors)
            print(f"❌ RxNorm ingestion failed!")
            print(f"   🚨 Errors: {result.errors}")
            return False
    
    except Exception as e:
        logger.error("💥 Test failed with exception", error=str(e))
        pipeline_logger.log_exception(logger, "Test failed", e)
        print(f"💥 Test failed: {e}")
        return False
    
    finally:
        # Cleanup
        try:
            if 'database_client' in locals():
                await database_client.disconnect()
        except Exception:
            pass


async def test_database_only():
    """Test just the database connection and basic operations"""
    
    pipeline_logger = setup_pipeline_logging()
    logger = get_logger(__name__)
    
    print("🔌 TESTING DATABASE CONNECTION ONLY")
    print("=" * 50)
    
    try:
        from core.database_factory import create_database_client
        
        logger.info("Creating database client...")
        database_client = await create_database_client()
        
        logger.info("Testing connection...")
        if hasattr(database_client, 'test_connection'):
            connection_ok = await database_client.test_connection()
            if connection_ok:
                logger.info("✅ Database connection successful")
                print("✅ Database connection successful")
                
                # Test a simple query
                if hasattr(database_client, 'execute_cypher'):
                    logger.info("Testing simple Cypher query...")
                    try:
                        await database_client.execute_cypher("MATCH (n) RETURN count(n) as total LIMIT 1")
                        logger.info("✅ Simple query successful")
                        print("✅ Simple query successful")
                    except Exception as e:
                        logger.warning("⚠️ Simple query failed", error=str(e))
                        print(f"⚠️ Simple query failed: {e}")
                
                return True
            else:
                logger.error("❌ Database connection failed")
                print("❌ Database connection failed")
                return False
        else:
            logger.warning("⚠️ No test_connection method available")
            print("⚠️ No test_connection method available")
            return True
    
    except Exception as e:
        logger.error("💥 Database test failed", error=str(e))
        pipeline_logger.log_exception(logger, "Database test failed", e)
        print(f"💥 Database test failed: {e}")
        return False
    
    finally:
        try:
            if 'database_client' in locals():
                await database_client.disconnect()
        except Exception:
            pass


def main():
    """Main test function"""
    
    print("🏥 CLINICAL KNOWLEDGE PIPELINE - SIMPLE TEST")
    print("=" * 60)
    
    # Test 1: Database connection only
    print("\n🔌 Step 1: Testing database connection...")
    db_result = asyncio.run(test_database_only())
    
    if not db_result:
        print("❌ Database test failed - stopping here")
        return False
    
    # Test 2: Single source ingestion
    print("\n🧬 Step 2: Testing single source ingestion...")
    ingestion_result = asyncio.run(test_single_source())
    
    if ingestion_result:
        print("\n🎉 All tests passed!")
        return True
    else:
        print("\n❌ Tests failed!")
        return False


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
