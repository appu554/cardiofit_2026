"""
Apollo Federation Schema for Encounter Management Service

This module defines the complete GraphQL schema with federation directives
for the Encounter Management Service with comprehensive FHIR-compliant types.
"""

import strawberry
from typing import List, Optional, Dict, Any
import logging
import os
import sys
from enum import Enum

# Remove the federation link for now - will add it differently

# Define a GenericScalar equivalent for compatibility with Graphene services
GenericScalar = strawberry.scalar(
    Any,
    name="GenericScalar",
    description="The GenericScalar scalar type represents a generic GraphQL scalar value that could be: String, Boolean, Int, Float, List or Object."
)

# Configure logging
logger = logging.getLogger(__name__)

# FHIR Enums
@strawberry.enum
class EncounterStatus(Enum):
    """FHIR Encounter status values"""
    PLANNED = "planned"
    ARRIVED = "arrived"
    TRIAGED = "triaged"
    IN_PROGRESS = "in-progress"
    ONLEAVE = "onleave"
    FINISHED = "finished"
    CANCELLED = "cancelled"
    ENTERED_IN_ERROR = "entered-in-error"
    UNKNOWN = "unknown"

@strawberry.enum
class EncounterClass(Enum):
    """FHIR Encounter class values"""
    INPATIENT = "IMP"
    OUTPATIENT = "AMB"
    AMBULATORY = "AMB"
    EMERGENCY = "EMER"
    HOME = "HH"
    FIELD = "FLD"
    DAYTIME = "SS"
    VIRTUAL = "VR"

@strawberry.enum
class ParticipantType(Enum):
    """FHIR Encounter participant type values"""
    TRANSLATOR = "translator"
    EMERGENCY = "emergency"
    ADMITTER = "admitter"
    DISCHARGER = "discharger"
    ATTENDER = "attender"
    REFERRER = "referrer"
    CONSULTANT = "consultant"
    PRIMARY_PERFORMER = "primary-performer"

@strawberry.enum
class LocationStatus(Enum):
    """FHIR Encounter location status values"""
    PLANNED = "planned"
    ACTIVE = "active"
    RESERVED = "reserved"
    COMPLETED = "completed"

@strawberry.enum
class DiagnosisUse(Enum):
    """FHIR Encounter diagnosis use values"""
    CHIEF_COMPLAINT = "CC"
    ADMITTING_DIAGNOSIS = "AD"
    DISCHARGE_DIAGNOSIS = "DD"
    WORKING_DIAGNOSIS = "WD"
    COMORBIDITY_DIAGNOSIS = "CM"
    PRE_OP_DIAGNOSIS = "pre-op"
    POST_OP_DIAGNOSIS = "post-op"
    BILLING = "billing"

# Shared FHIR Types (marked as shareable for federation)
@strawberry.type
class Coding:
    """FHIR Coding type"""
    system: Optional[str] = None
    version: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    user_selected: Optional[bool] = None

@strawberry.type
class CodeableConcept:
    """FHIR CodeableConcept type"""
    coding: Optional[List[Coding]] = None
    text: Optional[str] = None

@strawberry.type
class Reference:
    """FHIR Reference type"""
    reference: Optional[str] = None
    type: Optional[str] = None
    identifier: Optional["Identifier"] = None
    display: Optional[str] = None

@strawberry.type
class Identifier:
    """FHIR Identifier type"""
    use: Optional[str] = None
    type: Optional[CodeableConcept] = None
    system: Optional[str] = None
    value: Optional[str] = None
    period: Optional["Period"] = None
    assigner: Optional[Reference] = None

@strawberry.type
class Period:
    """FHIR Period type"""
    start: Optional[str] = None
    end: Optional[str] = None

# Duration type removed to avoid conflicts with orders service
# Will use GenericScalar for duration fields

@strawberry.type
class Quantity:
    """FHIR Quantity type - simplified to match other services"""
    value: Optional[float] = None
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None

# Encounter-specific types
@strawberry.type
class EncounterStatusHistory:
    """FHIR Encounter status history"""
    status: EncounterStatus
    period: Period

@strawberry.type
class EncounterParticipant:
    """FHIR Encounter participant"""
    type: Optional[List[CodeableConcept]] = None
    period: Optional[Period] = None
    individual: Optional[Reference] = None

@strawberry.type
class EncounterDiagnosis:
    """FHIR Encounter diagnosis"""
    condition: Reference
    use: Optional[CodeableConcept] = None
    rank: Optional[int] = None

@strawberry.type
class EncounterLocation:
    """FHIR Encounter location"""
    location: Reference
    status: Optional[LocationStatus] = None
    physical_type: Optional[CodeableConcept] = None
    period: Optional[Period] = None

@strawberry.type
class EncounterHospitalization:
    """FHIR Encounter hospitalization"""
    pre_admission_identifier: Optional[Identifier] = None
    origin: Optional[Reference] = None
    admit_source: Optional[CodeableConcept] = None
    re_admission: Optional[CodeableConcept] = None
    diet_preference: Optional[List[CodeableConcept]] = None
    special_courtesy: Optional[List[CodeableConcept]] = None
    special_arrangement: Optional[List[CodeableConcept]] = None
    destination: Optional[Reference] = None
    discharge_disposition: Optional[CodeableConcept] = None

# Main Encounter Type
@strawberry.federation.type(keys=["id"])
class Encounter:
    """FHIR Encounter resource for patient encounters and visits"""
    id: strawberry.ID
    resource_type: str = "Encounter"
    identifier: Optional[List[Identifier]] = None
    status: EncounterStatus
    status_history: Optional[List[EncounterStatusHistory]] = None
    encounter_class: EncounterClass
    class_history: Optional[List["EncounterClassHistory"]] = None
    type: Optional[List[CodeableConcept]] = None
    service_type: Optional[CodeableConcept] = None
    priority: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    episode_of_care: Optional[List[Reference]] = None
    based_on: Optional[List[Reference]] = None
    participant: Optional[List[EncounterParticipant]] = None
    appointment: Optional[List[Reference]] = None
    period: Optional[Period] = None
    length: Optional[GenericScalar] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    diagnosis: Optional[List[EncounterDiagnosis]] = None
    account: Optional[List[Reference]] = None
    hospitalization: Optional[EncounterHospitalization] = None
    location: Optional[List[EncounterLocation]] = None
    service_provider: Optional[Reference] = None
    part_of: Optional[Reference] = None

@strawberry.type
class EncounterClassHistory:
    """FHIR Encounter class history"""
    encounter_class: EncounterClass
    period: Period

# Location Type
@strawberry.federation.type(keys=["id"])
class Location:
    """FHIR Location resource for physical locations"""
    id: strawberry.ID
    resource_type: str = "Location"
    identifier: Optional[List[Identifier]] = None
    status: Optional[str] = None
    operational_status: Optional[CodeableConcept] = None
    name: Optional[str] = None
    alias: Optional[List[str]] = None
    description: Optional[str] = None
    mode: Optional[str] = None
    type: Optional[List[CodeableConcept]] = None
    telecom: Optional[List["ContactPoint"]] = None
    address: Optional["Address"] = None
    physical_type: Optional[CodeableConcept] = None
    position: Optional["LocationPosition"] = None
    managing_organization: Optional[Reference] = None
    part_of: Optional[Reference] = None
    hours_of_operation: Optional[List["LocationHoursOfOperation"]] = None
    availability_exceptions: Optional[str] = None
    endpoint: Optional[List[Reference]] = None

@strawberry.type
class ContactPoint:
    """FHIR ContactPoint type"""
    system: Optional[str] = None
    value: Optional[str] = None
    use: Optional[str] = None
    rank: Optional[int] = None
    period: Optional[GenericScalar] = None

@strawberry.type
class Address:
    """FHIR Address type"""
    use: Optional[str] = None
    type: Optional[str] = None
    text: Optional[str] = None
    line: Optional[List[str]] = None
    city: Optional[str] = None
    district: Optional[str] = None
    state: Optional[str] = None
    postal_code: Optional[str] = None
    country: Optional[str] = None
    period: Optional[GenericScalar] = None

@strawberry.type
class LocationPosition:
    """FHIR Location position"""
    longitude: float
    latitude: float
    altitude: Optional[float] = None

@strawberry.type
class LocationHoursOfOperation:
    """FHIR Location hours of operation"""
    days_of_week: Optional[List[str]] = None
    all_day: Optional[bool] = None
    opening_time: Optional[str] = None
    closing_time: Optional[str] = None

# Federation entity extensions
@strawberry.federation.type(keys=["id"], extend=True)
class Patient:
    """Extended Patient entity from Patient Service with encounter management."""
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def encounters(self) -> List[Encounter]:
        """Get all encounters for this patient"""
        try:
            fhir_service = await get_fhir_service()
            encounters = await fhir_service.get_encounters_by_patient(str(self.id))
            return [_convert_fhir_encounter_to_graphql(enc) for enc in encounters]
        except Exception as e:
            logger.error(f"Error retrieving encounters for patient {self.id}: {str(e)}")
            return []

@strawberry.federation.type(keys=["id"], extend=True)
class User:
    """Extended User entity from Organization Service with encounter management (represents practitioners)."""
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def encounters_as_participant(self) -> List[Encounter]:
        """Get all encounters where this user is a participant"""
        try:
            fhir_service = await get_fhir_service()
            search_params = {"participant": f"Practitioner/{self.id}"}
            bundle = await fhir_service.search_encounters(search_params)
            
            if bundle and "entry" in bundle:
                encounters = [entry["resource"] for entry in bundle["entry"]]
                return [_convert_fhir_encounter_to_graphql(enc) for enc in encounters]
            return []
        except Exception as e:
            logger.error(f"Error retrieving encounters for practitioner {self.id}: {str(e)}")
            return []

# Additional Encounter Types for Missing Responsibilities

@strawberry.type
class EncounterAccount:
    """FHIR Encounter account information for billing"""
    account: Reference
    status: Optional[str] = None
    type: Optional[CodeableConcept] = None
    period: Optional[Period] = None

@strawberry.type
class EncounterAuditEntry:
    """Audit trail entry for encounter changes"""
    id: strawberry.ID
    timestamp: str
    user: Reference
    action: str
    field_changed: str
    old_value: Optional[str] = None
    new_value: Optional[str] = None
    reason: Optional[str] = None

@strawberry.type
class BedAssignment:
    """Bed assignment information"""
    bed_id: str
    room_number: str
    ward: str
    status: str  # available, occupied, maintenance, reserved
    patient_reference: Optional[Reference] = None
    assignment_period: Optional[Period] = None

@strawberry.type
class ADTMessage:
    """HL7 ADT message processing result"""
    message_id: str
    message_type: str  # A01, A02, A03, etc.
    processed_at: str
    encounter_id: Optional[str] = None
    status: str  # processed, failed, pending
    error_message: Optional[str] = None

# State Machine Validation
@strawberry.enum
class EncounterStateTransition(Enum):
    """Valid encounter state transitions"""
    PLANNED_TO_ARRIVED = "planned_to_arrived"
    ARRIVED_TO_TRIAGED = "arrived_to_triaged"
    TRIAGED_TO_IN_PROGRESS = "triaged_to_in_progress"
    IN_PROGRESS_TO_ONLEAVE = "in_progress_to_onleave"
    ONLEAVE_TO_IN_PROGRESS = "onleave_to_in_progress"
    IN_PROGRESS_TO_FINISHED = "in_progress_to_finished"
    ANY_TO_CANCELLED = "any_to_cancelled"

# Input Types for Mutations
@strawberry.input
class ReferenceInput:
    """Input type for FHIR Reference"""
    reference: Optional[str] = None
    display: Optional[str] = None

@strawberry.input
class CodeableConceptInput:
    """Input type for FHIR CodeableConcept"""
    text: Optional[str] = None
    coding: Optional[List["CodingInput"]] = None

@strawberry.input
class CodingInput:
    """Input type for FHIR Coding"""
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    version: Optional[str] = None
    user_selected: Optional[bool] = None

@strawberry.input
class PeriodInput:
    """Input type for FHIR Period"""
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.input
class EncounterParticipantInput:
    """Input type for Encounter participant"""
    type: Optional[List[CodeableConceptInput]] = None
    period: Optional[PeriodInput] = None
    individual: Optional[ReferenceInput] = None

@strawberry.input
class EncounterLocationInput:
    """Input type for Encounter location"""
    location: ReferenceInput
    status: Optional[LocationStatus] = None
    period: Optional[PeriodInput] = None

@strawberry.input
class CreateEncounterInput:
    """Input type for creating an encounter"""
    status: EncounterStatus
    encounter_class: EncounterClass
    type: Optional[List[CodeableConceptInput]] = None
    priority: Optional[CodeableConceptInput] = None
    subject: ReferenceInput
    participant: Optional[List[EncounterParticipantInput]] = None
    period: Optional[PeriodInput] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    service_provider: Optional[ReferenceInput] = None

@strawberry.input
class UpdateEncounterStatusInput:
    """Input type for updating encounter status"""
    status: EncounterStatus
    reason: Optional[str] = None

@strawberry.input
class AdmitPatientInput:
    """Input type for patient admission"""
    patient_id: str
    admit_source: Optional[CodeableConceptInput] = None
    location: Optional[ReferenceInput] = None
    attending_physician: Optional[ReferenceInput] = None
    reason_code: Optional[List[CodeableConceptInput]] = None

@strawberry.input
class TransferPatientInput:
    """Input type for patient transfer"""
    new_location: ReferenceInput
    reason: Optional[str] = None
    effective_time: Optional[str] = None

@strawberry.input
class DischargePatientInput:
    """Input type for patient discharge"""
    discharge_disposition: CodeableConceptInput
    reason: Optional[str] = None
    discharge_time: Optional[str] = None

@strawberry.input
class ProcessADTMessageInput:
    """Input type for processing HL7 ADT messages"""
    message_content: str
    message_type: str  # A01, A02, A03, etc.
    sending_facility: Optional[str] = None
    receiving_facility: Optional[str] = None

@strawberry.input
class CheckInFromAppointmentInput:
    """Input type for checking in from appointment"""
    appointment_id: str
    arrival_time: Optional[str] = None
    location: Optional[ReferenceInput] = None

@strawberry.input
class BedAssignmentInput:
    """Input type for bed assignment"""
    bed_id: str
    room_number: str
    ward: str
    effective_time: Optional[str] = None

@strawberry.input
class EncounterSearchInput:
    """Enhanced search input for encounters"""
    patient_id: Optional[str] = None
    status: Optional[List[EncounterStatus]] = None
    encounter_class: Optional[EncounterClass] = None
    organization_id: Optional[str] = None
    location_id: Optional[str] = None
    practitioner_id: Optional[str] = None
    date_range_start: Optional[str] = None
    date_range_end: Optional[str] = None

# Global FHIR service instance
_fhir_service = None

async def get_fhir_service():
    """Get or create the FHIR service instance."""
    global _fhir_service
    if _fhir_service is None:
        from app.services.fhir_service_factory import initialize_fhir_service
        _fhir_service = await initialize_fhir_service()
    return _fhir_service

# Helper functions for data conversion
def _convert_fhir_encounter_to_graphql(fhir_encounter: Dict[str, Any]) -> Encounter:
    """Convert FHIR Encounter resource to GraphQL Encounter type."""
    try:
        # Extract basic fields
        encounter_id = fhir_encounter.get("id", "")
        status = EncounterStatus(fhir_encounter.get("status", "unknown"))
        encounter_class_value = EncounterClass(fhir_encounter.get("class", {}).get("code", "AMB"))

        # Convert period
        period = None
        if "period" in fhir_encounter:
            period_data = fhir_encounter["period"]
            period = Period(
                start=period_data.get("start"),
                end=period_data.get("end")
            )

        # Convert subject reference
        subject = None
        if "subject" in fhir_encounter:
            subject_data = fhir_encounter["subject"]
            subject = Reference(
                reference=subject_data.get("reference"),
                display=subject_data.get("display")
            )

        # Convert participants
        participants = []
        if "participant" in fhir_encounter:
            for p in fhir_encounter["participant"]:
                participant_period = None
                if "period" in p:
                    participant_period = Period(
                        start=p["period"].get("start"),
                        end=p["period"].get("end")
                    )

                individual = None
                if "individual" in p:
                    individual = Reference(
                        reference=p["individual"].get("reference"),
                        display=p["individual"].get("display")
                    )

                participants.append(EncounterParticipant(
                    period=participant_period,
                    individual=individual
                ))

        # Convert locations
        locations = []
        if "location" in fhir_encounter:
            for loc in fhir_encounter["location"]:
                location_period = None
                if "period" in loc:
                    location_period = Period(
                        start=loc["period"].get("start"),
                        end=loc["period"].get("end")
                    )

                location_ref = Reference(
                    reference=loc["location"].get("reference"),
                    display=loc["location"].get("display")
                )

                locations.append(EncounterLocation(
                    location=location_ref,
                    status=LocationStatus(loc.get("status", "active")) if loc.get("status") else None,
                    period=location_period
                ))

        return Encounter(
            id=strawberry.ID(encounter_id),
            status=status,
            encounter_class=encounter_class_value,
            subject=subject,
            participant=participants if participants else None,
            period=period,
            location=locations if locations else None
        )

    except Exception as e:
        logger.error(f"Error converting FHIR encounter to GraphQL: {e}")
        # Return a minimal encounter object
        return Encounter(
            id=strawberry.ID(fhir_encounter.get("id", "")),
            status=EncounterStatus.UNKNOWN,
            encounter_class=EncounterClass.AMBULATORY
        )

@strawberry.type
class Query:
    """Root query type for the Encounter Management Service."""

    @strawberry.field
    async def encounter(self, id: strawberry.ID) -> Optional[Encounter]:
        """Get a specific encounter by ID"""
        try:
            fhir_service = await get_fhir_service()
            fhir_resource = await fhir_service.get_encounter(str(id))

            if fhir_resource:
                return _convert_fhir_encounter_to_graphql(fhir_resource)
            return None

        except Exception as e:
            logger.error(f"Error retrieving encounter {id}: {str(e)}")
            return None

    @strawberry.field
    async def encounters(
        self,
        patient_id: Optional[strawberry.ID] = None,
        status: Optional[List[EncounterStatus]] = None,
        encounter_class: Optional[EncounterClass] = None,
        organization_id: Optional[strawberry.ID] = None
    ) -> List[Encounter]:
        """Find encounters with filtering options"""
        try:
            fhir_service = await get_fhir_service()

            # Build search parameters
            search_params = {}

            if patient_id:
                search_params["subject"] = f"Patient/{patient_id}"

            if status:
                search_params["status"] = ",".join([s.value for s in status])

            if encounter_class:
                search_params["class"] = encounter_class.value

            if organization_id:
                search_params["service-provider"] = f"Organization/{organization_id}"

            bundle = await fhir_service.search_encounters(search_params)

            if bundle and "entry" in bundle:
                encounters = [entry["resource"] for entry in bundle["entry"]]
                return [_convert_fhir_encounter_to_graphql(enc) for enc in encounters]

            return []

        except Exception as e:
            logger.error(f"Error searching encounters: {str(e)}")
            return []

    @strawberry.field
    async def active_inpatient_encounters(
        self,
        organization_id: Optional[strawberry.ID] = None
    ) -> List[Encounter]:
        """Find all currently admitted patients for a hospital"""
        try:
            fhir_service = await get_fhir_service()
            encounters = await fhir_service.get_active_inpatient_encounters(
                str(organization_id) if organization_id else None
            )

            return [_convert_fhir_encounter_to_graphql(enc) for enc in encounters]

        except Exception as e:
            logger.error(f"Error retrieving active inpatient encounters: {str(e)}")
            return []

    @strawberry.field
    async def location(self, id: strawberry.ID) -> Optional[Location]:
        """Get a specific location by ID"""
        try:
            fhir_service = await get_fhir_service()
            fhir_resource = await fhir_service.get_location(str(id))

            if fhir_resource:
                return _convert_fhir_location_to_graphql(fhir_resource)
            return None

        except Exception as e:
            logger.error(f"Error retrieving location {id}: {str(e)}")
            return None

    @strawberry.field
    async def encounter_audit_trail(self, encounter_id: strawberry.ID) -> List[EncounterAuditEntry]:
        """Get audit trail for an encounter"""
        try:
            # This would typically query an audit log table/collection
            # For now, return empty list as audit trail needs to be implemented
            logger.info(f"Audit trail requested for encounter {encounter_id}")
            return []
        except Exception as e:
            logger.error(f"Error retrieving audit trail for encounter {encounter_id}: {str(e)}")
            return []

    @strawberry.field
    async def available_beds(self, ward: Optional[str] = None) -> List[BedAssignment]:
        """Get available beds for assignment"""
        try:
            # This would typically query bed management system
            # For now, return mock data
            logger.info(f"Available beds requested for ward: {ward}")
            return []
        except Exception as e:
            logger.error(f"Error retrieving available beds: {str(e)}")
            return []

    @strawberry.field
    async def enhanced_encounter_search(self, search: EncounterSearchInput) -> List[Encounter]:
        """Enhanced encounter search with multiple parameters"""
        try:
            fhir_service = await get_fhir_service()

            # Build comprehensive search parameters
            search_params = {}

            if search.patient_id:
                search_params["subject"] = f"Patient/{search.patient_id}"

            if search.status:
                search_params["status"] = ",".join([s.value for s in search.status])

            if search.encounter_class:
                search_params["class"] = search.encounter_class.value

            if search.organization_id:
                search_params["service-provider"] = f"Organization/{search.organization_id}"

            if search.location_id:
                search_params["location"] = f"Location/{search.location_id}"

            if search.practitioner_id:
                search_params["participant"] = f"Practitioner/{search.practitioner_id}"

            if search.date_range_start:
                search_params["date"] = f"ge{search.date_range_start}"

            if search.date_range_end:
                if "date" in search_params:
                    search_params["date"] += f"&date=le{search.date_range_end}"
                else:
                    search_params["date"] = f"le{search.date_range_end}"

            bundle = await fhir_service.search_encounters(search_params)

            if bundle and "entry" in bundle:
                encounters = [entry["resource"] for entry in bundle["entry"]]
                return [_convert_fhir_encounter_to_graphql(enc) for enc in encounters]

            return []

        except Exception as e:
            logger.error(f"Error in enhanced encounter search: {str(e)}")
            return []

def _convert_fhir_location_to_graphql(fhir_location: Dict[str, Any]) -> Location:
    """Convert FHIR Location resource to GraphQL Location type."""
    try:
        location_id = fhir_location.get("id", "")
        name = fhir_location.get("name")
        status = fhir_location.get("status")
        description = fhir_location.get("description")

        return Location(
            id=strawberry.ID(location_id),
            name=name,
            status=status,
            description=description
        )

    except Exception as e:
        logger.error(f"Error converting FHIR location to GraphQL: {e}")
        return Location(
            id=strawberry.ID(fhir_location.get("id", "")),
            name=fhir_location.get("name", "Unknown Location")
        )

@strawberry.type
class Mutation:
    """Root mutation type for the Encounter Management Service."""

    @strawberry.field
    async def create_encounter(self, encounter: CreateEncounterInput) -> Optional[Encounter]:
        """Create a new encounter"""
        try:
            fhir_service = await get_fhir_service()

            # Convert GraphQL input to FHIR format
            fhir_data = {
                "resourceType": "Encounter",
                "status": encounter.status.value,
                "class": {
                    "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                    "code": encounter.encounter_class.value
                },
                "subject": {
                    "reference": encounter.subject.reference,
                    "display": encounter.subject.display
                }
            }

            # Add optional fields
            if encounter.type:
                fhir_data["type"] = [
                    {"text": t.text} for t in encounter.type if t.text
                ]

            if encounter.period:
                fhir_data["period"] = {}
                if encounter.period.start:
                    fhir_data["period"]["start"] = encounter.period.start
                if encounter.period.end:
                    fhir_data["period"]["end"] = encounter.period.end

            if encounter.participant:
                participants = []
                for p in encounter.participant:
                    participant_data = {}
                    if p.individual:
                        participant_data["individual"] = {
                            "reference": p.individual.reference,
                            "display": p.individual.display
                        }
                    if p.period:
                        participant_data["period"] = {}
                        if p.period.start:
                            participant_data["period"]["start"] = p.period.start
                        if p.period.end:
                            participant_data["period"]["end"] = p.period.end
                    participants.append(participant_data)
                fhir_data["participant"] = participants

            if encounter.service_provider:
                fhir_data["serviceProvider"] = {
                    "reference": encounter.service_provider.reference,
                    "display": encounter.service_provider.display
                }

            # Create the encounter
            created_resource = await fhir_service.create_encounter(fhir_data)

            if created_resource:
                return _convert_fhir_encounter_to_graphql(created_resource)
            return None

        except Exception as e:
            logger.error(f"Error creating encounter: {str(e)}")
            return None

    @strawberry.field
    async def update_encounter_status(
        self,
        encounter_id: strawberry.ID,
        input: UpdateEncounterStatusInput
    ) -> Optional[Encounter]:
        """Update the status of an encounter with state validation"""
        try:
            fhir_service = await get_fhir_service()

            # Get the existing encounter
            existing_encounter = await fhir_service.get_encounter(str(encounter_id))
            if not existing_encounter:
                logger.error(f"Encounter not found: {encounter_id}")
                return None

            # Validate state transition
            current_status = EncounterStatus(existing_encounter.get("status", "unknown"))
            new_status = input.status

            if not _validate_encounter_state_transition(current_status, new_status):
                logger.error(f"Invalid state transition from {current_status.value} to {new_status.value}")
                return None

            # Create audit entry (simplified - in production, this would be stored)
            audit_entry = {
                "timestamp": "2024-01-01T00:00:00Z",  # Would use actual timestamp
                "action": "status_update",
                "field_changed": "status",
                "old_value": current_status.value,
                "new_value": new_status.value,
                "reason": input.reason
            }
            logger.info(f"Audit: {audit_entry}")

            # Update the status
            existing_encounter["status"] = input.status.value

            # Add status history if it doesn't exist
            if "statusHistory" not in existing_encounter:
                existing_encounter["statusHistory"] = []

            # End the previous status period
            if existing_encounter["statusHistory"]:
                last_status = existing_encounter["statusHistory"][-1]
                if "period" in last_status and not last_status["period"].get("end"):
                    last_status["period"]["end"] = "2024-01-01T00:00:00Z"

            # Add current status to history
            status_period = {
                "start": "2024-01-01T00:00:00Z"  # Would use actual timestamp
            }
            # Don't include 'end' field for current status (Google Healthcare API doesn't accept null)

            existing_encounter["statusHistory"].append({
                "status": input.status.value,
                "period": status_period
            })

            # Update the encounter
            updated_resource = await fhir_service.update_encounter(str(encounter_id), existing_encounter)

            if updated_resource:
                return _convert_fhir_encounter_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error updating encounter status {encounter_id}: {str(e)}")
            return None

    @strawberry.field
    async def admit_patient(self, input: AdmitPatientInput) -> Optional[Encounter]:
        """Admit a patient (create inpatient encounter)"""
        try:
            fhir_service = await get_fhir_service()

            # Create admission encounter
            fhir_data = {
                "resourceType": "Encounter",
                "status": "in-progress",
                "class": {
                    "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                    "code": "IMP",
                    "display": "inpatient encounter"
                },
                "subject": {
                    "reference": f"Patient/{input.patient_id}"
                }
            }

            # Add optional fields
            if input.admit_source and input.admit_source.text:
                fhir_data["hospitalization"] = {
                    "admitSource": {
                        "text": input.admit_source.text
                    }
                }

            if input.location:
                fhir_data["location"] = [{
                    "location": {
                        "reference": input.location.reference,
                        "display": input.location.display
                    },
                    "status": "active"
                }]

            if input.attending_physician:
                fhir_data["participant"] = [{
                    "type": [{
                        "coding": [{
                            "system": "http://terminology.hl7.org/CodeSystem/v3-ParticipationType",
                            "code": "ATND",
                            "display": "attender"
                        }]
                    }],
                    "individual": {
                        "reference": input.attending_physician.reference,
                        "display": input.attending_physician.display
                    }
                }]

            if input.reason_code:
                fhir_data["reasonCode"] = [
                    {"text": reason.text} for reason in input.reason_code if reason.text
                ]

            # Create the encounter
            created_resource = await fhir_service.create_encounter(fhir_data)

            if created_resource:
                return _convert_fhir_encounter_to_graphql(created_resource)
            return None

        except Exception as e:
            logger.error(f"Error admitting patient: {str(e)}")
            return None

    @strawberry.field
    async def transfer_patient(
        self,
        encounter_id: strawberry.ID,
        input: TransferPatientInput
    ) -> Optional[Encounter]:
        """Transfer a patient to a new location"""
        try:
            fhir_service = await get_fhir_service()

            # Get the existing encounter
            existing_encounter = await fhir_service.get_encounter(str(encounter_id))
            if not existing_encounter:
                logger.error(f"Encounter not found: {encounter_id}")
                return None

            # Update location history
            if "location" not in existing_encounter:
                existing_encounter["location"] = []

            # End the current location period
            for location in existing_encounter["location"]:
                if location.get("status") == "active" and "period" in location:
                    location["period"]["end"] = input.effective_time or "2024-01-01T00:00:00Z"
                    location["status"] = "completed"

            # Add new location
            new_location = {
                "location": {
                    "reference": input.new_location.reference,
                    "display": input.new_location.display
                },
                "status": "active",
                "period": {
                    "start": input.effective_time or "2024-01-01T00:00:00Z"
                }
            }
            existing_encounter["location"].append(new_location)

            # Update the encounter
            updated_resource = await fhir_service.update_encounter(str(encounter_id), existing_encounter)

            if updated_resource:
                return _convert_fhir_encounter_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error transferring patient {encounter_id}: {str(e)}")
            return None

    @strawberry.field
    async def discharge_patient(
        self,
        encounter_id: strawberry.ID,
        input: DischargePatientInput
    ) -> Optional[Encounter]:
        """Discharge a patient (end encounter)"""
        try:
            fhir_service = await get_fhir_service()

            # Get the existing encounter
            existing_encounter = await fhir_service.get_encounter(str(encounter_id))
            if not existing_encounter:
                logger.error(f"Encounter not found: {encounter_id}")
                return None

            # Update status to finished
            existing_encounter["status"] = "finished"

            # Set end time for period
            if "period" not in existing_encounter:
                existing_encounter["period"] = {}
            existing_encounter["period"]["end"] = input.discharge_time or "2024-01-01T00:00:00Z"

            # Add discharge disposition
            if "hospitalization" not in existing_encounter:
                existing_encounter["hospitalization"] = {}

            existing_encounter["hospitalization"]["dischargeDisposition"] = {
                "text": input.discharge_disposition.text
            }

            # End all active locations
            if "location" in existing_encounter:
                for location in existing_encounter["location"]:
                    if location.get("status") == "active" and "period" in location:
                        location["period"]["end"] = input.discharge_time or "2024-01-01T00:00:00Z"
                        location["status"] = "completed"

            # Update the encounter
            updated_resource = await fhir_service.update_encounter(str(encounter_id), existing_encounter)

            if updated_resource:
                return _convert_fhir_encounter_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error discharging patient {encounter_id}: {str(e)}")
            return None

    @strawberry.field
    async def add_participant_to_encounter(
        self,
        encounter_id: strawberry.ID,
        practitioner_id: strawberry.ID,
        role: ParticipantType
    ) -> Optional[Encounter]:
        """Add a healthcare practitioner to an encounter"""
        try:
            fhir_service = await get_fhir_service()

            # Get the existing encounter
            existing_encounter = await fhir_service.get_encounter(str(encounter_id))
            if not existing_encounter:
                logger.error(f"Encounter not found: {encounter_id}")
                return None

            # Add participant
            if "participant" not in existing_encounter:
                existing_encounter["participant"] = []

            new_participant = {
                "type": [{
                    "coding": [{
                        "system": "http://terminology.hl7.org/CodeSystem/v3-ParticipationType",
                        "code": role.value,
                        "display": role.value.replace("-", " ")
                    }]
                }],
                "individual": {
                    "reference": f"Practitioner/{practitioner_id}"
                }
            }

            existing_encounter["participant"].append(new_participant)

            # Update the encounter
            updated_resource = await fhir_service.update_encounter(str(encounter_id), existing_encounter)

            if updated_resource:
                return _convert_fhir_encounter_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error adding participant to encounter {encounter_id}: {str(e)}")
            return None

    @strawberry.field
    async def process_adt_message(self, input: ProcessADTMessageInput) -> ADTMessage:
        """Process HL7 ADT message and create/update encounters"""
        try:
            import uuid
            from datetime import datetime

            message_id = str(uuid.uuid4())
            processed_at = datetime.utcnow().isoformat()

            # Parse HL7 message (simplified implementation)
            # In production, use proper HL7 parsing library
            lines = input.message_content.split('\r')
            msh_segment = None
            pid_segment = None
            pv1_segment = None

            for line in lines:
                if line.startswith('MSH'):
                    msh_segment = line.split('|')
                elif line.startswith('PID'):
                    pid_segment = line.split('|')
                elif line.startswith('PV1'):
                    pv1_segment = line.split('|')

            if not all([msh_segment, pid_segment, pv1_segment]):
                return ADTMessage(
                    message_id=message_id,
                    message_type=input.message_type,
                    processed_at=processed_at,
                    status="failed",
                    error_message="Invalid HL7 message format"
                )

            # Extract patient ID and encounter info
            patient_id = pid_segment[3] if len(pid_segment) > 3 else None
            encounter_class = "IMP" if input.message_type in ["A01", "A02"] else "AMB"

            fhir_service = await get_fhir_service()
            encounter_id = None

            # Process based on message type
            if input.message_type == "A01":  # Admit
                # Create new inpatient encounter
                fhir_data = {
                    "resourceType": "Encounter",
                    "status": "in-progress",
                    "class": {
                        "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                        "code": encounter_class
                    },
                    "subject": {
                        "reference": f"Patient/{patient_id}"
                    }
                }

                created_resource = await fhir_service.create_encounter(fhir_data)
                if created_resource:
                    encounter_id = created_resource.get("id")

            elif input.message_type == "A02":  # Transfer
                # Update existing encounter with new location
                # This would require finding the active encounter for the patient
                logger.info(f"Processing transfer for patient {patient_id}")

            elif input.message_type == "A03":  # Discharge
                # Update encounter status to finished
                # This would require finding the active encounter for the patient
                logger.info(f"Processing discharge for patient {patient_id}")

            return ADTMessage(
                message_id=message_id,
                message_type=input.message_type,
                processed_at=processed_at,
                encounter_id=encounter_id,
                status="processed"
            )

        except Exception as e:
            logger.error(f"Error processing ADT message: {str(e)}")
            return ADTMessage(
                message_id=str(uuid.uuid4()),
                message_type=input.message_type,
                processed_at=datetime.utcnow().isoformat(),
                status="failed",
                error_message=str(e)
            )

    @strawberry.field
    async def check_in_from_appointment(self, input: CheckInFromAppointmentInput) -> Optional[Encounter]:
        """Check in patient from appointment and create/activate encounter"""
        try:
            fhir_service = await get_fhir_service()

            # In a real implementation, this would:
            # 1. Query SchedulingService to get appointment details
            # 2. Validate appointment exists and is scheduled
            # 3. Create or activate corresponding encounter
            # 4. Update appointment status to "fulfilled"

            # For now, create a basic encounter
            fhir_data = {
                "resourceType": "Encounter",
                "status": "arrived",
                "class": {
                    "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                    "code": "AMB"
                },
                "appointment": [{
                    "reference": f"Appointment/{input.appointment_id}"
                }]
            }

            if input.location:
                fhir_data["location"] = [{
                    "location": {
                        "reference": input.location.reference,
                        "display": input.location.display
                    },
                    "status": "active"
                }]

            created_resource = await fhir_service.create_encounter(fhir_data)

            if created_resource:
                return _convert_fhir_encounter_to_graphql(created_resource)
            return None

        except Exception as e:
            logger.error(f"Error checking in from appointment {input.appointment_id}: {str(e)}")
            return None

    @strawberry.field
    async def assign_bed(
        self,
        encounter_id: strawberry.ID,
        bed_assignment: BedAssignmentInput
    ) -> Optional[Encounter]:
        """Assign a bed to an inpatient encounter"""
        try:
            fhir_service = await get_fhir_service()

            # Get existing encounter
            existing_encounter = await fhir_service.get_encounter(str(encounter_id))
            if not existing_encounter:
                logger.error(f"Encounter not found: {encounter_id}")
                return None

            # Check if bed is available (simplified check)
            # In production, this would check a bed management system

            # Update encounter with bed assignment
            bed_location_ref = f"Location/{bed_assignment.bed_id}"

            if "location" not in existing_encounter:
                existing_encounter["location"] = []

            # End any current active locations
            for location in existing_encounter["location"]:
                if location.get("status") == "active":
                    location["status"] = "completed"
                    if "period" in location:
                        location["period"]["end"] = bed_assignment.effective_time or "2024-01-01T00:00:00Z"

            # Add new bed assignment
            new_location = {
                "location": {
                    "reference": bed_location_ref,
                    "display": f"Room {bed_assignment.room_number}, {bed_assignment.ward}"
                },
                "status": "active",
                "period": {
                    "start": bed_assignment.effective_time or "2024-01-01T00:00:00Z"
                }
            }
            existing_encounter["location"].append(new_location)

            # Update the encounter
            updated_resource = await fhir_service.update_encounter(str(encounter_id), existing_encounter)

            if updated_resource:
                return _convert_fhir_encounter_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error assigning bed to encounter {encounter_id}: {str(e)}")
            return None

def _validate_encounter_state_transition(current_status: EncounterStatus, new_status: EncounterStatus) -> bool:
    """Validate if encounter state transition is allowed"""
    valid_transitions = {
        EncounterStatus.PLANNED: [EncounterStatus.ARRIVED, EncounterStatus.IN_PROGRESS, EncounterStatus.CANCELLED],
        EncounterStatus.ARRIVED: [EncounterStatus.TRIAGED, EncounterStatus.IN_PROGRESS, EncounterStatus.CANCELLED],
        EncounterStatus.TRIAGED: [EncounterStatus.IN_PROGRESS, EncounterStatus.CANCELLED],
        EncounterStatus.IN_PROGRESS: [EncounterStatus.ONLEAVE, EncounterStatus.FINISHED, EncounterStatus.CANCELLED],
        EncounterStatus.ONLEAVE: [EncounterStatus.IN_PROGRESS, EncounterStatus.FINISHED, EncounterStatus.CANCELLED],
        EncounterStatus.FINISHED: [],  # Terminal state
        EncounterStatus.CANCELLED: [],  # Terminal state
    }

    return new_status in valid_transitions.get(current_status, [])

# Create the comprehensive federated schema
schema = strawberry.federation.Schema(
    query=Query,
    mutation=Mutation,
    types=[
        Patient, User,  # Federation extensions
        Encounter, Location,  # Core encounter types
        EncounterParticipant, EncounterDiagnosis, EncounterLocation, EncounterHospitalization,  # Complex types
        EncounterStatusHistory, EncounterClassHistory,  # History types
        EncounterAccount, EncounterAuditEntry, BedAssignment, ADTMessage,  # New responsibility types
        ContactPoint, Address, LocationPosition, LocationHoursOfOperation,  # Location types
        CodeableConcept, Coding, Reference, Identifier, Period, Quantity,  # Shared FHIR types
        EncounterStatus, EncounterClass, ParticipantType, LocationStatus, DiagnosisUse, EncounterStateTransition,  # Enums
    ]
)

logger.info("GraphQL federation schema initialized for Encounter Management Service")
logger.info("Schema includes: Encounters, Locations with Patient and User federation")

# Export schema for use in the application
__all__ = ["schema"]
