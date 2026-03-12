"""
Outcome Analyzer for Clinical Assertion Engine

Real-time learning from clinical decisions, relationship strength updates,
and confidence score evolution based on clinical outcomes.
"""

import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
import httpx
import json
from dataclasses import dataclass
from enum import Enum

logger = logging.getLogger(__name__)


class OutcomeType(Enum):
    """Types of clinical outcomes"""
    PREVENTED_ERROR = "prevented_error"
    THERAPEUTIC_SUCCESS = "therapeutic_success"
    ADVERSE_EVENT = "adverse_event"
    THERAPEUTIC_FAILURE = "therapeutic_failure"
    INAPPROPRIATE_OVERRIDE = "inappropriate_override"
    APPROPRIATE_OVERRIDE = "appropriate_override"


@dataclass
class ClinicalOutcome:
    """Clinical outcome data"""
    outcome_id: str
    patient_id: str
    assertion_id: str
    clinician_id: Optional[str]
    outcome_type: OutcomeType
    outcome_severity: int  # 1-10 scale
    outcome_description: str
    clinical_context: Dict[str, Any]
    temporal_context: Dict[str, Any]
    recorded_at: datetime
    confidence_impact: float  # How this outcome affects confidence


@dataclass
class LearningUpdate:
    """Learning update for relationships"""
    relationship_id: str
    old_confidence: float
    new_confidence: float
    confidence_delta: float
    evidence_strength: str
    update_reason: str
    supporting_outcomes: List[str]


class OutcomeAnalyzer:
    """
    Real-time outcome analyzer for clinical intelligence learning
    
    Features:
    - Real-time learning from clinical decisions
    - Relationship strength updates based on outcomes
    - Confidence score evolution
    - Pattern validation and refinement
    """
    
    def __init__(self, graphdb_endpoint: str = "http://localhost:7201", 
                 repository: str = "cae-clinical-intelligence"):
        self.graphdb_endpoint = graphdb_endpoint
        self.repository = repository
        self.base_url = f"{graphdb_endpoint}/repositories/{repository}"
        
        # Learning parameters
        self.confidence_learning_rate = 0.1
        self.min_evidence_threshold = 3
        self.outcome_weight_map = {
            OutcomeType.PREVENTED_ERROR: 1.0,
            OutcomeType.THERAPEUTIC_SUCCESS: 0.8,
            OutcomeType.APPROPRIATE_OVERRIDE: 0.6,
            OutcomeType.ADVERSE_EVENT: -0.8,
            OutcomeType.THERAPEUTIC_FAILURE: -0.6,
            OutcomeType.INAPPROPRIATE_OVERRIDE: -1.0
        }
        
        # Learning statistics
        self.learning_stats = {
            "total_outcomes_processed": 0,
            "confidence_updates": 0,
            "relationship_updates": 0,
            "pattern_validations": 0
        }
        
        logger.info("Outcome Analyzer initialized")
    
    async def process_clinical_outcome(self, outcome: ClinicalOutcome) -> LearningUpdate:
        """
        Process a clinical outcome and update relationship confidence
        
        Args:
            outcome: Clinical outcome data
            
        Returns:
            Learning update information
        """
        try:
            # Find related assertions and relationships
            related_relationships = await self._find_related_relationships(outcome)
            
            if not related_relationships:
                logger.info(f"No related relationships found for outcome {outcome.outcome_id}")
                return None
            
            # Calculate confidence impact
            confidence_impact = self._calculate_confidence_impact(outcome)
            
            # Update relationship confidences
            learning_updates = []
            for relationship in related_relationships:
                update = await self._update_relationship_confidence(
                    relationship, outcome, confidence_impact
                )
                if update:
                    learning_updates.append(update)
            
            # Store outcome in graph for future learning
            await self._store_outcome_in_graph(outcome)
            
            # Update learning statistics
            self.learning_stats["total_outcomes_processed"] += 1
            self.learning_stats["confidence_updates"] += len(learning_updates)
            
            logger.info(f"Processed outcome {outcome.outcome_id}, updated {len(learning_updates)} relationships")
            
            # Return the most significant update
            if learning_updates:
                return max(learning_updates, key=lambda x: abs(x.confidence_delta))
            
            return None
            
        except Exception as e:
            logger.error(f"Error processing clinical outcome: {e}")
            return None
    
    async def analyze_override_patterns(self, lookback_days: int = 30) -> Dict[str, Any]:
        """
        Analyze clinician override patterns to identify learning opportunities
        
        Args:
            lookback_days: Number of days to analyze
            
        Returns:
            Override pattern analysis
        """
        try:
            # Query for override events
            override_query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
            
            SELECT ?assertion ?clinician ?override_reason ?outcome_type ?severity
                   ?assertion_confidence ?override_timestamp
            WHERE {
                ?clinician cae:OVERRODE ?assertion .
                ?assertion cae:confidenceScore ?assertion_confidence ;
                          cae:hasTimestamp ?override_timestamp .
                
                ?clinician cae:overrideReason ?override_reason .
                
                OPTIONAL {
                    ?assertion cae:resultedIn ?outcome .
                    ?outcome cae:outcomeType ?outcome_type ;
                            cae:outcomeSeverity ?severity .
                }
                
                FILTER(?override_timestamp >= "{cutoff_date}"^^xsd:dateTime)
            }
            ORDER BY DESC(?override_timestamp)
            """.format(cutoff_date=(datetime.utcnow() - timedelta(days=lookback_days)).isoformat())
            
            results = await self._execute_sparql_query(override_query)
            
            # Analyze override patterns
            override_analysis = {
                "total_overrides": len(results),
                "override_by_reason": {},
                "override_by_clinician": {},
                "high_confidence_overrides": [],
                "outcome_correlation": {},
                "learning_opportunities": []
            }
            
            for result in results:
                assertion_id = self._extract_entity_name(result["assertion"]["value"])
                clinician_id = self._extract_entity_name(result["clinician"]["value"])
                override_reason = result["override_reason"]["value"]
                confidence = float(result["assertion_confidence"]["value"])
                
                # Count by reason
                if override_reason not in override_analysis["override_by_reason"]:
                    override_analysis["override_by_reason"][override_reason] = 0
                override_analysis["override_by_reason"][override_reason] += 1
                
                # Count by clinician
                if clinician_id not in override_analysis["override_by_clinician"]:
                    override_analysis["override_by_clinician"][clinician_id] = 0
                override_analysis["override_by_clinician"][clinician_id] += 1
                
                # High confidence overrides (potential false positives)
                if confidence > 0.8:
                    override_analysis["high_confidence_overrides"].append({
                        "assertion_id": assertion_id,
                        "confidence": confidence,
                        "reason": override_reason,
                        "clinician": clinician_id
                    })
                
                # Outcome correlation
                if "outcome_type" in result:
                    outcome_type = result["outcome_type"]["value"]
                    if outcome_type not in override_analysis["outcome_correlation"]:
                        override_analysis["outcome_correlation"][outcome_type] = []
                    
                    severity = int(result["severity"]["value"])
                    override_analysis["outcome_correlation"][outcome_type].append({
                        "confidence": confidence,
                        "severity": severity,
                        "reason": override_reason
                    })
            
            # Identify learning opportunities
            override_analysis["learning_opportunities"] = self._identify_learning_opportunities(
                override_analysis
            )
            
            logger.info(f"Analyzed {len(results)} override events from last {lookback_days} days")
            return override_analysis
            
        except Exception as e:
            logger.error(f"Error analyzing override patterns: {e}")
            return {}
    
    async def evolve_relationship_confidence(self, drug1: str, drug2: str) -> Dict[str, Any]:
        """
        Evolve relationship confidence based on accumulated evidence
        
        Args:
            drug1: First drug in interaction
            drug2: Second drug in interaction
            
        Returns:
            Confidence evolution results
        """
        try:
            # Query for all outcomes related to this drug pair
            evidence_query = f"""
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?outcome_type ?severity ?confidence ?timestamp
            WHERE {{
                ?patient cae:hasMedication cae:{drug1} ;
                        cae:hasMedication cae:{drug2} ;
                        cae:EXPERIENCED ?outcome .
                
                ?outcome cae:outcomeType ?outcome_type ;
                        cae:outcomeSeverity ?severity ;
                        cae:confidence ?confidence ;
                        cae:hasTimestamp ?timestamp .
            }}
            ORDER BY ?timestamp
            """
            
            results = await self._execute_sparql_query(evidence_query)
            
            if len(results) < self.min_evidence_threshold:
                logger.info(f"Insufficient evidence for {drug1}-{drug2} interaction "
                           f"({len(results)} < {self.min_evidence_threshold})")
                return {"status": "insufficient_evidence", "evidence_count": len(results)}
            
            # Calculate evolved confidence
            evidence_weights = []
            for result in results:
                outcome_type = OutcomeType(result["outcome_type"]["value"])
                severity = int(result["severity"]["value"])
                
                # Weight evidence by outcome type and severity
                base_weight = self.outcome_weight_map.get(outcome_type, 0.0)
                severity_weight = severity / 10.0  # Normalize severity
                evidence_weight = base_weight * severity_weight
                evidence_weights.append(evidence_weight)
            
            # Calculate new confidence using weighted average
            if evidence_weights:
                # Sigmoid function to bound confidence between 0 and 1
                raw_confidence = sum(evidence_weights) / len(evidence_weights)
                new_confidence = 1 / (1 + math.exp(-raw_confidence))
            else:
                new_confidence = 0.5  # Default neutral confidence
            
            # Get current confidence
            current_confidence = await self._get_current_confidence(drug1, drug2)
            
            # Update confidence in graph
            await self._update_interaction_confidence(drug1, drug2, new_confidence)
            
            evolution_result = {
                "status": "confidence_evolved",
                "drug_pair": f"{drug1}-{drug2}",
                "old_confidence": current_confidence,
                "new_confidence": new_confidence,
                "confidence_delta": new_confidence - current_confidence,
                "evidence_count": len(results),
                "evidence_quality": self._assess_evidence_quality(evidence_weights)
            }
            
            self.learning_stats["relationship_updates"] += 1
            
            logger.info(f"Evolved confidence for {drug1}-{drug2}: "
                       f"{current_confidence:.3f} -> {new_confidence:.3f}")
            
            return evolution_result
            
        except Exception as e:
            logger.error(f"Error evolving relationship confidence: {e}")
            return {"status": "error", "message": str(e)}
    
    async def _find_related_relationships(self, outcome: ClinicalOutcome) -> List[Dict[str, Any]]:
        """Find relationships related to the clinical outcome"""
        # Query for relationships involving the patient's medications/conditions
        related_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?drug1 ?drug2 ?relationship_type ?confidence
        WHERE {{
            cae:{outcome.patient_id} cae:hasMedication ?drug1 ;
                                    cae:hasMedication ?drug2 .
            
            ?drug1 ?relationship_type ?drug2 .
            ?drug1 cae:confidenceScore ?confidence .
            
            FILTER(?drug1 != ?drug2)
            FILTER(?relationship_type = cae:INTERACTION_DYNAMIC)
        }}
        """
        
        results = await self._execute_sparql_query(related_query)
        
        relationships = []
        for result in results:
            relationships.append({
                "drug1": self._extract_entity_name(result["drug1"]["value"]),
                "drug2": self._extract_entity_name(result["drug2"]["value"]),
                "relationship_type": self._extract_entity_name(result["relationship_type"]["value"]),
                "current_confidence": float(result["confidence"]["value"])
            })
        
        return relationships
    
    def _calculate_confidence_impact(self, outcome: ClinicalOutcome) -> float:
        """Calculate how the outcome should impact confidence"""
        base_impact = self.outcome_weight_map.get(outcome.outcome_type, 0.0)
        severity_factor = outcome.outcome_severity / 10.0
        
        # Apply learning rate
        confidence_impact = base_impact * severity_factor * self.confidence_learning_rate
        
        return confidence_impact
    
    async def _update_relationship_confidence(self, relationship: Dict[str, Any], 
                                            outcome: ClinicalOutcome, 
                                            confidence_impact: float) -> LearningUpdate:
        """Update relationship confidence based on outcome"""
        old_confidence = relationship["current_confidence"]
        new_confidence = max(0.0, min(1.0, old_confidence + confidence_impact))
        
        # Update in GraphDB
        await self._update_interaction_confidence(
            relationship["drug1"], relationship["drug2"], new_confidence
        )
        
        return LearningUpdate(
            relationship_id=f"{relationship['drug1']}-{relationship['drug2']}",
            old_confidence=old_confidence,
            new_confidence=new_confidence,
            confidence_delta=new_confidence - old_confidence,
            evidence_strength=self._assess_evidence_strength(outcome),
            update_reason=f"Clinical outcome: {outcome.outcome_type.value}",
            supporting_outcomes=[outcome.outcome_id]
        )
    
    async def _store_outcome_in_graph(self, outcome: ClinicalOutcome):
        """Store clinical outcome in GraphDB for future learning"""
        store_outcome_sparql = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {{
            cae:{outcome.outcome_id} a cae:ClinicalOutcome ;
                                    cae:outcomeType "{outcome.outcome_type.value}" ;
                                    cae:outcomeSeverity "{outcome.outcome_severity}"^^xsd:integer ;
                                    cae:hasTimestamp "{outcome.recorded_at.isoformat()}"^^xsd:dateTime ;
                                    cae:confidence "{outcome.confidence_impact}"^^xsd:float .
            
            cae:{outcome.patient_id} cae:EXPERIENCED cae:{outcome.outcome_id} .
        }}
        """
        
        await self._execute_sparql_update(store_outcome_sparql)
    
    def _identify_learning_opportunities(self, override_analysis: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Identify learning opportunities from override patterns"""
        opportunities = []
        
        # High confidence overrides that might indicate false positives
        for override in override_analysis["high_confidence_overrides"]:
            if override["confidence"] > 0.9:
                opportunities.append({
                    "type": "potential_false_positive",
                    "description": f"High confidence assertion ({override['confidence']:.2f}) "
                                 f"overridden for: {override['reason']}",
                    "assertion_id": override["assertion_id"],
                    "recommendation": "Review assertion logic and evidence"
                })
        
        # Frequent override reasons
        for reason, count in override_analysis["override_by_reason"].items():
            if count > 5:  # Threshold for frequent overrides
                opportunities.append({
                    "type": "frequent_override_reason",
                    "description": f"Frequent overrides for reason: {reason} ({count} times)",
                    "recommendation": "Consider adjusting assertion sensitivity or adding context"
                })
        
        return opportunities
    
    def _assess_evidence_strength(self, outcome: ClinicalOutcome) -> str:
        """Assess the strength of evidence from an outcome"""
        if outcome.outcome_severity >= 8:
            return "strong"
        elif outcome.outcome_severity >= 5:
            return "moderate"
        else:
            return "weak"
    
    def _assess_evidence_quality(self, evidence_weights: List[float]) -> str:
        """Assess overall quality of evidence"""
        if not evidence_weights:
            return "none"
        
        avg_weight = sum(evidence_weights) / len(evidence_weights)
        if avg_weight > 0.7:
            return "high"
        elif avg_weight > 0.3:
            return "moderate"
        else:
            return "low"
    
    async def _get_current_confidence(self, drug1: str, drug2: str) -> float:
        """Get current confidence score for drug interaction"""
        confidence_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?confidence
        WHERE {{
            {{ cae:{drug1} cae:INTERACTION_DYNAMIC cae:{drug2} ;
                          cae:confidenceScore ?confidence . }}
            UNION
            {{ cae:{drug2} cae:INTERACTION_DYNAMIC cae:{drug1} ;
                          cae:confidenceScore ?confidence . }}
        }}
        """
        
        results = await self._execute_sparql_query(confidence_query)
        
        if results:
            return float(results[0]["confidence"]["value"])
        
        return 0.5  # Default confidence if not found
    
    async def _update_interaction_confidence(self, drug1: str, drug2: str, new_confidence: float):
        """Update interaction confidence in GraphDB"""
        update_sparql = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        DELETE {{
            cae:{drug1} cae:confidenceScore ?oldConfidence ;
                       cae:lastUpdated ?oldTimestamp .
        }}
        INSERT {{
            cae:{drug1} cae:confidenceScore "{new_confidence}"^^xsd:float ;
                       cae:lastUpdated "{datetime.utcnow().isoformat()}"^^xsd:dateTime .
        }}
        WHERE {{
            cae:{drug1} cae:INTERACTION_DYNAMIC cae:{drug2} ;
                       cae:confidenceScore ?oldConfidence ;
                       cae:lastUpdated ?oldTimestamp .
        }}
        """
        
        await self._execute_sparql_update(update_sparql)
    
    def _extract_entity_name(self, uri: str) -> str:
        """Extract entity name from URI"""
        return uri.split("/")[-1].split("#")[-1]
    
    async def _execute_sparql_query(self, sparql_query: str) -> List[Dict[str, Any]]:
        """Execute SPARQL SELECT query"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.base_url}",
                    headers={"Accept": "application/sparql-results+json"},
                    data={"query": sparql_query},
                    timeout=30.0
                )
                response.raise_for_status()
                
                result = response.json()
                return result.get("results", {}).get("bindings", [])
                
        except Exception as e:
            logger.error(f"SPARQL SELECT failed: {e}")
            return []
    
    async def _execute_sparql_update(self, sparql_query: str):
        """Execute SPARQL UPDATE query"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.base_url}/statements",
                    headers={"Content-Type": "application/sparql-update"},
                    data=sparql_query,
                    timeout=30.0
                )
                response.raise_for_status()
                
        except Exception as e:
            logger.error(f"SPARQL UPDATE failed: {e}")
            raise
    
    def get_learning_stats(self) -> Dict[str, Any]:
        """Get learning statistics"""
        return self.learning_stats.copy()
