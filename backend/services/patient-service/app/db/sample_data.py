"""
Sample data for the Patient Service.

This module provides sample patient data for testing and development.
"""

import logging
import uuid
from datetime import datetime, timezone
from typing import List, Dict, Any

# Import MongoDB connection
from app.db.mongodb import get_patients_collection

# Configure logging
logger = logging.getLogger(__name__)

# Sample patient data
SAMPLE_PATIENTS = [
    {
        "resourceType": "Patient",
        "id": "patient-001",
        "meta": {
            "versionId": "1",
            "lastUpdated": datetime.now(timezone.utc).isoformat()
        },
        "active": True,
        "name": [
            {
                "use": "official",
                "family": "Smith",
                "given": ["John", "Adam"]
            }
        ],
        "telecom": [
            {
                "system": "phone",
                "value": "555-123-4567",
                "use": "home"
            },
            {
                "system": "email",
                "value": "john.smith@example.com",
                "use": "work"
            }
        ],
        "gender": "male",
        "birthDate": "1970-01-01",
        "address": [
            {
                "use": "home",
                "line": ["123 Main St"],
                "city": "Anytown",
                "state": "CA",
                "postalCode": "12345",
                "country": "USA"
            }
        ]
    },
    {
        "resourceType": "Patient",
        "id": "patient-002",
        "meta": {
            "versionId": "1",
            "lastUpdated": datetime.now(timezone.utc).isoformat()
        },
        "active": True,
        "name": [
            {
                "use": "official",
                "family": "Johnson",
                "given": ["Emily", "Rose"]
            }
        ],
        "telecom": [
            {
                "system": "phone",
                "value": "555-987-6543",
                "use": "mobile"
            },
            {
                "system": "email",
                "value": "emily.johnson@example.com",
                "use": "work"
            }
        ],
        "gender": "female",
        "birthDate": "1985-05-15",
        "address": [
            {
                "use": "home",
                "line": ["456 Oak Ave"],
                "city": "Somewhere",
                "state": "NY",
                "postalCode": "67890",
                "country": "USA"
            }
        ]
    },
    {
        "resourceType": "Patient",
        "id": "patient-003",
        "meta": {
            "versionId": "1",
            "lastUpdated": datetime.now(timezone.utc).isoformat()
        },
        "active": True,
        "name": [
            {
                "use": "official",
                "family": "Williams",
                "given": ["Robert", "James"]
            }
        ],
        "telecom": [
            {
                "system": "phone",
                "value": "555-456-7890",
                "use": "home"
            }
        ],
        "gender": "male",
        "birthDate": "1965-08-22",
        "address": [
            {
                "use": "home",
                "line": ["789 Pine St"],
                "city": "Elsewhere",
                "state": "TX",
                "postalCode": "54321",
                "country": "USA"
            }
        ]
    }
]

async def import_sample_data() -> int:
    """
    Import sample patient data into the database.
    
    Returns:
        The number of patients imported
    """
    try:
        # Get the patients collection
        collection = get_patients_collection()
        if not collection:
            logger.error("Failed to get patients collection")
            return 0
            
        # Check if data already exists
        count = await collection.count_documents({})
        if count > 0:
            logger.info(f"Database already contains {count} patients, skipping import")
            return count
            
        # Import sample data
        result = await collection.insert_many(SAMPLE_PATIENTS)
        
        if result.inserted_ids:
            logger.info(f"Successfully imported {len(result.inserted_ids)} sample patients")
            return len(result.inserted_ids)
        else:
            logger.error("Failed to import sample patients")
            return 0
    except Exception as e:
        logger.error(f"Error importing sample data: {str(e)}")
        return 0
