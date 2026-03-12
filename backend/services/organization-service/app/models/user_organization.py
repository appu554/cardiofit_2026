from pydantic import BaseModel, Field
from typing import Optional, List
from datetime import datetime
from enum import Enum

class UserStatus(str, Enum):
    """User status within an organization."""
    ACTIVE = "active"
    INACTIVE = "inactive"
    PENDING_APPROVAL = "pending-approval"
    SUSPENDED = "suspended"
    INVITED = "invited"
    DISABLED = "disabled"

class InvitationStatus(str, Enum):
    """Invitation status enumeration."""
    SENT = "sent"
    ACCEPTED = "accepted"
    EXPIRED = "expired"
    REVOKED = "revoked"
    PENDING = "pending"

class UserOrganizationAccess(BaseModel):
    """
    User-Organization association model.
    
    This model manages the relationship between users and organizations,
    including their roles and status within each organization.
    """
    id: Optional[str] = Field(None, description="Access record ID")
    
    # User and organization references
    user_id: str = Field(..., description="Supabase user ID")
    organization_id: str = Field(..., description="Organization ID")
    
    # Role and permissions within the organization
    role_id: Optional[str] = Field(None, description="Role ID within the organization")
    role_name: Optional[str] = Field(None, description="Role name within the organization")
    custom_permissions: Optional[List[str]] = Field(None, description="Additional custom permissions")
    
    # User status within the organization
    status: UserStatus = Field(UserStatus.PENDING_APPROVAL, description="User status in organization")
    
    # Employment/association details
    employee_id: Optional[str] = Field(None, description="Employee ID within the organization")
    job_title: Optional[str] = Field(None, description="Job title within the organization")
    department: Optional[str] = Field(None, description="Department within the organization")
    
    # Professional credentials
    professional_license_number: Optional[str] = Field(None, description="Professional license number")
    license_type: Optional[str] = Field(None, description="Type of professional license")
    license_expiry: Optional[datetime] = Field(None, description="License expiry date")
    specialty: Optional[str] = Field(None, description="Medical/professional specialty")
    
    # Access period
    assignment_date: Optional[datetime] = Field(None, description="Date user was assigned to organization")
    effective_start_date: Optional[datetime] = Field(None, description="Access effective start date")
    effective_end_date: Optional[datetime] = Field(None, description="Access effective end date")
    
    # Audit fields
    created_at: Optional[datetime] = Field(None, description="Creation timestamp")
    updated_at: Optional[datetime] = Field(None, description="Last update timestamp")
    created_by: Optional[str] = Field(None, description="User ID who created this access")
    updated_by: Optional[str] = Field(None, description="User ID who last updated this access")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat() if v else None
        }
        schema_extra = {
            "example": {
                "user_id": "user-123",
                "organization_id": "org-456",
                "role_name": "doctor",
                "status": "active",
                "employee_id": "EMP-001",
                "job_title": "Cardiologist",
                "department": "Cardiology",
                "professional_license_number": "MD-12345",
                "license_type": "Medical Doctor",
                "specialty": "Cardiology"
            }
        }

class UserInvitation(BaseModel):
    """
    User invitation model for inviting users to join organizations.
    """
    id: Optional[str] = Field(None, description="Invitation ID")
    
    # Invitation details
    invited_email: str = Field(..., description="Email address of invited user", pattern=r'^[^@]+@[^@]+\.[^@]+$')
    organization_id: str = Field(..., description="Organization ID")
    role_id: Optional[str] = Field(None, description="Role ID for the invitation")
    role_name: Optional[str] = Field(None, description="Role name for the invitation")
    
    # Invitation token and security
    invitation_token: str = Field(..., description="Secure invitation token")
    token_expiry: datetime = Field(..., description="Token expiry date/time")
    
    # Invitation status and metadata
    status: InvitationStatus = Field(InvitationStatus.SENT, description="Invitation status")
    message: Optional[str] = Field(None, description="Custom invitation message")
    
    # Job details for the invitation
    job_title: Optional[str] = Field(None, description="Proposed job title")
    department: Optional[str] = Field(None, description="Proposed department")
    
    # Invitation tracking
    invited_by: str = Field(..., description="User ID who sent the invitation")
    accepted_by: Optional[str] = Field(None, description="User ID who accepted the invitation")
    accepted_at: Optional[datetime] = Field(None, description="Acceptance timestamp")
    
    # Audit fields
    created_at: Optional[datetime] = Field(None, description="Creation timestamp")
    updated_at: Optional[datetime] = Field(None, description="Last update timestamp")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat() if v else None
        }
        schema_extra = {
            "example": {
                "invited_email": "doctor@example.com",
                "organization_id": "org-456",
                "role_name": "doctor",
                "invitation_token": "secure-token-123",
                "token_expiry": "2024-02-01T00:00:00Z",
                "job_title": "Emergency Medicine Physician",
                "department": "Emergency Department",
                "message": "Welcome to our healthcare team!"
            }
        }

class UserOrganizationInput(BaseModel):
    """Input model for creating user-organization associations."""
    user_id: str = Field(..., description="Supabase user ID")
    role_name: str = Field(..., description="Role name within the organization")
    employee_id: Optional[str] = Field(None, description="Employee ID within the organization")
    job_title: Optional[str] = Field(None, description="Job title within the organization")
    department: Optional[str] = Field(None, description="Department within the organization")
    professional_license_number: Optional[str] = Field(None, description="Professional license number")
    license_type: Optional[str] = Field(None, description="Type of professional license")
    specialty: Optional[str] = Field(None, description="Medical/professional specialty")
    
    class Config:
        schema_extra = {
            "example": {
                "user_id": "user-789",
                "role_name": "nurse",
                "employee_id": "NURSE-001",
                "job_title": "Registered Nurse",
                "department": "ICU"
            }
        }

class InvitationInput(BaseModel):
    """Input model for creating user invitations."""
    invited_email: str = Field(..., description="Email address of invited user", pattern=r'^[^@]+@[^@]+\.[^@]+$')
    role_name: str = Field(..., description="Role name for the invitation")
    job_title: Optional[str] = Field(None, description="Proposed job title")
    department: Optional[str] = Field(None, description="Proposed department")
    message: Optional[str] = Field(None, description="Custom invitation message")
    
    class Config:
        schema_extra = {
            "example": {
                "invited_email": "newdoctor@example.com",
                "role_name": "doctor",
                "job_title": "Pediatrician",
                "department": "Pediatrics",
                "message": "We would love to have you join our pediatrics team!"
            }
        }
