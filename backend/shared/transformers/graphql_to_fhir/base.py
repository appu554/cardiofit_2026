"""
Base class for GraphQL to FHIR transformers.
"""

import logging
from typing import Any, Dict, List, Optional, Type, TypeVar, Generic, get_type_hints

from pydantic import BaseModel
from shared.models.base import FHIRBaseModel

from ..base import BaseTransformer
from ..exceptions import TransformationError, ValidationError
from ..utils.case_conversion import camel_to_snake_keys

# Set up logging
logger = logging.getLogger(__name__)

# Type variables
G = TypeVar('G')  # GraphQL input type
F = TypeVar('F', bound=FHIRBaseModel)  # FHIR model type

class GraphQLToFHIRTransformer(BaseTransformer[G, F], Generic[G, F]):
    """
    Base class for transformers that convert GraphQL input types to FHIR models.
    """
    
    def _transform(self, source_data: G) -> F:
        """
        Transform a GraphQL input type to a FHIR model.
        
        Args:
            source_data: The GraphQL input type to transform
            
        Returns:
            The FHIR model instance
        """
        try:
            # Convert the GraphQL input to a dictionary
            if hasattr(source_data, "__dict__"):
                graphql_dict = source_data.__dict__
            elif hasattr(source_data, "model_dump"):
                graphql_dict = source_data.model_dump(exclude_none=True)
            else:
                # Assume it's already a dictionary
                graphql_dict = dict(source_data)
            
            # Convert camelCase keys to snake_case for FHIR
            fhir_dict = camel_to_snake_keys(graphql_dict)
            
            # Transform nested objects
            fhir_dict = self._transform_nested_objects(fhir_dict)
            
            # Create the FHIR model instance
            return self._create_fhir_instance(fhir_dict)
        except Exception as e:
            logger.exception(f"Error transforming {type(source_data).__name__} to {self.target_type.__name__}")
            raise TransformationError(
                f"Error transforming {type(source_data).__name__} to {self.target_type.__name__}: {str(e)}",
                source_data=source_data,
                target_type=self.target_type,
                details=str(e)
            ) from e
    
    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the data.
        
        This method should be overridden by subclasses to handle specific
        nested object transformations.
        
        Args:
            data: The data to transform
            
        Returns:
            The transformed data
        """
        return data
    
    def _create_fhir_instance(self, data: Dict[str, Any]) -> F:
        """
        Create a FHIR model instance from the transformed data.
        
        Args:
            data: The transformed data
            
        Returns:
            The FHIR model instance
        """
        try:
            # Create the FHIR model instance
            return self.target_type.model_validate(data)
        except Exception as e:
            logger.exception(f"Error creating {self.target_type.__name__} instance")
            raise ValidationError(
                f"Error creating {self.target_type.__name__} instance: {str(e)}",
                source_data=data,
                target_type=self.target_type,
                validation_errors=[str(e)]
            ) from e
