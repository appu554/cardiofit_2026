"""
User Management Service

This service manages healthcare professionals (doctors, nurses, etc.) 
and their association with organizations in the FHIR store.
"""

import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
import uuid
from dateutil import parser

from app.services.google_fhir_service import GoogleOrganizationFHIRService

logger = logging.getLogger(__name__)

class UserManagementService:
    """
    Service for managing healthcare professionals and their association with organizations.
    """
    
    def __init__(self):
        self.fhir_service = GoogleOrganizationFHIRService()
        
    async def initialize(self):
        """Initialize the FHIR service."""
        await self.fhir_service.initialize()
    
    async def create_user(self, user_data: Dict[str, Any], created_by: str) -> Optional[Dict[str, Any]]:
        """
        Create a new healthcare professional user.
        
        Args:
            user_data: User data to create
            created_by: ID of the user creating this user
            
        Returns:
            Created user data if successful, None otherwise
        """
        try:
            # Generate user ID
            user_id = str(uuid.uuid4())
            
            # Create FHIR Practitioner resource
            practitioner_data = {
                "resourceType": "Practitioner",
                "id": user_id,
                "active": user_data.get("is_active", True),
                "name": [
                    {
                        "use": "official",
                        "family": user_data.get("last_name"),
                        "given": [user_data.get("first_name")]
                    }
                ],
                "telecom": [],
                "qualification": []
            }
            
            # Add email
            if user_data.get("email"):
                practitioner_data["telecom"].append({
                    "system": "email",
                    "value": user_data.get("email"),
                    "use": "work"
                })
            
            # Add phone
            if user_data.get("phone_number"):
                practitioner_data["telecom"].append({
                    "system": "phone",
                    "value": user_data.get("phone_number"),
                    "use": "work"
                })
            
            # Add identifiers
            if user_data.get("identifier"):
                practitioner_data["identifier"] = []
                for ident in user_data["identifier"]:
                    fhir_identifier = {
                        "use": ident.get("use", "usual"),
                        "system": ident.get("system"),
                        "value": ident.get("value")
                    }
                    if ident.get("type_code"):
                        fhir_identifier["type"] = {
                            "coding": [{
                                "code": ident.get("type_code"),
                                "display": ident.get("type_display", ident.get("type_code"))
                            }]
                        }
                    practitioner_data["identifier"].append(fhir_identifier)

            # Add license information
            if user_data.get("license_number"):
                if "identifier" not in practitioner_data:
                    practitioner_data["identifier"] = []

                practitioner_data["identifier"].append({
                    "use": "official",
                    "system": "http://hl7.org/fhir/sid/us-npi",
                    "value": user_data.get("license_number")
                })

                practitioner_data["qualification"].append({
                    "code": {
                        "coding": [
                            {
                                "system": "http://terminology.hl7.org/CodeSystem/v2-0360",
                                "code": user_data.get("role", "").upper(),
                                "display": user_data.get("role", "").title()
                            }
                        ]
                    }
                })
            
            # Add specialization
            if user_data.get("specialization"):
                practitioner_data["qualification"].append({
                    "code": {
                        "text": user_data.get("specialization")
                    }
                })
            
            # Add custom extensions for our specific fields
            # Format datetime properly for FHIR (without microseconds)
            current_time = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")

            practitioner_data["extension"] = [
                {
                    "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/user-role",
                    "valueString": user_data.get("role")
                },
                {
                    "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-id",
                    "valueString": user_data.get("organization_id")
                },
                {
                    "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/department",
                    "valueString": user_data.get("department")
                },
                {
                    "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/created-by",
                    "valueString": created_by
                },
                {
                    "url": "http://clinical-synthesis-hub.com/fhir/StructureDefinition/created-at",
                    "valueDateTime": current_time
                }
            ]
            
            # Create the practitioner in FHIR store
            result = await self.fhir_service.create_resource("Practitioner", practitioner_data)
            
            if result:
                logger.info(f"User created successfully: {user_id}")
                return self._convert_fhir_to_user_model(result)
            
            return None
            
        except Exception as e:
            logger.error(f"Error creating user: {str(e)}")
            return None
    
    async def get_user(self, user_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a user by ID.
        
        Args:
            user_id: User ID to retrieve
            
        Returns:
            User data if found, None otherwise
        """
        try:
            result = await self.fhir_service.get_resource("Practitioner", user_id)
            
            if result:
                return self._convert_fhir_to_user_model(result)
            
            return None
            
        except Exception as e:
            logger.error(f"Error getting user {user_id}: {str(e)}")
            return None
    
    async def search_users(self, search_params: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Search for users based on parameters.
        
        Args:
            search_params: Search parameters
            
        Returns:
            List of matching users
        """
        try:
            # Build FHIR search parameters
            fhir_params = {}
            
            if search_params.get("organization_id"):
                # This would require a custom search parameter in the FHIR store
                # For now, we'll get all practitioners and filter
                pass
            
            if search_params.get("role"):
                # This would also require custom search parameters
                pass
            
            # Get all practitioners (in a real implementation, you'd want pagination)
            results = await self.fhir_service.search_resources("Practitioner", fhir_params)
            
            # Filter results based on our custom criteria
            filtered_results = []
            logger.info(f"Processing {len(results)} Practitioner resources with search params: {search_params}")

            for result in results:
                user_model = self._convert_fhir_to_user_model(result)
                logger.info(f"Processing user: ID={user_model.get('id')}, role={user_model.get('role')}, org_id={user_model.get('organization_id')}")

                # Apply filters
                if search_params.get("organization_id"):
                    if user_model.get("organization_id") != search_params["organization_id"]:
                        logger.info(f"Filtered out user {user_model.get('id')} - org_id mismatch: {user_model.get('organization_id')} != {search_params['organization_id']}")
                        continue

                if search_params.get("role"):
                    if user_model.get("role") != search_params["role"]:
                        logger.info(f"Filtered out user {user_model.get('id')} - role mismatch: {user_model.get('role')} != {search_params['role']}")
                        continue

                if search_params.get("active") is not None:
                    if user_model.get("is_active") != search_params["active"]:
                        logger.info(f"Filtered out user {user_model.get('id')} - active status mismatch: {user_model.get('is_active')} != {search_params['active']}")
                        continue

                # Filter by Supabase ID
                if search_params.get("supabase_id"):
                    supabase_id_found = False
                    if user_model.get("identifier"):
                        for ident in user_model["identifier"]:
                            if (ident.get("system") == "https://auugxeqzgrnknklgwqrh.supabase.co/supabase-users" and
                                ident.get("value") == search_params["supabase_id"]):
                                supabase_id_found = True
                                break
                    if not supabase_id_found:
                        continue

                logger.info(f"User {user_model.get('id')} passed all filters")
                filtered_results.append(user_model)

            logger.info(f"Found {len(filtered_results)} users matching search criteria")
            for user in filtered_results:
                logger.info(f"Final result: ID={user.get('id')}, role={user.get('role')}, email={user.get('email')}")
            return filtered_results
            
        except Exception as e:
            logger.error(f"Error searching users: {str(e)}")
            return []
    
    async def update_user(self, user_id: str, update_data: Dict[str, Any], updated_by: str) -> Optional[Dict[str, Any]]:
        """
        Update a user.
        
        Args:
            user_id: User ID to update
            update_data: Data to update
            updated_by: ID of the user making the update
            
        Returns:
            Updated user data if successful, None otherwise
        """
        try:
            # Get existing user
            existing_user = await self.fhir_service.get_resource("Practitioner", user_id)
            if not existing_user:
                logger.error(f"User {user_id} not found")
                return None
            
            # Update the practitioner data
            # This is a simplified update - in a real implementation, you'd want to merge changes properly
            if update_data.get("first_name") or update_data.get("last_name"):
                existing_user["name"] = [
                    {
                        "use": "official",
                        "family": update_data.get("last_name", existing_user["name"][0].get("family")),
                        "given": [update_data.get("first_name", existing_user["name"][0]["given"][0])]
                    }
                ]
            
            # Update extensions
            if not existing_user.get("extension"):
                existing_user["extension"] = []
            
            # Update or add extensions
            # Format datetime properly for FHIR (without microseconds)
            current_time = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")

            extension_updates = {
                "http://clinical-synthesis-hub.com/fhir/StructureDefinition/user-role": update_data.get("role"),
                "http://clinical-synthesis-hub.com/fhir/StructureDefinition/organization-id": update_data.get("organization_id"),
                "http://clinical-synthesis-hub.com/fhir/StructureDefinition/department": update_data.get("department"),
                "http://clinical-synthesis-hub.com/fhir/StructureDefinition/updated-by": updated_by,
                "http://clinical-synthesis-hub.com/fhir/StructureDefinition/updated-at": current_time
            }
            
            for url, value in extension_updates.items():
                if value is not None:
                    # Find existing extension or add new one
                    found = False
                    for ext in existing_user["extension"]:
                        if ext.get("url") == url:
                            ext["valueString"] = str(value)
                            found = True
                            break
                    
                    if not found:
                        existing_user["extension"].append({
                            "url": url,
                            "valueString": str(value)
                        })
            
            # Update active status
            if update_data.get("is_active") is not None:
                existing_user["active"] = update_data["is_active"]
            
            # Update the resource in FHIR store
            result = await self.fhir_service.update_resource("Practitioner", user_id, existing_user)
            
            if result:
                logger.info(f"User updated successfully: {user_id}")
                return self._convert_fhir_to_user_model(result)
            
            return None
            
        except Exception as e:
            logger.error(f"Error updating user {user_id}: {str(e)}")
            return None
    
    async def deactivate_user(self, user_id: str, deactivated_by: str) -> bool:
        """
        Deactivate a user.
        
        Args:
            user_id: User ID to deactivate
            deactivated_by: ID of the user making the deactivation
            
        Returns:
            True if successful, False otherwise
        """
        try:
            update_data = {
                "is_active": False
            }
            
            result = await self.update_user(user_id, update_data, deactivated_by)
            return result is not None
            
        except Exception as e:
            logger.error(f"Error deactivating user {user_id}: {str(e)}")
            return False
    
    def _convert_fhir_to_user_model(self, fhir_practitioner: Dict[str, Any]) -> Dict[str, Any]:
        """
        Convert FHIR Practitioner resource to our user model.
        
        Args:
            fhir_practitioner: FHIR Practitioner resource
            
        Returns:
            User model dictionary
        """
        try:
            # Extract basic information
            user_model = {
                "id": fhir_practitioner.get("id"),
                "is_active": fhir_practitioner.get("active", True)
            }
            
            # Extract name
            if fhir_practitioner.get("name"):
                name = fhir_practitioner["name"][0]
                user_model["first_name"] = name.get("given", [""])[0]
                user_model["last_name"] = name.get("family", "")
            
            # Extract telecom
            if fhir_practitioner.get("telecom"):
                for telecom in fhir_practitioner["telecom"]:
                    if telecom.get("system") == "email":
                        user_model["email"] = telecom.get("value")
                    elif telecom.get("system") == "phone":
                        user_model["phone_number"] = telecom.get("value")
            
            # Extract identifiers
            if fhir_practitioner.get("identifier"):
                user_model["identifier"] = []
                for fhir_identifier in fhir_practitioner["identifier"]:
                    identifier = {
                        "use": fhir_identifier.get("use"),
                        "system": fhir_identifier.get("system"),
                        "value": fhir_identifier.get("value")
                    }
                    if fhir_identifier.get("type", {}).get("coding"):
                        coding = fhir_identifier["type"]["coding"][0]
                        identifier["type_code"] = coding.get("code")
                        identifier["type_display"] = coding.get("display")

                    user_model["identifier"].append(identifier)

                    # Extract license number from NPI identifier
                    if fhir_identifier.get("system") == "http://hl7.org/fhir/sid/us-npi":
                        user_model["license_number"] = fhir_identifier.get("value")

            # Extract additional info from qualification
            if fhir_practitioner.get("qualification"):
                for qual in fhir_practitioner["qualification"]:
                    if qual.get("code", {}).get("text"):
                        user_model["specialization"] = qual["code"]["text"]
            
            # Extract custom extensions
            if fhir_practitioner.get("extension"):
                for ext in fhir_practitioner["extension"]:
                    url = ext.get("url", "")

                    if "user-role" in url:
                        user_model["role"] = ext.get("valueString")
                    elif "organization-id" in url:
                        user_model["organization_id"] = ext.get("valueString")
                    elif "department" in url:
                        user_model["department"] = ext.get("valueString")
                    elif "created-at" in url:
                        datetime_str = ext.get("valueDateTime")
                        if datetime_str:
                            try:
                                user_model["created_at"] = parser.parse(datetime_str)
                            except Exception as e:
                                logger.warning(f"Failed to parse created_at datetime: {datetime_str}, error: {e}")
                                user_model["created_at"] = None
                    elif "updated-at" in url:
                        datetime_str = ext.get("valueDateTime")
                        if datetime_str:
                            try:
                                user_model["updated_at"] = parser.parse(datetime_str)
                            except Exception as e:
                                logger.warning(f"Failed to parse updated_at datetime: {datetime_str}, error: {e}")
                                user_model["updated_at"] = None
            
            return user_model
            
        except Exception as e:
            logger.error(f"Error converting FHIR practitioner to user model: {str(e)}")
            return {}

# Global instance
_user_management_service = None

def get_user_management_service() -> UserManagementService:
    """Get the global user management service instance."""
    global _user_management_service
    if _user_management_service is None:
        _user_management_service = UserManagementService()
    return _user_management_service
