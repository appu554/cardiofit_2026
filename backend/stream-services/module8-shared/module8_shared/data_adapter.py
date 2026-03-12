"""
Data adapter for Module 6 to Module 8 format conversion
Handles format mismatches between Flink output and Python Pydantic models
"""

import uuid
from typing import Dict, Any, Optional
from datetime import datetime
import structlog

logger = structlog.get_logger(__name__)


class Module6DataAdapter:
    """Adapter to convert Module 6 Flink output to Module 8 expected format"""

    @staticmethod
    def convert_timestamp(timestamp: Any) -> int:
        """
        Convert various timestamp formats to Unix milliseconds

        Handles:
        - Arrays: [2025, 11, 18, 6, 39, 2] -> milliseconds
        - Integers: Already in milliseconds
        - Strings: ISO format
        """
        if isinstance(timestamp, list):
            # Java LocalDateTime serialized as array [year, month, day, hour, minute, second]
            if len(timestamp) >= 3:
                year = timestamp[0]
                month = timestamp[1]
                day = timestamp[2]
                hour = timestamp[3] if len(timestamp) > 3 else 0
                minute = timestamp[4] if len(timestamp) > 4 else 0
                second = timestamp[5] if len(timestamp) > 5 else 0

                dt = datetime(year, month, day, hour, minute, second)
                return int(dt.timestamp() * 1000)

        elif isinstance(timestamp, int):
            return timestamp

        elif isinstance(timestamp, str):
            dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
            return int(dt.timestamp() * 1000)

        # Default: current timestamp
        return int(datetime.now().timestamp() * 1000)

    @staticmethod
    def ensure_raw_data(event: Dict[str, Any]) -> Dict[str, Any]:
        """
        Ensure rawData field exists with at least empty dict
        Module 6 may omit this field
        """
        if 'rawData' not in event or event['rawData'] is None:
            event['rawData'] = {}
        return event

    @staticmethod
    def ensure_id(event: Dict[str, Any]) -> Dict[str, Any]:
        """
        Ensure event has a valid ID
        Generate UUID if missing
        """
        if not event.get('id'):
            # Try to use eventId if available
            if event.get('eventId'):
                event['id'] = event['eventId']
            else:
                # Generate UUID
                event['id'] = str(uuid.uuid4())
        return event

    @staticmethod
    def ensure_event_type(event: Dict[str, Any]) -> Dict[str, Any]:
        """
        Ensure eventType field exists
        Use sourceEventType as fallback
        """
        if not event.get('eventType'):
            event['eventType'] = event.get('sourceEventType', 'UNKNOWN')
        return event

    @staticmethod
    def normalize_ml_predictions(event: Dict[str, Any]) -> Dict[str, Any]:
        """
        Normalize mlPredictions field
        Module 6 may send as list instead of dict
        """
        ml_preds = event.get('mlPredictions')

        if isinstance(ml_preds, list):
            # Convert list to dict with default keys
            if len(ml_preds) > 0:
                # Take first prediction and extract relevant fields
                first_pred = ml_preds[0]
                event['mlPredictions'] = {
                    'sepsis_risk_24h': first_pred.get('primaryScore'),
                    'confidence': first_pred.get('confidence'),
                    'model_type': first_pred.get('model_type'),
                }
            else:
                event['mlPredictions'] = None

        return event

    @staticmethod
    def adapt_event(raw_event: Dict[str, Any]) -> Dict[str, Any]:
        """
        Full adaptation pipeline for Module 6 events

        Args:
            raw_event: Raw event from Module 6 Kafka topic

        Returns:
            Adapted event compatible with Module 8 models
        """
        try:
            # Convert timestamp
            if 'timestamp' in raw_event:
                raw_event['timestamp'] = Module6DataAdapter.convert_timestamp(
                    raw_event['timestamp']
                )

            # Ensure required fields
            raw_event = Module6DataAdapter.ensure_raw_data(raw_event)
            raw_event = Module6DataAdapter.ensure_id(raw_event)
            raw_event = Module6DataAdapter.ensure_event_type(raw_event)
            raw_event = Module6DataAdapter.normalize_ml_predictions(raw_event)

            return raw_event

        except Exception as e:
            logger.error(
                "Failed to adapt event",
                error=str(e),
                event_id=raw_event.get('id', 'unknown')
            )
            raise
