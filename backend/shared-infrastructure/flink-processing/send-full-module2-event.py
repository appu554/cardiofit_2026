#!/usr/bin/env python3
"""Send full Module 2 enriched patient context event"""
import json
from kafka import KafkaProducer

# Full Module 2 output event
test_event = {
    "patientId": "PAT-ROHAN-001",
    "eventType": "VITAL_SIGN",
    "eventTime": 1760171000000,
    "processingTime": 1760786097934,
    "latencyMs": 615097934,
    "patientState": {
        "patientId": "PAT-ROHAN-001",
        "lastUpdated": 1760171000000,
        "lastVitalUpdate": 1760786097934,
        "lastLabUpdate": 1760786097933,
        "eventCount": 38,
        "hasFhirData": True,
        "hasNeo4jData": True,
        "enrichmentComplete": True,
        "latestVitals": {
            "heartrate": 110,
            "respiratoryrate": 28,
            "temperature": 39.0,
            "systolicbp": 110,
            "diastolicbp": 70,
            "oxygensaturation": 92,
            "consciousness": "Alert",
            "supplementaloxygen": False
        },
        "recentLabs": {
            "2524-7": {
                "timestamp": 1760786097679,
                "labCode": "2524-7",
                "labType": "2524-7",
                "value": 2.8,
                "unit": "mmol/L",
                "referenceRangeLow": 0.5,
                "referenceRangeHigh": 2.0,
                "abnormal": True,
                "abnormalFlag": "H"
            }
        },
        "activeMedications": {
            "83367": {
                "medicationName": "Telmisartan",
                "startTime": 1760701079000,
                "name": "Telmisartan",
                "code": "83367",
                "dosage": "40.0 mg",
                "route": "oral",
                "frequency": "daily",
                "status": "active",
                "startDate": 1760701079000,
                "display": "Telmisartan 40 mg Tablet"
            }
        },
        "news2Score": 8,
        "qsofaScore": 1
    }
}

# Create producer
producer = KafkaProducer(
    bootstrap_servers='localhost:9092',
    value_serializer=lambda v: json.dumps(v).encode('utf-8')
)

# Send event
print("Sending full Module 2 event for patient PAT-ROHAN-001...")
future = producer.send('clinical-patterns.v1', value=test_event)
result = future.get(timeout=10)

print(f"✅ Event sent successfully!")
print(f"   Topic: {result.topic}")
print(f"   Partition: {result.partition}")
print(f"   Offset: {result.offset}")

producer.flush()
producer.close()
