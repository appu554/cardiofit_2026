"""
Transformer for FHIR Patient to GraphQL PatientType.
"""

import logging
from typing import Any, Dict, List, Optional

from shared.models import Patient

from ..base import TransformerRegistry
from .base import FHIRToGraphQLTransformer

# Set up logging
logger = logging.getLogger(__name__)

# Import the GraphQL type dynamically to avoid circular imports
# In a real implementation, you would import the actual GraphQL type
# from app.graphql.types import PatientType
class PatientType:
    def __init__(self, **kwargs):
        for key, value in kwargs.items():
            setattr(self, key, value)

class PatientTransformer(FHIRToGraphQLTransformer[Patient, PatientType]):
    """
    Transformer for FHIR Patient to GraphQL PatientType.
    """

    source_type = Patient
    target_type = PatientType

    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the Patient data.

        Args:
            data: The Patient data to transform

        Returns:
            The transformed data
        """
        # Transform name
        if 'name' in data:
            data['name'] = self._transform_name(data['name'])

        # Transform telecom
        if 'telecom' in data:
            data['telecom'] = self._transform_telecom(data['telecom'])

        # Transform address
        if 'address' in data:
            data['address'] = self._transform_address(data['address'])

        return data

    def _transform_name(self, name_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform HumanName objects.

        Args:
            name_list: List of HumanName dictionaries

        Returns:
            Transformed list of HumanName dictionaries
        """
        transformed_names = []

        for name in name_list:
            # Create a copy of the name dictionary
            transformed_name = dict(name)

            # Handle given names (convert from list to string if needed)
            if 'given' in transformed_name and isinstance(transformed_name['given'], list):
                # Join multiple given names with a space
                transformed_name['givenName'] = ' '.join(transformed_name['given'])
                del transformed_name['given']

            # Rename family to familyName for GraphQL
            if 'family' in transformed_name:
                transformed_name['familyName'] = transformed_name['family']
                del transformed_name['family']

            transformed_names.append(transformed_name)

        return transformed_names

    def _transform_telecom(self, telecom_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform ContactPoint objects.

        Args:
            telecom_list: List of ContactPoint dictionaries

        Returns:
            Transformed list of ContactPoint dictionaries
        """
        transformed_telecoms = []

        for telecom in telecom_list:
            # Create a copy of the telecom dictionary
            transformed_telecom = dict(telecom)

            # Add a displayValue field that combines system and value
            if 'system' in telecom and 'value' in telecom:
                system = telecom['system'].capitalize() if telecom['system'] else ''
                transformed_telecom['displayValue'] = f"{system}: {telecom['value']}"

            transformed_telecoms.append(transformed_telecom)

        return transformed_telecoms

    def _transform_address(self, address_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform Address objects.

        Args:
            address_list: List of Address dictionaries

        Returns:
            Transformed list of Address dictionaries
        """
        transformed_addresses = []

        for address in address_list:
            # Create a copy of the address dictionary
            transformed_address = dict(address)

            # Handle address lines (convert from list to string if needed)
            if 'line' in transformed_address and isinstance(transformed_address['line'], list):
                # Join multiple lines with a newline character
                transformed_address['streetAddress'] = '\n'.join(transformed_address['line'])
                del transformed_address['line']

            # Add a formatted field that combines all address components
            formatted_parts = []

            if 'streetAddress' in transformed_address:
                formatted_parts.append(transformed_address['streetAddress'])
            elif 'line' in transformed_address:
                if isinstance(transformed_address['line'], list):
                    formatted_parts.append('\n'.join(transformed_address['line']))
                else:
                    formatted_parts.append(transformed_address['line'])

            city_state_zip = []
            if 'city' in transformed_address:
                city_state_zip.append(transformed_address['city'])
            if 'state' in transformed_address:
                city_state_zip.append(transformed_address['state'])
            if 'postalCode' in transformed_address:
                city_state_zip.append(transformed_address['postalCode'])

            if city_state_zip:
                formatted_parts.append(', '.join(city_state_zip))

            if 'country' in transformed_address:
                formatted_parts.append(transformed_address['country'])

            transformed_address['formatted'] = '\n'.join(formatted_parts)

            transformed_addresses.append(transformed_address)

        return transformed_addresses

# Register the transformer
TransformerRegistry.register(PatientTransformer)
