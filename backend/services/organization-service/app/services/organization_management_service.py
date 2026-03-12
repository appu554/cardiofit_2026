import logging
import uuid
from typing import Optional, List, Dict, Any
from datetime import datetime, timedelta
import secrets

from app.models.organization import Organization, OrganizationType, OrganizationStatus
from app.models.organization_setting import OrganizationSetting, OrganizationSettingInput, OrganizationSettingUpdate
from app.models.organization_relationship import OrganizationRelationship, OrganizationRelationshipInput
from app.models.user_organization import UserOrganizationAccess, UserInvitation, UserOrganizationInput, InvitationInput
from app.services.google_fhir_service import get_fhir_service
from app.services.state_management import state_manager, transition_logger, StateTransitionError

logger = logging.getLogger(__name__)

class OrganizationManagementService:
    """
    Business logic service for organization management.
    
    This service provides high-level operations for managing organizations,
    their settings, relationships, and user associations.
    """

    def __init__(self):
        """Initialize the organization management service."""
        self.fhir_service = get_fhir_service()
        # In a real implementation, you would also initialize:
        # - Database connections for settings, relationships, and user associations
        # - Notification service for sending invitations
        # - Audit logging service
        
    async def initialize(self) -> bool:
        """
        Initialize the service.

        Returns:
            bool: True if initialization was successful, False otherwise
        """
        return await self.fhir_service.initialize()

    # Organization CRUD Operations
    async def create_organization(self, organization_data: Dict[str, Any], created_by: str) -> Optional[Organization]:
        """
        Create a new organization.

        Args:
            organization_data: Organization data
            created_by: User ID who is creating the organization

        Returns:
            Created organization, or None if creation failed
        """
        try:
            # Set audit fields and initial state
            now = datetime.utcnow()
            initial_status = state_manager.get_initial_status()
            organization_data.update({
                "created_at": now,
                "updated_at": now,
                "created_by": created_by,
                "updated_by": created_by,
                "status": initial_status,
                "verification_status": "pending"
            })

            # Create organization model
            organization = Organization(**organization_data)
            
            # Create in FHIR store
            created_org = await self.fhir_service.create_organization(organization)
            
            if created_org:
                logger.info(f"Organization created successfully: {created_org.id}")
                
                # TODO: In a real implementation, you would:
                # 1. Create default settings for the organization
                # 2. Send notification to admins for verification
                # 3. Log the creation in audit trail
                
                return created_org
            else:
                logger.error("Failed to create organization in FHIR store")
                return None

        except Exception as e:
            logger.error(f"Error creating organization: {str(e)}")
            return None

    async def get_organization(self, organization_id: str) -> Optional[Organization]:
        """
        Get an organization by ID.

        Args:
            organization_id: Organization ID

        Returns:
            Organization if found, None otherwise
        """
        try:
            return await self.fhir_service.get_organization(organization_id)
        except Exception as e:
            logger.error(f"Error getting organization {organization_id}: {str(e)}")
            return None

    async def update_organization(self, organization_id: str, update_data: Dict[str, Any], updated_by: str) -> Optional[Organization]:
        """
        Update an organization.

        Args:
            organization_id: Organization ID to update
            update_data: Updated organization data
            updated_by: User ID who is updating the organization

        Returns:
            Updated organization, or None if update failed
        """
        try:
            # Get existing organization
            existing_org = await self.fhir_service.get_organization(organization_id)
            if not existing_org:
                logger.error(f"Organization not found: {organization_id}")
                return None

            # Update the organization data
            org_dict = existing_org.dict()
            org_dict.update(update_data)
            org_dict.update({
                "updated_at": datetime.utcnow(),
                "updated_by": updated_by
            })

            # Create updated organization model
            updated_org = Organization(**org_dict)
            
            # Update in FHIR store
            result = await self.fhir_service.update_organization(organization_id, updated_org)
            
            if result:
                logger.info(f"Organization updated successfully: {organization_id}")
                # TODO: Log the update in audit trail
                return result
            else:
                logger.error(f"Failed to update organization: {organization_id}")
                return None

        except Exception as e:
            logger.error(f"Error updating organization {organization_id}: {str(e)}")
            return None

    async def delete_organization(self, organization_id: str, deleted_by: str) -> bool:
        """
        Delete an organization.

        Args:
            organization_id: Organization ID to delete
            deleted_by: User ID who is deleting the organization

        Returns:
            True if deletion was successful, False otherwise
        """
        try:
            # TODO: In a real implementation, you would:
            # 1. Check if organization has dependent resources
            # 2. Handle cascade deletion or prevent deletion
            # 3. Archive related data instead of hard delete
            # 4. Log the deletion in audit trail
            
            success = await self.fhir_service.delete_organization(organization_id)
            
            if success:
                logger.info(f"Organization deleted successfully: {organization_id}")
            else:
                logger.error(f"Failed to delete organization: {organization_id}")
            
            return success

        except Exception as e:
            logger.error(f"Error deleting organization {organization_id}: {str(e)}")
            return False

    async def search_organizations(self, search_params: Optional[Dict[str, str]] = None) -> List[Organization]:
        """
        Search for organizations.

        Args:
            search_params: Optional search parameters

        Returns:
            List of organizations matching the search criteria
        """
        try:
            return await self.fhir_service.search_organizations(search_params)
        except Exception as e:
            logger.error(f"Error searching organizations: {str(e)}")
            return []

    # Verification Operations
    async def submit_for_verification(self, organization_id: str, documents: List[str], submitted_by: str) -> bool:
        """
        Submit an organization for verification.

        Args:
            organization_id: Organization ID
            documents: List of document URLs
            submitted_by: User ID who submitted for verification

        Returns:
            True if submission was successful, False otherwise
        """
        try:
            # Get existing organization
            organization = await self.fhir_service.get_organization(organization_id)
            if not organization:
                logger.error(f"Organization not found: {organization_id}")
                return False

            # Update verification status
            update_data = {
                "verification_status": "submitted",
                "verification_documents": documents,
                "status": OrganizationStatus.PENDING_VERIFICATION
            }

            updated_org = await self.update_organization(organization_id, update_data, submitted_by)
            
            if updated_org:
                logger.info(f"Organization submitted for verification: {organization_id}")
                # TODO: Send notification to verification team
                return True
            else:
                logger.error(f"Failed to submit organization for verification: {organization_id}")
                return False

        except Exception as e:
            logger.error(f"Error submitting organization for verification {organization_id}: {str(e)}")
            return False

    async def approve_organization(self, organization_id: str, approved_by: str, notes: Optional[str] = None) -> bool:
        """
        Approve an organization verification.

        Args:
            organization_id: Organization ID
            approved_by: User ID who approved the organization
            notes: Optional approval notes

        Returns:
            True if approval was successful, False otherwise
        """
        try:
            # Update verification status
            update_data = {
                "verification_status": "approved",
                "verified_by": approved_by,
                "verification_timestamp": datetime.utcnow(),
                "status": OrganizationStatus.VERIFIED
            }

            updated_org = await self.update_organization(organization_id, update_data, approved_by)
            
            if updated_org:
                logger.info(f"Organization approved: {organization_id}")
                # TODO: Send notification to organization contacts
                return True
            else:
                logger.error(f"Failed to approve organization: {organization_id}")
                return False

        except Exception as e:
            logger.error(f"Error approving organization {organization_id}: {str(e)}")
            return False

    # State Management Operations
    async def change_organization_status(self, organization_id: str, new_status: OrganizationStatus,
                                       user_id: str, user_permissions: List[str],
                                       reason: Optional[str] = None) -> bool:
        """
        Change organization status with validation and state management.

        Args:
            organization_id: Organization ID
            new_status: Target status
            user_id: User making the change
            user_permissions: User's permissions
            reason: Optional reason for the change

        Returns:
            True if status change was successful, False otherwise
        """
        try:
            # Get current organization
            organization = await self.fhir_service.get_organization(organization_id)
            if not organization:
                logger.error(f"Organization not found: {organization_id}")
                return False

            current_status = organization.status
            if not current_status:
                current_status = state_manager.get_initial_status()

            # Validate state transition
            is_valid, error_message = state_manager.validate_transition(
                current_status, new_status, user_permissions
            )

            if not is_valid:
                transition_logger.log_transition_error(
                    organization_id, current_status, new_status, user_id, error_message
                )
                logger.error(f"Invalid state transition: {error_message}")
                return False

            # Perform the status change
            update_data = {
                "status": new_status,
                "updated_at": datetime.utcnow(),
                "updated_by": user_id
            }

            # Add status-specific updates
            if new_status == OrganizationStatus.VERIFIED:
                update_data["verification_status"] = "approved"
                update_data["verification_timestamp"] = datetime.utcnow()
                update_data["verified_by"] = user_id
            elif new_status == OrganizationStatus.ACTIVE:
                update_data["verification_status"] = "approved"
            elif new_status == OrganizationStatus.SUSPENDED:
                update_data["suspension_reason"] = reason
                update_data["suspended_by"] = user_id
                update_data["suspension_timestamp"] = datetime.utcnow()

            updated_org = await self.update_organization(organization_id, update_data, user_id)

            if updated_org:
                # Log successful transition
                transition_logger.log_transition(
                    organization_id, current_status, new_status, user_id, reason
                )
                logger.info(f"Organization status changed: {organization_id} from {current_status.value} to {new_status.value}")
                return True
            else:
                logger.error(f"Failed to update organization status: {organization_id}")
                return False

        except Exception as e:
            logger.error(f"Error changing organization status {organization_id}: {str(e)}")
            return False

    def get_valid_status_transitions(self, current_status: OrganizationStatus) -> List[Dict[str, Any]]:
        """
        Get valid status transitions for an organization.

        Args:
            current_status: Current organization status

        Returns:
            List of valid transitions with descriptions
        """
        valid_transitions = state_manager.get_valid_transitions(current_status)

        return [
            {
                "status": status.value,
                "description": state_manager.get_state_description(status),
                "required_permissions": state_manager.get_required_permissions(current_status, status)
            }
            for status in valid_transitions
        ]

    def can_organization_operate(self, status: OrganizationStatus) -> bool:
        """
        Check if organization can perform operational activities.

        Args:
            status: Organization status

        Returns:
            True if organization can operate
        """
        return state_manager.is_operational_status(status)

    def can_delete_organization(self, status: OrganizationStatus) -> bool:
        """
        Check if organization can be deleted in its current status.

        Args:
            status: Organization status

        Returns:
            True if organization can be deleted
        """
        return state_manager.can_be_deleted(status)

# Global service instance
_management_service = None

def get_management_service() -> OrganizationManagementService:
    """Get the global organization management service instance."""
    global _management_service
    if _management_service is None:
        _management_service = OrganizationManagementService()
    return _management_service
