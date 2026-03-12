"""
Base transformer classes for converting between FHIR and GraphQL types.
"""

from typing import Any, Dict, List, Optional, Type, TypeVar, Union, Generic

T = TypeVar('T')  # GraphQL type
F = TypeVar('F', bound=Dict[str, Any])  # FHIR resource type

class TransformationError(Exception):
    """Exception raised when transformation fails."""
    pass

class BaseTransformer(Generic[T, F]):
    """
    Base class for transformers that convert between FHIR and GraphQL types.
    """
    
    @classmethod
    def fhir_to_graphql(cls, fhir_data: F) -> T:
        """
        Convert FHIR data to a GraphQL type.
        
        Args:
            fhir_data: FHIR resource data
            
        Returns:
            An instance of the GraphQL type
            
        Raises:
            TransformationError: If transformation fails
        """
        raise NotImplementedError("Subclasses must implement fhir_to_graphql")
    
    @classmethod
    def graphql_to_fhir(cls, graphql_data: T) -> F:
        """
        Convert GraphQL data to FHIR format.
        
        Args:
            graphql_data: GraphQL type instance
            
        Returns:
            FHIR resource data
            
        Raises:
            TransformationError: If transformation fails
        """
        raise NotImplementedError("Subclasses must implement graphql_to_fhir")

class TransformerRegistry:
    """
    Registry for transformers that convert between FHIR and GraphQL types.
    """
    
    _fhir_to_graphql_transformers = {}
    _graphql_to_fhir_transformers = {}
    
    @classmethod
    def register_fhir_to_graphql(cls, fhir_type: str, graphql_type: Type[T], transformer_cls: Type[BaseTransformer]):
        """
        Register a transformer for converting from FHIR to GraphQL.
        
        Args:
            fhir_type: FHIR resource type (e.g., "Patient")
            graphql_type: GraphQL type class
            transformer_cls: Transformer class
        """
        key = (fhir_type, graphql_type)
        cls._fhir_to_graphql_transformers[key] = transformer_cls
    
    @classmethod
    def register_graphql_to_fhir(cls, graphql_type: Type[T], fhir_type: str, transformer_cls: Type[BaseTransformer]):
        """
        Register a transformer for converting from GraphQL to FHIR.
        
        Args:
            graphql_type: GraphQL type class
            fhir_type: FHIR resource type (e.g., "Patient")
            transformer_cls: Transformer class
        """
        key = (graphql_type, fhir_type)
        cls._graphql_to_fhir_transformers[key] = transformer_cls
    
    @classmethod
    def get_fhir_to_graphql_transformer(cls, fhir_type: str, graphql_type: Type[T]) -> Optional[Type[BaseTransformer]]:
        """
        Get a transformer for converting from FHIR to GraphQL.
        
        Args:
            fhir_type: FHIR resource type (e.g., "Patient")
            graphql_type: GraphQL type class
            
        Returns:
            Transformer class or None if not found
        """
        key = (fhir_type, graphql_type)
        return cls._fhir_to_graphql_transformers.get(key)
    
    @classmethod
    def get_graphql_to_fhir_transformer(cls, graphql_type: Type[T], fhir_type: str) -> Optional[Type[BaseTransformer]]:
        """
        Get a transformer for converting from GraphQL to FHIR.
        
        Args:
            graphql_type: GraphQL type class
            fhir_type: FHIR resource type (e.g., "Patient")
            
        Returns:
            Transformer class or None if not found
        """
        key = (graphql_type, fhir_type)
        return cls._graphql_to_fhir_transformers.get(key)
    
    @classmethod
    def transform_fhir_to_graphql(cls, fhir_data: Dict[str, Any], graphql_type: Type[T]) -> T:
        """
        Transform FHIR data to a GraphQL type using the registered transformer.
        
        Args:
            fhir_data: FHIR resource data
            graphql_type: GraphQL type class
            
        Returns:
            An instance of the GraphQL type
            
        Raises:
            TransformationError: If transformation fails or no transformer is found
        """
        if not fhir_data:
            raise TransformationError("FHIR data is empty")
        
        fhir_type = fhir_data.get("resourceType")
        if not fhir_type:
            raise TransformationError("FHIR data is missing resourceType")
        
        transformer_cls = cls.get_fhir_to_graphql_transformer(fhir_type, graphql_type)
        if not transformer_cls:
            raise TransformationError(f"No transformer found for FHIR type {fhir_type} to GraphQL type {graphql_type.__name__}")
        
        return transformer_cls.fhir_to_graphql(fhir_data)
    
    @classmethod
    def transform_graphql_to_fhir(cls, graphql_data: T, fhir_type: str) -> Dict[str, Any]:
        """
        Transform GraphQL data to FHIR format using the registered transformer.
        
        Args:
            graphql_data: GraphQL type instance
            fhir_type: FHIR resource type (e.g., "Patient")
            
        Returns:
            FHIR resource data
            
        Raises:
            TransformationError: If transformation fails or no transformer is found
        """
        if not graphql_data:
            raise TransformationError("GraphQL data is empty")
        
        graphql_type = type(graphql_data)
        transformer_cls = cls.get_graphql_to_fhir_transformer(graphql_type, fhir_type)
        if not transformer_cls:
            raise TransformationError(f"No transformer found for GraphQL type {graphql_type.__name__} to FHIR type {fhir_type}")
        
        return transformer_cls.graphql_to_fhir(graphql_data)
