"""
Complex FHIR datatypes for Clinical Synthesis Hub.

This module provides Pydantic models for complex FHIR datatypes
used across all microservices in the Clinical Synthesis Hub.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
from datetime import datetime
from ..base import FHIRBaseModel

class Attachment(FHIRBaseModel):
    """
    Content in a format that can be attached to a resource.
    """
    contentType: Optional[str] = None
    language: Optional[str] = None
    data: Optional[str] = None
    url: Optional[str] = None
    size: Optional[int] = None
    hash: Optional[str] = None
    title: Optional[str] = None
    creation: Optional[str] = None

class Coding(FHIRBaseModel):
    """
    A reference to a code defined by a terminology system.
    """
    system: str
    code: str
    display: Optional[str] = None
    version: Optional[str] = None
    userSelected: Optional[bool] = None

class CodeableConcept(FHIRBaseModel):
    """
    A concept that may be defined by a formal reference to a terminology
    or ontology or may be provided by text.
    """
    coding: List[Coding]
    text: Optional[str] = None

class Reference(FHIRBaseModel):
    """
    A reference from one resource to another.
    """
    reference: str
    display: Optional[str] = None
    type: Optional[str] = None
    identifier: Optional[Dict[str, Any]] = None

class Identifier(FHIRBaseModel):
    """
    An identifier - identifies some entity uniquely and unambiguously.
    """
    system: str
    value: str
    use: Optional[str] = None
    type: Optional[CodeableConcept] = None
    period: Optional[Dict[str, Any]] = None
    assigner: Optional[Reference] = None

class HumanName(FHIRBaseModel):
    """
    A human's name with the ability to identify parts and usage.
    """
    family: Optional[str] = None
    given: Optional[List[str]] = None
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None
    period: Optional[Dict[str, Any]] = None
    text: Optional[str] = None

class ContactPoint(FHIRBaseModel):
    """
    Details for all kinds of technology-mediated contact points for a person
    or organization, including telephone, email, etc.
    """
    system: str  # phone | email | etc.
    value: str
    use: Optional[str] = None  # home | work | mobile | etc.
    rank: Optional[int] = None
    period: Optional[Dict[str, Any]] = None

class Address(FHIRBaseModel):
    """
    An address expressed using postal conventions (as opposed to GPS or other location definition formats).
    """
    line: Optional[List[str]] = None
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None  # home | work | etc.
    type: Optional[str] = None  # postal | physical | both
    text: Optional[str] = None
    period: Optional[Dict[str, Any]] = None

class Quantity(FHIRBaseModel):
    """
    A measured amount (or an amount that can potentially be measured).
    """
    value: float
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None
    comparator: Optional[str] = None

class Period(FHIRBaseModel):
    """
    A time period defined by a start and end date/time.
    """
    start: Optional[str] = None
    end: Optional[str] = None

class Range(FHIRBaseModel):
    """
    A set of ordered Quantities defined by a low and high limit.
    """
    low: Optional[Quantity] = None
    high: Optional[Quantity] = None

class Ratio(FHIRBaseModel):
    """
    A relationship of two Quantity values - expressed as a numerator and a denominator.
    """
    numerator: Optional[Quantity] = None
    denominator: Optional[Quantity] = None

class Annotation(FHIRBaseModel):
    """
    A text note which also contains information about who made the statement and when.
    """
    authorString: Optional[str] = None
    authorReference: Optional[Reference] = None
    time: Optional[str] = None
    text: str
