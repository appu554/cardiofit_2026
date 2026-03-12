from pydantic import BaseModel, Field
from typing import Optional, Any, Dict
from datetime import datetime
from enum import Enum

class SettingType(str, Enum):
    """Setting type enumeration."""
    STRING = "string"
    INTEGER = "integer"
    BOOLEAN = "boolean"
    JSON = "json"
    ARRAY = "array"

class SettingCategory(str, Enum):
    """Setting category enumeration."""
    GENERAL = "general"
    BILLING = "billing"
    CLINICAL = "clinical"
    SECURITY = "security"
    INTEGRATION = "integration"
    UI_PREFERENCES = "ui_preferences"
    WORKFLOW = "workflow"
    NOTIFICATION = "notification"

class OrganizationSetting(BaseModel):
    """
    Organization-specific configuration settings.
    
    This model stores configuration settings that are specific to each organization,
    allowing for customization of system behavior per organization.
    """
    id: Optional[str] = Field(None, description="Setting ID")
    organization_id: str = Field(..., description="Organization ID this setting belongs to")
    
    # Setting identification
    key: str = Field(..., description="Setting key/name")
    category: Optional[SettingCategory] = Field(SettingCategory.GENERAL, description="Setting category")
    
    # Setting value and metadata
    value: Optional[Any] = Field(None, description="Setting value")
    value_type: Optional[SettingType] = Field(SettingType.STRING, description="Setting value type")
    default_value: Optional[Any] = Field(None, description="Default value for this setting")
    
    # Setting description and validation
    display_name: Optional[str] = Field(None, description="Human-readable setting name")
    description: Optional[str] = Field(None, description="Setting description")
    validation_rules: Optional[Dict[str, Any]] = Field(None, description="Validation rules for the setting")
    
    # Setting behavior
    is_required: bool = Field(False, description="Whether this setting is required")
    is_sensitive: bool = Field(False, description="Whether this setting contains sensitive data")
    is_inherited: bool = Field(False, description="Whether this setting is inherited from parent organization")
    can_override: bool = Field(True, description="Whether child organizations can override this setting")
    
    # Audit fields
    created_at: Optional[datetime] = Field(None, description="Creation timestamp")
    updated_at: Optional[datetime] = Field(None, description="Last update timestamp")
    created_by: Optional[str] = Field(None, description="User ID who created the setting")
    updated_by: Optional[str] = Field(None, description="User ID who last updated the setting")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat() if v else None
        }
        schema_extra = {
            "example": {
                "organization_id": "org-123",
                "key": "default_billing_contact",
                "category": "billing",
                "value": "billing@hospital.com",
                "value_type": "string",
                "display_name": "Default Billing Contact",
                "description": "Default email address for billing inquiries",
                "is_required": True,
                "is_sensitive": False
            }
        }

class OrganizationSettingInput(BaseModel):
    """Input model for creating/updating organization settings."""
    key: str = Field(..., description="Setting key/name")
    category: Optional[SettingCategory] = Field(SettingCategory.GENERAL, description="Setting category")
    value: Optional[Any] = Field(None, description="Setting value")
    value_type: Optional[SettingType] = Field(SettingType.STRING, description="Setting value type")
    display_name: Optional[str] = Field(None, description="Human-readable setting name")
    description: Optional[str] = Field(None, description="Setting description")
    is_required: bool = Field(False, description="Whether this setting is required")
    is_sensitive: bool = Field(False, description="Whether this setting contains sensitive data")
    
    class Config:
        schema_extra = {
            "example": {
                "key": "patient_portal_theme",
                "category": "ui_preferences",
                "value": {"primary_color": "#0078d7", "logo_url": "/assets/logo.png"},
                "value_type": "json",
                "display_name": "Patient Portal Theme",
                "description": "Customization settings for the patient portal interface"
            }
        }

class OrganizationSettingUpdate(BaseModel):
    """Update model for organization settings."""
    value: Optional[Any] = Field(None, description="Setting value")
    display_name: Optional[str] = Field(None, description="Human-readable setting name")
    description: Optional[str] = Field(None, description="Setting description")
    is_required: Optional[bool] = Field(None, description="Whether this setting is required")
    is_sensitive: Optional[bool] = Field(None, description="Whether this setting contains sensitive data")
    
    class Config:
        schema_extra = {
            "example": {
                "value": "updated-value@hospital.com",
                "description": "Updated setting description"
            }
        }
