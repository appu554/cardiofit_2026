#!/usr/bin/env python3
"""
SNOMED CT Ingestion Pipeline
This script runs the ingestion process for the full SNOMED CT terminology.

Prerequisites:
1. Download the 'SNOMED CT International RF2' zip file.
2. Extract the zip file.
3. Move the 'Snapshot' folder from the extracted contents into:
   'backend/services/knowledge-pipeline-service/data/snomed/'
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

# Setup logging first
from core.logging_config import setup_pipeline_logging, get_logger
from core.database_factory import create_database_client
from ingesters.snomed_ingester import SNOMEDIngester

async def main():
    """Main function to run the SNOMED CT ingestion pipeline."""
    
    pipeline_logger = setup_pipeline_logging()
    logger = get_logger(__name__)
    
    print("\U0001f3e5 CLINICAL KNOWLEDGE PIPELINE - SNOMED CT")
    print("=" * 60)
    
    database_client = None
    success = True
    try:
        # Step 1: Connect to the database
        logger.info("\U0001f50c Connecting to the database...")
        print("\n\U0001f50c Step 1: Connecting to the database...")
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            logger.error("\u274c Database connection failed. Aborting.")
            print("\u274c Database connection failed. Aborting.")
            return False
        logger.info("\u2705 Database connection successful.")
        print("\u2705 Database connection successful.")

        # Step 2: Run SNOMED CT Ingester
        logger.info("\U0001f9ec Running SNOMED CT Ingester...")
        print("\n\U0001f9ec Step 2: Running SNOMED CT Ingester...")
        ingester = SNOMEDIngester(database_client)
        result = await ingester.ingest(force_download=False)
        if result.success:
            logger.info("\u2705 SNOMED CT ingestion successful.", records=result.total_records_processed, duration=result.duration)
            print(f"\u2705 Ingestion successful! ({result.total_records_processed} records in {result.duration:.2f}s)")
        else:
            logger.error("\u274c SNOMED CT ingestion failed.", errors=result.errors)
            print(f"\u274c Ingestion failed. Check logs in {pipeline_logger.log_dir}")
            success = False

    except Exception as e:
        logger.error("\U0001f4a5 An unexpected error occurred in the pipeline", error=str(e), exc_info=True)
        print(f"\U0001f4a5 Pipeline failed with an unexpected error: {e}")
        success = False
    finally:
        if database_client:
            await database_client.disconnect()
            logger.info("\U0001f4a4 Database connection closed.")
            print("\U0001f4a4 Database connection closed.")

    if success:
        print("\n\U0001f389 SNOMED CT ingestion completed successfully!")
    else:
        print("\n\u26a0\ufe0f The ingestion task failed. Please review the logs.")
        
    return success

if __name__ == "__main__":
    print("NOTE: This script requires manual download and extraction of the full SNOMED CT International release.")
    print("Please see the script's docstring for setup instructions.")
    
    pipeline_success = asyncio.run(main())
    sys.exit(0 if pipeline_success else 1)
