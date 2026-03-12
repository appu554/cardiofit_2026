#!/usr/bin/env python3
"""
Check and Fix Global Outbox Service Partitions

This script checks existing partitions and creates missing ones for services
that need to use the Global Outbox Service.
"""

import asyncio
import asyncpg
import logging
from app.core.config import settings

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def check_and_fix_partitions():
    """Check existing partitions and create missing ones"""
    try:
        logger.info("Connecting to database...")
        conn = await asyncpg.connect(settings.get_database_url())
        
        # Check existing partitions
        partitions = await conn.fetch('''
            SELECT schemaname, tablename 
            FROM pg_tables 
            WHERE tablename LIKE 'outbox_%' 
            ORDER BY tablename
        ''')
        
        logger.info("Existing outbox partitions:")
        for partition in partitions:
            logger.info(f"  - {partition['tablename']}")
        
        # List of services that need partitions
        services_needing_partitions = [
            'enhanced-mixin-test-service',
            'device-data-ingestion-service',  # Make sure this exists
            'test-service'  # For general testing
        ]
        
        logger.info("\nChecking and creating missing partitions...")
        
        for service_name in services_needing_partitions:
            try:
                # Check if partition already exists
                partition_name = f"outbox_{service_name.replace('-', '_')}"
                existing = await conn.fetchval('''
                    SELECT tablename 
                    FROM pg_tables 
                    WHERE tablename = $1
                ''', partition_name)
                
                if existing:
                    logger.info(f"✅ Partition {partition_name} already exists")
                else:
                    # Create partition using the utility function
                    await conn.execute('SELECT add_service_partition($1)', service_name)
                    logger.info(f"✅ Created partition for service: {service_name}")
                    
            except Exception as e:
                logger.error(f"❌ Failed to create partition for {service_name}: {e}")
        
        # Test inserting a sample event to verify the partition works
        logger.info("\nTesting event insertion...")
        
        test_query = """
            INSERT INTO global_event_outbox (
                origin_service, idempotency_key, kafka_topic, 
                event_payload, event_type, status
            ) VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id
        """
        
        test_event_id = await conn.fetchval(
            test_query,
            'enhanced-mixin-test-service',
            'test-key-123',
            'test-topic',
            b'{"test": "data"}',
            'test.event',
            'pending'
        )
        
        if test_event_id:
            logger.info(f"✅ Test event inserted successfully: {test_event_id}")
            
            # Clean up test event
            await conn.execute("DELETE FROM global_event_outbox WHERE id = $1", test_event_id)
            logger.info("✅ Test event cleaned up")
        else:
            logger.error("❌ Test event insertion failed")
        
        # Show final partition list
        logger.info("\nFinal partition list:")
        final_partitions = await conn.fetch('''
            SELECT tablename 
            FROM pg_tables 
            WHERE tablename LIKE 'outbox_%' 
            ORDER BY tablename
        ''')
        
        for partition in final_partitions:
            logger.info(f"  - {partition['tablename']}")
        
        await conn.close()
        logger.info("✅ Partition check and fix completed successfully!")
        
    except Exception as e:
        logger.error(f"❌ Error during partition check: {e}")
        raise

if __name__ == "__main__":
    asyncio.run(check_and_fix_partitions())
