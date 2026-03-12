"""
Apollo Federation Schema for Scheduling Service

This module defines the complete GraphQL schema with federation directives
for the Scheduling Service with comprehensive FHIR-compliant types.
"""

import strawberry
from typing import List, Optional, Dict, Any
import logging
import os
import sys
from enum import Enum

# Define a GenericScalar equivalent for compatibility with Graphene services
GenericScalar = strawberry.scalar(
    Any,
    name="GenericScalar",
    description="The GenericScalar scalar type represents a generic GraphQL scalar value that could be: String, Boolean, Int, Float, List or Object."
)

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

logger = logging.getLogger(__name__)

# FHIR Enums for Scheduling
@strawberry.enum
class AppointmentStatus(str, Enum):
    """FHIR AppointmentStatus enum"""
    PROPOSED = "proposed"
    PENDING = "pending"
    BOOKED = "booked"
    ARRIVED = "arrived"
    FULFILLED = "fulfilled"
    CANCELLED = "cancelled"
    NOSHOW = "noshow"
    ENTERED_IN_ERROR = "entered-in-error"
    CHECKED_IN = "checked-in"
    WAITLIST = "waitlist"

@strawberry.enum
class ParticipationStatus(str, Enum):
    """FHIR ParticipationStatus enum"""
    ACCEPTED = "accepted"
    DECLINED = "declined"
    TENTATIVE = "tentative"
    NEEDS_ACTION = "needs-action"

@strawberry.enum
class SlotStatus(str, Enum):
    """FHIR SlotStatus enum"""
    BUSY = "busy"
    FREE = "free"
    BUSY_UNAVAILABLE = "busy-unavailable"
    BUSY_TENTATIVE = "busy-tentative"
    ENTERED_IN_ERROR = "entered-in-error"

@strawberry.enum
class ParticipantRequired(str, Enum):
    """FHIR ParticipantRequired enum"""
    REQUIRED = "required"
    OPTIONAL = "optional"
    INFORMATION_ONLY = "information-only"

# Shared FHIR Types (compatible with other services)
# ---- SHARED TYPES ----
# These types are shared across services - using simple definitions to avoid federation conflicts

@strawberry.type
class CodeableConcept:
    """FHIR CodeableConcept type"""
    text: Optional[str] = None
    coding: Optional[List["Coding"]] = None

@strawberry.type
class Coding:
    """FHIR Coding type"""
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    version: Optional[str] = None
    user_selected: Optional[bool] = None

@strawberry.type
class Reference:
    """FHIR Reference type"""
    reference: Optional[str] = None
    display: Optional[str] = None
    type: Optional[str] = None
    identifier: Optional["Identifier"] = None

@strawberry.type
class Identifier:
    """FHIR Identifier type"""
    use: Optional[str] = None
    type: Optional["CodeableConcept"] = None
    system: Optional[str] = None
    value: Optional[str] = None
    period: Optional["Period"] = None
    assigner: Optional["Reference"] = None

@strawberry.type
class Period:
    """FHIR Period type"""
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.type
class ContactPoint:
    """FHIR ContactPoint type"""
    system: Optional[str] = None
    value: Optional[str] = None
    use: Optional[str] = None
    rank: Optional[int] = None
    period: Optional[GenericScalar] = None

# Appointment Participant
@strawberry.type
class AppointmentParticipant:
    """FHIR Appointment Participant"""
    type: Optional[List[CodeableConcept]] = None
    actor: Optional[Reference] = None
    required: Optional[ParticipantRequired] = None
    status: ParticipationStatus
    period: Optional[Period] = None

# Core Appointment Type
@strawberry.federation.type(keys=["id"])
class Appointment:
    """FHIR Appointment resource for appointment scheduling"""
    id: strawberry.ID
    resource_type: str = "Appointment"
    identifier: Optional[List[Identifier]] = None
    status: AppointmentStatus
    cancellation_reason: Optional[CodeableConcept] = None
    service_category: Optional[List[CodeableConcept]] = None
    service_type: Optional[List[CodeableConcept]] = None
    specialty: Optional[List[CodeableConcept]] = None
    appointment_type: Optional[CodeableConcept] = None
    reason_code: Optional[List[CodeableConcept]] = None
    reason_reference: Optional[List[Reference]] = None
    priority: Optional[int] = None
    description: Optional[str] = None
    supporting_information: Optional[List[Reference]] = None
    start: Optional[str] = None
    end: Optional[str] = None
    minutes_duration: Optional[int] = None
    slot: Optional[List[Reference]] = None
    created: Optional[str] = None
    comment: Optional[str] = None
    patient_instruction: Optional[str] = None
    based_on: Optional[List[Reference]] = None
    participant: Optional[List[AppointmentParticipant]] = None
    requested_period: Optional[List[Period]] = None

# Schedule Type
@strawberry.federation.type(keys=["id"])
class Schedule:
    """FHIR Schedule resource for provider schedules"""
    id: strawberry.ID
    resource_type: str = "Schedule"
    identifier: Optional[List[Identifier]] = None
    active: Optional[bool] = None
    service_category: Optional[List[CodeableConcept]] = None
    service_type: Optional[List[CodeableConcept]] = None
    specialty: Optional[List[CodeableConcept]] = None
    actor: List[Reference]
    planning_horizon: Optional[Period] = None
    comment: Optional[str] = None

# Slot Type
@strawberry.federation.type(keys=["id"])
class Slot:
    """FHIR Slot resource for available appointment slots"""
    id: strawberry.ID
    resource_type: str = "Slot"
    identifier: Optional[List[Identifier]] = None
    service_category: Optional[List[CodeableConcept]] = None
    service_type: Optional[List[CodeableConcept]] = None
    specialty: Optional[List[CodeableConcept]] = None
    appointment_type: Optional[CodeableConcept] = None
    schedule: Reference
    status: SlotStatus
    start: str
    end: str
    overbooked: Optional[bool] = None
    comment: Optional[str] = None

# Input Types for Mutations
@strawberry.input
class CodingInput:
    """Input type for Coding"""
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    version: Optional[str] = None
    user_selected: Optional[bool] = None

@strawberry.input
class CodeableConceptInput:
    """Input type for CodeableConcept"""
    text: Optional[str] = None
    coding: Optional[List[CodingInput]] = None

@strawberry.input
class ReferenceInput:
    """Input type for Reference"""
    reference: Optional[str] = None
    display: Optional[str] = None
    type: Optional[str] = None

@strawberry.input
class IdentifierInput:
    """Input type for Identifier"""
    use: Optional[str] = None
    system: Optional[str] = None
    value: Optional[str] = None
    period: Optional["PeriodInput"] = None

@strawberry.input
class PeriodInput:
    """Input type for Period"""
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.input
class AppointmentParticipantInput:
    """Input type for Appointment Participant"""
    type: Optional[List[CodeableConceptInput]] = None
    actor: Optional[ReferenceInput] = None
    required: Optional[ParticipantRequired] = None
    status: ParticipationStatus
    period: Optional[PeriodInput] = None

@strawberry.input
class AppointmentInput:
    """Input type for creating/updating appointments"""
    status: AppointmentStatus
    service_category: Optional[List[CodeableConceptInput]] = None
    service_type: Optional[List[CodeableConceptInput]] = None
    specialty: Optional[List[CodeableConceptInput]] = None
    appointment_type: Optional[CodeableConceptInput] = None
    reason_code: Optional[List[CodeableConceptInput]] = None
    priority: Optional[int] = None
    description: Optional[str] = None
    start: Optional[str] = None
    end: Optional[str] = None
    minutes_duration: Optional[int] = None
    comment: Optional[str] = None
    patient_instruction: Optional[str] = None
    participant: Optional[List[AppointmentParticipantInput]] = None

@strawberry.input
class ScheduleInput:
    """Input type for creating/updating schedules"""
    identifier: Optional[List[IdentifierInput]] = None
    active: Optional[bool] = None
    service_category: Optional[List[CodeableConceptInput]] = None
    service_type: Optional[List[CodeableConceptInput]] = None
    specialty: Optional[List[CodeableConceptInput]] = None
    actor: List[ReferenceInput]
    planning_horizon: Optional[PeriodInput] = None
    comment: Optional[str] = None

@strawberry.input
class SlotInput:
    """Input type for creating/updating slots"""
    identifier: Optional[List[IdentifierInput]] = None
    service_category: Optional[List[CodeableConceptInput]] = None
    service_type: Optional[List[CodeableConceptInput]] = None
    specialty: Optional[List[CodeableConceptInput]] = None
    appointment_type: Optional[CodeableConceptInput] = None
    schedule: ReferenceInput
    status: SlotStatus
    start: str
    end: str
    overbooked: Optional[bool] = None
    comment: Optional[str] = None

# Federation Extensions for other services
@strawberry.federation.type(keys=["id"], extend=True)
class Patient:
    """Extended Patient entity from Patient Service with scheduling."""
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def appointments(self) -> List[Appointment]:
        """Get appointments for this patient"""
        try:
            from app.services.fhir_service_factory import get_fhir_service
            fhir_service = get_fhir_service()

            if fhir_service:
                # Search for appointments by patient
                search_params = {"patient": f"Patient/{self.id}"}
                appointments_data = await fhir_service.search_appointments(search_params)

                # Convert to GraphQL types
                appointments = []
                for apt_data in appointments_data:
                    appointment = _convert_fhir_appointment_to_graphql(apt_data)
                    if appointment:
                        appointments.append(appointment)

                return appointments

            return []
        except Exception as e:
            logger.error(f"Error fetching patient appointments: {e}")
            return []

@strawberry.federation.type(keys=["id"], extend=True)
class User:
    """Extended User entity from Organization Service with scheduling (represents practitioners)."""
    id: strawberry.ID = strawberry.federation.field(external=True)

    @strawberry.field
    async def appointments(self) -> List[Appointment]:
        """Get appointments for this user/practitioner"""
        try:
            from app.services.fhir_service_factory import get_fhir_service
            fhir_service = get_fhir_service()

            if fhir_service:
                # Search for appointments by practitioner (using User ID as Practitioner ID)
                search_params = {"practitioner": f"Practitioner/{self.id}"}
                appointments_data = await fhir_service.search_appointments(search_params)

                # Convert to GraphQL types
                appointments = []
                for apt_data in appointments_data:
                    appointment = _convert_fhir_appointment_to_graphql(apt_data)
                    if appointment:
                        appointments.append(appointment)

                return appointments

            return []
        except Exception as e:
            logger.error(f"Error fetching user appointments: {e}")
            return []

    @strawberry.field
    async def schedules(self) -> List[Schedule]:
        """Get schedules for this user/practitioner"""
        try:
            from app.services.fhir_service_factory import get_fhir_service
            fhir_service = get_fhir_service()

            if fhir_service:
                # Search for schedules by actor (using User ID as Practitioner ID)
                search_params = {"actor": f"Practitioner/{self.id}"}
                schedules_data = await fhir_service.search_schedules(search_params)

                # Convert to GraphQL types
                schedules = []
                for schedule_data in schedules_data:
                    schedule = _convert_fhir_schedule_to_graphql(schedule_data)
                    if schedule:
                        schedules.append(schedule)

                return schedules

            return []
        except Exception as e:
            logger.error(f"Error fetching user schedules: {e}")
            return []

# Helper functions to convert FHIR data to GraphQL types
def _convert_fhir_appointment_to_graphql(fhir_data: Dict[str, Any]) -> Optional[Appointment]:
    """Convert FHIR Appointment data to GraphQL Appointment type"""
    try:
        # Extract basic fields
        appointment_id = fhir_data.get("id")
        status = fhir_data.get("status", "proposed")

        # Convert participants
        participants = []
        if "participant" in fhir_data:
            for p in fhir_data["participant"]:
                participant = AppointmentParticipant(
                    status=ParticipationStatus(p.get("status", "needs-action")),
                    actor=Reference(
                        reference=p.get("actor", {}).get("reference"),
                        display=p.get("actor", {}).get("display")
                    ) if p.get("actor") else None,
                    required=ParticipantRequired(p.get("required", "optional")) if p.get("required") else None
                )
                participants.append(participant)

        # Create appointment
        appointment = Appointment(
            id=strawberry.ID(appointment_id),
            status=AppointmentStatus(status),
            description=fhir_data.get("description"),
            start=fhir_data.get("start"),
            end=fhir_data.get("end"),
            minutes_duration=fhir_data.get("minutesDuration"),
            comment=fhir_data.get("comment"),
            participant=participants if participants else None
        )

        return appointment

    except Exception as e:
        logger.error(f"Error converting FHIR appointment to GraphQL: {e}")
        return None

def _convert_fhir_schedule_to_graphql(fhir_data: Dict[str, Any]) -> Optional[Schedule]:
    """Convert FHIR Schedule data to GraphQL Schedule type"""
    try:
        # Extract basic fields
        schedule_id = fhir_data.get("id")

        # Convert actors
        actors = []
        if "actor" in fhir_data:
            for actor_data in fhir_data["actor"]:
                actor = Reference(
                    reference=actor_data.get("reference"),
                    display=actor_data.get("display")
                )
                actors.append(actor)

        # Create schedule
        schedule = Schedule(
            id=strawberry.ID(schedule_id),
            active=fhir_data.get("active", True),
            actor=actors,
            comment=fhir_data.get("comment")
        )

        return schedule

    except Exception as e:
        logger.error(f"Error converting FHIR schedule to GraphQL: {e}")
        return None

def _convert_fhir_slot_to_graphql(fhir_data: Dict[str, Any]) -> Optional[Slot]:
    """Convert FHIR Slot data to GraphQL Slot type"""
    try:
        # Extract basic fields
        slot_id = fhir_data.get("id")
        status = fhir_data.get("status", "free")

        # Create slot
        slot = Slot(
            id=strawberry.ID(slot_id),
            status=SlotStatus(status),
            start=fhir_data.get("start"),
            end=fhir_data.get("end"),
            schedule=Reference(
                reference=fhir_data.get("schedule", {}).get("reference"),
                display=fhir_data.get("schedule", {}).get("display")
            ),
            overbooked=fhir_data.get("overbooked"),
            comment=fhir_data.get("comment")
        )

        return slot

    except Exception as e:
        logger.error(f"Error converting FHIR slot to GraphQL: {e}")
        return None

# Global FHIR service instance
_fhir_service = None

async def get_fhir_service():
    """Get or create the FHIR service instance."""
    global _fhir_service
    if _fhir_service is None:
        from app.services.fhir_service_factory import initialize_fhir_service
        _fhir_service = await initialize_fhir_service()
    return _fhir_service

@strawberry.type
class Query:
    """Root query type for the Scheduling Service."""

    @strawberry.field
    async def appointment(self, id: strawberry.ID) -> Optional[Appointment]:
        """Get a specific appointment by ID"""
        try:
            fhir_service = await get_fhir_service()
            fhir_resource = await fhir_service.get_appointment(str(id))

            if fhir_resource:
                return _convert_fhir_appointment_to_graphql(fhir_resource)
            return None

        except Exception as e:
            logger.error(f"Error retrieving appointment {id}: {str(e)}")
            return None

    @strawberry.field
    async def appointments(
        self,
        patient_id: Optional[str] = None,
        practitioner_id: Optional[str] = None,
        status: Optional[AppointmentStatus] = None,
        date: Optional[str] = None
    ) -> List[Appointment]:
        """Search for appointments with optional filters"""
        try:
            fhir_service = await get_fhir_service()

            # Build search parameters
            search_params = {}
            if patient_id:
                search_params["patient"] = f"Patient/{patient_id}"
            if practitioner_id:
                search_params["practitioner"] = f"Practitioner/{practitioner_id}"
            if status:
                search_params["status"] = status.value
            if date:
                search_params["date"] = date

            appointments_data = await fhir_service.search_appointments(search_params)

            # Convert to GraphQL types
            appointments = []
            for apt_data in appointments_data:
                appointment = _convert_fhir_appointment_to_graphql(apt_data)
                if appointment:
                    appointments.append(appointment)

            return appointments

        except Exception as e:
            logger.error(f"Error searching appointments: {str(e)}")
            return []

    @strawberry.field
    async def schedule(self, id: strawberry.ID) -> Optional[Schedule]:
        """Get a specific schedule by ID"""
        try:
            fhir_service = await get_fhir_service()
            fhir_resource = await fhir_service.get_schedule(str(id))

            if fhir_resource:
                return _convert_fhir_schedule_to_graphql(fhir_resource)
            return None

        except Exception as e:
            logger.error(f"Error retrieving schedule {id}: {str(e)}")
            return None

    @strawberry.field
    async def schedules(
        self,
        actor_id: Optional[str] = None,
        active: Optional[bool] = None
    ) -> List[Schedule]:
        """Search for schedules with optional filters"""
        try:
            fhir_service = await get_fhir_service()

            # Build search parameters
            search_params = {}
            if actor_id:
                search_params["actor"] = f"Practitioner/{actor_id}"
            if active is not None:
                search_params["active"] = str(active).lower()

            schedules_data = await fhir_service.search_schedules(search_params)

            # Convert to GraphQL types
            schedules = []
            for schedule_data in schedules_data:
                schedule = _convert_fhir_schedule_to_graphql(schedule_data)
                if schedule:
                    schedules.append(schedule)

            return schedules

        except Exception as e:
            logger.error(f"Error searching schedules: {str(e)}")
            return []

    @strawberry.field
    async def slot(self, id: strawberry.ID) -> Optional[Slot]:
        """Get a specific slot by ID"""
        try:
            fhir_service = await get_fhir_service()
            fhir_resource = await fhir_service.get_slot(str(id))

            if fhir_resource:
                return _convert_fhir_slot_to_graphql(fhir_resource)
            return None

        except Exception as e:
            logger.error(f"Error retrieving slot {id}: {str(e)}")
            return None

    @strawberry.field
    async def slots(
        self,
        schedule_id: Optional[str] = None,
        status: Optional[SlotStatus] = None,
        start: Optional[str] = None,
        end: Optional[str] = None
    ) -> List[Slot]:
        """Search for slots with optional filters"""
        try:
            fhir_service = await get_fhir_service()

            # Build search parameters
            search_params = {}
            if schedule_id:
                search_params["schedule"] = f"Schedule/{schedule_id}"
            if status:
                search_params["status"] = status.value
            if start:
                search_params["start"] = start
            if end:
                search_params["end"] = end

            slots_data = await fhir_service.search_slots(search_params)

            # Convert to GraphQL types
            slots = []
            for slot_data in slots_data:
                slot = _convert_fhir_slot_to_graphql(slot_data)
                if slot:
                    slots.append(slot)

            return slots

        except Exception as e:
            logger.error(f"Error searching slots: {str(e)}")
            return []

@strawberry.type
class Mutation:
    """Root mutation type for the Scheduling Service."""

    @strawberry.field
    async def create_appointment(self, appointment: AppointmentInput) -> Optional[Appointment]:
        """Create a new appointment"""
        try:
            fhir_service = await get_fhir_service()

            # Convert GraphQL input to FHIR format
            fhir_data = {
                "resourceType": "Appointment",
                "status": appointment.status.value,
                "description": appointment.description,
                "start": appointment.start,
                "end": appointment.end,
                "minutesDuration": appointment.minutes_duration,
                "comment": appointment.comment,
                "patientInstruction": appointment.patient_instruction
            }

            # Add participants if provided
            if appointment.participant:
                participants = []
                for p in appointment.participant:
                    participant_data = {
                        "status": p.status.value,
                        "required": p.required.value if p.required else "optional"
                    }
                    if p.actor:
                        participant_data["actor"] = {
                            "reference": p.actor.reference,
                            "display": p.actor.display
                        }
                    participants.append(participant_data)
                fhir_data["participant"] = participants

            # Create the appointment
            created_resource = await fhir_service.create_appointment(fhir_data)

            if created_resource:
                return _convert_fhir_appointment_to_graphql(created_resource)
            return None

        except Exception as e:
            logger.error(f"Error creating appointment: {str(e)}")
            return None

    @strawberry.field
    async def update_appointment(self, id: strawberry.ID, appointment: AppointmentInput) -> Optional[Appointment]:
        """Update an existing appointment"""
        try:
            fhir_service = await get_fhir_service()

            # Get the existing appointment first
            existing_appointment = await fhir_service.get_appointment(str(id))
            if not existing_appointment:
                logger.error(f"Appointment {id} not found")
                return None

            # Create update data with only the fields that are provided
            update_data = {}

            # Only update fields that are provided in the input
            if appointment.status is not None:
                update_data["status"] = appointment.status.value
            if appointment.description is not None:
                update_data["description"] = appointment.description
            if appointment.start is not None:
                update_data["start"] = appointment.start
            if appointment.end is not None:
                update_data["end"] = appointment.end
            if appointment.minutes_duration is not None:
                update_data["minutesDuration"] = appointment.minutes_duration
            if appointment.comment is not None:
                update_data["comment"] = appointment.comment
            if appointment.patient_instruction is not None:
                update_data["patientInstruction"] = appointment.patient_instruction

            # Add participants if provided
            if appointment.participant is not None:
                participants = []
                for p in appointment.participant:
                    participant_data = {
                        "status": p.status.value,
                        "required": p.required.value if p.required else "optional"
                    }
                    if p.actor:
                        participant_data["actor"] = {
                            "reference": p.actor.reference,
                            "display": p.actor.display
                        }
                    participants.append(participant_data)
                update_data["participant"] = participants

            # Update the appointment with only the changed fields
            updated_resource = await fhir_service.update_appointment(str(id), update_data)

            if updated_resource:
                return _convert_fhir_appointment_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error updating appointment: {str(e)}")
            return None

    @strawberry.field
    async def cancel_appointment(self, id: strawberry.ID, reason: Optional[str] = None) -> Optional[Appointment]:
        """Cancel an appointment"""
        try:
            fhir_service = await get_fhir_service()

            # Get the current appointment
            current_appointment = await fhir_service.get_appointment(str(id))
            if not current_appointment:
                return None

            # Update status to cancelled
            current_appointment["status"] = "cancelled"
            if reason:
                current_appointment["comment"] = reason

            # Update the appointment
            updated_resource = await fhir_service.update_appointment(str(id), current_appointment)

            if updated_resource:
                return _convert_fhir_appointment_to_graphql(updated_resource)
            return None

        except Exception as e:
            logger.error(f"Error cancelling appointment: {str(e)}")
            return None

    @strawberry.field
    async def create_schedule(self, schedule: ScheduleInput) -> Optional[Schedule]:
        """Create a new schedule"""
        try:
            fhir_service = await get_fhir_service()

            # Convert GraphQL input to FHIR format
            fhir_data = {
                "resourceType": "Schedule",
                "active": schedule.active if schedule.active is not None else True,
                "comment": schedule.comment
            }

            # Add actors
            if schedule.actor:
                actors = []
                for actor in schedule.actor:
                    actor_data = {
                        "reference": actor.reference,
                        "display": actor.display
                    }
                    actors.append(actor_data)
                fhir_data["actor"] = actors

            # Create the schedule
            created_resource = await fhir_service.create_schedule(fhir_data)

            if created_resource:
                return _convert_fhir_schedule_to_graphql(created_resource)
            return None

        except Exception as e:
            logger.error(f"Error creating schedule: {str(e)}")
            return None

    @strawberry.field
    async def create_slot(self, slot: SlotInput) -> Optional[Slot]:
        """Create a new slot"""
        try:
            fhir_service = await get_fhir_service()

            # Convert GraphQL input to FHIR format
            fhir_data = {
                "resourceType": "Slot",
                "status": slot.status.value,
                "start": slot.start,
                "end": slot.end,
                "overbooked": slot.overbooked,
                "comment": slot.comment,
                "schedule": {
                    "reference": slot.schedule.reference,
                    "display": slot.schedule.display
                }
            }

            # Create the slot
            created_resource = await fhir_service.create_slot(fhir_data)

            if created_resource:
                return _convert_fhir_slot_to_graphql(created_resource)
            return None

        except Exception as e:
            logger.error(f"Error creating slot: {str(e)}")
            return None

# Create the comprehensive federated schema
schema = strawberry.federation.Schema(
    query=Query,
    mutation=Mutation,
    types=[
        Patient, User,  # Federation extensions
        Appointment, Schedule, Slot,  # Core scheduling types
        AppointmentParticipant,  # Complex types
        CodeableConcept, Coding, Reference, Identifier, Period, ContactPoint,  # Shared FHIR types
        AppointmentStatus, ParticipationStatus, SlotStatus, ParticipantRequired,  # Enums
    ]
)
