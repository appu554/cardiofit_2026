"""
Graph Schema Manager for Clinical Assertion Engine

Manages dynamic relationship schema with learning capabilities and temporal pattern storage.
Implements the enhanced GraphDB schema from the comprehensive implementation plan.
"""

import logging
from datetime import datetime
from typing import Dict, List, Optional, Any
import httpx
import json

from app.graph.graphdb_client import graphdb_client, GraphDBResult

logger = logging.getLogger(__name__)


class GraphSchemaManager:
    """
    Enhanced GraphDB schema manager with dynamic learning capabilities
    
    Features:
    - Dynamic relationship schema with confidence scores
    - Learning relationship types (OVERRODE, EXPERIENCED, SIMILAR_TO, LEARNED_FROM)
    - Temporal pattern storage for medication sequences
    - Context vector storage for patient similarity
    """
    
    def __init__(self):
        # Use the global GraphDB client
        self.graphdb_client = graphdb_client

        # Schema templates for dynamic relationships
        self.relationship_templates = self._initialize_relationship_templates()

        logger.info("Graph Schema Manager initialized with real GraphDB client")
    
    async def initialize_schema(self):
        """Initialize the enhanced GraphDB schema with dynamic relationships"""
        try:
            # Create core entity types
            await self._create_core_entities()
            
            # Create dynamic relationship types
            await self._create_dynamic_relationships()
            
            # Create learning relationship types
            await self._create_learning_relationships()
            
            # Create temporal pattern structures
            await self._create_temporal_patterns()
            
            # Create context vector storage
            await self._create_context_vectors()
            
            logger.info("Enhanced GraphDB schema initialized successfully")
            
        except Exception as e:
            logger.error(f"Failed to initialize GraphDB schema: {e}")
            raise
    
    async def _create_core_entities(self):
        """Create core clinical entities"""
        core_entities_sparql = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {
            # Core Entity Types
            cae:Patient rdfs:subClassOf cae:ClinicalEntity .
            cae:Drug rdfs:subClassOf cae:ClinicalEntity .
            cae:Condition rdfs:subClassOf cae:ClinicalEntity .
            cae:Clinician rdfs:subClassOf cae:ClinicalEntity .
            cae:ClinicalAssertion rdfs:subClassOf cae:ClinicalEntity .
            cae:ClinicalOutcome rdfs:subClassOf cae:ClinicalEntity .
            cae:ClinicalContext rdfs:subClassOf cae:ClinicalEntity .
            
            # Core Properties
            cae:hasConfidence rdfs:domain cae:ClinicalEntity ;
                             rdfs:range xsd:float .
            cae:hasTimestamp rdfs:domain cae:ClinicalEntity ;
                            rdfs:range xsd:dateTime .
            cae:hasPatientCount rdfs:domain cae:ClinicalEntity ;
                               rdfs:range xsd:integer .
        }
        """
        
        await self._execute_sparql_update(core_entities_sparql)
    
    async def _create_dynamic_relationships(self):
        """Create dynamic relationship types with learning capabilities"""
        dynamic_relationships_sparql = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {
            # Dynamic Drug Interaction Relationships
            cae:INTERACTION_DYNAMIC rdfs:subPropertyOf cae:clinicalRelationship ;
                                   rdfs:domain cae:Drug ;
                                   rdfs:range cae:Drug .
            
            # Relationship Properties for Learning
            cae:confidenceScore rdfs:domain cae:INTERACTION_DYNAMIC ;
                               rdfs:range xsd:float .
            cae:severityScore rdfs:domain cae:INTERACTION_DYNAMIC ;
                             rdfs:range xsd:float .
            cae:mechanism rdfs:domain cae:INTERACTION_DYNAMIC ;
                         rdfs:range xsd:string .
            cae:evidenceStrength rdfs:domain cae:INTERACTION_DYNAMIC ;
                                rdfs:range xsd:string .
            cae:outcomeCorrelation rdfs:domain cae:INTERACTION_DYNAMIC ;
                                  rdfs:range xsd:float .
            cae:overrideRate rdfs:domain cae:INTERACTION_DYNAMIC ;
                            rdfs:range xsd:float .
            cae:contextFactors rdfs:domain cae:INTERACTION_DYNAMIC ;
                              rdfs:range xsd:string .
            cae:learningSource rdfs:domain cae:INTERACTION_DYNAMIC ;
                             rdfs:range xsd:string .
            cae:patientCount rdfs:domain cae:INTERACTION_DYNAMIC ;
                            rdfs:range xsd:integer .
            cae:lastUpdated rdfs:domain cae:INTERACTION_DYNAMIC ;
                           rdfs:range xsd:dateTime .
        }
        """
        
        await self._execute_sparql_update(dynamic_relationships_sparql)
    
    async def _create_learning_relationships(self):
        """Create learning relationship types for clinical intelligence"""
        learning_relationships_sparql = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {
            # Learning Relationship Types
            cae:OVERRODE rdfs:subPropertyOf cae:learningRelationship ;
                        rdfs:domain cae:Clinician ;
                        rdfs:range cae:ClinicalAssertion .
            
            cae:EXPERIENCED rdfs:subPropertyOf cae:learningRelationship ;
                           rdfs:domain cae:Patient ;
                           rdfs:range cae:ClinicalOutcome .
            
            cae:SIMILAR_TO rdfs:subPropertyOf cae:learningRelationship ;
                          rdfs:domain cae:Patient ;
                          rdfs:range cae:Patient .
            
            cae:LEARNED_FROM rdfs:subPropertyOf cae:learningRelationship ;
                            rdfs:domain cae:ClinicalAssertion ;
                            rdfs:range cae:ClinicalOutcome .
            
            # Learning Properties
            cae:overrideReason rdfs:domain cae:OVERRODE ;
                              rdfs:range xsd:string .
            cae:clinicalJustification rdfs:domain cae:OVERRODE ;
                                     rdfs:range xsd:string .
            cae:outcomeType rdfs:domain cae:EXPERIENCED ;
                           rdfs:range xsd:string .
            cae:outcomeSeverity rdfs:domain cae:EXPERIENCED ;
                               rdfs:range xsd:integer .
            cae:similarityScore rdfs:domain cae:SIMILAR_TO ;
                               rdfs:range xsd:float .
            cae:learningWeight rdfs:domain cae:LEARNED_FROM ;
                              rdfs:range xsd:float .
        }
        """
        
        await self._execute_sparql_update(learning_relationships_sparql)
    
    async def _create_temporal_patterns(self):
        """Create temporal pattern storage structures"""
        temporal_patterns_sparql = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {
            # Temporal Pattern Entities
            cae:TemporalPattern rdfs:subClassOf cae:ClinicalEntity .
            cae:MedicationSequence rdfs:subClassOf cae:TemporalPattern .
            cae:ClinicalTimeline rdfs:subClassOf cae:TemporalPattern .
            
            # Temporal Relationships
            cae:PRECEDED_BY rdfs:subPropertyOf cae:temporalRelationship ;
                           rdfs:domain cae:ClinicalEntity ;
                           rdfs:range cae:ClinicalEntity .
            
            cae:FOLLOWED_BY rdfs:subPropertyOf cae:temporalRelationship ;
                           rdfs:domain cae:ClinicalEntity ;
                           rdfs:range cae:ClinicalEntity .
            
            cae:CONCURRENT_WITH rdfs:subPropertyOf cae:temporalRelationship ;
                               rdfs:domain cae:ClinicalEntity ;
                               rdfs:range cae:ClinicalEntity .
            
            # Temporal Properties
            cae:timeInterval rdfs:domain cae:temporalRelationship ;
                            rdfs:range xsd:duration .
            cae:clinicalSignificance rdfs:domain cae:temporalRelationship ;
                                    rdfs:range xsd:string .
            cae:patternFrequency rdfs:domain cae:TemporalPattern ;
                                rdfs:range xsd:integer .
            cae:patternConfidence rdfs:domain cae:TemporalPattern ;
                                 rdfs:range xsd:float .
        }
        """
        
        await self._execute_sparql_update(temporal_patterns_sparql)
    
    async def _create_context_vectors(self):
        """Create context vector storage for patient similarity"""
        context_vectors_sparql = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {
            # Context Vector Entities
            cae:ContextVector rdfs:subClassOf cae:ClinicalEntity .
            cae:PatientVector rdfs:subClassOf cae:ContextVector .
            cae:ClinicalVector rdfs:subClassOf cae:ContextVector .
            
            # Vector Properties
            cae:hasVector rdfs:domain cae:ContextVector ;
                         rdfs:range xsd:string .  # JSON-encoded vector
            cae:vectorDimensions rdfs:domain cae:ContextVector ;
                                rdfs:range xsd:integer .
            cae:vectorType rdfs:domain cae:ContextVector ;
                          rdfs:range xsd:string .
            cae:computedAt rdfs:domain cae:ContextVector ;
                          rdfs:range xsd:dateTime .
            
            # Similarity Relationships
            cae:VECTOR_SIMILAR rdfs:subPropertyOf cae:similarityRelationship ;
                              rdfs:domain cae:ContextVector ;
                              rdfs:range cae:ContextVector .
            
            cae:cosineSimilarity rdfs:domain cae:VECTOR_SIMILAR ;
                               rdfs:range xsd:float .
            cae:euclideanDistance rdfs:domain cae:VECTOR_SIMILAR ;
                                 rdfs:range xsd:float .
        }
        """
        
        await self._execute_sparql_update(context_vectors_sparql)
    
    async def create_dynamic_interaction(self, drug1: str, drug2: str, 
                                       interaction_data: Dict[str, Any]) -> str:
        """Create a dynamic drug interaction relationship with learning capabilities"""
        
        interaction_id = f"interaction_{drug1}_{drug2}_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"
        
        create_interaction_sparql = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        INSERT DATA {{
            cae:{drug1} cae:INTERACTION_DYNAMIC cae:{drug2} .
            
            cae:{drug1} cae:confidenceScore "{interaction_data.get('confidence', 0.5)}"^^xsd:float ;
                       cae:severityScore "{interaction_data.get('severity_score', 5.0)}"^^xsd:float ;
                       cae:mechanism "{interaction_data.get('mechanism', 'unknown')}" ;
                       cae:evidenceStrength "{interaction_data.get('evidence_strength', 'low')}" ;
                       cae:outcomeCorrelation "{interaction_data.get('outcome_correlation', 0.0)}"^^xsd:float ;
                       cae:overrideRate "{interaction_data.get('override_rate', 0.0)}"^^xsd:float ;
                       cae:learningSource "{interaction_data.get('learning_source', 'clinical_evidence')}" ;
                       cae:patientCount "{interaction_data.get('patient_count', 0)}"^^xsd:integer ;
                       cae:lastUpdated "{datetime.utcnow().isoformat()}"^^xsd:dateTime .
        }}
        """
        
        await self._execute_sparql_update(create_interaction_sparql)
        return interaction_id
    
    async def update_interaction_confidence(self, drug1: str, drug2: str, 
                                          new_confidence: float, outcome_data: Dict[str, Any]):
        """Update interaction confidence based on clinical outcomes"""
        
        update_sparql = f"""
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
        
        DELETE {{
            cae:{drug1} cae:confidenceScore ?oldConfidence ;
                       cae:outcomeCorrelation ?oldCorrelation ;
                       cae:overrideRate ?oldOverrideRate ;
                       cae:patientCount ?oldCount ;
                       cae:lastUpdated ?oldTimestamp .
        }}
        INSERT {{
            cae:{drug1} cae:confidenceScore "{new_confidence}"^^xsd:float ;
                       cae:outcomeCorrelation "{outcome_data.get('correlation', 0.0)}"^^xsd:float ;
                       cae:overrideRate "{outcome_data.get('override_rate', 0.0)}"^^xsd:float ;
                       cae:patientCount "{outcome_data.get('patient_count', 0)}"^^xsd:integer ;
                       cae:lastUpdated "{datetime.utcnow().isoformat()}"^^xsd:dateTime .
        }}
        WHERE {{
            cae:{drug1} cae:INTERACTION_DYNAMIC cae:{drug2} ;
                       cae:confidenceScore ?oldConfidence ;
                       cae:outcomeCorrelation ?oldCorrelation ;
                       cae:overrideRate ?oldOverrideRate ;
                       cae:patientCount ?oldCount ;
                       cae:lastUpdated ?oldTimestamp .
        }}
        """
        
        await self._execute_sparql_update(update_sparql)
    
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
            raise
    
    def _initialize_relationship_templates(self) -> Dict[str, str]:
        """Initialize relationship templates for dynamic creation"""
        return {
            "drug_interaction": """
                cae:{drug1} cae:INTERACTION_DYNAMIC cae:{drug2} ;
                           cae:confidenceScore "{confidence}"^^xsd:float ;
                           cae:severityScore "{severity_score}"^^xsd:float ;
                           cae:mechanism "{mechanism}" ;
                           cae:evidenceStrength "{evidence_strength}" ;
                           cae:learningSource "{learning_source}" ;
                           cae:lastUpdated "{timestamp}"^^xsd:dateTime .
            """,
            "clinical_override": """
                cae:{clinician} cae:OVERRODE cae:{assertion} ;
                               cae:overrideReason "{reason}" ;
                               cae:clinicalJustification "{justification}" ;
                               cae:hasTimestamp "{timestamp}"^^xsd:dateTime .
            """,
            "patient_similarity": """
                cae:{patient1} cae:SIMILAR_TO cae:{patient2} ;
                              cae:similarityScore "{similarity_score}"^^xsd:float ;
                              cae:computedAt "{timestamp}"^^xsd:dateTime .
            """
        }
    
    async def get_schema_stats(self) -> Dict[str, Any]:
        """Get statistics about the current schema"""
        stats_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        
        SELECT 
            (COUNT(DISTINCT ?patient) AS ?total_patients)
            (COUNT(DISTINCT ?drug) AS ?total_drugs)
            (COUNT(DISTINCT ?interaction) AS ?total_interactions)
            (COUNT(DISTINCT ?override) AS ?total_overrides)
        WHERE {
            OPTIONAL { ?patient a cae:Patient . }
            OPTIONAL { ?drug a cae:Drug . }
            OPTIONAL { ?drug1 cae:INTERACTION_DYNAMIC ?drug2 . BIND(?drug1 AS ?interaction) }
            OPTIONAL { ?clinician cae:OVERRODE ?assertion . BIND(?clinician AS ?override) }
        }
        """
        
        results = await self._execute_sparql_query(stats_query)
        
        if results:
            result = results[0]
            return {
                "total_patients": int(result.get("total_patients", {}).get("value", 0)),
                "total_drugs": int(result.get("total_drugs", {}).get("value", 0)),
                "total_interactions": int(result.get("total_interactions", {}).get("value", 0)),
                "total_overrides": int(result.get("total_overrides", {}).get("value", 0))
            }
        
        return {"total_patients": 0, "total_drugs": 0, "total_interactions": 0, "total_overrides": 0}
