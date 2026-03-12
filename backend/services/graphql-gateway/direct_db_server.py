import strawberry
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from strawberry.fastapi import GraphQLRouter
import httpx
from typing import List, Optional, Dict, Any
from datetime import datetime
import uuid
import os
from dotenv import load_dotenv
from app.db.mongodb import connect_to_mongo, close_mongo_connection, get_patients, get_patient, create_patient, update_patient, delete_patient

# Load environment variables
load_dotenv()

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

# Helper functions to convert between types
def convert_input_to_dict(input_obj):
    """Convert an input object to a dictionary."""
    if input_obj is None:
        return None

    # Handle specific input types
    if isinstance(input_obj, IdentifierInput):
        return {
            "system": input_obj.system,
            "value": input_obj.value,
            "use": input_obj.use
        }
    elif isinstance(input_obj, HumanNameInput):
        return {
            "family": input_obj.family,
            "given": input_obj.given,
            "use": input_obj.use,
            "prefix": input_obj.prefix,
            "suffix": input_obj.suffix
        }
    elif isinstance(input_obj, ContactPointInput):
        return {
            "system": input_obj.system,
            "value": input_obj.value,
            "use": input_obj.use,
            "rank": input_obj.rank
        }
    elif isinstance(input_obj, AddressInput):
        return {
            "line": input_obj.line,
            "city": input_obj.city,
            "state": input_obj.state,
            "postalCode": input_obj.postalCode,
            "country": input_obj.country,
            "use": input_obj.use,
            "type": input_obj.type
        }
    elif isinstance(input_obj, CodingInput):
        return {
            "system": input_obj.system,
            "code": input_obj.code,
            "display": input_obj.display
        }
    elif isinstance(input_obj, CodeableConceptInput):
        return {
            "coding": [convert_input_to_dict(coding) for coding in input_obj.coding],
            "text": input_obj.text
        }
    elif isinstance(input_obj, ReferenceInput):
        return {
            "reference": input_obj.reference,
            "display": input_obj.display
        }
    elif isinstance(input_obj, QuantityInput):
        return {
            "value": input_obj.value,
            "unit": input_obj.unit,
            "system": input_obj.system,
            "code": input_obj.code
        }
    elif isinstance(input_obj, AnnotationInput):
        return {
            "text": input_obj.text,
            "authorString": input_obj.authorString,
            "time": input_obj.time
        }
    elif isinstance(input_obj, PeriodInput):
        return {
            "start": input_obj.start,
            "end": input_obj.end
        }
    elif isinstance(input_obj, PatientInput):
        result = {}
        # Handle identifier list
        if input_obj.identifier:
            result["identifier"] = [convert_input_to_dict(ident) for ident in input_obj.identifier]
        # Handle name list
        if input_obj.name:
            result["name"] = [convert_input_to_dict(name) for name in input_obj.name]
        # Handle other fields
        if input_obj.gender is not None:
            result["gender"] = input_obj.gender
        if input_obj.birthDate is not None:
            result["birthDate"] = input_obj.birthDate
        if input_obj.active is not None:
            result["active"] = input_obj.active
        # Handle telecom list
        if input_obj.telecom:
            result["telecom"] = [convert_input_to_dict(telecom) for telecom in input_obj.telecom]
        # Handle address list
        if input_obj.address:
            result["address"] = [convert_input_to_dict(addr) for addr in input_obj.address]
        return result

    # Generic handling for other types
    result = {}
    for key, value in input_obj.__dict__.items():
        if value is not None:
            if hasattr(value, '__dict__'):
                result[key] = convert_input_to_dict(value)
            elif isinstance(value, list) and len(value) > 0 and hasattr(value[0], '__dict__'):
                result[key] = [convert_input_to_dict(item) for item in value]
            else:
                result[key] = value
    return result

def create_patient_object(patient_data):
    """Create a Patient object from a dictionary."""
    try:
        # Handle MongoDB _id field
        if '_id' in patient_data and 'id' not in patient_data:
            patient_data['id'] = str(patient_data['_id'])

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
                # Check if it's already an Identifier object
                if isinstance(ident, Identifier):
                    identifiers.append(ident)
                else:
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
                # Check if it's already a HumanName object
                if isinstance(name, HumanName):
                    names.append(name)
                else:
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
                    if isinstance(telecom, ContactPoint):
                        telecoms.append(telecom)
                    else:
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
                    if isinstance(addr, Address):
                        addresses.append(addr)
                    else:
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
        # Generate a unique ID if none exists
        patient_id = patient_data.get("id", None)
        if not patient_id and '_id' in patient_data:
            patient_id = str(patient_data['_id'])
        if not patient_id:
            import uuid
            patient_id = str(uuid.uuid4())

        return Patient(
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

        # For direct DB access, we'll just return a mock user
        return User(
            id="1",
            email="user@example.com",
            full_name="Test User",
            role="admin",
            is_active=True,
            created_at=datetime.now()
        )

    @strawberry.field
    async def patient(self, info, id: str) -> Optional[Patient]:
        # Get patient directly from MongoDB
        patient_data = await get_patient(id)
        if not patient_data:
            return None

        return create_patient_object(patient_data)

    @strawberry.field
    async def search_patients(self, info, name: Optional[str] = None, gender: Optional[str] = None, birthDate: Optional[str] = None) -> List[Patient]:
        # Get all patients from MongoDB
        patients_data = await get_patients()

        # Filter patients if search parameters are provided
        filtered_patients = patients_data
        if name:
            filtered_patients = [
                p for p in filtered_patients
                if "name" in p and any(
                    name.lower() in n.get("family", "").lower() or
                    any(name.lower() in g.lower() for g in n.get("given", []))
                    for n in p["name"]
                )
            ]
        if gender:
            filtered_patients = [p for p in filtered_patients if p.get("gender") == gender]
        if birthDate:
            filtered_patients = [p for p in filtered_patients if p.get("birthDate") == birthDate]

        # Convert to Patient objects
        return [create_patient_object(patient) for patient in filtered_patients]

    @strawberry.field
    async def count_patients(self, info) -> int:
        # Count patients in MongoDB
        patients_data = await get_patients()
        return len(patients_data)

# Define mutations
@strawberry.type
class Mutation:
    @strawberry.mutation
    async def login(self, info, username: str, password: str) -> AuthResponse:
        # For direct DB access, we'll just return a mock token
        return AuthResponse(
            success=True,
            token="mock_token_for_direct_db_access",
            message="Logged in successfully"
        )

    @strawberry.mutation
    async def create_patient(self, info, patient_data: PatientInput) -> Optional[Patient]:
        # Convert input to dict
        patient_dict = convert_input_to_dict(patient_data)

        # Add resourceType and id
        patient_dict["resourceType"] = "Patient"
        patient_dict["id"] = str(uuid.uuid4())

        # Create patient in MongoDB
        created_patient = await create_patient(patient_dict)
        if not created_patient:
            return None

        return create_patient_object(created_patient)

    @strawberry.mutation
    async def update_patient(self, info, id: str, patient_data: PatientInput) -> Optional[Patient]:
        # Convert input to dict
        patient_dict = convert_input_to_dict(patient_data)

        # Add resourceType
        patient_dict["resourceType"] = "Patient"

        # Update patient in MongoDB
        updated_patient = await update_patient(id, patient_dict)
        if not updated_patient:
            return None

        return create_patient_object(updated_patient)

    @strawberry.mutation
    async def delete_patient(self, info, id: str) -> bool:
        # Delete patient from MongoDB
        return await delete_patient(id)

# Create schema
schema = strawberry.Schema(query=Query, mutation=Mutation)

# Create FastAPI app
app = FastAPI()

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Connect to MongoDB on startup
@app.on_event("startup")
async def startup_db_client():
    # Use the existing MongoDB connection string from the .env file
    mongodb_uri = os.getenv("MONGODB_URI", "mongodb+srv://admin:Apoorva%40554@cluster0.yqdzbvb.mongodb.net/fhirdb?retryWrites=true&w=majority&appName=Cluster0")
    print(f"Connecting to MongoDB...")
    success = await connect_to_mongo(mongodb_uri)
    if success:
        print("Successfully connected to MongoDB")
    else:
        print("Failed to connect to MongoDB")

# Close MongoDB connection on shutdown
@app.on_event("shutdown")
async def shutdown_db_client():
    await close_mongo_connection()

# Create GraphQL router with context
async def get_context(request: Request):
    return {"request": request}

graphql_app = GraphQLRouter(
    schema,
    context_getter=get_context,
)

# Mount GraphQL endpoint
app.include_router(graphql_app, prefix="/graphql")

# Health check endpoint
@app.get("/health")
async def health_check():
    return {"status": "ok"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8006)
