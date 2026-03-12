"""
Base models for FHIR resources in the Clinical Synthesis Hub.

This module provides base classes and utilities for working with FHIR resources
across all microservices in the Clinical Synthesis Hub.
"""

import logging
from typing import Any, Dict, List, Optional, Type, TypeVar, Union, cast, Type as TypeType
from pydantic import BaseModel, Field, model_validator
from datetime import date, datetime

# Try to import FHIR resources, but make it optional
try:
    from fhir.resources import FHIRAbstractModel
    from fhir.resources import get_fhir_model_class
    FHIR_AVAILABLE = True
except ImportError:
    logging.warning(
        "fhir.resources module not found. FHIR-specific functionality will be limited. "
        "Install with 'pip install fhir.resources' for full FHIR support."
    )
    FHIR_AVAILABLE = False
    
    # Define a dummy class for type hints when FHIR is not available
    class FHIRAbstractModel:
        pass

# Type variable for generic methods
T = TypeVar('T', bound='FHIRBaseModel')

class FHIRBaseModel(BaseModel):
    """
    Base model for all FHIR resources in the Clinical Synthesis Hub.

    This model extends Pydantic's BaseModel with FHIR-specific functionality
    and provides methods for converting between our models and standard FHIR models.
    """

    model_config = {
        "extra": "allow",  # Allow extra fields for FHIR resources
        "arbitrary_types_allowed": True
    }

    @classmethod
    def from_fhir(cls: Type[T], fhir_dict: Dict[str, Any]) -> T:
        """
        Create an instance from a FHIR JSON dictionary.

        Args:
            fhir_dict: A dictionary representing a FHIR resource

        Returns:
            An instance of this model
        """
        return cls.model_validate(fhir_dict)

    @classmethod
    def from_fhir_model(cls: Type[T], fhir_model: Any) -> T:  # type: ignore
        """
        Create an instance from a FHIR model from fhir.resources.

        Args:
            fhir_model: A FHIR model instance from fhir.resources

        Returns:
            An instance of this model
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE:
            raise ImportError(
                "fhir.resources is required for from_fhir_model. "
                "Install with 'pip install fhir.resources'"
            )
            
        if not isinstance(fhir_model, FHIRAbstractModel):
            raise ValueError("fhir_model must be a valid FHIRAbstractModel instance")
            
        # Convert date objects to strings to avoid validation errors
        fhir_dict = fhir_model.model_dump(exclude_unset=True)

        # Handle date conversions
        for key, value in fhir_dict.items():
            if isinstance(value, (date, datetime)):
                fhir_dict[key] = value.isoformat()

        return cls.from_fhir(fhir_dict)

    def to_fhir(self) -> Dict[str, Any]:
        """
        Convert this model to a FHIR JSON dictionary.

        Returns:
            A dictionary representing a FHIR resource
        """
        return self.model_dump(exclude_none=True, by_alias=True)

    @classmethod
    def get_fhir_resource_type(cls) -> str:
        """
        Get the FHIR resource type for this model.

        Returns:
            The FHIR resource type as a string (e.g., 'Patient', 'Observation')
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE:
            raise ImportError(
                "fhir.resources is required for get_fhir_resource_type. "
                "Install with 'pip install fhir.resources'"
            )
            
        # Default implementation returns the class name
        return cls.__name__

    def to_fhir_model(self, resource_type: Optional[str] = None) -> Any:  # type: ignore
        """
        Convert this model to a FHIR model from fhir.resources.

        Args:
            resource_type: The FHIR resource type (e.g., 'Patient', 'Observation')
                           If not provided, uses the resourceType field if available

        Returns:
            A FHIR model instance from fhir.resources
            
        Raises:
            ImportError: If fhir.resources is not installed
        """
        if not FHIR_AVAILABLE:
            raise ImportError(
                "fhir.resources is required for to_fhir_model. "
                "Install with 'pip install fhir.resources'"
            )
        
        data = self.to_fhir()

        # Determine the resource type
        if resource_type is None:
            resource_type = data.get('resourceType')
            if resource_type is None:
                raise ValueError("Resource type not specified and not found in model")

        # Get the appropriate FHIR model class
        fhir_class = get_fhir_model_class(resource_type)

        # Create and return the FHIR model instance
        return fhir_class.model_validate(data)

# Export the FHIR_AVAILABLE constant
__all__ = ['FHIRBaseModel', 'FHIR_AVAILABLE']
