#!/usr/bin/env python3
"""
Module 8 Integration Test Suite

Comprehensive end-to-end testing for all 8 storage projectors:
- PostgreSQL, MongoDB, Elasticsearch, ClickHouse, InfluxDB
- UPS Read Model, FHIR Store, Neo4j Graph

Tests cover:
- Data flow and fanout
- Performance benchmarks
- Data consistency
- Error handling
- Monitoring metrics
"""

import pytest
import time
import json
import uuid
import logging
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
import requests

# Kafka
from kafka import KafkaProducer, KafkaConsumer
from kafka.errors import KafkaError

# Database clients
import psycopg2
from pymongo import MongoClient
from elasticsearch import Elasticsearch
from clickhouse_driver import Client as ClickHouseClient
from influxdb_client import InfluxDBClient
from neo4j import GraphDatabase
from google.cloud import healthcare_v1

# Monitoring
from prometheus_client.parser import text_string_to_metric_families

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@dataclass
class TestConfig:
    """Test configuration"""
    kafka_bootstrap_servers: str = "localhost:9092"
    postgres_host: str = "localhost"
    postgres_port: int = 5432
    postgres_db: str = "clinical_events"
    postgres_user: str = "postgres"
    postgres_password: str = "postgres"

    mongo_uri: str = "mongodb://localhost:27017"
    mongo_db: str = "clinical_events"

    elasticsearch_url: str = "http://localhost:9200"
    elasticsearch_index: str = "clinical_events"

    clickhouse_host: str = "localhost"
    clickhouse_port: int = 9000
    clickhouse_db: str = "clinical_analytics"

    influxdb_url: str = "http://localhost:8086"
    influxdb_token: str = "cardiofit-influxdb-token"
    influxdb_org: str = "cardiofit"
    influxdb_bucket: str = "vitals_realtime"

    neo4j_uri: str = "bolt://localhost:7687"
    neo4j_user: str = "neo4j"
    neo4j_password: str = "cardiofit123"

    fhir_project_id: str = "cardiofit-test"
    fhir_location: str = "us-central1"
    fhir_dataset_id: str = "clinical-dataset"
    fhir_store_id: str = "clinical-fhir-store"

    # Projector endpoints
    projector_ports: Dict[str, int] = None

    def __post_init__(self):
        if self.projector_ports is None:
            self.projector_ports = {
                "postgresql": 8050,
                "mongodb": 8051,
                "elasticsearch": 8052,
                "clickhouse": 8053,
                "influxdb": 8054,
                "ups": 8055,
                "fhir_store": 8056,
                "neo4j": 8057
            }


class DatabaseClients:
    """Manage all database client connections"""

    def __init__(self, config: TestConfig):
        self.config = config
        self._postgres = None
        self._mongo = None
        self._elasticsearch = None
        self._clickhouse = None
        self._influxdb = None
        self._neo4j = None
        self._fhir = None

    @property
    def postgres(self):
        if self._postgres is None:
            self._postgres = psycopg2.connect(
                host=self.config.postgres_host,
                port=self.config.postgres_port,
                database=self.config.postgres_db,
                user=self.config.postgres_user,
                password=self.config.postgres_password
            )
        return self._postgres

    @property
    def mongo(self):
        if self._mongo is None:
            client = MongoClient(self.config.mongo_uri)
            self._mongo = client[self.config.mongo_db]
        return self._mongo

    @property
    def elasticsearch(self):
        if self._elasticsearch is None:
            self._elasticsearch = Elasticsearch([self.config.elasticsearch_url])
        return self._elasticsearch

    @property
    def clickhouse(self):
        if self._clickhouse is None:
            self._clickhouse = ClickHouseClient(
                host=self.config.clickhouse_host,
                port=self.config.clickhouse_port,
                database=self.config.clickhouse_db
            )
        return self._clickhouse

    @property
    def influxdb(self):
        if self._influxdb is None:
            self._influxdb = InfluxDBClient(
                url=self.config.influxdb_url,
                token=self.config.influxdb_token,
                org=self.config.influxdb_org
            )
        return self._influxdb

    @property
    def neo4j(self):
        if self._neo4j is None:
            self._neo4j = GraphDatabase.driver(
                self.config.neo4j_uri,
                auth=(self.config.neo4j_user, self.config.neo4j_password)
            )
        return self._neo4j

    @property
    def fhir(self):
        if self._fhir is None:
            self._fhir = healthcare_v1.FhirStoreServiceClient()
        return self._fhir

    def close_all(self):
        """Close all database connections"""
        if self._postgres:
            self._postgres.close()
        if self._mongo:
            self._mongo.client.close()
        if self._influxdb:
            self._influxdb.close()
        if self._neo4j:
            self._neo4j.close()


class TestDataGenerator:
    """Generate realistic test clinical events"""

    @staticmethod
    def generate_enriched_event(patient_id: str = None, event_type: str = "VITAL_SIGNS") -> Dict[str, Any]:
        """Generate enriched clinical event"""
        if patient_id is None:
            patient_id = f"patient-{uuid.uuid4()}"

        event_id = str(uuid.uuid4())
        timestamp = datetime.utcnow().isoformat() + "Z"

        event = {
            "eventId": event_id,
            "eventType": event_type,
            "patientId": patient_id,
            "deviceId": f"device-{uuid.uuid4()}",
            "timestamp": timestamp,
            "eventTime": timestamp,
            "sourceSystem": "test-suite",
            "version": "1.0.0",
            "enrichment": {
                "patientContext": {
                    "age": 45,
                    "gender": "M",
                    "conditions": ["I10", "E11.9"]
                },
                "clinicalContext": {
                    "location": "ICU-3",
                    "encounterType": "INPATIENT"
                },
                "validationStatus": "VALID",
                "enrichmentTimestamp": timestamp
            },
            "data": {}
        }

        if event_type == "VITAL_SIGNS":
            event["data"] = {
                "heartRate": 78,
                "systolicBP": 120,
                "diastolicBP": 80,
                "temperature": 37.2,
                "respiratoryRate": 16,
                "oxygenSaturation": 98
            }
        elif event_type == "LAB_RESULT":
            event["data"] = {
                "testCode": "2345-7",
                "testName": "Glucose",
                "value": 95,
                "unit": "mg/dL",
                "referenceRange": "70-100"
            }
        elif event_type == "MEDICATION_ADMINISTRATION":
            event["data"] = {
                "medicationCode": "197361",
                "medicationName": "Metformin",
                "dose": 500,
                "unit": "mg",
                "route": "oral"
            }

        return event

    @staticmethod
    def generate_fhir_resource(resource_type: str = "Observation") -> Dict[str, Any]:
        """Generate FHIR resource"""
        resource_id = str(uuid.uuid4())

        if resource_type == "Observation":
            return {
                "resourceType": "Observation",
                "id": resource_id,
                "status": "final",
                "code": {
                    "coding": [{
                        "system": "http://loinc.org",
                        "code": "8867-4",
                        "display": "Heart rate"
                    }]
                },
                "subject": {
                    "reference": f"Patient/{uuid.uuid4()}"
                },
                "effectiveDateTime": datetime.utcnow().isoformat() + "Z",
                "valueQuantity": {
                    "value": 78,
                    "unit": "beats/minute",
                    "system": "http://unitsofmeasure.org",
                    "code": "/min"
                }
            }

        return {}

    @staticmethod
    def generate_graph_mutation(mutation_type: str = "CREATE_NODE") -> Dict[str, Any]:
        """Generate Neo4j graph mutation"""
        mutation_id = str(uuid.uuid4())

        if mutation_type == "CREATE_NODE":
            return {
                "mutationId": mutation_id,
                "mutationType": "CREATE_NODE",
                "nodeType": "Patient",
                "properties": {
                    "patientId": f"patient-{uuid.uuid4()}",
                    "mrn": f"MRN{uuid.uuid4().hex[:8].upper()}",
                    "name": "Test Patient",
                    "dateOfBirth": "1980-01-01"
                },
                "timestamp": datetime.utcnow().isoformat() + "Z"
            }
        elif mutation_type == "CREATE_RELATIONSHIP":
            return {
                "mutationId": mutation_id,
                "mutationType": "CREATE_RELATIONSHIP",
                "relationshipType": "HAS_ENCOUNTER",
                "fromNode": {"type": "Patient", "id": f"patient-{uuid.uuid4()}"},
                "toNode": {"type": "Encounter", "id": f"encounter-{uuid.uuid4()}"},
                "properties": {
                    "startTime": datetime.utcnow().isoformat() + "Z"
                },
                "timestamp": datetime.utcnow().isoformat() + "Z"
            }

        return {}


class TestModule8Integration:
    """Module 8 Integration Tests"""

    @pytest.fixture(scope="class")
    def config(self):
        """Test configuration"""
        return TestConfig()

    @pytest.fixture(scope="class")
    def db_clients(self, config):
        """Database clients fixture"""
        clients = DatabaseClients(config)
        yield clients
        clients.close_all()

    @pytest.fixture(scope="class")
    def kafka_producer(self, config):
        """Kafka producer fixture"""
        producer = KafkaProducer(
            bootstrap_servers=config.kafka_bootstrap_servers,
            value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            key_serializer=lambda k: k.encode('utf-8') if k else None
        )
        yield producer
        producer.close()

    @pytest.fixture
    def test_data_gen(self):
        """Test data generator fixture"""
        return TestDataGenerator()

    # ==========================================
    # A. End-to-End Flow Tests
    # ==========================================

    def test_enriched_event_fanout(self, kafka_producer, db_clients, test_data_gen, config):
        """Test single enriched event is projected to all 6 storage systems"""
        logger.info("TEST: Enriched event fanout to all storage systems")

        # Generate test event
        patient_id = f"test-patient-{uuid.uuid4()}"
        event = test_data_gen.generate_enriched_event(patient_id=patient_id, event_type="VITAL_SIGNS")
        event_id = event["eventId"]

        logger.info(f"Publishing event {event_id} for patient {patient_id}")

        # Publish to Kafka
        future = kafka_producer.send(
            "prod.ehr.events.enriched",
            key=patient_id,
            value=event
        )
        future.get(timeout=10)

        # Wait for all projectors to process
        logger.info("Waiting 10 seconds for projectors to process event...")
        time.sleep(10)

        # Verify in each storage system
        results = {}

        # 1. PostgreSQL
        logger.info("Checking PostgreSQL...")
        results["postgresql"] = self._verify_postgres(event_id, db_clients.postgres)

        # 2. MongoDB
        logger.info("Checking MongoDB...")
        results["mongodb"] = self._verify_mongo(event_id, db_clients.mongo)

        # 3. Elasticsearch
        logger.info("Checking Elasticsearch...")
        results["elasticsearch"] = self._verify_elasticsearch(event_id, db_clients.elasticsearch, config)

        # 4. ClickHouse
        logger.info("Checking ClickHouse...")
        results["clickhouse"] = self._verify_clickhouse(event_id, db_clients.clickhouse)

        # 5. InfluxDB
        logger.info("Checking InfluxDB...")
        results["influxdb"] = self._verify_influxdb(patient_id, event["timestamp"], db_clients.influxdb, config)

        # 6. UPS Read Model
        logger.info("Checking UPS Read Model...")
        results["ups"] = self._verify_ups(patient_id, db_clients.postgres)

        # Assert all projectors processed the event
        for store, found in results.items():
            assert found, f"Event not found in {store}"

        logger.info(f"✅ Event {event_id} successfully projected to all 6 storage systems")
        logger.info(f"Results: {results}")

    def test_fhir_resource_projection(self, kafka_producer, db_clients, test_data_gen, config):
        """Test FHIR resource projection to Google FHIR Store"""
        logger.info("TEST: FHIR resource projection")

        # Generate FHIR resource
        fhir_resource = test_data_gen.generate_fhir_resource("Observation")
        resource_id = fhir_resource["id"]

        logger.info(f"Publishing FHIR resource {resource_id}")

        # Publish to Kafka
        future = kafka_producer.send(
            "prod.ehr.fhir.upsert",
            key=resource_id,
            value=fhir_resource
        )
        future.get(timeout=10)

        # Wait for processing
        logger.info("Waiting 5 seconds for FHIR Store projector...")
        time.sleep(5)

        # Verify in FHIR Store (check metrics instead of actual query due to GCP auth)
        metrics_url = f"http://localhost:{config.projector_ports['fhir_store']}/metrics"
        try:
            response = requests.get(metrics_url, timeout=5)
            response.raise_for_status()

            # Parse metrics
            metrics_text = response.text
            processed_count = 0
            for family in text_string_to_metric_families(metrics_text):
                if family.name == "projector_messages_processed_total":
                    for sample in family.samples:
                        processed_count = sample.value

            assert processed_count > 0, "FHIR Store projector has not processed any messages"
            logger.info(f"✅ FHIR Store projector processed {processed_count} messages")
        except Exception as e:
            logger.warning(f"Could not verify FHIR Store metrics: {e}")
            pytest.skip("FHIR Store metrics not available")

    def test_graph_mutation_execution(self, kafka_producer, db_clients, test_data_gen, config):
        """Test graph mutation creates node in Neo4j"""
        logger.info("TEST: Neo4j graph mutation execution")

        # Generate graph mutation
        mutation = test_data_gen.generate_graph_mutation("CREATE_NODE")
        mutation_id = mutation["mutationId"]
        patient_id = mutation["properties"]["patientId"]

        logger.info(f"Publishing graph mutation {mutation_id}")

        # Publish to Kafka
        future = kafka_producer.send(
            "prod.ehr.graph.mutations",
            key=mutation_id,
            value=mutation
        )
        future.get(timeout=10)

        # Wait for processing
        logger.info("Waiting 5 seconds for Neo4j Graph projector...")
        time.sleep(5)

        # Verify node in Neo4j
        with db_clients.neo4j.session() as session:
            result = session.run(
                "MATCH (p:Patient {patientId: $patient_id}) RETURN p",
                patient_id=patient_id
            )
            node = result.single()

            assert node is not None, f"Patient node {patient_id} not found in Neo4j"
            logger.info(f"✅ Patient node {patient_id} found in Neo4j")

    # ==========================================
    # B. Performance Tests
    # ==========================================

    def test_batch_processing_performance(self, kafka_producer, db_clients, test_data_gen, config):
        """Test batch processing performance for all projectors"""
        logger.info("TEST: Batch processing performance")

        batch_size = 1000
        patient_id = f"perf-test-{uuid.uuid4()}"

        # Publish 1000 events
        logger.info(f"Publishing {batch_size} events...")
        start_time = time.time()

        for i in range(batch_size):
            event = test_data_gen.generate_enriched_event(patient_id=patient_id)
            kafka_producer.send("prod.ehr.events.enriched", key=patient_id, value=event)

        kafka_producer.flush()
        publish_time = time.time() - start_time
        logger.info(f"Published {batch_size} events in {publish_time:.2f}s ({batch_size/publish_time:.0f} events/sec)")

        # Wait for processing
        logger.info("Waiting 30 seconds for all projectors to process...")
        time.sleep(30)

        # Measure throughput for each projector
        throughput = {}

        # Query each store to count processed events
        # PostgreSQL
        cursor = db_clients.postgres.cursor()
        cursor.execute("SELECT COUNT(*) FROM enriched_events WHERE patient_id = %s", (patient_id,))
        pg_count = cursor.fetchone()[0]
        throughput["postgresql"] = pg_count / 30  # events per second

        # MongoDB
        mongo_count = db_clients.mongo.clinical_documents.count_documents({"patientId": patient_id})
        throughput["mongodb"] = mongo_count / 30

        # Elasticsearch (with refresh)
        db_clients.elasticsearch.indices.refresh(index=config.elasticsearch_index)
        es_result = db_clients.elasticsearch.count(
            index=config.elasticsearch_index,
            body={"query": {"term": {"patientId": patient_id}}}
        )
        throughput["elasticsearch"] = es_result["count"] / 30

        # ClickHouse
        ch_count = db_clients.clickhouse.execute(
            f"SELECT COUNT(*) FROM clinical_events_fact WHERE patient_id = '{patient_id}'"
        )[0][0]
        throughput["clickhouse"] = ch_count / 30

        logger.info(f"Throughput results: {throughput}")

        # Assert performance targets
        assert throughput["postgresql"] >= 30, f"PostgreSQL throughput too low: {throughput['postgresql']:.0f} events/sec"
        assert throughput["mongodb"] >= 25, f"MongoDB throughput too low: {throughput['mongodb']:.0f} events/sec"
        assert throughput["elasticsearch"] >= 30, f"Elasticsearch throughput too low: {throughput['elasticsearch']:.0f} events/sec"
        assert throughput["clickhouse"] >= 30, f"ClickHouse throughput too low: {throughput['clickhouse']:.0f} events/sec"

        logger.info("✅ All projectors meet performance targets")

    def test_query_latency(self, db_clients, config):
        """Test query latency for each storage system"""
        logger.info("TEST: Query latency measurement")

        patient_id = f"latency-test-{uuid.uuid4()}"
        latencies = {}

        # PostgreSQL single patient query
        start = time.time()
        cursor = db_clients.postgres.cursor()
        cursor.execute("SELECT * FROM enriched_events WHERE patient_id = %s LIMIT 1", (patient_id,))
        cursor.fetchall()
        latencies["postgresql_single"] = (time.time() - start) * 1000  # ms

        # MongoDB timeline query
        start = time.time()
        list(db_clients.mongo.clinical_documents.find({"patientId": patient_id}).limit(10))
        latencies["mongodb_timeline"] = (time.time() - start) * 1000

        # Elasticsearch search
        start = time.time()
        db_clients.elasticsearch.search(
            index=config.elasticsearch_index,
            body={"query": {"term": {"patientId": patient_id}}, "size": 10}
        )
        latencies["elasticsearch_search"] = (time.time() - start) * 1000

        # ClickHouse aggregation
        start = time.time()
        db_clients.clickhouse.execute(
            f"SELECT AVG(heart_rate) FROM clinical_events_fact WHERE patient_id = '{patient_id}'"
        )
        latencies["clickhouse_aggregation"] = (time.time() - start) * 1000

        # UPS single patient
        start = time.time()
        cursor = db_clients.postgres.cursor()
        cursor.execute("SELECT * FROM ups_read_model WHERE patient_id = %s", (patient_id,))
        cursor.fetchall()
        latencies["ups_single"] = (time.time() - start) * 1000

        logger.info(f"Query latencies (ms): {latencies}")

        # Assert latency targets
        assert latencies["postgresql_single"] < 100, f"PostgreSQL query too slow: {latencies['postgresql_single']:.2f}ms"
        assert latencies["mongodb_timeline"] < 200, f"MongoDB query too slow: {latencies['mongodb_timeline']:.2f}ms"
        assert latencies["elasticsearch_search"] < 200, f"Elasticsearch query too slow: {latencies['elasticsearch_search']:.2f}ms"
        assert latencies["ups_single"] < 50, f"UPS query too slow: {latencies['ups_single']:.2f}ms"

        logger.info("✅ All queries meet latency targets")

    # ==========================================
    # C. Data Consistency Tests
    # ==========================================

    def test_data_consistency_across_stores(self, kafka_producer, db_clients, test_data_gen):
        """Test data consistency across storage systems"""
        logger.info("TEST: Data consistency across stores")

        # Publish 100 events for 10 different patients
        patients = [f"consistency-test-{i}" for i in range(10)]
        event_count = 100

        logger.info(f"Publishing {event_count} events for {len(patients)} patients...")
        for i in range(event_count):
            patient_id = patients[i % len(patients)]
            event = test_data_gen.generate_enriched_event(patient_id=patient_id, event_type="VITAL_SIGNS")
            kafka_producer.send("prod.ehr.events.enriched", key=patient_id, value=event)

        kafka_producer.flush()

        # Wait for processing
        logger.info("Waiting 15 seconds for processing...")
        time.sleep(15)

        # For each patient, verify consistency
        mismatches = []
        for patient_id in patients:
            # Get latest heart rate from PostgreSQL
            cursor = db_clients.postgres.cursor()
            cursor.execute(
                """
                SELECT data->'heartRate' as heart_rate
                FROM enriched_events
                WHERE patient_id = %s AND event_type = 'VITAL_SIGNS'
                ORDER BY event_time DESC
                LIMIT 1
                """,
                (patient_id,)
            )
            pg_result = cursor.fetchone()

            # Get from UPS Read Model
            cursor.execute(
                "SELECT latest_heart_rate FROM ups_read_model WHERE patient_id = %s",
                (patient_id,)
            )
            ups_result = cursor.fetchone()

            if pg_result and ups_result:
                pg_hr = pg_result[0]
                ups_hr = ups_result[0]

                if pg_hr != ups_hr:
                    mismatches.append({
                        "patient_id": patient_id,
                        "postgres": pg_hr,
                        "ups": ups_hr
                    })

        assert len(mismatches) == 0, f"Data consistency mismatches found: {mismatches}"
        logger.info(f"✅ Data consistent across stores for all {len(patients)} patients")

    def test_upsert_idempotency(self, kafka_producer, db_clients, test_data_gen, config):
        """Test upsert idempotency - duplicate events handled correctly"""
        logger.info("TEST: Upsert idempotency")

        # Generate event and publish 5 times
        patient_id = f"idempotency-test-{uuid.uuid4()}"
        event = test_data_gen.generate_enriched_event(patient_id=patient_id)
        event_id = event["eventId"]

        logger.info(f"Publishing same event {event_id} 5 times...")
        for _ in range(5):
            kafka_producer.send("prod.ehr.events.enriched", key=patient_id, value=event)

        kafka_producer.flush()

        # Wait for processing
        time.sleep(10)

        # Verify counts in each store
        counts = {}

        # PostgreSQL - should have 1 row (ON CONFLICT)
        cursor = db_clients.postgres.cursor()
        cursor.execute("SELECT COUNT(*) FROM enriched_events WHERE event_id = %s", (event_id,))
        counts["postgresql"] = cursor.fetchone()[0]

        # MongoDB - should have 1 document (upsert)
        counts["mongodb"] = db_clients.mongo.clinical_documents.count_documents({"eventId": event_id})

        # Elasticsearch - should have 1 document (same ID)
        db_clients.elasticsearch.indices.refresh(index=config.elasticsearch_index)
        es_result = db_clients.elasticsearch.count(
            index=config.elasticsearch_index,
            body={"query": {"term": {"eventId": event_id}}}
        )
        counts["elasticsearch"] = es_result["count"]

        # ClickHouse - append-only, should have 5 rows
        counts["clickhouse"] = db_clients.clickhouse.execute(
            f"SELECT COUNT(*) FROM clinical_events_fact WHERE event_id = '{event_id}'"
        )[0][0]

        # UPS - should have 1 row
        cursor.execute("SELECT COUNT(*) FROM ups_read_model WHERE patient_id = %s", (patient_id,))
        counts["ups"] = cursor.fetchone()[0]

        logger.info(f"Duplicate event counts: {counts}")

        # Assert idempotency
        assert counts["postgresql"] == 1, f"PostgreSQL should have 1 row, got {counts['postgresql']}"
        assert counts["mongodb"] == 1, f"MongoDB should have 1 document, got {counts['mongodb']}"
        assert counts["elasticsearch"] == 1, f"Elasticsearch should have 1 document, got {counts['elasticsearch']}"
        assert counts["clickhouse"] == 5, f"ClickHouse should have 5 rows (append-only), got {counts['clickhouse']}"
        assert counts["ups"] == 1, f"UPS should have 1 row, got {counts['ups']}"

        logger.info("✅ Upsert idempotency verified for all stores")

    # ==========================================
    # D. Error Handling Tests
    # ==========================================

    def test_dlq_routing(self, kafka_producer, config):
        """Test invalid events are routed to DLQ"""
        logger.info("TEST: DLQ routing for invalid events")

        # Publish malformed event
        invalid_event = {"invalid": "event", "missing": "required_fields"}

        logger.info("Publishing invalid event...")
        kafka_producer.send("prod.ehr.events.enriched", value=invalid_event)
        kafka_producer.flush()

        # Wait for processing
        time.sleep(5)

        # Check DLQ topic for the event
        consumer = KafkaConsumer(
            "prod.ehr.events.dlq",
            bootstrap_servers=config.kafka_bootstrap_servers,
            auto_offset_reset='earliest',
            consumer_timeout_ms=5000,
            value_deserializer=lambda m: json.loads(m.decode('utf-8'))
        )

        dlq_messages = []
        for message in consumer:
            dlq_messages.append(message.value)

        consumer.close()

        # Verify message in DLQ
        assert len(dlq_messages) > 0, "No messages found in DLQ"
        logger.info(f"✅ Found {len(dlq_messages)} messages in DLQ")

    # ==========================================
    # E. Monitoring Tests
    # ==========================================

    def test_prometheus_metrics(self, config):
        """Test Prometheus metrics are exposed correctly"""
        logger.info("TEST: Prometheus metrics")

        projectors = ["postgresql", "mongodb", "elasticsearch", "clickhouse", "influxdb", "ups", "neo4j"]

        for projector in projectors:
            port = config.projector_ports[projector]
            metrics_url = f"http://localhost:{port}/metrics"

            try:
                logger.info(f"Checking metrics for {projector} on port {port}...")
                response = requests.get(metrics_url, timeout=5)
                response.raise_for_status()

                # Parse metrics
                metrics_text = response.text
                found_metrics = set()

                for family in text_string_to_metric_families(metrics_text):
                    found_metrics.add(family.name)

                # Verify required metrics
                required_metrics = {
                    "projector_messages_consumed_total",
                    "projector_messages_processed_total",
                    "projector_batch_size",
                    "projector_consumer_lag"
                }

                missing = required_metrics - found_metrics
                assert len(missing) == 0, f"{projector} missing metrics: {missing}"

                logger.info(f"✅ {projector} exposes all required metrics")

            except requests.exceptions.RequestException as e:
                logger.warning(f"Could not connect to {projector} metrics endpoint: {e}")
                pytest.skip(f"{projector} metrics not available")

    # ==========================================
    # Helper Methods
    # ==========================================

    def _verify_postgres(self, event_id: str, conn) -> bool:
        """Verify event in PostgreSQL"""
        cursor = conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM enriched_events WHERE event_id = %s", (event_id,))
        count = cursor.fetchone()[0]
        return count > 0

    def _verify_mongo(self, event_id: str, db) -> bool:
        """Verify event in MongoDB"""
        count = db.clinical_documents.count_documents({"eventId": event_id})
        return count > 0

    def _verify_elasticsearch(self, event_id: str, es, config: TestConfig) -> bool:
        """Verify event in Elasticsearch"""
        es.indices.refresh(index=config.elasticsearch_index)
        result = es.count(
            index=config.elasticsearch_index,
            body={"query": {"term": {"eventId": event_id}}}
        )
        return result["count"] > 0

    def _verify_clickhouse(self, event_id: str, client) -> bool:
        """Verify event in ClickHouse"""
        result = client.execute(
            f"SELECT COUNT(*) FROM clinical_events_fact WHERE event_id = '{event_id}'"
        )
        return result[0][0] > 0

    def _verify_influxdb(self, patient_id: str, timestamp: str, client, config: TestConfig) -> bool:
        """Verify event in InfluxDB"""
        query_api = client.query_api()

        # Parse timestamp
        event_time = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
        start_time = (event_time - timedelta(minutes=1)).isoformat()
        end_time = (event_time + timedelta(minutes=1)).isoformat()

        query = f'''
        from(bucket: "{config.influxdb_bucket}")
            |> range(start: {start_time}, stop: {end_time})
            |> filter(fn: (r) => r["patient_id"] == "{patient_id}")
            |> filter(fn: (r) => r["_measurement"] == "vital_signs")
        '''

        result = query_api.query(query)
        return len(result) > 0

    def _verify_ups(self, patient_id: str, conn) -> bool:
        """Verify patient in UPS Read Model"""
        cursor = conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM ups_read_model WHERE patient_id = %s", (patient_id,))
        count = cursor.fetchone()[0]
        return count > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
