from pydantic import BaseModel, Field
from typing import Optional
from datetime import datetime
from enum import Enum

class RelationshipType(str, Enum):
    """Organization relationship type enumeration."""
    PARENT_CHILD = "parent-child"
    AFFILIATED_WITH = "affiliated-with"
    OWNS = "owns"
    PROVIDES_SERVICES_TO = "provides-services-to"
    CONTRACTS_WITH = "contracts-with"
    REFERS_TO = "refers-to"
    PARTNERSHIP = "partnership"
    SUBSIDIARY = "subsidiary"
    BRANCH = "branch"
    DEPARTMENT = "department"

class RelationshipStatus(str, Enum):
    """Relationship status enumeration."""
    ACTIVE = "active"
    INACTIVE = "inactive"
    PENDING = "pending"
    SUSPENDED = "suspended"
    TERMINATED = "terminated"

class OrganizationRelationship(BaseModel):
    """
    Organization relationship model for managing inter-organization relationships.
    
    This model defines relationships between organizations such as parent-child,
    affiliations, partnerships, and service relationships.
    """
    id: Optional[str] = Field(None, description="Relationship ID")
    
    # Relationship participants
    source_organization_id: str = Field(..., description="Source organization ID")
    target_organization_id: str = Field(..., description="Target organization ID")
    
    # Relationship definition
    relationship_type: RelationshipType = Field(..., description="Type of relationship")
    status: RelationshipStatus = Field(RelationshipStatus.ACTIVE, description="Relationship status")
    
    # Relationship details
    description: Optional[str] = Field(None, description="Relationship description")
    notes: Optional[str] = Field(None, description="Additional notes about the relationship")
    
    # Relationship validity period
    start_date: Optional[datetime] = Field(None, description="Relationship start date")
    end_date: Optional[datetime] = Field(None, description="Relationship end date")
    
    # Contract/agreement information
    contract_reference: Optional[str] = Field(None, description="Reference to contract or agreement")
    contract_start_date: Optional[datetime] = Field(None, description="Contract start date")
    contract_end_date: Optional[datetime] = Field(None, description="Contract end date")
    
    # Audit fields
    created_at: Optional[datetime] = Field(None, description="Creation timestamp")
    updated_at: Optional[datetime] = Field(None, description="Last update timestamp")
    created_by: Optional[str] = Field(None, description="User ID who created the relationship")
    updated_by: Optional[str] = Field(None, description="User ID who last updated the relationship")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat() if v else None
        }
        schema_extra = {
            "example": {
                "source_organization_id": "org-hospital-main",
                "target_organization_id": "org-clinic-branch",
                "relationship_type": "parent-child",
                "status": "active",
                "description": "Main hospital owns and operates the branch clinic",
                "start_date": "2023-01-01T00:00:00Z"
            }
        }

class OrganizationRelationshipInput(BaseModel):
    """Input model for creating organization relationships."""
    target_organization_id: str = Field(..., description="Target organization ID")
    relationship_type: RelationshipType = Field(..., description="Type of relationship")
    status: RelationshipStatus = Field(RelationshipStatus.ACTIVE, description="Relationship status")
    description: Optional[str] = Field(None, description="Relationship description")
    notes: Optional[str] = Field(None, description="Additional notes about the relationship")
    start_date: Optional[datetime] = Field(None, description="Relationship start date")
    end_date: Optional[datetime] = Field(None, description="Relationship end date")
    contract_reference: Optional[str] = Field(None, description="Reference to contract or agreement")
    contract_start_date: Optional[datetime] = Field(None, description="Contract start date")
    contract_end_date: Optional[datetime] = Field(None, description="Contract end date")
    
    class Config:
        schema_extra = {
            "example": {
                "target_organization_id": "org-clinic-partner",
                "relationship_type": "affiliated-with",
                "description": "Strategic partnership for patient referrals",
                "start_date": "2024-01-01T00:00:00Z"
            }
        }

class OrganizationRelationshipUpdate(BaseModel):
    """Update model for organization relationships."""
    status: Optional[RelationshipStatus] = Field(None, description="Relationship status")
    description: Optional[str] = Field(None, description="Relationship description")
    notes: Optional[str] = Field(None, description="Additional notes about the relationship")
    end_date: Optional[datetime] = Field(None, description="Relationship end date")
    contract_reference: Optional[str] = Field(None, description="Reference to contract or agreement")
    contract_end_date: Optional[datetime] = Field(None, description="Contract end date")
    
    class Config:
        schema_extra = {
            "example": {
                "status": "suspended",
                "notes": "Temporarily suspended pending contract renegotiation"
            }
        }
