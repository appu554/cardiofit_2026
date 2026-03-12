import strawberry
import httpx
import logging
from typing import Optional, Dict, Any, List
from .types import (
    AuthResponse, User, Patient, Note, LabResult, Condition,
    MedicationRequest, DiagnosticReport, Encounter, DocumentReference
)
from .inputs import (
    PatientInput, ObservationInput, ConditionInput, MedicationRequestInput,
    DiagnosticReportInput, EncounterInput, DocumentReferenceInput, NoteInput
)
from .utils import convert_graphql_to_fhir, convert_fhir_to_graphql
from app.config import settings

# Set up logging
logger = logging.getLogger(__name__)

@strawberry.type
class Mutation:
    @strawberry.mutation
    async def login(self, info, username: str, password: str) -> AuthResponse:
        # Call auth service to login
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{settings.AUTH_SERVICE_URL}/api/auth/token",
                json={"username": username, "password": password}
            )

            if response.status_code != 200:
                return AuthResponse(
                    success=False,
                    message="Invalid credentials"
                )

            token_data = response.json()
            return AuthResponse(
                success=True,
                token=token_data["access_token"]
            )

    # Patient mutations
    @strawberry.mutation
    async def create_patient(
        self,
        info,
        patient_data: Optional[PatientInput] = None,
        input: Optional[PatientInput] = None
    ) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Use whichever parameter is provided (input or patient_data)
        data = input if input is not None else patient_data

        if data is None:
            logger.error("No patient data provided")
            return None

        # Convert the input to a dictionary
        if hasattr(data, "__dict__"):
            # If it's a Strawberry input type, convert to dict
            patient_dict = {k: v for k, v in data.__dict__.items() if v is not None}
        else:
            # If it's already a dict, use it directly
            patient_dict = data

        # Ensure resourceType is set
        if "resourceType" not in patient_dict:
            patient_dict["resourceType"] = "Patient"

        # Skip validation - pass the data directly to the Patient service

        # Create patient in Patient service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.PATIENT_SERVICE_URL}/api/patients",
                    json=patient_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                patient_data = response.json()
                # Convert FHIR data to GraphQL type using DTL
                return convert_fhir_to_graphql(patient_data, Patient)
        except Exception as e:
            print(f"Error creating patient: {str(e)}")
            logger.error(f"Error creating patient: {str(e)}")
            return None

    @strawberry.mutation
    async def update_patient(
        self,
        info,
        id: str,
        patient_data: Optional[PatientInput] = None,
        input: Optional[PatientInput] = None
    ) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Use whichever parameter is provided (input or patient_data)
        data = input if input is not None else patient_data

        if data is None:
            logger.error("No patient data provided")
            return None

        # Convert the input to a dictionary
        if hasattr(data, "__dict__"):
            # If it's a Strawberry input type, convert to dict
            patient_dict = {k: v for k, v in data.__dict__.items() if v is not None}
        else:
            # If it's already a dict, use it directly
            patient_dict = data

        # Ensure resourceType is set
        if "resourceType" not in patient_dict:
            patient_dict["resourceType"] = "Patient"

        # Add id
        patient_dict["id"] = id

        # Skip validation - pass the data directly to the Patient service

        # Update patient in Patient service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.put(
                    f"{settings.PATIENT_SERVICE_URL}/api/patients/{id}",
                    json=patient_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                patient_data = response.json()
                # Convert FHIR data to GraphQL type using DTL
                return convert_fhir_to_graphql(patient_data, Patient)
        except Exception as e:
            print(f"Error updating patient: {str(e)}")
            logger.error(f"Error updating patient: {str(e)}")
            return None

    # Note mutations
    @strawberry.mutation
    async def create_note(self, info, note_data: NoteInput) -> Optional[Note]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict (Note is not a FHIR resource, so we don't use DTL)
        note_dict = {k: v for k, v in note_data.__dict__.items() if v is not None}

        # Create note in notes service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.NOTES_SERVICE_URL}/api/notes",
                    json=note_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                note_data = response.json()
                return Note(**note_data)
        except Exception as e:
            print(f"Error creating note: {str(e)}")
            return None

    # FHIR resource mutations
    @strawberry.mutation
    async def create_observation(self, info, observation_data: ObservationInput) -> Optional[LabResult]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to FHIR format using DTL
        observation_dict = convert_graphql_to_fhir(observation_data, "Observation")

        if not observation_dict:
            logger.error("Failed to convert observation data to FHIR format")
            return None

        # Create observation directly in Observation service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                    json=observation_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                observation_data = response.json()
                # Convert FHIR data to GraphQL type using DTL
                return convert_fhir_to_graphql(observation_data, LabResult)
        except Exception as e:
            print(f"Error creating observation: {str(e)}")
            logger.error(f"Error creating observation: {str(e)}")
            return None

    @strawberry.mutation
    async def create_condition(self, info, condition_data: ConditionInput) -> Optional[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to FHIR format using DTL
        condition_dict = convert_graphql_to_fhir(condition_data, "Condition")

        if not condition_dict:
            logger.error("Failed to convert condition data to FHIR format")
            return None

        # Create condition directly in Condition service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.CONDITION_SERVICE_URL}/api/conditions",
                    json=condition_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                condition_data = response.json()
                # Convert FHIR data to GraphQL type using DTL
                return convert_fhir_to_graphql(condition_data, Condition)
        except Exception as e:
            print(f"Error creating condition: {str(e)}")
            logger.error(f"Error creating condition: {str(e)}")
            return None

    @strawberry.mutation
    async def create_medication_request(self, info, medication_data: MedicationRequestInput) -> Optional[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to FHIR format using DTL
        medication_dict = convert_graphql_to_fhir(medication_data, "MedicationRequest")

        if not medication_dict:
            logger.error("Failed to convert medication request data to FHIR format")
            return None

        # Create medication request directly in Medication service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.MEDICATION_SERVICE_URL}/api/medications",
                    json=medication_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                medication_data = response.json()
                # Convert FHIR data to GraphQL type using DTL
                return convert_fhir_to_graphql(medication_data, MedicationRequest)
        except Exception as e:
            print(f"Error creating medication request: {str(e)}")
            logger.error(f"Error creating medication request: {str(e)}")
            return None

    @strawberry.mutation
    async def create_diagnostic_report(self, info, report_data: DiagnosticReportInput) -> Optional[DiagnosticReport]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to FHIR format using DTL
        report_dict = convert_graphql_to_fhir(report_data, "DiagnosticReport")

        if not report_dict:
            logger.error("Failed to convert diagnostic report data to FHIR format")
            return None

        # Create diagnostic report directly in Labs service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.LABS_SERVICE_URL}/api/labs/reports",
                    json=report_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                report_data = response.json()
                # Convert FHIR data to GraphQL type using DTL
                return convert_fhir_to_graphql(report_data, DiagnosticReport)
        except Exception as e:
            print(f"Error creating diagnostic report: {str(e)}")
            logger.error(f"Error creating diagnostic report: {str(e)}")
            return None