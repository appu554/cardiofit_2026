"""
Base class for FHIR to GraphQL transformers.
"""

import logging
from typing import Any, Dict, List, Optional, Type, TypeVar, Generic, get_type_hints

from pydantic import BaseModel
from shared.models.base import FHIRBaseModel

from ..base import BaseTransformer
from ..exceptions import TransformationError, ValidationError
from ..utils.case_conversion import snake_to_camel_keys

# Set up logging
logger = logging.getLogger(__name__)

# Type variables
F = TypeVar('F', bound=FHIRBaseModel)  # FHIR model type
G = TypeVar('G')  # GraphQL type

class FHIRToGraphQLTransformer(BaseTransformer[F, G], Generic[F, G]):
    """
    Base class for transformers that convert FHIR models to GraphQL types.
    """

    def _transform(self, source_data: F) -> G:
        """
        Transform a FHIR model to a GraphQL type.

        Args:
            source_data: The FHIR model to transform

        Returns:
            The GraphQL type instance
        """
        try:
            # Convert the FHIR model to a dictionary
            if hasattr(source_data, 'model_dump'):
                # If it's a Pydantic model
                fhir_dict = source_data.model_dump(exclude_none=True)
            elif isinstance(source_data, dict):
                # If it's already a dictionary
                fhir_dict = source_data
            else:
                # Try to convert to a dictionary using __dict__
                fhir_dict = source_data.__dict__

            # Convert snake_case keys to camelCase for GraphQL
            graphql_dict = snake_to_camel_keys(fhir_dict)

            # Transform nested objects
            graphql_dict = self._transform_nested_objects(graphql_dict)

            # Create the GraphQL type instance
            return self._create_graphql_instance(graphql_dict)
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

    def _create_graphql_instance(self, data: Dict[str, Any]) -> G:
        """
        Create a GraphQL type instance from the transformed data.

        Args:
            data: The transformed data

        Returns:
            The GraphQL type instance
        """
        try:
            # Create the GraphQL type instance
            # This assumes the GraphQL type has a constructor that accepts keyword arguments
            return self.target_type(**data)
        except Exception as e:
            logger.exception(f"Error creating {self.target_type.__name__} instance")
            raise TransformationError(
                f"Error creating {self.target_type.__name__} instance: {str(e)}",
                source_data=data,
                target_type=self.target_type,
                details=str(e)
            ) from e
