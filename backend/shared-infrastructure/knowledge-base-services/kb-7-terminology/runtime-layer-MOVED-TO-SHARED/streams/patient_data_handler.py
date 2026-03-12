"""
Patient Data Stream Handler for KB7 Terminology Service
Handles real-time patient data stream into Neo4j patient database
Processes Kafka events for medication prescriptions, diagnoses, and lab results
"""

from aiokafka import AIOKafkaConsumer
import json
import asyncio
from typing import Dict, Any, Optional
from datetime import datetime
from loguru import logger


class PatientDataStreamHandler:
    """Handles real-time patient data stream into Neo4j patient database"""

    def __init__(self, neo4j_manager, kafka_config: Dict[str, Any]):
        """
        Initialize Patient Data Stream Handler

        Args:
            neo4j_manager: Neo4j Dual Stream Manager instance
            kafka_config: Kafka configuration dictionary
        """
        self.neo4j = neo4j_manager
        self.consumer = AIOKafkaConsumer(
            'patient-events',
            'medication-events',
            'diagnosis-events',
            'lab-events',
            'encounter-events',
            bootstrap_servers=kafka_config.get('brokers', ['localhost:9092']),
            group_id='patient-data-handler',
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            auto_offset_reset='latest',
            enable_auto_commit=True
        )

        # Processing statistics
        self.processing_stats = {
            'events_processed': 0,
            'medications_processed': 0,
            'diagnoses_processed': 0,
            'lab_results_processed': 0,
            'encounters_processed': 0,
            'errors': 0,
            'start_time': datetime.utcnow()
        }

        logger.info("Patient Data Stream Handler initialized")

    async def start_processing(self) -> None:
        """Start processing patient events from Kafka"""
        await self.consumer.start()

        try:
            logger.info("Started patient data stream processing")

            async for msg in self.consumer:
                try:
                    event = msg.value
                    event_type = event.get('type', 'unknown')

                    logger.debug(f"Processing patient event: {event_type}")

                    if event_type == 'medication_prescribed':
                        await self._handle_medication_event(event)
                        self.processing_stats['medications_processed'] += 1

                    elif event_type == 'diagnosis_recorded':
                        await self._handle_diagnosis_event(event)
                        self.processing_stats['diagnoses_processed'] += 1

                    elif event_type == 'lab_result':
                        await self._handle_lab_event(event)
                        self.processing_stats['lab_results_processed'] += 1

                    elif event_type == 'encounter_started':
                        await self._handle_encounter_event(event)
                        self.processing_stats['encounters_processed'] += 1

                    elif event_type == 'patient_admitted':
                        await self._handle_patient_admission(event)

                    elif event_type == 'medication_discontinued':
                        await self._handle_medication_discontinuation(event)

                    else:
                        logger.warning(f"Unknown event type: {event_type}")

                    self.processing_stats['events_processed'] += 1

                except Exception as e:
                    logger.error(f"Error processing patient event: {e}")
                    self.processing_stats['errors'] += 1

        except Exception as e:
            logger.error(f"Error in patient data processing loop: {e}")
        finally:
            await self.consumer.stop()

    async def _handle_medication_event(self, event: Dict[str, Any]) -> None:
        """
        Add medication prescription to patient graph

        Args:
            event: Medication prescription event
        """
        data = event.get('data', {})

        async with self.neo4j.driver.session(database="patient_data") as session:
            cypher = """
            MERGE (p:Patient {id: $patient_id})
            SET p.last_updated = datetime()

            MERGE (m:Medication {
                rxnorm: $rxnorm,
                name: $drug_name,
                generic_name: $generic_name
            })

            CREATE (p)-[r:PRESCRIBED {
                prescription_id: $prescription_id,
                start_date: datetime($start_date),
                end_date: CASE WHEN $end_date IS NOT NULL THEN datetime($end_date) ELSE null END,
                dose: $dose,
                dose_unit: $dose_unit,
                frequency: $frequency,
                route: $route,
                duration_days: $duration_days,
                quantity: $quantity,
                refills: $refills,
                prescriber_id: $prescriber_id,
                encounter_id: $encounter_id,
                status: $status,
                created_at: datetime()
            }]->(m)

            // Link to encounter if provided
            WITH p, r
            OPTIONAL MATCH (e:Encounter {id: $encounter_id})
            FOREACH (encounter IN CASE WHEN e IS NOT NULL THEN [e] ELSE [] END |
                CREATE (r)-[:PRESCRIBED_DURING]->(encounter)
            )
            """

            await session.run(cypher, {
                'patient_id': data.get('patient_id'),
                'prescription_id': data.get('prescription_id'),
                'rxnorm': data.get('rxnorm'),
                'drug_name': data.get('drug_name'),
                'generic_name': data.get('generic_name'),
                'start_date': data.get('start_date'),
                'end_date': data.get('end_date'),
                'dose': data.get('dose'),
                'dose_unit': data.get('dose_unit'),
                'frequency': data.get('frequency'),
                'route': data.get('route'),
                'duration_days': data.get('duration_days'),
                'quantity': data.get('quantity'),
                'refills': data.get('refills'),
                'prescriber_id': data.get('prescriber_id'),
                'encounter_id': data.get('encounter_id'),
                'status': data.get('status', 'active')
            })

        logger.debug(f"Processed medication prescription for patient {data.get('patient_id')}")

    async def _handle_diagnosis_event(self, event: Dict[str, Any]) -> None:
        """
        Add diagnosis to patient graph

        Args:
            event: Diagnosis event
        """
        data = event.get('data', {})

        async with self.neo4j.driver.session(database="patient_data") as session:
            cypher = """
            MERGE (p:Patient {id: $patient_id})
            SET p.last_updated = datetime()

            MERGE (c:Condition {
                code: $code,
                system: $system,
                name: $condition_name
            })

            CREATE (p)-[r:HAS_CONDITION {
                diagnosis_id: $diagnosis_id,
                onset_date: CASE WHEN $onset_date IS NOT NULL THEN datetime($onset_date) ELSE null END,
                resolved_date: CASE WHEN $resolved_date IS NOT NULL THEN datetime($resolved_date) ELSE null END,
                severity: $severity,
                status: $status,
                encounter_id: $encounter_id,
                diagnosed_by: $diagnosed_by,
                primary_diagnosis: $primary_diagnosis,
                created_at: datetime()
            }]->(c)

            // Link to encounter if provided
            WITH p, r
            OPTIONAL MATCH (e:Encounter {id: $encounter_id})
            FOREACH (encounter IN CASE WHEN e IS NOT NULL THEN [e] ELSE [] END |
                CREATE (r)-[:DIAGNOSED_DURING]->(encounter)
            )
            """

            await session.run(cypher, {
                'patient_id': data.get('patient_id'),
                'diagnosis_id': data.get('diagnosis_id'),
                'code': data.get('code'),
                'system': data.get('system', 'ICD10'),
                'condition_name': data.get('condition_name'),
                'onset_date': data.get('onset_date'),
                'resolved_date': data.get('resolved_date'),
                'severity': data.get('severity'),
                'status': data.get('status', 'active'),
                'encounter_id': data.get('encounter_id'),
                'diagnosed_by': data.get('diagnosed_by'),
                'primary_diagnosis': data.get('primary_diagnosis', False)
            })

        logger.debug(f"Processed diagnosis for patient {data.get('patient_id')}")

    async def _handle_lab_event(self, event: Dict[str, Any]) -> None:
        """
        Add lab result to patient graph

        Args:
            event: Lab result event
        """
        data = event.get('data', {})

        async with self.neo4j.driver.session(database="patient_data") as session:
            cypher = """
            MERGE (p:Patient {id: $patient_id})
            SET p.last_updated = datetime()

            MERGE (o:Observation {
                loinc: $loinc,
                name: $observation_name,
                category: $category
            })

            CREATE (p)-[r:HAS_OBSERVATION {
                result_id: $result_id,
                value: $value,
                value_unit: $value_unit,
                reference_range: $reference_range,
                status: $status,
                abnormal_flag: $abnormal_flag,
                collected_date: datetime($collected_date),
                reported_date: CASE WHEN $reported_date IS NOT NULL THEN datetime($reported_date) ELSE null END,
                encounter_id: $encounter_id,
                ordered_by: $ordered_by,
                performed_by: $performed_by,
                created_at: datetime()
            }]->(o)

            // Link to encounter if provided
            WITH p, r
            OPTIONAL MATCH (e:Encounter {id: $encounter_id})
            FOREACH (encounter IN CASE WHEN e IS NOT NULL THEN [e] ELSE [] END |
                CREATE (r)-[:COLLECTED_DURING]->(encounter)
            )
            """

            await session.run(cypher, {
                'patient_id': data.get('patient_id'),
                'result_id': data.get('result_id'),
                'loinc': data.get('loinc'),
                'observation_name': data.get('observation_name'),
                'category': data.get('category', 'laboratory'),
                'value': data.get('value'),
                'value_unit': data.get('value_unit'),
                'reference_range': data.get('reference_range'),
                'status': data.get('status', 'final'),
                'abnormal_flag': data.get('abnormal_flag'),
                'collected_date': data.get('collected_date'),
                'reported_date': data.get('reported_date'),
                'encounter_id': data.get('encounter_id'),
                'ordered_by': data.get('ordered_by'),
                'performed_by': data.get('performed_by')
            })

        logger.debug(f"Processed lab result for patient {data.get('patient_id')}")

    async def _handle_encounter_event(self, event: Dict[str, Any]) -> None:
        """
        Add encounter to patient graph

        Args:
            event: Encounter event
        """
        data = event.get('data', {})

        async with self.neo4j.driver.session(database="patient_data") as session:
            cypher = """
            MERGE (p:Patient {id: $patient_id})
            SET p.last_updated = datetime()

            CREATE (e:Encounter {
                id: $encounter_id,
                class: $encounter_class,
                type: $encounter_type,
                status: $status,
                start_time: datetime($start_time),
                end_time: CASE WHEN $end_time IS NOT NULL THEN datetime($end_time) ELSE null END,
                location: $location,
                department: $department,
                attending_physician: $attending_physician,
                admission_type: $admission_type,
                discharge_disposition: $discharge_disposition,
                created_at: datetime()
            })

            CREATE (p)-[:HAS_ENCOUNTER]->(e)
            """

            await session.run(cypher, {
                'patient_id': data.get('patient_id'),
                'encounter_id': data.get('encounter_id'),
                'encounter_class': data.get('encounter_class', 'outpatient'),
                'encounter_type': data.get('encounter_type'),
                'status': data.get('status', 'in-progress'),
                'start_time': data.get('start_time'),
                'end_time': data.get('end_time'),
                'location': data.get('location'),
                'department': data.get('department'),
                'attending_physician': data.get('attending_physician'),
                'admission_type': data.get('admission_type'),
                'discharge_disposition': data.get('discharge_disposition')
            })

        logger.debug(f"Processed encounter for patient {data.get('patient_id')}")

    async def _handle_patient_admission(self, event: Dict[str, Any]) -> None:
        """
        Handle patient admission event

        Args:
            event: Patient admission event
        """
        data = event.get('data', {})

        async with self.neo4j.driver.session(database="patient_data") as session:
            cypher = """
            MERGE (p:Patient {id: $patient_id})
            SET p.mrn = $mrn,
                p.first_name = $first_name,
                p.last_name = $last_name,
                p.date_of_birth = date($date_of_birth),
                p.gender = $gender,
                p.phone = $phone,
                p.email = $email,
                p.address = $address,
                p.emergency_contact = $emergency_contact,
                p.insurance = $insurance,
                p.last_updated = datetime()
            """

            await session.run(cypher, {
                'patient_id': data.get('patient_id'),
                'mrn': data.get('mrn'),
                'first_name': data.get('first_name'),
                'last_name': data.get('last_name'),
                'date_of_birth': data.get('date_of_birth'),
                'gender': data.get('gender'),
                'phone': data.get('phone'),
                'email': data.get('email'),
                'address': data.get('address'),
                'emergency_contact': data.get('emergency_contact'),
                'insurance': data.get('insurance')
            })

        logger.debug(f"Processed patient admission for {data.get('patient_id')}")

    async def _handle_medication_discontinuation(self, event: Dict[str, Any]) -> None:
        """
        Handle medication discontinuation

        Args:
            event: Medication discontinuation event
        """
        data = event.get('data', {})

        async with self.neo4j.driver.session(database="patient_data") as session:
            cypher = """
            MATCH (p:Patient {id: $patient_id})-[r:PRESCRIBED]->(m:Medication {rxnorm: $rxnorm})
            WHERE r.prescription_id = $prescription_id OR r.prescription_id IS NULL
            SET r.end_date = datetime($discontinuation_date),
                r.status = 'discontinued',
                r.discontinuation_reason = $reason,
                r.discontinued_by = $discontinued_by
            """

            await session.run(cypher, {
                'patient_id': data.get('patient_id'),
                'rxnorm': data.get('rxnorm'),
                'prescription_id': data.get('prescription_id'),
                'discontinuation_date': data.get('discontinuation_date'),
                'reason': data.get('reason'),
                'discontinued_by': data.get('discontinued_by')
            })

        logger.debug(f"Processed medication discontinuation for patient {data.get('patient_id')}")

    async def get_processing_statistics(self) -> Dict[str, Any]:
        """Get patient data processing statistics"""
        uptime = datetime.utcnow() - self.processing_stats['start_time']

        return {
            'processing_stats': self.processing_stats,
            'uptime_seconds': uptime.total_seconds(),
            'events_per_second': (
                self.processing_stats['events_processed'] / max(uptime.total_seconds(), 1)
            ),
            'error_rate': (
                self.processing_stats['errors'] /
                max(self.processing_stats['events_processed'], 1)
            ),
            'timestamp': datetime.utcnow().isoformat()
        }

    async def stop_processing(self) -> None:
        """Stop patient data stream processing"""
        await self.consumer.stop()
        logger.info("Patient data stream processing stopped")

    async def test_patient_stream(self) -> Dict[str, Any]:
        """Test patient data stream with sample data"""
        logger.info("Testing patient data stream with sample data")

        # Sample test events
        test_events = [
            {
                'type': 'patient_admitted',
                'data': {
                    'patient_id': 'test-patient-001',
                    'mrn': 'TEST-MRN-001',
                    'first_name': 'John',
                    'last_name': 'Doe',
                    'date_of_birth': '1975-06-15',
                    'gender': 'male'
                }
            },
            {
                'type': 'medication_prescribed',
                'data': {
                    'patient_id': 'test-patient-001',
                    'prescription_id': 'RX-001',
                    'rxnorm': '197361',
                    'drug_name': 'Lisinopril 10mg',
                    'dose': '10',
                    'dose_unit': 'mg',
                    'frequency': 'once daily',
                    'start_date': datetime.utcnow().isoformat()
                }
            },
            {
                'type': 'diagnosis_recorded',
                'data': {
                    'patient_id': 'test-patient-001',
                    'diagnosis_id': 'DX-001',
                    'code': 'I10',
                    'system': 'ICD10',
                    'condition_name': 'Essential hypertension',
                    'onset_date': datetime.utcnow().isoformat()
                }
            }
        ]

        # Process test events
        results = []
        for event in test_events:
            try:
                if event['type'] == 'patient_admitted':
                    await self._handle_patient_admission(event)
                elif event['type'] == 'medication_prescribed':
                    await self._handle_medication_event(event)
                elif event['type'] == 'diagnosis_recorded':
                    await self._handle_diagnosis_event(event)

                results.append({
                    'event_type': event['type'],
                    'status': 'success'
                })
            except Exception as e:
                results.append({
                    'event_type': event['type'],
                    'status': 'error',
                    'error': str(e)
                })

        return {
            'test_results': results,
            'total_events': len(test_events),
            'successful_events': len([r for r in results if r['status'] == 'success']),
            'timestamp': datetime.utcnow().isoformat()
        }


# CLI script functionality
if __name__ == "__main__":
    import sys
    import argparse
    from ..neo4j_setup.dual_stream_manager import Neo4jDualStreamManager

    async def main():
        parser = argparse.ArgumentParser(description='Patient Data Stream Handler')
        parser.add_argument('--test', action='store_true',
                          help='Run test with sample data')
        parser.add_argument('--start', action='store_true',
                          help='Start patient data stream processing')

        args = parser.parse_args()

        # Configuration
        config = {
            'neo4j': {
                'neo4j_uri': 'bolt://localhost:7687',
                'neo4j_user': 'neo4j',
                'neo4j_password': 'kb7password'
            },
            'kafka': {
                'brokers': ['localhost:9092']
            }
        }

        # Initialize Neo4j manager
        neo4j_manager = Neo4jDualStreamManager(config['neo4j'])
        await neo4j_manager.initialize_databases()

        # Initialize patient data handler
        handler = PatientDataStreamHandler(neo4j_manager, config['kafka'])

        if args.test:
            result = await handler.test_patient_stream()
            print(json.dumps(result, indent=2))
        elif args.start:
            await handler.start_processing()
        else:
            print("Use --test to run tests or --start to begin processing")

        await neo4j_manager.close()

    asyncio.run(main())