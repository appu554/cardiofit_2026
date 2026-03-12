"""
Clinical Outcome Tracking System for CAE Learning Foundation

Tracks clinical outcomes and correlates them with CAE assertions for continuous learning.
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

class OutcomeType(Enum):
    """Types of clinical outcomes"""
    BLEEDING_EVENT = "bleeding_event"
    MAJOR_BLEEDING = "major_bleeding"
    GI_BLEEDING = "gi_bleeding"
    ALLERGIC_REACTION = "allergic_reaction"
    ANAPHYLAXIS = "anaphylaxis"
    ACUTE_KIDNEY_INJURY = "acute_kidney_injury"
    THERAPEUTIC_FAILURE = "therapeutic_failure"
    ADVERSE_EVENT_PREVENTED = "adverse_event_prevented"
    ADVERSE_EVENT = "adverse_event"
    MEDICATION_ERROR = "medication_error"
    DOSING_ERROR = "dosing_error"
    DOSE_ADJUSTMENT_SUCCESS = "dose_adjustment_success"

class OutcomeSeverity(Enum):
    """Severity levels for clinical outcomes"""
    NONE = 0
    MILD = 1
    MODERATE = 2
    SEVERE = 3
    CRITICAL = 4
    FATAL = 5

@dataclass
class ClinicalOutcome:
    """Clinical outcome data structure"""
    outcome_id: str
    patient_id: str
    assertion_id: str
    outcome_type: OutcomeType
    severity: OutcomeSeverity
    outcome_date: datetime
    description: Optional[str] = None
    related_medications: Optional[List[str]] = None
    clinician_id: Optional[str] = None
    facility_id: Optional[str] = None
    created_at: datetime = None
    
    def __post_init__(self):
        if self.created_at is None:
            self.created_at = datetime.utcnow()

class OutcomeTracker:
    """Clinical outcome tracking and learning system"""
    
    def __init__(self):
        self.graphdb_client = graphdb_client
        self.enabled = settings.OUTCOME_TRACKING_ENABLED
        
        logger.info(f"Outcome Tracker initialized - Enabled: {self.enabled}")
    
    async def track_outcome(self, outcome: ClinicalOutcome) -> bool:
        """Track a clinical outcome and store in GraphDB"""
        if not self.enabled:
            logger.debug("Outcome tracking disabled")
            return False
        
        try:
            # Store outcome in GraphDB
            turtle_data = self._generate_outcome_turtle(outcome)
            result = await self.graphdb_client.insert_data(turtle_data)
            
            if result.success:
                logger.info(f"Clinical outcome tracked: {outcome.outcome_id} - {outcome.outcome_type.value}")
                
                # Update assertion confidence based on outcome
                await self._update_assertion_confidence(outcome)
                
                # Update interaction confidence if applicable
                await self._update_interaction_confidence(outcome)
                
                return True
            else:
                logger.error(f"Failed to track outcome: {result.error}")
                return False
                
        except Exception as e:
            logger.error(f"Error tracking outcome: {e}")
            return False
    
    def _generate_outcome_turtle(self, outcome: ClinicalOutcome) -> str:
        """Generate Turtle RDF for clinical outcome"""
        outcome_uri = f"cae:outcome_{outcome.outcome_id}"
        
        turtle = f"""
        @prefix cae: <http://clinical-assertion-engine.org/ontology/> .
        @prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

        {outcome_uri} a cae:ClinicalOutcome ;
            cae:hasOutcomeId "{outcome.outcome_id}" ;
            cae:hasPatientId "{outcome.patient_id}" ;
            cae:hasOutcomeType "{outcome.outcome_type.value}" ;
            cae:hasOutcomeSeverity {outcome.severity.value} ;
            cae:hasOutcomeDate "{outcome.outcome_date.isoformat()}Z"^^xsd:dateTime ;
            cae:resultedFrom cae:assertion_{outcome.assertion_id} ;
            cae:hasCreatedAt "{outcome.created_at.isoformat()}Z"^^xsd:dateTime"""

        # Add semicolon if we're going to append more properties
        has_additional_properties = (outcome.description or outcome.clinician_id or
                                   outcome.facility_id or outcome.related_medications)
        if has_additional_properties:
            turtle += " ;"

        if outcome.description:
            # Escape quotes in description
            escaped_description = outcome.description.replace('"', '\\"')
            turtle += f'\n        cae:hasDescription "{escaped_description}" ;'
        
        if outcome.clinician_id:
            turtle += f'\n        cae:reportedBy cae:clinician_{outcome.clinician_id} ;'
        
        if outcome.facility_id:
            turtle += f'\n        cae:occurredAt cae:facility_{outcome.facility_id} ;'
        
        if outcome.related_medications:
            for med in outcome.related_medications:
                turtle += f'\n        cae:involvedMedication cae:{med.replace(" ", "_").lower()} ;'
        
        turtle = turtle.rstrip(' ;') + ' .'

        # Debug: Log the generated turtle
        logger.debug(f"Generated turtle for outcome {outcome.outcome_id}:\n{turtle}")

        return turtle
    
    async def _update_assertion_confidence(self, outcome: ClinicalOutcome):
        """Update assertion confidence based on outcome"""
        try:
            # Calculate confidence adjustment based on outcome
            confidence_delta = self._calculate_confidence_delta(outcome)
            
            if confidence_delta != 0:
                update_query = f"""
                PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

                DELETE {{ ?assertion cae:hasAssertionConfidence ?oldConfidence }}
                INSERT {{ ?assertion cae:hasAssertionConfidence ?newConfidence }}
                WHERE {{
                    ?assertion cae:hasAssertionId "{outcome.assertion_id}" ;
                               cae:hasAssertionConfidence ?oldConfidence .

                    BIND(IF(?oldConfidence + {confidence_delta} > 1.0, 1.0,
                         IF(?oldConfidence + {confidence_delta} < 0.0, 0.0,
                            ?oldConfidence + {confidence_delta})) AS ?newConfidence)
                }}
                """
                
                result = await self.graphdb_client.update(update_query)
                if result.success:
                    logger.debug(f"Updated assertion confidence for {outcome.assertion_id}")
                else:
                    logger.error(f"Failed to update assertion confidence: {result.error}")
        
        except Exception as e:
            logger.error(f"Error updating assertion confidence: {e}")
    
    async def _update_interaction_confidence(self, outcome: ClinicalOutcome):
        """Update drug interaction confidence based on outcome"""
        try:
            if outcome.related_medications and len(outcome.related_medications) >= 2:
                # Find the interaction between medications
                med1, med2 = outcome.related_medications[0], outcome.related_medications[1]
                
                confidence_delta = self._calculate_confidence_delta(outcome)
                
                if confidence_delta != 0:
                    update_query = f"""
                    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>

                    DELETE {{ ?interaction cae:hasConfidenceScore ?oldScore }}
                    INSERT {{ ?interaction cae:hasConfidenceScore ?newScore }}
                    WHERE {{
                        ?med1 cae:hasGenericName "{med1}" .
                        ?med2 cae:hasGenericName "{med2}" .
                        ?med1 cae:interactsWith ?med2 .
                        ?interaction a cae:DrugInteraction ;
                                     cae:hasConfidenceScore ?oldScore .

                        BIND(IF(?oldScore + {confidence_delta} > 1.0, 1.0,
                             IF(?oldScore + {confidence_delta} < 0.0, 0.0,
                                ?oldScore + {confidence_delta})) AS ?newScore)
                    }}
                    """
                    
                    result = await self.graphdb_client.update(update_query)
                    if result.success:
                        logger.debug(f"Updated interaction confidence for {med1} + {med2}")
        
        except Exception as e:
            logger.error(f"Error updating interaction confidence: {e}")
    
    def _calculate_confidence_delta(self, outcome: ClinicalOutcome) -> float:
        """Calculate confidence adjustment based on outcome"""
        if outcome.outcome_type == OutcomeType.ADVERSE_EVENT_PREVENTED:
            # Positive outcome - increase confidence
            return 0.05
        elif outcome.severity in [OutcomeSeverity.SEVERE, OutcomeSeverity.CRITICAL, OutcomeSeverity.FATAL]:
            # Severe negative outcome - increase confidence (alert was correct)
            return 0.1
        elif outcome.severity in [OutcomeSeverity.MILD, OutcomeSeverity.MODERATE]:
            # Moderate negative outcome - slight increase
            return 0.02
        elif outcome.outcome_type == OutcomeType.THERAPEUTIC_FAILURE:
            # Therapeutic failure - decrease confidence
            return -0.05
        else:
            return 0.0
    
    async def get_outcomes_for_assertion(self, assertion_id: str) -> List[Dict[str, Any]]:
        """Get all outcomes for a specific assertion"""
        query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?outcome ?outcomeType ?severity ?date ?description WHERE {{
            ?outcome a cae:ClinicalOutcome ;
                     cae:resultedFrom cae:assertion_{assertion_id} ;
                     cae:hasOutcomeType ?outcomeType ;
                     cae:hasOutcomeSeverity ?severity ;
                     cae:hasOutcomeDate ?date .
            
            OPTIONAL {{ ?outcome cae:hasDescription ?description }}
        }}
        ORDER BY DESC(?date)
        """
        
        result = await self.graphdb_client.query(query)
        
        if result.success and result.data:
            return result.data.get('results', {}).get('bindings', [])
        else:
            return []
    
    async def get_outcome_statistics(self, days: int = 30) -> Dict[str, Any]:
        """Get outcome statistics for the last N days"""
        start_date = (datetime.utcnow() - timedelta(days=days)).isoformat()
        
        query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?outcomeType 
               (COUNT(?outcome) AS ?count)
               (AVG(?severity) AS ?avgSeverity) WHERE {{
            ?outcome a cae:ClinicalOutcome ;
                     cae:hasOutcomeType ?outcomeType ;
                     cae:hasOutcomeSeverity ?severity ;
                     cae:hasOutcomeDate ?date .
            
            FILTER(?date >= "{start_date}Z"^^xsd:dateTime)
        }}
        GROUP BY ?outcomeType
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
    
    async def create_test_outcome(self, patient_id: str, assertion_id: str) -> ClinicalOutcome:
        """Create a test outcome for demonstration"""
        outcome = ClinicalOutcome(
            outcome_id=str(uuid.uuid4()),
            patient_id=patient_id,
            assertion_id=assertion_id,
            outcome_type=OutcomeType.BLEEDING_EVENT,
            severity=OutcomeSeverity.MODERATE,
            outcome_date=datetime.utcnow(),
            description="Minor bleeding event observed after warfarin + aspirin combination",
            related_medications=["warfarin", "aspirin"],
            clinician_id="clinician_001"
        )
        
        success = await self.track_outcome(outcome)
        if success:
            logger.info(f"Test outcome created: {outcome.outcome_id}")
            return outcome
        else:
            raise Exception("Failed to create test outcome")

# Global outcome tracker instance
outcome_tracker = OutcomeTracker()
