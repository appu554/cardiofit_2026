"""
Transformer for GraphQL ObservationInput to FHIR Observation.
"""

import logging
from typing import Any, Dict, List, Optional

from shared.models import Observation

from ..base import TransformerRegistry
from .base import GraphQLToFHIRTransformer

# Set up logging
logger = logging.getLogger(__name__)

# Import the GraphQL type dynamically to avoid circular imports
# In a real implementation, you would import the actual GraphQL type
# from app.graphql.types import ObservationInput
ObservationInput = type('ObservationInput', (), {})

class ObservationInputTransformer(GraphQLToFHIRTransformer[ObservationInput, Observation]):
    """
    Transformer for GraphQL ObservationInput to FHIR Observation.
    """
    
    source_type = ObservationInput
    target_type = Observation
    
    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the ObservationInput data.
        
        Args:
            data: The ObservationInput data to transform
            
        Returns:
            The transformed data
        """
        # Transform code
        if 'code' in data:
            data['code'] = self._transform_codeable_concept(data['code'])
        
        # Transform subject
        if 'subject' in data:
            data['subject'] = self._transform_reference(data['subject'])
        
        # Transform valueQuantity
        if 'value_quantity' in data:
            data['value_quantity'] = self._transform_quantity(data['value_quantity'])
        
        # Ensure resourceType is set
        data['resource_type'] = 'Observation'
        
        return data
    
    def _transform_codeable_concept(self, codeable_concept: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform GraphQL CodeableConcept objects to FHIR CodeableConcept.
        
        Args:
            codeable_concept: GraphQL CodeableConcept dictionary
            
        Returns:
            Transformed FHIR CodeableConcept dictionary
        """
        # Create a copy of the codeable_concept dictionary
        transformed_concept = dict(codeable_concept)
        
        # Transform coding
        if 'coding' in transformed_concept and isinstance(transformed_concept['coding'], list):
            transformed_codings = []
            
            for coding in transformed_concept['coding']:
                # Create a copy of the coding dictionary
                transformed_coding = dict(coding)
                
                # Remove GraphQL-specific fields
                if 'display_with_code' in transformed_coding:
                    del transformed_coding['display_with_code']
                
                transformed_codings.append(transformed_coding)
            
            transformed_concept['coding'] = transformed_codings
        
        return transformed_concept
    
    def _transform_reference(self, reference: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform GraphQL Reference objects to FHIR Reference.
        
        Args:
            reference: GraphQL Reference dictionary
            
        Returns:
            Transformed FHIR Reference dictionary
        """
        # Create a copy of the reference dictionary
        transformed_reference = dict(reference)
        
        # Remove GraphQL-specific fields
        if 'resource_type' in transformed_reference:
            del transformed_reference['resource_type']
        
        return transformed_reference
    
    def _transform_quantity(self, quantity: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform GraphQL Quantity objects to FHIR Quantity.
        
        Args:
            quantity: GraphQL Quantity dictionary
            
        Returns:
            Transformed FHIR Quantity dictionary
        """
        # Create a copy of the quantity dictionary
        transformed_quantity = dict(quantity)
        
        # Remove GraphQL-specific fields
        if 'formatted' in transformed_quantity:
            del transformed_quantity['formatted']
        
        return transformed_quantity

# Register the transformer
TransformerRegistry.register(ObservationInputTransformer)
