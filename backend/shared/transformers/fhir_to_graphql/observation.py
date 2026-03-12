"""
Transformer for FHIR Observation to GraphQL ObservationType.
"""

import logging
from typing import Any, Dict, List, Optional

from shared.models import Observation

from ..base import TransformerRegistry
from .base import FHIRToGraphQLTransformer

# Set up logging
logger = logging.getLogger(__name__)

# Import the GraphQL type dynamically to avoid circular imports
# In a real implementation, you would import the actual GraphQL type
# from app.graphql.types import ObservationType
ObservationType = type('ObservationType', (), {})

class ObservationTransformer(FHIRToGraphQLTransformer[Observation, ObservationType]):
    """
    Transformer for FHIR Observation to GraphQL ObservationType.
    """
    
    source_type = Observation
    target_type = ObservationType
    
    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the Observation data.
        
        Args:
            data: The Observation data to transform
            
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
        if 'valueQuantity' in data:
            data['valueQuantity'] = self._transform_quantity(data['valueQuantity'])
        
        return data
    
    def _transform_codeable_concept(self, codeable_concept: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform CodeableConcept objects.
        
        Args:
            codeable_concept: CodeableConcept dictionary
            
        Returns:
            Transformed CodeableConcept dictionary
        """
        # Create a copy of the codeable_concept dictionary
        transformed_concept = dict(codeable_concept)
        
        # Transform coding
        if 'coding' in transformed_concept and isinstance(transformed_concept['coding'], list):
            transformed_codings = []
            
            for coding in transformed_concept['coding']:
                # Create a copy of the coding dictionary
                transformed_coding = dict(coding)
                
                # Add a displayWithCode field that combines code and display
                if 'code' in coding and 'display' in coding:
                    transformed_coding['displayWithCode'] = f"{coding['code']} - {coding['display']}"
                
                transformed_codings.append(transformed_coding)
            
            transformed_concept['coding'] = transformed_codings
        
        return transformed_concept
    
    def _transform_reference(self, reference: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform Reference objects.
        
        Args:
            reference: Reference dictionary
            
        Returns:
            Transformed Reference dictionary
        """
        # Create a copy of the reference dictionary
        transformed_reference = dict(reference)
        
        # Add a resourceType field based on the reference
        if 'reference' in transformed_reference and isinstance(transformed_reference['reference'], str):
            # Extract the resource type from the reference (e.g., "Patient/123" -> "Patient")
            parts = transformed_reference['reference'].split('/')
            if len(parts) > 0:
                transformed_reference['resourceType'] = parts[0]
        
        return transformed_reference
    
    def _transform_quantity(self, quantity: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform Quantity objects.
        
        Args:
            quantity: Quantity dictionary
            
        Returns:
            Transformed Quantity dictionary
        """
        # Create a copy of the quantity dictionary
        transformed_quantity = dict(quantity)
        
        # Add a formatted field that combines value and unit
        if 'value' in transformed_quantity and 'unit' in transformed_quantity:
            transformed_quantity['formatted'] = f"{transformed_quantity['value']} {transformed_quantity['unit']}"
        
        return transformed_quantity

# Register the transformer
TransformerRegistry.register(ObservationTransformer)
