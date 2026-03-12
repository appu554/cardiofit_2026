#!/usr/bin/env python3
"""
Verify Knowledge Graph Schema
This script checks the Neo4j database for the correct node labels.
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.logging_config import get_logger
from core.database_factory import create_database_client

logger = get_logger(__name__)

async def verify_schema():
    """Connects to Neo4j and verifies node labels."""
    print("\U0001f575\ufe0f SCRIPT TO VERIFY NEO4J SCHEMA")
    print("=" * 50)

    database_client = None
    all_checks_passed = True
    try:
        # Connect to the database
        logger.info("\U0001f50c Connecting to the database...")
        print("\U0001f50c Connecting to the database...")
        database_client = await create_database_client()
        connection_ok = await database_client.test_connection()
        if not connection_ok:
            logger.error("\u274c Database connection failed. Aborting.")
            print("\u274c Database connection failed. Aborting.")
            return False
        logger.info("\u2705 Database connection successful.")
        print("\u2705 Database connection successful.")

        # Verification checks
        checks = {
            "cae_LOINCConcept": True, # Expect > 0
            "LOINCCode": False,        # Expect 0
            "cae_Drug": True,          # Expect > 0
            "cae_SNOMEDConcept": True  # Expect > 0
        }

        for label, should_exist in checks.items():
            print(f"\n\U0001f50d Verifying label: '{label}'...")
            query = f"MATCH (n:`{label}`) RETURN count(n) AS count"
            result = await database_client.execute_cypher(query)
            count = result[0]['count'] if result else 0

            if should_exist and count > 0:
                print(f"\u2705 SUCCESS: Found {count} nodes with label '{label}'.")
                logger.info(f"Check passed for label '{label}'", count=count)
            elif not should_exist and count == 0:
                print(f"\u2705 SUCCESS: Confirmed 0 nodes with label '{label}'.")
                logger.info(f"Check passed for label '{label}'", count=count)
            else:
                if should_exist:
                    print(f"\u274c FAILED: Expected nodes with label '{label}', but found {count}.")
                    logger.error(f"Check failed for label '{label}'", expected=">0", found=count)
                else:
                    print(f"\u274c FAILED: Expected 0 nodes with label '{label}', but found {count}.")
                    logger.error(f"Check failed for label '{label}'", expected=0, found=count)
                all_checks_passed = False

    except Exception as e:
        logger.error("\U0001f4a5 An unexpected error occurred", error=str(e), exc_info=True)
        print(f"\U0001f4a5 An unexpected error occurred: {e}")
        all_checks_passed = False
    finally:
        if database_client:
            await database_client.disconnect()
            logger.info("\U0001f4a4 Database connection closed.")
            print("\U0001f4a4 Database connection closed.")
    
    return all_checks_passed

if __name__ == "__main__":
    success = asyncio.run(verify_schema())
    
    if success:
        print("\n\U0001f389 All schema checks passed! The knowledge graph is correctly set up.")
    else:
        print("\n\u26a0\ufe0f Some schema checks failed. Please review the output.")
    
    sys.exit(0 if success else 1)
