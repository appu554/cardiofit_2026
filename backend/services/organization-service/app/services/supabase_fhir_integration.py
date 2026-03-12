"""
Supabase FHIR Integration Service

This service connects Supabase authentication with Google Healthcare API FHIR operations.
It allows authenticated doctors to perform FHIR operations based on their roles and permissions.
"""

import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
import json

from app.services.google_fhir_service import GoogleFHIRService
from shared.auth.middleware import get_current_user_from_token

logger = logging.getLogger(__name__)

class SupabaseFHIRIntegration:
    """
    Integration service that connects Supabase authentication with Google Healthcare API FHIR operations.
    """
    
    def __init__(self):
        self.fhir_service = GoogleFHIRService()
        
    async def initialize(self):
        """Initialize the FHIR service."""
        await self.fhir_service.initialize()
    
    async def create_organization_with_auth(self, organization_data: Dict[str, Any], auth_token: str) -> Optional[Dict[str, Any]]:
        """
        Create an organization in FHIR store with Supabase authentication.
        
        Args:
            organization_data: Organization data to create
            auth_token: Supabase JWT token
            
        Returns:
            Created organization data if successful, None otherwise
        """
        try:
            # Verify user authentication and get user info
            user_info = await get_current_user_from_token(auth_token)
            
            if not user_info:
                logger.error("Failed to authenticate user")
                return None
            
            # Check if user has permission to create organizations
            if not self._has_permission(user_info, "organization:write"):
                logger.error(f"User {user_info.get('sub')} does not have permission to create organizations")
                return None
            
            # Add audit information
            organization_data["created_by"] = user_info.get("sub")
            organization_data["created_at"] = datetime.utcnow().isoformat()
            
            # Add user's organization context if available
            user_org_id = self._get_user_organization(user_info)
            if user_org_id and not organization_data.get("part_of"):
                organization_data["part_of"] = user_org_id
            
            # Create organization in FHIR store
            result = await self.fhir_service.create_organization(organization_data)
            
            if result:
                logger.info(f"Organization created successfully by user {user_info.get('sub')}: {result.get('id')}")
                
                # Log the action for audit trail
                await self._log_fhir_action(
                    user_info=user_info,
                    action="CREATE_ORGANIZATION",
                    resource_type="Organization",
                    resource_id=result.get("id"),
                    details={"organization_name": organization_data.get("name")}
                )
            
            return result
            
        except Exception as e:
            logger.error(f"Error creating organization with auth: {str(e)}")
            return None
    
    async def get_organization_with_auth(self, organization_id: str, auth_token: str) -> Optional[Dict[str, Any]]:
        """
        Get an organization from FHIR store with Supabase authentication.
        
        Args:
            organization_id: Organization ID to retrieve
            auth_token: Supabase JWT token
            
        Returns:
            Organization data if successful and authorized, None otherwise
        """
        try:
            # Verify user authentication
            user_info = await get_current_user_from_token(auth_token)
            
            if not user_info:
                logger.error("Failed to authenticate user")
                return None
            
            # Check if user has permission to read organizations
            if not self._has_permission(user_info, "organization:read"):
                logger.error(f"User {user_info.get('sub')} does not have permission to read organizations")
                return None
            
            # Get organization from FHIR store
            result = await self.fhir_service.get_organization(organization_id)
            
            if result:
                # Check if user has access to this specific organization
                if not self._has_organization_access(user_info, result):
                    logger.error(f"User {user_info.get('sub')} does not have access to organization {organization_id}")
                    return None
                
                logger.info(f"Organization retrieved successfully by user {user_info.get('sub')}: {organization_id}")
                
                # Log the action for audit trail
                await self._log_fhir_action(
                    user_info=user_info,
                    action="READ_ORGANIZATION",
                    resource_type="Organization",
                    resource_id=organization_id
                )
            
            return result
            
        except Exception as e:
            logger.error(f"Error getting organization with auth: {str(e)}")
            return None
    
    async def search_organizations_with_auth(self, search_params: Dict[str, Any], auth_token: str) -> List[Dict[str, Any]]:
        """
        Search organizations in FHIR store with Supabase authentication.
        
        Args:
            search_params: Search parameters
            auth_token: Supabase JWT token
            
        Returns:
            List of organizations the user has access to
        """
        try:
            # Verify user authentication
            user_info = await get_current_user_from_token(auth_token)
            
            if not user_info:
                logger.error("Failed to authenticate user")
                return []
            
            # Check if user has permission to read organizations
            if not self._has_permission(user_info, "organization:read"):
                logger.error(f"User {user_info.get('sub')} does not have permission to read organizations")
                return []
            
            # Add user's organization context to search if they're not an admin
            if not self._is_admin(user_info):
                user_org_id = self._get_user_organization(user_info)
                if user_org_id:
                    search_params["part_of"] = user_org_id
            
            # Search organizations in FHIR store
            results = await self.fhir_service.search_organizations(search_params)
            
            # Filter results based on user access
            accessible_results = []
            for org in results:
                if self._has_organization_access(user_info, org):
                    accessible_results.append(org)
            
            logger.info(f"Found {len(accessible_results)} accessible organizations for user {user_info.get('sub')}")
            
            # Log the action for audit trail
            await self._log_fhir_action(
                user_info=user_info,
                action="SEARCH_ORGANIZATIONS",
                resource_type="Organization",
                details={"search_params": search_params, "results_count": len(accessible_results)}
            )
            
            return accessible_results
            
        except Exception as e:
            logger.error(f"Error searching organizations with auth: {str(e)}")
            return []
    
    def _has_permission(self, user_info: Dict[str, Any], permission: str) -> bool:
        """
        Check if user has a specific permission.
        
        Args:
            user_info: User information from Supabase token
            permission: Permission to check (e.g., "organization:read")
            
        Returns:
            True if user has permission, False otherwise
        """
        try:
            # Check app_metadata for permissions
            app_metadata = user_info.get("app_metadata", {})
            permissions = app_metadata.get("permissions", [])
            
            # Check if user has the specific permission
            if permission in permissions:
                return True
            
            # Check for wildcard permissions
            permission_parts = permission.split(":")
            if len(permission_parts) == 2:
                resource, action = permission_parts
                wildcard_permission = f"{resource}:*"
                if wildcard_permission in permissions:
                    return True
            
            # Check roles for implicit permissions
            roles = app_metadata.get("roles", [])
            
            # Admins have all permissions
            if "admin" in roles:
                return True
            
            # Doctors have organization read/write permissions
            if "doctor" in roles and permission.startswith("organization:"):
                return True
            
            return False
            
        except Exception as e:
            logger.error(f"Error checking permission {permission}: {str(e)}")
            return False
    
    def _is_admin(self, user_info: Dict[str, Any]) -> bool:
        """Check if user is an admin."""
        try:
            app_metadata = user_info.get("app_metadata", {})
            roles = app_metadata.get("roles", [])
            return "admin" in roles
        except:
            return False
    
    def _get_user_organization(self, user_info: Dict[str, Any]) -> Optional[str]:
        """Get the user's organization ID from their profile."""
        try:
            app_metadata = user_info.get("app_metadata", {})
            return app_metadata.get("organization_id")
        except:
            return None
    
    def _has_organization_access(self, user_info: Dict[str, Any], organization: Dict[str, Any]) -> bool:
        """
        Check if user has access to a specific organization.
        
        Args:
            user_info: User information from Supabase token
            organization: Organization data
            
        Returns:
            True if user has access, False otherwise
        """
        try:
            # Admins have access to all organizations
            if self._is_admin(user_info):
                return True
            
            # Users have access to their own organization
            user_org_id = self._get_user_organization(user_info)
            org_id = organization.get("id")
            
            if user_org_id and org_id == user_org_id:
                return True
            
            # Users have access to child organizations
            part_of = organization.get("part_of")
            if user_org_id and part_of == user_org_id:
                return True
            
            return False
            
        except Exception as e:
            logger.error(f"Error checking organization access: {str(e)}")
            return False
    
    async def _log_fhir_action(self, user_info: Dict[str, Any], action: str, resource_type: str, 
                              resource_id: Optional[str] = None, details: Optional[Dict[str, Any]] = None):
        """
        Log FHIR actions for audit trail.
        
        Args:
            user_info: User information
            action: Action performed
            resource_type: FHIR resource type
            resource_id: Resource ID (if applicable)
            details: Additional details
        """
        try:
            log_entry = {
                "timestamp": datetime.utcnow().isoformat(),
                "user_id": user_info.get("sub"),
                "user_email": user_info.get("email"),
                "action": action,
                "resource_type": resource_type,
                "resource_id": resource_id,
                "details": details or {}
            }
            
            # Log to application logs
            logger.info(f"FHIR Audit: {json.dumps(log_entry)}")
            
            # TODO: Store in audit database or send to audit service
            
        except Exception as e:
            logger.error(f"Error logging FHIR action: {str(e)}")

# Global instance
_supabase_fhir_integration = None

def get_supabase_fhir_integration() -> SupabaseFHIRIntegration:
    """Get the global Supabase FHIR integration instance."""
    global _supabase_fhir_integration
    if _supabase_fhir_integration is None:
        _supabase_fhir_integration = SupabaseFHIRIntegration()
    return _supabase_fhir_integration
