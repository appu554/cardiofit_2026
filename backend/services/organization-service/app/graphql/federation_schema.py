import strawberry
from typing import List, Optional
import logging

from app.graphql.types import (
    Organization,
    OrganizationInput,
    OrganizationUpdateInput,
    OrganizationSearchResult,
    OrganizationType,
    OrganizationStatus,
    User,
    UserInput,
    UserUpdateInput,
    UserSearchResult,
    UserRole
)
from app.graphql.resolvers import (
    get_organization_resolver,
    search_organizations_resolver,
    create_organization_resolver,
    update_organization_resolver,
    get_user_resolver,
    search_users_resolver,
    create_user_resolver,
    update_user_resolver,
    deactivate_user_resolver,
    get_user_by_supabase_id_resolver
)

logger = logging.getLogger(__name__)

@strawberry.type
class Query:
    """Root query type for the Organization Service."""
    
    @strawberry.field
    async def organization(self, id: strawberry.ID) -> Optional[Organization]:
        """Get an organization by ID."""
        return await get_organization_resolver(str(id), None)
    
    @strawberry.field
    async def organizations(
        self,
        name: Optional[str] = None,
        organization_type: Optional[OrganizationType] = None,
        status: Optional[OrganizationStatus] = None,
        active: Optional[bool] = None
    ) -> OrganizationSearchResult:
        """Search for organizations."""
        type_value = organization_type.value if organization_type else None
        status_value = status.value if status else None
        
        return await search_organizations_resolver(
            name=name,
            organization_type=type_value,
            status=status_value,
            active=active,
            info=None
        )

    @strawberry.field
    async def user(self, id: strawberry.ID) -> Optional[User]:
        """Get a user by ID."""
        return await get_user_resolver(str(id), None)

    @strawberry.field
    async def users(
        self,
        organization_id: Optional[strawberry.ID] = None,
        role: Optional[UserRole] = None,
        active: Optional[bool] = None
    ) -> UserSearchResult:
        """Search for users."""
        role_value = role.value if role else None
        org_id = str(organization_id) if organization_id else None

        return await search_users_resolver(
            organization_id=org_id,
            role=role_value,
            active=active,
            info=None
        )

    @strawberry.field
    async def user_by_supabase_id(self, supabase_id: str) -> Optional[User]:
        """Find a user by their Supabase user ID."""
        return await get_user_by_supabase_id_resolver(supabase_id, None)

@strawberry.type
class Mutation:
    """Root mutation type for the Organization Service."""
    
    @strawberry.field
    async def create_organization(self, organization_data: OrganizationInput) -> Optional[Organization]:
        """Create a new organization."""
        return await create_organization_resolver(organization_data, None)
    
    @strawberry.field
    async def update_organization(
        self, 
        id: strawberry.ID, 
        update_data: OrganizationUpdateInput
    ) -> Optional[Organization]:
        """Update an existing organization."""
        return await update_organization_resolver(str(id), update_data, None)
    
    @strawberry.field
    async def submit_organization_for_verification(
        self, 
        id: strawberry.ID, 
        documents: Optional[List[str]] = None
    ) -> bool:
        """Submit an organization for verification."""
        try:
            from app.services.organization_management_service import get_management_service
            
            management_service = get_management_service()
            await management_service.initialize()
            
            # TODO: Extract user ID from GraphQL context
            current_user_id = "system"
            
            success = await management_service.submit_for_verification(
                str(id), 
                documents or [], 
                current_user_id
            )
            
            return success
            
        except Exception as e:
            logger.error(f"Error submitting organization for verification: {str(e)}")
            return False
    
    @strawberry.field
    async def approve_organization(
        self, 
        id: strawberry.ID, 
        notes: Optional[str] = None
    ) -> bool:
        """Approve an organization verification."""
        try:
            from app.services.organization_management_service import get_management_service
            
            management_service = get_management_service()
            await management_service.initialize()
            
            # TODO: Extract user ID from GraphQL context
            current_user_id = "system"
            
            success = await management_service.approve_organization(
                str(id), 
                current_user_id, 
                notes
            )
            
            return success
            
        except Exception as e:
            logger.error(f"Error approving organization: {str(e)}")
            return False

    @strawberry.field
    async def create_user(self, user_data: UserInput) -> Optional[User]:
        """Create a new user."""
        return await create_user_resolver(user_data, None)

    @strawberry.field
    async def update_user(
        self,
        id: strawberry.ID,
        update_data: UserUpdateInput
    ) -> Optional[User]:
        """Update an existing user."""
        return await update_user_resolver(str(id), update_data, None)

    @strawberry.field
    async def deactivate_user(self, id: strawberry.ID) -> bool:
        """Deactivate a user."""
        return await deactivate_user_resolver(str(id), None)

# Create the federation schema
schema = strawberry.federation.Schema(
    query=Query,
    mutation=Mutation,
    enable_federation_2=True
)
