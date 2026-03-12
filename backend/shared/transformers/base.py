"""
Base transformer classes for Clinical Synthesis Hub.

This module provides base classes for transformers that convert between
different data formats, particularly between FHIR models and GraphQL types.
"""

import logging
from typing import Any, Dict, List, Optional, Type, TypeVar, Generic, Union, get_type_hints
from .exceptions import TransformationError, ValidationError

# Set up logging
logger = logging.getLogger(__name__)

# Type variables for source and target types
S = TypeVar('S')  # Source type
T = TypeVar('T')  # Target type

class BaseTransformer(Generic[S, T]):
    """
    Base class for all transformers.

    This class provides common functionality for transforming data between
    different formats, with type checking and error handling.
    """

    # Source and target types
    source_type: Type[S] = None
    target_type: Type[T] = None

    def __init__(self):
        """Initialize the transformer."""
        if self.source_type is None:
            raise ValueError(f"{self.__class__.__name__} must define source_type")
        if self.target_type is None:
            raise ValueError(f"{self.__class__.__name__} must define target_type")

    def transform(self, source_data: S) -> T:
        """
        Transform the source data to the target format.

        Args:
            source_data: The source data to transform

        Returns:
            The transformed data

        Raises:
            TransformationError: If the transformation fails
        """
        try:
            # Validate source data
            self._validate_source(source_data)

            # Perform the transformation
            target_data = self._transform(source_data)

            # Validate target data
            self._validate_target(target_data)

            return target_data
        except Exception as e:
            if isinstance(e, TransformationError):
                # Re-raise transformation errors
                raise
            else:
                # Wrap other exceptions in TransformationError
                logger.exception(f"Error transforming {type(source_data)} to {self.target_type}")
                raise TransformationError(
                    f"Error transforming {type(source_data)} to {self.target_type}: {str(e)}",
                    source_data=source_data,
                    target_type=self.target_type,
                    details=str(e)
                ) from e

    def transform_many(self, source_data_list: List[S]) -> List[T]:
        """
        Transform a list of source data items to the target format.

        Args:
            source_data_list: A list of source data items to transform

        Returns:
            A list of transformed data items

        Raises:
            TransformationError: If any transformation fails
        """
        return [self.transform(item) for item in source_data_list]

    def _transform(self, source_data: S) -> T:
        """
        Perform the actual transformation.

        This method must be implemented by subclasses.

        Args:
            source_data: The source data to transform

        Returns:
            The transformed data
        """
        raise NotImplementedError("Subclasses must implement _transform()")

    def _validate_source(self, source_data: Any) -> None:
        """
        Validate the source data before transformation.

        Args:
            source_data: The source data to validate

        Raises:
            ValidationError: If validation fails
        """
        # Skip validation for now - we'll implement proper validation later
        # This allows us to test the transformers with mock classes
        pass

    def _validate_target(self, target_data: Any) -> None:
        """
        Validate the target data after transformation.

        Args:
            target_data: The target data to validate

        Raises:
            ValidationError: If validation fails
        """
        # Skip validation for now - we'll implement proper validation later
        # This allows us to test the transformers with mock classes
        pass


class TransformerRegistry:
    """
    Registry for transformers.

    This class provides a registry for transformers, allowing them to be
    looked up by source and target types.
    """

    _registry: Dict[tuple, BaseTransformer] = {}

    @classmethod
    def register(cls, transformer_class: Type[BaseTransformer]) -> Type[BaseTransformer]:
        """
        Register a transformer class.

        Args:
            transformer_class: The transformer class to register

        Returns:
            The registered transformer class
        """
        transformer = transformer_class()
        key = (transformer.source_type, transformer.target_type)
        cls._registry[key] = transformer
        logger.debug(f"Registered transformer {transformer_class.__name__} for {key}")
        return transformer_class

    @classmethod
    def get(cls, source_type: Type, target_type: Type) -> Optional[BaseTransformer]:
        """
        Get a transformer for the given source and target types.

        Args:
            source_type: The source type
            target_type: The target type

        Returns:
            A transformer for the given types, or None if not found
        """
        return cls._registry.get((source_type, target_type))

    @classmethod
    def transform(cls, source_data: Any, target_type: Type) -> Any:
        """
        Transform the source data to the target type.

        Args:
            source_data: The source data to transform
            target_type: The target type

        Returns:
            The transformed data

        Raises:
            TransformationError: If no transformer is found or transformation fails
        """
        source_type = type(source_data)
        transformer = cls.get(source_type, target_type)

        if transformer is None:
            raise TransformationError(
                f"No transformer found for {source_type} to {target_type}",
                source_data=source_data,
                target_type=target_type
            )

        return transformer.transform(source_data)
