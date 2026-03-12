"""
Transformer utilities for GraphQL gateway.

This module provides utility functions for using the Data Transformation Layer (DTL)
to convert between FHIR and GraphQL formats.
"""

import logging
from typing import Any, Dict, List, Optional, Type, TypeVar, Union

# Import DTL transformers
try:
    from shared.transformers import (
        TransformerRegistry, TransformationError,
        PatientTransformer, ObservationTransformer, ConditionTransformer,
        PatientInputTransformer, ObservationInputTransformer, ConditionInputTransformer
    )
    DTL_AVAILABLE = True
except ImportError:
    # Fallback if DTL is not available
    DTL_AVAILABLE = False

# Set up logging
logger = logging.getLogger(__name__)

# Type variable for target type
T = TypeVar('T')

def transform_fhir_to_graphql(fhir_data: Union[Dict[str, Any], List[Dict[str, Any]]], target_type: Type[T]) -> Union[T, List[T], None]:
    """
    Transform FHIR data to GraphQL type using the DTL.
    
    Args:
        fhir_data: FHIR resource data (single resource or list of resources)
        target_type: The GraphQL type class
        
    Returns:
        An instance of the target class, a list of instances, or None if transformation fails
    """
    if fhir_data is None:
        return None
        
    # If DTL is not available, use fallback conversion
    if not DTL_AVAILABLE:
        return _fallback_convert_fhir_to_graphql(fhir_data, target_type)
    
    try:
        # Handle list of resources
        if isinstance(fhir_data, list):
            result = []
            for item in fhir_data:
                transformed = transform_fhir_to_graphql(item, target_type)
                if transformed:
                    result.append(transformed)
            return result
            
        # Get the appropriate transformer based on resource type
        resource_type = fhir_data.get("resourceType")
        if not resource_type:
            logger.warning(f"Missing resourceType in FHIR data: {fhir_data}")
            return _fallback_convert_fhir_to_graphql(fhir_data, target_type)
            
        # Use the transformer registry to find the right transformer
        try:
            # Try to get a transformer for this specific resource type
            transformer = TransformerRegistry.get(Dict[str, Any], target_type)
            if transformer:
                return transformer.transform(fhir_data)
                
            # If no specific transformer is found, use a generic approach
            logger.info(f"No specific transformer found for {resource_type} to {target_type.__name__}, using fallback")
            return _fallback_convert_fhir_to_graphql(fhir_data, target_type)
        except TransformationError as e:
            logger.error(f"Transformation error: {str(e)}")
            # Fall back to manual conversion
            return _fallback_convert_fhir_to_graphql(fhir_data, target_type)
    except Exception as e:
        logger.exception(f"Error transforming FHIR data to {target_type.__name__}: {str(e)}")
        return None

def transform_graphql_to_fhir(graphql_data: Any, resource_type: str) -> Optional[Dict[str, Any]]:
    """
    Transform GraphQL input data to FHIR format using the DTL.
    
    Args:
        graphql_data: GraphQL input data
        resource_type: The FHIR resource type
        
    Returns:
        FHIR resource data or None if transformation fails
    """
    if graphql_data is None:
        return None
        
    # If DTL is not available, use fallback conversion
    if not DTL_AVAILABLE:
        return _fallback_convert_graphql_to_fhir(graphql_data, resource_type)
    
    try:
        # Use the transformer registry to find the right transformer
        try:
            # Try to get a transformer for this specific input type
            transformer = TransformerRegistry.get(type(graphql_data), Dict[str, Any])
            if transformer:
                result = transformer.transform(graphql_data)
                # Ensure resourceType is set
                if result and "resourceType" not in result:
                    result["resourceType"] = resource_type
                return result
                
            # If no specific transformer is found, use a generic approach
            logger.info(f"No specific transformer found for {type(graphql_data).__name__} to {resource_type}, using fallback")
            return _fallback_convert_graphql_to_fhir(graphql_data, resource_type)
        except TransformationError as e:
            logger.error(f"Transformation error: {str(e)}")
            # Fall back to manual conversion
            return _fallback_convert_graphql_to_fhir(graphql_data, resource_type)
    except Exception as e:
        logger.exception(f"Error transforming GraphQL data to {resource_type}: {str(e)}")
        return None

def _fallback_convert_fhir_to_graphql(fhir_data: Union[Dict[str, Any], List[Dict[str, Any]]], target_type: Type[T]) -> Union[T, List[T], None]:
    """
    Fallback method to convert FHIR data to GraphQL type without using the DTL.
    
    Args:
        fhir_data: FHIR resource data (single resource or list of resources)
        target_type: The GraphQL type class
        
    Returns:
        An instance of the target class, a list of instances, or None if conversion fails
    """
    if fhir_data is None:
        return None
        
    try:
        # Handle list of resources
        if isinstance(fhir_data, list):
            result = []
            for item in fhir_data:
                converted = _fallback_convert_fhir_to_graphql(item, target_type)
                if converted:
                    result.append(converted)
            return result
            
        # Create an instance of the target type with the FHIR data
        # This assumes the GraphQL type has field names matching the FHIR data
        # with camelCase convention
        kwargs = {}
        for key, value in fhir_data.items():
            # Skip None values
            if value is None:
                continue
                
            # Handle nested objects based on their type
            if isinstance(value, dict):
                # For now, just pass through dictionaries
                kwargs[key] = value
            elif isinstance(value, list) and value and isinstance(value[0], dict):
                # For lists of dictionaries, just pass through
                kwargs[key] = value
            else:
                # For primitive values, just pass through
                kwargs[key] = value
                
        # Create the instance
        return target_type(**kwargs)
    except Exception as e:
        logger.exception(f"Error in fallback conversion from FHIR to {target_type.__name__}: {str(e)}")
        return None

def _fallback_convert_graphql_to_fhir(graphql_data: Any, resource_type: str) -> Optional[Dict[str, Any]]:
    """
    Fallback method to convert GraphQL input data to FHIR format without using the DTL.
    
    Args:
        graphql_data: GraphQL input data
        resource_type: The FHIR resource type
        
    Returns:
        FHIR resource data or None if conversion fails
    """
    if graphql_data is None:
        return None
        
    try:
        # Convert the GraphQL input to a dictionary
        if hasattr(graphql_data, "__dict__"):
            result = graphql_data.__dict__.copy()
        else:
            # Try to convert to a dictionary
            result = dict(graphql_data)
            
        # Ensure resourceType is set
        result["resourceType"] = resource_type
        
        return result
    except Exception as e:
        logger.exception(f"Error in fallback conversion from GraphQL to {resource_type}: {str(e)}")
        return None
