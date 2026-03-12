import strawberry
import httpx
from typing import List, Optional
from .types import (
    User, Patient, LabResult, Condition, MedicationRequest,
    DiagnosticReport, Encounter, DocumentReference,
    ObservationCodeEntry, ObservationSubjectEntry, ObservationValueEntry,
    ObservationInterpretationEntry, ObservationReferenceRangeEntry,
    ObservationReferenceRangeQuantity, VitalSign, PhysicalMeasurement,
    CompleteObservation, ObservationCode, ObservationSubject, ObservationValueQuantity,
    ObservationInterpretation, ObservationReferenceRange,
    Identifier, HumanName, ContactPoint, Address,
    Medication, MedicationAdministration, MedicationStatement,
    Coding, CodeableConcept, Reference, Quantity, Annotation, DosageInstruction,
    ProblemListItem, Diagnosis, HealthConcern, Meta, Period,
    TimelineEvent, PatientTimeline, EventDetails
)
from app.config import settings

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

        # Get patient from FHIR service directly
        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                # Get from FHIR service
                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{id}",
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return None

                patient_data = response.json()
                print(f"Retrieved patient: {patient_data}")

                # Convert the FHIR response to a format compatible with our GraphQL types
                try:
                    # Handle identifier list
                    identifiers = []
                    for ident in patient_data.get("identifier", []):
                        identifiers.append(
                            Identifier(
                                system=ident.get("system", ""),
                                value=ident.get("value", ""),
                                use=ident.get("use")
                            )
                        )

                    # Handle name list
                    names = []
                    for name in patient_data.get("name", []):
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
                    if "telecom" in patient_data:
                        telecoms = []
                        for telecom in patient_data.get("telecom", []):
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
                    if "address" in patient_data:
                        addresses = []
                        for addr in patient_data.get("address", []):
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
                        id=patient_data.get("id"),
                        resourceType=patient_data.get("resourceType", "Patient"),
                        identifier=identifiers,
                        name=names,
                        gender=patient_data.get("gender"),
                        birthDate=patient_data.get("birthDate"),
                        active=patient_data.get("active", True),
                        telecom=telecoms,
                        address=addresses
                    )
                except Exception as e:
                    print(f"Error creating Patient object: {str(e)}")
                    return None
        except Exception as e:
            print(f"Error getting patient: {str(e)}")
            return None

    @strawberry.field
    async def search_patients(self, info, name: Optional[str] = None) -> List[Patient]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Search patients in FHIR service
        try:
            async with httpx.AsyncClient() as client:
                params = {}
                if name:
                    params["name"] = name

                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient",
                    params=params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                patients_data = response.json()
                result_patients = []

                # Convert each patient to a Patient object
                for patient_data in patients_data:
                    try:
                        # Handle identifier list
                        identifiers = []
                        for ident in patient_data.get("identifier", []):
                            identifiers.append(
                                Identifier(
                                    system=ident.get("system", ""),
                                    value=ident.get("value", ""),
                                    use=ident.get("use")
                                )
                            )

                        # Handle name list
                        names = []
                        for name in patient_data.get("name", []):
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
                        if "telecom" in patient_data:
                            telecoms = []
                            for telecom in patient_data.get("telecom", []):
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
                        if "address" in patient_data:
                            addresses = []
                            for addr in patient_data.get("address", []):
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
                        patient_obj = Patient(
                            id=patient_data.get("id"),
                            resourceType=patient_data.get("resourceType", "Patient"),
                            identifier=identifiers,
                            name=names,
                            gender=patient_data.get("gender"),
                            birthDate=patient_data.get("birthDate"),
                            active=patient_data.get("active", True),
                            telecom=telecoms,
                            address=addresses
                        )
                        result_patients.append(patient_obj)
                    except Exception as e:
                        print(f"Error creating Patient object: {str(e)}")
                        # Skip this patient and continue with the next one
                        continue

                return result_patients
        except Exception as e:
            print(f"Error searching patients: {str(e)}")
            return []

    # FHIR resource queries
    @strawberry.field
    async def observations(self, info, category: Optional[str] = None, code: Optional[str] = None) -> List[LabResult]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            params = {}
            if category:
                params["category"] = category
            if code:
                params["code"] = code

            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                # Fallback to FHIR service if Observation service fails
                fallback_params = {}
                if category:
                    fallback_params["category"] = category
                if code:
                    fallback_params["code"] = code

                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Observation",
                    params=fallback_params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                observations_data = response.json()

                # Process observations
                processed_observations = []
                for obs in observations_data:
                    # Create a new LabResult object
                    lab_result = LabResult(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        status=obs.get('status', ''),
                        category=obs.get('category', ''),
                        effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                    )
                    processed_observations.append(lab_result)

                return processed_observations

            # Process observations from Observation service
            observations_data = response.json()

            # Process observations
            processed_observations = []
            for obs in observations_data:
                # Create a new LabResult object
                lab_result = LabResult(
                    id=str(obs.get('_id', obs.get('id', ''))),
                    status=obs.get('status', ''),
                    category=obs.get('category', ''),
                    effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                )
                processed_observations.append(lab_result)

            return processed_observations

    @strawberry.field
    async def patient_observations(self, info, patient_id: str, code: Optional[str] = None, category: Optional[str] = None) -> List[CompleteObservation]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Validate patient ID
        try:
            async with httpx.AsyncClient() as client:
                patient_response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}",
                    headers={"Authorization": auth_header}
                )

                # If patient not found, return empty list
                if patient_response.status_code != 200:
                    print(f"Patient with ID {patient_id} not found")
                    return []
        except Exception as e:
            print(f"Error validating patient: {str(e)}")
            # Continue with the request even if patient validation fails

        # Get observations from Observation service
        async with httpx.AsyncClient() as client:
            params = {}
            if code:
                params["code"] = code
            if category:
                params["category"] = category

            # Try to get observations from Observation service first
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                params={"patient_id": patient_id, **params},
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                # Fallback to FHIR service if Observation service fails
                params = {"subject": f"Patient/{patient_id}"}
                if code:
                    params["code"] = code
                if category:
                    params["category"] = category

                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Observation",
                    params=params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                observations_data = response.json()

                # Process observations
                complete_observations = []
                for obs in observations_data:
                    try:
                        # Create code object if it exists
                        code_obj = None
                        if 'code' in obs and isinstance(obs['code'], dict):
                            code_obj = ObservationCode(
                                system=obs['code'].get('system', ''),
                                code=obs['code'].get('code', ''),
                                display=obs['code'].get('display', '')
                            )

                        # Create subject object if it exists
                        subject_obj = None
                        if 'subject' in obs and isinstance(obs['subject'], dict):
                            subject_obj = ObservationSubject(
                                reference=obs['subject'].get('reference', '')
                            )

                        # Create value quantity object if it exists
                        value_quantity_obj = None
                        if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                            vq = obs['value_quantity']
                            value_quantity_obj = ObservationValueQuantity(
                                value=float(vq.get('value', 0)),
                                unit=vq.get('unit', ''),
                                system=vq.get('system', ''),
                                code=vq.get('code', '')
                            )

                        # Create interpretation objects if they exist
                        interpretation_objs = []
                        if 'interpretation' in obs and isinstance(obs['interpretation'], list):
                            for interp in obs['interpretation']:
                                if isinstance(interp, dict):
                                    interpretation_objs.append(ObservationInterpretation(
                                        system=interp.get('system', ''),
                                        code=interp.get('code', ''),
                                        display=interp.get('display', '')
                                    ))

                        # Create reference range objects if they exist
                        reference_range_objs = []
                        if 'reference_range' in obs and isinstance(obs['reference_range'], list):
                            for range_item in obs['reference_range']:
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
                        complete_obs = CompleteObservation(
                            id=str(obs.get('_id', obs.get('id', ''))),
                            status=obs.get('status', ''),
                            category=obs.get('category', ''),
                            type=obs.get('category', 'observation'),  # Default to 'observation' if category not specified
                            code=code_obj,
                            subject=subject_obj,
                            effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                            value_quantity=value_quantity_obj,
                            interpretation=interpretation_objs,
                            reference_range=reference_range_objs
                        )
                        complete_observations.append(complete_obs)
                    except Exception as e:
                        print(f"Error processing observation: {str(e)}")
                        print(f"Observation data: {obs}")

                return complete_observations

            # Process observations from Observation service
            observations_data = response.json()

            # Process observations
            complete_observations = []
            for obs in observations_data:
                try:
                    # Create code object if it exists
                    code_obj = None
                    if 'code' in obs and isinstance(obs['code'], dict):
                        code_obj = ObservationCode(
                            system=obs['code'].get('system', ''),
                            code=obs['code'].get('code', ''),
                            display=obs['code'].get('display', '')
                        )

                    # Create subject object if it exists
                    subject_obj = None
                    if 'subject' in obs and isinstance(obs['subject'], dict):
                        subject_obj = ObservationSubject(
                            reference=obs['subject'].get('reference', '')
                        )

                    # Create value quantity object if it exists
                    value_quantity_obj = None
                    if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                        vq = obs['value_quantity']
                        value_quantity_obj = ObservationValueQuantity(
                            value=float(vq.get('value', 0)),
                            unit=vq.get('unit', ''),
                            system=vq.get('system', ''),
                            code=vq.get('code', '')
                        )

                    # Create interpretation objects if they exist
                    interpretation_objs = []
                    if 'interpretation' in obs and isinstance(obs['interpretation'], list):
                        for interp in obs['interpretation']:
                            if isinstance(interp, dict):
                                interpretation_objs.append(ObservationInterpretation(
                                    system=interp.get('system', ''),
                                    code=interp.get('code', ''),
                                    display=interp.get('display', '')
                                ))

                    # Create reference range objects if they exist
                    reference_range_objs = []
                    if 'reference_range' in obs and isinstance(obs['reference_range'], list):
                        for range_item in obs['reference_range']:
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
                    complete_obs = CompleteObservation(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        status=obs.get('status', ''),
                        category=obs.get('category', ''),
                        type=obs.get('category', 'observation'),  # Default to 'observation' if category not specified
                        code=code_obj,
                        subject=subject_obj,
                        effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                        value_quantity=value_quantity_obj,
                        interpretation=interpretation_objs,
                        reference_range=reference_range_objs
                    )
                    complete_observations.append(complete_obs)
                except Exception as e:
                    print(f"Error processing observation: {str(e)}")
                    print(f"Observation data: {obs}")

            return complete_observations

    @strawberry.field
    async def patient_lab_results(self, info, patient_id: str, code: Optional[str] = None) -> List[LabResult]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get lab results from Observation service
        async with httpx.AsyncClient() as client:
            params = {}
            if code:
                params["code"] = code

            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/laboratory/patient/{patient_id}",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                # Fallback to FHIR service if Observation service fails
                params = {
                    "subject": f"Patient/{patient_id}",
                    "category": "laboratory"
                }
                if code:
                    params["code"] = code

                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Observation",
                    params=params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                observations_data = response.json()

                # Process observations
                processed_observations = []
                for obs in observations_data:
                    # Create a new LabResult object
                    lab_result = LabResult(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        status=obs.get('status', ''),
                        category=obs.get('category', ''),
                        effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                    )
                    processed_observations.append(lab_result)

                return processed_observations

            # Process observations from Observation service
            observations_data = response.json()

            # Process observations
            processed_observations = []
            for obs in observations_data:
                # Create a new LabResult object
                lab_result = LabResult(
                    id=str(obs.get('_id', obs.get('id', ''))),
                    status=obs.get('status', ''),
                    category=obs.get('category', ''),
                    effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                )
                processed_observations.append(lab_result)

            return processed_observations



    @strawberry.field
    async def observation_codes(self, info) -> List[ObservationCodeEntry]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process observation codes
            code_entries = []
            for obs in observations_data:
                if 'code' in obs and isinstance(obs['code'], dict):
                    code_entry = ObservationCodeEntry(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        code=obs['code'].get('code', ''),
                        system=obs['code'].get('system', ''),
                        display=obs['code'].get('display', '')
                    )
                    code_entries.append(code_entry)

            return code_entries

    @strawberry.field
    async def observation_subjects(self, info) -> List[ObservationSubjectEntry]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process observation subjects
            subject_entries = []
            for obs in observations_data:
                if 'subject' in obs and isinstance(obs['subject'], dict):
                    subject_entry = ObservationSubjectEntry(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        reference=obs['subject'].get('reference', '')
                    )
                    subject_entries.append(subject_entry)

            return subject_entries

    @strawberry.field
    async def observation_values(self, info) -> List[ObservationValueEntry]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process observation values
            value_entries = []
            for obs in observations_data:
                if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                    value_entry = ObservationValueEntry(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        value=float(obs['value_quantity'].get('value', 0)),
                        unit=obs['value_quantity'].get('unit', ''),
                        system=obs['value_quantity'].get('system', ''),
                        code=obs['value_quantity'].get('code', '')
                    )
                    value_entries.append(value_entry)

            return value_entries

    @strawberry.field
    async def observation_interpretations(self, info) -> List[ObservationInterpretationEntry]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process observation interpretations
            interpretation_entries = []
            for obs in observations_data:
                if 'interpretation' in obs and isinstance(obs['interpretation'], list):
                    for interp in obs['interpretation']:
                        if isinstance(interp, dict):
                            interpretation_entry = ObservationInterpretationEntry(
                                id=str(obs.get('_id', obs.get('id', ''))),
                                system=interp.get('system', ''),
                                code=interp.get('code', ''),
                                display=interp.get('display', '')
                            )
                            interpretation_entries.append(interpretation_entry)

            return interpretation_entries

    @strawberry.field
    async def complete_observations(self, info, category: Optional[str] = None) -> List[CompleteObservation]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            params = {}
            if category:
                params["category"] = category

            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                # Fallback to FHIR service if Observation service fails
                fallback_params = {}
                if category:
                    fallback_params["category"] = category

                response = await client.get(
                    f"{settings.FHIR_SERVICE_URL}/api/fhir/Observation",
                    params=fallback_params,
                    headers={"Authorization": auth_header}
                )

                if response.status_code != 200:
                    return []

                observations_data = response.json()

                # Process observations
                complete_observations = []
                for obs in observations_data:
                    try:
                        # Create code object if it exists
                        code_obj = None
                        if 'code' in obs and isinstance(obs['code'], dict):
                            code_obj = ObservationCode(
                                system=obs['code'].get('system', ''),
                                code=obs['code'].get('code', ''),
                                display=obs['code'].get('display', '')
                            )

                        # Create subject object if it exists
                        subject_obj = None
                        if 'subject' in obs and isinstance(obs['subject'], dict):
                            subject_obj = ObservationSubject(
                                reference=obs['subject'].get('reference', '')
                            )

                        # Create value quantity object if it exists
                        value_quantity_obj = None
                        if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                            vq = obs['value_quantity']
                            value_quantity_obj = ObservationValueQuantity(
                                value=float(vq.get('value', 0)),
                                unit=vq.get('unit', ''),
                                system=vq.get('system', ''),
                                code=vq.get('code', '')
                            )

                        # Create interpretation objects if they exist
                        interpretation_objs = []
                        if 'interpretation' in obs and isinstance(obs['interpretation'], list):
                            for interp in obs['interpretation']:
                                if isinstance(interp, dict):
                                    interpretation_objs.append(ObservationInterpretation(
                                        system=interp.get('system', ''),
                                        code=interp.get('code', ''),
                                        display=interp.get('display', '')
                                    ))

                        # Create reference range objects if they exist
                        reference_range_objs = []
                        if 'reference_range' in obs and isinstance(obs['reference_range'], list):
                            for range_item in obs['reference_range']:
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
                        complete_obs = CompleteObservation(
                            id=str(obs.get('_id', obs.get('id', ''))),
                            status=obs.get('status', ''),
                            category=obs.get('category', ''),
                            type=obs.get('category', 'observation'),  # Default to 'observation' if category not specified
                            code=code_obj,
                            subject=subject_obj,
                            effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                            value_quantity=value_quantity_obj,
                            interpretation=interpretation_objs,
                            reference_range=reference_range_objs
                        )
                        complete_observations.append(complete_obs)
                    except Exception as e:
                        print(f"Error processing observation: {str(e)}")
                        print(f"Observation data: {obs}")

                return complete_observations

            # Process observations from Observation service
            observations_data = response.json()

            # Process observations
            complete_observations = []
            for obs in observations_data:
                try:
                    # Create code object if it exists
                    code_obj = None
                    if 'code' in obs and isinstance(obs['code'], dict):
                        code_obj = ObservationCode(
                            system=obs['code'].get('system', ''),
                            code=obs['code'].get('code', ''),
                            display=obs['code'].get('display', '')
                        )

                    # Create subject object if it exists
                    subject_obj = None
                    if 'subject' in obs and isinstance(obs['subject'], dict):
                        subject_obj = ObservationSubject(
                            reference=obs['subject'].get('reference', '')
                        )

                    # Create value quantity object if it exists
                    value_quantity_obj = None
                    if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                        vq = obs['value_quantity']
                        value_quantity_obj = ObservationValueQuantity(
                            value=float(vq.get('value', 0)),
                            unit=vq.get('unit', ''),
                            system=vq.get('system', ''),
                            code=vq.get('code', '')
                        )

                    # Create interpretation objects if they exist
                    interpretation_objs = []
                    if 'interpretation' in obs and isinstance(obs['interpretation'], list):
                        for interp in obs['interpretation']:
                            if isinstance(interp, dict):
                                interpretation_objs.append(ObservationInterpretation(
                                    system=interp.get('system', ''),
                                    code=interp.get('code', ''),
                                    display=interp.get('display', '')
                                ))

                    # Create reference range objects if they exist
                    reference_range_objs = []
                    if 'reference_range' in obs and isinstance(obs['reference_range'], list):
                        for range_item in obs['reference_range']:
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
                    complete_obs = CompleteObservation(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        status=obs.get('status', ''),
                        category=obs.get('category', ''),
                        type=obs.get('category', 'observation'),  # Default to 'observation' if category not specified
                        code=code_obj,
                        subject=subject_obj,
                        effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                        value_quantity=value_quantity_obj,
                        interpretation=interpretation_objs,
                        reference_range=reference_range_objs
                    )
                    complete_observations.append(complete_obs)
                except Exception as e:
                    print(f"Error processing observation: {str(e)}")
                    print(f"Observation data: {obs}")

            return complete_observations

    @strawberry.field
    async def vital_signs(self, info) -> List[VitalSign]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get vital signs from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                params={"category": "vital-signs"},
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process vital signs
            vital_signs = []
            for obs in observations_data:
                try:
                    # Create code object if it exists
                    code_obj = None
                    if 'code' in obs and isinstance(obs['code'], dict):
                        code_obj = ObservationCode(
                            system=obs['code'].get('system', ''),
                            code=obs['code'].get('code', ''),
                            display=obs['code'].get('display', '')
                        )

                    # Create subject object if it exists
                    subject_obj = None
                    if 'subject' in obs and isinstance(obs['subject'], dict):
                        subject_obj = ObservationSubject(
                            reference=obs['subject'].get('reference', '')
                        )

                    # Create value quantity object if it exists
                    value_quantity_obj = None
                    if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                        vq = obs['value_quantity']
                        value_quantity_obj = ObservationValueQuantity(
                            value=float(vq.get('value', 0)),
                            unit=vq.get('unit', ''),
                            system=vq.get('system', ''),
                            code=vq.get('code', '')
                        )

                    # Create vital sign object
                    vital_sign = VitalSign(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        status=obs.get('status', ''),
                        category=obs.get('category', ''),
                        code=code_obj,
                        subject=subject_obj,
                        effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                        value_quantity=value_quantity_obj
                    )
                    vital_signs.append(vital_sign)
                except Exception as e:
                    print(f"Error processing vital sign: {str(e)}")
                    print(f"Vital sign data: {obs}")

            return vital_signs

    @strawberry.field
    async def medications(self, info) -> List[Medication]:
        """Get all medications"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        print(f"Fetching medications with auth header: {auth_header}")

        if not auth_header:
            print("No authorization header found")
            return []

        # Get medications from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medications"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                medications_data = response.json()
                print(f"Received {len(medications_data)} medications")
                result = []

                for med_data in medications_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle code
                        code_obj = None
                        if 'code' in med_data and isinstance(med_data['code'], dict):
                            coding_list = []
                            for coding in med_data['code'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_data['code'].get('text', '')
                            )

                        # Handle form
                        form_obj = None
                        if 'form' in med_data and isinstance(med_data['form'], dict):
                            coding_list = []
                            for coding in med_data['form'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            form_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_data['form'].get('text', '')
                            )

                        # Handle amount
                        amount_obj = None
                        if 'amount' in med_data and isinstance(med_data['amount'], dict):
                            amount_obj = Quantity(
                                value=float(med_data['amount'].get('value', 0)),
                                unit=med_data['amount'].get('unit', ''),
                                system=med_data['amount'].get('system', ''),
                                code=med_data['amount'].get('code', '')
                            )

                        # Create Medication object
                        medication = Medication(
                            id=med_data.get('id', ''),
                            resourceType=med_data.get('resourceType', 'Medication'),
                            status=med_data.get('status', ''),
                            code=code_obj,
                            form=form_obj,
                            amount=amount_obj,
                            ingredient=med_data.get('ingredient', []),
                            batch=med_data.get('batch', '')
                        )
                        result.append(medication)
                    except Exception as e:
                        print(f"Error creating Medication object: {str(e)}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching medications: {str(e)}")
            return []

    @strawberry.field
    async def medication(self, info, id: str) -> Optional[Medication]:
        """Get a medication by ID"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get medication from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medications/{id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return None

                med_data = response.json()

                try:
                    # Convert the REST response to a GraphQL type
                    # Handle code
                    code_obj = None
                    if 'code' in med_data and isinstance(med_data['code'], dict):
                        coding_list = []
                        for coding in med_data['code'].get('coding', []):
                            coding_list.append(
                                Coding(
                                    system=coding.get('system', ''),
                                    code=coding.get('code', ''),
                                    display=coding.get('display', '')
                                )
                            )
                        code_obj = CodeableConcept(
                            coding=coding_list,
                            text=med_data['code'].get('text', '')
                        )

                    # Handle form
                    form_obj = None
                    if 'form' in med_data and isinstance(med_data['form'], dict):
                        coding_list = []
                        for coding in med_data['form'].get('coding', []):
                            coding_list.append(
                                Coding(
                                    system=coding.get('system', ''),
                                    code=coding.get('code', ''),
                                    display=coding.get('display', '')
                                )
                            )
                        form_obj = CodeableConcept(
                            coding=coding_list,
                            text=med_data['form'].get('text', '')
                        )

                    # Handle amount
                    amount_obj = None
                    if 'amount' in med_data and isinstance(med_data['amount'], dict):
                        amount_obj = Quantity(
                            value=float(med_data['amount'].get('value', 0)),
                            unit=med_data['amount'].get('unit', ''),
                            system=med_data['amount'].get('system', ''),
                            code=med_data['amount'].get('code', '')
                        )

                    # Create Medication object
                    return Medication(
                        id=med_data.get('id', ''),
                        resourceType=med_data.get('resourceType', 'Medication'),
                        status=med_data.get('status', ''),
                        code=code_obj,
                        form=form_obj,
                        amount=amount_obj,
                        ingredient=med_data.get('ingredient', []),
                        batch=med_data.get('batch', '')
                    )
                except Exception as e:
                    print(f"Error creating Medication object: {str(e)}")
                    return None
        except Exception as e:
            print(f"Error fetching medication: {str(e)}")
            return None

    @strawberry.field
    async def medication_requests(self, info) -> List[MedicationRequest]:
        """Get all medication requests"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication requests from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-requests"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                med_requests_data = response.json()
                print(f"Received {len(med_requests_data)} medication requests")
                result = []

                # Process medication requests
                for med_req_data in med_requests_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_req_data and isinstance(med_req_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_req_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_req_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_req_data and isinstance(med_req_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_req_data['subject'].get('reference', ''),
                                display=med_req_data['subject'].get('display', '')
                            )

                        # Handle requester
                        requester_obj = None
                        if 'requester' in med_req_data and isinstance(med_req_data['requester'], dict):
                            requester_obj = Reference(
                                reference=med_req_data['requester'].get('reference', ''),
                                display=med_req_data['requester'].get('display', '')
                            )

                        # Handle dosageInstruction
                        dosage_list = None
                        if 'dosageInstruction' in med_req_data and isinstance(med_req_data['dosageInstruction'], list):
                            dosage_list = []
                            for dosage in med_req_data['dosageInstruction']:
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
                        if 'note' in med_req_data and isinstance(med_req_data['note'], list):
                            note_list = []
                            for note in med_req_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationRequest object
                        medication_request = MedicationRequest(
                            id=med_req_data.get('id', ''),
                            resourceType=med_req_data.get('resourceType', 'MedicationRequest'),
                            status=med_req_data.get('status', ''),
                            intent=med_req_data.get('intent', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            authoredOn=med_req_data.get('authoredOn', ''),
                            requester=requester_obj,
                            dosageInstruction=dosage_list,
                            note=note_list
                        )
                        result.append(medication_request)
                    except Exception as e:
                        print(f"Error creating MedicationRequest object: {str(e)}")
                        print(f"MedicationRequest data: {med_req_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching medication requests: {str(e)}")
            return []

    @strawberry.field
    async def patient_medication_requests(self, info, patient_id: str) -> List[MedicationRequest]:
        """Get medication requests for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication requests from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-requests/patient/{patient_id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                med_requests_data = response.json()
                print(f"Received {len(med_requests_data)} patient medication requests")
                result = []

                # Process medication requests
                for med_req_data in med_requests_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_req_data and isinstance(med_req_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_req_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_req_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_req_data and isinstance(med_req_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_req_data['subject'].get('reference', ''),
                                display=med_req_data['subject'].get('display', '')
                            )

                        # Handle requester
                        requester_obj = None
                        if 'requester' in med_req_data and isinstance(med_req_data['requester'], dict):
                            requester_obj = Reference(
                                reference=med_req_data['requester'].get('reference', ''),
                                display=med_req_data['requester'].get('display', '')
                            )

                        # Handle dosageInstruction
                        dosage_list = None
                        if 'dosageInstruction' in med_req_data and isinstance(med_req_data['dosageInstruction'], list):
                            dosage_list = []
                            for dosage in med_req_data['dosageInstruction']:
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
                        if 'note' in med_req_data and isinstance(med_req_data['note'], list):
                            note_list = []
                            for note in med_req_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationRequest object
                        medication_request = MedicationRequest(
                            id=med_req_data.get('id', ''),
                            resourceType=med_req_data.get('resourceType', 'MedicationRequest'),
                            status=med_req_data.get('status', ''),
                            intent=med_req_data.get('intent', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            authoredOn=med_req_data.get('authoredOn', ''),
                            requester=requester_obj,
                            dosageInstruction=dosage_list,
                            note=note_list
                        )
                        result.append(medication_request)
                    except Exception as e:
                        print(f"Error creating MedicationRequest object: {str(e)}")
                        print(f"MedicationRequest data: {med_req_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient medication requests: {str(e)}")
            return []

    @strawberry.field
    async def physical_measurements(self, info) -> List[PhysicalMeasurement]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get physical measurements from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                params={"category": "physical-measurements"},
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process physical measurements
            physical_measurements = []
            for obs in observations_data:
                try:
                    # Create code object if it exists
                    code_obj = None
                    if 'code' in obs and isinstance(obs['code'], dict):
                        code_obj = ObservationCode(
                            system=obs['code'].get('system', ''),
                            code=obs['code'].get('code', ''),
                            display=obs['code'].get('display', '')
                        )

                    # Create subject object if it exists
                    subject_obj = None
                    if 'subject' in obs and isinstance(obs['subject'], dict):
                        subject_obj = ObservationSubject(
                            reference=obs['subject'].get('reference', '')
                        )

                    # Create value quantity object if it exists
                    value_quantity_obj = None
                    if 'value_quantity' in obs and isinstance(obs['value_quantity'], dict):
                        vq = obs['value_quantity']
                        value_quantity_obj = ObservationValueQuantity(
                            value=float(vq.get('value', 0)),
                            unit=vq.get('unit', ''),
                            system=vq.get('system', ''),
                            code=vq.get('code', '')
                        )

                    # Create physical measurement object
                    physical_measurement = PhysicalMeasurement(
                        id=str(obs.get('_id', obs.get('id', ''))),
                        status=obs.get('status', ''),
                        category=obs.get('category', ''),
                        code=code_obj,
                        subject=subject_obj,
                        effective_datetime=obs.get('effective_datetime', obs.get('effectiveDateTime', None)),
                        value_quantity=value_quantity_obj
                    )
                    physical_measurements.append(physical_measurement)
                except Exception as e:
                    print(f"Error processing physical measurement: {str(e)}")
                    print(f"Physical measurement data: {obs}")

            return physical_measurements

    @strawberry.field
    async def observation_reference_ranges(self, info) -> List[ObservationReferenceRangeEntry]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all observations from Observation service
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{settings.OBSERVATION_SERVICE_URL}/api/observations",
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            observations_data = response.json()

            # Process observation reference ranges
            range_entries = []
            for obs in observations_data:
                if 'reference_range' in obs and isinstance(obs['reference_range'], list):
                    for range_item in obs['reference_range']:
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
                            range_entry = ObservationReferenceRangeEntry(
                                id=str(obs.get('_id', obs.get('id', ''))),
                                low=low,
                                high=high,
                                text=range_item.get('text', '')
                            )
                            range_entries.append(range_entry)

            return range_entries

    @strawberry.field
    async def patient_conditions(self, info, patient_id: str) -> List[Condition]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get conditions from FHIR service
        async with httpx.AsyncClient() as client:
            params = {"subject": f"Patient/{patient_id}"}
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/Condition",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            conditions_data = response.json()
            return [Condition(**condition) for condition in conditions_data]

    @strawberry.field
    async def patient_medications(self, info, patient_id: str) -> List[MedicationRequest]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication requests from FHIR service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-requests/patient/{patient_id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    # Try FHIR service as fallback
                    params = {"subject": f"Patient/{patient_id}"}
                    response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/MedicationRequest",
                        params=params,
                        headers={"Authorization": auth_header}
                    )

                    if response.status_code != 200:
                        print(f"FHIR fallback error: {response.text}")
                        return []

                medications_data = response.json()
                print(f"Received {len(medications_data)} patient medications")
                result = []

                # Process medication requests
                for med_req_data in medications_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_req_data and isinstance(med_req_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_req_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_req_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_req_data and isinstance(med_req_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_req_data['subject'].get('reference', ''),
                                display=med_req_data['subject'].get('display', '')
                            )

                        # Handle requester
                        requester_obj = None
                        if 'requester' in med_req_data and isinstance(med_req_data['requester'], dict):
                            requester_obj = Reference(
                                reference=med_req_data['requester'].get('reference', ''),
                                display=med_req_data['requester'].get('display', '')
                            )

                        # Handle dosageInstruction
                        dosage_list = None
                        if 'dosageInstruction' in med_req_data and isinstance(med_req_data['dosageInstruction'], list):
                            dosage_list = []
                            for dosage in med_req_data['dosageInstruction']:
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
                        if 'note' in med_req_data and isinstance(med_req_data['note'], list):
                            note_list = []
                            for note in med_req_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationRequest object
                        medication_request = MedicationRequest(
                            id=med_req_data.get('id', ''),
                            resourceType=med_req_data.get('resourceType', 'MedicationRequest'),
                            status=med_req_data.get('status', ''),
                            intent=med_req_data.get('intent', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            authoredOn=med_req_data.get('authoredOn', ''),
                            requester=requester_obj,
                            dosageInstruction=dosage_list,
                            note=note_list
                        )
                        result.append(medication_request)
                    except Exception as e:
                        print(f"Error creating MedicationRequest object: {str(e)}")
                        print(f"MedicationRequest data: {med_req_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient medications: {str(e)}")
            return []

    @strawberry.field
    async def medication_statements(self, info) -> List[MedicationStatement]:
        """Get all medication statements"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication statements from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-statements"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                med_statements_data = response.json()
                print(f"Received {len(med_statements_data)} medication statements")
                result = []

                # Process medication statements
                for med_stmt_data in med_statements_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_stmt_data and isinstance(med_stmt_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_stmt_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_stmt_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_stmt_data and isinstance(med_stmt_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_stmt_data['subject'].get('reference', ''),
                                display=med_stmt_data['subject'].get('display', '')
                            )

                        # Handle informationSource
                        info_source_obj = None
                        if 'informationSource' in med_stmt_data and isinstance(med_stmt_data['informationSource'], dict):
                            info_source_obj = Reference(
                                reference=med_stmt_data['informationSource'].get('reference', ''),
                                display=med_stmt_data['informationSource'].get('display', '')
                            )

                        # Handle dosage
                        dosage_list = None
                        if 'dosage' in med_stmt_data and isinstance(med_stmt_data['dosage'], list):
                            dosage_list = []
                            for dosage in med_stmt_data['dosage']:
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
                        if 'note' in med_stmt_data and isinstance(med_stmt_data['note'], list):
                            note_list = []
                            for note in med_stmt_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationStatement object
                        medication_statement = MedicationStatement(
                            id=med_stmt_data.get('id', ''),
                            resourceType=med_stmt_data.get('resourceType', 'MedicationStatement'),
                            status=med_stmt_data.get('status', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            effectiveDateTime=med_stmt_data.get('effectiveDateTime', ''),
                            dateAsserted=med_stmt_data.get('dateAsserted', ''),
                            informationSource=info_source_obj,
                            dosage=dosage_list,
                            note=note_list
                        )
                        result.append(medication_statement)
                    except Exception as e:
                        print(f"Error creating MedicationStatement object: {str(e)}")
                        print(f"MedicationStatement data: {med_stmt_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching medication statements: {str(e)}")
            return []

    @strawberry.field
    async def medication_statement(self, info, id: str) -> Optional[MedicationStatement]:
        """Get a medication statement by ID"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get medication statement from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-statements/{id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return None

                med_stmt_data = response.json()

                try:
                    # Convert the REST response to a GraphQL type
                    # Handle medicationCodeableConcept
                    med_code_obj = None
                    if 'medicationCodeableConcept' in med_stmt_data and isinstance(med_stmt_data['medicationCodeableConcept'], dict):
                        coding_list = []
                        for coding in med_stmt_data['medicationCodeableConcept'].get('coding', []):
                            coding_list.append(
                                Coding(
                                    system=coding.get('system', ''),
                                    code=coding.get('code', ''),
                                    display=coding.get('display', '')
                                )
                            )
                        med_code_obj = CodeableConcept(
                            coding=coding_list,
                            text=med_stmt_data['medicationCodeableConcept'].get('text', '')
                        )

                    # Handle subject
                    subject_obj = None
                    if 'subject' in med_stmt_data and isinstance(med_stmt_data['subject'], dict):
                        subject_obj = Reference(
                            reference=med_stmt_data['subject'].get('reference', ''),
                            display=med_stmt_data['subject'].get('display', '')
                        )

                    # Handle informationSource
                    info_source_obj = None
                    if 'informationSource' in med_stmt_data and isinstance(med_stmt_data['informationSource'], dict):
                        info_source_obj = Reference(
                            reference=med_stmt_data['informationSource'].get('reference', ''),
                            display=med_stmt_data['informationSource'].get('display', '')
                        )

                    # Handle dosage
                    dosage_list = None
                    if 'dosage' in med_stmt_data and isinstance(med_stmt_data['dosage'], list):
                        dosage_list = []
                        for dosage in med_stmt_data['dosage']:
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
                    if 'note' in med_stmt_data and isinstance(med_stmt_data['note'], list):
                        note_list = []
                        for note in med_stmt_data['note']:
                            note_list.append(
                                Annotation(
                                    text=note.get('text', ''),
                                    authorString=note.get('authorString'),
                                    time=note.get('time')
                                )
                            )

                    # Create MedicationStatement object
                    return MedicationStatement(
                        id=med_stmt_data.get('id', ''),
                        resourceType=med_stmt_data.get('resourceType', 'MedicationStatement'),
                        status=med_stmt_data.get('status', ''),
                        medicationCodeableConcept=med_code_obj,
                        subject=subject_obj,
                        effectiveDateTime=med_stmt_data.get('effectiveDateTime', ''),
                        dateAsserted=med_stmt_data.get('dateAsserted', ''),
                        informationSource=info_source_obj,
                        dosage=dosage_list,
                        note=note_list
                    )
                except Exception as e:
                    print(f"Error creating MedicationStatement object: {str(e)}")
                    return None
        except Exception as e:
            print(f"Error fetching medication statement: {str(e)}")
            return None

    @strawberry.field
    async def medication_administrations(self, info) -> List[MedicationAdministration]:
        """Get all medication administrations"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication administrations from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-administrations"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                med_admins_data = response.json()
                print(f"Received {len(med_admins_data)} medication administrations")
                result = []

                # Process medication administrations
                for med_admin_data in med_admins_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_admin_data and isinstance(med_admin_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_admin_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_admin_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_admin_data and isinstance(med_admin_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_admin_data['subject'].get('reference', ''),
                                display=med_admin_data['subject'].get('display', '')
                            )

                        # Handle performer
                        performer_list = None
                        if 'performer' in med_admin_data and isinstance(med_admin_data['performer'], list):
                            performer_list = []
                            for performer in med_admin_data['performer']:
                                if 'actor' in performer and isinstance(performer['actor'], dict):
                                    performer_list.append(
                                        Reference(
                                            reference=performer['actor'].get('reference', ''),
                                            display=performer['actor'].get('display', '')
                                        )
                                    )

                        # Handle request
                        request_obj = None
                        if 'request' in med_admin_data and isinstance(med_admin_data['request'], dict):
                            request_obj = Reference(
                                reference=med_admin_data['request'].get('reference', ''),
                                display=med_admin_data['request'].get('display', ''),
                                type=med_admin_data['request'].get('type'),
                                identifier=None
                            )

                        # Handle note
                        note_list = None
                        if 'note' in med_admin_data and isinstance(med_admin_data['note'], list):
                            note_list = []
                            for note in med_admin_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationAdministration object
                        medication_administration = MedicationAdministration(
                            id=med_admin_data.get('id', ''),
                            resourceType=med_admin_data.get('resourceType', 'MedicationAdministration'),
                            status=med_admin_data.get('status', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            effectiveDateTime=med_admin_data.get('effectiveDateTime', ''),
                            performer=performer_list,
                            request=request_obj,
                            dosage=med_admin_data.get('dosage', ''),
                            note=note_list
                        )
                        result.append(medication_administration)
                    except Exception as e:
                        print(f"Error creating MedicationAdministration object: {str(e)}")
                        print(f"MedicationAdministration data: {med_admin_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching medication administrations: {str(e)}")
            return []

    @strawberry.field
    async def medication_administration(self, info, id: str) -> Optional[MedicationAdministration]:
        """Get a medication administration by ID"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get medication administration from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-administrations/{id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return None

                med_admin_data = response.json()

                try:
                    # Convert the REST response to a GraphQL type
                    # Handle medicationCodeableConcept
                    med_code_obj = None
                    if 'medicationCodeableConcept' in med_admin_data and isinstance(med_admin_data['medicationCodeableConcept'], dict):
                        coding_list = []
                        for coding in med_admin_data['medicationCodeableConcept'].get('coding', []):
                            coding_list.append(
                                Coding(
                                    system=coding.get('system', ''),
                                    code=coding.get('code', ''),
                                    display=coding.get('display', '')
                                )
                            )
                        med_code_obj = CodeableConcept(
                            coding=coding_list,
                            text=med_admin_data['medicationCodeableConcept'].get('text', '')
                        )

                    # Handle subject
                    subject_obj = None
                    if 'subject' in med_admin_data and isinstance(med_admin_data['subject'], dict):
                        subject_obj = Reference(
                            reference=med_admin_data['subject'].get('reference', ''),
                            display=med_admin_data['subject'].get('display', '')
                        )

                    # Handle performer
                    performer_list = None
                    if 'performer' in med_admin_data and isinstance(med_admin_data['performer'], list):
                        performer_list = []
                        for performer in med_admin_data['performer']:
                            if 'actor' in performer and isinstance(performer['actor'], dict):
                                performer_list.append(
                                    Reference(
                                        reference=performer['actor'].get('reference', ''),
                                        display=performer['actor'].get('display', '')
                                    )
                                )

                    # Handle request
                    request_obj = None
                    if 'request' in med_admin_data and isinstance(med_admin_data['request'], dict):
                        request_obj = Reference(
                            reference=med_admin_data['request'].get('reference', ''),
                            display=med_admin_data['request'].get('display', '')
                        )

                    # Handle note
                    note_list = None
                    if 'note' in med_admin_data and isinstance(med_admin_data['note'], list):
                        note_list = []
                        for note in med_admin_data['note']:
                            note_list.append(
                                Annotation(
                                    text=note.get('text', ''),
                                    authorString=note.get('authorString'),
                                    time=note.get('time')
                                )
                            )

                    # Create MedicationAdministration object
                    return MedicationAdministration(
                        id=med_admin_data.get('id', ''),
                        resourceType=med_admin_data.get('resourceType', 'MedicationAdministration'),
                        status=med_admin_data.get('status', ''),
                        medicationCodeableConcept=med_code_obj,
                        subject=subject_obj,
                        effectiveDateTime=med_admin_data.get('effectiveDateTime', ''),
                        performer=performer_list,
                        request=request_obj,
                        dosage=med_admin_data.get('dosage', ''),
                        note=note_list
                    )
                except Exception as e:
                    print(f"Error creating MedicationAdministration object: {str(e)}")
                    return None
        except Exception as e:
            print(f"Error fetching medication administration: {str(e)}")
            return None

    @strawberry.field
    async def patient_medication_administrations(self, info, patient_id: str) -> List[MedicationAdministration]:
        """Get medication administrations for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication administrations from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-administrations/patient/{patient_id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                med_admins_data = response.json()
                print(f"Received {len(med_admins_data)} patient medication administrations")
                result = []

                # Process medication administrations
                for med_admin_data in med_admins_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_admin_data and isinstance(med_admin_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_admin_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_admin_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_admin_data and isinstance(med_admin_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_admin_data['subject'].get('reference', ''),
                                display=med_admin_data['subject'].get('display', '')
                            )

                        # Handle performer
                        performer_list = None
                        if 'performer' in med_admin_data and isinstance(med_admin_data['performer'], list):
                            performer_list = []
                            for performer in med_admin_data['performer']:
                                if 'actor' in performer and isinstance(performer['actor'], dict):
                                    performer_list.append(
                                        Reference(
                                            reference=performer['actor'].get('reference', ''),
                                            display=performer['actor'].get('display', '')
                                        )
                                    )

                        # Handle request
                        request_obj = None
                        if 'request' in med_admin_data and isinstance(med_admin_data['request'], dict):
                            request_obj = Reference(
                                reference=med_admin_data['request'].get('reference', ''),
                                display=med_admin_data['request'].get('display', ''),
                                type=med_admin_data['request'].get('type'),
                                identifier=None
                            )

                        # Handle note
                        note_list = None
                        if 'note' in med_admin_data and isinstance(med_admin_data['note'], list):
                            note_list = []
                            for note in med_admin_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationAdministration object
                        medication_administration = MedicationAdministration(
                            id=med_admin_data.get('id', ''),
                            resourceType=med_admin_data.get('resourceType', 'MedicationAdministration'),
                            status=med_admin_data.get('status', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            effectiveDateTime=med_admin_data.get('effectiveDateTime', ''),
                            performer=performer_list,
                            request=request_obj,
                            dosage=med_admin_data.get('dosage', ''),
                            note=note_list
                        )
                        result.append(medication_administration)
                    except Exception as e:
                        print(f"Error creating MedicationAdministration object: {str(e)}")
                        print(f"MedicationAdministration data: {med_admin_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient medication administrations: {str(e)}")
            return []

    @strawberry.field
    async def patient_medication_statements(self, info, patient_id: str) -> List[MedicationStatement]:
        """Get medication statements for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get medication statements from Medication service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.MEDICATION_SERVICE_URL}/api/medication-statements/patient/{patient_id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response headers: {response.headers}")
                print(f"Response URL after redirects: {response.url}")

                if response.status_code != 200:
                    print(f"Error response: {response.text}")
                    return []

                med_statements_data = response.json()
                print(f"Received {len(med_statements_data)} patient medication statements")
                result = []

                # Process medication statements
                for med_stmt_data in med_statements_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle medicationCodeableConcept
                        med_code_obj = None
                        if 'medicationCodeableConcept' in med_stmt_data and isinstance(med_stmt_data['medicationCodeableConcept'], dict):
                            coding_list = []
                            for coding in med_stmt_data['medicationCodeableConcept'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            med_code_obj = CodeableConcept(
                                coding=coding_list,
                                text=med_stmt_data['medicationCodeableConcept'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in med_stmt_data and isinstance(med_stmt_data['subject'], dict):
                            subject_obj = Reference(
                                reference=med_stmt_data['subject'].get('reference', ''),
                                display=med_stmt_data['subject'].get('display', '')
                            )

                        # Handle informationSource
                        info_source_obj = None
                        if 'informationSource' in med_stmt_data and isinstance(med_stmt_data['informationSource'], dict):
                            info_source_obj = Reference(
                                reference=med_stmt_data['informationSource'].get('reference', ''),
                                display=med_stmt_data['informationSource'].get('display', '')
                            )

                        # Handle dosage
                        dosage_list = None
                        if 'dosage' in med_stmt_data and isinstance(med_stmt_data['dosage'], list):
                            dosage_list = []
                            for dosage in med_stmt_data['dosage']:
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
                        if 'note' in med_stmt_data and isinstance(med_stmt_data['note'], list):
                            note_list = []
                            for note in med_stmt_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create MedicationStatement object
                        medication_statement = MedicationStatement(
                            id=med_stmt_data.get('id', ''),
                            resourceType=med_stmt_data.get('resourceType', 'MedicationStatement'),
                            status=med_stmt_data.get('status', ''),
                            medicationCodeableConcept=med_code_obj,
                            subject=subject_obj,
                            effectiveDateTime=med_stmt_data.get('effectiveDateTime', ''),
                            dateAsserted=med_stmt_data.get('dateAsserted', ''),
                            informationSource=info_source_obj,
                            dosage=dosage_list,
                            note=note_list
                        )
                        result.append(medication_statement)
                    except Exception as e:
                        print(f"Error creating MedicationStatement object: {str(e)}")
                        print(f"MedicationStatement data: {med_stmt_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient medication statements: {str(e)}")
            return []

    @strawberry.field
    async def patient_diagnostic_reports(self, info, patient_id: str) -> List[DiagnosticReport]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get diagnostic reports from FHIR service
        async with httpx.AsyncClient() as client:
            params = {"subject": f"Patient/{patient_id}"}
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/DiagnosticReport",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            reports_data = response.json()
            return [DiagnosticReport(**report) for report in reports_data]

    @strawberry.field
    async def encounters(self, info, status: Optional[str] = None) -> List[Encounter]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get all encounters from Encounter service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.ENCOUNTER_SERVICE_URL}/api/encounters"
                params = {}
                if status:
                    params["status"] = status
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Encounter service fails
                    fallback_params = {}
                    if status:
                        fallback_params["status"] = status
                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Encounter",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    encounters_data = fallback_response.json()
                else:
                    encounters_data = response.json()

                print(f"Received {len(encounters_data)} encounters")
                result = []
                for encounter_data in encounters_data:
                    # Handle 'class' field as a CodeableConcept
                    if 'class' in encounter_data:
                        class_data = encounter_data.pop('class')
                        if isinstance(class_data, dict):
                            # Create a CodeableConcept from the class data
                            coding_list = []
                            if 'system' in class_data and 'code' in class_data:
                                coding_list.append(Coding(
                                    system=class_data.get('system', ''),
                                    code=class_data.get('code', ''),
                                    display=class_data.get('display', '')
                                ))
                            encounter_data['class_'] = CodeableConcept(
                                coding=coding_list,
                                text=class_data.get('text', '')
                            )
                        else:
                            # If it's not a dict, just use it as is
                            encounter_data['class_'] = class_data

                    # Handle 'subject' field as a Reference
                    if 'subject' in encounter_data and isinstance(encounter_data['subject'], dict):
                        subject_data = encounter_data.pop('subject')
                        encounter_data['subject'] = Reference(
                            reference=subject_data.get('reference', ''),
                            display=subject_data.get('display', '')
                        )

                    # Handle meta field
                    if 'meta' in encounter_data and encounter_data['meta']:
                        meta_data = encounter_data.pop('meta')
                        encounter_data['meta'] = Meta(
                            versionId=meta_data.get('versionId'),
                            lastUpdated=meta_data.get('lastUpdated'),
                            source=meta_data.get('source'),
                            profile=meta_data.get('profile'),
                            security=None,  # Would need to convert to CodeableConcept if present
                            tag=None  # Would need to convert to CodeableConcept if present
                        )

                    result.append(Encounter(**encounter_data))
                return result
        except Exception as e:
            print(f"Error fetching encounters: {str(e)}")
            return []

    @strawberry.field
    async def encounter(self, info, id: str) -> Optional[Encounter]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get encounter from Encounter service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                url = f"{settings.ENCOUNTER_SERVICE_URL}/api/encounters/{id}"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Encounter service fails
                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Encounter/{id}",
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return None

                    encounter_data = fallback_response.json()
                else:
                    encounter_data = response.json()

                # Handle 'class' field as a CodeableConcept
                if 'class' in encounter_data:
                    class_data = encounter_data.pop('class')
                    if isinstance(class_data, dict):
                        # Create a CodeableConcept from the class data
                        coding_list = []
                        if 'system' in class_data and 'code' in class_data:
                            coding_list.append(Coding(
                                system=class_data.get('system', ''),
                                code=class_data.get('code', ''),
                                display=class_data.get('display', '')
                            ))
                        encounter_data['class_'] = CodeableConcept(
                            coding=coding_list,
                            text=class_data.get('text', '')
                        )
                    else:
                        # If it's not a dict, just use it as is
                        encounter_data['class_'] = class_data

                # Handle 'subject' field as a Reference
                if 'subject' in encounter_data and isinstance(encounter_data['subject'], dict):
                    subject_data = encounter_data.pop('subject')
                    encounter_data['subject'] = Reference(
                        reference=subject_data.get('reference', ''),
                        display=subject_data.get('display', '')
                    )

                # Handle 'period' field as a Period object
                if 'period' in encounter_data and isinstance(encounter_data['period'], dict):
                    period_data = encounter_data.pop('period')
                    encounter_data['period'] = Period(
                        start=period_data.get('start'),
                        end=period_data.get('end')
                    )

                # Handle 'type' field as a list of CodeableConcept objects
                if 'type' in encounter_data and isinstance(encounter_data['type'], list):
                    type_list = []
                    for type_item in encounter_data.pop('type'):
                        if isinstance(type_item, dict):
                            coding_list = []
                            if 'coding' in type_item and isinstance(type_item['coding'], list):
                                for coding in type_item['coding']:
                                    coding_list.append(Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    ))
                            type_list.append(CodeableConcept(
                                coding=coding_list,
                                text=type_item.get('text', '')
                            ))
                    encounter_data['type'] = type_list

                # Handle meta field
                if 'meta' in encounter_data and encounter_data['meta']:
                    meta_data = encounter_data.pop('meta')
                    encounter_data['meta'] = Meta(
                        versionId=meta_data.get('versionId'),
                        lastUpdated=meta_data.get('lastUpdated'),
                        source=meta_data.get('source'),
                        profile=meta_data.get('profile'),
                        security=None,  # Would need to convert to CodeableConcept if present
                        tag=None  # Would need to convert to CodeableConcept if present
                    )

                return Encounter(**encounter_data)
        except Exception as e:
            print(f"Error fetching encounter: {str(e)}")
            return None

    @strawberry.field
    async def patient_encounters(self, info, patient_id: str, status: Optional[str] = None) -> List[Encounter]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get encounters from Encounter service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if status:
                    params["status"] = status

                url = f"{settings.ENCOUNTER_SERVICE_URL}/api/encounters/patient/{patient_id}"
                print(f"Making request to: {url}")
                print(f"With params: {params}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Encounter service fails
                    fallback_params = {"subject": f"Patient/{patient_id}"}
                    if status:
                        fallback_params["status"] = status

                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Encounter",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    encounters_data = fallback_response.json()
                else:
                    encounters_data = response.json()

                print(f"Received {len(encounters_data)} encounters")

                # Additional filtering to ensure we only get encounters for this patient
                filtered_encounters = []
                for encounter in encounters_data:
                    if 'subject' in encounter and isinstance(encounter['subject'], dict):
                        subject_ref = encounter['subject'].get('reference', '')
                        if subject_ref == f"Patient/{patient_id}" or subject_ref.endswith(f"/{patient_id}"):
                            filtered_encounters.append(encounter)

                if len(filtered_encounters) != len(encounters_data):
                    print(f"Filtered to {len(filtered_encounters)} encounters for patient {patient_id} (removed {len(encounters_data) - len(filtered_encounters)} incorrect encounters)")
                    encounters_data = filtered_encounters

                result = []
                for encounter_data in encounters_data:
                    # Handle 'class' field as a CodeableConcept
                    if 'class' in encounter_data:
                        class_data = encounter_data.pop('class')
                        if isinstance(class_data, dict):
                            # Create a CodeableConcept from the class data
                            coding_list = []
                            if 'system' in class_data and 'code' in class_data:
                                coding_list.append(Coding(
                                    system=class_data.get('system', ''),
                                    code=class_data.get('code', ''),
                                    display=class_data.get('display', '')
                                ))
                            encounter_data['class_'] = CodeableConcept(
                                coding=coding_list,
                                text=class_data.get('text', '')
                            )
                        else:
                            # If it's not a dict, just use it as is
                            encounter_data['class_'] = class_data

                    # Handle 'subject' field as a Reference
                    if 'subject' in encounter_data and isinstance(encounter_data['subject'], dict):
                        subject_data = encounter_data.pop('subject')
                        encounter_data['subject'] = Reference(
                            reference=subject_data.get('reference', ''),
                            display=subject_data.get('display', '')
                        )

                    # Handle 'period' field as a Period object
                    if 'period' in encounter_data and isinstance(encounter_data['period'], dict):
                        period_data = encounter_data.pop('period')
                        encounter_data['period'] = Period(
                            start=period_data.get('start'),
                            end=period_data.get('end')
                        )

                    # Handle 'type' field as a list of CodeableConcept objects
                    if 'type' in encounter_data and isinstance(encounter_data['type'], list):
                        type_list = []
                        for type_item in encounter_data.pop('type'):
                            if isinstance(type_item, dict):
                                coding_list = []
                                if 'coding' in type_item and isinstance(type_item['coding'], list):
                                    for coding in type_item['coding']:
                                        coding_list.append(Coding(
                                            system=coding.get('system', ''),
                                            code=coding.get('code', ''),
                                            display=coding.get('display', '')
                                        ))
                                type_list.append(CodeableConcept(
                                    coding=coding_list,
                                    text=type_item.get('text', '')
                                ))
                        encounter_data['type'] = type_list

                    # Handle meta field
                    if 'meta' in encounter_data and encounter_data['meta']:
                        meta_data = encounter_data.pop('meta')
                        encounter_data['meta'] = Meta(
                            versionId=meta_data.get('versionId'),
                            lastUpdated=meta_data.get('lastUpdated'),
                            source=meta_data.get('source'),
                            profile=meta_data.get('profile'),
                            security=None,  # Would need to convert to CodeableConcept if present
                            tag=None  # Would need to convert to CodeableConcept if present
                        )

                    result.append(Encounter(**encounter_data))
                return result
        except Exception as e:
            print(f"Error fetching patient encounters: {str(e)}")
            return []

    @strawberry.field
    async def patient_documents(self, info, patient_id: str) -> List[DocumentReference]:
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get document references from FHIR service
        async with httpx.AsyncClient() as client:
            params = {"subject": f"Patient/{patient_id}"}
            response = await client.get(
                f"{settings.FHIR_SERVICE_URL}/api/fhir/DocumentReference",
                params=params,
                headers={"Authorization": auth_header}
            )

            if response.status_code != 200:
                return []

            documents_data = response.json()
            return [DocumentReference(**doc) for doc in documents_data]

    @strawberry.field
    async def conditions(self, info, clinical_status: Optional[str] = None, verification_status: Optional[str] = None, category: Optional[str] = None) -> List[Condition]:
        """Get all conditions"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get conditions from Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if clinical_status:
                    params["clinical_status"] = clinical_status
                if verification_status:
                    params["verification_status"] = verification_status
                if category:
                    params["category"] = category

                url = f"{settings.CONDITION_SERVICE_URL}/api/conditions"
                print(f"Making request to: {url}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Condition service fails
                    fallback_params = {}
                    if clinical_status:
                        fallback_params["clinical-status"] = clinical_status
                    if verification_status:
                        fallback_params["verification-status"] = verification_status
                    if category:
                        fallback_params["category"] = category

                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Condition",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    conditions_data = fallback_response.json()
                else:
                    conditions_data = response.json()

                print(f"Received {len(conditions_data)} conditions")
                result = []

                for condition_data in conditions_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle clinicalStatus
                        clinical_status_obj = None
                        if 'clinicalStatus' in condition_data and isinstance(condition_data['clinicalStatus'], dict):
                            coding_list = []
                            for coding in condition_data['clinicalStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            clinical_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=condition_data['clinicalStatus'].get('text', '')
                            )

                        # Handle verificationStatus
                        verification_status_obj = None
                        if 'verificationStatus' in condition_data and isinstance(condition_data['verificationStatus'], dict):
                            coding_list = []
                            for coding in condition_data['verificationStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            verification_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=condition_data['verificationStatus'].get('text', '')
                            )

                        # Handle category
                        category_list = []
                        if 'category' in condition_data and isinstance(condition_data['category'], list):
                            for cat in condition_data['category']:
                                if isinstance(cat, dict):
                                    coding_list = []
                                    for coding in cat.get('coding', []):
                                        coding_list.append(
                                            Coding(
                                                system=coding.get('system', ''),
                                                code=coding.get('code', ''),
                                                display=coding.get('display', '')
                                            )
                                        )
                                    category_list.append(
                                        CodeableConcept(
                                            coding=coding_list,
                                            text=cat.get('text', '')
                                        )
                                    )

                        # Handle code
                        code_obj = None
                        if 'code' in condition_data and isinstance(condition_data['code'], dict):
                            coding_list = []
                            for coding in condition_data['code'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            code_obj = CodeableConcept(
                                coding=coding_list,
                                text=condition_data['code'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in condition_data and isinstance(condition_data['subject'], dict):
                            subject_obj = Reference(
                                reference=condition_data['subject'].get('reference', ''),
                                display=condition_data['subject'].get('display', '')
                            )

                        # Handle note
                        note_list = None
                        if 'note' in condition_data and isinstance(condition_data['note'], list):
                            note_list = []
                            for note in condition_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create Condition object
                        condition = Condition(
                            id=condition_data.get('id', ''),
                            resourceType=condition_data.get('resourceType', 'Condition'),
                            clinicalStatus=clinical_status_obj,
                            verificationStatus=verification_status_obj,
                            category=category_list,
                            code=code_obj,
                            subject=subject_obj,
                            onsetDateTime=condition_data.get('onsetDateTime'),
                            abatementDateTime=condition_data.get('abatementDateTime'),
                            recordedDate=condition_data.get('recordedDate'),
                            note=note_list
                        )
                        result.append(condition)
                    except Exception as e:
                        print(f"Error creating Condition object: {str(e)}")
                        print(f"Condition data: {condition_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching conditions: {str(e)}")
            return []

    @strawberry.field
    async def patient_conditions(self, info, patient_id: str, clinical_status: Optional[str] = None, verification_status: Optional[str] = None, category: Optional[str] = None) -> List[Condition]:
        """Get conditions for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get conditions from Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if clinical_status:
                    params["clinical_status"] = clinical_status
                if verification_status:
                    params["verification_status"] = verification_status
                if category:
                    params["category"] = category

                # Use the patient-specific endpoint
                url = f"{settings.CONDITION_SERVICE_URL}/api/conditions/patient/{patient_id}"
                print(f"Making request to: {url}")
                print(f"With params: {params}")
                print(f"With auth header: {auth_header}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response body: {response.text}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Condition service fails
                    fallback_params = {"subject": f"Patient/{patient_id}"}
                    if clinical_status:
                        fallback_params["clinical-status"] = clinical_status
                    if verification_status:
                        fallback_params["verification-status"] = verification_status
                    if category:
                        fallback_params["category"] = category

                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Condition",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    conditions_data = fallback_response.json()
                else:
                    conditions_data = response.json()

                print(f"Received {len(conditions_data)} patient conditions")

                # Additional filtering to ensure we only get conditions for this patient
                filtered_conditions = []
                for condition in conditions_data:
                    if 'subject' in condition and isinstance(condition['subject'], dict):
                        subject_ref = condition['subject'].get('reference', '')
                        if subject_ref == f"Patient/{patient_id}" or subject_ref.endswith(f"/{patient_id}"):
                            filtered_conditions.append(condition)

                if len(filtered_conditions) != len(conditions_data):
                    print(f"Filtered to {len(filtered_conditions)} conditions for patient {patient_id} (removed {len(conditions_data) - len(filtered_conditions)} incorrect conditions)")
                    conditions_data = filtered_conditions

                result = []

                for condition_data in conditions_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle clinicalStatus
                        clinical_status_obj = None
                        if 'clinicalStatus' in condition_data and isinstance(condition_data['clinicalStatus'], dict):
                            coding_list = []
                            for coding in condition_data['clinicalStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            clinical_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=condition_data['clinicalStatus'].get('text', '')
                            )

                        # Handle verificationStatus
                        verification_status_obj = None
                        if 'verificationStatus' in condition_data and isinstance(condition_data['verificationStatus'], dict):
                            coding_list = []
                            for coding in condition_data['verificationStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            verification_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=condition_data['verificationStatus'].get('text', '')
                            )

                        # Handle category
                        category_list = []
                        if 'category' in condition_data and isinstance(condition_data['category'], list):
                            for cat in condition_data['category']:
                                if isinstance(cat, dict):
                                    coding_list = []
                                    for coding in cat.get('coding', []):
                                        coding_list.append(
                                            Coding(
                                                system=coding.get('system', ''),
                                                code=coding.get('code', ''),
                                                display=coding.get('display', '')
                                            )
                                        )
                                    category_list.append(
                                        CodeableConcept(
                                            coding=coding_list,
                                            text=cat.get('text', '')
                                        )
                                    )

                        # Handle code
                        code_obj = None
                        if 'code' in condition_data and isinstance(condition_data['code'], dict):
                            coding_list = []
                            for coding in condition_data['code'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            code_obj = CodeableConcept(
                                coding=coding_list,
                                text=condition_data['code'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in condition_data and isinstance(condition_data['subject'], dict):
                            subject_obj = Reference(
                                reference=condition_data['subject'].get('reference', ''),
                                display=condition_data['subject'].get('display', '')
                            )

                        # Handle note
                        note_list = None
                        if 'note' in condition_data and isinstance(condition_data['note'], list):
                            note_list = []
                            for note in condition_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create Condition object
                        condition = Condition(
                            id=condition_data.get('id', ''),
                            resourceType=condition_data.get('resourceType', 'Condition'),
                            clinicalStatus=clinical_status_obj,
                            verificationStatus=verification_status_obj,
                            category=category_list,
                            code=code_obj,
                            subject=subject_obj,
                            onsetDateTime=condition_data.get('onsetDateTime'),
                            abatementDateTime=condition_data.get('abatementDateTime'),
                            recordedDate=condition_data.get('recordedDate'),
                            note=note_list
                        )
                        result.append(condition)
                    except Exception as e:
                        print(f"Error creating Condition object: {str(e)}")
                        print(f"Condition data: {condition_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient conditions: {str(e)}")
            return []

    @strawberry.field
    async def patient_problems(self, info, patient_id: str, clinical_status: Optional[str] = None, verification_status: Optional[str] = None) -> List[ProblemListItem]:
        """Get problem list items for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get problems from Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if clinical_status:
                    params["clinical_status"] = clinical_status
                if verification_status:
                    params["verification_status"] = verification_status

                url = f"{settings.CONDITION_SERVICE_URL}/api/conditions/patient/{patient_id}/problems"
                print(f"Making request to: {url}")
                print(f"With params: {params}")
                print(f"With auth header: {auth_header}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response body: {response.text}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Condition service fails
                    fallback_params = {
                        "subject": f"Patient/{patient_id}",
                        "category": "problem-list-item"
                    }
                    if clinical_status:
                        fallback_params["clinical-status"] = clinical_status
                    if verification_status:
                        fallback_params["verification-status"] = verification_status

                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/fhir/Condition",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    problems_data = fallback_response.json()
                else:
                    problems_data = response.json()

                print(f"Received {len(problems_data)} patient problems")

                # Filter problems by patient ID and category since the API returns all conditions
                filtered_problems = []
                for problem in problems_data:
                    if 'subject' in problem and isinstance(problem['subject'], dict):
                        subject_ref = problem['subject'].get('reference', '')
                        is_correct_patient = subject_ref == f"Patient/{patient_id}" or subject_ref.endswith(f"/{patient_id}")

                        # Check if it's a problem list item
                        is_problem = False
                        if 'category' in problem and isinstance(problem['category'], list):
                            for cat in problem['category']:
                                if isinstance(cat, dict) and 'coding' in cat:
                                    for coding in cat['coding']:
                                        if coding.get('code') == 'problem-list-item':
                                            is_problem = True
                                            break
                                if is_problem:
                                    break

                        if is_correct_patient and is_problem:
                            filtered_problems.append(problem)

                print(f"Filtered to {len(filtered_problems)} problems for patient {patient_id}")
                problems_data = filtered_problems

                result = []

                for problem_data in problems_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle clinicalStatus
                        clinical_status_obj = None
                        if 'clinicalStatus' in problem_data and isinstance(problem_data['clinicalStatus'], dict):
                            coding_list = []
                            for coding in problem_data['clinicalStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            clinical_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=problem_data['clinicalStatus'].get('text', '')
                            )

                        # Handle verificationStatus
                        verification_status_obj = None
                        if 'verificationStatus' in problem_data and isinstance(problem_data['verificationStatus'], dict):
                            coding_list = []
                            for coding in problem_data['verificationStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            verification_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=problem_data['verificationStatus'].get('text', '')
                            )

                        # Handle category
                        category_list = []
                        if 'category' in problem_data and isinstance(problem_data['category'], list):
                            for cat in problem_data['category']:
                                if isinstance(cat, dict):
                                    coding_list = []
                                    for coding in cat.get('coding', []):
                                        coding_list.append(
                                            Coding(
                                                system=coding.get('system', ''),
                                                code=coding.get('code', ''),
                                                display=coding.get('display', '')
                                            )
                                        )
                                    category_list.append(
                                        CodeableConcept(
                                            coding=coding_list,
                                            text=cat.get('text', '')
                                        )
                                    )

                        # Handle code
                        code_obj = None
                        if 'code' in problem_data and isinstance(problem_data['code'], dict):
                            coding_list = []
                            for coding in problem_data['code'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            code_obj = CodeableConcept(
                                coding=coding_list,
                                text=problem_data['code'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in problem_data and isinstance(problem_data['subject'], dict):
                            subject_obj = Reference(
                                reference=problem_data['subject'].get('reference', ''),
                                display=problem_data['subject'].get('display', '')
                            )

                        # Handle note
                        note_list = None
                        if 'note' in problem_data and isinstance(problem_data['note'], list):
                            note_list = []
                            for note in problem_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create ProblemListItem object
                        problem = ProblemListItem(
                            id=problem_data.get('id', ''),
                            resourceType=problem_data.get('resourceType', 'Condition'),
                            clinicalStatus=clinical_status_obj,
                            verificationStatus=verification_status_obj,
                            category=category_list,
                            code=code_obj,
                            subject=subject_obj,
                            onsetDateTime=problem_data.get('onsetDateTime'),
                            abatementDateTime=problem_data.get('abatementDateTime'),
                            recordedDate=problem_data.get('recordedDate'),
                            note=note_list
                        )
                        result.append(problem)
                    except Exception as e:
                        print(f"Error creating ProblemListItem object: {str(e)}")
                        print(f"Problem data: {problem_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient problems: {str(e)}")
            return []

    @strawberry.field
    async def patient_diagnoses(self, info, patient_id: str, clinical_status: Optional[str] = None, verification_status: Optional[str] = None) -> List[Diagnosis]:
        """Get diagnoses for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get diagnoses from Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if clinical_status:
                    params["clinical_status"] = clinical_status
                if verification_status:
                    params["verification_status"] = verification_status

                url = f"{settings.CONDITION_SERVICE_URL}/api/conditions/patient/{patient_id}/diagnoses"
                print(f"Making request to: {url}")
                print(f"With params: {params}")
                print(f"With auth header: {auth_header}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response body: {response.text}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Condition service fails
                    fallback_params = {
                        "subject": f"Patient/{patient_id}",
                        "category": "encounter-diagnosis"
                    }
                    if clinical_status:
                        fallback_params["clinical-status"] = clinical_status
                    if verification_status:
                        fallback_params["verification-status"] = verification_status

                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/fhir/Condition",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    diagnoses_data = fallback_response.json()
                else:
                    diagnoses_data = response.json()

                print(f"Received {len(diagnoses_data)} patient diagnoses")

                # Filter diagnoses by patient ID and category since the API returns all conditions
                filtered_diagnoses = []
                for diagnosis in diagnoses_data:
                    if 'subject' in diagnosis and isinstance(diagnosis['subject'], dict):
                        subject_ref = diagnosis['subject'].get('reference', '')
                        is_correct_patient = subject_ref == f"Patient/{patient_id}" or subject_ref.endswith(f"/{patient_id}")

                        # Check if it's a diagnosis
                        is_diagnosis = False
                        if 'category' in diagnosis and isinstance(diagnosis['category'], list):
                            for cat in diagnosis['category']:
                                if isinstance(cat, dict) and 'coding' in cat:
                                    for coding in cat['coding']:
                                        if coding.get('code') == 'encounter-diagnosis':
                                            is_diagnosis = True
                                            break
                                if is_diagnosis:
                                    break

                        if is_correct_patient and is_diagnosis:
                            filtered_diagnoses.append(diagnosis)

                print(f"Filtered to {len(filtered_diagnoses)} diagnoses for patient {patient_id}")
                diagnoses_data = filtered_diagnoses

                result = []

                for diagnosis_data in diagnoses_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle clinicalStatus
                        clinical_status_obj = None
                        if 'clinicalStatus' in diagnosis_data and isinstance(diagnosis_data['clinicalStatus'], dict):
                            coding_list = []
                            for coding in diagnosis_data['clinicalStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            clinical_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=diagnosis_data['clinicalStatus'].get('text', '')
                            )

                        # Handle verificationStatus
                        verification_status_obj = None
                        if 'verificationStatus' in diagnosis_data and isinstance(diagnosis_data['verificationStatus'], dict):
                            coding_list = []
                            for coding in diagnosis_data['verificationStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            verification_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=diagnosis_data['verificationStatus'].get('text', '')
                            )

                        # Handle category
                        category_list = []
                        if 'category' in diagnosis_data and isinstance(diagnosis_data['category'], list):
                            for cat in diagnosis_data['category']:
                                if isinstance(cat, dict):
                                    coding_list = []
                                    for coding in cat.get('coding', []):
                                        coding_list.append(
                                            Coding(
                                                system=coding.get('system', ''),
                                                code=coding.get('code', ''),
                                                display=coding.get('display', '')
                                            )
                                        )
                                    category_list.append(
                                        CodeableConcept(
                                            coding=coding_list,
                                            text=cat.get('text', '')
                                        )
                                    )

                        # Handle code
                        code_obj = None
                        if 'code' in diagnosis_data and isinstance(diagnosis_data['code'], dict):
                            coding_list = []
                            for coding in diagnosis_data['code'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            code_obj = CodeableConcept(
                                coding=coding_list,
                                text=diagnosis_data['code'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in diagnosis_data and isinstance(diagnosis_data['subject'], dict):
                            subject_obj = Reference(
                                reference=diagnosis_data['subject'].get('reference', ''),
                                display=diagnosis_data['subject'].get('display', '')
                            )

                        # Handle note
                        note_list = None
                        if 'note' in diagnosis_data and isinstance(diagnosis_data['note'], list):
                            note_list = []
                            for note in diagnosis_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create Diagnosis object
                        diagnosis = Diagnosis(
                            id=diagnosis_data.get('id', ''),
                            resourceType=diagnosis_data.get('resourceType', 'Condition'),
                            clinicalStatus=clinical_status_obj,
                            verificationStatus=verification_status_obj,
                            category=category_list,
                            code=code_obj,
                            subject=subject_obj,
                            onsetDateTime=diagnosis_data.get('onsetDateTime'),
                            abatementDateTime=diagnosis_data.get('abatementDateTime'),
                            recordedDate=diagnosis_data.get('recordedDate'),
                            note=note_list
                        )
                        result.append(diagnosis)
                    except Exception as e:
                        print(f"Error creating Diagnosis object: {str(e)}")
                        print(f"Diagnosis data: {diagnosis_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient diagnoses: {str(e)}")
            return []

    @strawberry.field
    async def patient_health_concerns(self, info, patient_id: str, clinical_status: Optional[str] = None, verification_status: Optional[str] = None) -> List[HealthConcern]:
        """Get health concerns for a patient"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return []

        # Get health concerns from Condition service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if clinical_status:
                    params["clinical_status"] = clinical_status
                if verification_status:
                    params["verification_status"] = verification_status

                url = f"{settings.CONDITION_SERVICE_URL}/api/conditions/patient/{patient_id}/health-concerns"
                print(f"Making request to: {url}")
                print(f"With params: {params}")
                print(f"With auth header: {auth_header}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")
                print(f"Response body: {response.text}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Condition service fails
                    fallback_params = {
                        "subject": f"Patient/{patient_id}",
                        "category": "health-concern"
                    }
                    if clinical_status:
                        fallback_params["clinical-status"] = clinical_status
                    if verification_status:
                        fallback_params["verification-status"] = verification_status

                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/fhir/Condition",
                        params=fallback_params,
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return []

                    concerns_data = fallback_response.json()
                else:
                    concerns_data = response.json()

                print(f"Received {len(concerns_data)} patient health concerns")

                # Filter health concerns by patient ID and category since the API returns all conditions
                filtered_concerns = []
                for concern in concerns_data:
                    if 'subject' in concern and isinstance(concern['subject'], dict):
                        subject_ref = concern['subject'].get('reference', '')
                        is_correct_patient = subject_ref == f"Patient/{patient_id}" or subject_ref.endswith(f"/{patient_id}")

                        # Check if it's a health concern
                        is_health_concern = False
                        if 'category' in concern and isinstance(concern['category'], list):
                            for cat in concern['category']:
                                if isinstance(cat, dict) and 'coding' in cat:
                                    for coding in cat['coding']:
                                        if coding.get('code') == 'health-concern':
                                            is_health_concern = True
                                            break
                                if is_health_concern:
                                    break

                        if is_correct_patient and is_health_concern:
                            filtered_concerns.append(concern)

                print(f"Filtered to {len(filtered_concerns)} health concerns for patient {patient_id}")
                concerns_data = filtered_concerns

                result = []

                for concern_data in concerns_data:
                    try:
                        # Convert the REST response to a GraphQL type
                        # Handle clinicalStatus
                        clinical_status_obj = None
                        if 'clinicalStatus' in concern_data and isinstance(concern_data['clinicalStatus'], dict):
                            coding_list = []
                            for coding in concern_data['clinicalStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            clinical_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=concern_data['clinicalStatus'].get('text', '')
                            )

                        # Handle verificationStatus
                        verification_status_obj = None
                        if 'verificationStatus' in concern_data and isinstance(concern_data['verificationStatus'], dict):
                            coding_list = []
                            for coding in concern_data['verificationStatus'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            verification_status_obj = CodeableConcept(
                                coding=coding_list,
                                text=concern_data['verificationStatus'].get('text', '')
                            )

                        # Handle category
                        category_list = []
                        if 'category' in concern_data and isinstance(concern_data['category'], list):
                            for cat in concern_data['category']:
                                if isinstance(cat, dict):
                                    coding_list = []
                                    for coding in cat.get('coding', []):
                                        coding_list.append(
                                            Coding(
                                                system=coding.get('system', ''),
                                                code=coding.get('code', ''),
                                                display=coding.get('display', '')
                                            )
                                        )
                                    category_list.append(
                                        CodeableConcept(
                                            coding=coding_list,
                                            text=cat.get('text', '')
                                        )
                                    )

                        # Handle code
                        code_obj = None
                        if 'code' in concern_data and isinstance(concern_data['code'], dict):
                            coding_list = []
                            for coding in concern_data['code'].get('coding', []):
                                coding_list.append(
                                    Coding(
                                        system=coding.get('system', ''),
                                        code=coding.get('code', ''),
                                        display=coding.get('display', '')
                                    )
                                )
                            code_obj = CodeableConcept(
                                coding=coding_list,
                                text=concern_data['code'].get('text', '')
                            )

                        # Handle subject
                        subject_obj = None
                        if 'subject' in concern_data and isinstance(concern_data['subject'], dict):
                            subject_obj = Reference(
                                reference=concern_data['subject'].get('reference', ''),
                                display=concern_data['subject'].get('display', '')
                            )

                        # Handle note
                        note_list = None
                        if 'note' in concern_data and isinstance(concern_data['note'], list):
                            note_list = []
                            for note in concern_data['note']:
                                note_list.append(
                                    Annotation(
                                        text=note.get('text', ''),
                                        authorString=note.get('authorString'),
                                        time=note.get('time')
                                    )
                                )

                        # Create HealthConcern object
                        concern = HealthConcern(
                            id=concern_data.get('id', ''),
                            resourceType=concern_data.get('resourceType', 'Condition'),
                            clinicalStatus=clinical_status_obj,
                            verificationStatus=verification_status_obj,
                            category=category_list,
                            code=code_obj,
                            subject=subject_obj,
                            onsetDateTime=concern_data.get('onsetDateTime'),
                            abatementDateTime=concern_data.get('abatementDateTime'),
                            recordedDate=concern_data.get('recordedDate'),
                            note=note_list
                        )
                        result.append(concern)
                    except Exception as e:
                        print(f"Error creating HealthConcern object: {str(e)}")
                        print(f"Health concern data: {concern_data}")
                        continue

                return result
        except Exception as e:
            print(f"Error fetching patient health concerns: {str(e)}")
            return []

    @strawberry.field
    async def patient_timeline(self, info, patient_id: str, start_date: Optional[str] = None, end_date: Optional[str] = None, event_types: Optional[List[str]] = None, resource_types: Optional[List[str]] = None) -> Optional[PatientTimeline]:
        """Get a patient's timeline with optional filtering"""
        # Get authorization header from request
        context = info.context
        auth_header = context.get("request").headers.get("Authorization")

        if not auth_header:
            return None

        # Get timeline from Timeline service
        try:
            async with httpx.AsyncClient(follow_redirects=True) as client:
                # Build query parameters
                params = {}
                if start_date:
                    params["start_date"] = start_date
                if end_date:
                    params["end_date"] = end_date
                if event_types:
                    params["event_types"] = event_types
                if resource_types:
                    params["resource_types"] = resource_types

                url = f"{settings.TIMELINE_SERVICE_URL}/api/timeline/patients/{patient_id}"
                print(f"Making request to: {url}")
                print(f"With params: {params}")

                response = await client.get(
                    url,
                    params=params,
                    headers={"Authorization": auth_header}
                )

                print(f"Response status code: {response.status_code}")

                if response.status_code != 200:
                    # Fallback to FHIR service if Timeline service fails
                    fallback_response = await client.get(
                        f"{settings.FHIR_SERVICE_URL}/api/fhir/Patient/{patient_id}/timeline",
                        headers={"Authorization": auth_header}
                    )

                    if fallback_response.status_code != 200:
                        print(f"Error response from fallback: {fallback_response.text}")
                        return None

                    timeline_data = fallback_response.json()
                else:
                    timeline_data = response.json()

                print(f"Received timeline data with {len(timeline_data.get('events', []))} events")

                # Create PatientTimeline object
                try:
                    # Process events
                    events = []
                    for event_data in timeline_data.get('events', []):
                        # Create EventDetails object if it exists
                        details_obj = None
                        if 'details' in event_data and event_data['details']:
                            details_obj = EventDetails(
                                code=event_data['details'].get('code'),
                                value=event_data['details'].get('value'),
                                unit=event_data['details'].get('unit'),
                                display=event_data['details'].get('display')
                            )

                        # Create TimelineEvent object
                        event = TimelineEvent(
                            id=event_data.get('id', ''),
                            patient_id=event_data.get('patient_id', ''),
                            event_type=event_data.get('event_type', ''),
                            resource_type=event_data.get('resource_type', ''),
                            resource_id=event_data.get('resource_id', ''),
                            title=event_data.get('title', ''),
                            description=event_data.get('description'),
                            date=event_data.get('date', ''),
                            details=details_obj
                        )
                        events.append(event)

                    # Create PatientTimeline object
                    return PatientTimeline(
                        patient_id=timeline_data.get('patient_id', ''),
                        events=events
                    )
                except Exception as e:
                    print(f"Error creating PatientTimeline object: {str(e)}")
                    print(f"Timeline data: {timeline_data}")
                    return None
        except Exception as e:
            print(f"Error fetching patient timeline: {str(e)}")
            return None
