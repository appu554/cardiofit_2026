from fastapi import APIRouter, HTTPException, status, Request, Query, Path
from typing import List, Optional, Dict, Any
import logging

# Import shared auth decorators
import sys
import os

# Ensure shared module is importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

try:
    from shared.auth.decorators import require_permissions, require_role
except ImportError:
    # Fallback: create simple decorators if shared module not available
    def require_permissions(permissions):
        def decorator(func):
            return func
        return decorator

    def require_role(role):
        def decorator(func):
            return func
        return decorator

from app.models.organization import Organization, OrganizationType, OrganizationStatus
from app.models.organization_setting import OrganizationSetting, OrganizationSettingInput, OrganizationSettingUpdate
from app.models.organization_relationship import OrganizationRelationship, OrganizationRelationshipInput
from app.models.user_organization import UserOrganizationAccess, UserInvitation, InvitationInput
from app.services.organization_management_service import get_management_service
from app.api.dependencies import get_current_user_id

logger = logging.getLogger(__name__)

# Create router (no prefix since main app adds /api prefix)
router = APIRouter(tags=["organizations"])

# Organization CRUD endpoints
@router.post("/organizations", response_model=Organization)
@require_permissions(["organization:write"])
async def create_organization(
    request: Request,
    organization_data: Dict[str, Any]
) -> Organization:
    """
    Create a new organization.
    
    Requires: organization:write permission
    """
    try:
        current_user_id = get_current_user_id(request)
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Create organization
        organization = await management_service.create_organization(organization_data, current_user_id)
        
        if not organization:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Failed to create organization"
            )
        
        return organization
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error creating organization: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

@router.get("/organizations/{organization_id}", response_model=Organization)
@require_permissions(["organization:read"])
async def get_organization(
    request: Request,
    organization_id: str = Path(..., description="Organization ID")
) -> Organization:
    """
    Get an organization by ID.
    
    Requires: organization:read permission
    """
    try:
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Get organization
        organization = await management_service.get_organization(organization_id)
        
        if not organization:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Organization not found"
            )
        
        return organization
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting organization {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

@router.put("/organizations/{organization_id}", response_model=Organization)
@require_permissions(["organization:write"])
async def update_organization(
    request: Request,
    organization_id: str = Path(..., description="Organization ID"),
    update_data: Dict[str, Any] = None
) -> Organization:
    """
    Update an organization.
    
    Requires: organization:write permission
    """
    try:
        current_user_id = get_current_user_id(request)
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Update organization
        organization = await management_service.update_organization(
            organization_id, 
            update_data or {}, 
            current_user_id
        )
        
        if not organization:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Organization not found or update failed"
            )
        
        return organization
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating organization {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

@router.delete("/organizations/{organization_id}")
@require_permissions(["organization:delete"])
async def delete_organization(
    request: Request,
    organization_id: str = Path(..., description="Organization ID")
) -> Dict[str, str]:
    """
    Delete an organization.
    
    Requires: organization:delete permission
    """
    try:
        current_user_id = get_current_user_id(request)
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Delete organization
        success = await management_service.delete_organization(organization_id, current_user_id)
        
        if not success:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Organization not found or deletion failed"
            )
        
        return {"message": "Organization deleted successfully"}
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting organization {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

@router.get("/organizations", response_model=List[Organization])
@require_permissions(["organization:read"])
async def search_organizations(
    request: Request,
    name: Optional[str] = Query(None, description="Organization name filter"),
    type: Optional[OrganizationType] = Query(None, description="Organization type filter"),
    status: Optional[OrganizationStatus] = Query(None, description="Organization status filter"),
    active: Optional[bool] = Query(None, description="Active status filter")
) -> List[Organization]:
    """
    Search for organizations.
    
    Requires: organization:read permission
    """
    try:
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Build search parameters
        search_params = {}
        if name:
            search_params["name"] = name
        if type:
            search_params["type"] = type.value
        if status:
            search_params["status"] = status.value
        if active is not None:
            search_params["active"] = str(active).lower()
        
        # Search organizations
        organizations = await management_service.search_organizations(search_params)
        
        return organizations
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error searching organizations: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

# Verification endpoints
@router.post("/organizations/{organization_id}/verify")
@require_permissions(["organization:verify"])
async def submit_for_verification(
    request: Request,
    organization_id: str = Path(..., description="Organization ID"),
    documents: List[str] = []
) -> Dict[str, str]:
    """
    Submit an organization for verification.
    
    Requires: organization:verify permission
    """
    try:
        current_user_id = get_current_user_id(request)
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Submit for verification
        success = await management_service.submit_for_verification(
            organization_id, 
            documents, 
            current_user_id
        )
        
        if not success:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Failed to submit organization for verification"
            )
        
        return {"message": "Organization submitted for verification successfully"}
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error submitting organization for verification {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

@router.post("/organizations/{organization_id}/approve")
@require_permissions(["organization:approve"])
async def approve_organization(
    request: Request,
    organization_id: str = Path(..., description="Organization ID"),
    notes: Optional[str] = None
) -> Dict[str, str]:
    """
    Approve an organization verification.
    
    Requires: organization:approve permission
    """
    try:
        current_user_id = get_current_user_id(request)
        management_service = get_management_service()
        
        # Initialize service if needed
        await management_service.initialize()
        
        # Approve organization
        success = await management_service.approve_organization(
            organization_id, 
            current_user_id, 
            notes
        )
        
        if not success:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Failed to approve organization"
            )
        
        return {"message": "Organization approved successfully"}
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error approving organization {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

# State Management endpoints
@router.post("/organizations/{organization_id}/change-status")
@require_permissions(["organization:write"])
async def change_organization_status(
    request: Request,
    organization_id: str = Path(..., description="Organization ID"),
    new_status: str = Query(..., description="New organization status"),
    reason: Optional[str] = Query(None, description="Reason for status change")
) -> Dict[str, str]:
    """
    Change organization status with validation.

    Requires: organization:write permission (or higher for certain transitions)
    """
    try:
        from app.models.organization import OrganizationStatus
        from app.api.dependencies import get_user_permissions

        current_user_id = get_current_user_id(request)
        user_permissions = get_user_permissions(request)
        management_service = get_management_service()

        # Initialize service if needed
        await management_service.initialize()

        # Validate status value
        try:
            target_status = OrganizationStatus(new_status)
        except ValueError:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Invalid status: {new_status}"
            )

        # Change status
        success = await management_service.change_organization_status(
            organization_id,
            target_status,
            current_user_id,
            user_permissions,
            reason
        )

        if not success:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Failed to change organization status"
            )

        return {"message": f"Organization status changed to {new_status} successfully"}

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error changing organization status {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )

@router.get("/organizations/{organization_id}/valid-transitions")
@require_permissions(["organization:read"])
async def get_valid_status_transitions(
    request: Request,
    organization_id: str = Path(..., description="Organization ID")
) -> Dict[str, Any]:
    """
    Get valid status transitions for an organization.

    Requires: organization:read permission
    """
    try:
        from app.models.organization import OrganizationStatus

        management_service = get_management_service()

        # Initialize service if needed
        await management_service.initialize()

        # Get organization to check current status
        organization = await management_service.get_organization(organization_id)

        if not organization:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Organization not found"
            )

        # Get valid transitions
        current_status = organization.status or OrganizationStatus.PENDING_VERIFICATION
        valid_transitions = management_service.get_valid_status_transitions(current_status)

        return {
            "organization_id": organization_id,
            "current_status": current_status.value,
            "valid_transitions": valid_transitions
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting valid transitions for organization {organization_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error"
        )
