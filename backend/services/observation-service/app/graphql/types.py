from __future__ import annotations

import strawberry
import logging
from typing import List, Optional, Any, Dict

# Configure logging
logger = logging.getLogger(__name__)

# from .identifier_type import Identifier # Now using LazyType
# from .reference_type import Reference # Now using LazyType

LazyIdentifier = strawberry.LazyType("Identifier", "app.graphql.identifier_type")
LazyReference = strawberry.LazyType("Reference", "app.graphql.reference_type")

# Placeholder for shared FHIR models if needed later
# from shared.models.fhir import (
#     Coding as CodingModel,
#     CodeableConcept as CodeableConceptModel,
#     Reference as ReferenceModel,
#     Quantity as QuantityModel,
#     Period as PeriodModel,
#     Annotation as AnnotationModel,
#     ObservationComponent as ObservationComponentModel,
#     ObservationReferenceRange as ObservationReferenceRangeModel
# )

# Base Strawberry types - Class definitions first
@strawberry.type(description="A time period defined by a start and end date/time.")
class Period:
    start: Optional[str] = strawberry.field(default=None, description="Starting time with inclusive boundary (dateTime).")
    end: Optional[str] = strawberry.field(default=None, description="End time with inclusive boundary, if not ongoing (dateTime).")

@strawberry.type(description="A coding defined by a terminology system.")
class Coding:
    system: Optional[str] = strawberry.field(default=None, description="Identity of the terminology system.")
    version: Optional[str] = strawberry.field(default=None, description="Version of the system - if relevant.")
    code: Optional[str] = strawberry.field(default=None, description="Symbol in syntax defined by the system.")
    display: Optional[str] = strawberry.field(default=None, description="Representation defined by the system.")
    user_selected: Optional[bool] = strawberry.field(default=None, description="If this coding was chosen directly by the user.")

@strawberry.type(description="A measured amount (or an amount that can potentially be measured).")
class Quantity:
    value: Optional[float] = strawberry.field(default=None, description="Numerical value (with implicit precision).")
    unit: Optional[str] = strawberry.field(default=None, description="Unit representation.")
    system: Optional[str] = strawberry.field(default=None, description="System that defines coded unit form.")
    code: Optional[str] = strawberry.field(default=None, description="Coded form of the unit.")

@strawberry.type(description="A concept that may be defined by a code from a terminology or by text.")
class CodeableConcept:
    coding: Optional[List['Coding']] = strawberry.field(default_factory=list, description="A coding representation of the concept.")
    text: Optional[str] = strawberry.field(default=None, description="Plain text representation of the concept.")

@strawberry.type(description="Text node with attribution.")
class Annotation:
    author_reference: Optional[LazyReference] = strawberry.field(default=None, description="Individual responsible for the annotation (if a resource).")
    author_string: Optional[str] = strawberry.field(default=None, description="Individual responsible for the annotation (if a string).")
    time: Optional[str] = strawberry.field(default=None, description="When the annotation was made (dateTime).")
    text: str = strawberry.field(description="The annotation - text content (markdown).")

@strawberry.federation.type(keys=["id"], description="Represents a FHIR Observation resource.")
class Observation:
    id: strawberry.ID = strawberry.field(description="Logical id of this artifact.")
    identifier: Optional[List[LazyIdentifier]] = strawberry.field(default_factory=list, description="Business Identifier for observation.")
    status: str = strawberry.field(description="registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown.")
    category: Optional[List['CodeableConcept']] = strawberry.field(default_factory=list, description="Classification of type of observation.")
    code: 'CodeableConcept' = strawberry.field(description="Type of observation (code / type).")
    subject: Optional[LazyReference] = strawberry.field(default=None, description="Who and/or what this is about (Patient, Group, Device, Location).")
    encounter: Optional[LazyReference] = strawberry.field(default=None, description="Healthcare event during which this observation is made.")
    effective_date_time: Optional[str] = strawberry.field(default=None, description="Clinically relevant date/time of observation.")
    effective_period: Optional['Period'] = strawberry.field(default=None, description="Clinically relevant time period of observation.")
    issued: Optional[str] = strawberry.field(default=None, description="Date/Time this version was made available (instant).")
    performer: Optional[List[LazyReference]] = strawberry.field(default_factory=list, description="Who is responsible for the observation.")
    value_quantity: Optional['Quantity'] = strawberry.field(default=None, description="Actual result as a Quantity.")
    value_codeable_concept: Optional['CodeableConcept'] = strawberry.field(default=None, description="Actual result as a CodeableConcept.")
    value_string: Optional[str] = strawberry.field(default=None, description="Actual result as a string.")
    value_boolean: Optional[bool] = strawberry.field(default=None, description="Actual result as a boolean.")
    value_integer: Optional[int] = strawberry.field(default=None, description="Actual result as an integer.")
    value_date_time: Optional[str] = strawberry.field(default=None, description="Actual result as a dateTime.")
    value_period: Optional['Period'] = strawberry.field(default=None, description="Actual result as a Period.")
    data_absent_reason: Optional['CodeableConcept'] = strawberry.field(default=None, description="Why the result is missing.")
    interpretation: Optional[List['CodeableConcept']] = strawberry.field(default_factory=list, description="High, low, normal, etc.")
    note: Optional[List['Annotation']] = strawberry.field(default_factory=list, description="Comments about the observation.")
    body_site: Optional['CodeableConcept'] = strawberry.field(default=None, description="Observed body part.")
    method: Optional['CodeableConcept'] = strawberry.field(default=None, description="How it was done.")

# Deferred from_fhir implementations
def _coding_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional['Coding']:
    if not fhir_data:
        return None
    return cls(
        system=fhir_data.get("system"),
        version=fhir_data.get("version"),
        code=fhir_data.get("code"),
        display=fhir_data.get("display"),
        user_selected=fhir_data.get("userSelected")
    )

def _codeable_concept_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional['CodeableConcept']:
    if not fhir_data:
        return None
    codings_data = fhir_data.get("coding")
    codings_list = []
    if codings_data:
        for coding_entry in codings_data:
            coding_obj = Coding.from_fhir(coding_entry) # type: ignore
            if coding_obj:
                codings_list.append(coding_obj)
    return cls(
        coding=codings_list,
        text=fhir_data.get("text")
    )

def _period_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional['Period']:
    if not fhir_data:
        return None
    return cls(
        start=fhir_data.get("start"),
        end=fhir_data.get("end")
    )

def _quantity_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional['Quantity']:
    if not fhir_data:
        return None
    return cls(
        value=fhir_data.get("value"),
        unit=fhir_data.get("unit"),
        system=fhir_data.get("system"),
        code=fhir_data.get("code")
    )

def _annotation_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional['Annotation']:
    if not fhir_data:
        return None
    author_ref_data = fhir_data.get("authorReference")
    author_str_data = fhir_data.get("authorString")
    return cls(
        author_reference=Reference.from_fhir(author_ref_data) if author_ref_data else None, # type: ignore
        author_string=author_str_data,
        time=fhir_data.get("time"),
        text=fhir_data.get("text")
    )

def _observation_from_fhir_impl(cls, fhir_observation: Optional[Dict[str, Any]]) -> Optional['Observation']:
    if not fhir_observation:
        return None

    def _map_list(fhir_list_data, mapper_func):
        if not fhir_list_data: return []
        # The mapper_func will be the .from_fhir of another class, which is now correctly typed
        return [item for item in (mapper_func(elem) for elem in fhir_list_data) if item is not None]

    fhir_id = fhir_observation.get("id")
    if not fhir_id:
        logger.warning("FHIR Observation is missing an ID. Using a placeholder.")
        fhir_id = "UNKNOWN_ID_OBSERVATION"
        
    fhir_status = fhir_observation.get("status")
    if not fhir_status:
        logger.error(f"FHIR Observation with ID '{fhir_id}' is missing mandatory 'status' field.")
        return None
            
    fhir_code = fhir_observation.get("code")
    if not fhir_code:
        logger.error(f"FHIR Observation with ID '{fhir_id}' is missing mandatory 'code' field.")
        return None
    
    strawberry_code = CodeableConcept.from_fhir(fhir_code) # type: ignore
    if not strawberry_code:
        logger.error(f"Failed to map FHIR Observation 'code' for ID '{fhir_id}'.")
        return None

    # Import the actual classes for from_fhir calls to avoid circular dependency
    from .identifier_type import Identifier
    from .reference_type import Reference

    return cls(
        id=strawberry.ID(str(fhir_id)),
        identifier=_map_list(fhir_observation.get("identifier"), Identifier.from_fhir), # type: ignore
        status=fhir_status,
        category=_map_list(fhir_observation.get("category"), CodeableConcept.from_fhir), # type: ignore
        code=strawberry_code,
        subject=Reference.from_fhir(fhir_observation.get("subject")) if fhir_observation.get("subject") else None, # type: ignore
        encounter=Reference.from_fhir(fhir_observation.get("encounter")) if fhir_observation.get("encounter") else None, # type: ignore
        effective_date_time=fhir_observation.get("effectiveDateTime"),
        effective_period=Period.from_fhir(fhir_observation.get("effectivePeriod")) if fhir_observation.get("effectivePeriod") else None, # type: ignore
        issued=fhir_observation.get("issued"),
        performer=_map_list(fhir_observation.get("performer"), Reference.from_fhir), # type: ignore
        value_quantity=Quantity.from_fhir(fhir_observation.get("valueQuantity")) if fhir_observation.get("valueQuantity") else None, # type: ignore
        value_codeable_concept=CodeableConcept.from_fhir(fhir_observation.get("valueCodeableConcept")) if fhir_observation.get("valueCodeableConcept") else None, # type: ignore
        value_string=fhir_observation.get("valueString"),
        value_boolean=fhir_observation.get("valueBoolean"),
        value_integer=fhir_observation.get("valueInteger"),
        value_date_time=fhir_observation.get("valueDateTime"),
        value_period=Period.from_fhir(fhir_observation.get("valuePeriod")) if fhir_observation.get("valuePeriod") else None, # type: ignore
        data_absent_reason=CodeableConcept.from_fhir(fhir_observation.get("dataAbsentReason")) if fhir_observation.get("dataAbsentReason") else None, # type: ignore
        interpretation=_map_list(fhir_observation.get("interpretation"), CodeableConcept.from_fhir), # type: ignore
        note=_map_list(fhir_observation.get("note"), Annotation.from_fhir), # type: ignore
        body_site=CodeableConcept.from_fhir(fhir_observation.get("bodySite")) if fhir_observation.get("bodySite") else None, # type: ignore
        method=CodeableConcept.from_fhir(fhir_observation.get("method")) if fhir_observation.get("method") else None # type: ignore
    )

# Assign from_fhir methods after all class definitions
Coding.from_fhir = classmethod(_coding_from_fhir_impl)
CodeableConcept.from_fhir = classmethod(_codeable_concept_from_fhir_impl)
Period.from_fhir = classmethod(_period_from_fhir_impl)

# Input Types for Mutations

@strawberry.input(description="Input for creating a Coding.")
class CodingInput:
    system: Optional[str] = strawberry.field(default=None, description="Identity of the terminology system.")
    code: Optional[str] = strawberry.field(default=None, description="Symbol in syntax defined by the system.")
    display: Optional[str] = strawberry.field(default=None, description="Representation defined by the system.")

@strawberry.input(description="Input for creating a CodeableConcept.")
class CodeableConceptInput:
    coding: Optional[List[CodingInput]] = strawberry.field(default=None, description="A list of codings.")
    text: Optional[str] = strawberry.field(default=None, description="Plain text representation of the concept.")

@strawberry.input(description="Input for creating a Quantity.")
class QuantityInput:
    value: Optional[float] = strawberry.field(default=None, description="Numerical value.")
    unit: Optional[str] = strawberry.field(default=None, description="Unit representation.")
    system: Optional[str] = strawberry.field(default=None, description="System that defines coded unit form (e.g., http://unitsofmeasure.org).")
    code: Optional[str] = strawberry.field(default=None, description="Coded form of the unit (e.g., kg).")

@strawberry.input(description="Input for a Period.")
class PeriodInput:
    start: Optional[str] = strawberry.field(default=None, description="Starting time (dateTime).")
    end: Optional[str] = strawberry.field(default=None, description="End time (dateTime).")


@strawberry.input(description="Input for an Annotation.")
class AnnotationInput:
    author_reference: Optional[ReferenceInput] = strawberry.field(default=None, description="Individual responsible for the annotation (if a resource). Example: { reference: 'Practitioner/example' }")
    author_string: Optional[str] = strawberry.field(default=None, description="Individual responsible for the annotation (if a string).")
    time: Optional[str] = strawberry.field(default=None, description="When the annotation was made (dateTime).")
    text: str = strawberry.field(description="The annotation - text content (markdown).")


@strawberry.input(description="Input for creating a Reference. Provide the reference string like 'Patient/123'.")
class ReferenceInput:
    reference: str = strawberry.field(description="Reference string, e.g., 'Patient/123' or 'Practitioner/abc'.")
    display: Optional[str] = strawberry.field(default=None, description="Display text for the reference (optional, server may fill if not provided).")

@strawberry.input(description="Input for an Identifier.")
class IdentifierInput:
    use: Optional[str] = strawberry.field(default=None, description="usual | official | temp | secondary | old (If known).")
    type: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Description of identifier (e.g., MRN).")
    system: Optional[str] = strawberry.field(default=None, description="The namespace for the identifier value.")
    value: Optional[str] = strawberry.field(default=None, description="The value of the identifier.")
    period: Optional[PeriodInput] = strawberry.field(default=None, description="Time period when id is/was valid for use.")
    assigner: Optional[ReferenceInput] = strawberry.field(default=None, description="Organization that issued id (e.g., { reference: 'Organization/123' }).")


@strawberry.input(description="Input for creating an Observation resource.")
class ObservationInput:
    identifier: Optional[List[IdentifierInput]] = strawberry.field(default_factory=list, description="Business Identifier for observation.")
    status: str = strawberry.field(description="registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown.")
    category: Optional[List[CodeableConceptInput]] = strawberry.field(default_factory=list, description="Classification of type of observation (e.g., 'vital-signs', 'laboratory').")
    code: CodeableConceptInput = strawberry.field(description="Type of observation (code / type).")
    subject: ReferenceInput = strawberry.field(description="Who and/or what this is about (Patient, Group, Device, Location). Example: { reference: 'Patient/example' }")
    encounter: Optional[ReferenceInput] = strawberry.field(default=None, description="Healthcare event during which this observation is made. Example: { reference: 'Encounter/example' }")
    effective_date_time: Optional[str] = strawberry.field(default=None, description="Clinically relevant date/time of observation (ISO 8601 dateTime string).")
    effective_period: Optional[PeriodInput] = strawberry.field(default=None, description="Clinically relevant time period of observation.")
    issued: Optional[str] = strawberry.field(default=None, description="Date/Time this version was made available (ISO 8601 instant string).")
    performer: Optional[List[ReferenceInput]] = strawberry.field(default_factory=list, description="Who is responsible for the observation. Example: [{ reference: 'Practitioner/example' }]")
    value_quantity: Optional[QuantityInput] = strawberry.field(default=None, description="Actual result as a Quantity.")
    value_codeable_concept: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Actual result as a CodeableConcept.")
    value_string: Optional[str] = strawberry.field(default=None, description="Actual result as a string.")
    value_boolean: Optional[bool] = strawberry.field(default=None, description="Actual result as a boolean.")
    value_integer: Optional[int] = strawberry.field(default=None, description="Actual result as an integer.")
    value_date_time: Optional[str] = strawberry.field(default=None, description="Actual result as a dateTime.")
    value_period: Optional[PeriodInput] = strawberry.field(default=None, description="Actual result as a Period.")
    data_absent_reason: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Why the result is missing (if applicable).")
    interpretation: Optional[List[CodeableConceptInput]] = strawberry.field(default_factory=list, description="High, low, normal, etc.")
    note: Optional[List[AnnotationInput]] = strawberry.field(default_factory=list, description="Comments about the observation.")
    body_site: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Observed body part.")
    method: Optional[CodeableConceptInput] = strawberry.field(default=None, description="How it was done.")

# Alias for backward compatibility
CreateObservationInput = ObservationInput

@strawberry.input(description="Input for updating an Observation resource. All fields are optional.")
class UpdateObservationInput:
    identifier: Optional[List[IdentifierInput]] = strawberry.field(default=None, description="Business Identifier for observation.")
    status: Optional[str] = strawberry.field(default=None, description="registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown.")
    category: Optional[List[CodeableConceptInput]] = strawberry.field(default=None, description="Classification of type of observation (e.g., 'vital-signs', 'laboratory').")
    code: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Type of observation (code / type).")
    subject: Optional[ReferenceInput] = strawberry.field(default=None, description="Who and/or what this is about. Example: { reference: 'Patient/example' }")
    encounter: Optional[ReferenceInput] = strawberry.field(default=None, description="Healthcare event during which this observation is made. Example: { reference: 'Encounter/example' }")
    effective_date_time: Optional[str] = strawberry.field(default=None, description="Clinically relevant date/time of observation (ISO 8601 dateTime string).")
    effective_period: Optional[PeriodInput] = strawberry.field(default=None, description="Clinically relevant time period of observation.")
    issued: Optional[str] = strawberry.field(default=None, description="Date/Time this version was made available (ISO 8601 instant string).")
    performer: Optional[List[ReferenceInput]] = strawberry.field(default=None, description="Who is responsible for the observation. Example: [{ reference: 'Practitioner/example' }]")
    value_quantity: Optional[QuantityInput] = strawberry.field(default=None, description="Actual result as a Quantity.")
    value_codeable_concept: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Actual result as a CodeableConcept.")
    value_string: Optional[str] = strawberry.field(default=None, description="Actual result as a string.")
    value_boolean: Optional[bool] = strawberry.field(default=None, description="Actual result as a boolean.")
    value_integer: Optional[int] = strawberry.field(default=None, description="Actual result as an integer.")
    value_date_time: Optional[str] = strawberry.field(default=None, description="Actual result as a dateTime.")
    value_period: Optional[PeriodInput] = strawberry.field(default=None, description="Actual result as a Period.")
    data_absent_reason: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Why the result is missing (if applicable).")
    interpretation: Optional[List[CodeableConceptInput]] = strawberry.field(default=None, description="High, low, normal, etc.")
    note: Optional[List[AnnotationInput]] = strawberry.field(default=None, description="Comments about the observation.")
    body_site: Optional[CodeableConceptInput] = strawberry.field(default=None, description="Observed body part.")
    method: Optional[CodeableConceptInput] = strawberry.field(default=None, description="How it was done.")

Period.from_fhir = classmethod(_period_from_fhir_impl)
Quantity.from_fhir = classmethod(_quantity_from_fhir_impl)
Annotation.from_fhir = classmethod(_annotation_from_fhir_impl)
Observation.from_fhir = classmethod(_observation_from_fhir_impl)
# End of deferred implementations


@strawberry.input(description="Input for filtering Observations.")
class ObservationFilterInput:
    patient_id: Optional[str] = strawberry.field(default=None, description="Filter by subject's patient ID (e.g., 'Patient/123').")
    category: Optional[str] = strawberry.field(default=None, description="Filter by observation category code (e.g., 'vital-signs').")
    code: Optional[str] = strawberry.field(default=None, description="Filter by observation code (e.g., LOINC code '8310-5' for body temperature). Can be a system|code string.")
    date: Optional[str] = strawberry.field(default=None, description="Filter by date or date range. Supports FHIR date search parameters (e.g., '2023-01-01', 'ge2023-01-01', 'le2023-12-31').")
    status: Optional[str] = strawberry.field(default=None, description="Filter by observation status (e.g., 'final', 'preliminary').")


@strawberry.type
class ObservationListResponse:
    observations: List[Observation] = strawberry.field(default_factory=list, description="A list of retrieved Observations.")
    total_count: Optional[int] = strawberry.field(default=0, description="Total number of observations matching the query (if applicable).")
    error: Optional[str] = strawberry.field(default=None, description="An error message if the operation failed.")

    # def from_fhir(cls, fhir_observation: Dict[str, Any]) -> 'Observation':
        # """Create a GraphQL Observation from a FHIR Observation."""
        # if not fhir_observation:
            # return None
            
        # return cls(
            # **(BaseObservation.from_fhir(fhir_observation).__dict__),
            # part_of=[Reference.from_fhir(p) for p in fhir_observation.get("partOf", [])],
            # based_on=[Reference.from_fhir(b) for b in fhir_observation.get("basedOn", [])],
            # instantiates_canonical=fhir_observation.get("instantiatesCanonical", []),
            # instantiates_uri=fhir_observation.get("instantiatesUri", [])
        # )


@strawberry.type
class ObservationResponse:
    """Response type for observation operations."""
    success: bool
    message: Optional[str] = None
    observation: Optional['Observation'] = None

    @classmethod
    def from_observation(cls, observation: 'Observation', success: bool = True, message: str = None) -> 'ObservationResponse':
        """Create a response from an observation."""
        return cls(
            success=success,
            message=message,
            observation=observation
        )

    @classmethod
    def from_error(cls, message: str) -> 'ObservationResponse':
        """Create an error response."""
        return cls(
            success=False,
            message=message,
            observation=None
        )

