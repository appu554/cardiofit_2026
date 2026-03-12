"""
FHIR to GraphQL resolver utilities.

This module provides utilities for resolving GraphQL fields from FHIR data.
"""

import logging
import strawberry
from typing import Any, Dict, List, Optional, Type, TypeVar, Union, get_type_hints

# Set up logging
logger = logging.getLogger(__name__)

T = TypeVar('T')

class FHIRResolver:
    """
    Resolver for FHIR data in GraphQL.
    
    This class provides methods for resolving GraphQL fields from FHIR data.
    """
    
    @staticmethod
    def resolve_field(source: Dict[str, Any], field_name: str) -> Any:
        """
        Resolve a field from FHIR data.
        
        Args:
            source: The FHIR data
            field_name: The name of the field to resolve
            
        Returns:
            The resolved field value
        """
        # Handle special case for 'class' field
        if field_name == 'class_field':
            return source.get('class')
            
        # Convert camelCase to snake_case for FHIR compatibility
        fhir_field_name = ''.join(['_' + c.lower() if c.isupper() else c for c in field_name]).lstrip('_')
        
        # Try to get the field from the source
        value = source.get(fhir_field_name)
        if value is not None:
            return value
            
        # Try camelCase as a fallback
        value = source.get(field_name)
        if value is not None:
            return value
            
        # Try with first letter capitalized as a fallback (e.g., resourceType)
        capitalized_field_name = field_name[0].upper() + field_name[1:]
        value = source.get(capitalized_field_name)
        if value is not None:
            return value
            
        # Return None if the field is not found
        return None
    
    @classmethod
    def create_resolver_for_type(cls, graphql_type: Type[T]) -> Any:
        """
        Create a resolver for a GraphQL type.
        
        Args:
            graphql_type: The GraphQL type
            
        Returns:
            A resolver function for the type
        """
        def resolver(root: Any, info: strawberry.types.Info, **kwargs: Any) -> T:
            """
            Resolver function for a GraphQL type.
            
            Args:
                root: The parent object
                info: The GraphQL info object
                kwargs: Additional arguments
                
            Returns:
                An instance of the GraphQL type
            """
            # If root is None, return None
            if root is None:
                return None
                
            # If root is already the correct type, return it
            if isinstance(root, graphql_type):
                return root
                
            # If root is a dictionary, create a new instance of the type
            if isinstance(root, dict):
                # Create a dictionary of field values
                field_values = {}
                
                # Get the fields from the GraphQL type
                try:
                    fields = graphql_type.__strawberry_definition__.fields
                except AttributeError:
                    logger.warning(f"Could not get fields from {graphql_type.__name__}")
                    return None
                    
                # Resolve each field
                for field in fields:
                    field_name = field.name
                    field_value = cls.resolve_field(root, field_name)
                    
                    # If the field is a list of objects, resolve each object
                    if isinstance(field_value, list) and field_value and isinstance(field_value[0], dict):
                        # Get the type of the list items
                        field_type = field.type
                        item_type = getattr(field_type, '__args__', [None])[0]
                        
                        # If the item type is a GraphQL type, resolve each item
                        if hasattr(item_type, '__strawberry_definition__'):
                            field_value = [cls.create_resolver_for_type(item_type)(item, info) for item in field_value]
                    
                    # If the field is an object, resolve it
                    elif isinstance(field_value, dict):
                        # Get the type of the field
                        field_type = field.type
                        
                        # If the field type is a GraphQL type, resolve it
                        if hasattr(field_type, '__strawberry_definition__'):
                            field_value = cls.create_resolver_for_type(field_type)(field_value, info)
                    
                    # Add the field value to the dictionary
                    field_values[field_name] = field_value
                
                # Create a new instance of the type
                return graphql_type(**field_values)
            
            # If root is not a dictionary, return None
            logger.warning(f"Could not resolve {graphql_type.__name__} from {type(root).__name__}")
            return None
        
        return resolver

# Create resolvers for common GraphQL types
def create_resolvers_for_types(*types: Type[T]) -> None:
    """
    Create resolvers for GraphQL types.
    
    Args:
        types: The GraphQL types to create resolvers for
    """
    for graphql_type in types:
        # Create a resolver for the type
        resolver = FHIRResolver.create_resolver_for_type(graphql_type)
        
        # Set the resolver for the type
        setattr(graphql_type, '__resolve__', resolver)
        
        # Log that the resolver was created
        logger.info(f"Created resolver for {graphql_type.__name__}")
