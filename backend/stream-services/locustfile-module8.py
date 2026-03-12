#!/usr/bin/env python3
"""
Module 8 Load Test with Locust

Simulates realistic clinical event publishing load:
- Multiple concurrent publishers
- All 3 topic types (enriched, FHIR, graph)
- Ramp-up and sustained load patterns
- Measure throughput degradation under load

Usage:
    locust -f locustfile-module8.py --headless -u 100 -r 10 -t 30m
"""

import json
import uuid
import random
import logging
from datetime import datetime
from typing import Dict, Any

from locust import User, task, events, constant_throughput
from kafka import KafkaProducer
from kafka.errors import KafkaError

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class KafkaClient:
    """Kafka client for publishing events"""

    def __init__(self, bootstrap_servers: str = "localhost:9092"):
        self.bootstrap_servers = bootstrap_servers
        self.producer = None
        self._connect()

    def _connect(self):
        """Connect to Kafka"""
        try:
            self.producer = KafkaProducer(
                bootstrap_servers=self.bootstrap_servers,
                value_serializer=lambda v: json.dumps(v).encode('utf-8'),
                key_serializer=lambda k: k.encode('utf-8') if k else None,
                acks=1,  # Leader acknowledgment only for speed
                linger_ms=10,
                batch_size=32768,
                compression_type='snappy'
            )
            logger.info("Connected to Kafka")
        except KafkaError as e:
            logger.error(f"Failed to connect to Kafka: {e}")
            raise

    def publish_event(self, topic: str, key: str, value: Dict[str, Any]) -> bool:
        """Publish event to Kafka topic"""
        try:
            future = self.producer.send(topic, key=key, value=value)
            future.get(timeout=10)
            return True
        except KafkaError as e:
            logger.error(f"Failed to publish to {topic}: {e}")
            return False

    def close(self):
        """Close Kafka producer"""
        if self.producer:
            self.producer.flush()
            self.producer.close()


class EventGenerator:
    """Generate realistic clinical events"""

    def __init__(self):
        self.patient_ids = [f"load-test-patient-{i}" for i in range(1000)]
        self.event_types = [
            "VITAL_SIGNS",
            "LAB_RESULT",
            "MEDICATION_ADMINISTRATION",
            "DIAGNOSTIC_PROCEDURE",
            "CLINICAL_NOTE"
        ]

    def generate_enriched_event(self) -> tuple[str, Dict[str, Any]]:
        """Generate enriched clinical event"""
        patient_id = random.choice(self.patient_ids)
        event_type = random.choice(self.event_types)
        event_id = str(uuid.uuid4())
        timestamp = datetime.utcnow().isoformat() + "Z"

        event = {
            "eventId": event_id,
            "eventType": event_type,
            "patientId": patient_id,
            "deviceId": f"device-{uuid.uuid4()}",
            "timestamp": timestamp,
            "eventTime": timestamp,
            "sourceSystem": "load-test",
            "version": "1.0.0",
            "enrichment": {
                "patientContext": {
                    "age": random.randint(18, 90),
                    "gender": random.choice(["M", "F"]),
                    "conditions": random.sample(["I10", "E11.9", "J44.9", "N18.3"], k=random.randint(0, 3))
                },
                "clinicalContext": {
                    "location": random.choice(["ICU-1", "ICU-2", "Ward-A", "Ward-B", "ER"]),
                    "encounterType": random.choice(["INPATIENT", "OUTPATIENT", "EMERGENCY"])
                },
                "validationStatus": "VALID",
                "enrichmentTimestamp": timestamp
            },
            "data": self._generate_event_data(event_type)
        }

        return patient_id, event

    def _generate_event_data(self, event_type: str) -> Dict[str, Any]:
        """Generate event-specific data"""
        if event_type == "VITAL_SIGNS":
            return {
                "heartRate": random.randint(60, 120),
                "systolicBP": random.randint(90, 160),
                "diastolicBP": random.randint(60, 100),
                "temperature": round(random.uniform(36.0, 39.0), 1),
                "respiratoryRate": random.randint(12, 24),
                "oxygenSaturation": random.randint(90, 100)
            }
        elif event_type == "LAB_RESULT":
            return {
                "testCode": random.choice(["2345-7", "2160-0", "2951-2"]),
                "testName": random.choice(["Glucose", "Creatinine", "Sodium"]),
                "value": random.uniform(50, 150),
                "unit": random.choice(["mg/dL", "mmol/L"]),
                "referenceRange": "70-100"
            }
        elif event_type == "MEDICATION_ADMINISTRATION":
            return {
                "medicationCode": str(random.randint(100000, 999999)),
                "medicationName": random.choice(["Metformin", "Lisinopril", "Atorvastatin"]),
                "dose": random.randint(10, 100),
                "unit": "mg",
                "route": random.choice(["oral", "IV", "subcutaneous"])
            }
        else:
            return {"note": f"Generated {event_type} data"}

    def generate_fhir_resource(self) -> tuple[str, Dict[str, Any]]:
        """Generate FHIR Observation resource"""
        resource_id = str(uuid.uuid4())
        patient_id = random.choice(self.patient_ids)

        resource = {
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
                "reference": f"Patient/{patient_id}"
            },
            "effectiveDateTime": datetime.utcnow().isoformat() + "Z",
            "valueQuantity": {
                "value": random.randint(60, 120),
                "unit": "beats/minute",
                "system": "http://unitsofmeasure.org",
                "code": "/min"
            }
        }

        return resource_id, resource

    def generate_graph_mutation(self) -> tuple[str, Dict[str, Any]]:
        """Generate Neo4j graph mutation"""
        mutation_id = str(uuid.uuid4())
        patient_id = random.choice(self.patient_ids)

        mutation = {
            "mutationId": mutation_id,
            "mutationType": random.choice(["CREATE_NODE", "CREATE_RELATIONSHIP"]),
            "nodeType": "Encounter",
            "properties": {
                "encounterId": f"encounter-{uuid.uuid4()}",
                "patientId": patient_id,
                "encounterType": random.choice(["INPATIENT", "OUTPATIENT", "EMERGENCY"]),
                "startTime": datetime.utcnow().isoformat() + "Z"
            },
            "timestamp": datetime.utcnow().isoformat() + "Z"
        }

        return mutation_id, mutation


class ClinicalEventPublisher(User):
    """Locust user simulating clinical event publisher"""

    wait_time = constant_throughput(1)  # 1 task per second per user

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.kafka_client = KafkaClient()
        self.event_generator = EventGenerator()

    @task(70)
    def publish_enriched_event(self):
        """Publish enriched clinical event (70% of traffic)"""
        patient_id, event = self.event_generator.generate_enriched_event()

        start_time = datetime.utcnow()
        try:
            success = self.kafka_client.publish_event(
                "prod.ehr.events.enriched",
                patient_id,
                event
            )

            elapsed_ms = (datetime.utcnow() - start_time).total_seconds() * 1000

            if success:
                events.request.fire(
                    request_type="kafka",
                    name="publish_enriched_event",
                    response_time=elapsed_ms,
                    response_length=len(json.dumps(event)),
                    exception=None,
                    context={}
                )
            else:
                events.request.fire(
                    request_type="kafka",
                    name="publish_enriched_event",
                    response_time=elapsed_ms,
                    response_length=0,
                    exception=Exception("Publish failed"),
                    context={}
                )
        except Exception as e:
            elapsed_ms = (datetime.utcnow() - start_time).total_seconds() * 1000
            events.request.fire(
                request_type="kafka",
                name="publish_enriched_event",
                response_time=elapsed_ms,
                response_length=0,
                exception=e,
                context={}
            )

    @task(20)
    def publish_fhir_resource(self):
        """Publish FHIR resource (20% of traffic)"""
        resource_id, resource = self.event_generator.generate_fhir_resource()

        start_time = datetime.utcnow()
        try:
            success = self.kafka_client.publish_event(
                "prod.ehr.fhir.upsert",
                resource_id,
                resource
            )

            elapsed_ms = (datetime.utcnow() - start_time).total_seconds() * 1000

            events.request.fire(
                request_type="kafka",
                name="publish_fhir_resource",
                response_time=elapsed_ms,
                response_length=len(json.dumps(resource)),
                exception=None if success else Exception("Publish failed"),
                context={}
            )
        except Exception as e:
            elapsed_ms = (datetime.utcnow() - start_time).total_seconds() * 1000
            events.request.fire(
                request_type="kafka",
                name="publish_fhir_resource",
                response_time=elapsed_ms,
                response_length=0,
                exception=e,
                context={}
            )

    @task(10)
    def publish_graph_mutation(self):
        """Publish graph mutation (10% of traffic)"""
        mutation_id, mutation = self.event_generator.generate_graph_mutation()

        start_time = datetime.utcnow()
        try:
            success = self.kafka_client.publish_event(
                "prod.ehr.graph.mutations",
                mutation_id,
                mutation
            )

            elapsed_ms = (datetime.utcnow() - start_time).total_seconds() * 1000

            events.request.fire(
                request_type="kafka",
                name="publish_graph_mutation",
                response_time=elapsed_ms,
                response_length=len(json.dumps(mutation)),
                exception=None if success else Exception("Publish failed"),
                context={}
            )
        except Exception as e:
            elapsed_ms = (datetime.utcnow() - start_time).total_seconds() * 1000
            events.request.fire(
                request_type="kafka",
                name="publish_graph_mutation",
                response_time=elapsed_ms,
                response_length=0,
                exception=e,
                context={}
            )

    def on_stop(self):
        """Clean up when user stops"""
        self.kafka_client.close()


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Called when test starts"""
    logger.info("=" * 80)
    logger.info("MODULE 8 LOAD TEST STARTING")
    logger.info("=" * 80)
    logger.info(f"Users: {environment.runner.target_user_count}")
    logger.info(f"Spawn rate: {environment.runner.spawn_rate}")
    logger.info(f"Run time: {environment.runner.run_time}")
    logger.info("=" * 80)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Called when test stops"""
    logger.info("=" * 80)
    logger.info("MODULE 8 LOAD TEST COMPLETE")
    logger.info("=" * 80)

    stats = environment.stats

    logger.info("SUMMARY STATISTICS:")
    logger.info(f"  Total requests: {stats.total.num_requests}")
    logger.info(f"  Failed requests: {stats.total.num_failures}")
    logger.info(f"  Failure rate: {stats.total.fail_ratio * 100:.2f}%")
    logger.info(f"  Average response time: {stats.total.avg_response_time:.2f}ms")
    logger.info(f"  p95 response time: {stats.total.get_response_time_percentile(0.95):.2f}ms")
    logger.info(f"  p99 response time: {stats.total.get_response_time_percentile(0.99):.2f}ms")
    logger.info(f"  Requests per second: {stats.total.total_rps:.2f}")
    logger.info("=" * 80)


if __name__ == "__main__":
    import os
    os.system("locust -f locustfile-module8.py")
