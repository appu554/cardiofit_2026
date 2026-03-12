"""
Shared Neo4j Multi-KB Stream Manager
Manages data streams for ALL CardioFit Knowledge Bases using logical partitioning:

Knowledge Base Streams:
- KB1_PatientStream: Patient data
- KB2_GuidelineStream: Clinical guidelines
- KB3_DrugCalculationStream: Drug calculations
- KB4_SafetyStream: Safety rules
- KB5_InteractionStream: Drug interactions
- KB6_EvidenceStream: Evidence base
- KB7_TerminologyStream: Medical terminology
- KB8_WorkflowStream: Clinical workflows
- SharedSemanticMesh: Cross-KB relationships

Compatible with both Neo4j Community and Enterprise editions.
"""

import asyncio
from neo4j import AsyncGraphDatabase
from typing import Dict, List, Optional, Any, Union
import json
import hashlib
from datetime import datetime
from loguru import logger
from enum import Enum


class KnowledgeBase(Enum):
    """Supported Knowledge Bases"""
    KB1_PATIENT = "kb1"
    KB2_GUIDELINES = "kb2"
    KB3_DRUG_CALCULATIONS = "kb3"
    KB4_SAFETY_RULES = "kb4"
    KB5_DRUG_INTERACTIONS = "kb5"
    KB6_EVIDENCE = "kb6"
    KB7_TERMINOLOGY = "kb7"
    KB8_WORKFLOWS = "kb8"


class StreamType(Enum):
    """Stream types for each KB"""
    PATIENT = "PatientStream"
    SEMANTIC = "SemanticStream"
    ANALYTICS = "AnalyticsStream"
    WORKFLOW = "WorkflowStream"


class MultiKBStreamManager:
    """
    Manages data streams for ALL CardioFit Knowledge Bases using logical partitioning.

    This shared component serves as the central Neo4j interface for all KBs,
    providing isolation through labels while enabling cross-KB queries when needed.
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Multi-KB Stream Manager

        Args:
            config: Configuration dictionary with Neo4j connection and KB definitions
        """
        self.driver = AsyncGraphDatabase.driver(
            config.get('neo4j_uri', 'bolt://localhost:7687'),
            auth=(config.get('neo4j_user', 'neo4j'),
                  config.get('neo4j_password', 'password')),
            max_connection_pool_size=100,  # Increased for multi-KB support
            connection_acquisition_timeout=30
        )

        # KB-specific stream labels
        self.kb_streams = {
            KnowledgeBase.KB1_PATIENT: {
                'primary': 'KB1_PatientStream',
                'semantic': 'KB1_SemanticStream'
            },
            KnowledgeBase.KB2_GUIDELINES: {
                'primary': 'KB2_GuidelineStream',
                'semantic': 'KB2_SemanticStream'
            },
            KnowledgeBase.KB3_DRUG_CALCULATIONS: {
                'primary': 'KB3_DrugCalculationStream',
                'semantic': 'KB3_SemanticStream'
            },
            KnowledgeBase.KB4_SAFETY_RULES: {
                'primary': 'KB4_SafetyStream',
                'semantic': 'KB4_SemanticStream'
            },
            KnowledgeBase.KB5_DRUG_INTERACTIONS: {
                'primary': 'KB5_InteractionStream',
                'semantic': 'KB5_SemanticStream'
            },
            KnowledgeBase.KB6_EVIDENCE: {
                'primary': 'KB6_EvidenceStream',
                'semantic': 'KB6_SemanticStream'
            },
            KnowledgeBase.KB7_TERMINOLOGY: {
                'primary': 'KB7_TerminologyStream',
                'semantic': 'KB7_SemanticStream'
            },
            KnowledgeBase.KB8_WORKFLOWS: {
                'primary': 'KB8_WorkflowStream',
                'semantic': 'KB8_SemanticStream'
            }
        }

        # Shared streams for cross-KB data
        self.shared_streams = {
            'semantic_mesh': 'SharedSemanticMesh',
            'global_patient': 'GlobalPatientStream',
            'cross_kb_relationships': 'CrossKBRelationshipStream'
        }

        self.config = config
        logger.info("Multi-KB Stream Manager initialized for all CardioFit Knowledge Bases")

    async def initialize_all_streams(self) -> bool:
        """
        Initialize logical partitions for all knowledge bases
        """
        try:
            async with self.driver.session() as session:
                # Check Neo4j version and edition
                version_result = await session.run("CALL dbms.components()")
                version_info = await version_result.single()
                logger.info(f"Neo4j Version: {version_info['versions'][0]} {version_info['edition']}")

                logger.info("Initializing streams for all Knowledge Bases")

                # Initialize indexes for all KB streams
                await self._create_multi_kb_indexes(session)

                # Create constraints for data integrity
                await self._create_multi_kb_constraints(session)

                # Initialize shared stream indexes
                await self._create_shared_stream_indexes(session)

                logger.info("Multi-KB stream initialization completed successfully")
                return True

        except Exception as e:
            logger.error(f"Error initializing multi-KB streams: {e}")
            return False

    async def _create_multi_kb_indexes(self, session) -> None:
        """Create indexes for all KB streams"""

        # KB-specific indexes
        indexes = []

        # KB1 - Patient Data
        indexes.extend([
            "CREATE INDEX kb1_patient_id IF NOT EXISTS FOR (p:Patient:KB1_PatientStream) ON (p.id)",
            "CREATE INDEX kb1_patient_mrn IF NOT EXISTS FOR (p:Patient:KB1_PatientStream) ON (p.mrn)",
        ])

        # KB2 - Guidelines
        indexes.extend([
            "CREATE INDEX kb2_guideline_id IF NOT EXISTS FOR (g:Guideline:KB2_GuidelineStream) ON (g.id)",
            "CREATE INDEX kb2_recommendation_id IF NOT EXISTS FOR (r:Recommendation:KB2_GuidelineStream) ON (r.id)",
        ])

        # KB3 - Drug Calculations
        indexes.extend([
            "CREATE INDEX kb3_calculation_rule IF NOT EXISTS FOR (c:CalculationRule:KB3_DrugCalculationStream) ON (c.drug_rxnorm)",
            "CREATE INDEX kb3_dosing_rule IF NOT EXISTS FOR (d:DosingRule:KB3_DrugCalculationStream) ON (d.indication)",
        ])

        # KB5 - Drug Interactions
        indexes.extend([
            "CREATE INDEX kb5_interaction_drugs IF NOT EXISTS FOR (i:Interaction:KB5_InteractionStream) ON (i.drug1_rxnorm, i.drug2_rxnorm)",
            "CREATE INDEX kb5_interaction_severity IF NOT EXISTS FOR (i:Interaction:KB5_InteractionStream) ON (i.severity)",
        ])

        # KB7 - Terminology
        indexes.extend([
            "CREATE INDEX kb7_concept_code IF NOT EXISTS FOR (c:Concept:KB7_TerminologyStream) ON (c.code, c.system)",
            "CREATE INDEX kb7_term_rxnorm IF NOT EXISTS FOR (t:Term:KB7_TerminologyStream) ON (t.rxnorm)",
        ])

        # Create all indexes
        for idx in indexes:
            try:
                await session.run(idx)
                logger.debug(f"Created index: {idx.split('FOR')[1].split('ON')[0].strip()}")
            except Exception as e:
                logger.warning(f"Index creation warning: {e}")

    async def _create_multi_kb_constraints(self, session) -> None:
        """Create constraints for multi-KB data integrity"""

        constraints = [
            # KB1 constraints
            "CREATE CONSTRAINT kb1_patient_id_unique IF NOT EXISTS FOR (p:Patient) REQUIRE (p.id, p.kb_source) IS UNIQUE",

            # KB7 constraints
            "CREATE CONSTRAINT kb7_concept_uri_unique IF NOT EXISTS FOR (c:Concept) REQUIRE (c.uri, c.kb_source) IS UNIQUE",

            # Shared constraints
            "CREATE CONSTRAINT shared_entity_id IF NOT EXISTS FOR (e:SharedEntity) REQUIRE e.global_id IS UNIQUE",
        ]

        for constraint in constraints:
            try:
                await session.run(constraint)
                logger.debug(f"Created constraint: {constraint.split('FOR')[1].split('REQUIRE')[0].strip()}")
            except Exception as e:
                logger.warning(f"Constraint creation warning: {e}")

    async def _create_shared_stream_indexes(self, session) -> None:
        """Create indexes for shared streams (cross-KB data)"""

        shared_indexes = [
            # Shared semantic mesh
            "CREATE INDEX shared_semantic_concept IF NOT EXISTS FOR (c:Concept:SharedSemanticMesh) ON (c.global_uri)",
            "CREATE INDEX shared_semantic_relationship IF NOT EXISTS FOR ()-[r:RELATES_TO:SharedSemanticMesh]-() ON (r.relationship_type)",

            # Global patient stream
            "CREATE INDEX global_patient_id IF NOT EXISTS FOR (p:Patient:GlobalPatientStream) ON (p.global_patient_id)",

            # Cross-KB relationships
            "CREATE INDEX cross_kb_source_target IF NOT EXISTS FOR ()-[r:CROSS_KB_RELATION]-() ON (r.source_kb, r.target_kb)",
        ]

        for idx in shared_indexes:
            try:
                await session.run(idx)
                logger.debug(f"Created shared index: {idx.split('FOR')[1].split('ON')[0].strip()}")
            except Exception as e:
                logger.warning(f"Shared index creation warning: {e}")

    async def load_kb_data(self, kb_id: Union[KnowledgeBase, str],
                          stream_type: StreamType,
                          entity_id: str,
                          data: Dict[str, Any]) -> bool:
        """
        Load data into a specific KB stream

        Args:
            kb_id: Knowledge Base identifier
            stream_type: Stream type (Patient, Semantic, etc.)
            entity_id: Unique entity identifier
            data: Entity data dictionary

        Returns:
            Success status
        """
        try:
            # Convert string to enum if needed
            if isinstance(kb_id, str):
                kb_id = KnowledgeBase(kb_id)

            # Get appropriate stream label
            if stream_type == StreamType.PATIENT:
                stream_label = self.kb_streams[kb_id]['primary']
            else:
                stream_label = self.kb_streams[kb_id]['semantic']

            async with self.driver.session() as session:
                # Add KB metadata to data
                enhanced_data = {
                    **data,
                    'kb_source': kb_id.value,
                    'stream_type': stream_type.value,
                    'updated': datetime.utcnow().isoformat()
                }

                # Create or update entity with KB-specific label
                result = await session.run(f"""
                    MERGE (e:{data.get('entity_type', 'Entity')}:{stream_label} {{id: $entity_id}})
                    SET e += $properties
                    SET e.updated = datetime()
                    RETURN e.id as entity_id
                """, entity_id=entity_id, properties=enhanced_data)

                record = await result.single()
                logger.info(f"Loaded data for {kb_id.value} {stream_type.value}: {record['entity_id']}")
                return True

        except Exception as e:
            logger.error(f"Failed to load {kb_id} data: {e}")
            return False

    async def query_kb_stream(self, kb_id: Union[KnowledgeBase, str],
                             stream_type: StreamType,
                             query: str,
                             params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """
        Query a specific KB stream

        Args:
            kb_id: Knowledge Base identifier
            stream_type: Stream type to query
            query: Cypher query (without MATCH clause - will be added automatically)
            params: Query parameters

        Returns:
            Query results
        """
        try:
            # Convert string to enum if needed
            if isinstance(kb_id, str):
                kb_id = KnowledgeBase(kb_id)

            # Get stream label
            if stream_type == StreamType.PATIENT:
                stream_label = self.kb_streams[kb_id]['primary']
            else:
                stream_label = self.kb_streams[kb_id]['semantic']

            async with self.driver.session() as session:
                # Execute query with KB-specific stream label
                full_query = f"""
                    MATCH (n:{stream_label})
                    {query}
                """

                result = await session.run(full_query, params or {})
                records = []
                async for record in result:
                    records.append(dict(record))

                logger.info(f"Query on {kb_id.value} {stream_type.value} returned {len(records)} results")
                return records

        except Exception as e:
            logger.error(f"Failed to query {kb_id} stream: {e}")
            return []

    async def cross_kb_query(self, kb_list: List[Union[KnowledgeBase, str]],
                            query: str,
                            params: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
        """
        Execute a query across multiple knowledge bases

        Args:
            kb_list: List of KB identifiers to query
            query: Cypher query spanning multiple KBs
            params: Query parameters

        Returns:
            Combined query results
        """
        try:
            # Convert strings to enums if needed
            kb_enums = []
            for kb in kb_list:
                if isinstance(kb, str):
                    kb_enums.append(KnowledgeBase(kb))
                else:
                    kb_enums.append(kb)

            async with self.driver.session() as session:
                # Execute cross-KB query
                result = await session.run(query, params or {})
                records = []
                async for record in result:
                    records.append(dict(record))

                logger.info(f"Cross-KB query across {[kb.value for kb in kb_enums]} returned {len(records)} results")
                return records

        except Exception as e:
            logger.error(f"Failed to execute cross-KB query: {e}")
            return []

    async def health_check_all_streams(self) -> Dict[str, bool]:
        """
        Check health of all KB streams

        Returns:
            Health status for each KB stream
        """
        health_status = {}

        try:
            async with self.driver.session() as session:
                # Test basic connectivity
                await session.run("RETURN 1 as test")

                # Check each KB stream
                for kb, streams in self.kb_streams.items():
                    primary_count = await session.run(
                        f"MATCH (n:{streams['primary']}) RETURN count(n) as count"
                    )
                    primary_result = await primary_count.single()

                    semantic_count = await session.run(
                        f"MATCH (n:{streams['semantic']}) RETURN count(n) as count"
                    )
                    semantic_result = await semantic_count.single()

                    health_status[kb.value] = {
                        'primary_nodes': primary_result['count'],
                        'semantic_nodes': semantic_result['count'],
                        'healthy': True
                    }

                logger.info(f"Multi-KB Health Check completed: {len(health_status)} KBs checked")
                return health_status

        except Exception as e:
            logger.error(f"Multi-KB Health check failed: {e}")
            return {kb.value: {'healthy': False, 'error': str(e)} for kb in self.kb_streams.keys()}

    async def close(self) -> None:
        """Close the Neo4j driver connection"""
        await self.driver.close()
        logger.info("Multi-KB Neo4j connection closed")


# Backward compatibility for KB7-specific usage
class Neo4jDualStreamManager(MultiKBStreamManager):
    """
    Backward compatibility wrapper for KB7-specific usage
    Maps old KB7-specific methods to new multi-KB interface
    """

    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.kb_id = KnowledgeBase.KB7_TERMINOLOGY

        # Legacy properties for backward compatibility
        self.patient_stream_label = "KB7_TerminologyStream"
        self.semantic_stream_label = "KB7_SemanticStream"
        self.patient_db = "patient_data"
        self.semantic_db = "semantic_mesh"

    async def initialize_databases(self) -> bool:
        """Legacy method - maps to new multi-KB initialization"""
        return await self.initialize_all_streams()

    async def load_patient_data(self, patient_id: str, patient_data: Dict[str, Any]) -> bool:
        """Legacy method - maps to new KB-specific data loading"""
        return await self.load_kb_data(
            self.kb_id,
            StreamType.PATIENT,
            patient_id,
            {'entity_type': 'Patient', **patient_data}
        )

    async def load_semantic_concept(self, concept_uri: str, concept_data: Dict[str, Any]) -> bool:
        """Legacy method - maps to new KB-specific data loading"""
        return await self.load_kb_data(
            self.kb_id,
            StreamType.SEMANTIC,
            concept_uri,
            {'entity_type': 'Concept', **concept_data}
        )

    async def health_check(self) -> bool:
        """Legacy method - maps to new health check"""
        health_status = await self.health_check_all_streams()
        return health_status.get(self.kb_id.value, {}).get('healthy', False)