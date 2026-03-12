import strawberry
import httpx
from typing import Optional, Dict, Any, List
from .types import (
    AuthResponse, User, Patient, LabResult, Condition,
    MedicationRequest, DiagnosticReport, Encounter, DocumentReference,
    CompleteObservation, CreateObservationInput, UpdateObservationInput,
    ObservationCode, ObservationSubject, ObservationValueQuantity,
    ObservationInterpretation, ObservationReferenceRange,
    ObservationReferenceRangeQuantity, VitalSign, PhysicalMeasurement,
    CreateVitalSignInput, UpdateVitalSignInput,
    CreatePhysicalMeasurementInput, UpdatePhysicalMeasurementInput,
    Identifier, HumanName, ContactPoint, Address,
    Medication, MedicationAdministration, MedicationStatement,
    Coding, CodeableConcept, Reference, Quantity, Annotation, DosageInstruction,
    ProblemListItem, Diagnosis, HealthConcern, Meta
)
from .inputs import (
    PatientInput, ObservationInput, ConditionInput, MedicationRequestInput,
    DiagnosticReportInput, EncounterInput, DocumentReferenceInput,
    MedicationInput, MedicationAdministrationInput, MedicationStatementInput
)
from app.config import settings

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
    async def create_patient(self, info, patient_data: PatientInput) -> Optional[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict - handle nested objects properly
        patient_dict = {}

        # Handle identifier list
        if hasattr(patient_data, 'identifier') and patient_data.identifier:
            patient_dict["identifier"] = []
            for ident in patient_data.identifier:
                ident_dict = {}
                if hasattr(ident, 'system'):
                    ident_dict["system"] = ident.system
                if hasattr(ident, 'value'):
                    ident_dict["value"] = ident.value
                if hasattr(ident, 'use') and ident.use:
                    ident_dict["use"] = ident.use
                patient_dict["identifier"].append(ident_dict)

        # Handle name list
        if hasattr(patient_data, 'name') and patient_data.name:
            patient_dict["name"] = []
            for name in patient_data.name:
                name_dict = {}
                if hasattr(name, 'family'):
                    name_dict["family"] = name.family
                if hasattr(name, 'given'):
                    name_dict["given"] = name.given
                if hasattr(name, 'use') and name.use:
                    name_dict["use"] = name.use
                if hasattr(name, 'prefix') and name.prefix:
                    name_dict["prefix"] = name.prefix
                if hasattr(name, 'suffix') and name.suffix:
                    name_dict["suffix"] = name.suffix
                patient_dict["name"].append(name_dict)

        # Handle simple fields
        if hasattr(patient_data, 'gender') and patient_data.gender:
            patient_dict["gender"] = patient_data.gender
        if hasattr(patient_data, 'birthDate') and patient_data.birthDate:
            patient_dict["birthDate"] = patient_data.birthDate
        if hasattr(patient_data, 'active'):
            patient_dict["active"] = patient_data.active

        # Handle telecom list
        if hasattr(patient_data, 'telecom') and patient_data.telecom:
            patient_dict["telecom"] = []
            for telecom in patient_data.telecom:
                telecom_dict = {}
                if hasattr(telecom, 'system'):
                    telecom_dict["system"] = telecom.system
                if hasattr(telecom, 'value'):
                    telecom_dict["value"] = telecom.value
                if hasattr(telecom, 'use') and telecom.use:
                    telecom_dict["use"] = telecom.use
                patient_dict["telecom"].append(telecom_dict)

        # Handle address list
        if hasattr(patient_data, 'address') and patient_data.address:
            patient_dict["address"] = []
            for addr in patient_data.address:
                addr_dict = {}
                if hasattr(addr, 'line'):
                    addr_dict["line"] = addr.line
                if hasattr(addr, 'city') and addr.city:
                    addr_dict["city"] = addr.city
                if hasattr(addr, 'state') and addr.state:
                    addr_dict["state"] = addr.state
                if hasattr(addr, 'postalCode') and addr.postalCode:
                    addr_dict["postalCode"] = addr.postalCode
                if hasattr(addr, 'country') and addr.country:
                    addr_dict["country"] = addr.country
                if hasattr(addr, 'use') and addr.use:
                    addr_dict["use"] = addr.use
                if hasattr(addr, 'type') and addr.type:
                    addr_dict["type"] = addr.type
                patient_dict["address"].append(addr_dict)

        # Add resourceType
        patient_dict["resourceType"] = "Patient"

        # Create patient in FHIR service directly
        try:
            async with httpx.AsyncClient() as client:
                # Try FHIR service directly
                print(f"Creating patient in FHIR service: {patient_dict}")
                response = await client.post(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient",
                    json=patient_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                created_patient = response.json()
                print(f"Created patient: {created_patient}")

                # Create Patient object from response
                try:
                    # Convert the FHIR response to a format compatible with our GraphQL types
                    # Handle identifier list
                    identifiers = []
                    for ident in created_patient.get("identifier", []):
                        identifiers.append(
                            Identifier(
                                system=ident.get("system", ""),
                                value=ident.get("value", ""),
                                use=ident.get("use")
                            )
                        )

                    # Handle name list
                    names = []
                    for name in created_patient.get("name", []):
                        names.append(
                            HumanName(
                                family=name.get("family", ""),
                                given=name.get("given", []),
                                use=name.get("use"),
                                prefix=name.get("prefix"),
                                suffix=name.get("suffix")
                            )
                        )

                    # Handle telecom list
                    telecoms = None
                    if "telecom" in created_patient:
                        telecoms = []
                        for telecom in created_patient.get("telecom", []):
                            telecoms.append(
                                ContactPoint(
                                    system=telecom.get("system", ""),
                                    value=telecom.get("value", ""),
                                    use=telecom.get("use"),
                                    rank=telecom.get("rank")
                                )
                            )

                    # Handle address list
                    addresses = None
                    if "address" in created_patient:
                        addresses = []
                        for addr in created_patient.get("address", []):
                            addresses.append(
                                Address(
                                    line=addr.get("line", []),
                                    city=addr.get("city"),
                                    state=addr.get("state"),
                                    postalCode=addr.get("postalCode"),
                                    country=addr.get("country"),
                                    use=addr.get("use"),
                                    type=addr.get("type")
                                )
                            )

                    # Create the Patient object
                    return Patient(
                        id=created_patient.get("id"),
                        resourceType=created_patient.get("resourceType", "Patient"),
                        identifier=identifiers,
                        name=names,
                        gender=created_patient.get("gender"),
                        birthDate=created_patient.get("birthDate"),
                        active=created_patient.get("active", True),
                        telecom=telecoms,
                        address=addresses
                    )
                except Exception as e:
                    print(f"Error creating Patient object: {str(e)}")
                    # Create a minimal Patient object with the essential data
                    return None
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

        # Convert input to dict - handle nested objects properly
        patient_dict = {}

        # Handle identifier list
        if hasattr(patient_data, 'identifier') and patient_data.identifier:
            patient_dict["identifier"] = []
            for ident in patient_data.identifier:
                ident_dict = {}
                if hasattr(ident, 'system'):
                    ident_dict["system"] = ident.system
                if hasattr(ident, 'value'):
                    ident_dict["value"] = ident.value
                if hasattr(ident, 'use') and ident.use:
                    ident_dict["use"] = ident.use
                patient_dict["identifier"].append(ident_dict)

        # Handle name list
        if hasattr(patient_data, 'name') and patient_data.name:
            patient_dict["name"] = []
            for name in patient_data.name:
                name_dict = {}
                if hasattr(name, 'family'):
                    name_dict["family"] = name.family
                if hasattr(name, 'given'):
                    name_dict["given"] = name.given
                if hasattr(name, 'use') and name.use:
                    name_dict["use"] = name.use
                if hasattr(name, 'prefix') and name.prefix:
                    name_dict["prefix"] = name.prefix
                if hasattr(name, 'suffix') and name.suffix:
                    name_dict["suffix"] = name.suffix
                patient_dict["name"].append(name_dict)

        # Handle simple fields
        if hasattr(patient_data, 'gender') and patient_data.gender:
            patient_dict["gender"] = patient_data.gender
        if hasattr(patient_data, 'birthDate') and patient_data.birthDate:
            patient_dict["birthDate"] = patient_data.birthDate
        if hasattr(patient_data, 'active'):
            patient_dict["active"] = patient_data.active

        # Handle telecom list
        if hasattr(patient_data, 'telecom') and patient_data.telecom:
            patient_dict["telecom"] = []
            for telecom in patient_data.telecom:
                telecom_dict = {}
                if hasattr(telecom, 'system'):
                    telecom_dict["system"] = telecom.system
                if hasattr(telecom, 'value'):
                    telecom_dict["value"] = telecom.value
                if hasattr(telecom, 'use') and telecom.use:
                    telecom_dict["use"] = telecom.use
                patient_dict["telecom"].append(telecom_dict)

        # Handle address list
        if hasattr(patient_data, 'address') and patient_data.address:
            patient_dict["address"] = []
            for addr in patient_data.address:
                addr_dict = {}
                if hasattr(addr, 'line'):
                    addr_dict["line"] = addr.line
                if hasattr(addr, 'city') and addr.city:
                    addr_dict["city"] = addr.city
                if hasattr(addr, 'state') and addr.state:
                    addr_dict["state"] = addr.state
                if hasattr(addr, 'postalCode') and addr.postalCode:
                    addr_dict["postalCode"] = addr.postalCode
                if hasattr(addr, 'country') and addr.country:
                    addr_dict["country"] = addr.country
                if hasattr(addr, 'use') and addr.use:
                    addr_dict["use"] = addr.use
                if hasattr(addr, 'type') and addr.type:
                    addr_dict["type"] = addr.type
                patient_dict["address"].append(addr_dict)

        # Add resourceType and id
        patient_dict["resourceType"] = "Patient"
        patient_dict["id"] = id

        # Update patient in FHIR service directly
        try:
            async with httpx.AsyncClient() as client:
                # Try FHIR service directly
                print(f"Updating patient in FHIR service: {patient_dict}")
                response = await client.put(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                    json=patient_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                updated_patient = response.json()
                print(f"Updated patient: {updated_patient}")

                # Create Patient object from response
                try:
                    # Convert the FHIR response to a format compatible with our GraphQL types
                    # Handle identifier list
                    identifiers = []
                    for ident in updated_patient.get("identifier", []):
                        identifiers.append(
                            Identifier(
                                system=ident.get("system", ""),
                                value=ident.get("value", ""),
                                use=ident.get("use")
                            )
                        )

                    # Handle name list
                    names = []
                    for name in updated_patient.get("name", []):
                        names.append(
                            HumanName(
                                family=name.get("family", ""),
                                given=name.get("given", []),
                                use=name.get("use"),
                                prefix=name.get("prefix"),
                                suffix=name.get("suffix")
                            )
                        )

                    # Handle telecom list
                    telecoms = None
                    if "telecom" in updated_patient:
                        telecoms = []
                        for telecom in updated_patient.get("telecom", []):
                            telecoms.append(
                                ContactPoint(
                                    system=telecom.get("system", ""),
                                    value=telecom.get("value", ""),
                                    use=telecom.get("use"),
                                    rank=telecom.get("rank")
                                )
                            )

                    # Handle address list
                    addresses = None
                    if "address" in updated_patient:
                        addresses = []
                        for addr in updated_patient.get("address", []):
                            addresses.append(
                                Address(
                                    line=addr.get("line", []),
                                    city=addr.get("city"),
                                    state=addr.get("state"),
                                    postalCode=addr.get("postalCode"),
                                    country=addr.get("country"),
                                    use=addr.get("use"),
                                    type=addr.get("type")
                                )
                            )

                    # Create the Patient object
                    return Patient(
                        id=updated_patient.get("id"),
                        resourceType=updated_patient.get("resourceType", "Patient"),
                        identifier=identifiers,
                        name=names,
                        gender=updated_patient.get("gender"),
                        birthDate=updated_patient.get("birthDate"),
                        active=updated_patient.get("active", True),
                        telecom=telecoms,
                        address=addresses
                    )
                except Exception as e:
                    print(f"Error creating Patient object: {str(e)}")
                    # Create a minimal Patient object with the essential data
                    return None
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
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                return True
        except Exception as e:
            print(f"Error deleting patient: {str(e)}")
            return False

    # FHIR resource mutations
    @strawberry.mutation
    async def create_observation(self, info, observation_data: ObservationInput) -> Optional[LabResult]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        observation_dict = {k: v for k, v in observation_data.__dict__.items() if v is not None}

        # Convert field names to match observation service expectations
        if "effectiveDateTime" in observation_dict:
            observation_dict["effective_datetime"] = observation_dict.pop("effectiveDateTime")
        if "valueQuantity" in observation_dict:
            observation_dict["value_quantity"] = observation_dict.pop("valueQuantity")

        # Create a copy for FHIR service with original field names
        fhir_observation_dict = {k: v for k, v in observation_data.__dict__.items() if v is not None}

        # Add resourceType for FHIR
        fhir_observation_dict["resourceType"] = "Observation"

        # Create observation in Observation service
        try:
            async with httpx.AsyncClient() as client:
                # Try to create in Observation service first
                response = await client.post(
                    f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                    json=observation_dict,
                    headers={"Authorization": auth_header}
                )

                # If Observation service fails, fallback to FHIR service
                if response.status_code != 201 and response.status_code != 200:
                    response = await client.post(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Observation",
                        json=fhir_observation_dict,
                        headers={"Authorization": auth_header}
                    )

                response.raise_for_status()
                observation_data = response.json()

                # If document has _id field, convert it to id
                if '_id' in observation_data:
                    observation_data['id'] = str(observation_data.pop('_id'))

                # Convert field names to match LabResult type
                if 'effective_datetime' in observation_data:
                    observation_data['effectiveDateTime'] = observation_data.pop('effective_datetime')
                if 'value_quantity' in observation_data:
                    observation_data['valueQuantity'] = observation_data.pop('value_quantity')

                return LabResult(**observation_data)
        except Exception as e:
            print(f"Error creating observation: {str(e)}")
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

        # Create condition in Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Try to create in Condition service first
                response = await client.post(
                    f"{settings.CONDITION_SERVICE_URL}/api/conditions",
                    json=condition_dict,
                    headers={"Authorization": auth_header}
                )

                # If Condition service fails, fallback to FHIR service
                if response.status_code != 201 and response.status_code != 200:
                    print(f"Condition service failed with status {response.status_code}, falling back to FHIR service")
                    response = await client.post(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Condition",
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
    async def create_problem(self, info, condition_data: ConditionInput) -> Optional[ProblemListItem]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        condition_dict = {k: v for k, v in condition_data.__dict__.items() if v is not None}
        print(f"Input condition data: {condition_dict}")

        # Add resourceType
        condition_dict["resourceType"] = "Condition"
        print(f"Condition dict with resourceType: {condition_dict}")

        # Create problem in Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Try to create in Condition service first
                url = f"{settings.CONDITION_SERVICE_URL}/conditions/problems"
                print(f"Sending request to: {url}")

                # Ensure category is properly set for the condition service
                request_data = condition_dict.copy()
                if "category" not in request_data or not request_data["category"]:
                    request_data["category"] = [{
                        "coding": [
                            {
                                "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                                "code": "problem-list-item",
                                "display": "Problem List Item"
                            }
                        ],
                        "text": "Problem List Item"
                    }]

                print(f"Request data: {request_data}")
                response = await client.post(
                    url,
                    json=request_data,
                    headers={"Authorization": auth_header}
                )
                print(f"Response status: {response.status_code}")
                print(f"Response body: {response.text}")

                # If Condition service fails, fallback to FHIR service with problem category
                if response.status_code != 201 and response.status_code != 200:
                    print(f"Condition service failed with status {response.status_code}, falling back to FHIR service")

                    # Ensure the category is set to problem-list-item
                    if "category" not in condition_dict or not condition_dict["category"]:
                        condition_dict["category"] = [{
                            "coding": [
                                {
                                    "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                                    "code": "problem-list-item",
                                    "display": "Problem List Item"
                                }
                            ],
                            "text": "Problem List Item"
                        }]
                    else:
                        # Check if problem-list-item category already exists
                        has_problem_category = False
                        for cat in condition_dict.get("category", []):
                            if isinstance(cat, dict) and "coding" in cat:
                                for coding in cat["coding"]:
                                    if coding.get("code") == "problem-list-item":
                                        has_problem_category = True
                                        break

                        # Add problem-list-item category if not present
                        if not has_problem_category:
                            condition_dict["category"].append({
                                "coding": [
                                    {
                                        "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                                        "code": "problem-list-item",
                                        "display": "Problem List Item"
                                    }
                                ],
                                "text": "Problem List Item"
                            })

                    response = await client.post(
                        f"{settings.FHIR_SERVICE_URL}/fhir/Condition",
                        json=condition_dict,
                        headers={"Authorization": auth_header}
                    )

                response.raise_for_status()
                problem_data = response.json()
                print(f"Problem data from response: {problem_data}")
                try:
                    result = ProblemListItem(**problem_data)
                    print(f"Created ProblemListItem: {result}")
                    return result
                except Exception as e:
                    print(f"Error creating ProblemListItem object: {str(e)}")
                    print(f"Problem data: {problem_data}")
                    return None
        except Exception as e:
            print(f"Error creating problem: {str(e)}")
            return None

    @strawberry.mutation
    async def create_diagnosis(self, info, condition_data: ConditionInput) -> Optional[Diagnosis]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        condition_dict = {k: v for k, v in condition_data.__dict__.items() if v is not None}

        # Add resourceType
        condition_dict["resourceType"] = "Condition"

        # Create diagnosis in Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Try to create in Condition service first
                url = f"{settings.CONDITION_SERVICE_URL}/conditions/diagnoses"
                print(f"Sending request to: {url}")

                # Ensure category is properly set for the condition service
                request_data = condition_dict.copy()
                if "category" not in request_data or not request_data["category"]:
                    request_data["category"] = [{
                        "coding": [
                            {
                                "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                                "code": "encounter-diagnosis",
                                "display": "Encounter Diagnosis"
                            }
                        ],
                        "text": "Encounter Diagnosis"
                    }]

                print(f"Request data: {request_data}")
                response = await client.post(
                    url,
                    json=request_data,
                    headers={"Authorization": auth_header}
                )
                print(f"Response status: {response.status_code}")
                print(f"Response body: {response.text}")

                # If Condition service fails, fallback to FHIR service with diagnosis category
                if response.status_code != 201 and response.status_code != 200:
                    print(f"Condition service failed with status {response.status_code}, falling back to FHIR service")

                    # Ensure the category is set to encounter-diagnosis
                    if "category" not in condition_dict:
                        condition_dict["category"] = []

                    # Check if encounter-diagnosis category already exists
                    has_diagnosis_category = False
                    for cat in condition_dict.get("category", []):
                        if isinstance(cat, dict) and "coding" in cat:
                            for coding in cat["coding"]:
                                if coding.get("code") == "encounter-diagnosis":
                                    has_diagnosis_category = True
                                    break

                    # Add encounter-diagnosis category if not present
                    if not has_diagnosis_category:
                        condition_dict["category"].append({
                            "coding": [
                                {
                                    "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                                    "code": "encounter-diagnosis",
                                    "display": "Encounter Diagnosis"
                                }
                            ],
                            "text": "Encounter Diagnosis"
                        })

                    response = await client.post(
                        f"{settings.FHIR_SERVICE_URL}/fhir/Condition",
                        json=condition_dict,
                        headers={"Authorization": auth_header}
                    )

                response.raise_for_status()
                diagnosis_data = response.json()
                return Diagnosis(**diagnosis_data)
        except Exception as e:
            print(f"Error creating diagnosis: {str(e)}")
            return None

    @strawberry.mutation
    async def create_health_concern(self, info, condition_data: ConditionInput) -> Optional[HealthConcern]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        condition_dict = {k: v for k, v in condition_data.__dict__.items() if v is not None}

        # Add resourceType
        condition_dict["resourceType"] = "Condition"

        # Create health concern in Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Try to create in Condition service first
                response = await client.post(
                    f"{settings.CONDITION_SERVICE_URL}/conditions/health-concerns",
                    json=condition_dict,
                    headers={"Authorization": auth_header}
                )

                # If Condition service fails, fallback to FHIR service with health concern category
                if response.status_code != 201 and response.status_code != 200:
                    print(f"Condition service failed with status {response.status_code}, falling back to FHIR service")

                    # Ensure the category is set to health-concern
                    if "category" not in condition_dict:
                        condition_dict["category"] = []

                    # Check if health-concern category already exists
                    has_health_concern_category = False
                    for cat in condition_dict.get("category", []):
                        if isinstance(cat, dict) and "coding" in cat:
                            for coding in cat["coding"]:
                                if coding.get("code") == "health-concern":
                                    has_health_concern_category = True
                                    break

                    # Add health-concern category if not present
                    if not has_health_concern_category:
                        condition_dict["category"].append({
                            "coding": [
                                {
                                    "system": "http://terminology.hl7.org/CodeSystem/condition-category",
                                    "code": "health-concern",
                                    "display": "Health Concern"
                                }
                            ],
                            "text": "Health Concern"
                        })

                    response = await client.post(
                        f"{settings.FHIR_SERVICE_URL}/fhir/Condition",
                        json=condition_dict,
                        headers={"Authorization": auth_header}
                    )

                response.raise_for_status()
                concern_data = response.json()
                return HealthConcern(**concern_data)
        except Exception as e:
            print(f"Error creating health concern: {str(e)}")
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
                    f"{settings.FHIR_SERVICE_URL}/fhir/MedicationRequest",
                    json=medication_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                medication_data = response.json()
                return MedicationRequest(**medication_data)
        except Exception as e:
            print(f"Error creating medication request: {str(e)}")
            return None

    @strawberry.mutation
    async def create_diagnostic_report(self, info, report_data: DiagnosticReportInput) -> Optional[DiagnosticReport]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        report_dict = {k: v for k, v in report_data.__dict__.items() if v is not None}

        # Add resourceType
        report_dict["resourceType"] = "DiagnosticReport"

        # Create diagnostic report in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{settings.FHIR_SERVICE_URL}/fhir/DiagnosticReport",
                    json=report_dict,
                    headers={"Authorization": auth_header}
                )

                response.raise_for_status()
                report_data = response.json()
                return DiagnosticReport(**report_data)
        except Exception as e:
            print(f"Error creating diagnostic report: {str(e)}")
            return None

    @strawberry.mutation
    async def create_observation(self, info, input: CreateObservationInput) -> CompleteObservation:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Convert input to dictionary
        observation_data = {
            "status": input.status,
            "category": input.category,
            "code": {
                "system": input.code.system,
                "code": input.code.code,
                "display": input.code.display
            },
            "subject": {
                "reference": input.subject.reference
            }
        }

        # Validate patient ID if it's in the format "Patient/{id}"
        if input.subject.reference.startswith("Patient/"):
            patient_id = input.subject.reference.split("/")[1]
            # Check if patient exists in Patient service
            try:
                async with httpx.AsyncClient() as client:
                    patient_response = await client.get(
                        f"{settings.PATIENT_SERVICE_URL}/patients/{patient_id}",
                        headers={"Authorization": auth_header}
                    )

                    # If patient not found in Patient service, check FHIR service
                    if patient_response.status_code != 200:
                        patient_response = await client.get(
                            f"{settings.FHIR_SERVICE_URL}/fhir/Patient/{patient_id}",
                            headers={"Authorization": auth_header}
                        )

                    # If patient not found in either service, raise exception
                    if patient_response.status_code != 200:
                        raise Exception(f"Patient with ID {patient_id} not found")
            except Exception as e:
                print(f"Error validating patient: {str(e)}")
                # Continue with the request even if patient validation fails

        # Add optional fields if provided
        if input.effective_datetime:
            observation_data["effective_datetime"] = input.effective_datetime

        if input.value_quantity:
            observation_data["value_quantity"] = {
                "value": input.value_quantity.value,
                "unit": input.value_quantity.unit,
                "system": input.value_quantity.system,
                "code": input.value_quantity.code
            }

        if input.interpretation:
            observation_data["interpretation"] = [
                {
                    "system": interp.system,
                    "code": interp.code,
                    "display": interp.display
                }
                for interp in input.interpretation
            ]

        if input.reference_range:
            observation_data["reference_range"] = []
            for range_item in input.reference_range:
                range_dict = {}
                if range_item.low:
                    range_dict["low"] = {
                        "value": range_item.low.value,
                        "unit": range_item.low.unit,
                        "system": range_item.low.system,
                        "code": range_item.low.code
                    }
                if range_item.high:
                    range_dict["high"] = {
                        "value": range_item.high.value,
                        "unit": range_item.high.unit,
                        "system": range_item.high.system,
                        "code": range_item.high.code
                    }
                if range_item.text:
                    range_dict["text"] = range_item.text
                observation_data["reference_range"].append(range_dict)

        # Create observation in Observation service
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{settings.OBSERVATION_SERVICE_URL}/observations",
                json=observation_data,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 201:
                raise Exception(f"Failed to create observation: {response.text}")

            created_observation = response.json()

            # Create CompleteObservation object
            try:
                # Create code object if it exists
                code_obj = None
                if 'code' in created_observation and isinstance(created_observation['code'], dict):
                    code_obj = ObservationCode(
                        system=created_observation['code'].get('system', ''),
                        code=created_observation['code'].get('code', ''),
                        display=created_observation['code'].get('display', '')
                    )

                # Create subject object if it exists
                subject_obj = None
                if 'subject' in created_observation and isinstance(created_observation['subject'], dict):
                    subject_obj = ObservationSubject(
                        reference=created_observation['subject'].get('reference', '')
                    )

                # Create value quantity object if it exists
                value_quantity_obj = None
                if 'value_quantity' in created_observation and isinstance(created_observation['value_quantity'], dict):
                    vq = created_observation['value_quantity']
                    value_quantity_obj = ObservationValueQuantity(
                        value=float(vq.get('value', 0)),
                        unit=vq.get('unit', ''),
                        system=vq.get('system', ''),
                        code=vq.get('code', '')
                    )

                # Create interpretation objects if they exist
                interpretation_objs = []
                if 'interpretation' in created_observation and isinstance(created_observation['interpretation'], list):
                    for interp in created_observation['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_objs.append(ObservationInterpretation(
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            ))

                # Create reference range objects if they exist
                reference_range_objs = []
                if 'reference_range' in created_observation and isinstance(created_observation['reference_range'], list):
                    for range_item in created_observation['reference_range']:
                        if isinstance(range_item, dict):
                            low = None
                            high = None
                            if 'low' in range_item and isinstance(range_item['low'], dict):
                                low_dict = range_item['low']
                                low = ObservationReferenceRangeQuantity(
                                    value=float(low_dict.get('value', 0)),
                                    unit=low_dict.get('unit', ''),
                                    system=low_dict.get('system', ''),
                                    code=low_dict.get('code', '')
                                )
                            if 'high' in range_item and isinstance(range_item['high'], dict):
                                high_dict = range_item['high']
                                high = ObservationReferenceRangeQuantity(
                                    value=float(high_dict.get('value', 0)),
                                    unit=high_dict.get('unit', ''),
                                    system=high_dict.get('system', ''),
                                    code=high_dict.get('code', '')
                                )
                            reference_range_objs.append(ObservationReferenceRange(
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            ))

                # Create complete observation object
                return CompleteObservation(
                    id=str(created_observation.get('_id', created_observation.get('id', ''))),
                    status=created_observation.get('status', ''),
                    category=created_observation.get('category', ''),
                    type=created_observation.get('category', 'observation'),  # Default to 'observation' if category not specified
                    code=code_obj,
                    subject=subject_obj,
                    effective_datetime=created_observation.get('effective_datetime', created_observation.get('effectiveDateTime', None)),
                    value_quantity=value_quantity_obj,
                    interpretation=interpretation_objs,
                    reference_range=reference_range_objs
                )
            except Exception as e:
                print(f"Error processing created observation: {str(e)}")
                print(f"Created observation data: {created_observation}")
                raise Exception(f"Error processing created observation: {str(e)}")

    @strawberry.mutation
    async def update_observation(self, info, input: UpdateObservationInput) -> CompleteObservation:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Convert input to dictionary
        observation_data = {}

        # Add fields if provided
        if input.status:
            observation_data["status"] = input.status
        if input.category:
            observation_data["category"] = input.category
        if input.code:
            observation_data["code"] = {
                "system": input.code.system,
                "code": input.code.code,
                "display": input.code.display
            }
        if input.subject:
            observation_data["subject"] = {
                "reference": input.subject.reference
            }

            # Validate patient ID if it's in the format "Patient/{id}"
            if input.subject.reference.startswith("Patient/"):
                patient_id = input.subject.reference.split("/")[1]
                # Check if patient exists in Patient service
                try:
                    async with httpx.AsyncClient() as client:
                        patient_response = await client.get(
                            f"{settings.PATIENT_SERVICE_URL}/api/patients/{patient_id}",
                            headers={"Authorization": auth_header}
                        )

                        # If patient not found in Patient service, check FHIR service
                        if patient_response.status_code != 200:
                            patient_response = await client.get(
                                f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}",
                                headers={"Authorization": auth_header}
                            )

                        # If patient not found in either service, raise exception
                        if patient_response.status_code != 200:
                            raise Exception(f"Patient with ID {patient_id} not found")
                except Exception as e:
                    print(f"Error validating patient: {str(e)}")
                    # Continue with the request even if patient validation fails
        if input.effective_datetime:
            observation_data["effective_datetime"] = input.effective_datetime
        if input.value_quantity:
            observation_data["value_quantity"] = {
                "value": input.value_quantity.value,
                "unit": input.value_quantity.unit,
                "system": input.value_quantity.system,
                "code": input.value_quantity.code
            }
        if input.interpretation:
            observation_data["interpretation"] = [
                {
                    "system": interp.system,
                    "code": interp.code,
                    "display": interp.display
                }
                for interp in input.interpretation
            ]
        if input.reference_range:
            observation_data["reference_range"] = []
            for range_item in input.reference_range:
                range_dict = {}
                if range_item.low:
                    range_dict["low"] = {
                        "value": range_item.low.value,
                        "unit": range_item.low.unit,
                        "system": range_item.low.system,
                        "code": range_item.low.code
                    }
                if range_item.high:
                    range_dict["high"] = {
                        "value": range_item.high.value,
                        "unit": range_item.high.unit,
                        "system": range_item.high.system,
                        "code": range_item.high.code
                    }
                if range_item.text:
                    range_dict["text"] = range_item.text
                observation_data["reference_range"].append(range_dict)

        # Update observation in Observation service
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations/{input.id}",
                json=observation_data,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                raise Exception(f"Failed to update observation: {response.text}")

            updated_observation = response.json()

            # Create CompleteObservation object
            try:
                # Create code object if it exists
                code_obj = None
                if 'code' in updated_observation and isinstance(updated_observation['code'], dict):
                    code_obj = ObservationCode(
                        system=updated_observation['code'].get('system', ''),
                        code=updated_observation['code'].get('code', ''),
                        display=updated_observation['code'].get('display', '')
                    )

                # Create subject object if it exists
                subject_obj = None
                if 'subject' in updated_observation and isinstance(updated_observation['subject'], dict):
                    subject_obj = ObservationSubject(
                        reference=updated_observation['subject'].get('reference', '')
                    )

                # Create value quantity object if it exists
                value_quantity_obj = None
                if 'value_quantity' in updated_observation and isinstance(updated_observation['value_quantity'], dict):
                    vq = updated_observation['value_quantity']
                    value_quantity_obj = ObservationValueQuantity(
                        value=float(vq.get('value', 0)),
                        unit=vq.get('unit', ''),
                        system=vq.get('system', ''),
                        code=vq.get('code', '')
                    )

                # Create interpretation objects if they exist
                interpretation_objs = []
                if 'interpretation' in updated_observation and isinstance(updated_observation['interpretation'], list):
                    for interp in updated_observation['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_objs.append(ObservationInterpretation(
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            ))

                # Create reference range objects if they exist
                reference_range_objs = []
                if 'reference_range' in updated_observation and isinstance(updated_observation['reference_range'], list):
                    for range_item in updated_observation['reference_range']:
                        if isinstance(range_item, dict):
                            low = None
                            high = None
                            if 'low' in range_item and isinstance(range_item['low'], dict):
                                low_dict = range_item['low']
                                low = ObservationReferenceRangeQuantity(
                                    value=float(low_dict.get('value', 0)),
                                    unit=low_dict.get('unit', ''),
                                    system=low_dict.get('system', ''),
                                    code=low_dict.get('code', '')
                                )
                            if 'high' in range_item and isinstance(range_item['high'], dict):
                                high_dict = range_item['high']
                                high = ObservationReferenceRangeQuantity(
                                    value=float(high_dict.get('value', 0)),
                                    unit=high_dict.get('unit', ''),
                                    system=high_dict.get('system', ''),
                                    code=high_dict.get('code', '')
                                )
                            reference_range_objs.append(ObservationReferenceRange(
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            ))

                # Create complete observation object
                return CompleteObservation(
                    id=str(updated_observation.get('_id', updated_observation.get('id', ''))),
                    status=updated_observation.get('status', ''),
                    category=updated_observation.get('category', ''),
                    type=updated_observation.get('category', 'observation'),  # Default to 'observation' if category not specified
                    code=code_obj,
                    subject=subject_obj,
                    effective_datetime=updated_observation.get('effective_datetime', updated_observation.get('effectiveDateTime', None)),
                    value_quantity=value_quantity_obj,
                    interpretation=interpretation_objs,
                    reference_range=reference_range_objs
                )
            except Exception as e:
                print(f"Error processing updated observation: {str(e)}")
                print(f"Updated observation data: {updated_observation}")
                raise Exception(f"Error processing updated observation: {str(e)}")

    @strawberry.mutation
    async def delete_observation(self, info, id: str) -> bool:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Delete observation from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.delete(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations/{id}",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 204:
                raise Exception(f"Failed to delete observation: {response.text}")

            return True

    @strawberry.mutation
    async def create_vital_sign(self, info, input: CreateVitalSignInput) -> VitalSign:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Convert input to dictionary
        observation_data = {
            "status": input.status,
            "category": "vital-signs",
            "code": {
                "system": input.code.system,
                "code": input.code.code,
                "display": input.code.display
            },
            "subject": {
                "reference": input.subject.reference
            }
        }

        # Add optional fields if provided
        if input.effective_datetime:
            observation_data["effective_datetime"] = input.effective_datetime

        if input.value_quantity:
            observation_data["value_quantity"] = {
                "value": input.value_quantity.value,
                "unit": input.value_quantity.unit,
                "system": input.value_quantity.system,
                "code": input.value_quantity.code
            }

        if input.interpretation:
            observation_data["interpretation"] = [
                {
                    "system": interp.system,
                    "code": interp.code,
                    "display": interp.display
                }
                for interp in input.interpretation
            ]

        if input.reference_range:
            observation_data["reference_range"] = []
            for range_item in input.reference_range:
                range_dict = {}
                if range_item.low:
                    range_dict["low"] = {
                        "value": range_item.low.value,
                        "unit": range_item.low.unit,
                        "system": range_item.low.system,
                        "code": range_item.low.code
                    }
                if range_item.high:
                    range_dict["high"] = {
                        "value": range_item.high.value,
                        "unit": range_item.high.unit,
                        "system": range_item.high.system,
                        "code": range_item.high.code
                    }
                if range_item.text:
                    range_dict["text"] = range_item.text
                observation_data["reference_range"].append(range_dict)

        # Create vital sign in Observation service
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                json=observation_data,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 201:
                raise Exception(f"Failed to create vital sign: {response.text}")

            created_observation = response.json()

            # Create VitalSign object
            try:
                # Create code object if it exists
                code_obj = None
                if 'code' in created_observation and isinstance(created_observation['code'], dict):
                    code_obj = ObservationCode(
                        system=created_observation['code'].get('system', ''),
                        code=created_observation['code'].get('code', ''),
                        display=created_observation['code'].get('display', '')
                    )

                # Create subject object if it exists
                subject_obj = None
                if 'subject' in created_observation and isinstance(created_observation['subject'], dict):
                    subject_obj = ObservationSubject(
                        reference=created_observation['subject'].get('reference', '')
                    )

                # Create value quantity object if it exists
                value_quantity_obj = None
                if 'value_quantity' in created_observation and isinstance(created_observation['value_quantity'], dict):
                    vq = created_observation['value_quantity']
                    value_quantity_obj = ObservationValueQuantity(
                        value=float(vq.get('value', 0)),
                        unit=vq.get('unit', ''),
                        system=vq.get('system', ''),
                        code=vq.get('code', '')
                    )

                # Create interpretation objects if they exist
                interpretation_objs = []
                if 'interpretation' in created_observation and isinstance(created_observation['interpretation'], list):
                    for interp in created_observation['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_objs.append(ObservationInterpretation(
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            ))

                # Create reference range objects if they exist
                reference_range_objs = []
                if 'reference_range' in created_observation and isinstance(created_observation['reference_range'], list):
                    for range_item in created_observation['reference_range']:
                        if isinstance(range_item, dict):
                            low = None
                            high = None
                            if 'low' in range_item and isinstance(range_item['low'], dict):
                                low_dict = range_item['low']
                                low = ObservationReferenceRangeQuantity(
                                    value=float(low_dict.get('value', 0)),
                                    unit=low_dict.get('unit', ''),
                                    system=low_dict.get('system', ''),
                                    code=low_dict.get('code', '')
                                )
                            if 'high' in range_item and isinstance(range_item['high'], dict):
                                high_dict = range_item['high']
                                high = ObservationReferenceRangeQuantity(
                                    value=float(high_dict.get('value', 0)),
                                    unit=high_dict.get('unit', ''),
                                    system=high_dict.get('system', ''),
                                    code=high_dict.get('code', '')
                                )
                            reference_range_objs.append(ObservationReferenceRange(
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            ))

                # Create vital sign object
                return VitalSign(
                    id=str(created_observation.get('_id', created_observation.get('id', ''))),
                    status=created_observation.get('status', ''),
                    category=created_observation.get('category', ''),
                    code=code_obj,
                    subject=subject_obj,
                    effective_datetime=created_observation.get('effective_datetime', created_observation.get('effectiveDateTime', None)),
                    value_quantity=value_quantity_obj,
                    interpretation=interpretation_objs,
                    reference_range=reference_range_objs
                )
            except Exception as e:
                print(f"Error processing created vital sign: {str(e)}")
                print(f"Created vital sign data: {created_observation}")
                raise Exception(f"Error processing created vital sign: {str(e)}")

    @strawberry.mutation
    async def update_vital_sign(self, info, input: UpdateVitalSignInput) -> VitalSign:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Convert input to dictionary
        observation_data = {}

        # Add fields if provided
        if input.status:
            observation_data["status"] = input.status
        if input.code:
            observation_data["code"] = {
                "system": input.code.system,
                "code": input.code.code,
                "display": input.code.display
            }
        if input.subject:
            observation_data["subject"] = {
                "reference": input.subject.reference
            }
        if input.effective_datetime:
            observation_data["effective_datetime"] = input.effective_datetime
        if input.value_quantity:
            observation_data["value_quantity"] = {
                "value": input.value_quantity.value,
                "unit": input.value_quantity.unit,
                "system": input.value_quantity.system,
                "code": input.value_quantity.code
            }
        if input.interpretation:
            observation_data["interpretation"] = [
                {
                    "system": interp.system,
                    "code": interp.code,
                    "display": interp.display
                }
                for interp in input.interpretation
            ]
        if input.reference_range:
            observation_data["reference_range"] = []
            for range_item in input.reference_range:
                range_dict = {}
                if range_item.low:
                    range_dict["low"] = {
                        "value": range_item.low.value,
                        "unit": range_item.low.unit,
                        "system": range_item.low.system,
                        "code": range_item.low.code
                    }
                if range_item.high:
                    range_dict["high"] = {
                        "value": range_item.high.value,
                        "unit": range_item.high.unit,
                        "system": range_item.high.system,
                        "code": range_item.high.code
                    }
                if range_item.text:
                    range_dict["text"] = range_item.text
                observation_data["reference_range"].append(range_dict)

        # Update vital sign in Observation service
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations/{input.id}",
                json=observation_data,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                raise Exception(f"Failed to update vital sign: {response.text}")

            updated_observation = response.json()

            # Create VitalSign object
            try:
                # Create code object if it exists
                code_obj = None
                if 'code' in updated_observation and isinstance(updated_observation['code'], dict):
                    code_obj = ObservationCode(
                        system=updated_observation['code'].get('system', ''),
                        code=updated_observation['code'].get('code', ''),
                        display=updated_observation['code'].get('display', '')
                    )

                # Create subject object if it exists
                subject_obj = None
                if 'subject' in updated_observation and isinstance(updated_observation['subject'], dict):
                    subject_obj = ObservationSubject(
                        reference=updated_observation['subject'].get('reference', '')
                    )

                # Create value quantity object if it exists
                value_quantity_obj = None
                if 'value_quantity' in updated_observation and isinstance(updated_observation['value_quantity'], dict):
                    vq = updated_observation['value_quantity']
                    value_quantity_obj = ObservationValueQuantity(
                        value=float(vq.get('value', 0)),
                        unit=vq.get('unit', ''),
                        system=vq.get('system', ''),
                        code=vq.get('code', '')
                    )

                # Create interpretation objects if they exist
                interpretation_objs = []
                if 'interpretation' in updated_observation and isinstance(updated_observation['interpretation'], list):
                    for interp in updated_observation['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_objs.append(ObservationInterpretation(
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            ))

                # Create reference range objects if they exist
                reference_range_objs = []
                if 'reference_range' in updated_observation and isinstance(updated_observation['reference_range'], list):
                    for range_item in updated_observation['reference_range']:
                        if isinstance(range_item, dict):
                            low = None
                            high = None
                            if 'low' in range_item and isinstance(range_item['low'], dict):
                                low_dict = range_item['low']
                                low = ObservationReferenceRangeQuantity(
                                    value=float(low_dict.get('value', 0)),
                                    unit=low_dict.get('unit', ''),
                                    system=low_dict.get('system', ''),
                                    code=low_dict.get('code', '')
                                )
                            if 'high' in range_item and isinstance(range_item['high'], dict):
                                high_dict = range_item['high']
                                high = ObservationReferenceRangeQuantity(
                                    value=float(high_dict.get('value', 0)),
                                    unit=high_dict.get('unit', ''),
                                    system=high_dict.get('system', ''),
                                    code=high_dict.get('code', '')
                                )
                            reference_range_objs.append(ObservationReferenceRange(
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            ))

                # Create vital sign object
                return VitalSign(
                    id=str(updated_observation.get('_id', updated_observation.get('id', ''))),
                    status=updated_observation.get('status', ''),
                    category=updated_observation.get('category', ''),
                    code=code_obj,
                    subject=subject_obj,
                    effective_datetime=updated_observation.get('effective_datetime', updated_observation.get('effectiveDateTime', None)),
                    value_quantity=value_quantity_obj,
                    interpretation=interpretation_objs,
                    reference_range=reference_range_objs
                )
            except Exception as e:
                print(f"Error processing updated vital sign: {str(e)}")
                print(f"Updated vital sign data: {updated_observation}")
                raise Exception(f"Error processing updated vital sign: {str(e)}")

    @strawberry.mutation
    async def create_physical_measurement(self, info, input: CreatePhysicalMeasurementInput) -> PhysicalMeasurement:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Convert input to dictionary
        observation_data = {
            "status": input.status,
            "category": "vital-signs",  # Using vital-signs as the category for physical measurements
            "code": {
                "system": input.code.system,
                "code": input.code.code,
                "display": input.code.display
            },
            "subject": {
                "reference": input.subject.reference
            }
        }

        # Add optional fields if provided
        if input.effective_datetime:
            observation_data["effective_datetime"] = input.effective_datetime

        if input.value_quantity:
            observation_data["value_quantity"] = {
                "value": input.value_quantity.value,
                "unit": input.value_quantity.unit,
                "system": input.value_quantity.system,
                "code": input.value_quantity.code
            }

        if input.interpretation:
            observation_data["interpretation"] = [
                {
                    "system": interp.system,
                    "code": interp.code,
                    "display": interp.display
                }
                for interp in input.interpretation
            ]

        if input.reference_range:
            observation_data["reference_range"] = []
            for range_item in input.reference_range:
                range_dict = {}
                if range_item.low:
                    range_dict["low"] = {
                        "value": range_item.low.value,
                        "unit": range_item.low.unit,
                        "system": range_item.low.system,
                        "code": range_item.low.code
                    }
                if range_item.high:
                    range_dict["high"] = {
                        "value": range_item.high.value,
                        "unit": range_item.high.unit,
                        "system": range_item.high.system,
                        "code": range_item.high.code
                    }
                if range_item.text:
                    range_dict["text"] = range_item.text
                observation_data["reference_range"].append(range_dict)

        # Create physical measurement in Observation service
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                json=observation_data,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 201:
                raise Exception(f"Failed to create physical measurement: {response.text}")

            created_observation = response.json()

            # Create PhysicalMeasurement object
            try:
                # Create code object if it exists
                code_obj = None
                if 'code' in created_observation and isinstance(created_observation['code'], dict):
                    code_obj = ObservationCode(
                        system=created_observation['code'].get('system', ''),
                        code=created_observation['code'].get('code', ''),
                        display=created_observation['code'].get('display', '')
                    )

                # Create subject object if it exists
                subject_obj = None
                if 'subject' in created_observation and isinstance(created_observation['subject'], dict):
                    subject_obj = ObservationSubject(
                        reference=created_observation['subject'].get('reference', '')
                    )

                # Create value quantity object if it exists
                value_quantity_obj = None
                if 'value_quantity' in created_observation and isinstance(created_observation['value_quantity'], dict):
                    vq = created_observation['value_quantity']
                    value_quantity_obj = ObservationValueQuantity(
                        value=float(vq.get('value', 0)),
                        unit=vq.get('unit', ''),
                        system=vq.get('system', ''),
                        code=vq.get('code', '')
                    )

                # Create interpretation objects if they exist
                interpretation_objs = []
                if 'interpretation' in created_observation and isinstance(created_observation['interpretation'], list):
                    for interp in created_observation['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_objs.append(ObservationInterpretation(
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            ))

                # Create reference range objects if they exist
                reference_range_objs = []
                if 'reference_range' in created_observation and isinstance(created_observation['reference_range'], list):
                    for range_item in created_observation['reference_range']:
                        if isinstance(range_item, dict):
                            low = None
                            high = None
                            if 'low' in range_item and isinstance(range_item['low'], dict):
                                low_dict = range_item['low']
                                low = ObservationReferenceRangeQuantity(
                                    value=float(low_dict.get('value', 0)),
                                    unit=low_dict.get('unit', ''),
                                    system=low_dict.get('system', ''),
                                    code=low_dict.get('code', '')
                                )
                            if 'high' in range_item and isinstance(range_item['high'], dict):
                                high_dict = range_item['high']
                                high = ObservationReferenceRangeQuantity(
                                    value=float(high_dict.get('value', 0)),
                                    unit=high_dict.get('unit', ''),
                                    system=high_dict.get('system', ''),
                                    code=high_dict.get('code', '')
                                )
                            reference_range_objs.append(ObservationReferenceRange(
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            ))

                # Create physical measurement object
                return PhysicalMeasurement(
                    id=str(created_observation.get('_id', created_observation.get('id', ''))),
                    status=created_observation.get('status', ''),
                    category=created_observation.get('category', ''),
                    code=code_obj,
                    subject=subject_obj,
                    effective_datetime=created_observation.get('effective_datetime', created_observation.get('effectiveDateTime', None)),
                    value_quantity=value_quantity_obj,
                    interpretation=interpretation_objs,
                    reference_range=reference_range_objs
                )
            except Exception as e:
                print(f"Error processing created physical measurement: {str(e)}")
                print(f"Created physical measurement data: {created_observation}")
                raise Exception(f"Error processing created physical measurement: {str(e)}")

    @strawberry.mutation
    async def update_physical_measurement(self, info, input: UpdatePhysicalMeasurementInput) -> PhysicalMeasurement:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            raise Exception("Authentication required")

        # Convert input to dictionary
        observation_data = {}

        # Add fields if provided
        if input.status:
            observation_data["status"] = input.status
        if input.code:
            observation_data["code"] = {
                "system": input.code.system,
                "code": input.code.code,
                "display": input.code.display
            }
        if input.subject:
            observation_data["subject"] = {
                "reference": input.subject.reference
            }
        if input.effective_datetime:
            observation_data["effective_datetime"] = input.effective_datetime
        if input.value_quantity:
            observation_data["value_quantity"] = {
                "value": input.value_quantity.value,
                "unit": input.value_quantity.unit,
                "system": input.value_quantity.system,
                "code": input.value_quantity.code
            }
        if input.interpretation:
            observation_data["interpretation"] = [
                {
                    "system": interp.system,
                    "code": interp.code,
                    "display": interp.display
                }
                for interp in input.interpretation
            ]
        if input.reference_range:
            observation_data["reference_range"] = []
            for range_item in input.reference_range:
                range_dict = {}
                if range_item.low:
                    range_dict["low"] = {
                        "value": range_item.low.value,
                        "unit": range_item.low.unit,
                        "system": range_item.low.system,
                        "code": range_item.low.code
                    }
                if range_item.high:
                    range_dict["high"] = {
                        "value": range_item.high.value,
                        "unit": range_item.high.unit,
                        "system": range_item.high.system,
                        "code": range_item.high.code
                    }
                if range_item.text:
                    range_dict["text"] = range_item.text
                observation_data["reference_range"].append(range_dict)

        # Update physical measurement in Observation service
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations/{input.id}",
                json=observation_data,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                raise Exception(f"Failed to update physical measurement: {response.text}")

            updated_observation = response.json()

            # Create PhysicalMeasurement object
            try:
                # Create code object if it exists
                code_obj = None
                if 'code' in updated_observation and isinstance(updated_observation['code'], dict):
                    code_obj = ObservationCode(
                        system=updated_observation['code'].get('system', ''),
                        code=updated_observation['code'].get('code', ''),
                        display=updated_observation['code'].get('display', '')
                    )

                # Create subject object if it exists
                subject_obj = None
                if 'subject' in updated_observation and isinstance(updated_observation['subject'], dict):
                    subject_obj = ObservationSubject(
                        reference=updated_observation['subject'].get('reference', '')
                    )

                # Create value quantity object if it exists
                value_quantity_obj = None
                if 'value_quantity' in updated_observation and isinstance(updated_observation['value_quantity'], dict):
                    vq = updated_observation['value_quantity']
                    value_quantity_obj = ObservationValueQuantity(
                        value=float(vq.get('value', 0)),
                        unit=vq.get('unit', ''),
                        system=vq.get('system', ''),
                        code=vq.get('code', '')
                    )

                # Create interpretation objects if they exist
                interpretation_objs = []
                if 'interpretation' in updated_observation and isinstance(updated_observation['interpretation'], list):
                    for interp in updated_observation['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_objs.append(ObservationInterpretation(
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            ))

                # Create reference range objects if they exist
                reference_range_objs = []
                if 'reference_range' in updated_observation and isinstance(updated_observation['reference_range'], list):
                    for range_item in updated_observation['reference_range']:
                        if isinstance(range_item, dict):
                            low = None
                            high = None
                            if 'low' in range_item and isinstance(range_item['low'], dict):
                                low_dict = range_item['low']
                                low = ObservationReferenceRangeQuantity(
                                    value=float(low_dict.get('value', 0)),
                                    unit=low_dict.get('unit', ''),
                                    system=low_dict.get('system', ''),
                                    code=low_dict.get('code', '')
                                )
                            if 'high' in range_item and isinstance(range_item['high'], dict):
                                high_dict = range_item['high']
                                high = ObservationReferenceRangeQuantity(
                                    value=float(high_dict.get('value', 0)),
                                    unit=high_dict.get('unit', ''),
                                    system=high_dict.get('system', ''),
                                    code=high_dict.get('code', '')
                                )
                            reference_range_objs.append(ObservationReferenceRange(
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            ))

                # Create physical measurement object
                return PhysicalMeasurement(
                    id=str(updated_observation.get('_id', updated_observation.get('id', ''))),
                    status=updated_observation.get('status', ''),
                    category=updated_observation.get('category', ''),
                    code=code_obj,
                    subject=subject_obj,
                    effective_datetime=updated_observation.get('effective_datetime', updated_observation.get('effectiveDateTime', None)),
                    value_quantity=value_quantity_obj,
                    interpretation=interpretation_objs,
                    reference_range=reference_range_objs
                )
            except Exception as e:
                print(f"Error processing updated physical measurement: {str(e)}")
                print(f"Updated physical measurement data: {updated_observation}")
                raise Exception(f"Error processing updated physical measurement: {str(e)}")

    @strawberry.mutation
    async def create_medication(self, info, medication_data: MedicationInput) -> Optional[Medication]:
        """Create a new medication"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_dict = {}

        # Handle code
        if hasattr(medication_data, 'code'):
            medication_dict["code"] = {
                "coding": [],
                "text": medication_data.code.text or ""
            }
            for coding in medication_data.code.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_dict["code"]["coding"].append(coding_dict)

        # Handle form
        if hasattr(medication_data, 'form') and medication_data.form:
            medication_dict["form"] = {
                "coding": [],
                "text": medication_data.form.text or ""
            }
            for coding in medication_data.form.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_dict["form"]["coding"].append(coding_dict)

        # Handle amount
        if hasattr(medication_data, 'amount') and medication_data.amount:
            medication_dict["amount"] = {
                "value": medication_data.amount.value,
                "unit": medication_data.amount.unit
            }
            if medication_data.amount.system:
                medication_dict["amount"]["system"] = medication_data.amount.system
            if medication_data.amount.code:
                medication_dict["amount"]["code"] = medication_data.amount.code

        # Handle status
        if hasattr(medication_data, 'status') and medication_data.status:
            medication_dict["status"] = medication_data.status

        # Handle ingredient
        if hasattr(medication_data, 'ingredient') and medication_data.ingredient:
            medication_dict["ingredient"] = medication_data.ingredient

        # Handle batch
        if hasattr(medication_data, 'batch') and medication_data.batch:
            medication_dict["batch"] = medication_data.batch

        # Create medication via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medications"
                print(f"Making request to: {url}")

                response = await client.post(
                    url,
                    json=medication_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 201 and response.status_code != 200:
                    print(f"Error creating medication: {response.status_code} - {response.text}")
                    return None

                created_medication = response.json()

                # Convert the REST response to a GraphQL type
                # Handle code
                code_obj = None
                if 'code' in created_medication and isinstance(created_medication['code'], dict):
                    coding_list = []
                    for coding in created_medication['code'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    code_obj = CodeableConcept(
                        coding=coding_list,
                        text=created_medication['code'].get('text', '')
                    )

                # Handle form
                form_obj = None
                if 'form' in created_medication and isinstance(created_medication['form'], dict):
                    coding_list = []
                    for coding in created_medication['form'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', ''),
                                version=coding.get('version'),
                                user_selected=coding.get('userSelected')
                            )
                        )
                    form_obj = CodeableConcept(
                        coding=coding_list,
                        text=created_medication['form'].get('text', '')
                    )

                # Handle amount
                amount_obj = None
                if 'amount' in created_medication and isinstance(created_medication['amount'], dict):
                    amount_obj = Quantity(
                        value=float(created_medication['amount'].get('value', 0)),
                        unit=created_medication['amount'].get('unit', ''),
                        system=created_medication['amount'].get('system', ''),
                        code=created_medication['amount'].get('code', '')
                    )

                # Create Medication object
                return Medication(
                    id=created_medication.get('id', ''),
                    resourceType=created_medication.get('resourceType', 'Medication'),
                    status=created_medication.get('status', ''),
                    code=code_obj,
                    form=form_obj,
                    amount=amount_obj,
                    ingredient=created_medication.get('ingredient', []),
                    batch=created_medication.get('batch', '')
                )
        except Exception as e:
            print(f"Error creating medication: {str(e)}")
            return None

    @strawberry.mutation
    async def update_medication(self, info, id: str, medication_data: MedicationInput) -> Optional[Medication]:
        """Update an existing medication"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_dict = {}

        # Handle code
        if hasattr(medication_data, 'code'):
            medication_dict["code"] = {
                "coding": [],
                "text": medication_data.code.text or ""
            }
            for coding in medication_data.code.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_dict["code"]["coding"].append(coding_dict)

        # Handle form
        if hasattr(medication_data, 'form') and medication_data.form:
            medication_dict["form"] = {
                "coding": [],
                "text": medication_data.form.text or ""
            }
            for coding in medication_data.form.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_dict["form"]["coding"].append(coding_dict)

        # Handle amount
        if hasattr(medication_data, 'amount') and medication_data.amount:
            medication_dict["amount"] = {
                "value": medication_data.amount.value,
                "unit": medication_data.amount.unit
            }
            if medication_data.amount.system:
                medication_dict["amount"]["system"] = medication_data.amount.system
            if medication_data.amount.code:
                medication_dict["amount"]["code"] = medication_data.amount.code

        # Handle status
        if hasattr(medication_data, 'status') and medication_data.status:
            medication_dict["status"] = medication_data.status

        # Handle ingredient
        if hasattr(medication_data, 'ingredient') and medication_data.ingredient:
            medication_dict["ingredient"] = medication_data.ingredient

        # Handle batch
        if hasattr(medication_data, 'batch') and medication_data.batch:
            medication_dict["batch"] = medication_data.batch

        # Update medication via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medications/{id}"
                print(f"Making request to: {url}")

                response = await client.put(
                    url,
                    json=medication_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error updating medication: {response.status_code} - {response.text}")
                    return None

                updated_medication = response.json()

                # Convert the REST response to a GraphQL type
                # Handle code
                code_obj = None
                if 'code' in updated_medication and isinstance(updated_medication['code'], dict):
                    coding_list = []
                    for coding in updated_medication['code'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    code_obj = CodeableConcept(
                        coding=coding_list,
                        text=updated_medication['code'].get('text', '')
                    )

                # Handle form
                form_obj = None
                if 'form' in updated_medication and isinstance(updated_medication['form'], dict):
                    coding_list = []
                    for coding in updated_medication['form'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    form_obj = CodeableConcept(
                        coding=coding_list,
                        text=updated_medication['form'].get('text', '')
                    )

                # Handle amount
                amount_obj = None
                if 'amount' in updated_medication and isinstance(updated_medication['amount'], dict):
                    amount_obj = Quantity(
                        value=float(updated_medication['amount'].get('value', 0)),
                        unit=updated_medication['amount'].get('unit', ''),
                        system=updated_medication['amount'].get('system', ''),
                        code=updated_medication['amount'].get('code', '')
                    )

                # Create Medication object
                return Medication(
                    id=updated_medication.get('id', ''),
                    resourceType=updated_medication.get('resourceType', 'Medication'),
                    status=updated_medication.get('status', ''),
                    code=code_obj,
                    form=form_obj,
                    amount=amount_obj,
                    ingredient=updated_medication.get('ingredient', []),
                    batch=updated_medication.get('batch', '')
                )
        except Exception as e:
            print(f"Error updating medication: {str(e)}")
            return None

    @strawberry.mutation
    async def delete_medication(self, info, id: str) -> bool:
        """Delete a medication"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return False

        # Delete medication via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medications/{id}"
                print(f"Making request to: {url}")

                response = await client.delete(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error deleting medication: {response.status_code} - {response.text}")
                    return False

                return True
        except Exception as e:
            print(f"Error deleting medication: {str(e)}")
            return False

    @strawberry.mutation
    async def create_medication_request(self, info, medication_request_data: MedicationRequestInput) -> Optional[MedicationRequest]:
        """Create a new medication request"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_request_dict = {}

        # Handle status
        if hasattr(medication_request_data, 'status'):
            medication_request_dict["status"] = medication_request_data.status

        # Handle intent
        if hasattr(medication_request_data, 'intent'):
            medication_request_dict["intent"] = medication_request_data.intent

        # Handle medicationCodeableConcept
        if hasattr(medication_request_data, 'medicationCodeableConcept'):
            medication_request_dict["medicationCodeableConcept"] = {
                "coding": [],
                "text": medication_request_data.medicationCodeableConcept.text or ""
            }
            for coding in medication_request_data.medicationCodeableConcept.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_request_dict["medicationCodeableConcept"]["coding"].append(coding_dict)

        # Handle subject
        if hasattr(medication_request_data, 'subject'):
            medication_request_dict["subject"] = {
                "reference": medication_request_data.subject.reference
            }
            if medication_request_data.subject.display:
                medication_request_dict["subject"]["display"] = medication_request_data.subject.display

        # Handle authoredOn
        if hasattr(medication_request_data, 'authoredOn') and medication_request_data.authoredOn:
            medication_request_dict["authoredOn"] = medication_request_data.authoredOn

        # Handle requester
        if hasattr(medication_request_data, 'requester') and medication_request_data.requester:
            medication_request_dict["requester"] = {
                "reference": medication_request_data.requester.reference
            }
            if medication_request_data.requester.display:
                medication_request_dict["requester"]["display"] = medication_request_data.requester.display

        # Handle dosageInstruction
        if hasattr(medication_request_data, 'dosageInstruction') and medication_request_data.dosageInstruction:
            medication_request_dict["dosageInstruction"] = []
            for dosage in medication_request_data.dosageInstruction:
                dosage_dict = {}
                if hasattr(dosage, 'text') and dosage.text:
                    dosage_dict["text"] = dosage.text
                if hasattr(dosage, 'timing') and dosage.timing:
                    dosage_dict["timing"] = dosage.timing
                if hasattr(dosage, 'asNeededBoolean'):
                    dosage_dict["asNeededBoolean"] = dosage.asNeededBoolean
                if hasattr(dosage, 'route') and dosage.route:
                    dosage_dict["route"] = {
                        "coding": [],
                        "text": dosage.route.text or ""
                    }
                    for coding in dosage.route.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        dosage_dict["route"]["coding"].append(coding_dict)
                medication_request_dict["dosageInstruction"].append(dosage_dict)

        # Handle note
        if hasattr(medication_request_data, 'note') and medication_request_data.note:
            medication_request_dict["note"] = []
            for note in medication_request_data.note:
                note_dict = {
                    "text": note.text
                }
                if note.authorString:
                    note_dict["authorString"] = note.authorString
                if note.time:
                    note_dict["time"] = note.time
                medication_request_dict["note"].append(note_dict)

        # Add resourceType
        medication_request_dict["resourceType"] = "MedicationRequest"

        # Create medication request via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-requests"
                print(f"Making request to: {url}")

                response = await client.post(
                    url,
                    json=medication_request_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 201 and response.status_code != 200:
                    print(f"Error creating medication request: {response.status_code} - {response.text}")
                    return None

                created_med_request = response.json()

                # Convert the REST response to a GraphQL type
                # Handle medicationCodeableConcept
                med_code_obj = None
                if 'medicationCodeableConcept' in created_med_request and isinstance(created_med_request['medicationCodeableConcept'], dict):
                    coding_list = []
                    for coding in created_med_request['medicationCodeableConcept'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', ''),
                                version=coding.get('version'),
                                user_selected=coding.get('userSelected')
                            )
                        )
                    med_code_obj = CodeableConcept(
                        coding=coding_list,
                        text=created_med_request['medicationCodeableConcept'].get('text', '')
                    )

                # Handle subject
                subject_obj = None
                if 'subject' in created_med_request and isinstance(created_med_request['subject'], dict):
                    subject_obj = Reference(
                        reference=created_med_request['subject'].get('reference', ''),
                        display=created_med_request['subject'].get('display', '')
                    )

                # Handle requester
                requester_obj = None
                if 'requester' in created_med_request and isinstance(created_med_request['requester'], dict):
                    requester_obj = Reference(
                        reference=created_med_request['requester'].get('reference', ''),
                        display=created_med_request['requester'].get('display', '')
                    )

                # Handle dosageInstruction
                dosage_list = None
                if 'dosageInstruction' in created_med_request and isinstance(created_med_request['dosageInstruction'], list):
                    dosage_list = []
                    for dosage in created_med_request['dosageInstruction']:
                        route_obj = None
                        if 'route' in dosage and isinstance(dosage['route'], dict):
                            coding_list = []
                            for coding in dosage['route'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            route_obj = CodeableConcept(
                                coding=coding_list,
                                text=dosage['route'].get('text', '')
                            )

                        dosage_list.append(
                            DosageInstruction(
                                text=dosage.get('text'),
                                timing=dosage.get('timing'),
                                asNeededBoolean=dosage.get('asNeededBoolean'),
                                route=route_obj
                            )
                        )

                # Handle note
                note_list = None
                if 'note' in created_med_request and isinstance(created_med_request['note'], list):
                    note_list = []
                    for note in created_med_request['note']:
                        note_list.append(
                            Annotation(
                                text=note.get('text', ''),
                                authorString=note.get('authorString'),
                                time=note.get('time')
                            )
                        )

                # Create MedicationRequest object
                return MedicationRequest(
                    id=created_med_request.get('id', ''),
                    resourceType=created_med_request.get('resourceType', 'MedicationRequest'),
                    status=created_med_request.get('status', ''),
                    intent=created_med_request.get('intent', ''),
                    medicationCodeableConcept=med_code_obj,
                    subject=subject_obj,
                    authoredOn=created_med_request.get('authoredOn', ''),
                    requester=requester_obj,
                    dosageInstruction=dosage_list,
                    note=note_list
                )
        except Exception as e:
            print(f"Error creating medication request: {str(e)}")
            return None

    @strawberry.mutation
    async def update_medication_request(self, info, id: str, medication_request_data: MedicationRequestInput) -> Optional[MedicationRequest]:
        """Update an existing medication request"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_request_dict = {}

        # Handle status
        if hasattr(medication_request_data, 'status'):
            medication_request_dict["status"] = medication_request_data.status

        # Handle intent
        if hasattr(medication_request_data, 'intent'):
            medication_request_dict["intent"] = medication_request_data.intent

        # Handle medicationCodeableConcept
        if hasattr(medication_request_data, 'medicationCodeableConcept'):
            medication_request_dict["medicationCodeableConcept"] = {
                "coding": [],
                "text": medication_request_data.medicationCodeableConcept.text or ""
            }
            for coding in medication_request_data.medicationCodeableConcept.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_request_dict["medicationCodeableConcept"]["coding"].append(coding_dict)

        # Handle subject
        if hasattr(medication_request_data, 'subject'):
            medication_request_dict["subject"] = {
                "reference": medication_request_data.subject.reference
            }
            if medication_request_data.subject.display:
                medication_request_dict["subject"]["display"] = medication_request_data.subject.display

        # Handle authoredOn
        if hasattr(medication_request_data, 'authoredOn') and medication_request_data.authoredOn:
            medication_request_dict["authoredOn"] = medication_request_data.authoredOn

        # Handle requester
        if hasattr(medication_request_data, 'requester') and medication_request_data.requester:
            medication_request_dict["requester"] = {
                "reference": medication_request_data.requester.reference
            }
            if medication_request_data.requester.display:
                medication_request_dict["requester"]["display"] = medication_request_data.requester.display

        # Handle dosageInstruction
        if hasattr(medication_request_data, 'dosageInstruction') and medication_request_data.dosageInstruction:
            medication_request_dict["dosageInstruction"] = []
            for dosage in medication_request_data.dosageInstruction:
                dosage_dict = {}
                if hasattr(dosage, 'text') and dosage.text:
                    dosage_dict["text"] = dosage.text
                if hasattr(dosage, 'timing') and dosage.timing:
                    dosage_dict["timing"] = dosage.timing
                if hasattr(dosage, 'asNeededBoolean'):
                    dosage_dict["asNeededBoolean"] = dosage.asNeededBoolean
                if hasattr(dosage, 'route') and dosage.route:
                    dosage_dict["route"] = {
                        "coding": [],
                        "text": dosage.route.text or ""
                    }
                    for coding in dosage.route.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        dosage_dict["route"]["coding"].append(coding_dict)
                medication_request_dict["dosageInstruction"].append(dosage_dict)

        # Handle note
        if hasattr(medication_request_data, 'note') and medication_request_data.note:
            medication_request_dict["note"] = []
            for note in medication_request_data.note:
                note_dict = {
                    "text": note.text
                }
                if note.authorString:
                    note_dict["authorString"] = note.authorString
                if note.time:
                    note_dict["time"] = note.time
                medication_request_dict["note"].append(note_dict)

        # Add resourceType and id
        medication_request_dict["resourceType"] = "MedicationRequest"
        medication_request_dict["id"] = id

        # Update medication request via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-requests/{id}"
                print(f"Making request to: {url}")

                response = await client.put(
                    url,
                    json=medication_request_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error updating medication request: {response.status_code} - {response.text}")
                    return None

                updated_med_request = response.json()

                # Convert the REST response to a GraphQL type
                # Handle medicationCodeableConcept
                med_code_obj = None
                if 'medicationCodeableConcept' in updated_med_request and isinstance(updated_med_request['medicationCodeableConcept'], dict):
                    coding_list = []
                    for coding in updated_med_request['medicationCodeableConcept'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    med_code_obj = CodeableConcept(
                        coding=coding_list,
                        text=updated_med_request['medicationCodeableConcept'].get('text', '')
                    )

                # Handle subject
                subject_obj = None
                if 'subject' in updated_med_request and isinstance(updated_med_request['subject'], dict):
                    subject_obj = Reference(
                        reference=updated_med_request['subject'].get('reference', ''),
                        display=updated_med_request['subject'].get('display', '')
                    )

                # Handle requester
                requester_obj = None
                if 'requester' in updated_med_request and isinstance(updated_med_request['requester'], dict):
                    requester_obj = Reference(
                        reference=updated_med_request['requester'].get('reference', ''),
                        display=updated_med_request['requester'].get('display', '')
                    )

                # Handle dosageInstruction
                dosage_list = None
                if 'dosageInstruction' in updated_med_request and isinstance(updated_med_request['dosageInstruction'], list):
                    dosage_list = []
                    for dosage in updated_med_request['dosageInstruction']:
                        route_obj = None
                        if 'route' in dosage and isinstance(dosage['route'], dict):
                            coding_list = []
                            for coding in dosage['route'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            route_obj = CodeableConcept(
                                coding=coding_list,
                                text=dosage['route'].get('text', '')
                            )

                        dosage_list.append(
                            DosageInstruction(
                                text=dosage.get('text'),
                                timing=dosage.get('timing'),
                                asNeededBoolean=dosage.get('asNeededBoolean'),
                                route=route_obj
                            )
                        )

                # Handle note
                note_list = None
                if 'note' in updated_med_request and isinstance(updated_med_request['note'], list):
                    note_list = []
                    for note in updated_med_request['note']:
                        note_list.append(
                            Annotation(
                                text=note.get('text', ''),
                                authorString=note.get('authorString'),
                                time=note.get('time')
                            )
                        )

                # Create MedicationRequest object
                return MedicationRequest(
                    id=updated_med_request.get('id', ''),
                    resourceType=updated_med_request.get('resourceType', 'MedicationRequest'),
                    status=updated_med_request.get('status', ''),
                    intent=updated_med_request.get('intent', ''),
                    medicationCodeableConcept=med_code_obj,
                    subject=subject_obj,
                    authoredOn=updated_med_request.get('authoredOn', ''),
                    requester=requester_obj,
                    dosageInstruction=dosage_list,
                    note=note_list
                )
        except Exception as e:
            print(f"Error updating medication request: {str(e)}")
            return None

    @strawberry.mutation
    async def delete_medication_request(self, info, id: str) -> bool:
        """Delete a medication request"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return False

        # Delete medication request via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-requests/{id}"
                print(f"Making request to: {url}")

                response = await client.delete(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error deleting medication request: {response.status_code} - {response.text}")
                    return False

                return True
        except Exception as e:
            print(f"Error deleting medication request: {str(e)}")
            return False

    @strawberry.mutation
    async def create_medication_statement(self, info, medication_statement_data: MedicationStatementInput) -> Optional[MedicationStatement]:
        """Create a new medication statement"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_statement_dict = {}

        # Handle status
        if hasattr(medication_statement_data, 'status'):
            medication_statement_dict["status"] = medication_statement_data.status

        # Handle medicationCodeableConcept
        if hasattr(medication_statement_data, 'medicationCodeableConcept'):
            medication_statement_dict["medicationCodeableConcept"] = {
                "coding": [],
                "text": medication_statement_data.medicationCodeableConcept.text or ""
            }
            for coding in medication_statement_data.medicationCodeableConcept.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_statement_dict["medicationCodeableConcept"]["coding"].append(coding_dict)

        # Handle subject
        if hasattr(medication_statement_data, 'subject'):
            medication_statement_dict["subject"] = {
                "reference": medication_statement_data.subject.reference
            }
            if medication_statement_data.subject.display:
                medication_statement_dict["subject"]["display"] = medication_statement_data.subject.display

        # Handle effectiveDateTime
        if hasattr(medication_statement_data, 'effectiveDateTime') and medication_statement_data.effectiveDateTime:
            medication_statement_dict["effectiveDateTime"] = medication_statement_data.effectiveDateTime

        # Handle dateAsserted
        if hasattr(medication_statement_data, 'dateAsserted') and medication_statement_data.dateAsserted:
            medication_statement_dict["dateAsserted"] = medication_statement_data.dateAsserted

        # Handle informationSource
        if hasattr(medication_statement_data, 'informationSource') and medication_statement_data.informationSource:
            medication_statement_dict["informationSource"] = {
                "reference": medication_statement_data.informationSource.reference
            }
            if medication_statement_data.informationSource.display:
                medication_statement_dict["informationSource"]["display"] = medication_statement_data.informationSource.display

        # Handle dosage
        if hasattr(medication_statement_data, 'dosage') and medication_statement_data.dosage:
            medication_statement_dict["dosage"] = []
            for dosage in medication_statement_data.dosage:
                dosage_dict = {}
                if hasattr(dosage, 'text') and dosage.text:
                    dosage_dict["text"] = dosage.text
                if hasattr(dosage, 'timing') and dosage.timing:
                    dosage_dict["timing"] = dosage.timing
                if hasattr(dosage, 'asNeededBoolean'):
                    dosage_dict["asNeededBoolean"] = dosage.asNeededBoolean
                if hasattr(dosage, 'route') and dosage.route:
                    dosage_dict["route"] = {
                        "coding": [],
                        "text": dosage.route.text or ""
                    }
                    for coding in dosage.route.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        dosage_dict["route"]["coding"].append(coding_dict)
                medication_statement_dict["dosage"].append(dosage_dict)

        # Handle note
        if hasattr(medication_statement_data, 'note') and medication_statement_data.note:
            medication_statement_dict["note"] = []
            for note in medication_statement_data.note:
                note_dict = {
                    "text": note.text
                }
                if note.authorString:
                    note_dict["authorString"] = note.authorString
                if note.time:
                    note_dict["time"] = note.time
                medication_statement_dict["note"].append(note_dict)

        # Add resourceType
        medication_statement_dict["resourceType"] = "MedicationStatement"

        # Create medication statement via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-statements"
                print(f"Making request to: {url}")

                response = await client.post(
                    url,
                    json=medication_statement_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 201 and response.status_code != 200:
                    print(f"Error creating medication statement: {response.status_code} - {response.text}")
                    return None

                created_med_stmt = response.json()

                # Convert the REST response to a GraphQL type
                # Handle medicationCodeableConcept
                med_code_obj = None
                if 'medicationCodeableConcept' in created_med_stmt and isinstance(created_med_stmt['medicationCodeableConcept'], dict):
                    coding_list = []
                    for coding in created_med_stmt['medicationCodeableConcept'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    med_code_obj = CodeableConcept(
                        coding=coding_list,
                        text=created_med_stmt['medicationCodeableConcept'].get('text', '')
                    )

                # Handle subject
                subject_obj = None
                if 'subject' in created_med_stmt and isinstance(created_med_stmt['subject'], dict):
                    subject_obj = Reference(
                        reference=created_med_stmt['subject'].get('reference', ''),
                        display=created_med_stmt['subject'].get('display', '')
                    )

                # Handle informationSource
                info_source_obj = None
                if 'informationSource' in created_med_stmt and isinstance(created_med_stmt['informationSource'], dict):
                    info_source_obj = Reference(
                        reference=created_med_stmt['informationSource'].get('reference', ''),
                        display=created_med_stmt['informationSource'].get('display', '')
                    )

                # Handle dosage
                dosage_list = None
                if 'dosage' in created_med_stmt and isinstance(created_med_stmt['dosage'], list):
                    dosage_list = []
                    for dosage in created_med_stmt['dosage']:
                        route_obj = None
                        if 'route' in dosage and isinstance(dosage['route'], dict):
                            coding_list = []
                            for coding in dosage['route'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            route_obj = CodeableConcept(
                                coding=coding_list,
                                text=dosage['route'].get('text', '')
                            )

                        dosage_list.append(
                            DosageInstruction(
                                text=dosage.get('text'),
                                timing=dosage.get('timing'),
                                asNeededBoolean=dosage.get('asNeededBoolean'),
                                route=route_obj
                            )
                        )

                # Handle note
                note_list = None
                if 'note' in created_med_stmt and isinstance(created_med_stmt['note'], list):
                    note_list = []
                    for note in created_med_stmt['note']:
                        note_list.append(
                            Annotation(
                                text=note.get('text', ''),
                                authorString=note.get('authorString'),
                                time=note.get('time')
                            )
                        )

                # Create MedicationStatement object
                return MedicationStatement(
                    id=created_med_stmt.get('id', ''),
                    resourceType=created_med_stmt.get('resourceType', 'MedicationStatement'),
                    status=created_med_stmt.get('status', ''),
                    medicationCodeableConcept=med_code_obj,
                    subject=subject_obj,
                    effectiveDateTime=created_med_stmt.get('effectiveDateTime', ''),
                    dateAsserted=created_med_stmt.get('dateAsserted', ''),
                    informationSource=info_source_obj,
                    dosage=dosage_list,
                    note=note_list
                )
        except Exception as e:
            print(f"Error creating medication statement: {str(e)}")
            return None

    @strawberry.mutation
    async def update_medication_statement(self, info, id: str, medication_statement_data: MedicationStatementInput) -> Optional[MedicationStatement]:
        """Update an existing medication statement"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_statement_dict = {}

        # Handle status
        if hasattr(medication_statement_data, 'status'):
            medication_statement_dict["status"] = medication_statement_data.status

        # Handle medicationCodeableConcept
        if hasattr(medication_statement_data, 'medicationCodeableConcept'):
            medication_statement_dict["medicationCodeableConcept"] = {
                "coding": [],
                "text": medication_statement_data.medicationCodeableConcept.text or ""
            }
            for coding in medication_statement_data.medicationCodeableConcept.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_statement_dict["medicationCodeableConcept"]["coding"].append(coding_dict)

        # Handle subject
        if hasattr(medication_statement_data, 'subject'):
            medication_statement_dict["subject"] = {
                "reference": medication_statement_data.subject.reference
            }
            if medication_statement_data.subject.display:
                medication_statement_dict["subject"]["display"] = medication_statement_data.subject.display

        # Handle effectiveDateTime
        if hasattr(medication_statement_data, 'effectiveDateTime') and medication_statement_data.effectiveDateTime:
            medication_statement_dict["effectiveDateTime"] = medication_statement_data.effectiveDateTime

        # Handle dateAsserted
        if hasattr(medication_statement_data, 'dateAsserted') and medication_statement_data.dateAsserted:
            medication_statement_dict["dateAsserted"] = medication_statement_data.dateAsserted

        # Handle informationSource
        if hasattr(medication_statement_data, 'informationSource') and medication_statement_data.informationSource:
            medication_statement_dict["informationSource"] = {
                "reference": medication_statement_data.informationSource.reference
            }
            if medication_statement_data.informationSource.display:
                medication_statement_dict["informationSource"]["display"] = medication_statement_data.informationSource.display

        # Handle dosage
        if hasattr(medication_statement_data, 'dosage') and medication_statement_data.dosage:
            medication_statement_dict["dosage"] = []
            for dosage in medication_statement_data.dosage:
                dosage_dict = {}
                if hasattr(dosage, 'text') and dosage.text:
                    dosage_dict["text"] = dosage.text
                if hasattr(dosage, 'timing') and dosage.timing:
                    dosage_dict["timing"] = dosage.timing
                if hasattr(dosage, 'asNeededBoolean'):
                    dosage_dict["asNeededBoolean"] = dosage.asNeededBoolean
                if hasattr(dosage, 'route') and dosage.route:
                    dosage_dict["route"] = {
                        "coding": [],
                        "text": dosage.route.text or ""
                    }
                    for coding in dosage.route.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        dosage_dict["route"]["coding"].append(coding_dict)
                medication_statement_dict["dosage"].append(dosage_dict)

        # Handle note
        if hasattr(medication_statement_data, 'note') and medication_statement_data.note:
            medication_statement_dict["note"] = []
            for note in medication_statement_data.note:
                note_dict = {
                    "text": note.text
                }
                if note.authorString:
                    note_dict["authorString"] = note.authorString
                if note.time:
                    note_dict["time"] = note.time
                medication_statement_dict["note"].append(note_dict)

        # Add resourceType and id
        medication_statement_dict["resourceType"] = "MedicationStatement"
        medication_statement_dict["id"] = id

        # Update medication statement via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-statements/{id}"
                print(f"Making request to: {url}")

                response = await client.put(
                    url,
                    json=medication_statement_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error updating medication statement: {response.status_code} - {response.text}")
                    return None

                updated_med_stmt = response.json()

                # Convert the REST response to a GraphQL type
                # Handle medicationCodeableConcept
                med_code_obj = None
                if 'medicationCodeableConcept' in updated_med_stmt and isinstance(updated_med_stmt['medicationCodeableConcept'], dict):
                    coding_list = []
                    for coding in updated_med_stmt['medicationCodeableConcept'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    med_code_obj = CodeableConcept(
                        coding=coding_list,
                        text=updated_med_stmt['medicationCodeableConcept'].get('text', '')
                    )

                # Handle subject
                subject_obj = None
                if 'subject' in updated_med_stmt and isinstance(updated_med_stmt['subject'], dict):
                    subject_obj = Reference(
                        reference=updated_med_stmt['subject'].get('reference', ''),
                        display=updated_med_stmt['subject'].get('display', '')
                    )

                # Handle informationSource
                info_source_obj = None
                if 'informationSource' in updated_med_stmt and isinstance(updated_med_stmt['informationSource'], dict):
                    info_source_obj = Reference(
                        reference=updated_med_stmt['informationSource'].get('reference', ''),
                        display=updated_med_stmt['informationSource'].get('display', '')
                    )

                # Handle dosage
                dosage_list = None
                if 'dosage' in updated_med_stmt and isinstance(updated_med_stmt['dosage'], list):
                    dosage_list = []
                    for dosage in updated_med_stmt['dosage']:
                        route_obj = None
                        if 'route' in dosage and isinstance(dosage['route'], dict):
                            coding_list = []
                            for coding in dosage['route'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            route_obj = CodeableConcept(
                                coding=coding_list,
                                text=dosage['route'].get('text', '')
                            )

                        dosage_list.append(
                            DosageInstruction(
                                text=dosage.get('text'),
                                timing=dosage.get('timing'),
                                asNeededBoolean=dosage.get('asNeededBoolean'),
                                route=route_obj
                            )
                        )

                # Handle note
                note_list = None
                if 'note' in updated_med_stmt and isinstance(updated_med_stmt['note'], list):
                    note_list = []
                    for note in updated_med_stmt['note']:
                        note_list.append(
                            Annotation(
                                text=note.get('text', ''),
                                authorString=note.get('authorString'),
                                time=note.get('time')
                            )
                        )

                # Create MedicationStatement object
                return MedicationStatement(
                    id=updated_med_stmt.get('id', ''),
                    resourceType=updated_med_stmt.get('resourceType', 'MedicationStatement'),
                    status=updated_med_stmt.get('status', ''),
                    medicationCodeableConcept=med_code_obj,
                    subject=subject_obj,
                    effectiveDateTime=updated_med_stmt.get('effectiveDateTime', ''),
                    dateAsserted=updated_med_stmt.get('dateAsserted', ''),
                    informationSource=info_source_obj,
                    dosage=dosage_list,
                    note=note_list
                )
        except Exception as e:
            print(f"Error updating medication statement: {str(e)}")
            return None

    @strawberry.mutation
    async def delete_medication_statement(self, info, id: str) -> bool:
        """Delete a medication statement"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return False

        # Delete medication statement via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-statements/{id}"
                print(f"Making request to: {url}")

                response = await client.delete(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error deleting medication statement: {response.status_code} - {response.text}")
                    return False

                return True
        except Exception as e:
            print(f"Error deleting medication statement: {str(e)}")
            return False

    @strawberry.mutation
    async def create_medication_administration(self, info, medication_administration_data: MedicationAdministrationInput) -> Optional[MedicationAdministration]:
        """Create a new medication administration"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_administration_dict = {}

        # Handle status
        if hasattr(medication_administration_data, 'status'):
            medication_administration_dict["status"] = medication_administration_data.status

        # Handle medicationCodeableConcept
        if hasattr(medication_administration_data, 'medicationCodeableConcept'):
            medication_administration_dict["medicationCodeableConcept"] = {
                "coding": [],
                "text": medication_administration_data.medicationCodeableConcept.text or ""
            }
            for coding in medication_administration_data.medicationCodeableConcept.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_administration_dict["medicationCodeableConcept"]["coding"].append(coding_dict)

        # Handle subject
        if hasattr(medication_administration_data, 'subject'):
            medication_administration_dict["subject"] = {
                "reference": medication_administration_data.subject.reference
            }
            if medication_administration_data.subject.display:
                medication_administration_dict["subject"]["display"] = medication_administration_data.subject.display

        # Handle effectiveDateTime
        if hasattr(medication_administration_data, 'effectiveDateTime') and medication_administration_data.effectiveDateTime:
            medication_administration_dict["effectiveDateTime"] = medication_administration_data.effectiveDateTime

        # Handle performer
        if hasattr(medication_administration_data, 'performer') and medication_administration_data.performer:
            medication_administration_dict["performer"] = []
            for performer in medication_administration_data.performer:
                performer_dict = {
                    "actor": {
                        "reference": performer.reference
                    }
                }
                if performer.display:
                    performer_dict["actor"]["display"] = performer.display
                medication_administration_dict["performer"].append(performer_dict)

        # Handle request
        if hasattr(medication_administration_data, 'request') and medication_administration_data.request:
            medication_administration_dict["request"] = {
                "reference": medication_administration_data.request.reference
            }
            if medication_administration_data.request.display:
                medication_administration_dict["request"]["display"] = medication_administration_data.request.display

        # Handle dosage
        if hasattr(medication_administration_data, 'dosage') and medication_administration_data.dosage:
            medication_administration_dict["dosage"] = medication_administration_data.dosage

        # Handle note
        if hasattr(medication_administration_data, 'note') and medication_administration_data.note:
            medication_administration_dict["note"] = []
            for note in medication_administration_data.note:
                note_dict = {
                    "text": note.text
                }
                if note.authorString:
                    note_dict["authorString"] = note.authorString
                if note.time:
                    note_dict["time"] = note.time
                medication_administration_dict["note"].append(note_dict)

        # Add resourceType
        medication_administration_dict["resourceType"] = "MedicationAdministration"

        # Create medication administration via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-administrations"
                print(f"Making request to: {url}")

                response = await client.post(
                    url,
                    json=medication_administration_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 201 and response.status_code != 200:
                    print(f"Error creating medication administration: {response.status_code} - {response.text}")
                    return None

                created_med_admin = response.json()

                # Convert the REST response to a GraphQL type
                # Handle medicationCodeableConcept
                med_code_obj = None
                if 'medicationCodeableConcept' in created_med_admin and isinstance(created_med_admin['medicationCodeableConcept'], dict):
                    coding_list = []
                    for coding in created_med_admin['medicationCodeableConcept'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    med_code_obj = CodeableConcept(
                        coding=coding_list,
                        text=created_med_admin['medicationCodeableConcept'].get('text', '')
                    )

                # Handle subject
                subject_obj = None
                if 'subject' in created_med_admin and isinstance(created_med_admin['subject'], dict):
                    subject_obj = Reference(
                        reference=created_med_admin['subject'].get('reference', ''),
                        display=created_med_admin['subject'].get('display', '')
                    )

                # Handle performer
                performer_list = None
                if 'performer' in created_med_admin and isinstance(created_med_admin['performer'], list):
                    performer_list = []
                    for performer in created_med_admin['performer']:
                        if 'actor' in performer and isinstance(performer['actor'], dict):
                            performer_list.append(
                                Reference(
                                    reference=performer['actor'].get('reference', ''),
                                    display=performer['actor'].get('display', '')
                                )
                            )

                # Handle request
                request_obj = None
                if 'request' in created_med_admin and isinstance(created_med_admin['request'], dict):
                    request_obj = Reference(
                        reference=created_med_admin['request'].get('reference', ''),
                        display=created_med_admin['request'].get('display', ''),
                        type=created_med_admin['request'].get('type'),
                        identifier=None
                    )

                # Handle note
                note_list = None
                if 'note' in created_med_admin and isinstance(created_med_admin['note'], list):
                    note_list = []
                    for note in created_med_admin['note']:
                        note_list.append(
                            Annotation(
                                text=note.get('text', ''),
                                authorString=note.get('authorString'),
                                time=note.get('time')
                            )
                        )

                # Create MedicationAdministration object
                return MedicationAdministration(
                    id=created_med_admin.get('id', ''),
                    resourceType=created_med_admin.get('resourceType', 'MedicationAdministration'),
                    status=created_med_admin.get('status', ''),
                    medicationCodeableConcept=med_code_obj,
                    subject=subject_obj,
                    effectiveDateTime=created_med_admin.get('effectiveDateTime', ''),
                    performer=performer_list,
                    request=request_obj,
                    dosage=created_med_admin.get('dosage', ''),
                    note=note_list
                )
        except Exception as e:
            print(f"Error creating medication administration: {str(e)}")
            return None

    @strawberry.mutation
    async def update_medication_administration(self, info, id: str, medication_administration_data: MedicationAdministrationInput) -> Optional[MedicationAdministration]:
        """Update an existing medication administration"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        medication_administration_dict = {}

        # Handle status
        if hasattr(medication_administration_data, 'status'):
            medication_administration_dict["status"] = medication_administration_data.status

        # Handle medicationCodeableConcept
        if hasattr(medication_administration_data, 'medicationCodeableConcept'):
            medication_administration_dict["medicationCodeableConcept"] = {
                "coding": [],
                "text": medication_administration_data.medicationCodeableConcept.text or ""
            }
            for coding in medication_administration_data.medicationCodeableConcept.coding:
                coding_dict = {
                    "system": coding.system,
                    "code": coding.code
                }
                if coding.display:
                    coding_dict["display"] = coding.display
                medication_administration_dict["medicationCodeableConcept"]["coding"].append(coding_dict)

        # Handle subject
        if hasattr(medication_administration_data, 'subject'):
            medication_administration_dict["subject"] = {
                "reference": medication_administration_data.subject.reference
            }
            if medication_administration_data.subject.display:
                medication_administration_dict["subject"]["display"] = medication_administration_data.subject.display

        # Handle effectiveDateTime
        if hasattr(medication_administration_data, 'effectiveDateTime') and medication_administration_data.effectiveDateTime:
            medication_administration_dict["effectiveDateTime"] = medication_administration_data.effectiveDateTime

        # Handle performer
        if hasattr(medication_administration_data, 'performer') and medication_administration_data.performer:
            medication_administration_dict["performer"] = []
            for performer in medication_administration_data.performer:
                performer_dict = {
                    "actor": {
                        "reference": performer.reference
                    }
                }
                if performer.display:
                    performer_dict["actor"]["display"] = performer.display
                medication_administration_dict["performer"].append(performer_dict)

        # Handle request
        if hasattr(medication_administration_data, 'request') and medication_administration_data.request:
            medication_administration_dict["request"] = {
                "reference": medication_administration_data.request.reference
            }
            if medication_administration_data.request.display:
                medication_administration_dict["request"]["display"] = medication_administration_data.request.display

        # Handle dosage
        if hasattr(medication_administration_data, 'dosage') and medication_administration_data.dosage:
            medication_administration_dict["dosage"] = medication_administration_data.dosage

        # Handle note
        if hasattr(medication_administration_data, 'note') and medication_administration_data.note:
            medication_administration_dict["note"] = []
            for note in medication_administration_data.note:
                note_dict = {
                    "text": note.text
                }
                if note.authorString:
                    note_dict["authorString"] = note.authorString
                if note.time:
                    note_dict["time"] = note.time
                medication_administration_dict["note"].append(note_dict)

        # Add resourceType and id
        medication_administration_dict["resourceType"] = "MedicationAdministration"
        medication_administration_dict["id"] = id

        # Update medication administration via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-administrations/{id}"
                print(f"Making request to: {url}")

                response = await client.put(
                    url,
                    json=medication_administration_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error updating medication administration: {response.status_code} - {response.text}")
                    return None

                updated_med_admin = response.json()

                # Convert the REST response to a GraphQL type
                # Handle medicationCodeableConcept
                med_code_obj = None
                if 'medicationCodeableConcept' in updated_med_admin and isinstance(updated_med_admin['medicationCodeableConcept'], dict):
                    coding_list = []
                    for coding in updated_med_admin['medicationCodeableConcept'].get('coding', []):
                        coding_list.append(
                            Coding(
                                system=coding.get('system', ''),
                                code=coding.get('code', ''),
                                display=coding.get('display', '')
                            )
                        )
                    med_code_obj = CodeableConcept(
                        coding=coding_list,
                        text=updated_med_admin['medicationCodeableConcept'].get('text', '')
                    )

                # Handle subject
                subject_obj = None
                if 'subject' in updated_med_admin and isinstance(updated_med_admin['subject'], dict):
                    subject_obj = Reference(
                        reference=updated_med_admin['subject'].get('reference', ''),
                        display=updated_med_admin['subject'].get('display', '')
                    )

                # Handle performer
                performer_list = None
                if 'performer' in updated_med_admin and isinstance(updated_med_admin['performer'], list):
                    performer_list = []
                    for performer in updated_med_admin['performer']:
                        if 'actor' in performer and isinstance(performer['actor'], dict):
                            performer_list.append(
                                Reference(
                                    reference=performer['actor'].get('reference', ''),
                                    display=performer['actor'].get('display', '')
                                )
                            )

                # Handle request
                request_obj = None
                if 'request' in updated_med_admin and isinstance(updated_med_admin['request'], dict):
                    request_obj = Reference(
                        reference=updated_med_admin['request'].get('reference', ''),
                        display=updated_med_admin['request'].get('display', ''),
                        type=updated_med_admin['request'].get('type'),
                        identifier=None
                    )

                # Handle note
                note_list = None
                if 'note' in updated_med_admin and isinstance(updated_med_admin['note'], list):
                    note_list = []
                    for note in updated_med_admin['note']:
                        note_list.append(
                            Annotation(
                                text=note.get('text', ''),
                                authorString=note.get('authorString'),
                                time=note.get('time')
                            )
                        )

                # Create MedicationAdministration object
                return MedicationAdministration(
                    id=updated_med_admin.get('id', ''),
                    resourceType=updated_med_admin.get('resourceType', 'MedicationAdministration'),
                    status=updated_med_admin.get('status', ''),
                    medicationCodeableConcept=med_code_obj,
                    subject=subject_obj,
                    effectiveDateTime=updated_med_admin.get('effectiveDateTime', ''),
                    performer=performer_list,
                    request=request_obj,
                    dosage=updated_med_admin.get('dosage', ''),
                    note=note_list
                )
        except Exception as e:
            print(f"Error updating medication administration: {str(e)}")
            return None

    @strawberry.mutation
    async def create_encounter(self, info, encounter_data: EncounterInput) -> Optional[Encounter]:
        """Create a new encounter"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        encounter_dict = {}

        # Handle status
        if hasattr(encounter_data, 'status'):
            encounter_dict["status"] = encounter_data.status

        # Handle class
        if hasattr(encounter_data, 'class_'):
            class_data = encounter_data.class_
            if hasattr(class_data, 'coding') and class_data.coding:
                # Convert CodeableConcept to the format expected by the API
                coding = class_data.coding[0]
                encounter_dict["class"] = {
                    "system": coding.system,
                    "code": coding.code,
                    "display": coding.display
                }
                if class_data.text:
                    encounter_dict["class"]["text"] = class_data.text
            else:
                # If no coding is provided, just use a simple string
                encounter_dict["class"] = class_data

        # Handle type
        if hasattr(encounter_data, 'type') and encounter_data.type:
            encounter_dict["type"] = []
            for type_item in encounter_data.type:
                type_dict = {}
                if hasattr(type_item, 'coding') and type_item.coding:
                    type_dict["coding"] = []
                    for coding in type_item.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        type_dict["coding"].append(coding_dict)
                if hasattr(type_item, 'text') and type_item.text:
                    type_dict["text"] = type_item.text
                encounter_dict["type"].append(type_dict)

        # Handle subject
        if hasattr(encounter_data, 'subject'):
            encounter_dict["subject"] = {
                "reference": encounter_data.subject.reference
            }
            if encounter_data.subject.display:
                encounter_dict["subject"]["display"] = encounter_data.subject.display

        # Handle participant
        if hasattr(encounter_data, 'participant') and encounter_data.participant:
            encounter_dict["participant"] = []
            for participant in encounter_data.participant:
                participant_dict = {}
                if hasattr(participant, 'type') and participant.type:
                    participant_dict["type"] = []
                    for type_item in participant.type:
                        type_dict = {}
                        if hasattr(type_item, 'coding') and type_item.coding:
                            type_dict["coding"] = []
                            for coding in type_item.coding:
                                coding_dict = {
                                    "system": coding.system,
                                    "code": coding.code
                                }
                                if coding.display:
                                    coding_dict["display"] = coding.display
                                type_dict["coding"].append(coding_dict)
                        if hasattr(type_item, 'text') and type_item.text:
                            type_dict["text"] = type_item.text
                        participant_dict["type"].append(type_dict)
                if hasattr(participant, 'period') and participant.period:
                    participant_dict["period"] = {}
                    if participant.period.start:
                        participant_dict["period"]["start"] = participant.period.start
                    if participant.period.end:
                        participant_dict["period"]["end"] = participant.period.end
                if hasattr(participant, 'individual') and participant.individual:
                    participant_dict["individual"] = {
                        "reference": participant.individual.reference
                    }
                    if participant.individual.display:
                        participant_dict["individual"]["display"] = participant.individual.display
                encounter_dict["participant"].append(participant_dict)

        # Handle period
        if hasattr(encounter_data, 'period') and encounter_data.period:
            encounter_dict["period"] = {}
            if encounter_data.period.start:
                encounter_dict["period"]["start"] = encounter_data.period.start
            if encounter_data.period.end:
                encounter_dict["period"]["end"] = encounter_data.period.end

        # Handle reasonCode
        if hasattr(encounter_data, 'reasonCode') and encounter_data.reasonCode:
            encounter_dict["reasonCode"] = []
            for reason in encounter_data.reasonCode:
                reason_dict = {}
                if hasattr(reason, 'coding') and reason.coding:
                    reason_dict["coding"] = []
                    for coding in reason.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        reason_dict["coding"].append(coding_dict)
                if hasattr(reason, 'text') and reason.text:
                    reason_dict["text"] = reason.text
                encounter_dict["reasonCode"].append(reason_dict)

        # Handle diagnosis
        if hasattr(encounter_data, 'diagnosis') and encounter_data.diagnosis:
            encounter_dict["diagnosis"] = []
            for diagnosis in encounter_data.diagnosis:
                diagnosis_dict = {}
                if hasattr(diagnosis, 'condition'):
                    diagnosis_dict["condition"] = {
                        "reference": diagnosis.condition.reference
                    }
                    if diagnosis.condition.display:
                        diagnosis_dict["condition"]["display"] = diagnosis.condition.display
                if hasattr(diagnosis, 'use') and diagnosis.use:
                    diagnosis_dict["use"] = {}
                    if hasattr(diagnosis.use, 'coding') and diagnosis.use.coding:
                        diagnosis_dict["use"]["coding"] = []
                        for coding in diagnosis.use.coding:
                            coding_dict = {
                                "system": coding.system,
                                "code": coding.code
                            }
                            if coding.display:
                                coding_dict["display"] = coding.display
                            diagnosis_dict["use"]["coding"].append(coding_dict)
                    if hasattr(diagnosis.use, 'text') and diagnosis.use.text:
                        diagnosis_dict["use"]["text"] = diagnosis.use.text
                if hasattr(diagnosis, 'rank'):
                    diagnosis_dict["rank"] = diagnosis.rank
                encounter_dict["diagnosis"].append(diagnosis_dict)

        # Handle location
        if hasattr(encounter_data, 'location') and encounter_data.location:
            encounter_dict["location"] = []
            for location in encounter_data.location:
                location_dict = {}
                if hasattr(location, 'location'):
                    location_dict["location"] = {
                        "reference": location.location.reference
                    }
                    if location.location.display:
                        location_dict["location"]["display"] = location.location.display
                if hasattr(location, 'status'):
                    location_dict["status"] = location.status
                if hasattr(location, 'period') and location.period:
                    location_dict["period"] = {}
                    if location.period.start:
                        location_dict["period"]["start"] = location.period.start
                    if location.period.end:
                        location_dict["period"]["end"] = location.period.end
                encounter_dict["location"].append(location_dict)

        # Handle serviceProvider
        if hasattr(encounter_data, 'serviceProvider') and encounter_data.serviceProvider is not None:
            encounter_dict["serviceProvider"] = {
                "reference": encounter_data.serviceProvider.reference
            }
            if hasattr(encounter_data.serviceProvider, 'display') and encounter_data.serviceProvider.display:
                encounter_dict["serviceProvider"]["display"] = encounter_data.serviceProvider.display

        # Add resourceType
        encounter_dict["resourceType"] = "Encounter"

        # Create encounter via Encounter service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.ENCOUNTER_SERVICE_URL}/api/encounters"
                print(f"Making request to: {url}")

                response = await client.post(
                    url,
                    json=encounter_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 201 and response.status_code != 200:
                    print(f"Error creating encounter: {response.status_code} - {response.text}")
                    return None

                created_encounter = response.json()

                # Convert the REST response to a GraphQL type
                # Handle 'class' field as a CodeableConcept
                if 'class' in created_encounter:
                    class_data = created_encounter.pop('class')
                    if isinstance(class_data, dict):
                        # Create a CodeableConcept from the class data
                        coding_list = []
                        if 'system' in class_data and 'code' in class_data:
                            coding_list.append(Coding(
                                system=class_data.get('system', ''),
                                code=class_data.get('code', ''),
                                display=class_data.get('display', ''),
                                version=class_data.get('version'),
                                user_selected=class_data.get('userSelected')
                            ))
                        created_encounter['class_'] = CodeableConcept(
                            coding=coding_list,
                            text=class_data.get('text', '')
                        )
                    else:
                        # If it's not a dict, just use it as is
                        created_encounter['class_'] = class_data

                # Handle 'subject' field as a Reference
                if 'subject' in created_encounter and isinstance(created_encounter['subject'], dict):
                    subject_data = created_encounter.pop('subject')
                    created_encounter['subject'] = Reference(
                        reference=subject_data.get('reference', ''),
                        display=subject_data.get('display', ''),
                        type=subject_data.get('type'),
                        identifier=None
                    )

                # Handle meta field
                if 'meta' in created_encounter and created_encounter['meta']:
                    meta_data = created_encounter.pop('meta')
                    created_encounter['meta'] = Meta(
                        versionId=meta_data.get('versionId'),
                        lastUpdated=meta_data.get('lastUpdated'),
                        source=meta_data.get('source'),
                        profile=meta_data.get('profile'),
                        security=None,  # Would need to convert to CodeableConcept if present
                        tag=None  # Would need to convert to CodeableConcept if present
                    )

                return Encounter(**created_encounter)
        except Exception as e:
            print(f"Error creating encounter: {str(e)}")
            return None

    @strawberry.mutation
    async def update_encounter(self, info, id: str, encounter_data: EncounterInput) -> Optional[Encounter]:
        """Update an existing encounter"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Convert input to dict
        encounter_dict = {}

        # Handle status
        if hasattr(encounter_data, 'status'):
            encounter_dict["status"] = encounter_data.status

        # Handle class
        if hasattr(encounter_data, 'class_'):
            class_data = encounter_data.class_
            if hasattr(class_data, 'coding') and class_data.coding:
                # Convert CodeableConcept to the format expected by the API
                coding = class_data.coding[0]
                encounter_dict["class"] = {
                    "system": coding.system,
                    "code": coding.code,
                    "display": coding.display
                }
                if class_data.text:
                    encounter_dict["class"]["text"] = class_data.text
            else:
                # If no coding is provided, just use a simple string
                encounter_dict["class"] = class_data

        # Handle type
        if hasattr(encounter_data, 'type') and encounter_data.type:
            encounter_dict["type"] = []
            for type_item in encounter_data.type:
                type_dict = {}
                if hasattr(type_item, 'coding') and type_item.coding:
                    type_dict["coding"] = []
                    for coding in type_item.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        type_dict["coding"].append(coding_dict)
                if hasattr(type_item, 'text') and type_item.text:
                    type_dict["text"] = type_item.text
                encounter_dict["type"].append(type_dict)

        # Handle subject
        if hasattr(encounter_data, 'subject'):
            encounter_dict["subject"] = {
                "reference": encounter_data.subject.reference
            }
            if encounter_data.subject.display:
                encounter_dict["subject"]["display"] = encounter_data.subject.display

        # Handle participant
        if hasattr(encounter_data, 'participant') and encounter_data.participant:
            encounter_dict["participant"] = []
            for participant in encounter_data.participant:
                participant_dict = {}
                if hasattr(participant, 'type') and participant.type:
                    participant_dict["type"] = []
                    for type_item in participant.type:
                        type_dict = {}
                        if hasattr(type_item, 'coding') and type_item.coding:
                            type_dict["coding"] = []
                            for coding in type_item.coding:
                                coding_dict = {
                                    "system": coding.system,
                                    "code": coding.code
                                }
                                if coding.display:
                                    coding_dict["display"] = coding.display
                                type_dict["coding"].append(coding_dict)
                        if hasattr(type_item, 'text') and type_item.text:
                            type_dict["text"] = type_item.text
                        participant_dict["type"].append(type_dict)
                if hasattr(participant, 'period') and participant.period:
                    participant_dict["period"] = {}
                    if participant.period.start:
                        participant_dict["period"]["start"] = participant.period.start
                    if participant.period.end:
                        participant_dict["period"]["end"] = participant.period.end
                if hasattr(participant, 'individual') and participant.individual:
                    participant_dict["individual"] = {
                        "reference": participant.individual.reference
                    }
                    if participant.individual.display:
                        participant_dict["individual"]["display"] = participant.individual.display
                encounter_dict["participant"].append(participant_dict)

        # Handle period
        if hasattr(encounter_data, 'period') and encounter_data.period:
            encounter_dict["period"] = {}
            if encounter_data.period.start:
                encounter_dict["period"]["start"] = encounter_data.period.start
            if encounter_data.period.end:
                encounter_dict["period"]["end"] = encounter_data.period.end

        # Handle reasonCode
        if hasattr(encounter_data, 'reasonCode') and encounter_data.reasonCode:
            encounter_dict["reasonCode"] = []
            for reason in encounter_data.reasonCode:
                reason_dict = {}
                if hasattr(reason, 'coding') and reason.coding:
                    reason_dict["coding"] = []
                    for coding in reason.coding:
                        coding_dict = {
                            "system": coding.system,
                            "code": coding.code
                        }
                        if coding.display:
                            coding_dict["display"] = coding.display
                        reason_dict["coding"].append(coding_dict)
                if hasattr(reason, 'text') and reason.text:
                    reason_dict["text"] = reason.text
                encounter_dict["reasonCode"].append(reason_dict)

        # Handle diagnosis
        if hasattr(encounter_data, 'diagnosis') and encounter_data.diagnosis:
            encounter_dict["diagnosis"] = []
            for diagnosis in encounter_data.diagnosis:
                diagnosis_dict = {}
                if hasattr(diagnosis, 'condition'):
                    diagnosis_dict["condition"] = {
                        "reference": diagnosis.condition.reference
                    }
                    if diagnosis.condition.display:
                        diagnosis_dict["condition"]["display"] = diagnosis.condition.display
                if hasattr(diagnosis, 'use') and diagnosis.use:
                    diagnosis_dict["use"] = {}
                    if hasattr(diagnosis.use, 'coding') and diagnosis.use.coding:
                        diagnosis_dict["use"]["coding"] = []
                        for coding in diagnosis.use.coding:
                            coding_dict = {
                                "system": coding.system,
                                "code": coding.code
                            }
                            if coding.display:
                                coding_dict["display"] = coding.display
                            diagnosis_dict["use"]["coding"].append(coding_dict)
                    if hasattr(diagnosis.use, 'text') and diagnosis.use.text:
                        diagnosis_dict["use"]["text"] = diagnosis.use.text
                if hasattr(diagnosis, 'rank'):
                    diagnosis_dict["rank"] = diagnosis.rank
                encounter_dict["diagnosis"].append(diagnosis_dict)

        # Handle location
        if hasattr(encounter_data, 'location') and encounter_data.location:
            encounter_dict["location"] = []
            for location in encounter_data.location:
                location_dict = {}
                if hasattr(location, 'location'):
                    location_dict["location"] = {
                        "reference": location.location.reference
                    }
                    if location.location.display:
                        location_dict["location"]["display"] = location.location.display
                if hasattr(location, 'status'):
                    location_dict["status"] = location.status
                if hasattr(location, 'period') and location.period:
                    location_dict["period"] = {}
                    if location.period.start:
                        location_dict["period"]["start"] = location.period.start
                    if location.period.end:
                        location_dict["period"]["end"] = location.period.end
                encounter_dict["location"].append(location_dict)

        # Handle serviceProvider
        if hasattr(encounter_data, 'serviceProvider') and encounter_data.serviceProvider is not None:
            encounter_dict["serviceProvider"] = {
                "reference": encounter_data.serviceProvider.reference
            }
            if hasattr(encounter_data.serviceProvider, 'display') and encounter_data.serviceProvider.display:
                encounter_dict["serviceProvider"]["display"] = encounter_data.serviceProvider.display

        # Add resourceType and id
        encounter_dict["resourceType"] = "Encounter"
        encounter_dict["id"] = id

        # Update encounter via Encounter service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.ENCOUNTER_SERVICE_URL}/api/encounters/{id}"
                print(f"Making request to: {url}")

                response = await client.put(
                    url,
                    json=encounter_dict,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error updating encounter: {response.status_code} - {response.text}")
                    return None

                updated_encounter = response.json()

                # Convert the REST response to a GraphQL type
                # Handle 'class' field as a CodeableConcept
                if 'class' in updated_encounter:
                    class_data = updated_encounter.pop('class')
                    if isinstance(class_data, dict):
                        # Create a CodeableConcept from the class data
                        coding_list = []
                        if 'system' in class_data and 'code' in class_data:
                            coding_list.append(Coding(
                                system=class_data.get('system', ''),
                                code=class_data.get('code', ''),
                                display=class_data.get('display', '')
                            ))
                        updated_encounter['class_'] = CodeableConcept(
                            coding=coding_list,
                            text=class_data.get('text', '')
                        )
                    else:
                        # If it's not a dict, just use it as is
                        updated_encounter['class_'] = class_data

                # Handle 'subject' field as a Reference
                if 'subject' in updated_encounter and isinstance(updated_encounter['subject'], dict):
                    subject_data = updated_encounter.pop('subject')
                    updated_encounter['subject'] = Reference(
                        reference=subject_data.get('reference', ''),
                        display=subject_data.get('display', '')
                    )

                # Handle meta field
                if 'meta' in updated_encounter and updated_encounter['meta']:
                    meta_data = updated_encounter.pop('meta')
                    updated_encounter['meta'] = Meta(
                        versionId=meta_data.get('versionId'),
                        lastUpdated=meta_data.get('lastUpdated'),
                        source=meta_data.get('source'),
                        profile=meta_data.get('profile'),
                        security=None,  # Would need to convert to CodeableConcept if present
                        tag=None  # Would need to convert to CodeableConcept if present
                    )

                return Encounter(**updated_encounter)
        except Exception as e:
            print(f"Error updating encounter: {str(e)}")
            return None

    @strawberry.mutation
    async def delete_encounter(self, info, id: str) -> bool:
        """Delete an encounter"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return False

        # Delete encounter via Encounter service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.ENCOUNTER_SERVICE_URL}/api/encounters/{id}"
                print(f"Making request to: {url}")

                response = await client.delete(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error deleting encounter: {response.status_code} - {response.text}")
                    return False

                return True
        except Exception as e:
            print(f"Error deleting encounter: {str(e)}")
            return False

    @strawberry.mutation
    async def delete_medication_administration(self, info, id: str) -> bool:
        """Delete a medication administration"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return False

        # Delete medication administration via Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-administrations/{id}"
                print(f"Making request to: {url}")

                response = await client.delete(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error deleting medication administration: {response.status_code} - {response.text}")
                    return False

                return True
        except Exception as e:
            print(f"Error deleting medication administration: {str(e)}")
            return False
