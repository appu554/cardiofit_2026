#!/usr/bin/env python
"""
Script to fix duplicate IDs in the Patient collection.

This script finds all patients with duplicate IDs and assigns new IDs to them.
"""

import asyncio
import logging
import os
import sys
import uuid
from typing import Dict, List, Any

# Add the parent directory to the path so we can import from app
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)

# Import MongoDB connection
from app.db.mongodb import connect_to_mongo, close_mongo_connection, db
from app.core.config import settings

async def fix_duplicate_ids():
    """
    Fix duplicate IDs in the database.
    
    This function finds all patients with duplicate IDs and assigns new IDs to them.
    """
    logger.info("Connecting to MongoDB...")
    connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)
    
    if not connection_success:
        logger.error("Failed to connect to MongoDB. Exiting.")
        return
    
    logger.info(f"Connected to MongoDB. Database: {settings.MONGODB_DB_NAME}")
    
    try:
        # Get the patients collection
        collection = db.db.patients
        
        logger.info("Checking for duplicate Patient IDs in the database...")
        
        # Find all patient IDs
        pipeline = [
            {"$match": {"resourceType": "Patient"}},
            {"$group": {"_id": "$id", "count": {"$sum": 1}, "docs": {"$push": "$$ROOT"}}},
            {"$match": {"count": {"$gt": 1}}}
        ]
        
        cursor = collection.aggregate(pipeline)
        duplicate_groups = []
        async for group in await cursor.to_list(length=None):
            duplicate_groups.append(group)
        
        if not duplicate_groups:
            logger.info("No duplicate Patient IDs found in the database")
            return
            
        logger.warning(f"Found {len(duplicate_groups)} groups of duplicate Patient IDs")
        
        # Fix each group of duplicates
        for group in duplicate_groups:
            duplicate_id = group["_id"]
            docs = group["docs"]
            logger.warning(f"Fixing {len(docs)} patients with duplicate ID '{duplicate_id}'")
            
            # Keep the first document as is, update the rest with new IDs
            for i, doc in enumerate(docs[1:], 1):
                old_id = doc["id"]
                new_id = str(uuid.uuid4())
                
                # Update the document with a new ID
                doc["id"] = new_id
                
                try:
                    # Remove the old document
                    await collection.delete_one({"_id": doc["_id"]})
                    
                    # Insert the updated document
                    doc_copy = dict(doc)
                    if "_id" in doc_copy:
                        del doc_copy["_id"]  # Let MongoDB generate a new _id
                        
                    await collection.insert_one(doc_copy)
                    logger.info(f"Updated Patient ID from '{old_id}' to '{new_id}'")
                except Exception as e:
                    logger.error(f"Error updating Patient with duplicate ID: {str(e)}")
        
        logger.info("Finished fixing duplicate Patient IDs")
    except Exception as e:
        logger.error(f"Error fixing duplicate Patient IDs: {str(e)}")
    finally:
        # Close the MongoDB connection
        await close_mongo_connection()

async def main():
    """Main function."""
    logger.info("Starting fix_duplicate_ids script...")
    await fix_duplicate_ids()
    logger.info("Script completed.")

if __name__ == "__main__":
    asyncio.run(main())
