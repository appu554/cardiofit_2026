from typing import List, Optional, Union, Dict, Any
from pydantic import BaseModel
from datetime import datetime

# Import shared models
from shared.models import Observation as SharedObservation
from shared.models import Reference, CodeableConcept, Quantity

# Re-export the shared Observation model
Observation = SharedObservation

class LabTest(BaseModel):
    """Model for a lab test"""
    id: Optional[str] = None
    test_code: str
    test_name: str
    value: Union[float, str, bool]
    unit: Optional[str] = None
    reference_range: Optional[str] = None
    interpretation: Optional[str] = None
    status: str = "final"
    category: str = "laboratory"
    effective_date_time: datetime
    issued: Optional[datetime] = None
    performer: Optional[str] = None
    patient_id: str
    order_number: Optional[str] = None
    specimen_type: Optional[str] = None

    model_config = {
        "json_schema_extra": {
            "example": {
                "test_code": "WBC",
                "test_name": "WHITE BLOOD CELL COUNT",
                "value": 8.5,
                "unit": "10*3/uL",
                "reference_range": "4.0-11.0",
                "interpretation": "normal",
                "status": "final",
                "category": "laboratory",
                "effective_date_time": "2023-06-15T08:00:00",
                "patient_id": "123e4567-e89b-12d3-a456-426614174001",
                "order_number": "LAB12345",
                "specimen_type": "BLOOD"
            }
        }
    }

class LabPanel(BaseModel):
    """Model for a lab panel (group of related tests)"""
    id: Optional[str] = None
    panel_code: str
    panel_name: str
    tests: List[LabTest]
    effective_date_time: datetime
    issued: Optional[datetime] = None
    performer: Optional[str] = None
    patient_id: str
    order_number: Optional[str] = None
    specimen_type: Optional[str] = None

    model_config = {
        "json_schema_extra": {
            "example": {
                "panel_code": "CBC",
                "panel_name": "COMPLETE BLOOD COUNT",
                "tests": [
                    {
                        "test_code": "WBC",
                        "test_name": "WHITE BLOOD CELL COUNT",
                        "value": 8.5,
                        "unit": "10*3/uL",
                        "reference_range": "4.0-11.0",
                        "interpretation": "normal",
                        "status": "final",
                        "category": "laboratory",
                        "effective_date_time": "2023-06-15T08:00:00",
                        "patient_id": "123e4567-e89b-12d3-a456-426614174001"
                    }
                ],
                "effective_date_time": "2023-06-15T08:00:00",
                "patient_id": "123e4567-e89b-12d3-a456-426614174001",
                "order_number": "LAB12345",
                "specimen_type": "BLOOD"
            }
        }
    }

class LabTestCreate(BaseModel):
    """Model for creating a lab test"""
    test_code: str
    test_name: str
    value: Union[float, str, bool]
    unit: Optional[str] = None
    reference_range: Optional[str] = None
    interpretation: Optional[str] = None
    status: str = "final"
    category: str = "laboratory"
    effective_date_time: datetime
    issued: Optional[datetime] = None
    performer: Optional[str] = None
    patient_id: str
    order_number: Optional[str] = None
    specimen_type: Optional[str] = None

class LabPanelCreate(BaseModel):
    """Model for creating a lab panel"""
    panel_code: str
    panel_name: str
    tests: List[LabTestCreate]
    effective_date_time: datetime
    issued: Optional[datetime] = None
    performer: Optional[str] = None
    patient_id: str
    order_number: Optional[str] = None
    specimen_type: Optional[str] = None

def lab_test_to_observation(lab_test: LabTest) -> Observation:
    """
    Convert a LabTest to an Observation.

    Args:
        lab_test: The lab test to convert

    Returns:
        An Observation representing the lab test
    """
    # Create the observation
    observation = Observation(
        id=lab_test.id,
        resourceType="Observation",
        status=lab_test.status,
        category=[
            {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                        "code": lab_test.category,
                        "display": lab_test.category.capitalize()
                    }
                ],
                "text": lab_test.category.capitalize()
            }
        ],
        code=CodeableConcept(
            coding=[
                {
                    "system": "http://loinc.org",
                    "code": lab_test.test_code,
                    "display": lab_test.test_name
                }
            ],
            text=lab_test.test_name
        ),
        subject=Reference(
            reference=f"Patient/{lab_test.patient_id}"
        ),
        effectiveDateTime=lab_test.effective_date_time.isoformat(),
        issued=lab_test.issued.isoformat() if lab_test.issued else None
    )

    # Add value based on type
    if isinstance(lab_test.value, float) or isinstance(lab_test.value, int):
        observation.valueQuantity = Quantity(
            value=float(lab_test.value),
            unit=lab_test.unit,
            system="http://unitsofmeasure.org"
        )
    elif isinstance(lab_test.value, str):
        observation.valueString = lab_test.value
    elif isinstance(lab_test.value, bool):
        observation.valueBoolean = lab_test.value

    # Add reference range if available
    if lab_test.reference_range:
        observation.referenceRange = [
            {
                "text": lab_test.reference_range
            }
        ]

    # Add interpretation if available
    if lab_test.interpretation:
        observation.interpretation = [
            {
                "coding": [
                    {
                        "system": "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
                        "code": lab_test.interpretation,
                        "display": lab_test.interpretation.capitalize()
                    }
                ],
                "text": lab_test.interpretation.capitalize()
            }
        ]

    # Add performer if available
    if lab_test.performer:
        observation.performer = [
            Reference(
                reference=f"Practitioner/{lab_test.performer}"
            )
        ]

    # Add specimen if available
    if lab_test.specimen_type:
        observation.specimen = Reference(
            display=lab_test.specimen_type
        )

    return observation

def observation_to_lab_test(observation: Observation) -> LabTest:
    """
    Convert an Observation to a LabTest.

    Args:
        observation: The observation to convert

    Returns:
        A LabTest representing the observation
    """
    # Extract patient ID from subject reference
    patient_id = None
    if observation.subject and observation.subject.reference:
        parts = observation.subject.reference.split("/")
        if len(parts) == 2 and parts[0] == "Patient":
            patient_id = parts[1]

    # Extract test code and name from code
    test_code = None
    test_name = None
    if observation.code:
        if observation.code.coding and len(observation.code.coding) > 0:
            test_code = observation.code.coding[0].get("code")
            test_name = observation.code.coding[0].get("display")
        if not test_name and observation.code.text:
            test_name = observation.code.text

    # Extract category
    category = "laboratory"
    if observation.category and len(observation.category) > 0:
        if observation.category[0].get("coding") and len(observation.category[0].get("coding", [])) > 0:
            category = observation.category[0].get("coding")[0].get("code", "laboratory")

    # Extract value based on type
    value = None
    unit = None
    if observation.valueQuantity:
        value = observation.valueQuantity.value
        unit = observation.valueQuantity.unit
    elif observation.valueString:
        value = observation.valueString
    elif observation.valueBoolean is not None:
        value = observation.valueBoolean

    # Extract reference range
    reference_range = None
    if observation.referenceRange and len(observation.referenceRange) > 0:
        reference_range = observation.referenceRange[0].get("text")

    # Extract interpretation
    interpretation = None
    if observation.interpretation and len(observation.interpretation) > 0:
        if observation.interpretation[0].get("coding") and len(observation.interpretation[0].get("coding", [])) > 0:
            interpretation = observation.interpretation[0].get("coding")[0].get("code")

    # Extract performer
    performer = None
    if observation.performer and len(observation.performer) > 0:
        if observation.performer[0].reference:
            parts = observation.performer[0].reference.split("/")
            if len(parts) == 2 and parts[0] == "Practitioner":
                performer = parts[1]

    # Extract specimen
    specimen_type = None
    if observation.specimen and observation.specimen.display:
        specimen_type = observation.specimen.display

    # Parse dates
    effective_date_time = None
    if observation.effectiveDateTime:
        effective_date_time = datetime.fromisoformat(observation.effectiveDateTime.replace("Z", "+00:00"))

    issued = None
    if observation.issued:
        issued = datetime.fromisoformat(observation.issued.replace("Z", "+00:00"))

    return LabTest(
        id=observation.id,
        test_code=test_code,
        test_name=test_name,
        value=value,
        unit=unit,
        reference_range=reference_range,
        interpretation=interpretation,
        status=observation.status,
        category=category,
        effective_date_time=effective_date_time,
        issued=issued,
        performer=performer,
        patient_id=patient_id,
        specimen_type=specimen_type
    )
