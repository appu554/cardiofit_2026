"""
Type conversion utilities for transformers.

This module provides functions for converting between different data types,
such as strings, dates, and complex objects.
"""

import datetime
from typing import Any, Dict, List, Optional, Type, TypeVar, Union, get_type_hints

T = TypeVar('T')

def convert_date(value: Any) -> Optional[datetime.date]:
    """
    Convert a value to a date.
    
    Args:
        value: The value to convert (string, date, or datetime)
        
    Returns:
        A date object, or None if the value is None
        
    Raises:
        ValueError: If the value cannot be converted to a date
    """
    if value is None:
        return None
    
    if isinstance(value, datetime.date):
        return value
    
    if isinstance(value, datetime.datetime):
        return value.date()
    
    if isinstance(value, str):
        return datetime.date.fromisoformat(value)
    
    raise ValueError(f"Cannot convert {type(value)} to date: {value}")

def convert_datetime(value: Any) -> Optional[datetime.datetime]:
    """
    Convert a value to a datetime.
    
    Args:
        value: The value to convert (string, date, or datetime)
        
    Returns:
        A datetime object, or None if the value is None
        
    Raises:
        ValueError: If the value cannot be converted to a datetime
    """
    if value is None:
        return None
    
    if isinstance(value, datetime.datetime):
        return value
    
    if isinstance(value, datetime.date):
        return datetime.datetime.combine(value, datetime.time())
    
    if isinstance(value, str):
        try:
            return datetime.datetime.fromisoformat(value)
        except ValueError:
            # Try parsing with different formats
            for fmt in ('%Y-%m-%dT%H:%M:%S.%fZ', '%Y-%m-%dT%H:%M:%SZ', '%Y-%m-%dT%H:%M:%S'):
                try:
                    return datetime.datetime.strptime(value, fmt)
                except ValueError:
                    continue
            
            # If we get here, none of the formats worked
            raise ValueError(f"Cannot parse datetime from string: {value}")
    
    raise ValueError(f"Cannot convert {type(value)} to datetime: {value}")

def convert_primitive(value: Any, target_type: Type[T]) -> Optional[T]:
    """
    Convert a primitive value to the target type.
    
    Args:
        value: The value to convert
        target_type: The target type
        
    Returns:
        The converted value, or None if the value is None
        
    Raises:
        ValueError: If the value cannot be converted to the target type
    """
    if value is None:
        return None
    
    # Handle date and datetime specially
    if target_type is datetime.date:
        return convert_date(value)
    
    if target_type is datetime.datetime:
        return convert_datetime(value)
    
    # Handle other primitive types
    if isinstance(value, target_type):
        return value
    
    # Try to convert using the target type's constructor
    try:
        return target_type(value)
    except (ValueError, TypeError) as e:
        raise ValueError(f"Cannot convert {type(value)} to {target_type.__name__}: {value}") from e

def convert_list(value: Any, item_type: Type[T]) -> Optional[List[T]]:
    """
    Convert a list of values to a list of the target type.
    
    Args:
        value: The list to convert
        item_type: The target type for list items
        
    Returns:
        A list of converted values, or None if the value is None
        
    Raises:
        ValueError: If the value cannot be converted to a list of the target type
    """
    if value is None:
        return None
    
    if not isinstance(value, (list, tuple)):
        raise ValueError(f"Expected list or tuple, got {type(value)}: {value}")
    
    return [convert_primitive(item, item_type) for item in value]

def convert_dict(value: Any, key_type: Type, value_type: Type[T]) -> Optional[Dict[Any, T]]:
    """
    Convert a dictionary to a dictionary with keys and values of the target types.
    
    Args:
        value: The dictionary to convert
        key_type: The target type for dictionary keys
        value_type: The target type for dictionary values
        
    Returns:
        A dictionary with converted keys and values, or None if the value is None
        
    Raises:
        ValueError: If the value cannot be converted to a dictionary of the target types
    """
    if value is None:
        return None
    
    if not isinstance(value, dict):
        raise ValueError(f"Expected dict, got {type(value)}: {value}")
    
    return {
        convert_primitive(k, key_type): convert_primitive(v, value_type)
        for k, v in value.items()
    }
