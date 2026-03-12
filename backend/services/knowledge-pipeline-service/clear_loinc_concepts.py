#!/usr/bin/env python3
"""
Clear LOINC Concepts from Neo4j
This script removes all nodes with the label 'cae_LOINCConcept' from the database.
"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.append(str(Path(__file__).parent / "src"))

from core.logging_config import get_logger
from core.database_factory import create_database_client

logger = get_logger(__name__)

async def clear_loinc_data():
    """Connects to Neo4j and removes all cae_LOINCConcept nodes."""
    print("\U0001f9f9 SCRIPT TO CLEAR LOINC CONCEPTS")
    print("=" * 50)

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

        # Step 1: Count the nodes to be deleted
        count_query = "MATCH (n:cae_LOINCConcept) RETURN count(n) AS count"
        logger.info("Checking for existing LOINC concepts...", query=count_query)
        print("\n\U0001f50d Step 1: Checking for existing LOINC concepts...")
        count_result = await database_client.execute_cypher(count_query)
        
        node_count = count_result[0]['count'] if count_result else 0

        if node_count == 0:
            logger.info("No 'cae_LOINCConcept' nodes found to delete.")
            print("\u2705 No 'cae_LOINCConcept' nodes found. Nothing to do.")
            return True

        print(f"\u26a0\ufe0f Found {node_count} nodes with label 'cae_LOINCConcept' to delete.")

        # Step 2: Delete the nodes
        delete_query = "MATCH (n:cae_LOINCConcept) DETACH DELETE n"
        logger.info(f"Executing deletion of {node_count} nodes...", query=delete_query)
        print("\n\U0001f5d1\ufe0f Step 2: Deleting nodes and their relationships...")
        
        # Note: The result of a DELETE query is empty, but we can check for errors.
        await database_client.execute_cypher(delete_query)

        logger.info(f"Successfully deleted {node_count} 'cae_LOINCConcept' nodes.")
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
    # Run the main async function
    success = asyncio.run(clear_loinc_data())
    
    if success:
        print("\n\U0001f389 Cleanup script completed successfully!")
    else:
        print("\n\u274c Cleanup script failed. Please review the logs.")
    
    sys.exit(0 if success else 1)
