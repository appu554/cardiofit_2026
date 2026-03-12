"""
FHIR Store Projector
Consumes from prod.ehr.fhir.upsert and writes to Google Cloud Healthcare FHIR Store
"""

import json
from typing import List, Dict, Any
import structlog

from module8_shared.kafka_consumer_base import KafkaConsumerBase
from module8_shared.models.events import FHIRResource
from .fhir_store_handler import FHIRStoreHandler

logger = structlog.get_logger(__name__)


class FHIRStoreProjector(KafkaConsumerBase):
    """
    FHIR Store projector for Google Cloud Healthcare API

    Key Characteristics:
    - Input: prod.ehr.fhir.upsert (FHIRResource objects)
    - NO TRANSFORMATION: Resources are pre-transformed by Module 6
    - Output: Google Cloud Healthcare FHIR Store
    - Small batches (20) due to API rate limits
    - Retry with exponential backoff for transient errors

    Supported Resource Types:
    - Observation, RiskAssessment, DiagnosticReport
    - Condition, MedicationRequest, Procedure
    - Encounter, Patient
    """

    def __init__(self, config: Dict[str, Any]):
        """Initialize FHIR Store projector with handler and settings"""
        super().__init__(
            kafka_config=config['kafka'],
            topics=[config['topics']['fhir_upsert']],
            batch_size=config.get('batch_size', 20),  # Small batch for API
            batch_timeout_seconds=config.get('batch_timeout_seconds', 10.0),
            dlq_topic=config.get('topics', {}).get('dlq'),
        )

        # Initialize FHIR Store handler
        fhir_config = config['fhir_store']
        self.handler = FHIRStoreHandler(
            project_id=fhir_config['project_id'],
            location=fhir_config['location'],
            dataset_id=fhir_config['dataset_id'],
            store_id=fhir_config['store_id'],
            credentials_path=fhir_config['credentials_path'],
            max_retries=fhir_config.get('max_retries', 3),
            retry_backoff_factor=fhir_config.get('retry_backoff_factor', 2.0),
        )

        # Processing statistics
        self.processing_stats = {
            'total_processed': 0,
            'successful_upserts': 0,
            'failed_upserts': 0,
            'validation_errors': 0,
            'resource_type_counts': {},
        }

        logger.info(
            "FHIR Store Projector initialized",
            fhir_store_path=self.handler.fhir_store_path,
            batch_size=config.get('batch_size', 20),
            dlq_topic=config.get('topics', {}).get('dlq'),
        )

    def get_projector_name(self) -> str:
        """Return unique projector identifier"""
        return "fhir-store-projector"

    def process_batch(self, messages: List[Dict[str, Any]]) -> None:
        """
        Process batch of FHIR resources and write to Google Cloud FHIR Store

        Strategy:
        1. Parse FHIRResource objects
        2. Validate resource structure
        3. Upsert to FHIR Store (UPDATE or CREATE)
        4. Handle errors with DLQ

        Args:
            messages: List of FHIRResource dicts from prod.ehr.fhir.upsert

        Note: Resources are PRE-TRANSFORMED by Module 6, no transformation needed
        """
        success_count = 0
        failure_count = 0

        for message in messages:
            try:
                # Parse FHIRResource object
                fhir_resource = self._parse_fhir_resource(message)

                # Upsert to FHIR Store
                result = self.handler.upsert_resource(fhir_resource)

                if result['success']:
                    success_count += 1
                    self.processing_stats['successful_upserts'] += 1

                    # Track resource type
                    resource_type = result['resource_type']
                    self.processing_stats['resource_type_counts'][resource_type] = \
                        self.processing_stats['resource_type_counts'].get(resource_type, 0) + 1

                    logger.debug(
                        "FHIR resource upserted",
                        resource_type=result['resource_type'],
                        resource_id=result['resource_id'],
                        operation=result['operation'],
                    )
                else:
                    # Upsert failed (after retries)
                    failure_count += 1
                    self.processing_stats['failed_upserts'] += 1

                    logger.error(
                        "FHIR resource upsert failed",
                        resource_type=result['resource_type'],
                        resource_id=result['resource_id'],
                        operation=result['operation'],
                        error=result.get('error'),
                    )

                    # Send to DLQ
                    self._send_to_dlq_with_error(message, result.get('error'))

            except ValueError as e:
                # Validation error
                failure_count += 1
                self.processing_stats['validation_errors'] += 1

                logger.error(
                    "FHIR resource validation failed",
                    error=str(e),
                    message=message,
                )

                # Send to DLQ
                self._send_to_dlq_with_error(message, f"Validation error: {str(e)}")

            except Exception as e:
                # Unexpected error
                failure_count += 1
                self.processing_stats['failed_upserts'] += 1

                logger.error(
                    "Unexpected error processing FHIR resource",
                    error=str(e),
                    message=message,
                    exc_info=True,
                )

                # Send to DLQ
                self._send_to_dlq_with_error(message, f"Unexpected error: {str(e)}")

        self.processing_stats['total_processed'] += len(messages)

        logger.info(
            "Batch processed",
            batch_size=len(messages),
            success_count=success_count,
            failure_count=failure_count,
            total_processed=self.processing_stats['total_processed'],
        )

    def _parse_fhir_resource(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse FHIRResource object from Kafka message

        Args:
            message: Raw message dict from prod.ehr.fhir.upsert

        Returns:
            Dict with resourceType, resourceId, patientId, fhirData

        Raises:
            ValueError: If message structure is invalid
        """
        try:
            # Use Pydantic model for validation
            fhir_resource = FHIRResource(**message)

            # Return as dict for handler
            return {
                'resourceType': fhir_resource.resource_type,
                'resourceId': fhir_resource.resource_id,
                'patientId': fhir_resource.patient_id,
                'fhirData': fhir_resource.fhir_data,
                'lastUpdated': fhir_resource.last_updated,
            }

        except Exception as e:
            raise ValueError(f"Invalid FHIRResource structure: {str(e)}")

    def _send_to_dlq_with_error(self, message: Dict[str, Any], error: str) -> None:
        """
        Send failed message to DLQ with error details

        Args:
            message: Original message
            error: Error description
        """
        dlq_message = {
            'original_message': message,
            'error': error,
            'projector': self.get_projector_name(),
        }

        self.send_to_dlq_value(dlq_message)

    def get_processing_summary(self) -> Dict[str, Any]:
        """Get processing summary including handler stats"""
        handler_stats = self.handler.get_stats()

        return {
            'projector_stats': self.processing_stats,
            'handler_stats': handler_stats,
            'resource_type_breakdown': self._get_resource_type_breakdown(),
        }

    def _get_resource_type_breakdown(self) -> Dict[str, Dict[str, int]]:
        """Get detailed breakdown by resource type"""
        breakdown = {}

        # Combine projector and handler counts
        all_types = set(
            list(self.processing_stats['resource_type_counts'].keys()) +
            list(self.handler.stats['resource_type_counts'].keys())
        )

        for resource_type in all_types:
            breakdown[resource_type] = {
                'processed': self.processing_stats['resource_type_counts'].get(resource_type, 0),
                'handler_count': self.handler.stats['resource_type_counts'].get(resource_type, 0),
            }

        return breakdown

    def close(self) -> None:
        """Cleanup and close connections"""
        logger.info(
            "Shutting down FHIR Store Projector",
            final_stats=self.get_processing_summary(),
        )
        super().close()
