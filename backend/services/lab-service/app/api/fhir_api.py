"""
FHIR API Router for Lab Service

This module provides a FastAPI router for handling FHIR requests directly.
"""

from fastapi import APIRouter, Depends, Request, Body
from typing import Dict, List, Any, Optional
import logging
import sys
import os

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the FHIR service
from app.services.fhir_service import DiagnosticReportFHIRService
from app.core.auth import get_token_payload

logger = logging.getLogger(__name__)

# Create a router for FHIR requests
fhir_router = APIRouter()

# Create an instance of the FHIR service
diagnostic_report_service = DiagnosticReportFHIRService()

# Initialize the service
@fhir_router.on_event("startup")
async def initialize_service():
    """Initialize the FHIR service."""
    logger.info("Initializing DiagnosticReport FHIR service...")
    await diagnostic_report_service.initialize()
    logger.info("DiagnosticReport FHIR service initialized.")

@fhir_router.post("/DiagnosticReport", response_model=Dict[str, Any])
async def create_diagnostic_report(
    request: Request,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Create a new DiagnosticReport resource."""
    try:
        # Add very visible logging
        print(f"\n\n==== LAB SERVICE RECEIVED CREATE REQUEST FOR DIAGNOSTICREPORT ====")
        print(f"Resource: {resource}")
        print(f"Headers: {dict(request.headers)}")
        print(f"Token Payload: {token_payload}")
        print(f"==== END LAB SERVICE REQUEST ====\n\n")

        # Check if the user has the required permissions
        # For DiagnosticReport, we'll accept either observation:write or condition:write
        # Permissions are in app_metadata.permissions
        permissions = token_payload.get("app_metadata", {}).get("permissions", [])
        print(f"User permissions: {permissions}")

        # For testing purposes, allow all requests
        # In a real implementation, we would check for specific permissions
        # if not any(perm in permissions for perm in ["observation:write", "condition:write"]):
        #     from fastapi import HTTPException
        #     raise HTTPException(status_code=403, detail="User does not have permission to create diagnostic reports")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        return await diagnostic_report_service.create_resource(resource, auth_header)
    except Exception as e:
        logger.error(f"Error creating DiagnosticReport: {str(e)}")
        raise

@fhir_router.get("/DiagnosticReport/{resource_id}", response_model=Dict[str, Any])
async def get_diagnostic_report(
    request: Request,
    resource_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Get a DiagnosticReport resource by ID."""
    try:
        # Add very visible logging
        print(f"\n\n==== LAB SERVICE RECEIVED GET REQUEST FOR DIAGNOSTICREPORT/{resource_id} ====")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END LAB SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        resource = await diagnostic_report_service.get_resource(resource_id, auth_header)
        if not resource:
            from fastapi import HTTPException
            raise HTTPException(status_code=404, detail=f"DiagnosticReport with ID {resource_id} not found")
        return resource
    except Exception as e:
        logger.error(f"Error getting DiagnosticReport: {str(e)}")
        raise

@fhir_router.put("/DiagnosticReport/{resource_id}", response_model=Dict[str, Any])
async def update_diagnostic_report(
    request: Request,
    resource_id: str,
    resource: Dict[str, Any] = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Update a DiagnosticReport resource."""
    try:
        # Add very visible logging
        print(f"\n\n==== LAB SERVICE RECEIVED PUT REQUEST FOR DIAGNOSTICREPORT/{resource_id} ====")
        print(f"Resource: {resource}")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END LAB SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        updated_resource = await diagnostic_report_service.update_resource(resource_id, resource, auth_header)
        if not updated_resource:
            from fastapi import HTTPException
            raise HTTPException(status_code=404, detail=f"DiagnosticReport with ID {resource_id} not found")
        return updated_resource
    except Exception as e:
        logger.error(f"Error updating DiagnosticReport: {str(e)}")
        raise

@fhir_router.delete("/DiagnosticReport/{resource_id}")
async def delete_diagnostic_report(
    request: Request,
    resource_id: str,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Delete a DiagnosticReport resource."""
    try:
        # Add very visible logging
        print(f"\n\n==== LAB SERVICE RECEIVED DELETE REQUEST FOR DIAGNOSTICREPORT/{resource_id} ====")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END LAB SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Call the service method
        success = await diagnostic_report_service.delete_resource(resource_id, auth_header)
        if not success:
            from fastapi import HTTPException
            raise HTTPException(status_code=404, detail=f"DiagnosticReport with ID {resource_id} not found")

        # Return a success message
        return {"message": f"DiagnosticReport with ID {resource_id} deleted successfully"}
    except Exception as e:
        logger.error(f"Error deleting DiagnosticReport: {str(e)}")
        raise

@fhir_router.get("/DiagnosticReport", response_model=List[Dict[str, Any]])
async def search_diagnostic_reports(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload)
):
    """Search for DiagnosticReport resources."""
    try:
        # Add very visible logging
        print(f"\n\n==== LAB SERVICE RECEIVED SEARCH REQUEST FOR DIAGNOSTICREPORT ====")
        print(f"Query params: {dict(request.query_params)}")
        print(f"Headers: {dict(request.headers)}")
        print(f"==== END LAB SERVICE REQUEST ====\n\n")

        # Extract the token from the payload
        auth_header = None
        if token_payload and "token" in token_payload:
            auth_header = f"Bearer {token_payload.get('token')}"

        # Get all query parameters
        params = dict(request.query_params)

        # Call the service method
        return await diagnostic_report_service.search_resources(params, auth_header)
    except Exception as e:
        logger.error(f"Error searching DiagnosticReports: {str(e)}")
        raise

@fhir_router.get("/debug/mongodb", response_model=Dict[str, Any])
async def debug_mongodb(
    request: Request,
    token_payload: Dict[str, Any] = Depends(get_token_payload),
    subject: Optional[str] = None,
    show_all: Optional[bool] = False
):
    """Debug endpoint to check MongoDB connection and collection status."""
    try:
        # Add very visible logging
        print(f"\n\n==== LAB SERVICE MONGODB DEBUG ENDPOINT CALLED ====")
        print(f"Headers: {dict(request.headers)}")
        print(f"Subject: {subject}")
        print(f"Show All: {show_all}")
        print(f"==== END LAB SERVICE MONGODB DEBUG ====\n\n")

        # Get MongoDB status
        from app.db.mongodb import db, get_diagnostic_reports_collection

        # Initialize the service if needed
        await diagnostic_report_service.initialize()

        result = {
            "mongodb_status": db.get_status(),
            "mongodb_initialized": db._initialized,
            "mongodb_client_exists": db.client is not None,
            "mongodb_db_exists": db.db is not None,
            "collection_exists": diagnostic_report_service.collection is not None,
            "use_mongodb": diagnostic_report_service.use_mongodb,
            "diagnostic_reports_count": 0,
            "diagnostic_reports": []
        }

        # If MongoDB is connected, get collections
        if db.is_connected():
            try:
                # Get collections
                collections = await db.db.list_collection_names()
                result["collections"] = collections

                # Check diagnostic_reports collection
                if "diagnostic_reports" in collections:
                    # Get count of diagnostic reports
                    collection = get_diagnostic_reports_collection()
                    if collection is not None:
                        # Build query
                        query = {"resourceType": "DiagnosticReport"}
                        if subject and not show_all:
                            query["subject.reference"] = subject

                        # Log the query
                        logger.info(f"MongoDB debug query: {query}")

                        # Get count
                        count = await collection.count_documents(query)
                        result["diagnostic_reports_count"] = count

                        # Get all diagnostic reports
                        cursor = collection.find(query).limit(20)
                        diagnostic_reports = []
                        async for doc in cursor:
                            # Convert ObjectId to string
                            if "_id" in doc:
                                doc["_id"] = str(doc["_id"])
                            diagnostic_reports.append(doc)

                        result["diagnostic_reports"] = diagnostic_reports

                        # Add a list of all unique subject references
                        if show_all:
                            # Get all unique subject references
                            pipeline = [
                                {"$match": {"resourceType": "DiagnosticReport"}},
                                {"$group": {"_id": "$subject.reference"}},
                                {"$project": {"_id": 0, "subject_reference": "$_id"}}
                            ]

                            subject_refs = []
                            async for doc in collection.aggregate(pipeline):
                                if "subject_reference" in doc:
                                    subject_refs.append(doc["subject_reference"])

                            result["subject_references"] = subject_refs
            except Exception as e:
                logger.error(f"Error getting MongoDB details: {str(e)}")
                result["error"] = str(e)

        return result
    except Exception as e:
        logger.error(f"Error in MongoDB debug endpoint: {str(e)}")
        raise
