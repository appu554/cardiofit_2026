"""
Test script for Neo4j Graph Projector Service
Tests graph schema creation, mutation execution, and queries
"""
import os
import sys
import json
import time
from pathlib import Path

# Add shared module to path
shared_module_path = Path(__file__).parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from neo4j import GraphDatabase
from kafka import KafkaProducer
import structlog

structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer()
    ]
)

logger = structlog.get_logger(__name__)


class Neo4jGraphProjectorTester:
    """Test Neo4j graph projector functionality"""

    def __init__(self):
        # Neo4j configuration (using localhost port mapping)
        self.neo4j_uri = os.getenv("NEO4J_URI", "bolt://localhost:7687")
        self.neo4j_username = os.getenv("NEO4J_USERNAME", "neo4j")
        self.neo4j_password = os.getenv("NEO4J_PASSWORD", "CardioFit2024!")
        self.neo4j_database = os.getenv("NEO4J_DATABASE", "neo4j")

        # Kafka configuration
        self.kafka_bootstrap = os.getenv(
            "KAFKA_BOOTSTRAP_SERVERS",
            "pkc-p11xm.us-east-1.aws.confluent.cloud:9092"
        )
        self.kafka_api_key = os.getenv("KAFKA_API_KEY", "")
        self.kafka_api_secret = os.getenv("KAFKA_API_SECRET", "")

        # Initialize Neo4j driver
        self.driver = GraphDatabase.driver(
            self.neo4j_uri,
            auth=(self.neo4j_username, self.neo4j_password)
        )

        logger.info("Tester initialized", neo4j_uri=self.neo4j_uri)

    def test_neo4j_connection(self) -> bool:
        """Test Neo4j connection"""
        try:
            with self.driver.session(database=self.neo4j_database) as session:
                result = session.run("RETURN 1 as test")
                result.single()

            logger.info("✅ Neo4j connection successful")
            return True

        except Exception as e:
            logger.error("❌ Neo4j connection failed", error=str(e))
            return False

    def test_schema_creation(self) -> bool:
        """Test schema constraints and indexes"""
        try:
            with self.driver.session(database=self.neo4j_database) as session:
                # Check constraints
                result = session.run("SHOW CONSTRAINTS")
                constraints = list(result)

                logger.info(f"✅ Found {len(constraints)} constraints")

                # Check indexes
                result = session.run("SHOW INDEXES")
                indexes = list(result)

                logger.info(f"✅ Found {len(indexes)} indexes")

                return len(constraints) >= 7  # Should have at least 7 unique constraints

        except Exception as e:
            logger.error("❌ Schema check failed", error=str(e))
            return False

    def test_graph_mutation(self) -> bool:
        """Test executing a sample graph mutation"""
        try:
            with self.driver.session(database=self.neo4j_database) as session:
                # Create a test patient node
                query = """
                    MERGE (p:Patient {nodeId: 'TEST_P001'})
                    SET p.firstName = 'John',
                        p.lastName = 'Doe',
                        p.dateOfBirth = '1980-01-15',
                        p.lastUpdated = timestamp()
                    RETURN p
                """
                result = session.run(query)
                patient = result.single()

                if not patient:
                    logger.error("❌ Patient node creation failed")
                    return False

                # Create a test clinical event node
                query = """
                    MERGE (e:ClinicalEvent {nodeId: 'TEST_E001'})
                    SET e.patientId = 'TEST_P001',
                        e.eventType = 'VITAL_SIGNS',
                        e.timestamp = 1700000000000,
                        e.lastUpdated = timestamp()
                    RETURN e
                """
                result = session.run(query)
                event = result.single()

                if not event:
                    logger.error("❌ Event node creation failed")
                    return False

                # Create relationship
                query = """
                    MATCH (p:Patient {nodeId: 'TEST_P001'})
                    MATCH (e:ClinicalEvent {nodeId: 'TEST_E001'})
                    MERGE (p)-[r:HAS_EVENT]->(e)
                    SET r.lastUpdated = timestamp()
                    RETURN r
                """
                result = session.run(query)
                relationship = result.single()

                if not relationship:
                    logger.error("❌ Relationship creation failed")
                    return False

                logger.info("✅ Graph mutation successful - created patient, event, and relationship")
                return True

        except Exception as e:
            logger.error("❌ Graph mutation failed", error=str(e))
            return False

    def test_patient_journey_query(self) -> bool:
        """Test patient journey query"""
        try:
            with self.driver.session(database=self.neo4j_database) as session:
                query = """
                    MATCH (p:Patient {nodeId: 'TEST_P001'})-[:HAS_EVENT]->(e:ClinicalEvent)
                    RETURN p, e
                    ORDER BY e.timestamp
                """
                result = session.run(query)
                records = list(result)

                if len(records) > 0:
                    logger.info(f"✅ Patient journey query successful - found {len(records)} events")
                    return True
                else:
                    logger.warning("⚠️ Patient journey query returned no results")
                    return False

        except Exception as e:
            logger.error("❌ Patient journey query failed", error=str(e))
            return False

    def get_graph_stats(self) -> dict:
        """Get graph statistics"""
        try:
            with self.driver.session(database=self.neo4j_database) as session:
                # Count nodes by type
                node_types = ["Patient", "ClinicalEvent", "Condition", "Medication",
                             "Procedure", "Department", "Device"]
                node_counts = {}

                for node_type in node_types:
                    result = session.run(f"MATCH (n:{node_type}) RETURN count(n) as count")
                    node_counts[node_type] = result.single()["count"]

                # Count relationships
                result = session.run("MATCH ()-[r]->() RETURN count(r) as count")
                rel_count = result.single()["count"]

                stats = {
                    "node_counts": node_counts,
                    "relationship_count": rel_count,
                    "total_nodes": sum(node_counts.values()),
                }

                logger.info("Graph statistics", stats=stats)
                return stats

        except Exception as e:
            logger.error("Failed to get graph stats", error=str(e))
            return {}

    def send_test_mutation_to_kafka(self) -> bool:
        """Send a test GraphMutation to Kafka topic"""
        if not self.kafka_api_key or not self.kafka_api_secret:
            logger.warning("⚠️ Kafka credentials not set, skipping Kafka test")
            return False

        try:
            # Initialize Kafka producer
            producer = KafkaProducer(
                bootstrap_servers=self.kafka_bootstrap,
                security_protocol="SASL_SSL",
                sasl_mechanism="PLAIN",
                sasl_plain_username=self.kafka_api_key,
                sasl_plain_password=self.kafka_api_secret,
                value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            )

            # Create test mutation
            mutation = {
                "mutationType": "MERGE",
                "nodeType": "Patient",
                "nodeId": "KAFKA_TEST_P001",
                "timestamp": int(time.time() * 1000),
                "nodeProperties": {
                    "firstName": "Jane",
                    "lastName": "Smith",
                    "dateOfBirth": "1990-05-20",
                },
                "relationships": [
                    {
                        "relationType": "LOCATED_IN",
                        "targetNodeType": "Department",
                        "targetNodeId": "DEPT_ICU",
                        "relationshipProperties": {
                            "admissionTime": int(time.time() * 1000),
                        }
                    }
                ]
            }

            # Send to Kafka
            future = producer.send(
                "prod.ehr.graph.mutations",
                key=mutation["nodeId"].encode('utf-8'),
                value=mutation
            )

            # Wait for send to complete
            future.get(timeout=10)
            producer.flush()
            producer.close()

            logger.info("✅ Test mutation sent to Kafka topic prod.ehr.graph.mutations")
            return True

        except Exception as e:
            logger.error("❌ Failed to send mutation to Kafka", error=str(e))
            return False

    def cleanup_test_data(self) -> None:
        """Clean up test data"""
        try:
            with self.driver.session(database=self.neo4j_database) as session:
                # Delete test nodes and relationships
                query = """
                    MATCH (n)
                    WHERE n.nodeId STARTS WITH 'TEST_' OR n.nodeId STARTS WITH 'KAFKA_TEST_'
                    DETACH DELETE n
                """
                session.run(query)

                logger.info("✅ Test data cleaned up")

        except Exception as e:
            logger.error("Failed to clean up test data", error=str(e))

    def run_all_tests(self) -> None:
        """Run all tests"""
        logger.info("=" * 80)
        logger.info("Starting Neo4j Graph Projector Tests")
        logger.info("=" * 80)

        results = {}

        # Test 1: Neo4j connection
        results["connection"] = self.test_neo4j_connection()

        # Test 2: Schema creation
        results["schema"] = self.test_schema_creation()

        # Test 3: Graph mutation
        results["mutation"] = self.test_graph_mutation()

        # Test 4: Patient journey query
        results["query"] = self.test_patient_journey_query()

        # Test 5: Graph statistics
        stats = self.get_graph_stats()

        # Test 6: Kafka integration (optional)
        results["kafka"] = self.send_test_mutation_to_kafka()

        # Wait for projector to process (if Kafka test succeeded)
        if results["kafka"]:
            logger.info("Waiting 10 seconds for projector to process Kafka mutation...")
            time.sleep(10)

            # Check if mutation was processed
            stats_after = self.get_graph_stats()
            logger.info("Graph stats after Kafka mutation", stats=stats_after)

        # Cleanup
        self.cleanup_test_data()

        # Summary
        logger.info("=" * 80)
        logger.info("Test Results Summary")
        logger.info("=" * 80)

        for test_name, result in results.items():
            status = "✅ PASSED" if result else "❌ FAILED"
            logger.info(f"{test_name.upper()}: {status}")

        total_tests = len(results)
        passed_tests = sum(1 for r in results.values() if r)

        logger.info(f"\nTotal: {passed_tests}/{total_tests} tests passed")
        logger.info("=" * 80)

    def close(self) -> None:
        """Close connections"""
        if self.driver:
            self.driver.close()


if __name__ == "__main__":
    tester = Neo4jGraphProjectorTester()
    try:
        tester.run_all_tests()
    finally:
        tester.close()
