"""
Transformer utilities for GraphQL gateway.

This module provides utility functions for using the Data Transformation Layer (DTL)
to convert between FHIR and GraphQL formats.
"""

import logging
import sys
import os
from typing import Any, Dict, List, Optional, Type, TypeVar, Union

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '../../../../'))
if backend_dir not in sys.path:
    sys.path.append(backend_dir)

# Import DTL transformers
try:
    from shared.transformers import (
        TransformerRegistry, TransformationError
    )
    DTL_AVAILABLE = True
except ImportError as e:
    # Fallback if DTL is not available
    DTL_AVAILABLE = False
    logging.warning(f"DTL not available: {str(e)}")

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
        logger.warning("DTL not available, using fallback conversion")
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
            # Try to transform using the registry
            try:
                return TransformerRegistry.transform_fhir_to_graphql(fhir_data, target_type)
            except Exception as e:
                logger.warning(f"Error using TransformerRegistry: {str(e)}")

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
        logger.warning("DTL not available, using fallback conversion")
        return _fallback_convert_graphql_to_fhir(graphql_data, resource_type)

    try:
        # Use the transformer registry to find the right transformer
        try:
            # Try to transform using the registry
            try:
                result = TransformerRegistry.transform_graphql_to_fhir(graphql_data, resource_type)
                # Ensure resourceType is set
                if result and "resourceType" not in result:
                    result["resourceType"] = resource_type
                return result
            except Exception as e:
                logger.warning(f"Error using TransformerRegistry: {str(e)}")

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

        # Use the FHIR adapter to convert the FHIR data to a GraphQL type
        try:
            # Import the FHIR adapter
            from app.graphql.fhir_adapter import FHIRAdapter

            # Create a resolver for the target type
            resolver = FHIRAdapter.create_resolver(target_type)

            # Call the resolver with the FHIR data
            result = resolver(fhir_data, None)

            # If the resolver returned a result, return it
            if result is not None:
                return result
        except ImportError:
            logger.warning("Could not import FHIR adapter, falling back to manual conversion")
        except Exception as e:
            logger.warning(f"Error using FHIR adapter: {str(e)}")

        # If the FHIR adapter failed or is not available, fall back to manual conversion
        logger.info(f"Falling back to manual conversion for {target_type.__name__}")

        # Create an instance of the target type with the FHIR data
        # This assumes the GraphQL type has field names matching the FHIR data
        # with camelCase convention
        kwargs = {}

        # Get the fields defined in the target type
        try:
            target_fields = {field.name for field in target_type.__strawberry_definition__.fields}
            logger.debug(f"Available fields in {target_type.__name__}: {target_fields}")
        except Exception as e:
            logger.warning(f"Could not get fields from {target_type.__name__}: {str(e)}")
            target_fields = set()  # Empty set as fallback

        # Convert field names from camelCase to snake_case
        field_mapping = {}
        for field in target_fields:
            # Convert camelCase to snake_case
            snake_case = ''.join(['_' + c.lower() if c.isupper() else c for c in field]).lstrip('_')
            field_mapping[snake_case] = field
            # Also add the original field name
            field_mapping[field] = field
            # Handle special case for 'class' field
            if field == 'class_field':
                field_mapping['class'] = field

        # Process each field in the FHIR data
        for key, value in fhir_data.items():
            # Skip None values and MongoDB _id field
            if value is None or key == '_id':
                continue

            # Map the field name to the target field name
            target_key = field_mapping.get(key)
            if target_key is None:
                # Try with first letter capitalized
                capitalized_key = key[0].upper() + key[1:]
                target_key = field_mapping.get(capitalized_key)

            # Skip fields that aren't defined in the target type
            if target_key is None and key != 'id' and key != 'resourceType':
                logger.debug(f"Skipping field '{key}' as it's not defined in {target_type.__name__}")
                continue

            # Use the original key if no mapping is found
            if target_key is None:
                target_key = key

            # Add the field value to the kwargs
            kwargs[target_key] = value

        # Create the instance
        try:
            return target_type(**kwargs)
        except Exception as e:
            logger.exception(f"Error creating {target_type.__name__} instance: {str(e)}")
            logger.debug(f"kwargs: {kwargs}")
            # Try a more conservative approach with only basic fields
            basic_kwargs = {'id': fhir_data.get('id'), 'resourceType': fhir_data.get('resourceType', target_type.__name__)}
            return target_type(**basic_kwargs)
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
