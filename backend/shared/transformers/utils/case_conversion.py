"""
Case conversion utilities for transformers.

This module provides functions for converting between different case styles,
such as snake_case, camelCase, and PascalCase.
"""

import re

def snake_to_camel(snake_str: str) -> str:
    """
    Convert snake_case to camelCase.
    
    Args:
        snake_str: A string in snake_case
        
    Returns:
        The string converted to camelCase
    """
    components = snake_str.split('_')
    return components[0] + ''.join(x.title() for x in components[1:])

def camel_to_snake(camel_str: str) -> str:
    """
    Convert camelCase to snake_case.
    
    Args:
        camel_str: A string in camelCase
        
    Returns:
        The string converted to snake_case
    """
    s1 = re.sub('(.)([A-Z][a-z]+)', r'\1_\2', camel_str)
    return re.sub('([a-z0-9])([A-Z])', r'\1_\2', s1).lower()

def snake_to_pascal(snake_str: str) -> str:
    """
    Convert snake_case to PascalCase.
    
    Args:
        snake_str: A string in snake_case
        
    Returns:
        The string converted to PascalCase
    """
    return ''.join(x.title() for x in snake_str.split('_'))

def pascal_to_snake(pascal_str: str) -> str:
    """
    Convert PascalCase to snake_case.
    
    Args:
        pascal_str: A string in PascalCase
        
    Returns:
        The string converted to snake_case
    """
    s1 = re.sub('(.)([A-Z][a-z]+)', r'\1_\2', pascal_str)
    return re.sub('([a-z0-9])([A-Z])', r'\1_\2', s1).lower()

def camel_to_pascal(camel_str: str) -> str:
    """
    Convert camelCase to PascalCase.
    
    Args:
        camel_str: A string in camelCase
        
    Returns:
        The string converted to PascalCase
    """
    return camel_str[0].upper() + camel_str[1:]

def pascal_to_camel(pascal_str: str) -> str:
    """
    Convert PascalCase to camelCase.
    
    Args:
        pascal_str: A string in PascalCase
        
    Returns:
        The string converted to camelCase
    """
    return pascal_str[0].lower() + pascal_str[1:]

def convert_keys(obj: dict, converter_func) -> dict:
    """
    Convert all keys in a dictionary using the given converter function.
    
    Args:
        obj: The dictionary to convert
        converter_func: The function to use for converting keys
        
    Returns:
        A new dictionary with converted keys
    """
    if not isinstance(obj, dict):
        return obj
    
    result = {}
    for key, value in obj.items():
        # Convert the key
        new_key = converter_func(key)
        
        # Recursively convert nested dictionaries
        if isinstance(value, dict):
            result[new_key] = convert_keys(value, converter_func)
        elif isinstance(value, list):
            result[new_key] = [
                convert_keys(item, converter_func) if isinstance(item, dict) else item
                for item in value
            ]
        else:
            result[new_key] = value
    
    return result

def snake_to_camel_keys(obj: dict) -> dict:
    """
    Convert all keys in a dictionary from snake_case to camelCase.
    
    Args:
        obj: The dictionary to convert
        
    Returns:
        A new dictionary with keys in camelCase
    """
    return convert_keys(obj, snake_to_camel)

def camel_to_snake_keys(obj: dict) -> dict:
    """
    Convert all keys in a dictionary from camelCase to snake_case.
    
    Args:
        obj: The dictionary to convert
        
    Returns:
        A new dictionary with keys in snake_case
    """
    return convert_keys(obj, camel_to_snake)
