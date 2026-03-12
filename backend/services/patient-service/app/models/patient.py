"""
Patient models for the Patient Service.

This module provides models for the Patient Service API, using the shared FHIR models.
"""

from typing import Dict, List, Optional, Any
from pydantic import BaseModel, Field, EmailStr

# Import shared models
from shared.models import Patient as SharedPatient
from shared.models import HumanName, Address, ContactPoint, Identifier, Reference, CodeableConcept

# Re-export the shared Patient model
Patient = SharedPatient

class PatientCreate(BaseModel):
    """Model for creating a new patient."""
    identifier: Optional[List[Identifier]] = None
    active: Optional[bool] = True
    name: List[HumanName]
    telecom: Optional[List[ContactPoint]] = None
    gender: Optional[str] = None  # male | female | other | unknown
    birthDate: Optional[str] = None
    address: Optional[List[Address]] = None
    text: Optional[Dict[str, Any]] = None  # Narrative text summary
    deceasedBoolean: Optional[bool] = None
    deceasedDateTime: Optional[str] = None
    maritalStatus: Optional[CodeableConcept] = None
    multipleBirthBoolean: Optional[bool] = None
    multipleBirthInteger: Optional[int] = None
    contact: Optional[List[Dict[str, Any]]] = None
    communication: Optional[List[Dict[str, Any]]] = None
    generalPractitioner: Optional[List[Reference]] = None
    managingOrganization: Optional[Reference] = None

    class Config:
        schema_extra = {
            "example": {
                "text": {
                    "status": "generated",
                    "div": "<div xmlns=\"http://www.w3.org/1999/xhtml\">John Smith</div>"
                },
                "identifier": [
                    {
                        "use": "usual",
                        "type": {
                            "coding": [
                                {
                                    "system": "http://terminology.hl7.org/CodeSystem/v2-0203",
                                    "code": "MR"
                                }
                            ]
                        },
                        "system": "urn:oid:1.2.36.146.595.217.0.1",
                        "value": "12345",
                        "period": { "start": "2001-05-06" },
                        "assigner": {
                            "display": "Acme Healthcare",
                            "reference": "Organization/example"
                        }
                    }
                ],
                "active": True,
                "name": [
                    {
                        "use": "official",
                        "family": "Smith",
                        "given": ["John", "Michael"]
                    },
                    { "use": "usual", "given": ["Johnny"] }
                ],
                "telecom": [
                    {
                        "system": "phone",
                        "value": "555-555-5555",
                        "use": "home"
                    },
                    {
                        "system": "email",
                        "value": "john.smith@example.com",
                        "use": "work"
                    }
                ],
                "gender": "male",
                "birthDate": "1974-12-25",
                "deceasedBoolean": False,
                "address": [
                    {
                        "use": "home",
                        "type": "both",
                        "text": "123 Main St, Anytown, CA 12345, USA",
                        "line": ["123 Main St"],
                        "city": "Anytown",
                        "state": "CA",
                        "postalCode": "12345",
                        "country": "USA",
                        "period": { "start": "2010-03-23" }
                    }
                ],
                "maritalStatus": {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/v3-MaritalStatus",
                            "code": "M",
                            "display": "Married"
                        }
                    ],
                    "text": "Married"
                },
                "multipleBirthBoolean": False,
                "contact": [
                    {
                        "relationship": [
                            {
                                "coding": [
                                    {
                                        "system": "http://terminology.hl7.org/CodeSystem/v2-0131",
                                        "code": "C"
                                    }
                                ]
                            }
                        ],
                        "name": {
                            "family": "Smith",
                            "given": ["Jane"]
                        },
                        "telecom": [
                            {
                                "system": "phone",
                                "value": "555-555-5556"
                            }
                        ],
                        "address": {
                            "use": "home",
                            "line": ["123 Main St"],
                            "city": "Anytown",
                            "state": "CA",
                            "postalCode": "12345",
                            "country": "USA"
                        },
                        "gender": "female",
                        "period": { "start": "2012" }
                    }
                ],
                "communication": [
                    {
                        "language": {
                            "coding": [
                                {
                                    "system": "urn:ietf:bcp:47",
                                    "code": "en",
                                    "display": "English"
                                }
                            ],
                            "text": "English"
                        },
                        "preferred": True
                    }
                ],
                "generalPractitioner": [
                    {
                        "reference": "Practitioner/64868e2c-0f36-48fc-9a8a-46f1b5a96ea0",
                        "display": "Dr. John Doe"
                    }
                ],
                "managingOrganization": {
                    "reference": "Organization/example",
                    "display": "Acme Healthcare"
                }
            }
        }

class PatientUpdate(BaseModel):
    """Model for updating a patient."""
    identifier: Optional[List[Identifier]] = None
    active: Optional[bool] = None
    name: Optional[List[HumanName]] = None
    telecom: Optional[List[ContactPoint]] = None
    gender: Optional[str] = None  # male | female | other | unknown
    birthDate: Optional[str] = None
    address: Optional[List[Address]] = None
    text: Optional[Dict[str, Any]] = None  # Narrative text summary
    deceasedBoolean: Optional[bool] = None
    deceasedDateTime: Optional[str] = None
    maritalStatus: Optional[CodeableConcept] = None
    multipleBirthBoolean: Optional[bool] = None
    multipleBirthInteger: Optional[int] = None
    contact: Optional[List[Dict[str, Any]]] = None
    communication: Optional[List[Dict[str, Any]]] = None
    generalPractitioner: Optional[List[Reference]] = None
    managingOrganization: Optional[Reference] = None

    class Config:
        schema_extra = {
            "example": {
                "active": True,
                "name": [
                    {
                        "family": "Smith",
                        "given": ["John", "Adam"],
                        "use": "official"
                    }
                ],
                "telecom": [
                    {
                        "system": "phone",
                        "value": "555-987-6543",
                        "use": "mobile"
                    }
                ]
            }
        }

class PatientResponse(BaseModel):
    """Model for patient response."""
    id: str
    resourceType: str = "Patient"
    identifier: Optional[List[Identifier]] = None
    active: bool
    name: List[HumanName]
    telecom: Optional[List[ContactPoint]] = None
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    address: Optional[List[Address]] = None
    text: Optional[Dict[str, Any]] = None  # Narrative text summary
    deceasedBoolean: Optional[bool] = None
    deceasedDateTime: Optional[str] = None
    maritalStatus: Optional[CodeableConcept] = None
    multipleBirthBoolean: Optional[bool] = None
    multipleBirthInteger: Optional[int] = None
    contact: Optional[List[Dict[str, Any]]] = None
    communication: Optional[List[Dict[str, Any]]] = None
    generalPractitioner: Optional[List[Reference]] = None
    managingOrganization: Optional[Reference] = None

    class Config:
        schema_extra = {
            "example": {
                "id": "patient-123",
                "resourceType": "Patient",
                "identifier": [
                    {
                        "system": "http://hospital.example.org/identifiers/patients",
                        "value": "123456",
                        "use": "official"
                    }
                ],
                "active": True,
                "name": [
                    {
                        "family": "Smith",
                        "given": ["John", "Adam"],
                        "use": "official"
                    }
                ],
                "telecom": [
                    {
                        "system": "phone",
                        "value": "555-123-4567",
                        "use": "home"
                    },
                    {
                        "system": "email",
                        "value": "john.smith@example.com"
                    }
                ],
                "gender": "male",
                "birthDate": "1970-01-01",
                "address": [
                    {
                        "line": ["123 Main St"],
                        "city": "Anytown",
                        "state": "CA",
                        "postalCode": "12345",
                        "country": "USA",
                        "use": "home"
                    }
                ]
            }
        }

class PatientList(BaseModel):
    """Model for a list of patients."""
    patients: List[PatientResponse]
    total: int
    page: int
    count: int

    class Config:
        schema_extra = {
            "example": {
                "patients": [
                    {
                        "id": "patient-123",
                        "resourceType": "Patient",
                        "active": True,
                        "name": [
                            {
                                "family": "Smith",
                                "given": ["John"],
                                "use": "official"
                            }
                        ],
                        "gender": "male",
                        "birthDate": "1970-01-01"
                    }
                ],
                "total": 1,
                "page": 1,
                "count": 10
            }
        }
