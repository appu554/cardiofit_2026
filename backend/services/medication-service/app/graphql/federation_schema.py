"""
Apollo Federation schema for the Medication Service.

This module defines the GraphQL schema with Federation directives for the Medication Service,
allowing it to be part of a federated GraphQL gateway.
"""

import strawberry
from typing import List, Optional
import logging
from datetime import datetime

# Import the FHIR service
from app.services.fhir_service_factory import get_fhir_service

# Configure logging
logger = logging.getLogger(__name__)

@strawberry.type
class CodeableConcept:
    text: Optional[str] = strawberry.federation.field(shareable=True)
    coding: Optional[List["Coding"]] = strawberry.federation.field(shareable=True)

@strawberry.type
class Coding:
    system: Optional[str] = strawberry.federation.field(shareable=True)
    code: Optional[str] = strawberry.federation.field(shareable=True)
    display: Optional[str] = strawberry.federation.field(shareable=True)
    version: Optional[str] = strawberry.federation.field(shareable=True)
    user_selected: Optional[bool] = strawberry.federation.field(shareable=True)

@strawberry.type
class Reference:
    reference: Optional[str] = strawberry.federation.field(shareable=True)
    display: Optional[str] = strawberry.federation.field(shareable=True)
    type: Optional[str] = strawberry.federation.field(shareable=True)
    identifier: Optional["Identifier"] = strawberry.federation.field(shareable=True)

@strawberry.type
class Identifier:
    use: Optional[str] = strawberry.federation.field(shareable=True)
    type: Optional[CodeableConcept] = strawberry.federation.field(shareable=True)
    system: Optional[str] = strawberry.federation.field(shareable=True)
    value: Optional[str] = strawberry.federation.field(shareable=True)
    period: Optional["Period"] = strawberry.federation.field(shareable=True)
    assigner: Optional[Reference] = strawberry.federation.field(shareable=True)

@strawberry.type
class Period:
    start: Optional[str] = strawberry.federation.field(shareable=True)
    end: Optional[str] = strawberry.federation.field(shareable=True)

@strawberry.type
class Quantity:
    value: Optional[float] = strawberry.federation.field(shareable=True)
    unit: Optional[str] = strawberry.federation.field(shareable=True)
    system: Optional[str] = strawberry.federation.field(shareable=True)
    code: Optional[str] = strawberry.federation.field(shareable=True)

@strawberry.type
class Annotation:
    author_reference: Optional[Reference] = strawberry.federation.field(shareable=True)
    author_string: Optional[str] = strawberry.federation.field(shareable=True)
    time: Optional[str] = strawberry.federation.field(shareable=True)
    text: str = strawberry.federation.field(shareable=True)

@strawberry.type
class Ratio:
    numerator: Optional[Quantity] = strawberry.federation.field(shareable=True, default=None)
    denominator: Optional[Quantity] = strawberry.federation.field(shareable=True, default=None)

@strawberry.type
class DosageInstruction:
    text: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    timing: Optional[str] = strawberry.federation.field(shareable=True, default=None)
    route: Optional[CodeableConcept] = strawberry.federation.field(shareable=True, default=None)
    dose_quantity: Optional[Quantity] = strawberry.federation.field(shareable=True, default=None)

@strawberry.type
class MedicationAdministrationPerformer:
    actor: Optional[Reference] = None
    function: Optional[CodeableConcept] = None

@strawberry.type
class MedicationAdministrationDosage:
    text: Optional[str] = None
    dose: Optional[Quantity] = None
    route: Optional[CodeableConcept] = None

@strawberry.type
class Medication:
    id: str
    resource_type: str = "Medication"
    status: Optional[str] = None
    code: Optional[CodeableConcept] = None
    form: Optional[CodeableConcept] = None
    amount: Optional[Ratio] = None

@strawberry.type
class MedicationRequest:
    id: str
    resource_type: str = "MedicationRequest"
    status: str
    intent: str
    medication_codeable_concept: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    authored_on: Optional[str] = None
    requester: Optional[Reference] = None
    dosage_instruction: Optional[List[DosageInstruction]] = None

@strawberry.type
class MedicationStatement:
    id: str
    resource_type: str = "MedicationStatement"
    status: str
    medication_codeable_concept: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    effective_date_time: Optional[str] = None
    date_asserted: Optional[str] = None

@strawberry.type
class MedicationAdministration:
    id: str
    resource_type: str = "MedicationAdministration"
    status: str
    medication_codeable_concept: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    effective_date_time: Optional[str] = None
    performer: Optional[List[MedicationAdministrationPerformer]] = None
    dosage: Optional[MedicationAdministrationDosage] = None
    request: Optional[Reference] = None

@strawberry.type
class AllergyIntolerance:
    id: str
    resource_type: str = "AllergyIntolerance"
    clinical_status: Optional[CodeableConcept] = None
    verification_status: Optional[CodeableConcept] = None
    code: Optional[CodeableConcept] = None
    patient: Optional[Reference] = None
    criticality: Optional[str] = None

@strawberry.federation.type(keys=["id"])
class Patient:
    """
    Patient entity extended with medication-related fields.
    This extends the Patient type from the Patient service.
    """
    id: strawberry.ID = strawberry.federation.field(external=True)
    
    @strawberry.field
    async def medications(self, info) -> List[MedicationRequest]:
        """Get medication requests for this patient."""
        try:
            fhir_service = get_fhir_service()
            
            # Search for medication requests for this patient
            search_params = {"subject": f"Patient/{self.id}"}
            resources = await fhir_service.search_resources("MedicationRequest", search_params)
            
            # Convert to GraphQL types
            result = []
            for resource in resources:
                med_request = MedicationRequest(
                    id=resource.get("id", ""),
                    status=resource.get("status", ""),
                    intent=resource.get("intent", ""),
                    medication_codeable_concept=_convert_codeable_concept(resource.get("medicationCodeableConcept")),
                    subject=_convert_reference(resource.get("subject")),
                    authored_on=resource.get("authoredOn"),
                    requester=_convert_reference(resource.get("requester")),
                    dosage_instruction=_convert_dosage_instructions(resource.get("dosageInstruction", []))
                )
                result.append(med_request)
            
            return result
        except Exception as e:
            logger.error(f"Error fetching patient medications: {e}")
            return []
    
    @strawberry.field
    async def medication_statements(self, info) -> List[MedicationStatement]:
        """Get medication statements for this patient."""
        try:
            fhir_service = get_fhir_service()
            
            # Search for medication statements for this patient
            search_params = {"subject": f"Patient/{self.id}"}
            resources = await fhir_service.search_resources("MedicationStatement", search_params)
            
            # Convert to GraphQL types
            result = []
            for resource in resources:
                med_statement = MedicationStatement(
                    id=resource.get("id", ""),
                    status=resource.get("status", ""),
                    medication_codeable_concept=_convert_codeable_concept(resource.get("medicationCodeableConcept")),
                    subject=_convert_reference(resource.get("subject")),
                    effective_date_time=resource.get("effectiveDateTime"),
                    date_asserted=resource.get("dateAsserted")
                )
                result.append(med_statement)
            
            return result
        except Exception as e:
            logger.error(f"Error fetching patient medication statements: {e}")
            return []
    
    @strawberry.field
    async def medication_administrations(self, info) -> List[MedicationAdministration]:
        """Get medication administrations for this patient."""
        try:
            fhir_service = get_fhir_service()
            
            # Search for medication administrations for this patient
            search_params = {"subject": f"Patient/{self.id}"}
            resources = await fhir_service.search_resources("MedicationAdministration", search_params)
            
            # Convert to GraphQL types
            result = []
            for resource in resources:
                med_admin = MedicationAdministration(
                    id=resource.get("id", ""),
                    status=resource.get("status", ""),
                    medication_codeable_concept=_convert_codeable_concept(resource.get("medicationCodeableConcept")),
                    subject=_convert_reference(resource.get("subject")),
                    effective_date_time=resource.get("effectiveDateTime"),
                    performer=_convert_performers(resource.get("performer", [])),
                    dosage=_convert_administration_dosage(resource.get("dosage")),
                    request=_convert_reference(resource.get("request"))
                )
                result.append(med_admin)
            
            return result
        except Exception as e:
            logger.error(f"Error fetching patient medication administrations: {e}")
            return []
    
    @strawberry.field
    async def allergies(self, info) -> List[AllergyIntolerance]:
        """Get allergies for this patient."""
        try:
            fhir_service = get_fhir_service()
            
            # Search for allergy intolerances for this patient
            search_params = {"patient": f"Patient/{self.id}"}
            resources = await fhir_service.search_resources("AllergyIntolerance", search_params)
            
            # Convert to GraphQL types
            result = []
            for resource in resources:
                allergy = AllergyIntolerance(
                    id=resource.get("id", ""),
                    clinical_status=_convert_codeable_concept(resource.get("clinicalStatus")),
                    verification_status=_convert_codeable_concept(resource.get("verificationStatus")),
                    code=_convert_codeable_concept(resource.get("code")),
                    patient=_convert_reference(resource.get("patient")),
                    criticality=resource.get("criticality")
                )
                result.append(allergy)
            
            return result
        except Exception as e:
            logger.error(f"Error fetching patient allergies: {e}")
            return []

# Input types for mutations
@strawberry.input
class CodeableConceptInput:
    text: Optional[str] = None
    coding: Optional[List["CodingInput"]] = None

@strawberry.input
class CodingInput:
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None

@strawberry.input
class ReferenceInput:
    reference: Optional[str] = None
    display: Optional[str] = None

@strawberry.input
class QuantityInput:
    value: Optional[float] = None
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.input
class MedicationAdministrationPerformerInput:
    actor: Optional[ReferenceInput] = None
    function: Optional[CodeableConceptInput] = None

@strawberry.input
class MedicationAdministrationDosageInput:
    text: Optional[str] = None
    dose: Optional[QuantityInput] = None
    route: Optional[CodeableConceptInput] = None

@strawberry.input
class MedicationAdministrationInput:
    status: str
    medication_codeable_concept: Optional[CodeableConceptInput] = None
    subject: Optional[ReferenceInput] = None
    effective_date_time: Optional[str] = None
    performer: Optional[List[MedicationAdministrationPerformerInput]] = None
    dosage: Optional[MedicationAdministrationDosageInput] = None
    request: Optional[ReferenceInput] = None

@strawberry.type
class Query:
    """Root query type for the Medication Service."""
    
    @strawberry.field
    async def medications(self, info, page: Optional[int] = 1, limit: Optional[int] = 10) -> List[Medication]:
        """Get all medications."""
        try:
            fhir_service = get_fhir_service()
            
            # Search for medications
            search_params = {"_count": str(limit), "_offset": str((page - 1) * limit)}
            resources = await fhir_service.search_resources("Medication", search_params)
            
            # Convert to GraphQL types
            result = []
            for resource in resources:
                medication = Medication(
                    id=resource.get("id", ""),
                    status=resource.get("status"),
                    code=_convert_codeable_concept(resource.get("code")),
                    form=_convert_codeable_concept(resource.get("form")),
                    amount=_convert_ratio(resource.get("amount"))
                )
                result.append(medication)
            
            return result
        except Exception as e:
            logger.error(f"Error fetching medications: {e}")
            return []

@strawberry.type
class Mutation:
    """Root mutation type for the Medication Service."""
    
    @strawberry.field
    async def create_medication_request(self, info, patient_id: str, medication_code: str, status: str = "active", intent: str = "order") -> Optional[MedicationRequest]:
        """Create a new medication request."""
        try:
            fhir_service = get_fhir_service()
            
            # Create the medication request resource
            resource = {
                "resourceType": "MedicationRequest",
                "status": status,
                "intent": intent,
                "medicationCodeableConcept": {
                    "coding": [{"code": medication_code}]
                },
                "subject": {
                    "reference": f"Patient/{patient_id}"
                }
            }
            
            created_resource = await fhir_service.create_resource("MedicationRequest", resource)
            
            # Convert to GraphQL type
            return MedicationRequest(
                id=created_resource.get("id", ""),
                status=created_resource.get("status", ""),
                intent=created_resource.get("intent", ""),
                medication_codeable_concept=_convert_codeable_concept(created_resource.get("medicationCodeableConcept")),
                subject=_convert_reference(created_resource.get("subject"))
            )
        except Exception as e:
            logger.error(f"Error creating medication request: {e}")
            return None

    @strawberry.field
    async def create_allergy_intolerance(self, info, patient_id: str, allergen_code: str, allergen_display: str = None, criticality: str = "low") -> Optional[AllergyIntolerance]:
        """Create a new allergy intolerance."""
        try:
            fhir_service = get_fhir_service()

            # Create the allergy intolerance resource
            resource = {
                "resourceType": "AllergyIntolerance",
                "clinicalStatus": {
                    "coding": [{
                        "system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical",
                        "code": "active",
                        "display": "Active"
                    }]
                },
                "verificationStatus": {
                    "coding": [{
                        "system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-verification",
                        "code": "confirmed",
                        "display": "Confirmed"
                    }]
                },
                "code": {
                    "coding": [{
                        "system": "http://snomed.info/sct",
                        "code": allergen_code,
                        "display": allergen_display or allergen_code
                    }],
                    "text": allergen_display or allergen_code
                },
                "patient": {
                    "reference": f"Patient/{patient_id}"
                },
                "criticality": criticality
            }

            created_resource = await fhir_service.create_resource("AllergyIntolerance", resource)
            logger.info(f"Created AllergyIntolerance resource with ID {created_resource.get('id')}")

            # Convert to GraphQL type
            return AllergyIntolerance(
                id=created_resource.get("id", ""),
                clinical_status=_convert_codeable_concept(created_resource.get("clinicalStatus")),
                verification_status=_convert_codeable_concept(created_resource.get("verificationStatus")),
                code=_convert_codeable_concept(created_resource.get("code")),
                patient=_convert_reference(created_resource.get("patient")),
                criticality=created_resource.get("criticality")
            )
        except Exception as e:
            logger.error(f"Error creating allergy intolerance: {e}")
            return None

    @strawberry.field
    async def create_medication_statement(self, info, patient_id: str, medication_code: str, medication_display: str = None, status: str = "active") -> Optional[MedicationStatement]:
        """Create a new medication statement."""
        try:
            fhir_service = get_fhir_service()

            # Create the medication statement resource
            resource = {
                "resourceType": "MedicationStatement",
                "status": status,
                "medicationCodeableConcept": {
                    "coding": [{
                        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                        "code": medication_code,
                        "display": medication_display or medication_code
                    }],
                    "text": medication_display or medication_code
                },
                "subject": {
                    "reference": f"Patient/{patient_id}"
                },
                "effectiveDateTime": datetime.now().isoformat(),
                "dateAsserted": datetime.now().isoformat()
            }

            created_resource = await fhir_service.create_resource("MedicationStatement", resource)
            logger.info(f"Created MedicationStatement resource with ID {created_resource.get('id')}")

            # Convert to GraphQL type
            return MedicationStatement(
                id=created_resource.get("id", ""),
                status=created_resource.get("status", ""),
                medication_codeable_concept=_convert_codeable_concept(created_resource.get("medicationCodeableConcept")),
                subject=_convert_reference(created_resource.get("subject")),
                effective_date_time=created_resource.get("effectiveDateTime"),
                date_asserted=created_resource.get("dateAsserted")
            )
        except Exception as e:
            logger.error(f"Error creating medication statement: {e}")
            return None

    @strawberry.field
    async def create_medication_administration(self, info, administration_data: MedicationAdministrationInput) -> Optional[MedicationAdministration]:
        """Create a new medication administration."""
        try:
            fhir_service = get_fhir_service()

            # Build the medication administration resource
            resource = {
                "resourceType": "MedicationAdministration",
                "status": administration_data.status
            }

            # Add medication codeable concept
            if administration_data.medication_codeable_concept:
                resource["medicationCodeableConcept"] = _convert_input_codeable_concept(administration_data.medication_codeable_concept)

            # Add subject
            if administration_data.subject:
                resource["subject"] = _convert_input_reference(administration_data.subject)

            # Add effective date time
            if administration_data.effective_date_time:
                resource["effectiveDateTime"] = administration_data.effective_date_time

            # Add performer
            if administration_data.performer:
                resource["performer"] = []
                for performer_input in administration_data.performer:
                    performer = {}
                    if performer_input.actor:
                        performer["actor"] = _convert_input_reference(performer_input.actor)
                    if performer_input.function:
                        performer["function"] = _convert_input_codeable_concept(performer_input.function)
                    resource["performer"].append(performer)

            # Add dosage
            if administration_data.dosage:
                dosage = {}
                if administration_data.dosage.text:
                    dosage["text"] = administration_data.dosage.text
                if administration_data.dosage.dose:
                    dosage["dose"] = _convert_input_quantity(administration_data.dosage.dose)
                if administration_data.dosage.route:
                    dosage["route"] = _convert_input_codeable_concept(administration_data.dosage.route)
                resource["dosage"] = dosage

            # Add request
            if administration_data.request:
                resource["request"] = _convert_input_reference(administration_data.request)

            created_resource = await fhir_service.create_resource("MedicationAdministration", resource)
            logger.info(f"Created MedicationAdministration resource with ID {created_resource.get('id')}")

            # Convert to GraphQL type
            return MedicationAdministration(
                id=created_resource.get("id", ""),
                status=created_resource.get("status", ""),
                medication_codeable_concept=_convert_codeable_concept(created_resource.get("medicationCodeableConcept")),
                subject=_convert_reference(created_resource.get("subject")),
                effective_date_time=created_resource.get("effectiveDateTime"),
                performer=_convert_performers(created_resource.get("performer", [])),
                dosage=_convert_administration_dosage(created_resource.get("dosage")),
                request=_convert_reference(created_resource.get("request"))
            )
        except Exception as e:
            logger.error(f"Error creating medication administration: {e}")
            return None

# Helper functions for converting FHIR data to GraphQL types
def _convert_codeable_concept(cc_data) -> Optional[CodeableConcept]:
    """Convert FHIR CodeableConcept to GraphQL type."""
    if not cc_data:
        return None

    coding_list = []
    if "coding" in cc_data and isinstance(cc_data["coding"], list):
        for coding_data in cc_data["coding"]:
            coding = Coding(
                system=coding_data.get("system"),
                code=coding_data.get("code"),
                display=coding_data.get("display"),
                version=coding_data.get("version"),
                user_selected=coding_data.get("userSelected")
            )
            coding_list.append(coding)

    return CodeableConcept(
        text=cc_data.get("text"),
        coding=coding_list if coding_list else None
    )

def _convert_reference(ref_data) -> Optional[Reference]:
    """Convert FHIR Reference to GraphQL type."""
    if not ref_data:
        return None

    return Reference(
        reference=ref_data.get("reference"),
        display=ref_data.get("display"),
        type=ref_data.get("type"),
        identifier=None  # For now, set to None as it's optional
    )

def _convert_quantity(qty_data) -> Optional[Quantity]:
    """Convert FHIR Quantity to GraphQL type."""
    if not qty_data:
        return None
    
    return Quantity(
        value=qty_data.get("value"),
        unit=qty_data.get("unit"),
        system=qty_data.get("system"),
        code=qty_data.get("code")
    )

def _convert_ratio(ratio_data) -> Optional[Ratio]:
    """Convert FHIR Ratio to GraphQL type."""
    if not ratio_data:
        return None
    
    return Ratio(
        numerator=_convert_quantity(ratio_data.get("numerator")),
        denominator=_convert_quantity(ratio_data.get("denominator"))
    )

def _convert_dosage_instructions(dosage_list) -> List[DosageInstruction]:
    """Convert FHIR DosageInstruction list to GraphQL types."""
    result = []
    for dosage_data in dosage_list:
        dosage = DosageInstruction(
            text=dosage_data.get("text"),
            timing=str(dosage_data.get("timing", "")),
            route=_convert_codeable_concept(dosage_data.get("route")),
            dose_quantity=_convert_quantity(dosage_data.get("doseQuantity"))
        )
        result.append(dosage)
    return result

def _convert_performers(performer_list) -> List[MedicationAdministrationPerformer]:
    """Convert FHIR performer list to GraphQL types."""
    result = []
    for performer_data in performer_list:
        performer = MedicationAdministrationPerformer(
            actor=_convert_reference(performer_data.get("actor")),
            function=_convert_codeable_concept(performer_data.get("function"))
        )
        result.append(performer)
    return result

def _convert_administration_dosage(dosage_data) -> Optional[MedicationAdministrationDosage]:
    """Convert FHIR dosage to GraphQL type."""
    if not dosage_data:
        return None

    return MedicationAdministrationDosage(
        text=dosage_data.get("text"),
        dose=_convert_quantity(dosage_data.get("dose")),
        route=_convert_codeable_concept(dosage_data.get("route"))
    )

# Helper functions for converting input types to FHIR format
def _convert_input_codeable_concept(cc_input: CodeableConceptInput) -> dict:
    """Convert CodeableConceptInput to FHIR format."""
    result = {}
    if cc_input.text:
        result["text"] = cc_input.text
    if cc_input.coding:
        result["coding"] = []
        for coding_input in cc_input.coding:
            coding = {}
            if coding_input.system:
                coding["system"] = coding_input.system
            if coding_input.code:
                coding["code"] = coding_input.code
            if coding_input.display:
                coding["display"] = coding_input.display
            result["coding"].append(coding)
    return result

def _convert_input_reference(ref_input: ReferenceInput) -> dict:
    """Convert ReferenceInput to FHIR format."""
    result = {}
    if ref_input.reference:
        result["reference"] = ref_input.reference
    if ref_input.display:
        result["display"] = ref_input.display
    return result

def _convert_input_quantity(qty_input: QuantityInput) -> dict:
    """Convert QuantityInput to FHIR format."""
    result = {}
    if qty_input.value is not None:
        result["value"] = qty_input.value
    if qty_input.unit:
        result["unit"] = qty_input.unit
    if qty_input.system:
        result["system"] = qty_input.system
    if qty_input.code:
        result["code"] = qty_input.code
    return result

# Create the federated schema
schema = strawberry.federation.Schema(
    query=Query,
    mutation=Mutation,
    types=[Patient],
    enable_federation_2=True
)
