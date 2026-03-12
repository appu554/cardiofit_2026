from typing import Dict, List, Optional, Any
import httpx
import logging
from datetime import datetime
from app.models.timeline import TimelineEvent, PatientTimeline, TimelineFilter, EventType, ResourceType
from app.core.config import settings
from shared.models import (
    Patient, Observation, Condition, Encounter,
    Medication, MedicationRequest, MedicationAdministration, MedicationStatement,
    DiagnosticReport
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class TimelineService:
    """Service for aggregating and managing patient timeline data."""

    async def get_patient_timeline(self, patient_id: str, auth_header: str, filter_params: Optional[TimelineFilter] = None) -> PatientTimeline:
        """
        Get a patient's timeline by aggregating data from various clinical services.

        Args:
            patient_id: The patient ID
            auth_header: The authorization header for API calls
            filter_params: Optional parameters to filter the timeline

        Returns:
            The patient timeline with events from various sources
        """
        # Add very visible logging
        print(f"\n\n==== TIMELINE SERVICE RECEIVED REQUEST FOR PATIENT {patient_id} ====")
        print(f"Auth Header: {auth_header[:20]}...")  # Only show first 20 chars for security
        if filter_params:
            print(f"Filter Params: {filter_params}")
        print(f"==== END TIMELINE SERVICE REQUEST ====\n\n")

        logger.info(f"Getting timeline for patient {patient_id}")

        # Initialize timeline events list
        timeline_events = []

        # Fetch data from various services in parallel
        async with httpx.AsyncClient() as client:
            # Create tasks for fetching data from different services
            tasks = [
                self._get_observations(client, patient_id, auth_header),
                self._get_conditions(client, patient_id, auth_header),
                self._get_medications(client, patient_id, auth_header),
                self._get_encounters(client, patient_id, auth_header),
                self._get_documents(client, patient_id, auth_header)
            ]

            # Execute all tasks and gather results
            import asyncio
            logger.info(f"Fetching data from {len(tasks)} services in parallel")
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Process results and add to timeline events
            for i, result in enumerate(results):
                if isinstance(result, Exception):
                    logger.error(f"Error fetching timeline data from task {i}: {str(result)}")
                    continue

                logger.info(f"Task {i} returned {len(result)} events")
                timeline_events.extend(result)

        # Apply filters if provided
        if filter_params:
            timeline_events = self._apply_filters(timeline_events, filter_params)

        # Sort timeline events by date (newest first)
        timeline_events.sort(key=lambda event: event.date, reverse=True)

        # Create and return the patient timeline
        return PatientTimeline(
            patient_id=patient_id,
            events=timeline_events
        )

    async def _get_observations(self, client: httpx.AsyncClient, patient_id: str, auth_header: str) -> List[TimelineEvent]:
        """Fetch observation data for the patient."""
        try:
            # Log the request
            url = f"{settings.FHIR_SERVICE_URL}/api/fhir/Observation?subject=Patient/{patient_id}"
            logger.info(f"Fetching observations from {url}")

            # Get observations from FHIR service
            response = await client.get(
                url,
                headers={"Authorization": auth_header},
                timeout=10.0
            )

            # Log the response status
            logger.info(f"Observation service response status: {response.status_code}")

            if response.status_code != 200:
                logger.error(f"Error fetching observations: {response.status_code} - {response.text}")
                return []

            observations = response.json()
            logger.info(f"Received {len(observations)} observations from FHIR service")

            events = []

            for obs in observations:
                # Create a timeline event from the observation using the shared models
                event = TimelineEvent.from_observation(obs, patient_id)
                events.append(event)

            logger.info(f"Created {len(events)} timeline events from observations")
            return events
        except Exception as e:
            logger.error(f"Error processing observations: {str(e)}")
            return []

    async def _get_conditions(self, client: httpx.AsyncClient, patient_id: str, auth_header: str) -> List[TimelineEvent]:
        """Fetch condition data for the patient."""
        try:
            # Get conditions from FHIR service
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Condition?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header},
                timeout=10.0
            )

            if response.status_code != 200:
                logger.error(f"Error fetching conditions: {response.status_code} - {response.text}")
                return []

            conditions = response.json()
            events = []

            for cond in conditions:
                # Create a timeline event from the condition using the shared models
                event = TimelineEvent.from_condition(cond, patient_id)
                events.append(event)

            return events
        except Exception as e:
            logger.error(f"Error processing conditions: {str(e)}")
            return []

    async def _get_medications(self, client: httpx.AsyncClient, patient_id: str, auth_header: str) -> List[TimelineEvent]:
        """Fetch medication data for the patient."""
        try:
            # Get medication requests from FHIR service
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/MedicationRequest?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header},
                timeout=10.0
            )

            if response.status_code != 200:
                logger.error(f"Error fetching medications: {response.status_code} - {response.text}")
                return []

            medications = response.json()
            events = []

            for med in medications:
                # Create a timeline event from the medication request using the shared models
                event = TimelineEvent.from_medication_request(med, patient_id)
                events.append(event)

            return events
        except Exception as e:
            logger.error(f"Error processing medications: {str(e)}")
            return []

    async def _get_encounters(self, client: httpx.AsyncClient, patient_id: str, auth_header: str) -> List[TimelineEvent]:
        """Fetch encounter data for the patient."""
        try:
            # Get encounters from FHIR service
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Encounter?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header},
                timeout=10.0
            )

            if response.status_code != 200:
                logger.error(f"Error fetching encounters: {response.status_code} - {response.text}")
                return []

            encounters = response.json()
            events = []

            for enc in encounters:
                # Create a timeline event from the encounter using the shared models
                event = TimelineEvent.from_encounter(enc, patient_id)
                events.append(event)

            return events
        except Exception as e:
            logger.error(f"Error processing encounters: {str(e)}")
            return []

    async def _get_documents(self, client: httpx.AsyncClient, patient_id: str, auth_header: str) -> List[TimelineEvent]:
        """Fetch document data for the patient."""
        try:
            # Get documents from FHIR service
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/DocumentReference?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header},
                timeout=10.0
            )

            if response.status_code != 200:
                logger.error(f"Error fetching documents: {response.status_code} - {response.text}")
                return []

            documents = response.json()
            events = []

            for doc in documents:
                # Create a timeline event from the document using the shared models
                event = TimelineEvent.from_document(doc, patient_id)
                events.append(event)

            return events
        except Exception as e:
            logger.error(f"Error processing documents: {str(e)}")
            return []

    def _apply_filters(self, events: List[TimelineEvent], filter_params: TimelineFilter) -> List[TimelineEvent]:
        """Apply filters to timeline events."""
        filtered_events = events

        # Filter by date range
        if filter_params.start_date:
            filtered_events = [e for e in filtered_events if e.date >= filter_params.start_date]

        if filter_params.end_date:
            filtered_events = [e for e in filtered_events if e.date <= filter_params.end_date]

        # Filter by event types
        if filter_params.event_types:
            filtered_events = [e for e in filtered_events if e.event_type in filter_params.event_types]

        # Filter by resource types
        if filter_params.resource_types:
            filtered_events = [e for e in filtered_events if e.resource_type in filter_params.resource_types]

        return filtered_events

# Singleton instance
_timeline_service = None

def get_timeline_service() -> TimelineService:
    """Get the timeline service singleton instance."""
    global _timeline_service
    if _timeline_service is None:
        _timeline_service = TimelineService()
    return _timeline_service
