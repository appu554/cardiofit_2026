"""
FHIR validation utilities for Clinical Synthesis Hub.

This module provides functions for validating FHIR resources
against the FHIR specification.
"""

from typing import Any, Dict, List, Optional, Type, Union, TypeVar
from pydantic import ValidationError

# Define a type variable for FHIR model
try:
    from fhir.resources import FHIRAbstractModel, get_fhir_model_class
    FHIR_AVAILABLE = True
except ImportError:
    FHIR_AVAILABLE = False
    FHIRAbstractModel = TypeVar('FHIRAbstractModel')  # Dummy type for type checking
    
    def get_fhir_model_class(resource_type: str) -> Any:
        raise ImportError(
            "fhir.resources is required for FHIR validation. "
            "Install with 'pip install fhir.resources'"
        )

class FHIRValidationError(Exception):
    """Exception raised for FHIR validation errors."""

    def __init__(self, message: str, errors: Optional[List[Dict[str, Any]]] = None):
        self.message = message
        self.errors = errors or []
        super().__init__(self.message)

def validate_fhir_resource(
    resource_data: Dict[str, Any],
    resource_type: Optional[str] = None
) -> Dict[str, Any]:
    """
    Validate a FHIR resource against the FHIR specification.
    
    Note: If fhir.resources is not installed, this will perform basic validation
    but won't validate against the full FHIR specification.

    Args:
        resource_data: A dictionary representing a FHIR resource
        resource_type: The FHIR resource type (e.g., 'Patient', 'Observation')
                      If not provided, uses the resourceType field from resource_data

    Returns:
        The validated resource data

    Raises:
        FHIRValidationError: If the resource is invalid
    """
    if not FHIR_AVAILABLE:
        # Perform basic validation if fhir.resources is not available
        if not isinstance(resource_data, dict):
            raise FHIRValidationError("Resource data must be a dictionary")
            
        if resource_type is None:
            resource_type = resource_data.get('resourceType')
            if not resource_type:
                raise FHIRValidationError("Resource type not specified and not found in resource data")
                
        if not isinstance(resource_type, str):
            raise FHIRValidationError("Resource type must be a string")
            
        return resource_data
        
    # Full FHIR validation with fhir.resources
    if resource_type is None:
        resource_type = resource_data.get('resourceType')
        if resource_type is None:
            raise FHIRValidationError("Resource type not specified and not found in resource data")

    try:
        # Get the appropriate FHIR model class
        fhir_class = get_fhir_model_class(resource_type)

        # Validate the resource
        try:
            fhir_model = fhir_class.model_validate(resource_data)

            # Additional validation for specific resource types
            if resource_type == "Patient" and resource_data.get("gender") not in [None, "male", "female", "other", "unknown"]:
                raise ValueError(f"Invalid gender: {resource_data.get('gender')}. Must be one of: male, female, other, unknown")

            if resource_type == "Observation" and resource_data.get("status") not in [None, "registered", "preliminary", "final", "amended", "corrected", "cancelled", "entered-in-error", "unknown"]:
                raise ValueError(f"Invalid status: {resource_data.get('status')}. Must be one of: registered, preliminary, final, amended, corrected, cancelled, entered-in-error, unknown")

            # Return the validated data
            return fhir_model.model_dump(exclude_unset=True)

        except Exception as validation_error:
            # Re-raise as ValidationError for consistent handling
            raise ValidationError.from_exception_data(
                title="Validation Error",
                line_errors=[{
                    "type": "value_error",
                    "loc": ("resourceType",),
                    "msg": str(validation_error),
                    "input": resource_type
                }]
            )

    except ValidationError as e:
        # Convert Pydantic validation errors to our format
        errors = []
        for error in e.errors():
            errors.append({
                'loc': '.'.join(str(loc) for loc in error['loc']),
                'msg': error['msg'],
                'type': error['type']
            })

        raise FHIRValidationError(
            f"Invalid {resource_type} resource: {e}",
            errors=errors
        )

    except Exception as e:
        raise FHIRValidationError(f"Error validating {resource_type} resource: {str(e)}")
