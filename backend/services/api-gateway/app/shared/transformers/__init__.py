"""
Data Transformation Layer (DTL) for converting between FHIR and GraphQL types.
"""

from .base import BaseTransformer, TransformerRegistry, TransformationError
from .patient import PatientTransformer

# Register transformers
def register_transformers():
    """Register all transformers with the registry."""
    try:
        # Import here to avoid circular imports
        from app.graphql.types import Patient
        
        # Register Patient transformer
        TransformerRegistry.register_fhir_to_graphql("Patient", Patient, PatientTransformer)
        TransformerRegistry.register_graphql_to_fhir(Patient, "Patient", PatientTransformer)
        
        # Add more transformer registrations here as needed
        
    except ImportError as e:
        # This will happen when the transformers are imported outside of the API Gateway
        # For example, when imported by a microservice that doesn't have the GraphQL types
        print(f"Warning: Could not register transformers: {str(e)}")

# Export the classes and functions
__all__ = [
    'BaseTransformer',
    'TransformerRegistry',
    'TransformationError',
    'PatientTransformer',
    'register_transformers'
]
