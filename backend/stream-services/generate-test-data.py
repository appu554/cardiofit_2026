#!/usr/bin/env python3
"""
Module 8 Test Data Generator

Generate realistic clinical events for testing:
- All event types (VITAL_SIGNS, LAB_RESULT, MEDICATION, etc.)
- Configurable patient count and time range
- Output to Kafka topics or JSON files
- Support for replay scenarios

Usage:
    # Generate to Kafka
    python generate-test-data.py --kafka --patients 100 --events-per-patient 50

    # Generate to JSON files
    python generate-test-data.py --output ./test-data --patients 10 --events-per-patient 20

    # Generate historical data
    python generate-test-data.py --kafka --start-date 2024-01-01 --end-date 2024-12-31
"""

import argparse
import json
import uuid
import random
import logging
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from pathlib import Path

from kafka import KafkaProducer
from kafka.errors import KafkaError

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class ClinicalEventGenerator:
    """Generate realistic clinical events"""

    # Event type distributions
    EVENT_TYPE_WEIGHTS = {
        "VITAL_SIGNS": 40,
        "LAB_RESULT": 25,
        "MEDICATION_ADMINISTRATION": 15,
        "DIAGNOSTIC_PROCEDURE": 10,
        "CLINICAL_NOTE": 5,
        "ALERT": 3,
        "DEVICE_READING": 2
    }

    # Realistic clinical values
    VITAL_RANGES = {
        "heartRate": (60, 120),
        "systolicBP": (90, 160),
        "diastolicBP": (60, 100),
        "temperature": (36.0, 39.0),
        "respiratoryRate": (12, 24),
        "oxygenSaturation": (90, 100)
    }

    LAB_TESTS = [
        {"code": "2345-7", "name": "Glucose", "unit": "mg/dL", "range": (70, 200)},
        {"code": "2160-0", "name": "Creatinine", "unit": "mg/dL", "range": (0.6, 1.5)},
        {"code": "2951-2", "name": "Sodium", "unit": "mmol/L", "range": (135, 145)},
        {"code": "2823-3", "name": "Potassium", "unit": "mmol/L", "range": (3.5, 5.0)},
        {"code": "789-8", "name": "WBC", "unit": "10^3/uL", "range": (4.0, 11.0)},
        {"code": "718-7", "name": "Hemoglobin", "unit": "g/dL", "range": (12.0, 17.0)},
        {"code": "777-3", "name": "Platelet", "unit": "10^3/uL", "range": (150, 400)}
    ]

    MEDICATIONS = [
        {"code": "197361", "name": "Metformin", "dose_range": (500, 2000), "unit": "mg", "route": "oral"},
        {"code": "314076", "name": "Lisinopril", "dose_range": (5, 40), "unit": "mg", "route": "oral"},
        {"code": "617312", "name": "Atorvastatin", "dose_range": (10, 80), "unit": "mg", "route": "oral"},
        {"code": "854228", "name": "Furosemide", "dose_range": (20, 80), "unit": "mg", "route": "oral"},
        {"code": "308136", "name": "Insulin", "dose_range": (5, 50), "unit": "units", "route": "subcutaneous"},
        {"code": "197736", "name": "Warfarin", "dose_range": (2, 10), "unit": "mg", "route": "oral"}
    ]

    DIAGNOSES = [
        "I10",      # Hypertension
        "E11.9",    # Type 2 Diabetes
        "J44.9",    # COPD
        "N18.3",    # CKD Stage 3
        "I50.9",    # Heart Failure
        "I25.10",   # CAD
        "E78.5",    # Hyperlipidemia
        "K21.9"     # GERD
    ]

    LOCATIONS = ["ICU-1", "ICU-2", "ICU-3", "Ward-A", "Ward-B", "Ward-C", "ER-1", "ER-2"]
    ENCOUNTER_TYPES = ["INPATIENT", "OUTPATIENT", "EMERGENCY"]

    def __init__(self, seed: Optional[int] = None):
        if seed:
            random.seed(seed)

    def generate_patient_profile(self, patient_id: str) -> Dict[str, Any]:
        """Generate realistic patient profile"""
        age = random.randint(18, 90)
        gender = random.choice(["M", "F"])

        # Older patients more likely to have conditions
        num_conditions = min(random.randint(0, age // 15), len(self.DIAGNOSES))
        conditions = random.sample(self.DIAGNOSES, k=num_conditions)

        return {
            "patientId": patient_id,
            "age": age,
            "gender": gender,
            "conditions": conditions
        }

    def generate_enriched_event(
        self,
        patient_profile: Dict[str, Any],
        timestamp: datetime,
        event_type: Optional[str] = None
    ) -> Dict[str, Any]:
        """Generate enriched clinical event"""
        if event_type is None:
            event_type = random.choices(
                list(self.EVENT_TYPE_WEIGHTS.keys()),
                weights=list(self.EVENT_TYPE_WEIGHTS.values())
            )[0]

        event_id = str(uuid.uuid4())
        timestamp_iso = timestamp.isoformat() + "Z"

        event = {
            "eventId": event_id,
            "eventType": event_type,
            "patientId": patient_profile["patientId"],
            "deviceId": f"device-{uuid.uuid4()}",
            "timestamp": timestamp_iso,
            "eventTime": timestamp_iso,
            "sourceSystem": "test-data-generator",
            "version": "1.0.0",
            "enrichment": {
                "patientContext": {
                    "age": patient_profile["age"],
                    "gender": patient_profile["gender"],
                    "conditions": patient_profile["conditions"]
                },
                "clinicalContext": {
                    "location": random.choice(self.LOCATIONS),
                    "encounterType": random.choice(self.ENCOUNTER_TYPES)
                },
                "validationStatus": "VALID",
                "enrichmentTimestamp": timestamp_iso
            },
            "data": self._generate_event_data(event_type, patient_profile)
        }

        return event

    def _generate_event_data(self, event_type: str, patient_profile: Dict[str, Any]) -> Dict[str, Any]:
        """Generate event-specific data"""
        if event_type == "VITAL_SIGNS":
            return {
                key: self._generate_vital_value(key, patient_profile)
                for key in self.VITAL_RANGES.keys()
            }

        elif event_type == "LAB_RESULT":
            test = random.choice(self.LAB_TESTS)
            value = round(random.uniform(test["range"][0], test["range"][1]), 2)

            return {
                "testCode": test["code"],
                "testName": test["name"],
                "value": value,
                "unit": test["unit"],
                "referenceRange": f"{test['range'][0]}-{test['range'][1]}",
                "status": "final"
            }

        elif event_type == "MEDICATION_ADMINISTRATION":
            med = random.choice(self.MEDICATIONS)
            dose = random.randint(med["dose_range"][0], med["dose_range"][1])

            return {
                "medicationCode": med["code"],
                "medicationName": med["name"],
                "dose": dose,
                "unit": med["unit"],
                "route": med["route"],
                "status": "completed"
            }

        elif event_type == "DIAGNOSTIC_PROCEDURE":
            return {
                "procedureCode": f"PROC-{random.randint(1000, 9999)}",
                "procedureName": random.choice(["X-Ray", "CT Scan", "MRI", "Ultrasound", "ECG"]),
                "status": "completed",
                "findings": "Test findings documented"
            }

        elif event_type == "CLINICAL_NOTE":
            return {
                "noteType": random.choice(["Progress Note", "Consultation", "Discharge Summary"]),
                "author": f"Dr. {random.choice(['Smith', 'Johnson', 'Williams', 'Brown'])}",
                "content": "Clinical documentation"
            }

        elif event_type == "ALERT":
            return {
                "alertType": random.choice(["CRITICAL", "WARNING", "INFO"]),
                "alertCode": f"ALERT-{random.randint(100, 999)}",
                "message": "Clinical alert triggered",
                "priority": random.randint(1, 5)
            }

        return {}

    def _generate_vital_value(self, vital_key: str, patient_profile: Dict[str, Any]) -> float:
        """Generate realistic vital sign value based on patient profile"""
        min_val, max_val = self.VITAL_RANGES[vital_key]

        # Adjust ranges for age and conditions
        if patient_profile["age"] > 65:
            # Older patients tend to have higher BP, lower SpO2
            if vital_key in ["systolicBP", "diastolicBP"]:
                min_val += 10
                max_val += 15
            elif vital_key == "oxygenSaturation":
                min_val -= 3

        # Adjust for conditions
        if "I10" in patient_profile["conditions"]:  # Hypertension
            if vital_key in ["systolicBP", "diastolicBP"]:
                min_val += 15
                max_val += 20

        # Generate value
        if vital_key == "temperature":
            value = round(random.uniform(min_val, max_val), 1)
        else:
            value = random.randint(int(min_val), int(max_val))

        return value


class TestDataGenerator:
    """Main test data generator"""

    def __init__(
        self,
        output_mode: str = "kafka",
        kafka_bootstrap: str = "localhost:9092",
        output_dir: Optional[Path] = None
    ):
        self.output_mode = output_mode
        self.output_dir = output_dir
        self.event_generator = ClinicalEventGenerator()
        self.kafka_producer = None

        if output_mode == "kafka":
            self._init_kafka(kafka_bootstrap)
        elif output_mode == "json" and output_dir:
            output_dir.mkdir(parents=True, exist_ok=True)

    def _init_kafka(self, bootstrap_servers: str):
        """Initialize Kafka producer"""
        try:
            self.kafka_producer = KafkaProducer(
                bootstrap_servers=bootstrap_servers,
                value_serializer=lambda v: json.dumps(v).encode('utf-8'),
                key_serializer=lambda k: k.encode('utf-8') if k else None
            )
            logger.info(f"Connected to Kafka at {bootstrap_servers}")
        except KafkaError as e:
            logger.error(f"Failed to connect to Kafka: {e}")
            raise

    def generate_patient_timeline(
        self,
        patient_count: int,
        events_per_patient: int,
        start_date: datetime,
        end_date: datetime
    ):
        """Generate timeline of events for multiple patients"""
        logger.info(f"Generating data for {patient_count} patients, {events_per_patient} events each")
        logger.info(f"Time range: {start_date} to {end_date}")

        total_events = 0

        for i in range(patient_count):
            patient_id = f"test-patient-{i:06d}"
            patient_profile = self.event_generator.generate_patient_profile(patient_id)

            logger.info(f"Generating events for patient {i+1}/{patient_count}: {patient_id}")

            # Generate events distributed over time range
            time_delta = (end_date - start_date) / events_per_patient

            for j in range(events_per_patient):
                event_time = start_date + (time_delta * j)

                # Add random jitter
                jitter = timedelta(minutes=random.randint(-30, 30))
                event_time += jitter

                event = self.event_generator.generate_enriched_event(
                    patient_profile,
                    event_time
                )

                self._output_event(event, patient_id)
                total_events += 1

                if (total_events % 100) == 0:
                    logger.info(f"Generated {total_events} events...")

        logger.info(f"✅ Generated {total_events} total events")

    def _output_event(self, event: Dict[str, Any], patient_id: str):
        """Output event to configured destination"""
        if self.output_mode == "kafka":
            try:
                self.kafka_producer.send(
                    "prod.ehr.events.enriched",
                    key=patient_id,
                    value=event
                )
            except KafkaError as e:
                logger.error(f"Failed to publish event: {e}")

        elif self.output_mode == "json":
            # Write to patient-specific file
            patient_file = self.output_dir / f"{patient_id}.jsonl"
            with patient_file.open('a') as f:
                f.write(json.dumps(event) + "\n")

    def close(self):
        """Clean up resources"""
        if self.kafka_producer:
            self.kafka_producer.flush()
            self.kafka_producer.close()
            logger.info("Closed Kafka producer")


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description="Generate test clinical events")

    # Output mode
    parser.add_argument(
        "--kafka",
        action="store_true",
        help="Output to Kafka topics"
    )
    parser.add_argument(
        "--output",
        type=Path,
        help="Output directory for JSON files"
    )
    parser.add_argument(
        "--kafka-bootstrap",
        default="localhost:9092",
        help="Kafka bootstrap servers"
    )

    # Data generation parameters
    parser.add_argument(
        "--patients",
        type=int,
        default=10,
        help="Number of patients"
    )
    parser.add_argument(
        "--events-per-patient",
        type=int,
        default=20,
        help="Events per patient"
    )
    parser.add_argument(
        "--start-date",
        type=str,
        default=(datetime.utcnow() - timedelta(days=7)).strftime("%Y-%m-%d"),
        help="Start date (YYYY-MM-DD)"
    )
    parser.add_argument(
        "--end-date",
        type=str,
        default=datetime.utcnow().strftime("%Y-%m-%d"),
        help="End date (YYYY-MM-DD)"
    )
    parser.add_argument(
        "--seed",
        type=int,
        help="Random seed for reproducibility"
    )

    args = parser.parse_args()

    # Determine output mode
    if args.kafka:
        output_mode = "kafka"
        output_dir = None
    elif args.output:
        output_mode = "json"
        output_dir = args.output
    else:
        logger.error("Must specify either --kafka or --output")
        return

    # Parse dates
    start_date = datetime.strptime(args.start_date, "%Y-%m-%d")
    end_date = datetime.strptime(args.end_date, "%Y-%m-%d")

    # Generate data
    generator = TestDataGenerator(
        output_mode=output_mode,
        kafka_bootstrap=args.kafka_bootstrap,
        output_dir=output_dir
    )

    try:
        generator.generate_patient_timeline(
            patient_count=args.patients,
            events_per_patient=args.events_per_patient,
            start_date=start_date,
            end_date=end_date
        )
    finally:
        generator.close()


if __name__ == "__main__":
    main()
