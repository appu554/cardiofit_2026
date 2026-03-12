from typing import List, Optional, Dict, Any, Union
from bson import ObjectId
from app.db.mongodb import db
from app.models.fhir import FHIRResource, DocumentReference
from shared.models import (
    Patient, Observation, Condition, Encounter,
    MedicationRequest, DiagnosticReport
)

# Singleton instance
_fhir_service_instance = None

def get_fhir_service():
    """Get or create a singleton instance of the FHIR service."""
    global _fhir_service_instance
    if _fhir_service_instance is None:
        _fhir_service_instance = FHIRService()
    return _fhir_service_instance

class FHIRService:
    """Service for handling FHIR resources."""

    @staticmethod
    def _format_id(id: str) -> str:
        """Format MongoDB ObjectId to string."""
        if isinstance(id, ObjectId):
            return str(id)
        return id

    @staticmethod
    def _prepare_resource_for_db(resource: Dict[str, Any]) -> Dict[str, Any]:
        """Prepare resource for database storage."""
        # Make a copy to avoid modifying the original
        resource_copy = resource.copy()

        # Remove id if it's None (MongoDB will generate one)
        if "id" in resource_copy and resource_copy["id"] is None:
            del resource_copy["id"]

        return resource_copy

    @staticmethod
    def _prepare_resource_from_db(resource: Dict[str, Any]) -> Dict[str, Any]:
        """Prepare resource from database for API response."""
        # Make a copy to avoid modifying the original
        resource_copy = resource.copy()

        # Convert MongoDB _id to FHIR id
        if "_id" in resource_copy:
            resource_copy["id"] = str(resource_copy["_id"])
            del resource_copy["_id"]

        return resource_copy

    async def create_resource(self, resource_type: str, resource: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new FHIR resource."""
        import logging
        import traceback
        from app.db.mongodb import check_connection

        logger = logging.getLogger(__name__)

        try:
            # Temporarily bypass connection check
            # connection_ok = await check_connection()
            # if not connection_ok:
            #     logger.error(f"MongoDB connection check failed before creating {resource_type}")
            #     raise Exception("Database connection is not available")
            # Assume connection is OK for now
            logger.info(f"Proceeding with creating {resource_type} resource")

            # Log the resource being created
            logger.debug(f"Creating {resource_type} resource: {resource}")

            # Ensure resourceType is set correctly
            resource["resourceType"] = resource_type

            # Prepare resource for database
            db_resource = self._prepare_resource_for_db(resource)
            logger.debug(f"Prepared resource for DB: {db_resource}")

            # Insert into database
            logger.info(f"Inserting {resource_type} into database...")
            result = await db.db[resource_type].insert_one(db_resource)
            logger.info(f"Inserted {resource_type} with ID: {result.inserted_id}")

            # Get the created resource
            logger.info(f"Retrieving created {resource_type}...")
            created_resource = await db.db[resource_type].find_one({"_id": result.inserted_id})

            if not created_resource:
                logger.error(f"Could not retrieve created {resource_type} with ID: {result.inserted_id}")
                raise Exception(f"Resource was created but could not be retrieved")

            # Prepare for API response
            response = self._prepare_resource_from_db(created_resource)
            logger.debug(f"Returning created {resource_type}: {response}")
            return response
        except Exception as e:
            logger.error(f"Error creating {resource_type}: {str(e)}")
            logger.error(traceback.format_exc())
            raise

    async def get_resource(self, resource_type: str, resource_id: str) -> Optional[Dict[str, Any]]:
        """Get a FHIR resource by ID."""
        # Try to convert to ObjectId if it's a valid format
        try:
            if ObjectId.is_valid(resource_id):
                resource_id = ObjectId(resource_id)
        except:
            pass

        # Find the resource
        resource = await db.db[resource_type].find_one({"_id": resource_id})

        if not resource:
            return None

        # Prepare for API response
        return self._prepare_resource_from_db(resource)

    async def update_resource(self, resource_type: str, resource_id: str, resource: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Update a FHIR resource."""
        # Ensure resourceType is set correctly
        resource["resourceType"] = resource_type

        # Try to convert to ObjectId if it's a valid format
        try:
            if ObjectId.is_valid(resource_id):
                resource_id = ObjectId(resource_id)
        except:
            pass

        # Prepare resource for database
        db_resource = self._prepare_resource_for_db(resource)

        # Update in database
        result = await db.db[resource_type].replace_one({"_id": resource_id}, db_resource)

        if result.matched_count == 0:
            return None

        # Get the updated resource
        updated_resource = await db.db[resource_type].find_one({"_id": resource_id})

        # Prepare for API response
        return self._prepare_resource_from_db(updated_resource)

    async def delete_resource(self, resource_type: str, resource_id: str) -> bool:
        """Delete a FHIR resource."""
        # Try to convert to ObjectId if it's a valid format
        try:
            if ObjectId.is_valid(resource_id):
                resource_id = ObjectId(resource_id)
        except:
            pass

        # Delete from database
        result = await db.db[resource_type].delete_one({"_id": resource_id})

        return result.deleted_count > 0

    async def search_resources(self, resource_type: str, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Search for FHIR resources."""
        # Build query from params
        query = {}

        # Handle common search parameters
        for key, value in params.items():
            # Skip pagination parameters
            if key in ["_count", "_page"]:
                continue

            # Handle special parameters
            if key == "_id":
                try:
                    if ObjectId.is_valid(value):
                        query["_id"] = ObjectId(value)
                    else:
                        query["_id"] = value
                except:
                    query["_id"] = value
            elif key == "subject" or key == "patient":
                # Handle reference search (e.g., subject=Patient/123)
                query["subject.reference"] = value
            elif key == "code":
                # Handle token search (e.g., code=http://loinc.org|4548-4)
                if "|" in value:
                    system, code = value.split("|", 1)
                    query["code.coding"] = {"$elemMatch": {"system": system, "code": code}}
                else:
                    query["code.coding.code"] = value
            else:
                # Default handling
                query[key] = value

        # Get pagination parameters
        count = int(params.get("_count", 100))
        page = int(params.get("_page", 1))
        skip = (page - 1) * count

        # Execute query
        cursor = db.db[resource_type].find(query).skip(skip).limit(count)

        # Convert to list and prepare for API response
        resources = []
        async for resource in cursor:
            resources.append(self._prepare_resource_from_db(resource))

        return resources

    # Specialized methods for common resource types

    async def get_patient(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get a patient by ID."""
        return await self.get_resource("Patient", patient_id)

    async def search_patients(self, params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Search for patients."""
        return await self.search_resources("Patient", params)

    async def get_patient_observations(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get observations for a patient."""
        search_params = params or {}
        search_params["subject"] = f"Patient/{patient_id}"
        return await self.search_resources("Observation", search_params)

    async def get_patient_conditions(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get conditions for a patient."""
        search_params = params or {}
        search_params["subject"] = f"Patient/{patient_id}"
        return await self.search_resources("Condition", search_params)

    async def get_patient_medications(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get medication requests for a patient."""
        search_params = params or {}
        search_params["subject"] = f"Patient/{patient_id}"
        return await self.search_resources("MedicationRequest", search_params)

    async def get_patient_diagnostic_reports(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get diagnostic reports for a patient."""
        search_params = params or {}
        search_params["subject"] = f"Patient/{patient_id}"
        return await self.search_resources("DiagnosticReport", search_params)

    async def get_patient_encounters(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get encounters for a patient."""
        search_params = params or {}
        search_params["subject"] = f"Patient/{patient_id}"
        return await self.search_resources("Encounter", search_params)

    async def get_patient_documents(self, patient_id: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """Get document references for a patient."""
        search_params = params or {}
        search_params["subject"] = f"Patient/{patient_id}"
        return await self.search_resources("DocumentReference", search_params)
