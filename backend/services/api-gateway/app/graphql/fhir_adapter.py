"""
FHIR Plugin Adapter for GraphQL.

This module provides an adapter for the FHIR plugin to automatically map FHIR resources to GraphQL types.
"""

import logging
import inspect
from typing import Any, Dict, List, Optional, Type, TypeVar, Union, get_type_hints
import strawberry
from strawberry.types import Info

# Set up logging
logger = logging.getLogger(__name__)

T = TypeVar('T')

class FHIRAdapter:
    """
    Adapter for the FHIR plugin to automatically map FHIR resources to GraphQL types.
    """

    @staticmethod
    def is_fhir_resource(data: Any) -> bool:
        """
        Check if the data is a FHIR resource.

        Args:
            data: The data to check

        Returns:
            True if the data is a FHIR resource, False otherwise
        """
        if not isinstance(data, dict):
            return False

        # Check if the data has a resourceType field
        return 'resourceType' in data

    @staticmethod
    def convert_field_name(name: str) -> str:
        """
        Convert a field name from camelCase to snake_case.

        Args:
            name: The field name to convert

        Returns:
            The converted field name
        """
        # Handle special case for 'class' field
        if name == 'class':
            return 'class_field'

        # Convert camelCase to snake_case
        result = ''.join(['_' + c.lower() if c.isupper() else c for c in name]).lstrip('_')

        # Handle special case for 'class' field in FHIR
        if result == 'class':
            return 'class_field'

        return result

    @staticmethod
    def get_field_value(data: Dict[str, Any], field_name: str) -> Any:
        """
        Get a field value from FHIR data.

        Args:
            data: The FHIR data
            field_name: The name of the field to get

        Returns:
            The field value
        """
        # Try the field name as is
        if field_name in data:
            return data[field_name]

        # Try the converted field name
        converted_name = FHIRAdapter.convert_field_name(field_name)
        if converted_name in data:
            return data[converted_name]

        # Try with first letter capitalized
        capitalized_name = field_name[0].upper() + field_name[1:]
        if capitalized_name in data:
            return data[capitalized_name]

        # Try with camelCase
        if field_name == 'class_field':
            if 'class' in data:
                return data['class']

        # Return None if the field is not found
        return None

    @classmethod
    def create_resolver(cls, graphql_type: Type[T]) -> Any:
        """
        Create a resolver for a GraphQL type.

        Args:
            graphql_type: The GraphQL type

        Returns:
            A resolver function for the type
        """
        def resolver(root: Any, info: Info, **kwargs: Any) -> Optional[T]:
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

            # If root is a coroutine, it's an async function that needs to be awaited
            # This should be handled by the caller
            if inspect.iscoroutine(root):
                logger.warning(f"Received coroutine for {graphql_type.__name__}, this should be awaited by the caller")
                return None

            # If root is already the correct type, return it
            if isinstance(root, graphql_type):
                return root

            # If root is a FHIR resource, convert it to the GraphQL type
            if cls.is_fhir_resource(root):
                try:
                    # Get the field names from the GraphQL type
                    field_names = [f.name for f in graphql_type.__strawberry_definition__.fields]

                    # Create a dictionary of field values
                    field_values = {}
                    for field_name in field_names:
                        field_value = cls.get_field_value(root, field_name)
                        if field_value is not None:
                            field_values[field_name] = field_value

                    # Create an instance of the GraphQL type
                    return graphql_type(**field_values)
                except Exception as e:
                    logger.error(f"Error converting FHIR resource to {graphql_type.__name__}: {str(e)}")
                    return None

            # If root is not a FHIR resource, return None
            return None

        return resolver

def apply_fhir_resolvers(*types: Type[T]) -> None:
    """
    Apply FHIR resolvers to GraphQL types.

    Args:
        types: The GraphQL types to apply resolvers to
    """
    for graphql_type in types:
        # Create a resolver for the type
        resolver = FHIRAdapter.create_resolver(graphql_type)

        # Apply the resolver to the type
        strawberry.field(resolver=resolver)(graphql_type)

        # Log that the resolver was applied
        logger.info(f"Applied FHIR resolver to {graphql_type.__name__}")

def register_fhir_plugin() -> None:
    """
    Register the FHIR plugin with Strawberry.

    This function should be called before creating the GraphQL schema.
    """
    # Import GraphQL types
    from .types import (
        Patient, HumanName, Identifier, ContactPoint, Address,
        CodeableConcept, Coding, Reference, Quantity, Period,
        LabResult, Condition, MedicationRequest, DiagnosticReport,
        Encounter, DocumentReference, PatientTimeline
    )

    # Apply FHIR resolvers to GraphQL types
    apply_fhir_resolvers(
        Patient, HumanName, Identifier, ContactPoint, Address,
        CodeableConcept, Coding, Reference, Quantity, Period,
        LabResult, Condition, MedicationRequest, DiagnosticReport,
        Encounter, DocumentReference, PatientTimeline
    )

    logger.info("Registered FHIR plugin with Strawberry")
