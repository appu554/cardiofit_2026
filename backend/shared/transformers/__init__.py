"""
Transformers package for Clinical Synthesis Hub.

This package provides transformers for converting between different data formats,
particularly between FHIR models and GraphQL types.
"""

from .base import BaseTransformer, TransformerRegistry
from .exceptions import TransformationError, ValidationError

# FHIR to GraphQL transformers
from .fhir_to_graphql import (
    FHIRToGraphQLTransformer,
    PatientTransformer,
    ObservationTransformer,
    ConditionTransformer
)

# GraphQL to FHIR transformers
from .graphql_to_fhir import (
    GraphQLToFHIRTransformer,
    PatientInputTransformer,
    ObservationInputTransformer,
    ConditionInputTransformer
)

__all__ = [
    # Base classes
    "BaseTransformer",
    "TransformerRegistry",
    
    # Exceptions
    "TransformationError",
    "ValidationError",
    
    # FHIR to GraphQL transformers
    "FHIRToGraphQLTransformer",
    "PatientTransformer",
    "ObservationTransformer",
    "ConditionTransformer",
    
    # GraphQL to FHIR transformers
    "GraphQLToFHIRTransformer",
    "PatientInputTransformer",
    "ObservationInputTransformer",
    "ConditionInputTransformer"
]
