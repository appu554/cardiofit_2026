"""
Clinical Override Tracking System for CAE Learning Foundation

Tracks clinician overrides of CAE assertions for continuous learning and improvement.
"""

import asyncio
import logging
import uuid
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, asdict
from enum import Enum

from app.graph.graphdb_client import graphdb_client, GraphDBResult
from app.core.config import settings

logger = logging.getLogger(__name__)

class OverrideReason(Enum):
    """Common override reasons"""
    PATIENT_STABLE = "patient_stable_on_combination"
    EMERGENCY_SITUATION = "emergency_situation_benefits_outweigh_risks"
    NO_ALTERNATIVES = "no_alternative_treatment_options"
    CLINICAL_JUDGMENT = "clinical_judgment_override"
    PATIENT_PREFERENCE = "patient_preference"
    SPECIALIST_CONSULTATION = "specialist_consultation_approved"
    TEMPORARY_USE = "temporary_use_only"
    MONITORING_AVAILABLE = "enhanced_monitoring_available"
    FALSE_POSITIVE = "false_positive_alert"
    OUTDATED_INFORMATION = "outdated_clinical_information"

@dataclass
class ClinicalOverride:
    """Clinical override data structure"""
    override_id: str
    assertion_id: str
    patient_id: str
    clinician_id: str
    override_reason: str
    override_timestamp: datetime
    custom_reason: Optional[str] = None
    follow_up_required: bool = False
    monitoring_plan: Optional[str] = None
    expected_duration: Optional[str] = None
    facility_id: Optional[str] = None
    created_at: datetime = None
    
    def __post_init__(self):
        if self.created_at is None:
            self.created_at = datetime.utcnow()

@dataclass
class OverridePattern:
    """Override pattern analysis result"""
    assertion_type: str
    override_rate: float
    common_reasons: List[str]
    clinician_patterns: Dict[str, float]
    temporal_patterns: Dict[str, int]

class OverrideTracker:
    """Clinical override tracking and learning system"""
    
    def __init__(self):
        self.graphdb_client = graphdb_client
        self.enabled = settings.OVERRIDE_TRACKING_ENABLED
        
        logger.info(f"Override Tracker initialized - Enabled: {self.enabled}")
    
    async def track_override(self, override: ClinicalOverride) -> bool:
        """Track a clinical override and store in GraphDB"""
        if not self.enabled:
            logger.debug("Override tracking disabled")
            return False
        
        try:
            # Store override in GraphDB
            turtle_data = self._generate_override_turtle(override)
            result = await self.graphdb_client.insert_data(turtle_data)
            
            if result.success:
                logger.info(f"Clinical override tracked: {override.override_id} - {override.override_reason}")
                
                # Update assertion override rate
                await self._update_assertion_override_rate(override)
                
                # Update interaction override rate if applicable
                await self._update_interaction_override_rate(override)
                
                # Analyze override patterns
                await self._analyze_override_patterns(override)
                
                return True
            else:
                logger.error(f"Failed to track override: {result.error}")
                return False
                
        except Exception as e:
            logger.error(f"Error tracking override: {e}")
            return False
    
    def _generate_override_turtle(self, override: ClinicalOverride) -> str:
        """Generate Turtle RDF for clinical override"""
        override_uri = f"cae:override_{override.override_id}"
        
        turtle = f"""
        @prefix cae: <http://clinical-assertion-engine.org/ontology/> .
        @prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

        {override_uri} a cae:ClinicalOverride ;
            cae:hasOverrideId "{override.override_id}" ;
            cae:hasPatientId "{override.patient_id}" ;
            cae:hasOverrideReason "{override.override_reason}" ;
            cae:hasOverrideTimestamp "{override.override_timestamp.isoformat()}Z"^^xsd:dateTime ;
            cae:overrodeAssertion cae:assertion_{override.assertion_id} ;
            cae:performedBy cae:clinician_{override.clinician_id} ;
            cae:hasCreatedAt "{override.created_at.isoformat()}Z"^^xsd:dateTime"""

        # Add semicolon if we're going to append more properties
        has_additional_properties = (override.custom_reason or override.follow_up_required or
                                   override.monitoring_plan or override.expected_duration or
                                   override.facility_id)
        if has_additional_properties:
            turtle += " ;"

        if override.custom_reason:
            # Escape quotes in custom reason
            escaped_reason = override.custom_reason.replace('"', '\\"')
            turtle += f'\n        cae:hasCustomReason "{escaped_reason}" ;'
        
        if override.follow_up_required:
            turtle += f'\n        cae:requiresFollowUp true ;'
        
        if override.monitoring_plan:
            # Escape quotes in monitoring plan
            escaped_plan = override.monitoring_plan.replace('"', '\\"')
            turtle += f'\n        cae:hasMonitoringPlan "{escaped_plan}" ;'
        
        if override.expected_duration:
            turtle += f'\n        cae:hasExpectedDuration "{override.expected_duration}" ;'
        
        if override.facility_id:
            turtle += f'\n        cae:occurredAt cae:facility_{override.facility_id} ;'
        
        turtle = turtle.rstrip(' ;') + ' .'

        # Debug: Log the generated turtle
        logger.debug(f"Generated turtle for override {override.override_id}:\n{turtle}")

        return turtle
    
    async def _update_assertion_override_rate(self, override: ClinicalOverride):
        """Update assertion override rate statistics"""
        try:
            # Get current override count for this assertion type
            query = f"""
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT (COUNT(?override) AS ?overrideCount) WHERE {{
                ?assertion cae:hasAssertionId "{override.assertion_id}" ;
                           cae:hasAssertionType ?assertionType .
                ?override cae:overrodeAssertion ?assertion .
            }}
            """
            
            result = await self.graphdb_client.query(query)
            if result.success and result.data:
                bindings = result.data.get('results', {}).get('bindings', [])
                if bindings:
                    override_count = int(bindings[0]['overrideCount']['value'])
                    
                    # Update override rate (simplified calculation)
                    new_override_rate = min(1.0, override_count * 0.01)  # 1% per override, max 100%
                    
                    update_query = f"""
                    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
                    
                    DELETE {{ ?assertion cae:hasOverrideRate ?oldRate }}
                    INSERT {{ ?assertion cae:hasOverrideRate {new_override_rate} }}
                    WHERE {{
                        ?assertion cae:hasAssertionId "{override.assertion_id}" .
                        OPTIONAL {{ ?assertion cae:hasOverrideRate ?oldRate }}
                    }}
                    """
                    
                    await self.graphdb_client.update(update_query)
                    logger.debug(f"Updated override rate for assertion {override.assertion_id}: {new_override_rate}")
        
        except Exception as e:
            logger.error(f"Error updating assertion override rate: {e}")
    
    async def _update_interaction_override_rate(self, override: ClinicalOverride):
        """Update drug interaction override rate"""
        try:
            # Get the assertion details to find related medications
            query = f"""
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?assertionType WHERE {{
                ?assertion cae:hasAssertionId "{override.assertion_id}" ;
                           cae:hasAssertionType ?assertionType .
            }}
            """
            
            result = await self.graphdb_client.query(query)
            if result.success and result.data:
                bindings = result.data.get('results', {}).get('bindings', [])
                if bindings and bindings[0]['assertionType']['value'] == 'drug_interaction':
                    # Update interaction override rate
                    update_query = f"""
                    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
                    
                    DELETE {{ ?interaction cae:hasOverrideRate ?oldRate }}
                    INSERT {{ ?interaction cae:hasOverrideRate ?newRate }}
                    WHERE {{
                        ?interaction a cae:DrugInteraction ;
                                     cae:hasOverrideRate ?oldRate .
                        
                        BIND(?oldRate + 0.01 AS ?newRate)
                    }}
                    """
                    
                    await self.graphdb_client.update(update_query)
                    logger.debug(f"Updated interaction override rates")
        
        except Exception as e:
            logger.error(f"Error updating interaction override rate: {e}")
    
    async def _analyze_override_patterns(self, override: ClinicalOverride):
        """Analyze override patterns for learning"""
        try:
            # This could be expanded to include more sophisticated pattern analysis
            logger.debug(f"Analyzing override patterns for {override.override_id}")
            
            # Simple pattern: track override reasons by clinician
            pattern_query = f"""
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?reason (COUNT(?override) AS ?count) WHERE {{
                ?override a cae:ClinicalOverride ;
                          cae:performedBy cae:clinician_{override.clinician_id} ;
                          cae:hasOverrideReason ?reason .
            }}
            GROUP BY ?reason
            ORDER BY DESC(?count)
            """
            
            result = await self.graphdb_client.query(pattern_query)
            if result.success:
                logger.debug(f"Override pattern analysis completed for clinician {override.clinician_id}")
        
        except Exception as e:
            logger.error(f"Error analyzing override patterns: {e}")
    
    async def get_overrides_for_assertion(self, assertion_id: str) -> List[Dict[str, Any]]:
        """Get all overrides for a specific assertion"""
        query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?override ?reason ?timestamp ?clinician ?customReason WHERE {{
            ?override a cae:ClinicalOverride ;
                      cae:overrodeAssertion cae:assertion_{assertion_id} ;
                      cae:hasOverrideReason ?reason ;
                      cae:hasOverrideTimestamp ?timestamp ;
                      cae:performedBy ?clinician .
            
            OPTIONAL {{ ?override cae:hasCustomReason ?customReason }}
        }}
        ORDER BY DESC(?timestamp)
        """
        
        result = await self.graphdb_client.query(query)
        
        if result.success and result.data:
            return result.data.get('results', {}).get('bindings', [])
        else:
            return []
    
    async def get_override_statistics(self, days: int = 30) -> Dict[str, Any]:
        """Get override statistics for the last N days"""
        start_date = (datetime.utcnow() - timedelta(days=days)).isoformat()
        
        query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?reason 
               (COUNT(?override) AS ?count)
               (COUNT(DISTINCT ?clinician) AS ?clinicianCount) WHERE {{
            ?override a cae:ClinicalOverride ;
                      cae:hasOverrideReason ?reason ;
                      cae:hasOverrideTimestamp ?timestamp ;
                      cae:performedBy ?clinician .
            
            FILTER(?timestamp >= "{start_date}Z"^^xsd:dateTime)
        }}
        GROUP BY ?reason
        ORDER BY DESC(?count)
        """
        
        result = await self.graphdb_client.query(query)
        
        if result.success and result.data:
            return {
                "period_days": days,
                "statistics": result.data.get('results', {}).get('bindings', [])
            }
        else:
            return {"period_days": days, "statistics": []}
    
    async def analyze_clinician_patterns(self, clinician_id: str) -> Dict[str, Any]:
        """Analyze override patterns for a specific clinician"""
        query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?assertionType ?reason 
               (COUNT(?override) AS ?count)
               (AVG(?confidence) AS ?avgConfidence) WHERE {{
            ?override a cae:ClinicalOverride ;
                      cae:performedBy cae:clinician_{clinician_id} ;
                      cae:hasOverrideReason ?reason ;
                      cae:overrodeAssertion ?assertion .
            
            ?assertion cae:hasAssertionType ?assertionType ;
                       cae:hasAssertionConfidence ?confidence .
        }}
        GROUP BY ?assertionType ?reason
        ORDER BY DESC(?count)
        """
        
        result = await self.graphdb_client.query(query)
        
        if result.success and result.data:
            return {
                "clinician_id": clinician_id,
                "patterns": result.data.get('results', {}).get('bindings', [])
            }
        else:
            return {"clinician_id": clinician_id, "patterns": []}
    
    async def create_test_override(self, patient_id: str, assertion_id: str, clinician_id: str = "clinician_001") -> ClinicalOverride:
        """Create a test override for demonstration"""
        override = ClinicalOverride(
            override_id=str(uuid.uuid4()),
            assertion_id=assertion_id,
            patient_id=patient_id,
            clinician_id=clinician_id,
            override_reason=OverrideReason.PATIENT_STABLE.value,
            override_timestamp=datetime.utcnow(),
            custom_reason="Patient has been stable on this combination for 2 years with regular monitoring",
            follow_up_required=True,
            monitoring_plan="Weekly INR monitoring for 4 weeks",
            expected_duration="Ongoing with monitoring"
        )
        
        success = await self.track_override(override)
        if success:
            logger.info(f"Test override created: {override.override_id}")
            return override
        else:
            raise Exception("Failed to create test override")

# Global override tracker instance
override_tracker = OverrideTracker()
