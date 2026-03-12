#!/usr/bin/env python3
"""
Clear All Clinical Knowledge Data from Neo4j
This script removes all nodes with labels 'cae_Drug', 'cae_SNOMEDConcept',
and 'cae_LOINCConcept' from the database to prepare for a full re-ingestion.
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.logging_config import get_logger
from core.database_factory import create_database_client

logger = get_logger(__name__)

async def clear_all_data():
    """Connects to Neo4j and removes all clinical knowledge graph nodes."""
    print("\U0001f9f9 SCRIPT TO CLEAR ALL CLINICAL KNOWLEDGE DATA")
    print("=" * 60)

    database_client = None
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

        labels_to_delete = ["cae_Drug", "cae_SNOMEDConcept", "cae_LOINCConcept", "LOINCCode"]

        for label in labels_to_delete:
            # Count the nodes
            count_query = f"MATCH (n:`{label}`) RETURN count(n) AS count"
            logger.info(f"Checking for existing nodes with label '{label}'...", query=count_query)
            print(f"\n\U0001f50d Checking for nodes with label '{label}'...")
            count_result = await database_client.execute_cypher(count_query)
            node_count = count_result[0]['count'] if count_result else 0

            if node_count == 0:
                logger.info(f"No '{label}' nodes found to delete.")
                print(f"\u2705 No '{label}' nodes found. Skipping.")
                continue

            print(f"\u26a0\ufe0f Found {node_count} nodes with label '{label}' to delete.")

            # Delete the nodes
            delete_query = f"MATCH (n:`{label}`) DETACH DELETE n"
            logger.info(f"Executing deletion of {node_count} nodes...", query=delete_query)
            print(f"\U0001f5d1\ufe0f Deleting nodes and their relationships...")
            await database_client.execute_cypher(delete_query)

            logger.info(f"Successfully deleted {node_count} '{label}' nodes.")
            print(f"\u2705 Successfully deleted {node_count} nodes.")

        return True

    except Exception as e:
        logger.error("\U0001f4a5 An unexpected error occurred", error=str(e), exc_info=True)
        print(f"\U0001f4a5 An unexpected error occurred: {e}")
        return False
    finally:
        if database_client:
            await database_client.disconnect()
            logger.info("\U0001f4a4 Database connection closed.")
            print("\U0001f4a4 Database connection closed.")

if __name__ == "__main__":
    success = asyncio.run(clear_all_data())
    
    if success:
        print("\n\U0001f389 Full cleanup script completed successfully!")
    else:
        print("\n\u274c Cleanup script failed. Please review the logs.")
    
    sys.exit(0 if success else 1)
