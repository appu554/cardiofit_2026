"""
GraphQL type extensions for the Observation service.

This module contains additional GraphQL types and utilities that extend the base types
for more specific use cases in the Observation service.
"""

from typing import List, Optional, Dict, Any
import graphene
from datetime import datetime
from .types import (
    CodeableConcept,
    Reference,
    Quantity,
    Range,
    Ratio,
    Period,
    Annotation,
    ReferenceRange,
    Component,
    Observation
)

class ObservationConnection(graphene.relay.Connection):
    """Connection type for Observation with pagination support."""
    class Meta:
        node = Observation
    
    total_count = graphene.Int(description="Total number of observations matching the filter")
    
    def resolve_total_count(self, info, **kwargs):
        return self.iterable.count()

class ObservationFilterInput(graphene.InputObjectType):
    """Input type for filtering observations."""
    patient_id = graphene.ID(description="Filter by patient ID")
    category = graphene.String(description="Filter by observation category")
    code = graphene.String(description="Filter by observation code")
    date = graphene.String(description="Filter by date (YYYY-MM-DD)")
    status = graphene.String(description="Filter by status")
    _page = graphene.Int(description="Page number (1-based)", default_value=1)
    _count = graphene.Int(description="Number of items per page", default_value=10)

class ObservationOrderByInput(graphene.InputObjectType):
    """Input type for sorting observations."""
    field = graphene.String(required=True, description="Field to sort by")
    direction = graphene.String(default_value="ASC", description="Sort direction (ASC or DESC)")

class ObservationEdge(graphene.ObjectType):
    """Edge type for Observation connection."""
    node = graphene.Field(Observation)
    cursor = graphene.String(required=True)

class PageInfo(graphene.ObjectType):
    """Pagination information."""
    has_next_page = graphene.Boolean(required=True)
    has_previous_page = graphene.Boolean(required=True)
    start_cursor = graphene.String()
    end_cursor = graphene.String()
    total_count = graphene.Int(required=True)

class ObservationResponse(graphene.ObjectType):
    """Response type for observation operations."""
    success = graphene.Boolean(required=True)
    message = graphene.String()
    observation = graphene.Field(Observation)
    errors = graphene.List(graphene.String)

class CodeableConceptInput(graphene.InputObjectType):
    """Input type for CodeableConcept."""
    coding = graphene.List(lambda: CodingInput)
    text = graphene.String()

class CodingInput(graphene.InputObjectType):
    """Input type for Coding."""
    system = graphene.String()
    code = graphene.String()
    display = graphene.String()
    version = graphene.String()
    user_selected = graphene.Boolean(name="userSelected")

class ReferenceInput(graphene.InputObjectType):
    """Input type for Reference."""
    reference = graphene.String()
    type = graphene.String()
    display = graphene.String()
    identifier = graphene.Field(lambda: IdentifierInput)

class IdentifierInput(graphene.InputObjectType):
    """Input type for Identifier."""
    use = graphene.String()
    type = graphene.Field(CodeableConceptInput)
    system = graphene.String()
    value = graphene.String()
    period = graphene.Field(Period)
    assigner = graphene.Field(ReferenceInput)

class QuantityInput(graphene.InputObjectType):
    """Input type for Quantity."""
    value = graphene.Float()
    unit = graphene.String()
    system = graphene.String()
    code = graphene.String()

class RangeInput(graphene.InputObjectType):
    """Input type for Range."""
    low = graphene.Field(QuantityInput)
    high = graphene.Field(QuantityInput)

class RatioInput(graphene.InputObjectType):
    """Input type for Ratio."""
    numerator = graphene.Field(QuantityInput)
    denominator = graphene.Field(QuantityInput)

class PeriodInput(graphene.InputObjectType):
    """Input type for Period."""
    start = graphene.DateTime()
    end = graphene.DateTime()

class AnnotationInput(graphene.InputObjectType):
    """Input type for Annotation."""
    author_reference = graphene.Field(ReferenceInput, name="authorReference")
    author_string = graphene.String(name="authorString")
    time = graphene.DateTime()
    text = graphene.String(required=True)

class ReferenceRangeInput(graphene.InputObjectType):
    """Input type for ReferenceRange."""
    low = graphene.Field(QuantityInput)
    high = graphene.Field(QuantityInput)
    type = graphene.Field(CodeableConceptInput)
    applies_to = graphene.List(CodeableConceptInput, name="appliesTo")
    age = graphene.Field(RangeInput)
    text = graphene.String()

class ComponentInput(graphene.InputObjectType):
    """Input type for Component."""
    code = graphene.Field(CodeableConceptInput, required=True)
    value_quantity = graphene.Field(QuantityInput, name="valueQuantity")
    value_codeable_concept = graphene.Field(CodeableConceptInput, name="valueCodeableConcept")
    value_string = graphene.String(name="valueString")
    value_boolean = graphene.Boolean(name="valueBoolean")
    value_integer = graphene.Int(name="valueInteger")
    value_range = graphene.Field(RangeInput, name="valueRange")
    value_ratio = graphene.Field(RatioInput, name="valueRatio")
    value_sampled_data = graphene.String(name="valueSampledData")  # Simplified for now
    value_time = graphene.String(name="valueTime")
    value_date_time = graphene.DateTime(name="valueDateTime")
    value_period = graphene.Field(PeriodInput, name="valuePeriod")
    data_absent_reason = graphene.Field(CodeableConceptInput, name="dataAbsentReason")
    interpretation = graphene.List(CodeableConceptInput)
    reference_range = graphene.List(ReferenceRangeInput, name="referenceRange")

class CreateObservationInput(graphene.InputObjectType):
    """Input type for creating a new observation."""
    resource_type = graphene.String(default_value="Observation")
    identifier = graphene.List(IdentifierInput)
    based_on = graphene.List(ReferenceInput, name="basedOn")
    part_of = graphene.List(ReferenceInput, name="partOf")
    status = graphene.String(required=True)
    category = graphene.List(CodeableConceptInput)
    code = graphene.Field(CodeableConceptInput, required=True)
    subject = graphene.Field(ReferenceInput, required=True)
    focus = graphene.List(ReferenceInput)
    encounter = graphene.Field(ReferenceInput)
    effective_date_time = graphene.DateTime(name="effectiveDateTime")
    effective_period = graphene.Field(PeriodInput, name="effectivePeriod")
    effective_timing = graphene.String(name="effectiveTiming")  # Simplified for now
    effective_instant = graphene.String(name="effectiveInstant")
    issued = graphene.DateTime()
    performer = graphene.List(ReferenceInput)
    value_quantity = graphene.Field(QuantityInput, name="valueQuantity")
    value_codeable_concept = graphene.Field(CodeableConceptInput, name="valueCodeableConcept")
    value_string = graphene.String(name="valueString")
    value_boolean = graphene.Boolean(name="valueBoolean")
    value_integer = graphene.Int(name="valueInteger")
    value_range = graphene.Field(RangeInput, name="valueRange")
    value_ratio = graphene.Field(RatioInput, name="valueRatio")
    value_sampled_data = graphene.String(name="valueSampledData")  # Simplified for now
    value_time = graphene.String(name="valueTime")
    value_date_time = graphene.DateTime(name="valueDateTime")
    value_period = graphene.Field(PeriodInput, name="valuePeriod")
    data_absent_reason = graphene.Field(CodeableConceptInput, name="dataAbsentReason")
    interpretation = graphene.List(CodeableConceptInput)
    note = graphene.List(AnnotationInput)
    body_site = graphene.Field(CodeableConceptInput, name="bodySite")
    method = graphene.Field(CodeableConceptInput)
    specimen = graphene.Field(ReferenceInput)
    device = graphene.Field(ReferenceInput)
    reference_range = graphene.List(ReferenceRangeInput, name="referenceRange")
    has_member = graphene.List(ReferenceInput, name="hasMember")
    derived_from = graphene.List(ReferenceInput, name="derivedFrom")
    component = graphene.List(ComponentInput)

class UpdateObservationInput(graphene.InputObjectType):
    """Input type for updating an existing observation."""
    status = graphene.String()
    category = graphene.List(CodeableConceptInput)
    code = graphene.Field(CodeableConceptInput)
    subject = graphene.Field(ReferenceInput)
    effective_date_time = graphene.DateTime(name="effectiveDateTime")
    effective_period = graphene.Field(PeriodInput, name="effectivePeriod")
    issued = graphene.DateTime()
    performer = graphene.List(ReferenceInput)
    value_quantity = graphene.Field(QuantityInput, name="valueQuantity")
    value_codeable_concept = graphene.Field(CodeableConceptInput, name="valueCodeableConcept")
    value_string = graphene.String(name="valueString")
    value_boolean = graphene.Boolean(name="valueBoolean")
    value_integer = graphene.Int(name="valueInteger")
    value_range = graphene.Field(RangeInput, name="valueRange")
    value_ratio = graphene.Field(RatioInput, name="valueRatio")
    note = graphene.List(AnnotationInput)
    reference_range = graphene.List(ReferenceRangeInput, name="referenceRange")
    component = graphene.List(ComponentInput)

# Add these types to the GraphQL schema
class Query(graphene.ObjectType):
    """Root query type for the Observation service."""
    node = graphene.relay.Node.Field()
    
    # Add any additional queries here

class Mutation(graphene.ObjectType):
    """Root mutation type for the Observation service."""
    # Add mutations here
    pass

# Create the schema
schema = graphene.Schema(query=Query, mutation=Mutation, types=[
    ObservationConnection,
    ObservationEdge,
    PageInfo,
    ObservationResponse,
    # Add any additional types here
])
