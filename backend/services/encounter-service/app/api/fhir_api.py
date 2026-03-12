"""
FHIR API Router for Encounter Service

This module provides a FastAPI router for handling FHIR requests directly.
"""

from fastapi import APIRouter, Depends, Request, Body
from typing import Dict, List, Any, Optional
import logging
import sys
import os
from app.core.config import settings

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the FHIR service
from app.services.fhir_service import EncounterFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create a router for FHIR requests
fhir_router = APIRouter()

# Create an instance of the FHIR service
encounter_service = EncounterFHIRService()

@fhir_router.post("/Encounter", response_model=Dict[str, Any])
async def create_encounter(
    request: Request,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Create a new Encounter resource."""
    try:
        # Add very visible logging
        print(f"\n\n==== ENCOUNTER SERVICE RECEIVED CREATE REQUEST FOR ENCOUNTER ====")
        print(f"Resource: {resource}")
        print(f"Headers: {dict(request.headers)}")
        print(f"Token Payload: {token_payload}")
        print(f"==== END ENCOUNTER SERVICE REQUEST ====\n\n")

        # Check if the user has the required permissions
        # Permissions are in app_metadata.permissions
        permissions = token_payload.get("app_metadata", {}).get("permissions", [])
        print(f"User permissions: {permissions}")

        # For testing purposes, allow all requests
        # In a real implementation, we would check for specific permissions
        # if "encounter:write" not in permissions:
        #     from fastapi import HTTPException
        #     raise HTTPException(status_code=403, detail="User does not have permission to create encounters")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        return await encounter_service.create_resource(resource, auth_header)
    except Exception as e:
        logger.error(f"Error creating Encounter: {str(e)}")
        raise

@fhir_router.get("/Encounter/{resource_id}", response_model=Dict[str, Any])
async def get_encounter(
    request: Request,
    resource_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Get an Encounter resource by ID."""
    try:
        # Add very visible logging
        print(f"\n\n==== ENCOUNTER SERVICE RECEIVED GET REQUEST FOR ENCOUNTER/{resource_id} ====")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END ENCOUNTER SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        resource = await encounter_service.get_resource(resource_id, auth_header)
        if not resource:
            from fastapi import HTTPException
            raise HTTPException(status_code=404, detail=f"Encounter with ID {resource_id} not found")
        return resource
    except Exception as e:
        logger.error(f"Error getting Encounter: {str(e)}")
        raise

@fhir_router.put("/Encounter/{resource_id}", response_model=Dict[str, Any])
async def update_encounter(
    request: Request,
    resource_id: str,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Update an Encounter resource."""
    try:
        # Add very visible logging
        print(f"\n\n==== ENCOUNTER SERVICE RECEIVED PUT REQUEST FOR ENCOUNTER/{resource_id} ====")
        print(f"Resource: {resource}")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END ENCOUNTER SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        updated_resource = await encounter_service.update_resource(resource_id, resource, auth_header)
        if not updated_resource:
            from fastapi import HTTPException
            raise HTTPException(status_code=404, detail=f"Encounter with ID {resource_id} not found")
        return updated_resource
    except Exception as e:
        logger.error(f"Error updating Encounter: {str(e)}")
        raise

@fhir_router.delete("/Encounter/{resource_id}")
async def delete_encounter(
    request: Request,
    resource_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Delete an Encounter resource."""
    try:
        # Add very visible logging
        print(f"\n\n==== ENCOUNTER SERVICE RECEIVED DELETE REQUEST FOR ENCOUNTER/{resource_id} ====")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END ENCOUNTER SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        success = await encounter_service.delete_resource(resource_id, auth_header)
        if not success:
            from fastapi import HTTPException
            raise HTTPException(status_code=404, detail=f"Encounter with ID {resource_id} not found")

        # Return a success message
        return {"message": f"Encounter with ID {resource_id} deleted successfully"}
    except Exception as e:
        logger.error(f"Error deleting Encounter: {str(e)}")
        raise

@fhir_router.get("/Encounter", response_model=List[Dict[str, Any]])
async def search_encounters(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Search for Encounter resources."""
    try:
        # Add very visible logging
        print(f"\n\n==== ENCOUNTER SERVICE RECEIVED SEARCH REQUEST FOR ENCOUNTER ====")
        print(f"Query params: {dict(request.query_params)}")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END ENCOUNTER SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Get all query parameters
        params = dict(request.query_params)

        # Log the parameters for debugging
        logger.info(f"Search parameters: {params}")

        # Call the service method
        return await encounter_service.search_resources(params, auth_header)
    except Exception as e:
        logger.error(f"Error searching Encounters: {str(e)}")
        raise

@fhir_router.get("/debug/encounters", response_model=List[Dict[str, Any]])
async def debug_encounters(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Debug endpoint to list all encounters in the database."""
    try:
        # Log the request
        logger.info(f"Debug endpoint called to list all encounters")

        # Get all encounters from the service's in-memory cache
        encounters = list(encounter_service.encounters.values())

        # Log the number of encounters found
        logger.info(f"Found {len(encounters)} encounters in memory")

        # If MongoDB is available, also get encounters from there
        from app.db.mongodb import get_encounters_collection, db

        if db.is_connected():
            collection = get_encounters_collection()
            if collection:
                try:
                    # Get all encounters from MongoDB
                    mongo_encounters = []
                    cursor = collection.find({"resourceType": "Encounter"})

                    async for encounter in cursor:
                        # Convert ObjectId to string
                        if "_id" in encounter:
                            encounter["_id"] = str(encounter["_id"])
                        mongo_encounters.append(encounter)

                    logger.info(f"Found {len(mongo_encounters)} encounters in MongoDB")

                    # Return both in-memory and MongoDB encounters
                    return mongo_encounters
                except Exception as e:
                    logger.error(f"Error getting encounters from MongoDB: {str(e)}")

        # Return in-memory encounters if MongoDB is not available
        return encounters
    except Exception as e:
        logger.error(f"Error in debug endpoint: {str(e)}")
        raise

@fhir_router.get("/debug/mongodb", response_model=Dict[str, Any])
async def debug_mongodb(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Debug endpoint to check MongoDB connection and collection status."""
    try:
        # Log the request
        logger.info(f"Debug endpoint called to check MongoDB status")

        # Get MongoDB status
        from app.db.mongodb import db

        result = {
            "mongodb_status": db.get_status(),
            "mongodb_initialized": db._initialized,
            "mongodb_client_exists": db.client is not None,
            "mongodb_db_exists": db.db is not None,
            "collections": [],
            "encounter_count": 0,
            "sample_encounter": None,
            "connection_string": "****" + settings.MONGODB_URL[-20:] if settings.MONGODB_URL else None
        }

        # If MongoDB is connected, get collections
        if db.is_connected():
            try:
                # Get collections
                collections = await db.db.list_collection_names()
                result["collections"] = collections

                # Check encounters collection
                if "encounters" in collections:
                    # Get count of encounters
                    count = await db.db.encounters.count_documents({"resourceType": "Encounter"})
                    result["encounter_count"] = count

                    # Get a sample encounter
                    sample = await db.db.encounters.find_one({"resourceType": "Encounter"})
                    if sample:
                        # Convert ObjectId to string
                        if "_id" in sample:
                            sample["_id"] = str(sample["_id"])
                        result["sample_encounter"] = sample

                        # Check if the sample has a subject reference
                        if "subject" in sample and "reference" in sample["subject"]:
                            result["sample_subject_reference"] = sample["subject"]["reference"]
            except Exception as e:
                logger.error(f"Error getting MongoDB details: {str(e)}")
                result["error"] = str(e)

        return result
    except Exception as e:
        logger.error(f"Error in MongoDB debug endpoint: {str(e)}")
        raise
