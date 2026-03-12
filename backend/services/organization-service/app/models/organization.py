from pydantic import BaseModel, Field, EmailStr
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum

class OrganizationType(str, Enum):
    """Organization type enumeration based on FHIR Organization types."""
    HOSPITAL = "hospital"
    CLINIC = "clinic"
    SPECIALTY_PRACTICE = "specialty-practice"
    LABORATORY = "laboratory"
    PHARMACY = "pharmacy"
    DEPARTMENT = "department"
    HEALTHCARE_COMPANY = "healthcare-company"
    INSURANCE_COMPANY = "insurance-company"
    OTHER = "other"

class OrganizationStatus(str, Enum):
    """Organization status enumeration."""
    ACTIVE = "active"
    INACTIVE = "inactive"
    SUSPENDED = "suspended"
    PENDING_VERIFICATION = "pending-verification"
    VERIFIED = "verified"

class ContactPoint(BaseModel):
    """FHIR ContactPoint model for organization contact information."""
    system: Optional[str] = Field(None, description="Contact point system (phone, email, fax, etc.)")
    value: Optional[str] = Field(None, description="Contact point value")
    use: Optional[str] = Field(None, description="Contact point use (work, home, mobile, etc.)")
    rank: Optional[int] = Field(None, description="Contact point preference rank")
    period_start: Optional[datetime] = Field(None, description="Contact point valid from")
    period_end: Optional[datetime] = Field(None, description="Contact point valid until")

class Address(BaseModel):
    """FHIR Address model for organization address."""
    use: Optional[str] = Field(None, description="Address use (work, home, temp, etc.)")
    type: Optional[str] = Field(None, description="Address type (postal, physical, both)")
    text: Optional[str] = Field(None, description="Full address as text")
    line: Optional[List[str]] = Field(None, description="Street address lines")
    city: Optional[str] = Field(None, description="City name")
    district: Optional[str] = Field(None, description="District/county")
    state: Optional[str] = Field(None, description="State/province")
    postal_code: Optional[str] = Field(None, description="Postal/ZIP code")
    country: Optional[str] = Field(None, description="Country")
    period_start: Optional[datetime] = Field(None, description="Address valid from")
    period_end: Optional[datetime] = Field(None, description="Address valid until")

class Identifier(BaseModel):
    """FHIR Identifier model for organization identifiers."""
    use: Optional[str] = Field(None, description="Identifier use (usual, official, temp, etc.)")
    type_code: Optional[str] = Field(None, description="Identifier type code")
    type_display: Optional[str] = Field(None, description="Identifier type display name")
    system: Optional[str] = Field(None, description="Identifier system/namespace")
    value: Optional[str] = Field(None, description="Identifier value")
    period_start: Optional[datetime] = Field(None, description="Identifier valid from")
    period_end: Optional[datetime] = Field(None, description="Identifier valid until")
    assigner: Optional[str] = Field(None, description="Organization that assigned the identifier")

class OrganizationContact(BaseModel):
    """FHIR Organization contact information."""
    purpose_code: Optional[str] = Field(None, description="Contact purpose code")
    purpose_display: Optional[str] = Field(None, description="Contact purpose display")
    name_family: Optional[str] = Field(None, description="Contact family name")
    name_given: Optional[List[str]] = Field(None, description="Contact given names")
    name_prefix: Optional[List[str]] = Field(None, description="Contact name prefixes")
    name_suffix: Optional[List[str]] = Field(None, description="Contact name suffixes")
    telecom: Optional[List[ContactPoint]] = Field(None, description="Contact telecommunications")
    address: Optional[Address] = Field(None, description="Contact address")

class Organization(BaseModel):
    """
    FHIR-compliant Organization model for healthcare organizations.
    
    This model represents healthcare organizations such as hospitals, clinics,
    departments, and other healthcare entities.
    """
    # FHIR Resource metadata
    resource_type: str = Field(default="Organization", description="FHIR resource type")
    id: Optional[str] = Field(None, description="Logical resource ID")
    
    # Organization identifiers
    identifier: Optional[List[Identifier]] = Field(None, description="Organization identifiers")
    
    # Organization status and type
    active: bool = Field(True, description="Whether the organization is active")
    type: Optional[List[Dict[str, Any]]] = Field(None, description="Organization type coding")
    
    # Organization names
    name: Optional[str] = Field(None, description="Primary organization name")
    alias: Optional[List[str]] = Field(None, description="Alternative organization names")
    
    # Contact information
    telecom: Optional[List[ContactPoint]] = Field(None, description="Organization contact points")
    address: Optional[List[Address]] = Field(None, description="Organization addresses")
    
    # Hierarchical relationships
    part_of: Optional[str] = Field(None, description="Reference to parent organization")
    
    # Contact persons
    contact: Optional[List[OrganizationContact]] = Field(None, description="Organization contacts")
    
    # Endpoints
    endpoint: Optional[List[str]] = Field(None, description="Organization endpoints")
    
    # Custom fields for our implementation
    legal_name: Optional[str] = Field(None, description="Legal organization name")
    trading_name: Optional[str] = Field(None, description="Trading/business name")
    organization_type: Optional[OrganizationType] = Field(None, description="Organization type")
    status: Optional[OrganizationStatus] = Field(OrganizationStatus.PENDING_VERIFICATION, description="Organization status")
    tax_id: Optional[str] = Field(None, description="Tax identification number")
    license_number: Optional[str] = Field(None, description="Professional license number")
    website_url: Optional[str] = Field(None, description="Organization website URL")
    
    # Verification fields
    verification_status: Optional[str] = Field("pending", description="Verification status")
    verification_documents: Optional[List[str]] = Field(None, description="Verification document URLs")
    verified_by: Optional[str] = Field(None, description="User ID who verified the organization")
    verification_timestamp: Optional[datetime] = Field(None, description="Verification timestamp")
    
    # Audit fields
    created_at: Optional[datetime] = Field(None, description="Creation timestamp")
    updated_at: Optional[datetime] = Field(None, description="Last update timestamp")
    created_by: Optional[str] = Field(None, description="User ID who created the organization")
    updated_by: Optional[str] = Field(None, description="User ID who last updated the organization")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat() if v else None
        }
        schema_extra = {
            "example": {
                "resource_type": "Organization",
                "active": True,
                "name": "City General Hospital",
                "legal_name": "City General Hospital Inc.",
                "trading_name": "City General",
                "organization_type": "hospital",
                "status": "active",
                "tax_id": "12-3456789",
                "license_number": "HL-2023-001",
                "website_url": "https://citygeneral.com",
                "telecom": [
                    {
                        "system": "phone",
                        "value": "+1-555-123-4567",
                        "use": "work"
                    },
                    {
                        "system": "email",
                        "value": "info@citygeneral.com",
                        "use": "work"
                    }
                ],
                "address": [
                    {
                        "use": "work",
                        "type": "physical",
                        "line": ["123 Healthcare Drive"],
                        "city": "Medical City",
                        "state": "CA",
                        "postal_code": "90210",
                        "country": "US"
                    }
                ]
            }
        }
