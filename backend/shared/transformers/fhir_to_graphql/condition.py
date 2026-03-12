"""
Transformer for FHIR Condition to GraphQL ConditionType.
"""

import logging
from typing import Any, Dict, List, Optional

from shared.models import Condition

from ..base import TransformerRegistry
from .base import FHIRToGraphQLTransformer

# Set up logging
logger = logging.getLogger(__name__)

# Import the GraphQL type dynamically to avoid circular imports
# In a real implementation, you would import the actual GraphQL type
# from app.graphql.types import ConditionType
ConditionType = type('ConditionType', (), {})

class ConditionTransformer(FHIRToGraphQLTransformer[Condition, ConditionType]):
    """
    Transformer for FHIR Condition to GraphQL ConditionType.
    """
    
    source_type = Condition
    target_type = ConditionType
    
    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the Condition data.
        
        Args:
            data: The Condition data to transform
            
        Returns:
            The transformed data
        """
        # Transform clinicalStatus
        if 'clinicalStatus' in data:
            data['clinicalStatus'] = self._transform_codeable_concept(data['clinicalStatus'])
        
        # Transform verificationStatus
        if 'verificationStatus' in data:
            data['verificationStatus'] = self._transform_codeable_concept(data['verificationStatus'])
        
        # Transform category
        if 'category' in data and isinstance(data['category'], list):
            data['category'] = [self._transform_codeable_concept(cat) for cat in data['category']]
        
        # Transform severity
        if 'severity' in data:
            data['severity'] = self._transform_codeable_concept(data['severity'])
        
        # Transform code
        if 'code' in data:
            data['code'] = self._transform_codeable_concept(data['code'])
        
        # Transform subject
        if 'subject' in data:
            data['subject'] = self._transform_reference(data['subject'])
        
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

# Register the transformer
TransformerRegistry.register(ConditionTransformer)
