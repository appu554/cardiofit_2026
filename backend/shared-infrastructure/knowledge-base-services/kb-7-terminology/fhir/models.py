"""
FHIR Data Models for KB7 Terminology Service

This module provides Pydantic models for FHIR R4 terminology resources and operations
used in the hybrid architecture integration.
"""

from typing import Dict, List, Any, Optional, Union
from datetime import datetime
from pydantic import BaseModel, Field, validator
from enum import Enum


class PublicationStatus(str, Enum):
    """FHIR PublicationStatus enumeration"""
    DRAFT = "draft"
    ACTIVE = "active"
    RETIRED = "retired"
    UNKNOWN = "unknown"


class ConceptDesignationUse(BaseModel):
    """FHIR Coding for designation use"""
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None


class ConceptDesignation(BaseModel):
    """FHIR ConceptDesignation"""
    language: Optional[str] = None
    use: Optional[ConceptDesignationUse] = None
    value: str


class ConceptProperty(BaseModel):
    """FHIR concept property"""
    code: str
    value_code: Optional[str] = Field(None, alias="valueCode")
    value_coding: Optional[Dict[str, Any]] = Field(None, alias="valueCoding")
    value_string: Optional[str] = Field(None, alias="valueString")
    value_integer: Optional[int] = Field(None, alias="valueInteger")
    value_boolean: Optional[bool] = Field(None, alias="valueBoolean")
    value_date_time: Optional[datetime] = Field(None, alias="valueDateTime")
    value_decimal: Optional[float] = Field(None, alias="valueDecimal")


class CodeSystemConcept(BaseModel):
    """FHIR CodeSystem concept"""
    code: str
    display: Optional[str] = None
    definition: Optional[str] = None
    designation: Optional[List[ConceptDesignation]] = None
    property: Optional[List[ConceptProperty]] = None
    concept: Optional[List['CodeSystemConcept']] = None


class ValueSetComposeIncludeConcept(BaseModel):
    """FHIR ValueSet compose include concept"""
    code: str
    display: Optional[str] = None
    designation: Optional[List[ConceptDesignation]] = None


class ValueSetComposeIncludeFilter(BaseModel):
    """FHIR ValueSet compose include filter"""
    property: str
    op: str
    value: str


class ValueSetComposeInclude(BaseModel):
    """FHIR ValueSet compose include"""
    system: Optional[str] = None
    version: Optional[str] = None
    concept: Optional[List[ValueSetComposeIncludeConcept]] = None
    filter: Optional[List[ValueSetComposeIncludeFilter]] = None
    value_set: Optional[List[str]] = Field(None, alias="valueSet")


class ValueSetCompose(BaseModel):
    """FHIR ValueSet compose"""
    locked_date: Optional[datetime] = Field(None, alias="lockedDate")
    inactive: Optional[bool] = None
    include: List[ValueSetComposeInclude]
    exclude: Optional[List[ValueSetComposeInclude]] = None


class ValueSetExpansionParameter(BaseModel):
    """FHIR ValueSet expansion parameter"""
    name: str
    value_string: Optional[str] = Field(None, alias="valueString")
    value_boolean: Optional[bool] = Field(None, alias="valueBoolean")
    value_integer: Optional[int] = Field(None, alias="valueInteger")
    value_decimal: Optional[float] = Field(None, alias="valueDecimal")
    value_uri: Optional[str] = Field(None, alias="valueUri")
    value_code: Optional[str] = Field(None, alias="valueCode")
    value_date_time: Optional[datetime] = Field(None, alias="valueDateTime")


class ValueSetExpansionContains(BaseModel):
    """FHIR ValueSet expansion contains"""
    system: Optional[str] = None
    abstract: Optional[bool] = None
    inactive: Optional[bool] = None
    version: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    designation: Optional[List[ConceptDesignation]] = None
    contains: Optional[List['ValueSetExpansionContains']] = None


class ValueSetExpansion(BaseModel):
    """FHIR ValueSet expansion"""
    identifier: Optional[str] = None
    timestamp: datetime
    total: Optional[int] = None
    offset: Optional[int] = None
    parameter: Optional[List[ValueSetExpansionParameter]] = None
    contains: Optional[List[ValueSetExpansionContains]] = None


class ConceptMapGroup(BaseModel):
    """FHIR ConceptMap group"""
    source: Optional[str] = None
    source_version: Optional[str] = Field(None, alias="sourceVersion")
    target: Optional[str] = None
    target_version: Optional[str] = Field(None, alias="targetVersion")
    element: List[Dict[str, Any]]  # ConceptMapGroupElement
    unmapped: Optional[Dict[str, Any]] = None


class OperationOutcomeIssue(BaseModel):
    """FHIR OperationOutcome issue"""
    severity: str
    code: str
    details: Optional[Dict[str, Any]] = None
    diagnostics: Optional[str] = None
    location: Optional[List[str]] = None
    expression: Optional[List[str]] = None


class OperationOutcome(BaseModel):
    """FHIR OperationOutcome"""
    resource_type: str = Field("OperationOutcome", alias="resourceType")
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    implicit_rules: Optional[str] = Field(None, alias="implicitRules")
    language: Optional[str] = None
    text: Optional[Dict[str, Any]] = None
    contained: Optional[List[Dict[str, Any]]] = None
    extension: Optional[List[Dict[str, Any]]] = None
    modifier_extension: Optional[List[Dict[str, Any]]] = Field(None, alias="modifierExtension")
    issue: List[OperationOutcomeIssue]


class Parameters(BaseModel):
    """FHIR Parameters resource"""
    resource_type: str = Field("Parameters", alias="resourceType")
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    implicit_rules: Optional[str] = Field(None, alias="implicitRules")
    language: Optional[str] = None
    parameter: Optional[List[Dict[str, Any]]] = None


class CodeSystemLookupRequest(BaseModel):
    """Request model for CodeSystem $lookup operation"""
    system: Optional[str] = None
    code: Optional[str] = None
    version: Optional[str] = None
    coding: Optional[Dict[str, Any]] = None
    date: Optional[datetime] = None
    display_language: Optional[str] = Field(None, alias="displayLanguage")
    property: Optional[List[str]] = None


class CodeSystemLookupResponse(BaseModel):
    """Response model for CodeSystem $lookup operation"""
    resource_type: str = Field("Parameters", alias="resourceType")
    parameter: List[Dict[str, Any]]


class ValueSetExpandRequest(BaseModel):
    """Request model for ValueSet $expand operation"""
    url: Optional[str] = None
    value_set: Optional[Dict[str, Any]] = Field(None, alias="valueSet")
    value_set_version: Optional[str] = Field(None, alias="valueSetVersion")
    context: Optional[str] = None
    context_direction: Optional[str] = Field(None, alias="contextDirection")
    filter: Optional[str] = None
    date: Optional[datetime] = None
    offset: Optional[int] = None
    count: Optional[int] = None
    include_definition: Optional[bool] = Field(None, alias="includeDefinition")
    include_designation: Optional[bool] = Field(None, alias="includeDesignation")
    designation: Optional[List[str]] = None
    include_inactive: Optional[bool] = Field(None, alias="includeInactive")
    active_only: Optional[bool] = Field(None, alias="activeOnly")
    exclude_nested: Optional[bool] = Field(None, alias="excludeNested")
    exclude_not_for_ui: Optional[bool] = Field(None, alias="excludeNotForUI")
    exclude_post_coordinated: Optional[bool] = Field(None, alias="excludePostCoordinated")
    display_language: Optional[str] = Field(None, alias="displayLanguage")
    exclude_system: Optional[List[str]] = Field(None, alias="excludeSystem")
    system_version: Optional[List[str]] = Field(None, alias="systemVersion")
    check_system_version: Optional[List[str]] = Field(None, alias="checkSystemVersion")
    force_system_version: Optional[List[str]] = Field(None, alias="forceSystemVersion")


class ConceptMapTranslateRequest(BaseModel):
    """Request model for ConceptMap $translate operation"""
    url: Optional[str] = None
    concept_map: Optional[Dict[str, Any]] = Field(None, alias="conceptMap")
    concept_map_version: Optional[str] = Field(None, alias="conceptMapVersion")
    code: Optional[str] = None
    system: Optional[str] = None
    version: Optional[str] = None
    source: Optional[str] = None
    coding: Optional[Dict[str, Any]] = None
    codeable_concept: Optional[Dict[str, Any]] = Field(None, alias="codeableConcept")
    target: Optional[str] = None
    target_system: Optional[str] = Field(None, alias="targetsystem")
    dependency: Optional[List[Dict[str, Any]]] = None
    reverse: Optional[bool] = None


class ConceptMapTranslateResponse(BaseModel):
    """Response model for ConceptMap $translate operation"""
    resource_type: str = Field("Parameters", alias="resourceType")
    parameter: List[Dict[str, Any]]


class ValidateCodeRequest(BaseModel):
    """Request model for terminology $validate-code operation"""
    url: Optional[str] = None
    context: Optional[str] = None
    value_set: Optional[Dict[str, Any]] = Field(None, alias="valueSet")
    value_set_version: Optional[str] = Field(None, alias="valueSetVersion")
    code: Optional[str] = None
    system: Optional[str] = None
    version: Optional[str] = None
    display: Optional[str] = None
    coding: Optional[Dict[str, Any]] = None
    codeable_concept: Optional[Dict[str, Any]] = Field(None, alias="codeableConcept")
    date: Optional[datetime] = None
    abstract: Optional[bool] = None
    display_language: Optional[str] = Field(None, alias="displayLanguage")


class ValidateCodeResponse(BaseModel):
    """Response model for terminology $validate-code operation"""
    resource_type: str = Field("Parameters", alias="resourceType")
    parameter: List[Dict[str, Any]]


class TerminologyCapabilities(BaseModel):
    """FHIR TerminologyCapabilities resource"""
    resource_type: str = Field("TerminologyCapabilities", alias="resourceType")
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    implicit_rules: Optional[str] = Field(None, alias="implicitRules")
    language: Optional[str] = None
    text: Optional[Dict[str, Any]] = None
    contained: Optional[List[Dict[str, Any]]] = None
    extension: Optional[List[Dict[str, Any]]] = None
    modifier_extension: Optional[List[Dict[str, Any]]] = Field(None, alias="modifierExtension")
    url: Optional[str] = None
    version: Optional[str] = None
    name: Optional[str] = None
    title: Optional[str] = None
    status: PublicationStatus
    experimental: Optional[bool] = None
    date: datetime
    publisher: Optional[str] = None
    contact: Optional[List[Dict[str, Any]]] = None
    description: Optional[str] = None
    use_context: Optional[List[Dict[str, Any]]] = Field(None, alias="useContext")
    jurisdiction: Optional[List[Dict[str, Any]]] = None
    purpose: Optional[str] = None
    copyright: Optional[str] = None
    kind: str
    software: Optional[Dict[str, Any]] = None
    implementation: Optional[Dict[str, Any]] = None
    locked_date: Optional[bool] = Field(None, alias="lockedDate")
    code_system: Optional[List[Dict[str, Any]]] = Field(None, alias="codeSystem")
    expansion: Optional[Dict[str, Any]] = None
    code_search: Optional[str] = Field(None, alias="codeSearch")
    validate_code: Optional[Dict[str, Any]] = Field(None, alias="validateCode")
    translation: Optional[Dict[str, Any]] = None
    closure: Optional[Dict[str, Any]] = None


class FHIRBundle(BaseModel):
    """FHIR Bundle resource"""
    resource_type: str = Field("Bundle", alias="resourceType")
    id: Optional[str] = None
    meta: Optional[Dict[str, Any]] = None
    implicit_rules: Optional[str] = Field(None, alias="implicitRules")
    language: Optional[str] = None
    identifier: Optional[Dict[str, Any]] = None
    type: str
    timestamp: Optional[datetime] = None
    total: Optional[int] = None
    link: Optional[List[Dict[str, Any]]] = None
    entry: Optional[List[Dict[str, Any]]] = None
    signature: Optional[Dict[str, Any]] = None


# Update forward references
CodeSystemConcept.update_forward_refs()
ValueSetExpansionContains.update_forward_refs()