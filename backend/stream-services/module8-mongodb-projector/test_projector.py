"""Test script for MongoDB Projector service."""

import json
import time
from datetime import datetime, timezone
from confluent_kafka import Producer
from pymongo import MongoClient

# Test configuration
KAFKA_BOOTSTRAP_SERVERS = "localhost:9092"
KAFKA_TOPIC = "prod.ehr.events.enriched"
MONGODB_URI = "mongodb://localhost:27017"
MONGODB_DATABASE = "module8_clinical"


def create_test_event(patient_id: str, event_id: str, risk_level: str = "NORMAL") -> dict:
    """Create a test enriched event."""
    return {
        "eventId": event_id,
        "patientId": patient_id,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "eventType": "vital_signs",
        "deviceType": "patient_monitor",
        "vitalSigns": {
            "heartRate": 85,
            "bloodPressureSystolic": 120,
            "bloodPressureDiastolic": 80,
            "temperature": 37.2,
            "oxygenSaturation": 98,
            "respiratoryRate": 16,
        },
        "enrichments": {
            "riskLevel": risk_level,
            "earlyWarningScore": 2,
            "clinicalContext": {
                "setting": "ICU",
                "admissionDate": "2024-01-01T00:00:00Z",
            },
            "deviceContext": {
                "manufacturer": "Philips",
                "model": "IntelliVue MX800",
            },
        },
        "mlPredictions": {
            "predictions": {
                "sepsis_risk_24h": {
                    "modelName": "sepsis_xgboost_v1",
                    "prediction": 0.35,
                    "confidence": 0.82,
                    "threshold": 0.5,
                    "alertTriggered": False,
                    "shapValues": {
                        "heart_rate": 0.05,
                        "temperature": 0.12,
                        "wbc_count": 0.08,
                    },
                    "limeExplanation": {
                        "features": ["heart_rate", "temperature"],
                        "weights": [0.05, 0.12],
                    },
                },
                "mortality_risk_48h": {
                    "modelName": "mortality_lgbm_v1",
                    "prediction": 0.15,
                    "confidence": 0.88,
                    "threshold": 0.3,
                    "alertTriggered": False,
                },
            },
            "featureImportance": {
                "heart_rate": 0.25,
                "temperature": 0.18,
                "wbc_count": 0.15,
            },
        },
        "ingestionTime": datetime.now(timezone.utc).isoformat(),
        "processingTime": datetime.now(timezone.utc).isoformat(),
    }


def produce_test_events(num_events: int = 10, num_patients: int = 3):
    """Produce test events to Kafka."""
    print(f"\nProducing {num_events} test events for {num_patients} patients...")

    producer_config = {
        "bootstrap.servers": KAFKA_BOOTSTRAP_SERVERS,
        "client.id": "mongodb-projector-test",
    }

    producer = Producer(producer_config)

    events_produced = 0
    for i in range(num_events):
        patient_id = f"test_patient_{i % num_patients + 1}"
        event_id = f"test_event_{int(time.time())}_{i}"

        # Vary risk levels
        risk_levels = ["NORMAL", "ELEVATED", "HIGH", "CRITICAL"]
        risk_level = risk_levels[i % len(risk_levels)]

        event = create_test_event(patient_id, event_id, risk_level)

        # Produce to Kafka
        producer.produce(
            KAFKA_TOPIC,
            key=patient_id.encode("utf-8"),
            value=json.dumps(event).encode("utf-8"),
            callback=lambda err, msg: print(f"  Produced: {msg.key().decode('utf-8')}") if not err else print(f"  Error: {err}"),
        )

        events_produced += 1

    # Flush all messages
    producer.flush()
    print(f"\n✓ Produced {events_produced} events to {KAFKA_TOPIC}")


def verify_mongodb_data():
    """Verify data in MongoDB collections."""
    print("\nVerifying MongoDB data...")

    client = MongoClient(MONGODB_URI)
    db = client[MONGODB_DATABASE]

    # Check collections
    collections = db.list_collection_names()
    print(f"  Collections: {collections}")

    # Clinical documents
    clinical_docs_count = db.clinical_documents.count_documents({})
    print(f"\n  Clinical Documents: {clinical_docs_count}")

    if clinical_docs_count > 0:
        # Show sample document
        sample = db.clinical_documents.find_one()
        print(f"  Sample Document ID: {sample['_id']}")
        print(f"  Patient ID: {sample['patientId']}")
        print(f"  Summary: {sample.get('summary', 'N/A')}")

        # Show indexes
        indexes = db.clinical_documents.list_indexes()
        print(f"  Indexes: {[idx['name'] for idx in indexes]}")

    # Patient timelines
    timelines_count = db.patient_timelines.count_documents({})
    print(f"\n  Patient Timelines: {timelines_count}")

    if timelines_count > 0:
        # Show sample timeline
        sample = db.patient_timelines.find_one()
        print(f"  Sample Patient ID: {sample['_id']}")
        print(f"  Event Count: {sample.get('eventCount', 0)}")
        print(f"  Timeline Events: {len(sample.get('events', []))}")

        # Show indexes
        indexes = db.patient_timelines.list_indexes()
        print(f"  Indexes: {[idx['name'] for idx in indexes]}")

    # ML explanations
    explanations_count = db.ml_explanations.count_documents({})
    print(f"\n  ML Explanations: {explanations_count}")

    if explanations_count > 0:
        # Show sample explanation
        sample = db.ml_explanations.find_one()
        print(f"  Sample Patient ID: {sample['patientId']}")
        print(f"  Predictions: {list(sample.get('predictions', {}).keys())}")

        # Show indexes
        indexes = db.ml_explanations.list_indexes()
        print(f"  Indexes: {[idx['name'] for idx in indexes]}")

    # Risk level distribution
    print("\n  Risk Level Distribution:")
    pipeline = [
        {"$group": {"_id": "$enrichments.riskLevel", "count": {"$sum": 1}}},
        {"$sort": {"count": -1}},
    ]
    for result in db.clinical_documents.aggregate(pipeline):
        print(f"    {result['_id']}: {result['count']}")

    client.close()
    print("\n✓ MongoDB verification complete")


def main():
    """Main test function."""
    print("=" * 60)
    print("MongoDB Projector Test Script")
    print("=" * 60)

    # Step 1: Produce test events
    produce_test_events(num_events=20, num_patients=5)

    # Step 2: Wait for processing
    print("\nWaiting 15 seconds for projector to process events...")
    time.sleep(15)

    # Step 3: Verify MongoDB data
    verify_mongodb_data()

    print("\n" + "=" * 60)
    print("Test complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
