"""
Laboratory Order Models for Order Management Service

This module provides FHIR-compliant models for laboratory orders,
implementing the FHIR ServiceRequest resource for lab tests.
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
from datetime import datetime
from enum import Enum
import os
import sys

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import shared FHIR models and base clinical order
try:
    from shared.models import (
        FHIRBaseModel, CodeableConcept, Reference, Identifier, 
        Period, Annotation, Quantity
    )
    from .clinical_order import ClinicalOrder, OrderStatus, OrderIntent, OrderPriority
except ImportError:
    # Fallback if shared models are not available
    from pydantic import BaseModel as FHIRBaseModel
    
    class CodeableConcept(BaseModel):
        coding: Optional[List[Dict[str, Any]]] = None
        text: Optional[str] = None
    
    class Reference(BaseModel):
        reference: Optional[str] = None
        display: Optional[str] = None
    
    class Identifier(BaseModel):
        use: Optional[str] = None
        system: Optional[str] = None
        value: Optional[str] = None
    
    class Period(BaseModel):
        start: Optional[datetime] = None
        end: Optional[datetime] = None
    
    class Annotation(BaseModel):
        text: str
        author_string: Optional[str] = None
        time: Optional[datetime] = None
    
    class Quantity(BaseModel):
        value: Optional[float] = None
        unit: Optional[str] = None
        system: Optional[str] = None
        code: Optional[str] = None
    
    # Define fallback enums
    from enum import Enum
    
    class OrderStatus(str, Enum):
        DRAFT = "draft"
        ACTIVE = "active"
        ON_HOLD = "on-hold"
        REVOKED = "revoked"
        COMPLETED = "completed"
        ENTERED_IN_ERROR = "entered-in-error"
        UNKNOWN = "unknown"
    
    class OrderIntent(str, Enum):
        PROPOSAL = "proposal"
        PLAN = "plan"
        DIRECTIVE = "directive"
        ORDER = "order"
    
    class OrderPriority(str, Enum):
        ROUTINE = "routine"
        URGENT = "urgent"
        ASAP = "asap"
        STAT = "stat"

# Lab-specific enums
class SpecimenType(str, Enum):
    """Common specimen types for laboratory tests"""
    BLOOD = "blood"
    SERUM = "serum"
    PLASMA = "plasma"
    URINE = "urine"
    STOOL = "stool"
    SPUTUM = "sputum"
    CSF = "csf"  # Cerebrospinal fluid
    TISSUE = "tissue"
    SWAB = "swab"
    SALIVA = "saliva"
    OTHER = "other"

class CollectionMethod(str, Enum):
    """Methods for specimen collection"""
    VENIPUNCTURE = "venipuncture"
    CAPILLARY = "capillary"
    ARTERIAL = "arterial"
    CLEAN_CATCH = "clean-catch"
    CATHETER = "catheter"
    MIDSTREAM = "midstream"
    FIRST_MORNING = "first-morning"
    RANDOM = "random"
    TIMED = "timed"
    FASTING = "fasting"

# Specimen Collection Details
class SpecimenCollection(FHIRBaseModel):
    """
    Details about specimen collection for laboratory orders.
    """
    collected_datetime: Optional[datetime] = Field(None, alias="collectedDateTime", description="Collection time")
    collected_period: Optional[Period] = Field(None, alias="collectedPeriod", description="Collection time period")
    collector: Optional[Reference] = Field(None, description="Who collected the specimen")
    collection_method: Optional[CodeableConcept] = Field(None, alias="method", description="Technique used to perform collection")
    body_site: Optional[CodeableConcept] = Field(None, alias="bodySite", description="Anatomical collection site")
    quantity: Optional[Quantity] = Field(None, description="The quantity of specimen collected")
    fasting_status: Optional[CodeableConcept] = Field(None, alias="fastingStatusCodeableConcept", description="Whether or how long patient abstained from food and/or drink")
    
    class Config:
        extra = "allow"
        populate_by_name = True

# Laboratory Order Model
class LabOrder(ClinicalOrder):
    """
    FHIR ServiceRequest resource specialized for laboratory orders.
    
    This model extends the base ClinicalOrder with lab-specific fields.
    """
    
    # Override resource type for lab orders
    resourceType: str = Field(default="ServiceRequest", description="FHIR resource type")
    
    # Lab-specific category
    category: Optional[List[CodeableConcept]] = Field(
        default=[{
            "coding": [{
                "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                "code": "laboratory",
                "display": "Laboratory"
            }]
        }],
        description="Laboratory service category"
    )
    
    # Specimen details
    specimen: Optional[List[Reference]] = Field(None, description="Specimen to be tested")
    specimen_collection: Optional[SpecimenCollection] = Field(None, description="Specimen collection details")
    
    # Lab-specific timing
    collection_datetime_preference: Optional[datetime] = Field(None, description="Preferred collection time")
    fasting_required: Optional[bool] = Field(None, description="Whether fasting is required")
    fasting_duration: Optional[str] = Field(None, description="Required fasting duration")
    
    # Test-specific details
    test_code: Optional[CodeableConcept] = Field(None, description="Specific test code (e.g., LOINC)")
    test_name: Optional[str] = Field(None, description="Human-readable test name")
    specimen_source: Optional[SpecimenType] = Field(None, description="Type of specimen required")
    collection_method: Optional[CollectionMethod] = Field(None, description="Method for specimen collection")
    
    # Clinical context for lab
    clinical_history: Optional[str] = Field(None, description="Clinical history relevant to the test")
    suspected_diagnosis: Optional[List[CodeableConcept]] = Field(None, description="Suspected diagnoses")
    
    # Lab-specific instructions
    special_instructions: Optional[str] = Field(None, description="Special handling or processing instructions")
    transport_requirements: Optional[str] = Field(None, description="Specimen transport requirements")
    
    class Config:
        extra = "allow"
        populate_by_name = True

# Common Lab Test Panels
class LabTestPanel(FHIRBaseModel):
    """
    Predefined laboratory test panels for common orders.
    """
    panel_code: str = Field(..., description="Panel identifier code")
    panel_name: str = Field(..., description="Human-readable panel name")
    description: Optional[str] = Field(None, description="Panel description")
    tests: List[CodeableConcept] = Field(..., description="Individual tests in the panel")
    specimen_type: SpecimenType = Field(..., description="Required specimen type")
    fasting_required: bool = Field(default=False, description="Whether fasting is required")
    special_instructions: Optional[str] = Field(None, description="Special instructions for the panel")
    
    class Config:
        extra = "allow"

# Create and Update models for API endpoints
class LabOrderCreate(BaseModel):
    """Model for creating a laboratory order"""
    status: OrderStatus = OrderStatus.DRAFT
    intent: OrderIntent = OrderIntent.ORDER
    priority: Optional[OrderPriority] = OrderPriority.ROUTINE
    code: CodeableConcept
    subject: Reference
    encounter: Optional[Reference] = None
    requester: Optional[Reference] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    
    # Lab-specific fields
    test_code: Optional[CodeableConcept] = None
    test_name: Optional[str] = None
    specimen_source: Optional[SpecimenType] = None
    collection_method: Optional[CollectionMethod] = None
    collection_datetime_preference: Optional[datetime] = None
    fasting_required: Optional[bool] = False
    fasting_duration: Optional[str] = None
    clinical_history: Optional[str] = None
    suspected_diagnosis: Optional[List[CodeableConcept]] = None
    special_instructions: Optional[str] = None
    transport_requirements: Optional[str] = None
    
    def to_lab_order(self) -> LabOrder:
        """Convert to a FHIR LabOrder."""
        data = self.model_dump(exclude_unset=True)
        data["authored_on"] = datetime.utcnow()
        
        # Set lab-specific category
        data["category"] = [{
            "coding": [{
                "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                "code": "laboratory",
                "display": "Laboratory"
            }]
        }]
        
        return LabOrder(**data)

class LabOrderUpdate(BaseModel):
    """Model for updating a laboratory order"""
    status: Optional[OrderStatus] = None
    priority: Optional[OrderPriority] = None
    collection_datetime_preference: Optional[datetime] = None
    note: Optional[List[Annotation]] = None
    special_instructions: Optional[str] = None
    transport_requirements: Optional[str] = None

# Common lab test panels
COMMON_LAB_PANELS = {
    "CBC": LabTestPanel(
        panel_code="CBC",
        panel_name="Complete Blood Count",
        description="Basic blood panel including WBC, RBC, hemoglobin, hematocrit, and platelets",
        tests=[
            {"coding": [{"system": "http://loinc.org", "code": "6690-2", "display": "Leukocytes [#/volume] in Blood by Automated count"}]},
            {"coding": [{"system": "http://loinc.org", "code": "789-8", "display": "Erythrocytes [#/volume] in Blood by Automated count"}]},
            {"coding": [{"system": "http://loinc.org", "code": "718-7", "display": "Hemoglobin [Mass/volume] in Blood"}]},
            {"coding": [{"system": "http://loinc.org", "code": "4544-3", "display": "Hematocrit [Volume Fraction] of Blood by Automated count"}]},
            {"coding": [{"system": "http://loinc.org", "code": "777-3", "display": "Platelets [#/volume] in Blood by Automated count"}]}
        ],
        specimen_type=SpecimenType.BLOOD,
        fasting_required=False
    ),
    "BMP": LabTestPanel(
        panel_code="BMP",
        panel_name="Basic Metabolic Panel",
        description="Basic chemistry panel including glucose, electrolytes, and kidney function",
        tests=[
            {"coding": [{"system": "http://loinc.org", "code": "2345-7", "display": "Glucose [Mass/volume] in Serum or Plasma"}]},
            {"coding": [{"system": "http://loinc.org", "code": "2951-2", "display": "Sodium [Moles/volume] in Serum or Plasma"}]},
            {"coding": [{"system": "http://loinc.org", "code": "2823-3", "display": "Potassium [Moles/volume] in Serum or Plasma"}]},
            {"coding": [{"system": "http://loinc.org", "code": "2075-0", "display": "Chloride [Moles/volume] in Serum or Plasma"}]},
            {"coding": [{"system": "http://loinc.org", "code": "1963-8", "display": "Bicarbonate [Moles/volume] in Serum or Plasma"}]},
            {"coding": [{"system": "http://loinc.org", "code": "3094-0", "display": "Urea nitrogen [Mass/volume] in Serum or Plasma"}]},
            {"coding": [{"system": "http://loinc.org", "code": "2160-0", "display": "Creatinine [Mass/volume] in Serum or Plasma"}]}
        ],
        specimen_type=SpecimenType.SERUM,
        fasting_required=True,
        special_instructions="Patient should fast for 8-12 hours before collection"
    )
}
