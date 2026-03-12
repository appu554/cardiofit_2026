"""
GraphQL types for the Patient Service.

This module provides GraphQL types for the Patient Service using Graphene.
It automatically generates GraphQL types from FHIR models.
"""

import graphene
from graphene.types.generic import GenericScalar
from typing import Dict, Any, List, Optional
import logging

# Import shared models
from shared.models import (
    Patient as PatientModel,
    HumanName as HumanNameModel,
    Address as AddressModel,
    ContactPoint as ContactPointModel,
    Identifier as IdentifierModel,
    CodeableConcept as CodeableConceptModel,
    Coding as CodingModel,
    Reference as ReferenceModel
)

# Configure logging
logger = logging.getLogger(__name__)

class Coding(graphene.ObjectType):
    """GraphQL type for FHIR Coding."""
    system = graphene.String(description="The identification system")
    version = graphene.String(description="Version of the system")
    code = graphene.String(description="Symbol in syntax defined by the system")
    display = graphene.String(description="Representation defined by the system")
    user_selected = graphene.Boolean(description="If this coding was chosen directly by the user")

    @classmethod
    def from_fhir(cls, fhir_coding: Dict[str, Any]) -> 'Coding':
        """Create a GraphQL Coding from a FHIR Coding."""
        if not fhir_coding:
            return None

        return cls(
            system=fhir_coding.get("system"),
            version=fhir_coding.get("version"),
            code=fhir_coding.get("code"),
            display=fhir_coding.get("display"),
            user_selected=fhir_coding.get("userSelected")
        )

class CodeableConcept(graphene.ObjectType):
    """GraphQL type for FHIR CodeableConcept."""
    coding = graphene.List(Coding, description="Codes defined by a terminology system")
    text = graphene.String(description="Plain text representation")

    @classmethod
    def from_fhir(cls, fhir_concept: Dict[str, Any]) -> 'CodeableConcept':
        """Create a GraphQL CodeableConcept from a FHIR CodeableConcept."""
        if not fhir_concept:
            return None

        codings = []
        if "coding" in fhir_concept and fhir_concept["coding"]:
            for coding in fhir_concept["coding"]:
                codings.append(Coding.from_fhir(coding))

        return cls(
            coding=codings,
            text=fhir_concept.get("text")
        )

class Period(graphene.ObjectType):
    """GraphQL type for FHIR Period."""
    start = graphene.String(description="Starting time with inclusive boundary")
    end = graphene.String(description="End time with inclusive boundary, if not ongoing")

    @classmethod
    def from_fhir(cls, fhir_period: Dict[str, Any]) -> 'Period':
        """Create a GraphQL Period from a FHIR Period."""
        if not fhir_period:
            return None

        return cls(
            start=fhir_period.get("start"),
            end=fhir_period.get("end")
        )

class Identifier(graphene.ObjectType):
    """GraphQL type for FHIR Identifier."""
    use = graphene.String(description="usual | official | temp | secondary | old")
    type = graphene.Field(CodeableConcept, description="Type for this identifier")
    system = graphene.String(description="The namespace for the identifier value")
    value = graphene.String(description="The value that is unique")
    period = graphene.Field(lambda: Period, description="Time period when id is/was valid for use")
    assigner = graphene.Field(lambda: Reference, description="Organization that issued id")

    @classmethod
    def from_fhir(cls, fhir_identifier: Dict[str, Any]) -> 'Identifier':
        """Create a GraphQL Identifier from a FHIR Identifier."""
        if not fhir_identifier:
            return None

        return cls(
            use=fhir_identifier.get("use"),
            type=CodeableConcept.from_fhir(fhir_identifier.get("type")),
            system=fhir_identifier.get("system"),
            value=fhir_identifier.get("value"),
            period=Period.from_fhir(fhir_identifier.get("period")),
            assigner=Reference.from_fhir(fhir_identifier.get("assigner"))
        )

class Reference(graphene.ObjectType):
    """GraphQL type for FHIR Reference."""
    reference = graphene.String(description="A reference to a location at which the other resource is found")
    type = graphene.String(description="Type the reference refers to")
    identifier = graphene.Field(Identifier, description="Logical identifier for the referenced resource")
    display = graphene.String(description="Text alternative for the resource")

    @classmethod
    def from_fhir(cls, fhir_reference: Dict[str, Any]) -> 'Reference':
        """Create a GraphQL Reference from a FHIR Reference."""
        if not fhir_reference:
            return None

        return cls(
            reference=fhir_reference.get("reference"),
            type=fhir_reference.get("type"),
            identifier=Identifier.from_fhir(fhir_reference.get("identifier")),
            display=fhir_reference.get("display")
        )

class Language(graphene.ObjectType):
    """GraphQL type for FHIR Language."""
    coding = graphene.List(Coding, description="Codes for the language")
    text = graphene.String(description="Plain text representation")

    @classmethod
    def from_fhir(cls, fhir_language: Dict[str, Any]) -> 'Language':
        """Create a GraphQL Language from a FHIR Language."""
        if not fhir_language:
            return None

        codings = []
        if "coding" in fhir_language and fhir_language["coding"]:
            for coding in fhir_language["coding"]:
                codings.append(Coding.from_fhir(coding))

        return cls(
            coding=codings,
            text=fhir_language.get("text")
        )

class Communication(graphene.ObjectType):
    """GraphQL type for FHIR Communication."""
    language = graphene.Field(Language, description="The language which can be used to communicate with the patient")
    preferred = graphene.Boolean(description="Language preference indicator")

    @classmethod
    def from_fhir(cls, fhir_communication: Dict[str, Any]) -> 'Communication':
        """Create a GraphQL Communication from a FHIR Communication."""
        if not fhir_communication:
            return None

        return cls(
            language=Language.from_fhir(fhir_communication.get("language")),
            preferred=fhir_communication.get("preferred")
        )

class HumanName(graphene.ObjectType):
    """GraphQL type for FHIR HumanName."""
    use = graphene.String(description="usual | official | temp | nickname | anonymous | old | maiden")
    text = graphene.String(description="Text representation of the full name")
    family = graphene.String(description="Family name (often called 'Surname')")
    given = graphene.List(graphene.String, description="Given names (not always 'first'). Includes middle names")
    prefix = graphene.List(graphene.String, description="Parts that come before the name")
    suffix = graphene.List(graphene.String, description="Parts that come after the name")
    period = GenericScalar(description="Time period when name was/is in use")

    @classmethod
    def from_fhir(cls, fhir_name: Dict[str, Any]) -> 'HumanName':
        """Create a GraphQL HumanName from a FHIR HumanName."""
        if not fhir_name:
            return None

        return cls(
            use=fhir_name.get("use"),
            text=fhir_name.get("text"),
            family=fhir_name.get("family"),
            given=fhir_name.get("given"),
            prefix=fhir_name.get("prefix"),
            suffix=fhir_name.get("suffix"),
            period=fhir_name.get("period")
        )

class ContactPoint(graphene.ObjectType):
    """GraphQL type for FHIR ContactPoint."""
    system = graphene.String(description="phone | fax | email | pager | url | sms | other")
    value = graphene.String(description="The actual contact point details")
    use = graphene.String(description="home | work | temp | old | mobile")
    rank = graphene.Int(description="Specify preferred order of use (1 = highest)")
    period = GenericScalar(description="Time period when the contact point was/is in use")

    @classmethod
    def from_fhir(cls, fhir_contact: Dict[str, Any]) -> 'ContactPoint':
        """Create a GraphQL ContactPoint from a FHIR ContactPoint."""
        if not fhir_contact:
            return None

        return cls(
            system=fhir_contact.get("system"),
            value=fhir_contact.get("value"),
            use=fhir_contact.get("use"),
            rank=fhir_contact.get("rank"),
            period=fhir_contact.get("period")
        )

class Address(graphene.ObjectType):
    """GraphQL type for FHIR Address."""
    use = graphene.String(description="home | work | temp | old | billing")
    type = graphene.String(description="postal | physical | both")
    text = graphene.String(description="Text representation of the address")
    line = graphene.List(graphene.String, description="Street name, number, direction & P.O. Box etc.")
    city = graphene.String(description="Name of city, town etc.")
    district = graphene.String(description="District name")
    state = graphene.String(description="Sub-unit of country (abbreviations ok)")
    postal_code = graphene.String(description="Postal code for area")
    country = graphene.String(description="Country (e.g. can be ISO 3166 2 or 3 letter code)")
    period = GenericScalar(description="Time period when address was/is in use")

    @classmethod
    def from_fhir(cls, fhir_address: Dict[str, Any]) -> 'Address':
        """Create a GraphQL Address from a FHIR Address."""
        if not fhir_address:
            return None

        return cls(
            use=fhir_address.get("use"),
            type=fhir_address.get("type"),
            text=fhir_address.get("text"),
            line=fhir_address.get("line"),
            city=fhir_address.get("city"),
            district=fhir_address.get("district"),
            state=fhir_address.get("state"),
            postal_code=fhir_address.get("postalCode"),
            country=fhir_address.get("country"),
            period=fhir_address.get("period")
        )

class Patient(graphene.ObjectType):
    """GraphQL type for FHIR Patient."""
    id = graphene.ID(description="Logical id of this artifact")
    resource_type = graphene.String(description="Type of resource")
    meta = GenericScalar(description="Metadata about the resource")
    identifier = graphene.List(Identifier, description="An identifier for this patient")
    active = graphene.Boolean(description="Whether this patient's record is in active use")
    name = graphene.List(HumanName, description="A name associated with the patient")
    telecom = graphene.List(ContactPoint, description="A contact detail for the individual")
    gender = graphene.String(description="male | female | other | unknown")
    birth_date = graphene.String(description="The date of birth for the individual")
    deceased_boolean = graphene.Boolean(description="Indicates if the individual is deceased or not")
    deceased_date_time = graphene.String(description="Indicates if the individual is deceased or not")
    address = graphene.List(Address, description="An address for the individual")
    marital_status = graphene.Field(CodeableConcept, description="Marital (civil) status of a patient")
    multiple_birth_boolean = graphene.Boolean(description="Whether patient is part of a multiple birth")
    multiple_birth_integer = graphene.Int(description="Whether patient is part of a multiple birth")
    contact = GenericScalar(description="A contact party (e.g. guardian, partner, friend) for the patient")
    communication = graphene.List(Communication, description="A language which may be used to communicate with the patient")
    general_practitioner = graphene.List(Reference, description="Patient's nominated primary care provider")
    managing_organization = graphene.Field(Reference, description="Organization that is the custodian of the patient record")

    @classmethod
    def from_fhir(cls, fhir_patient: Dict[str, Any]) -> 'Patient':
        """Create a GraphQL Patient from a FHIR Patient."""
        if not fhir_patient:
            return None

        # Convert identifiers
        identifiers = []
        if "identifier" in fhir_patient and fhir_patient["identifier"]:
            for identifier in fhir_patient["identifier"]:
                identifiers.append(Identifier.from_fhir(identifier))

        # Convert names
        names = []
        if "name" in fhir_patient and fhir_patient["name"]:
            for name in fhir_patient["name"]:
                names.append(HumanName.from_fhir(name))

        # Convert telecom
        telecoms = []
        if "telecom" in fhir_patient and fhir_patient["telecom"]:
            for telecom in fhir_patient["telecom"]:
                telecoms.append(ContactPoint.from_fhir(telecom))

        # Convert addresses
        addresses = []
        if "address" in fhir_patient and fhir_patient["address"]:
            for address in fhir_patient["address"]:
                addresses.append(Address.from_fhir(address))

        # Convert communications
        communications = []
        if "communication" in fhir_patient and fhir_patient["communication"]:
            for comm in fhir_patient["communication"]:
                communications.append(Communication.from_fhir(comm))

        # Convert general practitioners
        practitioners = []
        if "generalPractitioner" in fhir_patient and fhir_patient["generalPractitioner"]:
            for pract in fhir_patient["generalPractitioner"]:
                practitioners.append(Reference.from_fhir(pract))

        return cls(
            id=fhir_patient.get("id"),
            resource_type=fhir_patient.get("resourceType", "Patient"),
            meta=fhir_patient.get("meta"),
            identifier=identifiers,
            active=fhir_patient.get("active"),
            name=names,
            telecom=telecoms,
            gender=fhir_patient.get("gender"),
            birth_date=fhir_patient.get("birthDate"),
            deceased_boolean=fhir_patient.get("deceasedBoolean"),
            deceased_date_time=fhir_patient.get("deceasedDateTime"),
            address=addresses,
            marital_status=CodeableConcept.from_fhir(fhir_patient.get("maritalStatus")),
            multiple_birth_boolean=fhir_patient.get("multipleBirthBoolean"),
            multiple_birth_integer=fhir_patient.get("multipleBirthInteger"),
            contact=fhir_patient.get("contact"),
            communication=communications,
            general_practitioner=practitioners,
            managing_organization=Reference.from_fhir(fhir_patient.get("managingOrganization"))
        )

class PatientConnection(graphene.ObjectType):
    """GraphQL connection type for Patient pagination."""
    items = graphene.List(Patient, description="List of patients")
    total = graphene.Int(description="Total number of patients")
    page = graphene.Int(description="Current page number")
    count = graphene.Int(description="Number of items per page")

    @classmethod
    def from_patient_list(cls, patient_list) -> 'PatientConnection':
        """Create a GraphQL PatientConnection from a PatientList."""
        if not patient_list:
            return cls(items=[], total=0, page=1, count=10)

        # Convert patients
        patients = []

        # Handle different types of patient_list
        if hasattr(patient_list, "patients"):
            # It's a PatientList model
            patient_items = patient_list.patients
            total = patient_list.total
            page = patient_list.page
            count = patient_list.count
        elif isinstance(patient_list, dict):
            # It's a dictionary
            patient_items = patient_list.get("patients", [])
            total = patient_list.get("total", 0)
            page = patient_list.get("page", 1)
            count = patient_list.get("count", 10)
        else:
            # Fallback
            logger.error(f"Unexpected patient_list type: {type(patient_list)}")
            return cls(items=[], total=0, page=1, count=10)

        # Process each patient
        for patient_data in patient_items:
            # Convert to dict if it's a model
            if hasattr(patient_data, "model_dump"):
                patient_dict = patient_data.model_dump()
            else:
                patient_dict = patient_data

            patient = Patient.from_fhir(patient_dict)
            if patient:
                patients.append(patient)

        return cls(
            items=patients,
            total=total,
            page=page,
            count=count
        )
