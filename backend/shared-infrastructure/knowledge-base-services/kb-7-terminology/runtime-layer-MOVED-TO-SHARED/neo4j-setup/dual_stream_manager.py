"""
Neo4j Dual-Stream Manager for KB7 Terminology Service
Manages two distinct data streams using logical partitioning:
1. Patient Data Stream (real-time clinical data) - :PatientStream label
2. Semantic Mesh Stream (from GraphDB OWL reasoning) - :SemanticStream label

Compatible with both Neo4j Community and Enterprise editions.
"""

import asyncio
from neo4j import AsyncGraphDatabase
from typing import Dict, List, Optional, Any
import json
import hashlib
from datetime import datetime
from loguru import logger


class Neo4jDualStreamManager:
    """
    Manages two distinct data streams in Neo4j using logical partitioning:
    1. Patient Data Stream (real-time clinical data) - nodes labeled :PatientStream
    2. Semantic Mesh Stream (from GraphDB OWL reasoning) - nodes labeled :SemanticStream

    This approach works with Neo4j Community Edition using label-based partitioning,
    and can seamlessly transition to Enterprise multi-database when available.
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Neo4j Dual Stream Manager

        Args:
            config: Configuration dictionary containing Neo4j connection details
        """
        self.driver = AsyncGraphDatabase.driver(
            config.get('neo4j_uri', 'bolt://localhost:7687'),
            auth=(config.get('neo4j_user', 'neo4j'),
                  config.get('neo4j_password', 'password')),
            max_connection_pool_size=50,
            connection_acquisition_timeout=30
        )

        # Logical stream labels for partitioning
        self.patient_stream_label = "PatientStream"
        self.semantic_stream_label = "SemanticStream"

        # Legacy database names (for Enterprise compatibility)
        self.patient_db = "patient_data"
        self.semantic_db = "semantic_mesh"
        self.config = config

        logger.info("Neo4j Dual Stream Manager initialized with logical partitioning")

    async def initialize_databases(self) -> bool:
        """
        Initialize logical partitions using stream labels
        Compatible with Neo4j Community Edition - uses single database with logical partitioning
        """

        try:
            async with self.driver.session() as session:
                # Check Neo4j version and edition to determine approach
                version_result = await session.run("CALL dbms.components()")
                version_info = await version_result.single()
                logger.info(f"Neo4j Version: {version_info['versions'][0]} {version_info['edition']}")

                # Always use logical partitioning (Community compatible)
                logger.info("Using logical partitioning with stream labels")

                # Initialize indexes for both streams in single database
                await self._create_patient_stream_indexes(session)
                await self._create_semantic_stream_indexes(session)

                # Create constraints for data integrity
                await self._create_stream_constraints(session)

                logger.info("KB7 dual-stream logical partitioning initialized successfully")
                return True

        except Exception as e:
            logger.error(f"Error initializing KB7 dual-stream partitioning: {e}")
            return False

    async def _create_patient_stream_indexes(self, session) -> None:
        """Create indexes for PatientStream labeled nodes"""

        indexes = [
            # Patient indexes
            "CREATE INDEX patient_id IF NOT EXISTS FOR (p:Patient) ON (p.id)",
            "CREATE INDEX patient_mrn IF NOT EXISTS FOR (p:Patient) ON (p.mrn)",

            # Medication indexes
            "CREATE INDEX medication_rxnorm IF NOT EXISTS FOR (m:Medication) ON (m.rxnorm)",
            "CREATE INDEX medication_name IF NOT EXISTS FOR (m:Medication) ON (m.name)",

            # Condition indexes
            "CREATE INDEX condition_icd10 IF NOT EXISTS FOR (c:Condition) ON (c.icd10)",
            "CREATE INDEX condition_snomed IF NOT EXISTS FOR (c:Condition) ON (c.snomed)",

            # Encounter indexes
            "CREATE INDEX encounter_id IF NOT EXISTS FOR (e:Encounter) ON (e.id)",
            "CREATE INDEX encounter_date IF NOT EXISTS FOR (e:Encounter) ON (e.date)",

            # Observation indexes
            "CREATE INDEX observation_loinc IF NOT EXISTS FOR (o:Observation) ON (o.loinc)",

            # Temporal indexes
            "CREATE INDEX timestamp_idx IF NOT EXISTS FOR (n) ON (n.timestamp)",
            "CREATE INDEX effective_date IF NOT EXISTS FOR (n) ON (n.effective_date)"
        ]

        for idx in indexes:
            try:
                await session.run(idx)
                logger.debug(f"Created index: {idx.split('FOR')[1].split('ON')[0].strip()}")
            except Exception as e:
                logger.warning(f"Index creation warning: {e}")

    async def _create_semantic_stream_indexes(self, session) -> None:
        """Create indexes for SemanticStream labeled nodes"""

        indexes = [
            # Concept indexes (SemanticStream)
            "CREATE INDEX concept_uri IF NOT EXISTS FOR (c:Concept) ON (c.uri)",
            "CREATE INDEX concept_code IF NOT EXISTS FOR (c:Concept) ON (c.code)",
            "CREATE INDEX concept_system IF NOT EXISTS FOR (c:Concept) ON (c.system)",

            # Drug class indexes (SemanticStream)
            "CREATE INDEX drug_class IF NOT EXISTS FOR (dc:DrugClass) ON (dc.code)",
            "CREATE INDEX drug_class_name IF NOT EXISTS FOR (dc:DrugClass) ON (dc.name)",

            # Terminology indexes (SemanticStream)
            "CREATE INDEX terminology_code IF NOT EXISTS FOR (t:Term) ON (t.code, t.system)",

            # Relationship indexes
            "CREATE INDEX interaction_severity IF NOT EXISTS FOR ()-[i:INTERACTS_WITH]-() ON (i.severity)",
            "CREATE INDEX contraindication IF NOT EXISTS FOR ()-[c:CONTRAINDICATED_IN]-() ON (c.severity)",
            "CREATE INDEX subsumption IF NOT EXISTS FOR ()-[s:IS_A]-() ON (s.source)",

            # Clinical guideline indexes (SemanticStream)
            "CREATE INDEX guideline_id IF NOT EXISTS FOR (g:Guideline) ON (g.id)",
            "CREATE INDEX recommendation_strength IF NOT EXISTS FOR (r:Recommendation) ON (r.strength)"
        ]

        for idx in indexes:
            try:
                await session.run(idx)
                logger.debug(f"Created semantic index: {idx.split('FOR')[1].split('ON')[0].strip()}")
            except Exception as e:
                logger.warning(f"Semantic index creation warning: {e}")

    async def _create_stream_constraints(self, session) -> None:
        """Create constraints for stream data integrity"""

        constraints = [
            # Patient stream constraints
            "CREATE CONSTRAINT patient_id_unique IF NOT EXISTS FOR (p:Patient:PatientStream) REQUIRE p.id IS UNIQUE",
            "CREATE CONSTRAINT patient_mrn_unique IF NOT EXISTS FOR (p:Patient:PatientStream) REQUIRE p.mrn IS UNIQUE",

            # Semantic stream constraints
            "CREATE CONSTRAINT concept_uri_unique IF NOT EXISTS FOR (c:Concept:SemanticStream) REQUIRE c.uri IS UNIQUE",
            "CREATE CONSTRAINT drug_class_code_unique IF NOT EXISTS FOR (dc:DrugClass:SemanticStream) REQUIRE dc.code IS UNIQUE"
        ]

        for constraint in constraints:
            try:
                await session.run(constraint)
                logger.debug(f"Created constraint: {constraint.split('FOR')[1].split('REQUIRE')[0].strip()}")
            except Exception as e:
                logger.warning(f"Constraint creation warning: {e}")

    async def load_patient_data(self, patient_id: str, patient_data: Dict[str, Any]) -> bool:
        """
        Load patient data into the PatientStream logical partition

        Args:
            patient_id: Unique patient identifier
            patient_data: Patient data dictionary

        Returns:
            Success status
        """
        try:
            async with self.driver.session() as session:
                result = await session.run("""
                    MERGE (p:Patient:PatientStream {id: $patient_id})
                    SET p += $properties
                    SET p.updated = datetime()
                    RETURN p.id as patient_id
                """, patient_id=patient_id, properties=patient_data)

                record = await result.single()
                logger.info(f"Loaded patient data for PatientStream: {record['patient_id']}")
                return True
        except Exception as e:
            logger.error(f"Failed to load patient data: {e}")
            return False

    async def load_semantic_concept(self, concept_uri: str, concept_data: Dict[str, Any]) -> bool:
        """
        Load semantic concept into the SemanticStream logical partition

        Args:
            concept_uri: Unique concept URI
            concept_data: Concept data from GraphDB/OWL reasoning

        Returns:
            Success status
        """
        try:
            async with self.driver.session() as session:
                result = await session.run("""
                    MERGE (c:Concept:SemanticStream {uri: $uri})
                    SET c += $properties
                    SET c.updated = datetime()
                    RETURN c.uri as concept_uri
                """, uri=concept_uri, properties=concept_data)

                record = await result.single()
                logger.info(f"Loaded semantic concept for SemanticStream: {record['concept_uri']}")
                return True
        except Exception as e:
            logger.error(f"Failed to load semantic concept: {e}")
            return False

    async def query_drug_interactions(self, drug_codes: List[str]) -> List[Dict[str, Any]]:
        """
        Query drug interactions from SemanticStream logical partition

        Args:
            drug_codes: List of drug codes (RxNorm)

        Returns:
            List of interaction details
        """
        try:
            async with self.driver.session() as session:
                result = await session.run("""
                    MATCH (d1:Drug:SemanticStream)-[i:INTERACTS_WITH]-(d2:Drug:SemanticStream)
                    WHERE d1.rxnorm IN $drug_codes AND d2.rxnorm IN $drug_codes
                    AND d1.rxnorm < d2.rxnorm  // Avoid duplicates
                    RETURN d1.rxnorm as drug1,
                           d2.rxnorm as drug2,
                           i.severity as severity,
                           i.mechanism as mechanism,
                           i.clinical_significance as significance
                """, drug_codes=drug_codes)

                interactions = []
                async for record in result:
                    interactions.append(dict(record))

                logger.info(f"Found {len(interactions)} drug interactions in SemanticStream")
                return interactions
        except Exception as e:
            logger.error(f"Failed to query drug interactions: {e}")
            return []

    async def get_patient_medications(self, patient_id: str) -> List[Dict[str, Any]]:
        """
        Get current medications for a patient from PatientStream logical partition

        Args:
            patient_id: Patient identifier

        Returns:
            List of current medications
        """
        try:
            async with self.driver.session() as session:
                result = await session.run("""
                    MATCH (p:Patient:PatientStream {id: $patient_id})-[r:PRESCRIBED]->(m:Medication:PatientStream)
                    WHERE r.end_date IS NULL OR r.end_date > datetime()
                    RETURN m.rxnorm as rxnorm,
                           m.name as name,
                           r.dose as dose,
                           r.frequency as frequency,
                           r.start_date as start_date
                """, patient_id=patient_id)

                medications = []
                async for record in result:
                    medications.append(dict(record))

                logger.info(f"Found {len(medications)} medications for patient {patient_id} in PatientStream")
                return medications
        except Exception as e:
            logger.error(f"Failed to get patient medications: {e}")
            return []

    async def find_contraindications(self, drug_code: str,
                                    condition_codes: List[str]) -> List[Dict[str, Any]]:
        """
        Find contraindications between a drug and patient conditions from SemanticStream

        Args:
            drug_code: Drug RxNorm code
            condition_codes: List of condition codes (ICD10/SNOMED)

        Returns:
            List of contraindications
        """
        try:
            async with self.driver.session() as session:
                result = await session.run("""
                    MATCH (d:Drug:SemanticStream {rxnorm: $drug_code})-[c:CONTRAINDICATED_IN]->(cond:Condition:SemanticStream)
                    WHERE cond.code IN $condition_codes OR cond.snomed IN $condition_codes
                    RETURN cond.code as condition_code,
                           cond.name as condition_name,
                           c.severity as severity,
                           c.rationale as rationale,
                           c.evidence_level as evidence_level
                """, drug_code=drug_code, condition_codes=condition_codes)

                contraindications = []
                async for record in result:
                    contraindications.append(dict(record))

                logger.info(f"Found {len(contraindications)} contraindications for drug {drug_code} in SemanticStream")
                return contraindications
        except Exception as e:
            logger.error(f"Failed to find contraindications: {e}")
            return []

    async def close(self) -> None:
        """Close the Neo4j driver connection"""
        await self.driver.close()
        logger.info("Neo4j connection closed")

    async def health_check(self) -> bool:
        """
        Check health of Neo4j dual-stream logical partitioning

        Returns:
            Health status
        """
        try:
            async with self.driver.session() as session:
                # Test basic connectivity
                await session.run("RETURN 1 as test")

                # Check PatientStream partition
                patient_count = await session.run(
                    "MATCH (n:PatientStream) RETURN count(n) as count"
                )
                patient_result = await patient_count.single()

                # Check SemanticStream partition
                semantic_count = await session.run(
                    "MATCH (n:SemanticStream) RETURN count(n) as count"
                )
                semantic_result = await semantic_count.single()

                logger.info(f"KB7 Health Check: PatientStream nodes: {patient_result['count']}, SemanticStream nodes: {semantic_result['count']}")
                return True

        except Exception as e:
            logger.error(f"KB7 Health check failed: {e}")
            return False