"""
Data models for Module 8 Kafka topics

Models based on MODULE_8_HYBRID_ARCHITECTURE_IMPLEMENTATION_PLAN.md
"""

from typing import Dict, List, Optional, Any
from pydantic import BaseModel, Field


# ============================================================================
# EnrichedClinicalEvent Models (from prod.ehr.events.enriched)
# ============================================================================

class RawData(BaseModel):
    """Raw clinical data from devices/systems"""
    heart_rate: Optional[int] = None
    blood_pressure_systolic: Optional[int] = None
    blood_pressure_diastolic: Optional[int] = None
    spo2: Optional[int] = None
    temperature_celsius: Optional[float] = None

    # Allow additional fields for other event types
    class Config:
        extra = "allow"


class ClinicalContext(BaseModel):
    """Clinical context enrichment"""
    active_conditions: List[str] = Field(default_factory=list)
    current_medications: List[str] = Field(default_factory=list)
    recent_procedures: List[str] = Field(default_factory=list)


class Enrichments(BaseModel):
    """Clinical enrichments and scores"""
    NEWS2Score: Optional[int] = None
    qSOFAScore: Optional[int] = None
    riskLevel: Optional[str] = None
    clinical_context: Optional[ClinicalContext] = Field(
        default=None,
        alias="clinicalContext"
    )

    class Config:
        populate_by_name = True


class SemanticAnnotations(BaseModel):
    """Semantic terminology annotations"""
    SNOMED_CT: List[str] = Field(default_factory=list)
    LOINC: List[str] = Field(default_factory=list)

    class Config:
        extra = "allow"


class MLPredictions(BaseModel):
    """Machine learning predictions"""
    sepsis_risk_24h: Optional[float] = None
    cardiac_event_risk_7d: Optional[float] = None
    readmission_risk_30d: Optional[float] = None

    class Config:
        extra = "allow"


class EnrichedClinicalEvent(BaseModel):
    """
    Enriched clinical event from prod.ehr.events.enriched topic

    Source: TransactionalMultiSinkRouter.java line 89-104

    Note: Fields are flexible to handle Module 6 Flink output variations
    """
    id: str
    timestamp: int
    event_type: str = Field(alias="eventType")
    patient_id: Optional[str] = Field(default=None, alias="patientId")
    encounter_id: Optional[str] = Field(default=None, alias="encounterId")
    department_id: Optional[str] = Field(default=None, alias="departmentId")
    device_id: Optional[str] = Field(default=None, alias="deviceId")

    raw_data: Optional[RawData] = Field(default=None, alias="rawData")
    enrichments: Optional[Enrichments] = None
    semantic_annotations: Optional[SemanticAnnotations] = Field(
        default=None,
        alias="semanticAnnotations"
    )
    ml_predictions: Optional[MLPredictions] = Field(
        default=None,
        alias="mlPredictions"
    )

    class Config:
        populate_by_name = True
        extra = "allow"


# ============================================================================
# FHIRResource Models (from prod.ehr.fhir.upsert)
# ============================================================================

class FHIRCoding(BaseModel):
    """FHIR Coding datatype"""
    system: str
    code: str
    display: Optional[str] = None


class FHIRCodeableConcept(BaseModel):
    """FHIR CodeableConcept datatype"""
    coding: List[FHIRCoding] = Field(default_factory=list)
    text: Optional[str] = None


class FHIRQuantity(BaseModel):
    """FHIR Quantity datatype"""
    value: float
    unit: str
    system: str
    code: str


class FHIRResource(BaseModel):
    """
    FHIR resource wrapper from prod.ehr.fhir.upsert topic

    Source: TransactionalMultiSinkRouter.java line 324-382

    Note: fhir_data contains the complete FHIR R4 resource
    """
    resource_type: str = Field(alias="resourceType")
    resource_id: str = Field(alias="resourceId")
    patient_id: str = Field(alias="patientId")
    last_updated: int = Field(alias="lastUpdated")

    # Complete FHIR R4 resource as dict
    fhir_data: Dict[str, Any] = Field(alias="fhirData")

    class Config:
        populate_by_name = True

    def get_kafka_key(self) -> str:
        """Generate Kafka key: {resourceType}|{resourceId}"""
        return f"{self.resource_type}|{self.resource_id}"


# ============================================================================
# GraphMutation Models (from prod.ehr.graph.mutations)
# ============================================================================

class Relationship(BaseModel):
    """Graph relationship specification"""
    relation_type: str = Field(alias="relationType")
    target_node_type: str = Field(alias="targetNodeType")
    target_node_id: str = Field(alias="targetNodeId")
    relationship_properties: Dict[str, Any] = Field(
        default_factory=dict,
        alias="relationshipProperties"
    )

    class Config:
        populate_by_name = True


class GraphMutation(BaseModel):
    """
    Graph mutation from prod.ehr.graph.mutations topic

    Source: TransactionalMultiSinkRouter.java line 386-417
    """
    mutation_type: str = Field(alias="mutationType")
    node_type: str = Field(alias="nodeType")
    node_id: str = Field(alias="nodeId")
    timestamp: int

    node_properties: Dict[str, Any] = Field(
        default_factory=dict,
        alias="nodeProperties"
    )
    relationships: List[Relationship] = Field(default_factory=list)

    class Config:
        populate_by_name = True

    def get_kafka_key(self) -> str:
        """Generate Kafka key: {nodeId}"""
        return self.node_id
