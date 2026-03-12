import strawberry
import httpx
import logging
from typing import List, Optional, Dict, Any
from .types import (
    User, Patient, Note, LabResult, Condition, MedicationRequest,
    DiagnosticReport, Encounter, DocumentReference, PatientTimeline, TimelineEvent,
    HumanName, Identifier, ContactPoint, Address
)
from .utils import convert_fhir_to_graphql, handle_request
from app.config import settings

# Set up logging
logger = logging.getLogger(__name__)

@strawberry.type
class Query:
    @strawberry.field
    async def me(self, info) -> Optional[User]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Verify token with auth service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.AUTH_SERVICE_URL}/api/auth/me",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            user_data = response.json()
            return User(**user_data)

    # Patient queries
    @strawberry.field
    async def patient(self, info, id: str) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Fetch patient data from FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    logger.warning(f"Failed to fetch patient {id}: {response.status_code}")
                    return None

                patient_data = response.json()

                # Remove any unexpected fields
                if 'query' in patient_data:
                    del patient_data['query']

                # Convert to Patient object
                patient = convert_fhir_to_graphql(patient_data, Patient)

                # If conversion failed, log the error and return None
                if patient is None:
                    logger.error(f"Failed to convert patient {id} to GraphQL type")
                    return None

                return patient
        except Exception as e:
            logger.exception(f"Error fetching patient {id}: {str(e)}")
            return None

    @strawberry.field
    async def search_patients(self, info, name: Optional[str] = None) -> List[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Define the request function
        async def fetch_patients():
            params = {}
            if name:
                params["name"] = name

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient",
                    params=params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                patients_data = response.json()

                # Convert FHIR data to GraphQL types
                result = []
                for patient_data in patients_data:
                    # Remove any unexpected fields
                    if 'query' in patient_data:
                        del patient_data['query']

                    # Convert to Patient object
                    patient = convert_fhir_to_graphql(patient_data, Patient)
                    if patient:
                        result.append(patient)

                return result

        # Execute the request with error handling
        return await handle_request(auth_header, fetch_patients)

    # Notes queries
    @strawberry.field
    async def patient_notes(self, info, patient_id: str) -> List[Note]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get notes from notes service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.NOTES_SERVICE_URL}/api/patients/{patient_id}/notes",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            notes_data = response.json()
            return [Note(**note) for note in notes_data]

    # FHIR resource queries
    @strawberry.field
    async def patient_observations(self, info, patient_id: str, code: Optional[str] = None) -> List[LabResult]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Define the request function
        async def fetch_observations():
            params = {}
            if code:
                params["code"] = code

            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/Observation",
                    params=params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                observations_data = response.json()

                # Convert FHIR data to GraphQL types
                result = []
                for obs_data in observations_data:
                    # Convert to LabResult object
                    lab_result = convert_fhir_to_graphql(obs_data, LabResult)
                    if lab_result:
                        result.append(lab_result)

                return result

        # Execute the request with error handling
        return await handle_request(auth_header, fetch_observations)

    @strawberry.field
    async def patient_conditions(self, info, patient_id: str) -> List[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Define the request function
        async def fetch_conditions():
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/Condition",
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                conditions_data = response.json()

                # Convert FHIR data to GraphQL types
                result = []
                for condition_data in conditions_data:
                    # Convert to Condition object
                    condition = convert_fhir_to_graphql(condition_data, Condition)
                    if condition:
                        result.append(condition)

                return result

        # Execute the request with error handling
        return await handle_request(auth_header, fetch_conditions)

    @strawberry.field
    async def patient_medications(self, info, patient_id: str) -> List[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication requests from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/MedicationRequest",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            medications_data = response.json()

            # Convert FHIR data to GraphQL types
            result = []
            for med_data in medications_data:
                # Convert to MedicationRequest object
                med_request = convert_fhir_to_graphql(med_data, MedicationRequest)
                if med_request:
                    result.append(med_request)

            return result

    @strawberry.field
    async def patient_diagnostic_reports(self, info, patient_id: str) -> List[DiagnosticReport]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get diagnostic reports from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/DiagnosticReport",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            reports_data = response.json()

            # Convert FHIR data to GraphQL types
            result = []
            for report_data in reports_data:
                # Convert to DiagnosticReport object
                report = convert_fhir_to_graphql(report_data, DiagnosticReport)
                if report:
                    result.append(report)

            return result

    @strawberry.field
    async def patient_encounters(self, info, patient_id: str) -> List[Encounter]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get encounters from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/Encounter",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            encounters_data = response.json()

            # Convert FHIR data to GraphQL types
            result = []
            for encounter_data in encounters_data:
                # Convert to Encounter object
                encounter = convert_fhir_to_graphql(encounter_data, Encounter)
                if encounter:
                    result.append(encounter)

            return result

    @strawberry.field
    async def patient_documents(self, info, patient_id: str) -> List[DocumentReference]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get document references from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/DocumentReference",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            documents_data = response.json()

            # Convert FHIR data to GraphQL types
            result = []
            for doc_data in documents_data:
                # Convert to DocumentReference object
                doc = convert_fhir_to_graphql(doc_data, DocumentReference)
                if doc:
                    result.append(doc)

            return result

    @strawberry.field
    async def patient_timeline(self, info, patient_id: str) -> Optional[PatientTimeline]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get timeline from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/timeline",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            timeline_data = response.json()

            # Convert to PatientTimeline object
            return convert_fhir_to_graphql(timeline_data, PatientTimeline)