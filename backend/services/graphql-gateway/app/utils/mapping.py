"""
Utility functions for mapping between REST API and GraphQL.
"""
from typing import Any, Dict, List, Optional, Type, TypeVar, get_type_hints, get_origin, get_args
import inspect

T = TypeVar('T')

def snake_to_camel(snake_str: str) -> str:
    """
    Convert snake_case to camelCase.
    """
    components = snake_str.split('_')
    return components[0] + ''.join(x.title() for x in components[1:])

def camel_to_snake(camel_str: str) -> str:
    """
    Convert camelCase to snake_case.
    """
    import re
    s1 = re.sub('(.)([A-Z][a-z]+)', r'\1_\2', camel_str)
    return re.sub('([a-z0-9])([A-Z])', r'\1_\2', s1).lower()

def map_dict_to_type(data: Dict[str, Any], target_type: Type[T]) -> T:
    """
    Map a dictionary to a strawberry type.
    """
    if data is None:
        return None

    # Get type hints for the target type
    type_hints = get_type_hints(target_type)

    # Get field names and their strawberry field names
    field_names = {}
    for name, field in inspect.getmembers(target_type):
        if hasattr(field, '__strawberry_type__'):
            field_names[name] = getattr(field, 'name', name)

    # Create a dictionary with the correct field names
    kwargs = {}

    for field_name, field_type in type_hints.items():
        # Skip private fields
        if field_name.startswith('_'):
            continue

        # Get the strawberry field name if it exists
        # strawberry_name = field_names.get(field_name, field_name)

        # Check if the field exists in the data
        if field_name in data:
            value = data[field_name]
        elif snake_to_camel(field_name) in data:
            value = data[snake_to_camel(field_name)]
        elif camel_to_snake(field_name) in data:
            value = data[camel_to_snake(field_name)]
        else:
            # Field not found, use default value
            continue

        # Handle nested types
        origin = get_origin(field_type)
        args = get_args(field_type)

        if origin is list or origin is List:
            # Handle list of items
            if args and value is not None and isinstance(value, list):
                item_type = args[0]
                try:
                    # Try to map each item
                    kwargs[field_name] = [map_dict_to_type(item, item_type) if isinstance(item, dict) else item for item in value]
                except Exception as e:
                    # If mapping fails, use the value as is
                    print(f"Error mapping list item: {str(e)}")
                    kwargs[field_name] = value
            else:
                kwargs[field_name] = value
        elif origin is Optional:
            # Handle optional types
            if args and value is not None:
                item_type = args[0]
                try:
                    # Try to map the value
                    kwargs[field_name] = map_dict_to_type(value, item_type) if isinstance(value, dict) else value
                except Exception as e:
                    # If mapping fails, use the value as is
                    print(f"Error mapping optional value: {str(e)}")
                    kwargs[field_name] = value
            else:
                kwargs[field_name] = value
        else:
            # Try to map the value if it's a dict and the field type has annotations
            try:
                if isinstance(value, dict) and hasattr(field_type, '__annotations__'):
                    kwargs[field_name] = map_dict_to_type(value, field_type)
                else:
                    kwargs[field_name] = value
            except Exception as e:
                # If mapping fails, use the value as is
                print(f"Error mapping value: {str(e)}")
                kwargs[field_name] = value

    # Create an instance of the target type
    return target_type(**kwargs)

def map_rest_to_graphql(data: Dict[str, Any], target_type: Type[T]) -> T:
    """
    Map a REST API response to a GraphQL type.
    """
    if data is None:
        return None

    # If the data is a list, map each item
    if isinstance(data, list):
        return [map_dict_to_type(item, target_type) for item in data]

    # Otherwise, map the single item
    return map_dict_to_type(data, target_type)
