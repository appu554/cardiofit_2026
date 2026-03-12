"""
Neo4j Graph Projector Service
Consumes graph mutations from Kafka and executes Cypher queries to build patient journey graphs
"""
import sys
from pathlib import Path
from typing import List, Any, Optional
from datetime import datetime

from neo4j import GraphDatabase, Session
from neo4j.exceptions import ServiceUnavailable, TransientError
import structlog

# Add shared module to path
shared_module_path = Path(__file__).parent.parent.parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from module8_shared.kafka_consumer_base import KafkaConsumerBase
from module8_shared.models.events import GraphMutation

from .cypher_query_builder import CypherQueryBuilder

logger = structlog.get_logger(__name__)


class Neo4jGraphProjector(KafkaConsumerBase):
    """
    Projects graph mutations to Neo4j patient journey graphs

    Input: GraphMutation objects from prod.ehr.graph.mutations
    Output: Neo4j graph with patient journeys, clinical events, and relationships

    Supported Node Types:
    - Patient
    - ClinicalEvent
    - Condition
    - Medication
    - Procedure
    - Department
    - Device

    Supported Relationship Types:
    - HAS_EVENT (Patient → ClinicalEvent)
    - HAS_CONDITION (Patient → Condition)
    - PRESCRIBED (Patient → Medication)
    - UNDERWENT (Patient → Procedure)
    - NEXT_EVENT (ClinicalEvent → ClinicalEvent)
    - TRIGGERED_BY (ClinicalEvent → Condition)
    - LOCATED_IN (Patient → Department)
    - MEASURED_BY (ClinicalEvent → Device)
    """

    def __init__(self, kafka_config: dict, neo4j_config: dict):
        super().__init__(
            kafka_config=kafka_config,
            topics=["prod.ehr.graph.mutations"],
            batch_size=50,
            batch_timeout_seconds=5.0,
            dlq_topic="prod.ehr.dlq.neo4j",
        )
        self.neo4j_config = neo4j_config
        self.last_processed_time: Optional[datetime] = None
        self.query_builder = CypherQueryBuilder()

        # Initialize Neo4j driver
        self.driver = GraphDatabase.driver(
            neo4j_config["uri"],
            auth=(neo4j_config["username"], neo4j_config["password"]),
            max_connection_lifetime=3600,
            max_connection_pool_size=50,
            connection_acquisition_timeout=60.0,
        )

        # Test connection and create schema
        self._test_connection()
        self._create_schema()

        logger.info(
            "Neo4j graph projector initialized",
            neo4j_uri=neo4j_config["uri"],
            neo4j_database=neo4j_config["database"],
        )

    def _test_connection(self) -> None:
        """Test Neo4j connection"""
        try:
            with self.driver.session(database=self.neo4j_config["database"]) as session:
                result = session.run("RETURN 1 as test")
                result.single()
                logger.info("Neo4j connection successful")
        except Exception as e:
            logger.error("Neo4j connection failed", error=str(e))
            raise

    def _create_schema(self) -> None:
        """Create constraints and indexes on startup"""
        try:
            with self.driver.session(database=self.neo4j_config["database"]) as session:
                constraint_queries = self.query_builder.get_constraint_queries()

                for query in constraint_queries:
                    try:
                        session.run(query)
                        logger.debug("Executed schema query", query=query[:50])
                    except Exception as e:
                        # Ignore if constraint already exists
                        if "already exists" in str(e).lower():
                            logger.debug("Constraint already exists", query=query[:50])
                        else:
                            logger.warning("Schema query failed", error=str(e), query=query[:50])

                logger.info("Neo4j schema creation complete")
        except Exception as e:
            logger.error("Failed to create schema", error=str(e))
            raise

    def get_projector_name(self) -> str:
        return "neo4j-graph-projector"

    def process_batch(self, messages: List[Any]) -> None:
        """Execute graph mutations in a Neo4j transaction"""
        if not messages:
            return

        # Parse messages as GraphMutation
        mutations = []
        for msg in messages:
            try:
                mutation = GraphMutation(**msg)
                mutations.append(mutation)
            except Exception as e:
                logger.error("Failed to parse mutation", error=str(e), message=msg)
                continue

        if not mutations:
            logger.warning("No valid mutations to process")
            return

        # Execute mutations in Neo4j transaction
        try:
            with self.driver.session(database=self.neo4j_config["database"]) as session:
                # Use write transaction for consistency
                result = session.execute_write(self._execute_mutations, mutations)

                self.last_processed_time = datetime.utcnow()

                logger.info(
                    "Batch written to Neo4j",
                    batch_size=len(mutations),
                    total_messages=len(messages),
                    nodes_created=result["nodes_created"],
                    relationships_created=result["relationships_created"],
                    merge_operations=len([m for m in mutations if m.mutation_type == "MERGE"]),
                    create_operations=len([m for m in mutations if m.mutation_type == "CREATE"]),
                )

        except (ServiceUnavailable, TransientError) as e:
            logger.error("Neo4j transient error, will retry", error=str(e))
            raise
        except Exception as e:
            logger.error("Neo4j batch write failed", error=str(e), exc_info=True)
            raise

    def _execute_mutations(self, tx, mutations: List[GraphMutation]) -> dict:
        """
        Execute mutations in a transaction

        Args:
            tx: Neo4j transaction
            mutations: List of GraphMutation objects

        Returns:
            Dictionary with execution statistics
        """
        nodes_created = 0
        relationships_created = 0

        for mutation in mutations:
            try:
                # Execute node mutation
                if mutation.mutation_type == "MERGE":
                    query, params = self.query_builder.build_merge_node(mutation)
                    result = tx.run(query, params)
                    result.single()
                    nodes_created += 1

                elif mutation.mutation_type == "CREATE":
                    query, params = self.query_builder.build_create_node(mutation)
                    result = tx.run(query, params)
                    result.single()
                    nodes_created += 1

                else:
                    logger.warning("Unknown mutation type", mutation_type=mutation.mutation_type)
                    continue

                # Execute relationship mutations
                for rel in mutation.relationships:
                    query, params = self.query_builder.build_relationship(mutation, rel)
                    result = tx.run(query, params)
                    result.single()
                    relationships_created += 1

            except Exception as e:
                logger.error(
                    "Failed to execute mutation",
                    error=str(e),
                    mutation_type=mutation.mutation_type,
                    node_type=mutation.node_type,
                    node_id=mutation.node_id
                )
                raise

        return {
            "nodes_created": nodes_created,
            "relationships_created": relationships_created,
        }

    def get_graph_stats(self) -> dict:
        """Get graph statistics"""
        try:
            with self.driver.session(database=self.neo4j_config["database"]) as session:
                # Count nodes by type
                node_counts = {}
                node_types = ["Patient", "ClinicalEvent", "Condition", "Medication",
                             "Procedure", "Department", "Device"]

                for node_type in node_types:
                    result = session.run(f"MATCH (n:{node_type}) RETURN count(n) as count")
                    node_counts[node_type] = result.single()["count"]

                # Count relationships
                rel_result = session.run("MATCH ()-[r]->() RETURN count(r) as count")
                rel_count = rel_result.single()["count"]

                return {
                    "node_counts": node_counts,
                    "relationship_count": rel_count,
                    "total_nodes": sum(node_counts.values()),
                }

        except Exception as e:
            logger.error("Failed to get graph stats", error=str(e))
            return {
                "node_counts": {},
                "relationship_count": 0,
                "total_nodes": 0,
            }

    def query_patient_journey(self, patient_id: str) -> List[dict]:
        """
        Query patient journey for a specific patient

        Args:
            patient_id: Patient node ID

        Returns:
            List of events in chronological order
        """
        try:
            with self.driver.session(database=self.neo4j_config["database"]) as session:
                query = """
                    MATCH (p:Patient {nodeId: $patientId})-[:HAS_EVENT]->(e:ClinicalEvent)
                    RETURN e
                    ORDER BY e.timestamp
                """
                result = session.run(query, patientId=patient_id)
                events = [record["e"] for record in result]
                return events

        except Exception as e:
            logger.error("Failed to query patient journey", error=str(e), patient_id=patient_id)
            return []

    def shutdown(self) -> None:
        """Gracefully shutdown projector"""
        logger.info("Shutting down Neo4j graph projector")

        # Close Neo4j driver
        if self.driver:
            self.driver.close()
            logger.info("Neo4j driver closed")

        # Call parent shutdown
        super().shutdown()
