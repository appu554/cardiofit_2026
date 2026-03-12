"""
Transformer for GraphQL ConditionInput to FHIR Condition.
"""

import logging
from typing import Any, Dict, List, Optional

from shared.models import Condition

from ..base import TransformerRegistry
from .base import GraphQLToFHIRTransformer

# Set up logging
logger = logging.getLogger(__name__)

# Import the GraphQL type dynamically to avoid circular imports
# In a real implementation, you would import the actual GraphQL type
# from app.graphql.types import ConditionInput
ConditionInput = type('ConditionInput', (), {})

class ConditionInputTransformer(GraphQLToFHIRTransformer[ConditionInput, Condition]):
    """
    Transformer for GraphQL ConditionInput to FHIR Condition.
    """
    
    source_type = ConditionInput
    target_type = Condition
    
    def _transform_nested_objects(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Transform nested objects in the ConditionInput data.
        
        Args:
            data: The ConditionInput data to transform
            
        Returns:
            The transformed data
        """
        # Transform clinicalStatus
        if 'clinical_status' in data:
            data['clinical_status'] = self._transform_codeable_concept(data['clinical_status'])
        
        # Transform verificationStatus
        if 'verification_status' in data:
            data['verification_status'] = self._transform_codeable_concept(data['verification_status'])
        
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
        
        # Ensure resourceType is set
        data['resource_type'] = 'Condition'
        
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

# Register the transformer
TransformerRegistry.register(ConditionInputTransformer)
