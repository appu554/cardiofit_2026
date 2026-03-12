import strawberry
import strawberry.asgi
from fastapi import FastAPI, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from fastapi.responses import JSONResponse, HTMLResponse
from fastapi.staticfiles import StaticFiles
import pathlib
from strawberry.fastapi import GraphQLRouter
import httpx
from typing import List, Optional
from datetime import datetime
import uuid
import logging
import time
import os
from dotenv import load_dotenv
from contextlib import asynccontextmanager
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request as StarletteRequest
from starlette.types import ASGIApp

# Load environment variables
load_dotenv()

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler("graphql_gateway.log")
    ]
)
logger = logging.getLogger("graphql_gateway")

# Environment variables
FHIR_SERVICE_URL = os.getenv("FHIR_SERVICE_URL", "http://localhost:8004")
AUTH_SERVICE_URL = os.getenv("AUTH_SERVICE_URL", "http://localhost:8000")
OBSERVATION_SERVICE_URL = os.getenv("OBSERVATION_SERVICE_URL", "http://localhost:8007")
NOTES_SERVICE_URL = os.getenv("NOTES_SERVICE_URL", "http://localhost:8008")
LABS_SERVICE_URL = os.getenv("LABS_SERVICE_URL", "http://localhost:8009")
USER_SERVICE_URL = os.getenv("USER_SERVICE_URL", "http://localhost:8001")
PATIENT_SERVICE_URL = os.getenv("PATIENT_SERVICE_URL", "http://localhost:8002")
ENVIRONMENT = os.getenv("ENVIRONMENT", "development")
ALLOWED_ORIGINS = os.getenv("ALLOWED_ORIGINS", "*").split(",")
REQUEST_TIMEOUT = int(os.getenv("REQUEST_TIMEOUT", "30"))
RATE_LIMIT_REQUESTS = int(os.getenv("RATE_LIMIT_REQUESTS", "100"))
RATE_LIMIT_WINDOW = int(os.getenv("RATE_LIMIT_WINDOW", "60"))  # seconds

# Rate limiting middleware
class RateLimitMiddleware(BaseHTTPMiddleware):
    def __init__(self, app: ASGIApp, requests_limit: int = 100, window_size: int = 60):
        super().__init__(app)
        self.requests_limit = requests_limit
        self.window_size = window_size  # in seconds
        self.requests = {}

    async def dispatch(self, request: StarletteRequest, call_next):
        # Get client IP
        client_ip = request.client.host

        # Check if client has exceeded rate limit
        current_time = time.time()
        if client_ip in self.requests:
            requests_times = self.requests[client_ip]
            # Remove old requests
            requests_times = [t for t in requests_times if current_time - t < self.window_size]

            # Check if limit exceeded
            if len(requests_times) >= self.requests_limit:
                logger.warning(f"Rate limit exceeded for IP: {client_ip}")
                return JSONResponse(
                    status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                    content={"detail": "Rate limit exceeded. Please try again later."}
                )

            # Add current request
            requests_times.append(current_time)
            self.requests[client_ip] = requests_times
        else:
            # First request from this IP
            self.requests[client_ip] = [current_time]

        # Process the request
        return await call_next(request)

# Request logging middleware
class RequestLoggingMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: StarletteRequest, call_next):
        start_time = time.time()

        # Process the request
        response = await call_next(request)

        # Log request details
        process_time = time.time() - start_time
        logger.info(
            f"Method: {request.method} Path: {request.url.path} "
            f"Status: {response.status_code} Time: {process_time:.4f}s"
        )

        return response

# Define types
@strawberry.type
class AuthResponse:
    success: bool
    token: Optional[str] = None
    message: Optional[str] = None

@strawberry.type
class User:
    id: str
    email: str
    full_name: Optional[str] = None
    role: str
    is_active: bool
    created_at: datetime

@strawberry.type
class Identifier:
    system: str
    value: str
    use: Optional[str] = None

@strawberry.type
class HumanName:
    family: str
    given: List[str]
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None

@strawberry.type
class ContactPoint:
    system: str
    value: str
    use: Optional[str] = None
    rank: Optional[int] = None

@strawberry.type
class Address:
    line: List[str]
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None
    type: Optional[str] = None

@strawberry.type
class Coding:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.type
class CodeableConcept:
    coding: List[Coding]
    text: Optional[str] = None

@strawberry.type
class Reference:
    reference: str
    display: Optional[str] = None

@strawberry.type
class Quantity:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class Annotation:
    text: str
    authorString: Optional[str] = None
    time: Optional[str] = None

@strawberry.type
class Period:
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.type
class Patient:
    id: str
    resourceType: str = "Patient"
    identifier: List[Identifier]
    name: List[HumanName]
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    active: bool = True
    telecom: Optional[List[ContactPoint]] = None
    address: Optional[List[Address]] = None

@strawberry.type
class Observation:
    id: str
    resourceType: str = "Observation"
    status: str
    category: List[CodeableConcept]
    code: CodeableConcept
    subject: Reference
    effectiveDateTime: str
    valueQuantity: Optional[Quantity] = None
    valueString: Optional[str] = None
    valueCodeableConcept: Optional[CodeableConcept] = None
    interpretation: Optional[List[CodeableConcept]] = None
    note: Optional[List[Annotation]] = None

@strawberry.type
class Condition:
    id: str
    resourceType: str = "Condition"
    clinicalStatus: Optional[CodeableConcept] = None
    verificationStatus: Optional[CodeableConcept] = None
    category: List[CodeableConcept]
    code: CodeableConcept
    subject: Reference
    onsetDateTime: Optional[str] = None
    abatementDateTime: Optional[str] = None
    recordedDate: Optional[str] = None
    note: Optional[List[Annotation]] = None

@strawberry.type
class MedicationRequest:
    id: str
    resourceType: str = "MedicationRequest"
    status: str
    intent: str
    medicationCodeableConcept: CodeableConcept
    subject: Reference
    authoredOn: str
    requester: Optional[Reference] = None
    dosageInstruction: Optional[List[str]] = None
    note: Optional[List[Annotation]] = None

@strawberry.type
class TimelineEvent:
    id: str
    patientId: str
    eventType: str
    resourceType: str
    resourceId: str
    title: str
    description: Optional[str] = None
    date: str

# Define input types
@strawberry.input
class IdentifierInput:
    system: str
    value: str
    use: Optional[str] = None

@strawberry.input
class HumanNameInput:
    family: str
    given: List[str]
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None

@strawberry.input
class ContactPointInput:
    system: str
    value: str
    use: Optional[str] = None
    rank: Optional[int] = None

@strawberry.input
class AddressInput:
    line: List[str]
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None
    type: Optional[str] = None

@strawberry.input
class CodingInput:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.input
class CodeableConceptInput:
    coding: List[CodingInput]
    text: Optional[str] = None

@strawberry.input
class ReferenceInput:
    reference: str
    display: Optional[str] = None

@strawberry.input
class QuantityInput:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.input
class AnnotationInput:
    text: str
    authorString: Optional[str] = None
    time: Optional[str] = None

@strawberry.input
class PeriodInput:
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.input
class PatientInput:
    identifier: List[IdentifierInput]
    name: List[HumanNameInput]
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    active: bool = True
    telecom: Optional[List[ContactPointInput]] = None
    address: Optional[List[AddressInput]] = None

@strawberry.input
class ObservationInput:
    status: str
    category: List[CodeableConceptInput]
    code: CodeableConceptInput
    subject: ReferenceInput
    effectiveDateTime: str
    valueQuantity: Optional[QuantityInput] = None
    valueString: Optional[str] = None
    valueCodeableConcept: Optional[CodeableConceptInput] = None
    interpretation: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class ConditionInput:
    clinicalStatus: Optional[CodeableConceptInput] = None
    verificationStatus: Optional[CodeableConceptInput] = None
    category: List[CodeableConceptInput]
    code: CodeableConceptInput
    subject: ReferenceInput
    onsetDateTime: Optional[str] = None
    abatementDateTime: Optional[str] = None
    recordedDate: Optional[str] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class MedicationRequestInput:
    status: str
    intent: str
    medicationCodeableConcept: CodeableConceptInput
    subject: ReferenceInput
    authoredOn: str
    requester: Optional[ReferenceInput] = None
    dosageInstruction: Optional[List[str]] = None
    note: Optional[List[AnnotationInput]] = None

# Define queries
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
                f"{AUTH_SERVICE_URL}/api/auth/me",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            user_data = response.json()
            return User(**user_data)

    @strawberry.field
    async def patient(self, info, id: str) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get patient from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            patient_data = response.json()

            # Create a properly formatted Patient object
            try:
                # Make sure all required fields are present and properly formatted
                if "name" not in patient_data or not patient_data["name"]:
                    patient_data["name"] = [{"family": "N/A", "given": ["N/A"]}]
                if "identifier" not in patient_data:
                    patient_data["identifier"] = [{"system": "N/A", "value": "N/A"}]
                if "gender" not in patient_data or not patient_data["gender"]:
                    patient_data["gender"] = "N/A"
                if "birthDate" not in patient_data or not patient_data["birthDate"]:
                    patient_data["birthDate"] = "N/A"

                # Create Identifier objects
                identifiers = []
                for ident in patient_data.get("identifier", []):
                    try:
                        # Make sure all required fields are present
                        if "system" not in ident:
                            ident["system"] = "N/A"
                        if "value" not in ident:
                            ident["value"] = "N/A"
                        identifiers.append(Identifier(**ident))
                    except Exception as e:
                        print(f"Error creating Identifier: {e}, using default")
                        identifiers.append(Identifier(system="N/A", value="N/A"))

                # If no identifiers were created, add a default one
                if not identifiers:
                    identifiers.append(Identifier(system="N/A", value="N/A"))

                # Create HumanName objects
                names = []
                for name in patient_data.get("name", []):
                    try:
                        # Make sure all required fields are present
                        if "family" not in name:
                            name["family"] = "N/A"
                        if "given" not in name or not name["given"]:
                            name["given"] = ["N/A"]
                        names.append(HumanName(**name))
                    except Exception as e:
                        print(f"Error creating HumanName: {e}, using default")
                        names.append(HumanName(family="N/A", given=["N/A"]))

                # If no names were created, add a default one
                if not names:
                    names.append(HumanName(family="N/A", given=["N/A"]))

                # Create ContactPoint objects if present
                telecoms = None
                if "telecom" in patient_data and patient_data["telecom"]:
                    try:
                        telecoms = []
                        for telecom in patient_data["telecom"]:
                            # Make sure all required fields are present
                            if "system" not in telecom:
                                telecom["system"] = "phone"
                            if "value" not in telecom:
                                telecom["value"] = "N/A"
                            telecoms.append(ContactPoint(**telecom))
                    except Exception as e:
                        print(f"Error creating ContactPoint: {e}, skipping")

                # Create Address objects if present
                addresses = None
                if "address" in patient_data and patient_data["address"]:
                    try:
                        addresses = []
                        for addr in patient_data["address"]:
                            # Make sure all required fields are present
                            if "line" not in addr or not addr["line"]:
                                addr["line"] = ["N/A"]
                            addresses.append(Address(**addr))
                    except Exception as e:
                        print(f"Error creating Address: {e}, skipping")

                # Create the Patient object
                return Patient(
                    id=patient_data["id"],
                    resourceType=patient_data.get("resourceType", "Patient"),
                    identifier=identifiers,
                    name=names,
                    gender=patient_data.get("gender", "N/A"),
                    birthDate=patient_data.get("birthDate", "N/A"),
                    active=patient_data.get("active", True),
                    telecom=telecoms,
                    address=addresses
                )
            except Exception as e:
                print(f"Error creating Patient object: {e}")
                print(f"Patient data: {patient_data}")
                # Create a default Patient object with minimal data
                return Patient(
                    id=patient_data.get("id", id),
                    resourceType="Patient",
                    identifier=[Identifier(system="N/A", value="N/A")],
                    name=[HumanName(family="N/A", given=["N/A"])],
                    gender="N/A",
                    birthDate="N/A",
                    active=True,
                    telecom=None,
                    address=None
                )

    @strawberry.field
    async def search_patients(self, info, name: Optional[str] = None, gender: Optional[str] = None, birthDate: Optional[str] = None) -> List[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Search patients in FHIR service
        async with httpx.AsyncClient() as client:
            params = {}
            if name:
                params["name"] = name
            if gender:
                params["gender"] = gender
            if birthDate:
                params["birthdate"] = birthDate

            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Patient",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            patients_data = response.json()
            print(f"Raw patients data: {patients_data}")

            # Filter out unexpected fields
            filtered_patients = []
            for patient in patients_data:
                # Only include fields that are defined in the Patient class
                filtered_patient = {}
                for key, value in patient.items():
                    if key in ["id", "resourceType", "identifier", "name", "gender", "birthDate", "active", "telecom", "address"]:
                        filtered_patient[key] = value
                print(f"Original patient: {patient}")
                print(f"Filtered patient: {filtered_patient}")
                filtered_patients.append(filtered_patient)

            # Filter out any fields that aren't part of the Patient model
            clean_patients = []
            for patient in filtered_patients:
                # Skip any patients with a 'query' field - these are likely artifacts
                if 'query' in patient and len(patient.keys()) <= 3:  # Only has query, resourceType, and id
                    print(f"Skipping patient with query field: {patient}")
                    continue

                # Create a clean patient object with only the fields we need
                clean_patient = {}

                # Always include id and resourceType
                clean_patient["id"] = patient.get("id", str(uuid.uuid4()))
                clean_patient["resourceType"] = patient.get("resourceType", "Patient")

                # Add default values for required fields if they're missing
                if "name" not in patient or not patient["name"]:
                    clean_patient["name"] = [{"family": "N/A", "given": ["N/A"]}]
                else:
                    clean_patient["name"] = patient["name"]

                if "identifier" not in patient:
                    clean_patient["identifier"] = [{"system": "N/A", "value": "N/A"}]
                else:
                    clean_patient["identifier"] = patient["identifier"]

                if "gender" not in patient or not patient["gender"]:
                    clean_patient["gender"] = "N/A"
                else:
                    clean_patient["gender"] = patient["gender"]

                if "birthDate" not in patient or not patient["birthDate"]:
                    clean_patient["birthDate"] = "N/A"
                else:
                    clean_patient["birthDate"] = patient["birthDate"]

                # Copy other optional fields if they exist
                if "active" in patient:
                    clean_patient["active"] = patient["active"]
                else:
                    clean_patient["active"] = True

                if "telecom" in patient:
                    clean_patient["telecom"] = patient["telecom"]

                if "address" in patient:
                    clean_patient["address"] = patient["address"]

                # Print the patient data after cleaning
                print(f"Clean patient: {clean_patient}")
                clean_patients.append(clean_patient)

            # Replace the filtered patients with the clean patients
            filtered_patients = clean_patients

            # Create Patient objects with proper nested objects
            result_patients = []
            for patient in filtered_patients:
                try:
                    # Make sure all required fields are present and properly formatted
                    if "name" not in patient or not patient["name"]:
                        patient["name"] = [{"family": "N/A", "given": ["N/A"]}]
                    if "identifier" not in patient:
                        patient["identifier"] = [{"system": "N/A", "value": "N/A"}]
                    if "gender" not in patient or not patient["gender"]:
                        patient["gender"] = "N/A"
                    if "birthDate" not in patient or not patient["birthDate"]:
                        patient["birthDate"] = "N/A"

                    # Create Identifier objects
                    identifiers = []
                    for ident in patient.get("identifier", []):
                        try:
                            # Make sure all required fields are present
                            if "system" not in ident:
                                ident["system"] = "N/A"
                            if "value" not in ident:
                                ident["value"] = "N/A"
                            identifiers.append(Identifier(**ident))
                        except Exception as e:
                            print(f"Error creating Identifier: {e}, using default")
                            identifiers.append(Identifier(system="N/A", value="N/A"))

                    # If no identifiers were created, add a default one
                    if not identifiers:
                        identifiers.append(Identifier(system="N/A", value="N/A"))

                    # Create HumanName objects
                    names = []
                    for name in patient.get("name", []):
                        try:
                            # Make sure all required fields are present
                            if "family" not in name:
                                name["family"] = "N/A"
                            if "given" not in name or not name["given"]:
                                name["given"] = ["N/A"]
                            names.append(HumanName(**name))
                        except Exception as e:
                            print(f"Error creating HumanName: {e}, using default")
                            names.append(HumanName(family="N/A", given=["N/A"]))

                    # If no names were created, add a default one
                    if not names:
                        names.append(HumanName(family="N/A", given=["N/A"]))

                    # Create ContactPoint objects if present
                    telecoms = None
                    if "telecom" in patient and patient["telecom"]:
                        try:
                            telecoms = []
                            for telecom in patient["telecom"]:
                                # Make sure all required fields are present
                                if "system" not in telecom:
                                    telecom["system"] = "phone"
                                if "value" not in telecom:
                                    telecom["value"] = "N/A"
                                telecoms.append(ContactPoint(**telecom))
                        except Exception as e:
                            print(f"Error creating ContactPoint: {e}, skipping")

                    # Create Address objects if present
                    addresses = None
                    if "address" in patient and patient["address"]:
                        try:
                            addresses = []
                            for addr in patient["address"]:
                                # Make sure all required fields are present
                                if "line" not in addr or not addr["line"]:
                                    addr["line"] = ["N/A"]
                                addresses.append(Address(**addr))
                        except Exception as e:
                            print(f"Error creating Address: {e}, skipping")

                    # Create the Patient object
                    patient_obj = Patient(
                        id=patient["id"],
                        resourceType=patient.get("resourceType", "Patient"),
                        identifier=identifiers,
                        name=names,
                        gender=patient.get("gender", "N/A"),
                        birthDate=patient.get("birthDate", "N/A"),
                        active=patient.get("active", True),
                        telecom=telecoms,
                        address=addresses
                    )
                    result_patients.append(patient_obj)
                    print(f"Successfully created Patient object: {patient_obj.id}")
                except Exception as e:
                    print(f"Error creating Patient object: {e}")
                    print(f"Patient data: {patient}")
                    # Create a default Patient object with minimal data
                    try:
                        # Generate a unique ID if none exists
                        patient_id = patient.get("id", None)
                        if not patient_id:
                            patient_id = str(uuid.uuid4())

                        patient_obj = Patient(
                            id=patient_id,
                            resourceType="Patient",
                            identifier=[Identifier(system="N/A", value="N/A")],
                            name=[HumanName(family="N/A", given=["N/A"])],
                            gender="N/A",
                            birthDate="N/A",
                            active=True,
                            telecom=None,
                            address=None
                        )
                        result_patients.append(patient_obj)
                        print(f"Created fallback Patient object: {patient_obj.id}")
                    except Exception as inner_e:
                        print(f"Error creating default Patient object: {inner_e}")

            return result_patients

    @strawberry.field
    async def count_patients(self, info) -> int:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return 0

        # Count patients in FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Patient",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return 0

            patients_data = response.json()
            return len(patients_data)

    @strawberry.field
    async def observation(self, info, id: str) -> Optional[Observation]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get observation from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Observation/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            observation_data = response.json()
            return Observation(**observation_data)

    @strawberry.field
    async def patient_observations(self, info, patient_id: str, category: Optional[str] = None) -> List[Observation]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Try to get observations from Observation service first
        try:
            async with httpx.AsyncClient() as client:
                params = {}
                if category:
                    params["category"] = category

                response = await client.get(
                    f"{OBSERVATION_SERVICE_URL}/api/observations/patient/{patient_id}",
                    params=params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code == 200:
                    observations_data = response.json()
                    return [Observation(**obs) for obs in observations_data]
        except Exception as e:
            logger.warning(f"Error getting observations from Observation service: {str(e)}. Falling back to FHIR service.")

        # Fallback to FHIR service if Observation service fails
        async with httpx.AsyncClient() as client:
            params = {"subject": f"Patient/{patient_id}"}
            if category:
                params["category"] = category

            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Observation",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()
            return [Observation(**obs) for obs in observations_data]

    @strawberry.field
    async def condition(self, info, id: str) -> Optional[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get condition from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Condition/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            condition_data = response.json()
            return Condition(**condition_data)

    @strawberry.field
    async def patient_conditions(self, info, patient_id: str) -> List[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get conditions for patient from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Condition?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            conditions_data = response.json()
            return [Condition(**cond) for cond in conditions_data]

    @strawberry.field
    async def condition(self, info, id: str) -> Optional[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get condition from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Condition/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            condition_data = response.json()
            return Condition(**condition_data)

    @strawberry.field
    async def medication_request(self, info, id: str) -> Optional[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get medication request from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/MedicationRequest/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            medication_data = response.json()
            return MedicationRequest(**medication_data)

    @strawberry.field
    async def patient_medications(self, info, patient_id: str) -> List[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication requests for patient from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/MedicationRequest?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            medications_data = response.json()
            return [MedicationRequest(**med) for med in medications_data]

    # Patient observations method is already defined above

    @strawberry.field
    async def patient_timeline(self, info, patient_id: str) -> List[TimelineEvent]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Create a timeline of events for the patient
        timeline_events = []

        # Get patient data
        async with httpx.AsyncClient() as client:
            # Get observations
            obs_response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Observation?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if obs_response.status_code == 200:
                observations = obs_response.json()
                for obs in observations:
                    event = TimelineEvent(
                        id=f"obs-{obs['id']}",
                        patientId=patient_id,
                        eventType="observation",
                        resourceType="Observation",
                        resourceId=obs['id'],
                        title=obs['code']['coding'][0]['display'] if 'code' in obs and 'coding' in obs['code'] and len(obs['code']['coding']) > 0 and 'display' in obs['code']['coding'][0] else "Observation",
                        description=f"Value: {obs['valueQuantity']['value']} {obs['valueQuantity']['unit']}" if 'valueQuantity' in obs else None,
                        date=obs['effectiveDateTime']
                    )
                    timeline_events.append(event)

            # Get conditions
            cond_response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/Condition?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if cond_response.status_code == 200:
                conditions = cond_response.json()
                for cond in conditions:
                    event = TimelineEvent(
                        id=f"cond-{cond['id']}",
                        patientId=patient_id,
                        eventType="condition",
                        resourceType="Condition",
                        resourceId=cond['id'],
                        title=cond['code']['coding'][0]['display'] if 'code' in cond and 'coding' in cond['code'] and len(cond['code']['coding']) > 0 and 'display' in cond['code']['coding'][0] else "Condition",
                        description=None,
                        date=cond['onsetDateTime'] if 'onsetDateTime' in cond else cond['recordedDate'] if 'recordedDate' in cond else ""
                    )
                    timeline_events.append(event)

            # Get medications
            med_response = await client.get(
                f"{FHIR_SERVICE_URL}/api/fhir/MedicationRequest?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if med_response.status_code == 200:
                medications = med_response.json()
                for med in medications:
                    event = TimelineEvent(
                        id=f"med-{med['id']}",
                        patientId=patient_id,
                        eventType="medication",
                        resourceType="MedicationRequest",
                        resourceId=med['id'],
                        title=med['medicationCodeableConcept']['coding'][0]['display'] if 'medicationCodeableConcept' in med and 'coding' in med['medicationCodeableConcept'] and len(med['medicationCodeableConcept']['coding']) > 0 and 'display' in med['medicationCodeableConcept']['coding'][0] else "Medication",
                        description=None,
                        date=med['authoredOn']
                    )
                    timeline_events.append(event)

        # Sort timeline events by date
        timeline_events.sort(key=lambda event: event.date, reverse=True)

        return timeline_events

    @strawberry.field
    async def medication_request(self, info, id: str) -> Optional[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get medication request from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"http://localhost:8004/api/fhir/MedicationRequest/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return None

            medication_data = response.json()
            return MedicationRequest(**medication_data)

    @strawberry.field
    async def patient_medications(self, info, patient_id: str) -> List[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication requests for patient from FHIR service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"http://localhost:8004/api/fhir/MedicationRequest?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            medications_data = response.json()
            return [MedicationRequest(**med) for med in medications_data]

    @strawberry.field
    async def patient_timeline(self, info, patient_id: str) -> List[TimelineEvent]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Create a timeline of events for the patient
        timeline_events = []

        # Get patient data
        async with httpx.AsyncClient() as client:
            # Get observations
            obs_response = await client.get(
                f"http://localhost:8004/api/fhir/Observation?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if obs_response.status_code == 200:
                observations = obs_response.json()
                for obs in observations:
                    event = TimelineEvent(
                        id=f"obs-{obs['id']}",
                        patientId=patient_id,
                        eventType="observation",
                        resourceType="Observation",
                        resourceId=obs['id'],
                        title=obs['code']['coding'][0]['display'] if 'code' in obs and 'coding' in obs['code'] and len(obs['code']['coding']) > 0 and 'display' in obs['code']['coding'][0] else "Observation",
                        description=f"Value: {obs['valueQuantity']['value']} {obs['valueQuantity']['unit']}" if 'valueQuantity' in obs else None,
                        date=obs['effectiveDateTime']
                    )
                    timeline_events.append(event)

            # Get conditions
            cond_response = await client.get(
                f"http://localhost:8004/api/fhir/Condition?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if cond_response.status_code == 200:
                conditions = cond_response.json()
                for cond in conditions:
                    event = TimelineEvent(
                        id=f"cond-{cond['id']}",
                        patientId=patient_id,
                        eventType="condition",
                        resourceType="Condition",
                        resourceId=cond['id'],
                        title=cond['code']['coding'][0]['display'] if 'code' in cond and 'coding' in cond['code'] and len(cond['code']['coding']) > 0 and 'display' in cond['code']['coding'][0] else "Condition",
                        description=None,
                        date=cond['onsetDateTime'] if 'onsetDateTime' in cond else cond['recordedDate'] if 'recordedDate' in cond else ""
                    )
                    timeline_events.append(event)

            # Get medications
            med_response = await client.get(
                f"http://localhost:8004/api/fhir/MedicationRequest?subject=Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            if med_response.status_code == 200:
                medications = med_response.json()
                for med in medications:
                    event = TimelineEvent(
                        id=f"med-{med['id']}",
                        patientId=patient_id,
                        eventType="medication",
                        resourceType="MedicationRequest",
                        resourceId=med['id'],
                        title=med['medicationCodeableConcept']['coding'][0]['display'] if 'medicationCodeableConcept' in med and 'coding' in med['medicationCodeableConcept'] and len(med['medicationCodeableConcept']['coding']) > 0 and 'display' in med['medicationCodeableConcept']['coding'][0] else "Medication",
                        description=None,
                        date=med['authoredOn']
                    )
                    timeline_events.append(event)

        # Sort timeline events by date
        timeline_events.sort(key=lambda event: event.date, reverse=True)

        return timeline_events

# Define mutations
@strawberry.type
class Mutation:
    @strawberry.mutation
    async def login(self, info, username: str, password: str) -> AuthResponse:
        # Call auth service to login
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{AUTH_SERVICE_URL}/api/auth/token",
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

    @strawberry.mutation
    async def create_patient(self, info, patient_data: PatientInput) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        # We need to handle complex nested objects
        patient_dict = {}

        # Handle identifier list
        if patient_data.identifier:
            patient_dict["identifier"] = []
            for ident in patient_data.identifier:
                patient_dict["identifier"].append({
                    "system": ident.system,
                    "value": ident.value,
                    "use": ident.use
                })

        # Handle name list
        if patient_data.name:
            patient_dict["name"] = []
            for name in patient_data.name:
                patient_dict["name"].append({
                    "family": name.family,
                    "given": name.given,
                    "use": name.use,
                    "prefix": name.prefix,
                    "suffix": name.suffix
                })

        # Handle other fields
        if patient_data.gender is not None:
            patient_dict["gender"] = patient_data.gender
        if patient_data.birthDate is not None:
            patient_dict["birthDate"] = patient_data.birthDate
        if patient_data.active is not None:
            patient_dict["active"] = patient_data.active

        # Handle telecom list
        if patient_data.telecom:
            patient_dict["telecom"] = []
            for telecom in patient_data.telecom:
                patient_dict["telecom"].append({
                    "system": telecom.system,
                    "value": telecom.value,
                    "use": telecom.use,
                    "rank": telecom.rank
                })

        # Handle address list
        if patient_data.address:
            patient_dict["address"] = []
            for addr in patient_data.address:
                patient_dict["address"].append({
                    "line": addr.line,
                    "city": addr.city,
                    "state": addr.state,
                    "postalCode": addr.postalCode,
                    "country": addr.country,
                    "use": addr.use,
                    "type": addr.type
                })

        # Add resourceType
        patient_dict["resourceType"] = "Patient"

        # Create patient in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{FHIR_SERVICE_URL}/api/fhir/Patient",
                    json=patient_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                patient_data = response.json()

                # Create a properly formatted Patient object
                try:
                    # Make sure all required fields are present and properly formatted
                    if "name" not in patient_data or not patient_data["name"]:
                        patient_data["name"] = [{"family": "N/A", "given": ["N/A"]}]
                    if "identifier" not in patient_data:
                        patient_data["identifier"] = [{"system": "N/A", "value": "N/A"}]
                    if "gender" not in patient_data or not patient_data["gender"]:
                        patient_data["gender"] = "N/A"
                    if "birthDate" not in patient_data or not patient_data["birthDate"]:
                        patient_data["birthDate"] = "N/A"

                    # Create Identifier objects
                    identifiers = []
                    for ident in patient_data.get("identifier", []):
                        try:
                            # Make sure all required fields are present
                            if "system" not in ident:
                                ident["system"] = "N/A"
                            if "value" not in ident:
                                ident["value"] = "N/A"
                            identifiers.append(Identifier(**ident))
                        except Exception as e:
                            print(f"Error creating Identifier: {e}, using default")
                            identifiers.append(Identifier(system="N/A", value="N/A"))

                    # If no identifiers were created, add a default one
                    if not identifiers:
                        identifiers.append(Identifier(system="N/A", value="N/A"))

                    # Create HumanName objects
                    names = []
                    for name in patient_data.get("name", []):
                        try:
                            # Make sure all required fields are present
                            if "family" not in name:
                                name["family"] = "N/A"
                            if "given" not in name or not name["given"]:
                                name["given"] = ["N/A"]
                            names.append(HumanName(**name))
                        except Exception as e:
                            print(f"Error creating HumanName: {e}, using default")
                            names.append(HumanName(family="N/A", given=["N/A"]))

                    # If no names were created, add a default one
                    if not names:
                        names.append(HumanName(family="N/A", given=["N/A"]))

                    # Create ContactPoint objects if present
                    telecoms = None
                    if "telecom" in patient_data and patient_data["telecom"]:
                        try:
                            telecoms = []
                            for telecom in patient_data["telecom"]:
                                # Make sure all required fields are present
                                if "system" not in telecom:
                                    telecom["system"] = "phone"
                                if "value" not in telecom:
                                    telecom["value"] = "N/A"
                                telecoms.append(ContactPoint(**telecom))
                        except Exception as e:
                            print(f"Error creating ContactPoint: {e}, skipping")

                    # Create Address objects if present
                    addresses = None
                    if "address" in patient_data and patient_data["address"]:
                        try:
                            addresses = []
                            for addr in patient_data["address"]:
                                # Make sure all required fields are present
                                if "line" not in addr or not addr["line"]:
                                    addr["line"] = ["N/A"]
                                addresses.append(Address(**addr))
                        except Exception as e:
                            print(f"Error creating Address: {e}, skipping")

                    # Create the Patient object
                    return Patient(
                        id=patient_data["id"],
                        resourceType=patient_data.get("resourceType", "Patient"),
                        identifier=identifiers,
                        name=names,
                        gender=patient_data.get("gender", "N/A"),
                        birthDate=patient_data.get("birthDate", "N/A"),
                        active=patient_data.get("active", True),
                        telecom=telecoms,
                        address=addresses
                    )
                except Exception as e:
                    print(f"Error creating Patient object: {e}")
                    print(f"Patient data: {patient_data}")
                    # Create a default Patient object with minimal data
                    return Patient(
                        id=patient_data.get("id", str(uuid.uuid4())),
                        resourceType="Patient",
                        identifier=[Identifier(system="N/A", value="N/A")],
                        name=[HumanName(family="N/A", given=["N/A"])],
                        gender="N/A",
                        birthDate="N/A",
                        active=True,
                        telecom=None,
                        address=None
                    )
        except Exception as e:
            print(f"Error creating patient: {str(e)}")
            return None

    @strawberry.mutation
    async def update_patient(self, info, id: str, patient_data: PatientInput) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        # We need to handle complex nested objects
        patient_dict = {}

        # Handle identifier list
        if patient_data.identifier:
            patient_dict["identifier"] = []
            for ident in patient_data.identifier:
                patient_dict["identifier"].append({
                    "system": ident.system,
                    "value": ident.value,
                    "use": ident.use
                })

        # Handle name list
        if patient_data.name:
            patient_dict["name"] = []
            for name in patient_data.name:
                patient_dict["name"].append({
                    "family": name.family,
                    "given": name.given,
                    "use": name.use,
                    "prefix": name.prefix,
                    "suffix": name.suffix
                })

        # Handle other fields
        if patient_data.gender is not None:
            patient_dict["gender"] = patient_data.gender
        if patient_data.birthDate is not None:
            patient_dict["birthDate"] = patient_data.birthDate
        if patient_data.active is not None:
            patient_dict["active"] = patient_data.active

        # Handle telecom list
        if patient_data.telecom:
            patient_dict["telecom"] = []
            for telecom in patient_data.telecom:
                patient_dict["telecom"].append({
                    "system": telecom.system,
                    "value": telecom.value,
                    "use": telecom.use,
                    "rank": telecom.rank
                })

        # Handle address list
        if patient_data.address:
            patient_dict["address"] = []
            for addr in patient_data.address:
                patient_dict["address"].append({
                    "line": addr.line,
                    "city": addr.city,
                    "state": addr.state,
                    "postalCode": addr.postalCode,
                    "country": addr.country,
                    "use": addr.use,
                    "type": addr.type
                })

        # Add resourceType and id
        patient_dict["resourceType"] = "Patient"
        patient_dict["id"] = id

        # Update patient in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.put(
                    f"{FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                    json=patient_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                patient_data = response.json()

                # Create a properly formatted Patient object
                try:
                    # Make sure all required fields are present and properly formatted
                    if "name" not in patient_data or not patient_data["name"]:
                        patient_data["name"] = [{"family": "N/A", "given": ["N/A"]}]
                    if "identifier" not in patient_data:
                        patient_data["identifier"] = [{"system": "N/A", "value": "N/A"}]
                    if "gender" not in patient_data or not patient_data["gender"]:
                        patient_data["gender"] = "N/A"
                    if "birthDate" not in patient_data or not patient_data["birthDate"]:
                        patient_data["birthDate"] = "N/A"

                    # Create Identifier objects
                    identifiers = []
                    for ident in patient_data.get("identifier", []):
                        try:
                            # Make sure all required fields are present
                            if "system" not in ident:
                                ident["system"] = "N/A"
                            if "value" not in ident:
                                ident["value"] = "N/A"
                            identifiers.append(Identifier(**ident))
                        except Exception as e:
                            print(f"Error creating Identifier: {e}, using default")
                            identifiers.append(Identifier(system="N/A", value="N/A"))

                    # If no identifiers were created, add a default one
                    if not identifiers:
                        identifiers.append(Identifier(system="N/A", value="N/A"))

                    # Create HumanName objects
                    names = []
                    for name in patient_data.get("name", []):
                        try:
                            # Make sure all required fields are present
                            if "family" not in name:
                                name["family"] = "N/A"
                            if "given" not in name or not name["given"]:
                                name["given"] = ["N/A"]
                            names.append(HumanName(**name))
                        except Exception as e:
                            print(f"Error creating HumanName: {e}, using default")
                            names.append(HumanName(family="N/A", given=["N/A"]))

                    # If no names were created, add a default one
                    if not names:
                        names.append(HumanName(family="N/A", given=["N/A"]))

                    # Create ContactPoint objects if present
                    telecoms = None
                    if "telecom" in patient_data and patient_data["telecom"]:
                        try:
                            telecoms = []
                            for telecom in patient_data["telecom"]:
                                # Make sure all required fields are present
                                if "system" not in telecom:
                                    telecom["system"] = "phone"
                                if "value" not in telecom:
                                    telecom["value"] = "N/A"
                                telecoms.append(ContactPoint(**telecom))
                        except Exception as e:
                            print(f"Error creating ContactPoint: {e}, skipping")

                    # Create Address objects if present
                    addresses = None
                    if "address" in patient_data and patient_data["address"]:
                        try:
                            addresses = []
                            for addr in patient_data["address"]:
                                # Make sure all required fields are present
                                if "line" not in addr or not addr["line"]:
                                    addr["line"] = ["N/A"]
                                addresses.append(Address(**addr))
                        except Exception as e:
                            print(f"Error creating Address: {e}, skipping")

                    # Create the Patient object
                    return Patient(
                        id=patient_data["id"],
                        resourceType=patient_data.get("resourceType", "Patient"),
                        identifier=identifiers,
                        name=names,
                        gender=patient_data.get("gender", "N/A"),
                        birthDate=patient_data.get("birthDate", "N/A"),
                        active=patient_data.get("active", True),
                        telecom=telecoms,
                        address=addresses
                    )
                except Exception as e:
                    print(f"Error creating Patient object: {e}")
                    print(f"Patient data: {patient_data}")
                    # Create a default Patient object with minimal data
                    return Patient(
                        id=patient_data.get("id", id),
                        resourceType="Patient",
                        identifier=[Identifier(system="N/A", value="N/A")],
                        name=[HumanName(family="N/A", given=["N/A"])],
                        gender="N/A",
                        birthDate="N/A",
                        active=True,
                        telecom=None,
                        address=None
                    )
        except Exception as e:
            print(f"Error updating patient: {str(e)}")
            return None

    @strawberry.mutation
    async def delete_patient(self, info, id: str) -> bool:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return False

        # Delete patient in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.delete(
                    f"{FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                return True
        except Exception as e:
            print(f"Error deleting patient: {str(e)}")
            return False

    @strawberry.mutation
    async def create_observation(self, info, observation_data: ObservationInput) -> Optional[Observation]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        observation_dict = {k: v for k, v in observation_data.__dict__.items() if v is not None}

        # Add resourceType
        observation_dict["resourceType"] = "Observation"

        # Try to create observation in Observation service first
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{OBSERVATION_SERVICE_URL}/api/observations",
                    json=observation_dict,
                    headers={"Authorization": auth_header}
                )

                if response.status_code == 200 or response.status_code == 201:
                    observation_data = response.json()
                    return Observation(**observation_data)
        except Exception as e:
            logger.warning(f"Error creating observation in Observation service: {str(e)}. Falling back to FHIR service.")

        # Fallback to FHIR service if Observation service fails
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{FHIR_SERVICE_URL}/api/fhir/Observation",
                    json=observation_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                observation_data = response.json()
                return Observation(**observation_data)
        except Exception as e:
            logger.error(f"Error creating observation: {str(e)}")
            return None

    @strawberry.mutation
    async def create_condition(self, info, condition_data: ConditionInput) -> Optional[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        condition_dict = {k: v for k, v in condition_data.__dict__.items() if v is not None}

        # Add resourceType
        condition_dict["resourceType"] = "Condition"

        # Create condition in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{FHIR_SERVICE_URL}/api/fhir/Condition",
                    json=condition_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                condition_data = response.json()
                return Condition(**condition_data)
        except Exception as e:
            print(f"Error creating condition: {str(e)}")
            return None

    @strawberry.mutation
    async def create_medication_request(self, info, medication_data: MedicationRequestInput) -> Optional[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_dict = {k: v for k, v in medication_data.__dict__.items() if v is not None}

        # Add resourceType
        medication_dict["resourceType"] = "MedicationRequest"

        # Create medication request in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{FHIR_SERVICE_URL}/api/fhir/MedicationRequest",
                    json=medication_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                medication_data = response.json()
                return MedicationRequest(**medication_data)
        except Exception as e:
            print(f"Error creating medication request: {str(e)}")
            return None

# Create schema
schema = strawberry.Schema(query=Query, mutation=Mutation)

# Lifespan context manager for startup and shutdown events
from contextlib import asynccontextmanager

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Initialize app state
    app.state.start_time = time.time()
    app.state.total_requests = 0
    app.state.error_requests = 0

    # Log startup
    logger.info(f"Starting GraphQL Gateway in {ENVIRONMENT} environment")
    logger.info(f"FHIR Service URL: {FHIR_SERVICE_URL}")
    logger.info(f"Auth Service URL: {AUTH_SERVICE_URL}")

    yield  # This is where the app runs

    # Shutdown: Cleanup
    logger.info("Shutting down GraphQL Gateway")

# Create FastAPI app with metadata and lifespan
app = FastAPI(
    title="Clinical Synthesis Hub GraphQL Gateway",
    description="GraphQL gateway for the Clinical Synthesis Hub FHIR microservices",
    version="1.0.0",
    docs_url="/docs" if ENVIRONMENT != "production" else None,
    redoc_url="/redoc" if ENVIRONMENT != "production" else None,
    openapi_url="/openapi.json" if ENVIRONMENT != "production" else None,
    lifespan=lifespan
)

# Add middlewares
# 1. Request logging middleware
app.add_middleware(RequestLoggingMiddleware)

# 2. Rate limiting middleware
app.add_middleware(
    RateLimitMiddleware,
    requests_limit=RATE_LIMIT_REQUESTS,
    window_size=RATE_LIMIT_WINDOW
)

# 3. GZIP compression middleware
app.add_middleware(GZipMiddleware, minimum_size=1000)

# 4. CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["Authorization", "Content-Type"],
    max_age=86400,  # 24 hours
)

# Create GraphQL router with context
async def get_context(request: Request):
    # Log request details for debugging
    try:
        if request.method == "POST":
            body = await request.body()
            logger.debug(f"Request body: {body.decode('utf-8')}")
    except Exception as e:
        logger.error(f"Error reading request body: {str(e)}")

    return {"request": request}

graphql_app = GraphQLRouter(
    schema,
    context_getter=get_context,
    graphiql=True  # Enable GraphiQL interface for testing
)

# Mount GraphQL endpoint
app.include_router(graphql_app, prefix="/graphql")

# Create static directory if it doesn't exist
static_dir = pathlib.Path(__file__).parent / "static"
static_dir.mkdir(exist_ok=True)

# Mount static files
app.mount("/static", StaticFiles(directory=str(static_dir)), name="static")

# Endpoint to serve the delete patient HTML form
@app.get("/delete-patient-form", response_class=HTMLResponse)
async def delete_patient_form():
    html_file = static_dir / "delete-patient.html"
    if html_file.exists():
        return HTMLResponse(content=html_file.read_text(), status_code=200)
    else:
        return HTMLResponse(content="<html><body><h1>Error: HTML file not found</h1></body></html>", status_code=404)

# Direct GraphQL endpoint for testing
@app.post("/api/graphql")
async def direct_graphql_endpoint(request: Request):
    try:
        # Parse the request body
        body = await request.json()
        query = body.get("query")
        variables = body.get("variables", {})

        # Log the request for debugging
        logger.info(f"GraphQL query: {query}")
        logger.info(f"GraphQL variables: {variables}")

        # Execute the query
        result = await strawberry.asgi.GraphQL(schema).execute(query, variables, context_value={"request": request})

        # Return the result
        return result
    except Exception as e:
        logger.error(f"Error executing GraphQL query: {str(e)}")
        return JSONResponse(
            status_code=400,
            content={"errors": [{"message": str(e)}]}
        )

# Direct endpoint for delete patient mutation
@app.post("/api/delete-patient/{patient_id}")
async def delete_patient_endpoint(patient_id: str, request: Request):
    # Get authorization header
    auth_header = request.headers.get("Authorization")
    if not auth_header:
        return JSONResponse(status_code=401, content={"error": "Unauthorized"})

    # Delete patient in FHIR service
    try:
        async with httpx.AsyncClient() as client:
            response = await client.delete(
                f"{FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}",
                headers={"Authorization": auth_header}
            )

            response.raise_for_status()
            return {"success": True, "message": f"Patient {patient_id} deleted successfully"}
    except Exception as e:
        logger.error(f"Error deleting patient: {str(e)}")
        return JSONResponse(
            status_code=500,
            content={"success": False, "error": str(e)}
        )

# Health check endpoint
@app.get("/health", tags=["Health"])
async def health_check():
    services_status = {}

    # Check FHIR service health
    services_status["fhir"] = await check_service_health(FHIR_SERVICE_URL)

    # Check Auth service health
    services_status["auth"] = await check_service_health(AUTH_SERVICE_URL)

    # Check Observation service health
    services_status["observation"] = await check_service_health(OBSERVATION_SERVICE_URL)

    # Check Notes service health
    services_status["notes"] = await check_service_health(NOTES_SERVICE_URL)

    # Check Labs service health
    services_status["labs"] = await check_service_health(LABS_SERVICE_URL)

    # Check User service health
    services_status["user"] = await check_service_health(USER_SERVICE_URL)

    # Check Patient service health
    services_status["patient"] = await check_service_health(PATIENT_SERVICE_URL)

    # Overall status is ok if at least FHIR and Auth services are up
    overall_status = "ok" if services_status["fhir"] == "ok" and services_status["auth"] == "ok" else "degraded"

    return {
        "status": overall_status,
        "timestamp": datetime.now().isoformat(),
        "environment": ENVIRONMENT,
        "services": services_status
    }

async def check_service_health(service_url: str) -> str:
    """Check the health of a service by calling its health endpoint"""
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get(f"{service_url}/health")
            if response.status_code == 200:
                return "ok"
            else:
                return f"error: {response.status_code}"
    except Exception as e:
        return f"error: {str(e)}"

# Metrics endpoint
@app.get("/metrics", tags=["Monitoring"])
async def metrics():
    # Return basic metrics
    return {
        "timestamp": datetime.now().isoformat(),
        "uptime": time.time() - app.state.start_time if hasattr(app.state, "start_time") else 0,
        "requests": {
            "total": app.state.total_requests if hasattr(app.state, "total_requests") else 0,
            "errors": app.state.error_requests if hasattr(app.state, "error_requests") else 0
        },
        "rate_limits": {
            "current_ips": len(app.middleware_stack.middlewares[0].requests) if hasattr(app.middleware_stack, "middlewares") else 0
        }
    }

# Documentation endpoint
@app.get("/", tags=["Documentation"])
async def root():
    return {
        "name": "Clinical Synthesis Hub GraphQL Gateway",
        "version": "1.0.0",
        "description": "GraphQL gateway for the Clinical Synthesis Hub FHIR microservices",
        "endpoints": {
            "graphql": "/graphql",
            "health": "/health",
            "metrics": "/metrics",
            "docs": "/docs" if ENVIRONMENT != "production" else None
        },
        "microservices": {
            "auth": AUTH_SERVICE_URL,
            "fhir": FHIR_SERVICE_URL,
            "observation": OBSERVATION_SERVICE_URL,
            "notes": NOTES_SERVICE_URL,
            "labs": LABS_SERVICE_URL,
            "user": USER_SERVICE_URL,
            "patient": PATIENT_SERVICE_URL
        }
    }

# This section has been moved before the app creation

# Exception handler
@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    # Log the exception
    logger.error(f"Unhandled exception: {str(exc)}")
    logger.exception(exc)

    # Increment error counter
    if hasattr(app.state, "error_requests"):
        app.state.error_requests += 1

    # Return a generic error response in production
    if ENVIRONMENT == "production":
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content={"detail": "An internal server error occurred."}
        )

    # Return more detailed error in development
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "detail": "An internal server error occurred.",
            "error": str(exc),
            "traceback": str(exc.__traceback__)
        }
    )

# Request counter middleware
@app.middleware("http")
async def count_requests(request: Request, call_next):
    # Increment request counter
    if hasattr(app.state, "total_requests"):
        app.state.total_requests += 1

    # Process the request
    response = await call_next(request)

    # Return the response
    return response

# Main entry point
if __name__ == "__main__":
    import uvicorn

    # Configure uvicorn logging
    log_config = uvicorn.config.LOGGING_CONFIG
    log_config["formatters"]["access"]["fmt"] = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    log_config["formatters"]["default"]["fmt"] = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"

    # Run the server directly
    # Note: This approach doesn't support auto-reload when files change
    # For development with reload, use the command line:
    # uvicorn services.graphql_gateway.standalone_server:app --reload
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=8006,
        log_config=log_config,
        log_level="info"
    )
