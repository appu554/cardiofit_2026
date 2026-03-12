"""
Test script for shared FHIR models.

This script demonstrates how to use the shared FHIR models
and validates that they work correctly.
"""

import sys
import os
import json
from datetime import datetime, timezone

# Add the backend directory to the Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import shared models
from shared.models import (
    Patient, Observation, Condition, Encounter,
    CodeableConcept, Coding, Reference, Quantity,
    validate_fhir_resource
)

def test_patient_model():
    """Test the Patient model."""
    print("\n=== Testing Patient Model ===")

    # Create a patient
    patient = Patient(
        id="patient-1",
        active=True,
        gender="male",
        birthDate="1970-01-01",
        name=[
            {
                "family": "Smith",
                "given": ["John", "Adam"],
                "use": "official"
            }
        ],
        telecom=[
            {
                "system": "phone",
                "value": "555-123-4567",
                "use": "home"
            },
            {
                "system": "email",
                "value": "john.smith@example.com"
            }
        ],
        address=[
            {
                "line": ["123 Main St"],
                "city": "Anytown",
                "state": "CA",
                "postalCode": "12345",
                "country": "USA",
                "use": "home"
            }
        ]
    )

    # Print the patient as JSON
    print("Patient as JSON:")
    print(json.dumps(patient.model_dump(exclude_none=True), indent=2))

    # Convert to FHIR model and back
    fhir_patient = patient.to_fhir_model()
    print("\nConverted to FHIR model and back:")
    patient2 = Patient.from_fhir_model(fhir_patient)
    print(json.dumps(patient2.model_dump(exclude_none=True), indent=2))

    # Validate the patient
    try:
        validate_fhir_resource(patient.model_dump(exclude_none=True), "Patient")
        print("\nPatient validation successful")
    except Exception as e:
        print(f"\nPatient validation failed: {e}")

    return patient

def test_observation_model(patient):
    """Test the Observation model."""
    print("\n=== Testing Observation Model ===")

    # Create an observation
    observation = Observation(
        id="observation-1",
        status="final",
        code=CodeableConcept(
            coding=[
                Coding(
                    system="http://loinc.org",
                    code="8867-4",
                    display="Heart rate"
                )
            ],
            text="Heart rate"
        ),
        subject=Reference(
            reference=f"Patient/{patient.id}",
            display=f"{patient.name[0].family}, {patient.name[0].given[0]}"
        ),
        effectiveDateTime=datetime.now(timezone.utc).isoformat(),
        valueQuantity=Quantity(
            value=80,
            unit="beats/minute",
            system="http://unitsofmeasure.org",
            code="/min"
        )
    )

    # Print the observation as JSON
    print("Observation as JSON:")
    print(json.dumps(observation.model_dump(exclude_none=True), indent=2))

    # Convert to FHIR model and back
    fhir_observation = observation.to_fhir_model()
    print("\nConverted to FHIR model and back:")
    observation2 = Observation.from_fhir_model(fhir_observation)
    print(json.dumps(observation2.model_dump(exclude_none=True), indent=2))

    # Validate the observation
    try:
        validate_fhir_resource(observation.model_dump(exclude_none=True), "Observation")
        print("\nObservation validation successful")
    except Exception as e:
        print(f"\nObservation validation failed: {e}")

    return observation

def test_condition_model(patient):
    """Test the Condition model."""
    print("\n=== Testing Condition Model ===")

    # Create a condition
    condition = Condition(
        id="condition-1",
        clinicalStatus=CodeableConcept(
            coding=[
                Coding(
                    system="http://terminology.hl7.org/CodeSystem/condition-clinical",
                    code="active",
                    display="Active"
                )
            ]
        ),
        verificationStatus=CodeableConcept(
            coding=[
                Coding(
                    system="http://terminology.hl7.org/CodeSystem/condition-ver-status",
                    code="confirmed",
                    display="Confirmed"
                )
            ]
        ),
        category=[
            CodeableConcept(
                coding=[
                    Coding(
                        system="http://terminology.hl7.org/CodeSystem/condition-category",
                        code="problem-list-item",
                        display="Problem List Item"
                    )
                ]
            )
        ],
        severity=CodeableConcept(
            coding=[
                Coding(
                    system="http://snomed.info/sct",
                    code="24484000",
                    display="Severe"
                )
            ]
        ),
        code=CodeableConcept(
            coding=[
                Coding(
                    system="http://snomed.info/sct",
                    code="44054006",
                    display="Diabetes mellitus type 2"
                )
            ],
            text="Type 2 diabetes mellitus"
        ),
        subject=Reference(
            reference=f"Patient/{patient.id}",
            display=f"{patient.name[0].family}, {patient.name[0].given[0]}"
        ),
        onsetDateTime="2020-01-01",
        recordedDate=datetime.now(timezone.utc).isoformat()
    )

    # Print the condition as JSON
    print("Condition as JSON:")
    print(json.dumps(condition.model_dump(exclude_none=True), indent=2))

    # Convert to FHIR model and back
    fhir_condition = condition.to_fhir_model()
    print("\nConverted to FHIR model and back:")
    condition2 = Condition.from_fhir_model(fhir_condition)
    print(json.dumps(condition2.model_dump(exclude_none=True), indent=2))

    # Validate the condition
    try:
        validate_fhir_resource(condition.model_dump(exclude_none=True), "Condition")
        print("\nCondition validation successful")
    except Exception as e:
        print(f"\nCondition validation failed: {e}")

    return condition

def main():
    """Run all tests."""
    print("Testing shared FHIR models...")

    # Test Patient model
    patient = test_patient_model()

    # Test Observation model
    observation = test_observation_model(patient)

    # Test Condition model
    condition = test_condition_model(patient)

    print("\nAll tests completed!")

if __name__ == "__main__":
    main()
