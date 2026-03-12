"""
Relationship Navigator for Clinical Assertion Engine

Context-aware graph traversal for similar patients, multi-hop reasoning
through relationship chains, and similarity-based clinical recommendations.
"""

import logging
from datetime import datetime
from typing import Dict, List, Optional, Any, Tuple
import httpx
import json
import math
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class PatientSimilarity:
    """Patient similarity result"""
    patient_id: str
    similarity_score: float
    shared_attributes: List[str]
    clinical_context: Dict[str, Any]
    reasoning_path: List[str]


@dataclass
class RelationshipPath:
    """Multi-hop relationship path"""
    path_id: str
    start_entity: str
    end_entity: str
    relationship_chain: List[str]
    path_strength: float
    clinical_relevance: str
    evidence_nodes: List[str]


class RelationshipNavigator:
    """
    Context-aware relationship navigator for clinical graph intelligence
    
    Features:
    - Patient similarity discovery using graph algorithms
    - Multi-hop reasoning through clinical relationships
    - Context-aware graph traversal
    - Similarity-based clinical recommendations
    """
    
    def __init__(self, graphdb_endpoint: str = "http://localhost:7201", 
                 repository: str = "cae-clinical-intelligence"):
        self.graphdb_endpoint = graphdb_endpoint
        self.repository = repository
        self.base_url = f"{graphdb_endpoint}/repositories/{repository}"
        
        # Navigation parameters
        self.similarity_threshold = 0.7
        self.max_hop_distance = 3
        self.max_similar_patients = 10
        
        # Similarity weights for different attributes
        self.attribute_weights = {
            "age_group": 0.15,
            "gender": 0.10,
            "conditions": 0.30,
            "medications": 0.25,
            "allergies": 0.20
        }
        
        logger.info("Relationship Navigator initialized")
    
    async def find_similar_patients(self, patient_id: str, 
                                  clinical_context: Dict[str, Any] = None) -> List[PatientSimilarity]:
        """
        Find similar patients using graph-based similarity algorithms
        
        Args:
            patient_id: Target patient ID
            clinical_context: Additional clinical context for similarity
            
        Returns:
            List of similar patients with similarity scores
        """
        try:
            # Get patient's clinical profile
            patient_profile = await self._get_patient_profile(patient_id)
            
            if not patient_profile:
                logger.warning(f"No profile found for patient {patient_id}")
                return []
            
            # Query for potential similar patients
            similarity_query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT DISTINCT ?similar_patient ?age_group ?gender
                   (GROUP_CONCAT(DISTINCT ?condition; separator=",") AS ?conditions)
                   (GROUP_CONCAT(DISTINCT ?medication; separator=",") AS ?medications)
                   (GROUP_CONCAT(DISTINCT ?allergy; separator=",") AS ?allergies)
            WHERE {
                ?similar_patient a cae:Patient ;
                                cae:ageGroup ?age_group ;
                                cae:gender ?gender .
                
                OPTIONAL { ?similar_patient cae:hasCondition ?condition . }
                OPTIONAL { ?similar_patient cae:hasMedication ?medication . }
                OPTIONAL { ?similar_patient cae:hasAllergy ?allergy . }
                
                FILTER(?similar_patient != cae:{patient_id})
            }
            GROUP BY ?similar_patient ?age_group ?gender
            """.format(patient_id=patient_id)
            
            results = await self._execute_sparql_query(similarity_query)
            
            # Calculate similarity scores
            similar_patients = []
            for result in results:
                similar_patient_id = self._extract_entity_name(result["similar_patient"]["value"])
                
                # Build candidate profile
                candidate_profile = {
                    "age_group": result.get("age_group", {}).get("value", ""),
                    "gender": result.get("gender", {}).get("value", ""),
                    "conditions": result.get("conditions", {}).get("value", "").split(",") if result.get("conditions", {}).get("value") else [],
                    "medications": result.get("medications", {}).get("value", "").split(",") if result.get("medications", {}).get("value") else [],
                    "allergies": result.get("allergies", {}).get("value", "").split(",") if result.get("allergies", {}).get("value") else []
                }
                
                # Calculate similarity score
                similarity_score = self._calculate_similarity_score(patient_profile, candidate_profile)
                
                if similarity_score >= self.similarity_threshold:
                    # Find shared attributes
                    shared_attributes = self._find_shared_attributes(patient_profile, candidate_profile)
                    
                    # Build reasoning path
                    reasoning_path = await self._build_reasoning_path(patient_id, similar_patient_id)
                    
                    similar_patient = PatientSimilarity(
                        patient_id=similar_patient_id,
                        similarity_score=similarity_score,
                        shared_attributes=shared_attributes,
                        clinical_context=candidate_profile,
                        reasoning_path=reasoning_path
                    )
                    similar_patients.append(similar_patient)
            
            # Sort by similarity score and limit results
            similar_patients.sort(key=lambda x: x.similarity_score, reverse=True)
            similar_patients = similar_patients[:self.max_similar_patients]
            
            logger.info(f"Found {len(similar_patients)} similar patients for {patient_id}")
            return similar_patients
            
        except Exception as e:
            logger.error(f"Error finding similar patients: {e}")
            return []
    
    async def navigate_relationship_paths(self, start_entity: str, end_entity: str,
                                        relationship_types: List[str] = None) -> List[RelationshipPath]:
        """
        Navigate multi-hop relationship paths between entities
        
        Args:
            start_entity: Starting entity
            end_entity: Target entity
            relationship_types: Optional filter for relationship types
            
        Returns:
            List of relationship paths
        """
        try:
            # Build path query with configurable hop distance
            path_query = """
            PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
            
            SELECT ?path ?relationship1 ?intermediate ?relationship2 ?strength
            WHERE {
                {
                    # Direct relationship (1-hop)
                    cae:{start_entity} ?relationship1 cae:{end_entity} .
                    BIND("direct" AS ?path)
                    BIND(1.0 AS ?strength)
                }
                UNION
                {
                    # 2-hop relationship
                    cae:{start_entity} ?relationship1 ?intermediate .
                    ?intermediate ?relationship2 cae:{end_entity} .
                    BIND("2-hop" AS ?path)
                    BIND(0.8 AS ?strength)
                }
                UNION
                {
                    # 3-hop relationship
                    cae:{start_entity} ?relationship1 ?intermediate1 .
                    ?intermediate1 ?relationship2 ?intermediate2 .
                    ?intermediate2 ?relationship3 cae:{end_entity} .
                    BIND("3-hop" AS ?path)
                    BIND(0.6 AS ?strength)
                }
            }
            ORDER BY DESC(?strength)
            """.format(start_entity=start_entity, end_entity=end_entity)
            
            results = await self._execute_sparql_query(path_query)
            
            # Process relationship paths
            relationship_paths = []
            for i, result in enumerate(results):
                path_type = result["path"]["value"]
                strength = float(result["strength"]["value"])
                
                # Build relationship chain
                relationship_chain = []
                if "relationship1" in result:
                    rel1 = self._extract_entity_name(result["relationship1"]["value"])
                    relationship_chain.append(rel1)
                
                if "relationship2" in result:
                    rel2 = self._extract_entity_name(result["relationship2"]["value"])
                    relationship_chain.append(rel2)
                
                # Assess clinical relevance
                clinical_relevance = self._assess_path_relevance(relationship_chain, strength)
                
                # Find evidence nodes
                evidence_nodes = []
                if "intermediate" in result:
                    evidence_nodes.append(self._extract_entity_name(result["intermediate"]["value"]))
                
                path = RelationshipPath(
                    path_id=f"path_{start_entity}_{end_entity}_{i}",
                    start_entity=start_entity,
                    end_entity=end_entity,
                    relationship_chain=relationship_chain,
                    path_strength=strength,
                    clinical_relevance=clinical_relevance,
                    evidence_nodes=evidence_nodes
                )
                relationship_paths.append(path)
            
            logger.info(f"Found {len(relationship_paths)} relationship paths from {start_entity} to {end_entity}")
            return relationship_paths
            
        except Exception as e:
            logger.error(f"Error navigating relationship paths: {e}")
            return []
    
    async def get_context_recommendations(self, patient_id: str, 
                                        clinical_scenario: str) -> List[Dict[str, Any]]:
        """
        Get context-aware clinical recommendations based on similar patients
        
        Args:
            patient_id: Target patient ID
            clinical_scenario: Clinical scenario (e.g., "medication_selection", "dosing_adjustment")
            
        Returns:
            List of clinical recommendations
        """
        try:
            # Find similar patients
            similar_patients = await self.find_similar_patients(patient_id)
            
            if not similar_patients:
                return []
            
            # Query for outcomes in similar patients
            recommendations = []
            for similar_patient in similar_patients[:5]:  # Top 5 similar patients
                outcome_query = f"""
                PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
                
                SELECT ?intervention ?outcome_type ?outcome_score ?confidence
                WHERE {{
                    cae:{similar_patient.patient_id} cae:receivedIntervention ?intervention .
                    ?intervention cae:resultedIn ?outcome .
                    ?outcome cae:outcomeType ?outcome_type ;
                            cae:outcomeScore ?outcome_score ;
                            cae:confidence ?confidence .
                    
                    FILTER(?outcome_type = "{clinical_scenario}")
                }}
                ORDER BY DESC(?outcome_score)
                """
                
                results = await self._execute_sparql_query(outcome_query)
                
                for result in results:
                    intervention = self._extract_entity_name(result["intervention"]["value"])
                    outcome_score = float(result["outcome_score"]["value"])
                    confidence = float(result["confidence"]["value"])
                    
                    # Weight recommendation by patient similarity
                    weighted_score = outcome_score * similar_patient.similarity_score
                    
                    recommendation = {
                        "intervention": intervention,
                        "recommendation_score": weighted_score,
                        "confidence": confidence,
                        "evidence_patient": similar_patient.patient_id,
                        "similarity_score": similar_patient.similarity_score,
                        "reasoning": f"Based on similar patient {similar_patient.patient_id} "
                                   f"with {similar_patient.similarity_score:.2f} similarity"
                    }
                    recommendations.append(recommendation)
            
            # Aggregate and rank recommendations
            aggregated_recommendations = self._aggregate_recommendations(recommendations)
            
            logger.info(f"Generated {len(aggregated_recommendations)} context recommendations for {patient_id}")
            return aggregated_recommendations
            
        except Exception as e:
            logger.error(f"Error getting context recommendations: {e}")
            return []
    
    async def _get_patient_profile(self, patient_id: str) -> Optional[Dict[str, Any]]:
        """Get comprehensive patient profile"""
        profile_query = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT ?age_group ?gender
               (GROUP_CONCAT(DISTINCT ?condition; separator=",") AS ?conditions)
               (GROUP_CONCAT(DISTINCT ?medication; separator=",") AS ?medications)
               (GROUP_CONCAT(DISTINCT ?allergy; separator=",") AS ?allergies)
        WHERE {{
            cae:{patient_id} a cae:Patient ;
                            cae:ageGroup ?age_group ;
                            cae:gender ?gender .
            
            OPTIONAL {{ cae:{patient_id} cae:hasCondition ?condition . }}
            OPTIONAL {{ cae:{patient_id} cae:hasMedication ?medication . }}
            OPTIONAL {{ cae:{patient_id} cae:hasAllergy ?allergy . }}
        }}
        GROUP BY ?age_group ?gender
        """
        
        results = await self._execute_sparql_query(profile_query)
        
        if results:
            result = results[0]
            return {
                "age_group": result.get("age_group", {}).get("value", ""),
                "gender": result.get("gender", {}).get("value", ""),
                "conditions": result.get("conditions", {}).get("value", "").split(",") if result.get("conditions", {}).get("value") else [],
                "medications": result.get("medications", {}).get("value", "").split(",") if result.get("medications", {}).get("value") else [],
                "allergies": result.get("allergies", {}).get("value", "").split(",") if result.get("allergies", {}).get("value") else []
            }
        
        return None
    
    def _calculate_similarity_score(self, profile1: Dict[str, Any], profile2: Dict[str, Any]) -> float:
        """Calculate similarity score between two patient profiles"""
        total_score = 0.0
        
        # Age group similarity
        if profile1.get("age_group") == profile2.get("age_group"):
            total_score += self.attribute_weights["age_group"]
        
        # Gender similarity
        if profile1.get("gender") == profile2.get("gender"):
            total_score += self.attribute_weights["gender"]
        
        # Condition similarity (Jaccard index)
        conditions1 = set(profile1.get("conditions", []))
        conditions2 = set(profile2.get("conditions", []))
        if conditions1 or conditions2:
            condition_similarity = len(conditions1 & conditions2) / len(conditions1 | conditions2)
            total_score += self.attribute_weights["conditions"] * condition_similarity
        
        # Medication similarity (Jaccard index)
        medications1 = set(profile1.get("medications", []))
        medications2 = set(profile2.get("medications", []))
        if medications1 or medications2:
            medication_similarity = len(medications1 & medications2) / len(medications1 | medications2)
            total_score += self.attribute_weights["medications"] * medication_similarity
        
        # Allergy similarity (Jaccard index)
        allergies1 = set(profile1.get("allergies", []))
        allergies2 = set(profile2.get("allergies", []))
        if allergies1 or allergies2:
            allergy_similarity = len(allergies1 & allergies2) / len(allergies1 | allergies2)
            total_score += self.attribute_weights["allergies"] * allergy_similarity
        
        return min(1.0, total_score)
    
    def _find_shared_attributes(self, profile1: Dict[str, Any], profile2: Dict[str, Any]) -> List[str]:
        """Find shared attributes between patient profiles"""
        shared = []
        
        if profile1.get("age_group") == profile2.get("age_group"):
            shared.append(f"age_group: {profile1.get('age_group')}")
        
        if profile1.get("gender") == profile2.get("gender"):
            shared.append(f"gender: {profile1.get('gender')}")
        
        shared_conditions = set(profile1.get("conditions", [])) & set(profile2.get("conditions", []))
        for condition in shared_conditions:
            shared.append(f"condition: {condition}")
        
        shared_medications = set(profile1.get("medications", [])) & set(profile2.get("medications", []))
        for medication in shared_medications:
            shared.append(f"medication: {medication}")
        
        shared_allergies = set(profile1.get("allergies", [])) & set(profile2.get("allergies", []))
        for allergy in shared_allergies:
            shared.append(f"allergy: {allergy}")
        
        return shared
    
    async def _build_reasoning_path(self, patient1: str, patient2: str) -> List[str]:
        """Build reasoning path between similar patients"""
        # For now, return a simple path
        # TODO: Implement full graph path analysis
        return [f"patient_similarity", f"shared_clinical_attributes"]
    
    def _assess_path_relevance(self, relationship_chain: List[str], strength: float) -> str:
        """Assess clinical relevance of relationship path"""
        if strength >= 0.8:
            return "high"
        elif strength >= 0.6:
            return "moderate"
        else:
            return "low"
    
    def _aggregate_recommendations(self, recommendations: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Aggregate and rank recommendations"""
        # Group by intervention
        intervention_groups = {}
        for rec in recommendations:
            intervention = rec["intervention"]
            if intervention not in intervention_groups:
                intervention_groups[intervention] = []
            intervention_groups[intervention].append(rec)
        
        # Aggregate scores
        aggregated = []
        for intervention, group in intervention_groups.items():
            avg_score = sum(r["recommendation_score"] for r in group) / len(group)
            avg_confidence = sum(r["confidence"] for r in group) / len(group)
            
            aggregated.append({
                "intervention": intervention,
                "aggregated_score": avg_score,
                "confidence": avg_confidence,
                "evidence_count": len(group),
                "supporting_patients": [r["evidence_patient"] for r in group]
            })
        
        # Sort by aggregated score
        aggregated.sort(key=lambda x: x["aggregated_score"], reverse=True)
        return aggregated
    
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
