#!/usr/bin/env python3
"""Send test patient event to Kafka topic"""
import json
from kafka import KafkaProducer

# Test patient data
test_event = {
    "patientId": "PAT-ROHAN-001",
    "eventTime": 1761633000000,
    "eventType": "CLINICAL_ASSESSMENT",
    "patientState": {
        "patientId": "PAT-ROHAN-001",
        "lastUpdateTime": 1761633000000,
        "activeMedications": {
            "83367": {
                "medicationName": "Telmisartan",
                "startTime": 1760701079000,
                "name": "Telmisartan",
                "code": "83367",
                "dosage": "40mg",
                "route": "oral",
                "frequency": "QD",
                "status": "active"
            }
        },
        "recentVitals": {
            "heart_rate": 115,
            "respiratory_rate": 28,
            "temperature": 39.0,
            "systolic_bp": 95,
            "diastolic_bp": 60,
            "spo2": 92
        },
        "recentLabResults": {
            "lactate": 2.8,
            "wbc": 15.2,
            "creatinine": 1.4
        },
        "activeConditions": [
            "sepsis_suspected",
            "respiratory_distress",
            "hypotension"
        ],
        "news2Score": 8,
        "qsofaScore": 2,
        "sepsisLikelihood": "HIGH"
    },
    "alerts": [
        {
            "severity": "CRITICAL",
            "type": "SEPSIS_ALERT",
            "message": "Sepsis likely: qSOFA=2, lactate=2.8",
            "triggerTime": 1761633000000
        },
        {
            "severity": "HIGH",
            "type": "RESPIRATORY_DISTRESS",
            "message": "SpO2 92%, RR 28/min",
            "triggerTime": 1761633000000
        }
    ],
    "detectedPatterns": [
        {
            "patternType": "VITAL_SIGN_DETERIORATION",
            "confidence": 0.92,
            "details": "NEWS2=8 (HIGH RISK)"
        },
        {
            "patternType": "SEPSIS_INDICATORS",
            "confidence": 0.87,
            "details": "Fever + tachycardia + elevated lactate"
        }
    ]
}

# Create producer
producer = KafkaProducer(
    bootstrap_servers='localhost:9092',
    value_serializer=lambda v: json.dumps(v).encode('utf-8')
)

# Send event
print("Sending test event for patient PAT-ROHAN-001...")
future = producer.send('clinical-patterns.v1', value=test_event)
result = future.get(timeout=10)

print(f"✅ Event sent successfully!")
print(f"   Topic: {result.topic}")
print(f"   Partition: {result.partition}")
print(f"   Offset: {result.offset}")

producer.flush()
producer.close()
