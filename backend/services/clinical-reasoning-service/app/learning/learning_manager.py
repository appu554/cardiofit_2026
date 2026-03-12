"""
Learning Foundation Manager for CAE

Coordinates outcome tracking, override tracking, and continuous learning processes.
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional

from app.learning.outcome_tracker import outcome_tracker, ClinicalOutcome, OutcomeType, OutcomeSeverity
from app.learning.override_tracker import override_tracker, ClinicalOverride, OverrideReason
from app.graph.graphdb_client import graphdb_client
from app.core.config import settings

logger = logging.getLogger(__name__)

class LearningManager:
    """Central manager for CAE learning foundation"""
    
    def __init__(self):
        self.outcome_tracker = outcome_tracker
        self.override_tracker = override_tracker
        self.graphdb_client = graphdb_client
        
        self.learning_enabled = settings.LEARNING_ENABLED
        self.update_interval = settings.LEARNING_UPDATE_INTERVAL
        
        # Learning statistics
        self.stats = {
            "outcomes_tracked": 0,
            "overrides_tracked": 0,
            "confidence_updates": 0,
            "last_update": None
        }
        
        logger.info(f"Learning Manager initialized - Enabled: {self.learning_enabled}")
    
    async def initialize(self):
        """Initialize learning foundation"""
        if not self.learning_enabled:
            logger.info("Learning foundation disabled")
            return
        
        try:
            # Test GraphDB connection
            await self.graphdb_client.connect()
            
            # Initialize learning components
            logger.info("Learning foundation initialized successfully")
            
            # Start background learning process
            asyncio.create_task(self._background_learning_process())
            
        except Exception as e:
            logger.error(f"Failed to initialize learning foundation: {e}")
            raise
    
    async def track_clinical_outcome(
        self,
        patient_id: str,
        assertion_id: str,
        outcome_type: str,
        severity: int,
        description: Optional[str] = None,
        related_medications: Optional[List[str]] = None,
        clinician_id: Optional[str] = None
    ) -> bool:
        """Track a clinical outcome"""
        try:
            outcome = ClinicalOutcome(
                outcome_id=f"outcome_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}_{patient_id[:8]}",
                patient_id=patient_id,
                assertion_id=assertion_id,
                outcome_type=OutcomeType(outcome_type),
                severity=OutcomeSeverity(severity),
                outcome_date=datetime.utcnow(),
                description=description,
                related_medications=related_medications,
                clinician_id=clinician_id
            )
            
            success = await self.outcome_tracker.track_outcome(outcome)
            if success:
                self.stats["outcomes_tracked"] += 1
                logger.info(f"Clinical outcome tracked: {outcome.outcome_id}")
            
            return success
            
        except Exception as e:
            logger.error(f"Error tracking clinical outcome: {e}")
            return False
    
    async def track_clinician_override(
        self,
        patient_id: str,
        assertion_id: str,
        clinician_id: str,
        override_reason: str,
        custom_reason: Optional[str] = None,
        follow_up_required: bool = False,
        monitoring_plan: Optional[str] = None
    ) -> bool:
        """Track a clinician override"""
        try:
            override = ClinicalOverride(
                override_id=f"override_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}_{clinician_id}",
                assertion_id=assertion_id,
                patient_id=patient_id,
                clinician_id=clinician_id,
                override_reason=override_reason,
                override_timestamp=datetime.utcnow(),
                custom_reason=custom_reason,
                follow_up_required=follow_up_required,
                monitoring_plan=monitoring_plan
            )
            
            success = await self.override_tracker.track_override(override)
            if success:
                self.stats["overrides_tracked"] += 1
                logger.info(f"Clinician override tracked: {override.override_id}")
            
            return success
            
        except Exception as e:
            logger.error(f"Error tracking clinician override: {e}")
            return False
    
    async def get_learning_insights(self, patient_id: Optional[str] = None) -> Dict[str, Any]:
        """Get learning insights and statistics"""
        try:
            insights = {
                "learning_stats": self.stats.copy(),
                "outcome_statistics": await self.outcome_tracker.get_outcome_statistics(),
                "override_statistics": await self.override_tracker.get_override_statistics(),
                "confidence_trends": await self._get_confidence_trends(),
                "generated_at": datetime.utcnow().isoformat()
            }
            
            if patient_id:
                insights["patient_specific"] = await self._get_patient_insights(patient_id)
            
            return insights
            
        except Exception as e:
            logger.error(f"Error getting learning insights: {e}")
            return {"error": str(e)}
    
    async def _get_confidence_trends(self) -> Dict[str, Any]:
        """Get confidence score trends"""
        try:
            query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?interactionType ?avgConfidence ?overrideRate WHERE {
                ?interaction a cae:DrugInteraction ;
                             cae:hasInteractionSeverity ?interactionType ;
                             cae:hasConfidenceScore ?avgConfidence ;
                             cae:hasOverrideRate ?overrideRate .
            }
            ORDER BY DESC(?avgConfidence)
            """
            
            result = await self.graphdb_client.query(query)
            
            if result.success and result.data:
                return {
                    "trends": result.data.get('results', {}).get('bindings', [])
                }
            else:
                return {"trends": []}
                
        except Exception as e:
            logger.error(f"Error getting confidence trends: {e}")
            return {"trends": [], "error": str(e)}
    
    async def _get_patient_insights(self, patient_id: str) -> Dict[str, Any]:
        """Get patient-specific learning insights"""
        try:
            # Get patient's assertions
            query = f"""
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?assertion ?assertionType ?confidence ?severity WHERE {{
                ?assertion a cae:ClinicalAssertion ;
                           cae:hasAssertionType ?assertionType ;
                           cae:hasAssertionConfidence ?confidence ;
                           cae:hasAssertionSeverity ?severity .
                
                # Filter by patient (this would need to be enhanced based on actual data model)
                FILTER(CONTAINS(STR(?assertion), "{patient_id}"))
            }}
            ORDER BY DESC(?confidence)
            """
            
            result = await self.graphdb_client.query(query)
            
            insights = {
                "patient_id": patient_id,
                "assertions": [],
                "outcomes": [],
                "overrides": []
            }
            
            if result.success and result.data:
                insights["assertions"] = result.data.get('results', {}).get('bindings', [])
            
            return insights
            
        except Exception as e:
            logger.error(f"Error getting patient insights: {e}")
            return {"patient_id": patient_id, "error": str(e)}
    
    async def _background_learning_process(self):
        """Background process for continuous learning"""
        logger.info("Starting background learning process")
        
        while self.learning_enabled:
            try:
                await asyncio.sleep(self.update_interval)
                
                # Perform periodic learning updates
                await self._update_confidence_scores()
                await self._analyze_patterns()
                
                self.stats["last_update"] = datetime.utcnow().isoformat()
                logger.debug("Background learning update completed")
                
            except Exception as e:
                logger.error(f"Error in background learning process: {e}")
                await asyncio.sleep(60)  # Wait before retrying
    
    async def _update_confidence_scores(self):
        """Update confidence scores based on recent outcomes"""
        try:
            # This is a simplified version - could be much more sophisticated
            query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?interaction ?currentScore (COUNT(?outcome) AS ?outcomeCount) WHERE {
                ?interaction a cae:DrugInteraction ;
                             cae:hasConfidenceScore ?currentScore .
                
                OPTIONAL {
                    ?outcome a cae:ClinicalOutcome ;
                             cae:hasOutcomeDate ?date .
                    
                    FILTER(?date >= NOW() - "P7D"^^xsd:duration)
                }
            }
            GROUP BY ?interaction ?currentScore
            """
            
            result = await self.graphdb_client.query(query)
            
            if result.success:
                self.stats["confidence_updates"] += 1
                logger.debug("Confidence scores updated")
                
        except Exception as e:
            logger.error(f"Error updating confidence scores: {e}")
    
    async def _analyze_patterns(self):
        """Analyze patterns in outcomes and overrides"""
        try:
            # Analyze override patterns
            override_stats = await self.override_tracker.get_override_statistics(days=7)
            
            # Analyze outcome patterns
            outcome_stats = await self.outcome_tracker.get_outcome_statistics(days=7)
            
            logger.debug(f"Pattern analysis completed - Overrides: {len(override_stats.get('statistics', []))}, Outcomes: {len(outcome_stats.get('statistics', []))}")
            
        except Exception as e:
            logger.error(f"Error analyzing patterns: {e}")
    
    async def create_demo_data(self, patient_id: str = None) -> Dict[str, Any]:
        """Create demonstration data for testing"""
        if not patient_id:
            patient_id = settings.PRIMARY_TEST_PATIENT_ID
        
        try:
            demo_results = {
                "patient_id": patient_id,
                "created_outcomes": [],
                "created_overrides": []
            }
            
            # Create test assertion ID
            assertion_id = f"assert_demo_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"
            
            # Create test outcome
            outcome_success = await self.track_clinical_outcome(
                patient_id=patient_id,
                assertion_id=assertion_id,
                outcome_type=OutcomeType.BLEEDING_EVENT.value,
                severity=OutcomeSeverity.MODERATE.value,
                description="Demo bleeding event for testing learning system",
                related_medications=["warfarin", "aspirin"],
                clinician_id="clinician_001"
            )
            
            if outcome_success:
                demo_results["created_outcomes"].append(assertion_id)
            
            # Create test override
            override_success = await self.track_clinician_override(
                patient_id=patient_id,
                assertion_id=assertion_id,
                clinician_id="clinician_001",
                override_reason=OverrideReason.PATIENT_STABLE.value,
                custom_reason="Demo override for testing learning system",
                follow_up_required=True,
                monitoring_plan="Weekly monitoring for demo purposes"
            )
            
            if override_success:
                demo_results["created_overrides"].append(assertion_id)
            
            logger.info(f"Demo data created for patient {patient_id}")
            return demo_results
            
        except Exception as e:
            logger.error(f"Error creating demo data: {e}")
            return {"error": str(e)}

# Global learning manager instance
learning_manager = LearningManager()
