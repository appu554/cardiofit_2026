import logging
from typing import List, Optional, Dict, Any
from strawberry.types import Info

from app.models.organization import Organization as OrganizationModel
from app.services.organization_management_service import get_management_service
from app.services.user_management_service import get_user_management_service
from app.graphql.types import (
    Organization, OrganizationInput, OrganizationUpdateInput, OrganizationSearchResult,
    User, UserInput, UserUpdateInput, UserSearchResult,
    ContactPoint, Address, Identifier
)

logger = logging.getLogger(__name__)

def convert_model_to_graphql(org_model: OrganizationModel) -> Organization:
    """
    Convert Organization model to GraphQL Organization type.
    
    Args:
        org_model: Organization model instance
        
    Returns:
        Organization GraphQL type
    """
    # Convert contact points
    telecom = None
    if org_model.telecom:
        telecom = [
            ContactPoint(
                system=cp.system,
                value=cp.value,
                use=cp.use,
                rank=cp.rank,
                period=None
            )
            for cp in org_model.telecom
        ]
    
    # Convert addresses
    address = None
    if org_model.address:
        address = [
            Address(
                use=addr.use,
                type=addr.type,
                text=addr.text,
                line=addr.line,
                city=addr.city,
                district=addr.district,
                state=addr.state,
                postal_code=addr.postal_code,
                country=addr.country,
                period=None
            )
            for addr in org_model.address
        ]
    
    # Convert identifiers
    identifier = None
    if org_model.identifier:
        identifier = [
            Identifier(
                use=ident.use,
                type=None,  # TODO: Convert type properly
                system=ident.system,
                value=ident.value,
                period=None,
                assigner=ident.assigner
            )
            for ident in org_model.identifier
        ]
    
    return Organization(
        id=org_model.id,
        resource_type=org_model.resource_type,
        active=org_model.active,
        identifier=identifier,
        name=org_model.name,
        alias=org_model.alias,
        legal_name=org_model.legal_name,
        trading_name=org_model.trading_name,
        organization_type=org_model.organization_type,
        status=org_model.status,
        telecom=telecom,
        address=address,
        website_url=org_model.website_url,
        tax_id=org_model.tax_id,
        license_number=org_model.license_number,
        part_of=org_model.part_of,
        verification_status=org_model.verification_status,
        verification_documents=org_model.verification_documents,
        verified_by=org_model.verified_by,
        verification_timestamp=org_model.verification_timestamp,
        created_at=org_model.created_at,
        updated_at=org_model.updated_at,
        created_by=org_model.created_by,
        updated_by=org_model.updated_by
    )

def convert_input_to_dict(org_input: OrganizationInput) -> Dict[str, Any]:
    """
    Convert OrganizationInput to dictionary for service layer.
    
    Args:
        org_input: OrganizationInput instance
        
    Returns:
        Dictionary representation
    """
    data = {
        "name": org_input.name,
        "active": org_input.active
    }
    
    # Add optional fields
    if org_input.legal_name is not None:
        data["legal_name"] = org_input.legal_name
    if org_input.trading_name is not None:
        data["trading_name"] = org_input.trading_name
    if org_input.organization_type is not None:
        data["organization_type"] = org_input.organization_type.value
    if org_input.website_url is not None:
        data["website_url"] = org_input.website_url
    if org_input.tax_id is not None:
        data["tax_id"] = org_input.tax_id
    if org_input.license_number is not None:
        data["license_number"] = org_input.license_number
    if org_input.part_of is not None:
        data["part_of"] = org_input.part_of
    if org_input.alias is not None:
        data["alias"] = org_input.alias
    
    # Convert contact points
    if org_input.telecom:
        data["telecom"] = [
            {
                "system": cp.system,
                "value": cp.value,
                "use": cp.use,
                "rank": cp.rank
            }
            for cp in org_input.telecom
        ]
    
    # Convert addresses
    if org_input.address:
        data["address"] = [
            {
                "use": addr.use,
                "type": addr.type,
                "text": addr.text,
                "line": addr.line,
                "city": addr.city,
                "district": addr.district,
                "state": addr.state,
                "postal_code": addr.postal_code,
                "country": addr.country
            }
            for addr in org_input.address
        ]
    
    # Convert identifiers
    if org_input.identifier:
        data["identifier"] = [
            {
                "use": ident.use,
                "type_code": ident.type_code,
                "type_display": ident.type_display,
                "system": ident.system,
                "value": ident.value,
                "assigner": ident.assigner
            }
            for ident in org_input.identifier
        ]
    
    return data

async def get_organization_resolver(organization_id: str, info: Info) -> Optional[Organization]:
    """
    Resolver for getting an organization by ID.
    
    Args:
        organization_id: Organization ID
        info: GraphQL resolver info
        
    Returns:
        Organization if found, None otherwise
    """
    try:
        management_service = get_management_service()
        await management_service.initialize()
        
        org_model = await management_service.get_organization(organization_id)
        
        if org_model:
            return convert_model_to_graphql(org_model)
        return None
        
    except Exception as e:
        logger.error(f"Error in get_organization_resolver: {str(e)}")
        return None

async def search_organizations_resolver(
    name: Optional[str] = None,
    organization_type: Optional[str] = None,
    status: Optional[str] = None,
    active: Optional[bool] = None,
    info: Info = None
) -> OrganizationSearchResult:
    """
    Resolver for searching organizations.
    
    Args:
        name: Organization name filter
        organization_type: Organization type filter
        status: Organization status filter
        active: Active status filter
        info: GraphQL resolver info
        
    Returns:
        OrganizationSearchResult with matching organizations
    """
    try:
        management_service = get_management_service()
        await management_service.initialize()
        
        # Build search parameters
        search_params = {}
        if name:
            search_params["name"] = name
        if organization_type:
            search_params["type"] = organization_type
        if status:
            search_params["status"] = status
        if active is not None:
            search_params["active"] = str(active).lower()
        
        org_models = await management_service.search_organizations(search_params)
        
        organizations = [convert_model_to_graphql(org) for org in org_models]
        
        return OrganizationSearchResult(
            organizations=organizations,
            total_count=len(organizations),
            has_more=False  # TODO: Implement pagination
        )
        
    except Exception as e:
        logger.error(f"Error in search_organizations_resolver: {str(e)}")
        return OrganizationSearchResult(
            organizations=[],
            total_count=0,
            has_more=False
        )

async def create_organization_resolver(organization_data: OrganizationInput, info: Info) -> Optional[Organization]:
    """
    Resolver for creating an organization.
    
    Args:
        organization_data: Organization input data
        info: GraphQL resolver info
        
    Returns:
        Created organization if successful, None otherwise
    """
    try:
        # TODO: Extract user ID from GraphQL context/headers
        # For now, using a placeholder
        current_user_id = "system"  # This should come from authentication context
        
        management_service = get_management_service()
        await management_service.initialize()
        
        # Convert input to dictionary
        org_dict = convert_input_to_dict(organization_data)
        
        org_model = await management_service.create_organization(org_dict, current_user_id)
        
        if org_model:
            return convert_model_to_graphql(org_model)
        return None
        
    except Exception as e:
        logger.error(f"Error in create_organization_resolver: {str(e)}")
        return None

async def update_organization_resolver(
    organization_id: str, 
    update_data: OrganizationUpdateInput, 
    info: Info
) -> Optional[Organization]:
    """
    Resolver for updating an organization.
    
    Args:
        organization_id: Organization ID to update
        update_data: Organization update data
        info: GraphQL resolver info
        
    Returns:
        Updated organization if successful, None otherwise
    """
    try:
        # TODO: Extract user ID from GraphQL context/headers
        current_user_id = "system"  # This should come from authentication context
        
        management_service = get_management_service()
        await management_service.initialize()
        
        # Convert update input to dictionary (only include non-None values)
        update_dict = {}
        if update_data.name is not None:
            update_dict["name"] = update_data.name
        if update_data.legal_name is not None:
            update_dict["legal_name"] = update_data.legal_name
        if update_data.trading_name is not None:
            update_dict["trading_name"] = update_data.trading_name
        if update_data.organization_type is not None:
            update_dict["organization_type"] = update_data.organization_type.value
        if update_data.active is not None:
            update_dict["active"] = update_data.active
        if update_data.website_url is not None:
            update_dict["website_url"] = update_data.website_url
        if update_data.tax_id is not None:
            update_dict["tax_id"] = update_data.tax_id
        if update_data.license_number is not None:
            update_dict["license_number"] = update_data.license_number
        if update_data.part_of is not None:
            update_dict["part_of"] = update_data.part_of
        if update_data.alias is not None:
            update_dict["alias"] = update_data.alias
        
        # TODO: Handle telecom, address, and identifier updates
        
        org_model = await management_service.update_organization(
            organization_id, 
            update_dict, 
            current_user_id
        )
        
        if org_model:
            return convert_model_to_graphql(org_model)
        return None
        
    except Exception as e:
        logger.error(f"Error in update_organization_resolver: {str(e)}")
        return None

# User Management Resolvers

def convert_user_model_to_graphql(user_model: Dict[str, Any]) -> User:
    """
    Convert user model dictionary to GraphQL User type.

    Args:
        user_model: User model dictionary

    Returns:
        User GraphQL type
    """
    from app.graphql.types import UserRole

    # Convert role string to enum
    role_enum = None
    if user_model.get("role"):
        try:
            role_enum = UserRole(user_model["role"])
        except ValueError:
            role_enum = UserRole.DOCTOR  # Default fallback

    # Convert identifiers
    identifier = None
    if user_model.get("identifier"):
        identifier = [
            Identifier(
                use=ident.get("use"),
                type=None,  # TODO: Convert type properly
                system=ident.get("system"),
                value=ident.get("value"),
                period=None,
                assigner=None
            )
            for ident in user_model["identifier"]
        ]

    return User(
        id=user_model.get("id"),
        email=user_model.get("email", ""),
        first_name=user_model.get("first_name", ""),
        last_name=user_model.get("last_name", ""),
        role=role_enum,
        organization_id=user_model.get("organization_id"),
        license_number=user_model.get("license_number"),
        specialization=user_model.get("specialization"),
        department=user_model.get("department"),
        phone_number=user_model.get("phone_number"),
        identifier=identifier,  # Include the converted identifiers
        is_active=user_model.get("is_active", True),
        created_at=user_model.get("created_at"),
        updated_at=user_model.get("updated_at")
    )

def convert_user_input_to_dict(user_input: UserInput) -> Dict[str, Any]:
    """
    Convert UserInput to dictionary for service layer.

    Args:
        user_input: UserInput instance

    Returns:
        Dictionary representation
    """
    data = {
        "email": user_input.email,
        "first_name": user_input.first_name,
        "last_name": user_input.last_name,
        "role": user_input.role.value,
        "is_active": user_input.is_active
    }

    # Add optional fields
    if user_input.organization_id is not None:
        data["organization_id"] = user_input.organization_id
    if user_input.license_number is not None:
        data["license_number"] = user_input.license_number
    if user_input.specialization is not None:
        data["specialization"] = user_input.specialization
    if user_input.department is not None:
        data["department"] = user_input.department
    if user_input.phone_number is not None:
        data["phone_number"] = user_input.phone_number

    # Convert identifiers
    if user_input.identifier:
        data["identifier"] = [
            {
                "use": ident.use,
                "system": ident.system,
                "value": ident.value,
                "type_code": ident.type.text if ident.type else None,
                "assigner": ident.assigner.reference if ident.assigner else None
            }
            for ident in user_input.identifier
            if ident.system and ident.value  # Only include identifiers with system and value
        ]

    return data

async def create_user_resolver(user_data: UserInput, info: Info) -> Optional[User]:
    """
    Resolver for creating a user.

    Args:
        user_data: User input data
        info: GraphQL resolver info

    Returns:
        Created user if successful, None otherwise
    """
    try:
        # TODO: Extract user ID from GraphQL context/headers
        current_user_id = "system"  # This should come from authentication context

        user_service = get_user_management_service()
        await user_service.initialize()

        # Convert input to dictionary
        user_dict = convert_user_input_to_dict(user_data)

        user_model = await user_service.create_user(user_dict, current_user_id)

        if user_model:
            return convert_user_model_to_graphql(user_model)
        return None

    except Exception as e:
        logger.error(f"Error in create_user_resolver: {str(e)}")
        return None

async def get_user_resolver(user_id: str, info: Info) -> Optional[User]:
    """
    Resolver for getting a user by ID.

    Args:
        user_id: User ID
        info: GraphQL resolver info

    Returns:
        User if found, None otherwise
    """
    try:
        user_service = get_user_management_service()
        await user_service.initialize()

        user_model = await user_service.get_user(user_id)

        if user_model:
            return convert_user_model_to_graphql(user_model)
        return None

    except Exception as e:
        logger.error(f"Error in get_user_resolver: {str(e)}")
        return None

async def search_users_resolver(
    organization_id: Optional[str] = None,
    role: Optional[str] = None,
    active: Optional[bool] = None,
    info: Info = None
) -> UserSearchResult:
    """
    Resolver for searching users.

    Args:
        organization_id: Organization ID filter
        role: User role filter
        active: Active status filter
        info: GraphQL resolver info

    Returns:
        UserSearchResult with matching users
    """
    try:
        user_service = get_user_management_service()
        await user_service.initialize()

        # Build search parameters
        search_params = {}
        if organization_id:
            search_params["organization_id"] = organization_id
        if role:
            search_params["role"] = role
        if active is not None:
            search_params["active"] = active

        user_models = await user_service.search_users(search_params)

        users = [convert_user_model_to_graphql(user) for user in user_models]

        return UserSearchResult(
            users=users,
            total_count=len(users),
            has_more=False  # TODO: Implement pagination
        )

    except Exception as e:
        logger.error(f"Error in search_users_resolver: {str(e)}")
        return UserSearchResult(
            users=[],
            total_count=0,
            has_more=False
        )

async def update_user_resolver(
    user_id: str,
    update_data: UserUpdateInput,
    info: Info
) -> Optional[User]:
    """
    Resolver for updating a user.

    Args:
        user_id: User ID to update
        update_data: User update data
        info: GraphQL resolver info

    Returns:
        Updated user if successful, None otherwise
    """
    try:
        # TODO: Extract user ID from GraphQL context/headers
        current_user_id = "system"  # This should come from authentication context

        user_service = get_user_management_service()
        await user_service.initialize()

        # Convert update input to dictionary (only include non-None values)
        update_dict = {}
        if update_data.first_name is not None:
            update_dict["first_name"] = update_data.first_name
        if update_data.last_name is not None:
            update_dict["last_name"] = update_data.last_name
        if update_data.role is not None:
            update_dict["role"] = update_data.role.value
        if update_data.organization_id is not None:
            update_dict["organization_id"] = update_data.organization_id
        if update_data.license_number is not None:
            update_dict["license_number"] = update_data.license_number
        if update_data.specialization is not None:
            update_dict["specialization"] = update_data.specialization
        if update_data.department is not None:
            update_dict["department"] = update_data.department
        if update_data.phone_number is not None:
            update_dict["phone_number"] = update_data.phone_number
        if update_data.is_active is not None:
            update_dict["is_active"] = update_data.is_active

        user_model = await user_service.update_user(user_id, update_dict, current_user_id)

        if user_model:
            return convert_user_model_to_graphql(user_model)
        return None

    except Exception as e:
        logger.error(f"Error in update_user_resolver: {str(e)}")
        return None

async def deactivate_user_resolver(user_id: str, info: Info) -> bool:
    """
    Resolver for deactivating a user.

    Args:
        user_id: User ID to deactivate
        info: GraphQL resolver info

    Returns:
        True if successful, False otherwise
    """
    try:
        # TODO: Extract user ID from GraphQL context/headers
        current_user_id = "system"  # This should come from authentication context

        user_service = get_user_management_service()
        await user_service.initialize()

        return await user_service.deactivate_user(user_id, current_user_id)

    except Exception as e:
        logger.error(f"Error in deactivate_user_resolver: {str(e)}")
        return False

async def get_user_by_supabase_id_resolver(supabase_id: str, info: Info) -> Optional[User]:
    """
    Resolver for finding a user by their Supabase user ID.

    Args:
        supabase_id: Supabase user ID to search for
        info: GraphQL resolver info

    Returns:
        User if found, None otherwise
    """
    try:
        user_service = get_user_management_service()
        await user_service.initialize()

        # Search for user with the specific Supabase identifier
        search_params = {
            "supabase_id": supabase_id
        }

        user_models = await user_service.search_users(search_params)

        if user_models:
            # Return the first match
            return convert_user_model_to_graphql(user_models[0])

        return None

    except Exception as e:
        logger.error(f"Error in get_user_by_supabase_id_resolver: {str(e)}")
        return None
