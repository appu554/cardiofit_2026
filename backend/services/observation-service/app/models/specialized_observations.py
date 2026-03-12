from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
from datetime import datetime

# Import shared models
from shared.models import Observation
from shared.models import (
    CodeableConcept, Reference, Quantity, Period, Range, Ratio, Annotation
)
from shared.models.resources.observation import ObservationReferenceRange

# Import local models
from app.models.observation import ObservationCategory, ObservationStatus

class LaboratoryObservation(Observation):
    """Model for laboratory observations"""
    category: ObservationCategory = ObservationCategory.LABORATORY
    specimen: Optional[Reference] = None

    class Config:
        schema_extra = {
            "example": {
                "status": "final",
                "category": "laboratory",
                "code": {
                    "system": "http://loinc.org",
                    "code": "718-7",
                    "display": "Hemoglobin [Mass/volume] in Blood"
                },
                "subject": {
                    "reference": "Patient/123"
                },
                "effective_datetime": "2023-06-15T08:00:00",
                "value_quantity": {
                    "value": 14.5,
                    "unit": "g/dL",
                    "system": "http://unitsofmeasure.org",
                    "code": "g/dL"
                },
                "interpretation": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
                        "code": "N",
                        "display": "Normal"
                    }
                ],
                "reference_range": [
                    {
                        "low": {
                            "value": 13.0,
                            "unit": "g/dL",
                            "system": "http://unitsofmeasure.org",
                            "code": "g/dL"
                        },
                        "high": {
                            "value": 17.0,
                            "unit": "g/dL",
                            "system": "http://unitsofmeasure.org",
                            "code": "g/dL"
                        },
                        "text": "13.0-17.0 g/dL"
                    }
                ],
                "specimen": {
                    "reference": "Specimen/456"
                }
            }
        }

class VitalSignObservation(Observation):
    """Model for vital sign observations"""
    category: ObservationCategory = ObservationCategory.VITAL_SIGNS
    body_site: Optional[CodeableConcept] = None

    class Config:
        schema_extra = {
            "example": {
                "status": "final",
                "category": "vital-signs",
                "code": {
                    "system": "http://loinc.org",
                    "code": "8867-4",
                    "display": "Heart rate"
                },
                "subject": {
                    "reference": "Patient/123"
                },
                "effective_datetime": "2023-06-15T08:00:00",
                "value_quantity": {
                    "value": 80,
                    "unit": "beats/min",
                    "system": "http://unitsofmeasure.org",
                    "code": "/min"
                },
                "interpretation": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
                        "code": "N",
                        "display": "Normal"
                    }
                ],
                "reference_range": [
                    {
                        "low": {
                            "value": 60,
                            "unit": "beats/min",
                            "system": "http://unitsofmeasure.org",
                            "code": "/min"
                        },
                        "high": {
                            "value": 100,
                            "unit": "beats/min",
                            "system": "http://unitsofmeasure.org",
                            "code": "/min"
                        },
                        "text": "60-100 beats/min"
                    }
                ]
            }
        }

class PhysicalMeasurementObservation(Observation):
    """Model for physical measurement observations"""
    category: ObservationCategory = ObservationCategory.EXAM

    class Config:
        schema_extra = {
            "example": {
                "status": "final",
                "category": "exam",
                "code": {
                    "system": "http://loinc.org",
                    "code": "8302-2",
                    "display": "Body height"
                },
                "subject": {
                    "reference": "Patient/123"
                },
                "effective_datetime": "2023-06-15T08:00:00",
                "value_quantity": {
                    "value": 180,
                    "unit": "cm",
                    "system": "http://unitsofmeasure.org",
                    "code": "cm"
                }
            }
        }

class SocialHistoryObservation(Observation):
    """Model for social history observations"""
    category: ObservationCategory = ObservationCategory.SOCIAL_HISTORY

    class Config:
        schema_extra = {
            "example": {
                "status": "final",
                "category": "social-history",
                "code": {
                    "system": "http://loinc.org",
                    "code": "72166-2",
                    "display": "Tobacco smoking status"
                },
                "subject": {
                    "reference": "Patient/123"
                },
                "effective_datetime": "2023-06-15T08:00:00",
                "value_string": "Current every day smoker"
            }
        }

class ImagingObservation(Observation):
    """Model for imaging observations"""
    category: ObservationCategory = ObservationCategory.IMAGING

    class Config:
        schema_extra = {
            "example": {
                "status": "final",
                "category": "imaging",
                "code": {
                    "system": "http://loinc.org",
                    "code": "24627-2",
                    "display": "Chest X-ray"
                },
                "subject": {
                    "reference": "Patient/123"
                },
                "effective_datetime": "2023-06-15T08:00:00",
                "value_string": "No acute cardiopulmonary process"
            }
        }

class SurveyObservation(Observation):
    """Model for survey observations"""
    category: ObservationCategory = ObservationCategory.SURVEY

    class Config:
        schema_extra = {
            "example": {
                "status": "final",
                "category": "survey",
                "code": {
                    "system": "http://loinc.org",
                    "code": "44250-9",
                    "display": "PHQ-9 quick depression assessment panel"
                },
                "subject": {
                    "reference": "Patient/123"
                },
                "effective_datetime": "2023-06-15T08:00:00",
                "value_integer": 5
            }
        }

class ObservationPanel(BaseModel):
    """Model for a panel of related observations"""
    id: Optional[str] = None
    panel_code: CodeableConcept
    panel_name: str
    observations: List[Observation]
    effectiveDateTime: str
    issued: Optional[str] = None
    performer: Optional[List[Reference]] = None
    subject: Reference
    encounter: Optional[Reference] = None

    class Config:
        schema_extra = {
            "example": {
                "id": "panel-1",
                "panel_code": {
                    "system": "http://loinc.org",
                    "code": "24323-8",
                    "display": "Comprehensive metabolic panel"
                },
                "panel_name": "Comprehensive Metabolic Panel",
                "observations": [],
                "effectiveDateTime": "2023-06-15T08:00:00Z",
                "subject": {
                    "reference": "Patient/123"
                }
            }
        }
