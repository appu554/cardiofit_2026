"""
Transformer for GraphQL PatientInput to FHIR Patient.
"""

import logging
from typing import Any, Dict, List, Optional

from shared.models import Patient

from ..base import TransformerRegistry
from .base import GraphQLToFHIRTransformer

# Set up logging
logger = logging.getLogger(__name__)

# Import the GraphQL type dynamically to avoid circular imports
# In a real implementation, you would import the actual GraphQL type
# from app.graphql.types import PatientInput
class PatientInput:
    def __init__(self, **kwargs):
        for key, value in kwargs.items():
            setattr(self, key, value)

class PatientInputTransformer(GraphQLToFHIRTransformer[PatientInput, Patient]):
    """
    Transformer for GraphQL PatientInput to FHIR Patient.
    """

    source_type = PatientInput
    target_type = Patient

    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the PatientInput data.

        Args:
            data: The PatientInput data to transform

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

        # Ensure resourceType is set
        data['resource_type'] = 'Patient'

        return data

    def _transform_name(self, name_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform GraphQL HumanName objects to FHIR HumanName.

        Args:
            name_list: List of GraphQL HumanName dictionaries

        Returns:
            Transformed list of FHIR HumanName dictionaries
        """
        transformed_names = []

        for name in name_list:
            # Create a copy of the name dictionary
            transformed_name = dict(name)

            # Handle given name (convert from string to list if needed)
            if 'given_name' in transformed_name:
                # Split the given name by spaces to get a list of given names
                transformed_name['given'] = transformed_name['given_name'].split()
                del transformed_name['given_name']

            # Rename family_name to family for FHIR
            if 'family_name' in transformed_name:
                transformed_name['family'] = transformed_name['family_name']
                del transformed_name['family_name']

            transformed_names.append(transformed_name)

        return transformed_names

    def _transform_telecom(self, telecom_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform GraphQL ContactPoint objects to FHIR ContactPoint.

        Args:
            telecom_list: List of GraphQL ContactPoint dictionaries

        Returns:
            Transformed list of FHIR ContactPoint dictionaries
        """
        transformed_telecoms = []

        for telecom in telecom_list:
            # Create a copy of the telecom dictionary
            transformed_telecom = dict(telecom)

            # Remove GraphQL-specific fields
            if 'display_value' in transformed_telecom:
                del transformed_telecom['display_value']

            transformed_telecoms.append(transformed_telecom)

        return transformed_telecoms

    def _transform_address(self, address_list: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Transform GraphQL Address objects to FHIR Address.

        Args:
            address_list: List of GraphQL Address dictionaries

        Returns:
            Transformed list of FHIR Address dictionaries
        """
        transformed_addresses = []

        for address in address_list:
            # Create a copy of the address dictionary
            transformed_address = dict(address)

            # Handle street address (convert from string to list)
            if 'street_address' in transformed_address:
                # Split the street address by newlines to get a list of lines
                transformed_address['line'] = transformed_address['street_address'].split('\n')
                del transformed_address['street_address']

            # Remove GraphQL-specific fields
            if 'formatted' in transformed_address:
                del transformed_address['formatted']

            transformed_addresses.append(transformed_address)

        return transformed_addresses

# Register the transformer
TransformerRegistry.register(PatientInputTransformer)
