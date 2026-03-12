import strawberry
from typing import List, Optional, Any
from datetime import datetime
from enum import Enum

# Define a GenericScalar equivalent for compatibility with Graphene services
GenericScalar = strawberry.scalar(
    Any,
    name="GenericScalar",
    description="The GenericScalar scalar type represents a generic GraphQL scalar value that could be: String, Boolean, Int, Float, List or Object."
)

@strawberry.enum
class OrganizationType(Enum):
    """Organization type enumeration for GraphQL."""
    HOSPITAL = "hospital"
    CLINIC = "clinic"
    SPECIALTY_PRACTICE = "specialty-practice"
    LABORATORY = "laboratory"
    PHARMACY = "pharmacy"
    DEPARTMENT = "department"
    HEALTHCARE_COMPANY = "healthcare-company"
    INSURANCE_COMPANY = "insurance-company"
    OTHER = "other"

@strawberry.enum
class OrganizationStatus(Enum):
    """Organization status enumeration for GraphQL."""
    ACTIVE = "active"
    INACTIVE = "inactive"
    SUSPENDED = "suspended"
    PENDING_VERIFICATION = "pending-verification"
    VERIFIED = "verified"

@strawberry.enum
class UserRole(Enum):
    """User roles in the healthcare system."""
    DOCTOR = "DOCTOR"
    NURSE = "NURSE"
    ADMIN = "ADMIN"
    TECHNICIAN = "TECHNICIAN"
    PHARMACIST = "PHARMACIST"

@strawberry.type
class Reference:
    """GraphQL type for FHIR Reference."""
    reference: Optional[str] = strawberry.federation.field(shareable=True)
    type: Optional[str] = strawberry.federation.field(shareable=True)
    identifier: Optional["Identifier"] = strawberry.federation.field(shareable=True)
    display: Optional[str] = strawberry.federation.field(shareable=True)

@strawberry.type
class ContactPoint:
    """GraphQL type for contact point information."""
    system: Optional[str] = strawberry.federation.field(shareable=True)
    value: Optional[str] = strawberry.federation.field(shareable=True)
    use: Optional[str] = strawberry.federation.field(shareable=True)
    rank: Optional[int] = strawberry.federation.field(shareable=True)
    period: Optional[GenericScalar] = strawberry.federation.field(shareable=True)

@strawberry.type
class Address:
    """GraphQL type for address information."""
    use: Optional[str] = strawberry.federation.field(shareable=True)
    type: Optional[str] = strawberry.federation.field(shareable=True)
    text: Optional[str] = strawberry.federation.field(shareable=True)
    line: Optional[List[str]] = strawberry.federation.field(shareable=True)
    city: Optional[str] = strawberry.federation.field(shareable=True)
    district: Optional[str] = strawberry.federation.field(shareable=True)
    state: Optional[str] = strawberry.federation.field(shareable=True)
    postal_code: Optional[str] = strawberry.federation.field(shareable=True)
    country: Optional[str] = strawberry.federation.field(shareable=True)
    period: Optional[GenericScalar] = strawberry.federation.field(shareable=True)

@strawberry.type
class CodeableConcept:
    """GraphQL type for FHIR CodeableConcept."""
    text: Optional[str] = strawberry.federation.field(shareable=True)
    coding: Optional[List["Coding"]] = strawberry.federation.field(shareable=True)

@strawberry.type
class Coding:
    """GraphQL type for FHIR Coding."""
    system: Optional[str] = strawberry.federation.field(shareable=True)
    code: Optional[str] = strawberry.federation.field(shareable=True)
    display: Optional[str] = strawberry.federation.field(shareable=True)
    version: Optional[str] = strawberry.federation.field(shareable=True)
    user_selected: Optional[bool] = strawberry.federation.field(shareable=True)

@strawberry.type
class Period:
    """GraphQL type for FHIR Period."""
    start: Optional[str] = strawberry.federation.field(shareable=True)
    end: Optional[str] = strawberry.federation.field(shareable=True)

@strawberry.type
class Identifier:
    """GraphQL type for identifier information."""
    use: Optional[str] = strawberry.federation.field(shareable=True)
    type: Optional["CodeableConcept"] = strawberry.federation.field(shareable=True)
    system: Optional[str] = strawberry.federation.field(shareable=True)
    value: Optional[str] = strawberry.federation.field(shareable=True)
    period: Optional["Period"] = strawberry.federation.field(shareable=True)
    assigner: Optional["Reference"] = strawberry.federation.field(shareable=True)

@strawberry.federation.type(keys=["id"])
class Organization:
    """
    GraphQL Organization type with federation support.
    
    This type represents healthcare organizations and supports
    Apollo Federation for distributed GraphQL schemas.
    """
    id: strawberry.ID
    resource_type: str = "Organization"
    active: bool = True
    
    # Organization identification
    identifier: Optional[List[Identifier]] = None
    name: Optional[str] = None
    alias: Optional[List[str]] = None
    legal_name: Optional[str] = None
    trading_name: Optional[str] = None
    
    # Organization classification
    organization_type: Optional[OrganizationType] = None
    status: Optional[OrganizationStatus] = None
    
    # Contact information
    telecom: Optional[List[ContactPoint]] = None
    address: Optional[List[Address]] = None
    website_url: Optional[str] = None
    
    # Business information
    tax_id: Optional[str] = None
    license_number: Optional[str] = None
    
    # Hierarchical relationships
    part_of: Optional[str] = None
    
    # Verification information
    verification_status: Optional[str] = None
    verification_documents: Optional[List[str]] = None
    verified_by: Optional[str] = None
    verification_timestamp: Optional[datetime] = None
    
    # Audit information
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None
    created_by: Optional[str] = None
    updated_by: Optional[str] = None

@strawberry.input
class ContactPointInput:
    """GraphQL input type for contact point information."""
    system: Optional[str] = None
    value: Optional[str] = None
    use: Optional[str] = None
    rank: Optional[int] = None
    period: Optional[GenericScalar] = None

@strawberry.input
class AddressInput:
    """GraphQL input type for address information."""
    use: Optional[str] = None
    type: Optional[str] = None
    text: Optional[str] = None
    line: Optional[List[str]] = None
    city: Optional[str] = None
    district: Optional[str] = None
    state: Optional[str] = None
    postal_code: Optional[str] = None
    country: Optional[str] = None
    period: Optional[GenericScalar] = None

@strawberry.input
class ReferenceInput:
    """GraphQL input type for FHIR Reference."""
    reference: Optional[str] = None
    type: Optional[str] = None
    display: Optional[str] = None

@strawberry.input
class CodeableConceptInput:
    """GraphQL input type for FHIR CodeableConcept."""
    text: Optional[str] = None
    coding: Optional[List["CodingInput"]] = None

@strawberry.input
class CodingInput:
    """GraphQL input type for FHIR Coding."""
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    version: Optional[str] = None
    user_selected: Optional[bool] = None

@strawberry.input
class PeriodInput:
    """GraphQL input type for FHIR Period."""
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.input
class IdentifierInput:
    """GraphQL input type for identifier information."""
    use: Optional[str] = None
    type: Optional["CodeableConceptInput"] = None
    system: Optional[str] = None
    value: Optional[str] = None
    period: Optional["PeriodInput"] = None
    assigner: Optional["ReferenceInput"] = None

@strawberry.input
class OrganizationInput:
    """GraphQL input type for creating organizations."""
    name: str
    legal_name: Optional[str] = None
    trading_name: Optional[str] = None
    organization_type: Optional[OrganizationType] = None
    active: bool = True
    
    # Contact information
    telecom: Optional[List[ContactPointInput]] = None
    address: Optional[List[AddressInput]] = None
    website_url: Optional[str] = None
    
    # Business information
    tax_id: Optional[str] = None
    license_number: Optional[str] = None
    
    # Hierarchical relationships
    part_of: Optional[str] = None
    
    # Identifiers
    identifier: Optional[List[IdentifierInput]] = None
    alias: Optional[List[str]] = None

@strawberry.input
class OrganizationUpdateInput:
    """GraphQL input type for updating organizations."""
    name: Optional[str] = None
    legal_name: Optional[str] = None
    trading_name: Optional[str] = None
    organization_type: Optional[OrganizationType] = None
    active: Optional[bool] = None
    
    # Contact information
    telecom: Optional[List[ContactPointInput]] = None
    address: Optional[List[AddressInput]] = None
    website_url: Optional[str] = None
    
    # Business information
    tax_id: Optional[str] = None
    license_number: Optional[str] = None
    
    # Hierarchical relationships
    part_of: Optional[str] = None
    
    # Identifiers
    identifier: Optional[List[IdentifierInput]] = None
    alias: Optional[List[str]] = None

@strawberry.type
class OrganizationSearchResult:
    """GraphQL type for organization search results."""
    organizations: List[Organization]
    total_count: int
    has_more: bool

@strawberry.federation.type(keys=["id"])
class User:
    """
    GraphQL User type for healthcare professionals.

    This type represents healthcare professionals (doctors, nurses, etc.)
    associated with organizations.
    """
    id: strawberry.ID
    email: str
    first_name: str
    last_name: str
    role: UserRole
    organization_id: Optional[str] = None

    # Professional information
    license_number: Optional[str] = None
    specialization: Optional[str] = None
    department: Optional[str] = None
    phone_number: Optional[str] = None

    # Identifiers
    identifier: Optional[List[Identifier]] = None

    # Status
    is_active: bool = True

    # Audit information
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

@strawberry.input
class UserInput:
    """GraphQL input type for creating users."""
    email: str
    first_name: str
    last_name: str
    role: UserRole
    organization_id: Optional[str] = None

    # Professional information
    license_number: Optional[str] = None
    specialization: Optional[str] = None
    department: Optional[str] = None
    phone_number: Optional[str] = None

    # Identifiers
    identifier: Optional[List[IdentifierInput]] = None

    # Status
    is_active: bool = True

@strawberry.input
class UserUpdateInput:
    """GraphQL input type for updating users."""
    first_name: Optional[str] = None
    last_name: Optional[str] = None
    role: Optional[UserRole] = None
    organization_id: Optional[str] = None

    # Professional information
    license_number: Optional[str] = None
    specialization: Optional[str] = None
    department: Optional[str] = None
    phone_number: Optional[str] = None

    # Identifiers
    identifier: Optional[List[IdentifierInput]] = None

    # Status
    is_active: Optional[bool] = None

@strawberry.type
class UserSearchResult:
    """GraphQL type for user search results."""
    users: List[User]
    total_count: int
    has_more: bool
