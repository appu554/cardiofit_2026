import asyncio
import os
import uuid
from motor.motor_asyncio import AsyncIOMotorClient
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# MongoDB connection string
mongodb_uri = os.getenv("MONGODB_URI", "mongodb+srv://admin:Apoorva@554@cluster0.yqdzbvb.mongodb.net/fhirdb?retryWrites=true&w=majority&appName=Cluster0")

# Sample patient data
sample_patients = [
    {
        "resourceType": "Patient",
        "id": str(uuid.uuid4()),
        "identifier": [
            {
                "system": "http://example.org/fhir/ids",
                "value": "12345"
            }
        ],
        "active": True,
        "name": [
            {
                "family": "Smith",
                "given": ["John"]
            }
        ],
        "gender": "male",
        "birthDate": "1970-01-01",
        "address": [
            {
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
        "id": str(uuid.uuid4()),
        "identifier": [
            {
                "system": "http://example.org/fhir/ids",
                "value": "67890"
            }
        ],
        "active": True,
        "name": [
            {
                "family": "Johnson",
                "given": ["Jane"]
            }
        ],
        "gender": "female",
        "birthDate": "1980-05-15",
        "address": [
            {
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
        "id": str(uuid.uuid4()),
        "identifier": [
            {
                "system": "http://example.org/fhir/ids",
                "value": "24680"
            }
        ],
        "active": True,
        "name": [
            {
                "family": "Williams",
                "given": ["Robert", "James"]
            }
        ],
        "gender": "male",
        "birthDate": "1965-12-25",
        "address": [
            {
                "line": ["789 Pine St"],
                "city": "Elsewhere",
                "state": "TX",
                "postalCode": "24680",
                "country": "USA"
            }
        ]
    },
    {
        "resourceType": "Patient",
        "id": str(uuid.uuid4()),
        "identifier": [
            {
                "system": "http://example.org/fhir/ids",
                "value": "13579"
            }
        ],
        "active": True,
        "name": [
            {
                "family": "Brown",
                "given": ["Mary", "Elizabeth"]
            }
        ],
        "gender": "female",
        "birthDate": "1990-08-30",
        "address": [
            {
                "line": ["321 Elm St"],
                "city": "Nowhere",
                "state": "FL",
                "postalCode": "13579",
                "country": "USA"
            }
        ]
    },
    {
        "resourceType": "Patient",
        "id": str(uuid.uuid4()),
        "identifier": [
            {
                "system": "http://example.org/fhir/ids",
                "value": "97531"
            }
        ],
        "active": True,
        "name": [
            {
                "family": "Davis",
                "given": ["Michael"]
            }
        ],
        "gender": "male",
        "birthDate": "1975-03-10",
        "address": [
            {
                "line": ["654 Maple Ave"],
                "city": "Somewhere Else",
                "state": "CA",
                "postalCode": "97531",
                "country": "USA"
            }
        ]
    }
]

async def init_db():
    """Initialize the database with sample data."""
    try:
        # Connect to MongoDB
        client = AsyncIOMotorClient(mongodb_uri)
        db = client.get_database()
        
        # Check if collection exists and has data
        count = await db.Patient.count_documents({})
        if count > 0:
            print(f"Database already contains {count} patients. Skipping initialization.")
            return
        
        # Insert sample patients
        result = await db.Patient.insert_many(sample_patients)
        print(f"Inserted {len(result.inserted_ids)} patients into the database.")
        
        # Print the inserted patients
        async for patient in db.Patient.find({}):
            print(f"Patient: {patient['name'][0]['family']}, {patient['name'][0]['given'][0]} (ID: {patient['id']})")
        
    except Exception as e:
        print(f"Error initializing database: {str(e)}")
    finally:
        # Close the connection
        client.close()

if __name__ == "__main__":
    # Run the async function
    asyncio.run(init_db())
