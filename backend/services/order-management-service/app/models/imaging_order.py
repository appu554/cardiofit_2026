"""
Imaging Order Models for Order Management Service

This module provides FHIR-compliant models for imaging orders,
implementing the FHIR ServiceRequest resource for diagnostic imaging.
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

# Imaging-specific enums
class ImagingModality(str, Enum):
    """Common imaging modalities"""
    XRAY = "XR"  # X-Ray
    CT = "CT"    # Computed Tomography
    MRI = "MR"   # Magnetic Resonance Imaging
    US = "US"    # Ultrasound
    NM = "NM"    # Nuclear Medicine
    PET = "PT"   # Positron Emission Tomography
    MAMMO = "MG" # Mammography
    FLUORO = "XA" # Fluoroscopy/Angiography
    DEXA = "DX"  # Dual-energy X-ray Absorptiometry
    OTHER = "OT"

class BodySite(str, Enum):
    """Common body sites for imaging"""
    HEAD = "head"
    NECK = "neck"
    CHEST = "chest"
    ABDOMEN = "abdomen"
    PELVIS = "pelvis"
    SPINE = "spine"
    UPPER_EXTREMITY = "upper-extremity"
    LOWER_EXTREMITY = "lower-extremity"
    BREAST = "breast"
    HEART = "heart"
    BRAIN = "brain"
    WHOLE_BODY = "whole-body"

class Laterality(str, Enum):
    """Laterality for bilateral structures"""
    LEFT = "left"
    RIGHT = "right"
    BILATERAL = "bilateral"
    UNILATERAL = "unilateral"

class ContrastType(str, Enum):
    """Types of contrast agents"""
    NONE = "none"
    ORAL = "oral"
    IV = "intravenous"
    RECTAL = "rectal"
    INTRATHECAL = "intrathecal"
    INTRA_ARTICULAR = "intra-articular"

class TransportMode(str, Enum):
    """Patient transport modes"""
    AMBULATORY = "ambulatory"
    WHEELCHAIR = "wheelchair"
    STRETCHER = "stretcher"
    BED = "bed"
    ISOLATION = "isolation"

# Imaging Order Model
class ImagingOrder(ClinicalOrder):
    """
    FHIR ServiceRequest resource specialized for imaging orders.
    
    This model extends the base ClinicalOrder with imaging-specific fields.
    """
    
    # Override resource type for imaging orders
    resourceType: str = Field(default="ServiceRequest", description="FHIR resource type")
    
    # Imaging-specific category
    category: Optional[List[CodeableConcept]] = Field(
        default=[{
            "coding": [{
                "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                "code": "imaging",
                "display": "Imaging"
            }]
        }],
        description="Imaging service category"
    )
    
    # Imaging procedure details
    procedure_code: Optional[CodeableConcept] = Field(None, description="Specific imaging procedure code (e.g., CPT, SNOMED CT)")
    procedure_name: Optional[str] = Field(None, description="Human-readable procedure name")
    modality: Optional[ImagingModality] = Field(None, description="Imaging modality")
    
    # Anatomical details
    body_site: Optional[BodySite] = Field(None, description="Body site to be imaged")
    body_site_codeable_concept: Optional[CodeableConcept] = Field(None, description="Coded body site")
    laterality: Optional[Laterality] = Field(None, description="Left/right/bilateral specification")
    
    # Contrast and preparation
    contrast_required: Optional[bool] = Field(None, description="Whether contrast is required")
    contrast_type: Optional[ContrastType] = Field(None, description="Type of contrast agent")
    contrast_allergy_check: Optional[bool] = Field(None, description="Whether contrast allergy was checked")
    prep_instructions: Optional[str] = Field(None, description="Patient preparation instructions")
    
    # Transport and logistics
    transport_mode: Optional[TransportMode] = Field(None, description="Required transport mode")
    isolation_required: Optional[bool] = Field(None, description="Whether isolation precautions are needed")
    portable_exam: Optional[bool] = Field(None, description="Whether portable/bedside exam is needed")
    
    # Clinical context for imaging
    clinical_history_for_radiologist: Optional[str] = Field(None, description="Clinical history for radiologist interpretation")
    clinical_question: Optional[str] = Field(None, description="Specific clinical question to be answered")
    suspected_diagnosis: Optional[List[CodeableConcept]] = Field(None, description="Suspected diagnoses")
    relevant_symptoms: Optional[str] = Field(None, description="Relevant symptoms")
    
    # Previous imaging
    previous_imaging: Optional[List[Reference]] = Field(None, description="References to previous relevant imaging")
    comparison_studies: Optional[str] = Field(None, description="Description of comparison studies")
    
    # Special considerations
    pregnancy_status: Optional[bool] = Field(None, description="Patient pregnancy status (for radiation safety)")
    weight_limit_concerns: Optional[bool] = Field(None, description="Whether patient weight may exceed equipment limits")
    claustrophobia_concerns: Optional[bool] = Field(None, description="Whether patient has claustrophobia concerns")
    pacemaker_implants: Optional[bool] = Field(None, description="Whether patient has pacemaker or other implants")
    
    # Urgency and timing
    stat_reading_required: Optional[bool] = Field(None, description="Whether stat reading is required")
    preferred_time: Optional[str] = Field(None, description="Preferred time for exam")
    
    class Config:
        extra = "allow"
        populate_by_name = True

# Imaging Protocol Model
class ImagingProtocol(FHIRBaseModel):
    """
    Predefined imaging protocols for common procedures.
    """
    protocol_code: str = Field(..., description="Protocol identifier code")
    protocol_name: str = Field(..., description="Human-readable protocol name")
    description: Optional[str] = Field(None, description="Protocol description")
    modality: ImagingModality = Field(..., description="Imaging modality")
    body_site: BodySite = Field(..., description="Target body site")
    contrast_required: bool = Field(default=False, description="Whether contrast is typically required")
    prep_instructions: Optional[str] = Field(None, description="Standard preparation instructions")
    clinical_indications: Optional[List[str]] = Field(None, description="Common clinical indications")
    
    class Config:
        extra = "allow"

# Create and Update models for API endpoints
class ImagingOrderCreate(BaseModel):
    """Model for creating an imaging order"""
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
    
    # Imaging-specific fields
    procedure_code: Optional[CodeableConcept] = None
    procedure_name: Optional[str] = None
    modality: Optional[ImagingModality] = None
    body_site: Optional[BodySite] = None
    body_site_codeable_concept: Optional[CodeableConcept] = None
    laterality: Optional[Laterality] = None
    contrast_required: Optional[bool] = False
    contrast_type: Optional[ContrastType] = None
    contrast_allergy_check: Optional[bool] = None
    prep_instructions: Optional[str] = None
    transport_mode: Optional[TransportMode] = None
    isolation_required: Optional[bool] = False
    portable_exam: Optional[bool] = False
    clinical_history_for_radiologist: Optional[str] = None
    clinical_question: Optional[str] = None
    suspected_diagnosis: Optional[List[CodeableConcept]] = None
    relevant_symptoms: Optional[str] = None
    previous_imaging: Optional[List[Reference]] = None
    comparison_studies: Optional[str] = None
    pregnancy_status: Optional[bool] = None
    weight_limit_concerns: Optional[bool] = False
    claustrophobia_concerns: Optional[bool] = False
    pacemaker_implants: Optional[bool] = False
    stat_reading_required: Optional[bool] = False
    preferred_time: Optional[str] = None
    
    def to_imaging_order(self) -> ImagingOrder:
        """Convert to a FHIR ImagingOrder."""
        data = self.model_dump(exclude_unset=True)
        data["authored_on"] = datetime.utcnow()
        
        # Set imaging-specific category
        data["category"] = [{
            "coding": [{
                "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                "code": "imaging",
                "display": "Imaging"
            }]
        }]
        
        return ImagingOrder(**data)

class ImagingOrderUpdate(BaseModel):
    """Model for updating an imaging order"""
    status: Optional[OrderStatus] = None
    priority: Optional[OrderPriority] = None
    contrast_required: Optional[bool] = None
    contrast_type: Optional[ContrastType] = None
    transport_mode: Optional[TransportMode] = None
    portable_exam: Optional[bool] = None
    clinical_history_for_radiologist: Optional[str] = None
    clinical_question: Optional[str] = None
    prep_instructions: Optional[str] = None
    stat_reading_required: Optional[bool] = None
    preferred_time: Optional[str] = None
    note: Optional[List[Annotation]] = None
