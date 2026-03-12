#!/usr/bin/env python3
"""
SNOMED CT and LOINC Ingestion Pipeline
This script runs the ingestion process for SNOMED CT and LOINC terminologies.

Prerequisites:
1. SNOMED CT:
   - Download 'SnomedCT_InternationalRF2_PRODUCTION.zip'
   - Place it in 'backend/services/knowledge-pipeline-service/data/snomed/'
2. LOINC:
   - Download 'Loinc_current.zip' (specifically 'Loinc_X.XX_Text.zip')
   - Rename it to 'Loinc_current.zip'
   - Place it in 'backend/services/knowledge-pipeline-service/data/loinc/'
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
from ingesters.loinc_ingester import LOINCIngester

async def main():
    """Main function to run the SNOMED and LOINC ingestion pipeline."""
    
    pipeline_logger = setup_pipeline_logging()
    logger = get_logger(__name__)
    
    print("\U0001f3e5 CLINICAL KNOWLEDGE PIPELINE - SNOMED & LOINC")
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

        # Step 2: Run SNOMED Ingester
        logger.info("\U0001f9ec Running SNOMED CT Ingester...")
        print("\n\U0001f9ec Step 2: Running SNOMED CT Ingester...")
        snomed_ingester = SNOMEDIngester(database_client)
        snomed_result = await snomed_ingester.ingest(force_download=False)
        if snomed_result.success:
            logger.info("\u2705 SNOMED CT ingestion successful.", records=snomed_result.total_records_processed, duration=snomed_result.duration)
            print(f"\u2705 SNOMED CT ingestion successful! ({snomed_result.total_records_processed} records in {snomed_result.duration:.2f}s)")
        else:
            logger.error("\u274c SNOMED CT ingestion failed.", errors=snomed_result.errors)
            print(f"\u274c SNOMED CT ingestion failed. Check logs in {pipeline_logger.log_dir}")
            success = False # Mark as failed but continue to LOINC

        # Step 3: Run LOINC Ingester
        logger.info("\U0001f52c Running LOINC Ingester...")
        print("\n\U0001f52c Step 3: Running LOINC Ingester...")
        loinc_ingester = LOINCIngester(database_client)
        loinc_result = await loinc_ingester.ingest(force_download=False)
        if loinc_result.success:
            logger.info("\u2705 LOINC ingestion successful.", records=loinc_result.total_records_processed, duration=loinc_result.duration)
            print(f"\u2705 LOINC ingestion successful! ({loinc_result.total_records_processed} records in {loinc_result.duration:.2f}s)")
        else:
            logger.error("\u274c LOINC ingestion failed.", errors=loinc_result.errors)
            print(f"\u274c LOINC ingestion failed. Check logs in {pipeline_logger.log_dir}")
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
        print("\n\U0001f389 All ingestion tasks completed successfully!")
    else:
        print("\n\u26a0\ufe0f Some ingestion tasks failed. Please review the logs.")
        
    return success

if __name__ == "__main__":
    # Ensure you have the required data files before running
    print("NOTE: This script requires manual download of SNOMED CT and LOINC data files.")
    print("Please see the script's docstring for instructions.")
    
    pipeline_success = asyncio.run(main())
    sys.exit(0 if pipeline_success else 1)
