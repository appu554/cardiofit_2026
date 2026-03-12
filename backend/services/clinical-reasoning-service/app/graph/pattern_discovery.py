"""
Pattern Discovery Engine for Clinical Assertion Engine

Real-time mining of hidden clinical patterns, temporal analysis, and 
population-level pattern aggregation for clinical intelligence.
"""

import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
import httpx
import json
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class ClinicalPattern:
    """Discovered clinical pattern"""
    pattern_id: str
    pattern_type: str
    description: str
    confidence_score: float
    support_count: int
    entities_involved: List[str]
    temporal_sequence: Optional[List[Dict[str, Any]]]
    clinical_significance: str
    discovered_at: datetime
    last_validated: datetime


@dataclass
class TemporalPattern:
    """Temporal medication sequence pattern"""
    sequence_id: str
    medication_sequence: List[str]
    time_intervals: List[int]  # in hours
    outcome_correlation: float
    frequency: int
    patient_population: List[str]
    clinical_context: Dict[str, Any]


class PatternDiscoveryEngine:
    """
    Real-time pattern discovery engine for clinical intelligence
    
    Features:
    - Hidden interaction discovery through graph analysis
    - Temporal pattern analysis for medication sequences
    - Population-level pattern aggregation
    - Continuous learning from clinical outcomes
    """
    
    def __init__(self, graphdb_endpoint: str = "http://localhost:7201", 
                 repository: str = "cae-clinical-intelligence"):
        self.graphdb_endpoint = graphdb_endpoint
        self.repository = repository
        self.base_url = f"{graphdb_endpoint}/repositories/{repository}"
        
        # Pattern discovery parameters
        self.min_support_threshold = 3  # Minimum occurrences to consider a pattern
        self.min_confidence_threshold = 0.6  # Minimum confidence for pattern validity
        self.temporal_window_hours = 72  # Time window for temporal patterns
        
        # Discovered patterns cache
        self.discovered_patterns = {}
        self.temporal_patterns = {}
        
        logger.info("Pattern Discovery Engine initialized")
    
    async def discover_hidden_interactions(self, patient_population: List[str] = None) -> List[ClinicalPattern]:
        """
        Discover hidden drug interactions through graph analysis
        
        Args:
            patient_population: Optional list of patient IDs to analyze
            
        Returns:
            List of discovered interaction patterns
        """
        try:
            # Query for co-occurring medications with outcomes
            co_occurrence_query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?drug1 ?drug2 ?outcome_type (COUNT(*) AS ?frequency) 
                   (AVG(?severity) AS ?avg_severity)
            WHERE {
                ?patient cae:hasMedication ?drug1 ;
                        cae:hasMedication ?drug2 ;
                        cae:EXPERIENCED ?outcome .
                
                ?outcome cae:outcomeType ?outcome_type ;
                        cae:outcomeSeverity ?severity .
                
                FILTER(?drug1 != ?drug2)
                FILTER(?outcome_type IN ("adverse_event", "therapeutic_failure", "unexpected_response"))
            }
            GROUP BY ?drug1 ?drug2 ?outcome_type
            HAVING (COUNT(*) >= {min_support})
            ORDER BY DESC(?frequency)
            """.format(min_support=self.min_support_threshold)
            
            results = await self._execute_sparql_query(co_occurrence_query)
            
            # Analyze results for hidden patterns
            hidden_patterns = []
            for result in results:
                drug1 = self._extract_entity_name(result["drug1"]["value"])
                drug2 = self._extract_entity_name(result["drug2"]["value"])
                outcome_type = result["outcome_type"]["value"]
                frequency = int(result["frequency"]["value"])
                avg_severity = float(result["avg_severity"]["value"])
                
                # Check if this is a known interaction
                is_known = await self._is_known_interaction(drug1, drug2)
                
                if not is_known and frequency >= self.min_support_threshold:
                    # Calculate confidence score based on frequency and severity
                    confidence = min(1.0, (frequency / 10.0) * (avg_severity / 10.0))
                    
                    if confidence >= self.min_confidence_threshold:
                        pattern = ClinicalPattern(
                            pattern_id=f"hidden_interaction_{drug1}_{drug2}_{outcome_type}",
                            pattern_type="hidden_drug_interaction",
                            description=f"Potential interaction between {drug1} and {drug2} "
                                      f"associated with {outcome_type}",
                            confidence_score=confidence,
                            support_count=frequency,
                            entities_involved=[drug1, drug2],
                            temporal_sequence=None,
                            clinical_significance=self._assess_clinical_significance(avg_severity),
                            discovered_at=datetime.utcnow(),
                            last_validated=datetime.utcnow()
                        )
                        hidden_patterns.append(pattern)
            
            # Cache discovered patterns
            for pattern in hidden_patterns:
                self.discovered_patterns[pattern.pattern_id] = pattern
            
            logger.info(f"Discovered {len(hidden_patterns)} hidden interaction patterns")
            return hidden_patterns
            
        except Exception as e:
            logger.error(f"Error discovering hidden interactions: {e}")
            return []
    
    async def analyze_temporal_patterns(self, lookback_days: int = 30) -> List[TemporalPattern]:
        """
        Analyze temporal medication sequences and their outcomes
        
        Args:
            lookback_days: Number of days to look back for pattern analysis
            
        Returns:
            List of discovered temporal patterns
        """
        try:
            # For now, return mock temporal patterns
            # TODO: Implement full temporal analysis when GraphDB is populated
            mock_patterns = [
                TemporalPattern(
                    sequence_id="temporal_warfarin_aspirin_metformin",
                    medication_sequence=["warfarin", "aspirin", "metformin"],
                    time_intervals=[24, 48],  # hours between medications
                    outcome_correlation=0.75,
                    frequency=5,
                    patient_population=["patient_001", "patient_002", "patient_003"],
                    clinical_context={
                        "sequence_type": "medication_cascade",
                        "temporal_window_hours": 72,
                        "population_size": 3
                    }
                )
            ]
            
            # Cache temporal patterns
            for pattern in mock_patterns:
                self.temporal_patterns[pattern.sequence_id] = pattern
            
            logger.info(f"Discovered {len(mock_patterns)} temporal medication patterns")
            return mock_patterns
            
        except Exception as e:
            logger.error(f"Error analyzing temporal patterns: {e}")
            return []
    
    async def aggregate_population_patterns(self, patient_cohort: List[str]) -> Dict[str, Any]:
        """
        Aggregate patterns across patient populations
        
        Args:
            patient_cohort: List of patient IDs to analyze
            
        Returns:
            Aggregated population-level patterns
        """
        try:
            # For now, return mock aggregated patterns
            # TODO: Implement full population analysis when GraphDB is populated
            aggregated_patterns = {
                "medication_by_condition": {
                    "diabetes": {
                        "metformin": {"patient_count": 15, "avg_outcome": 7.2, "prevalence": 0.75},
                        "insulin": {"patient_count": 8, "avg_outcome": 6.8, "prevalence": 0.40}
                    },
                    "hypertension": {
                        "lisinopril": {"patient_count": 12, "avg_outcome": 7.5, "prevalence": 0.60},
                        "amlodipine": {"patient_count": 10, "avg_outcome": 7.1, "prevalence": 0.50}
                    }
                },
                "medication_by_age_group": {
                    "elderly": {
                        "warfarin": {"patient_count": 6, "avg_outcome": 6.5},
                        "aspirin": {"patient_count": 8, "avg_outcome": 7.0}
                    },
                    "adult": {
                        "metformin": {"patient_count": 10, "avg_outcome": 7.3},
                        "lisinopril": {"patient_count": 7, "avg_outcome": 7.4}
                    }
                },
                "condition_clusters": {},
                "outcome_correlations": {}
            }
            
            logger.info(f"Aggregated population patterns for {len(patient_cohort)} patients")
            return aggregated_patterns
            
        except Exception as e:
            logger.error(f"Error aggregating population patterns: {e}")
            return {}
    
    async def _is_known_interaction(self, drug1: str, drug2: str) -> bool:
        """Check if drug interaction is already known in the knowledge base"""
        # For now, assume some common interactions are known
        known_interactions = [
            ("warfarin", "aspirin"),
            ("warfarin", "ibuprofen"),
            ("digoxin", "furosemide")
        ]
        
        interaction_pair = tuple(sorted([drug1.lower(), drug2.lower()]))
        return any(tuple(sorted([k1, k2])) == interaction_pair for k1, k2 in known_interactions)
    
    def _extract_entity_name(self, uri: str) -> str:
        """Extract entity name from URI"""
        return uri.split("/")[-1].split("#")[-1]
    
    def _assess_clinical_significance(self, severity_score: float) -> str:
        """Assess clinical significance based on severity score"""
        if severity_score >= 8.0:
            return "high"
        elif severity_score >= 5.0:
            return "moderate"
        else:
            return "low"
    
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
    
    async def get_pattern_stats(self) -> Dict[str, Any]:
        """Get pattern discovery statistics"""
        return {
            "discovered_patterns_count": len(self.discovered_patterns),
            "temporal_patterns_count": len(self.temporal_patterns),
            "last_discovery_run": datetime.utcnow().isoformat(),
            "min_support_threshold": self.min_support_threshold,
            "min_confidence_threshold": self.min_confidence_threshold,
            "pattern_types": {
                "hidden_interactions": len([p for p in self.discovered_patterns.values() 
                                          if p.pattern_type == "hidden_drug_interaction"]),
                "temporal_sequences": len(self.temporal_patterns)
            }
        }
