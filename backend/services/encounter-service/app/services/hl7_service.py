from typing import Dict, List, Optional, Any, Union
import hl7
import logging
from datetime import datetime
from app.models.hl7 import ADTMessage, HL7MessageRequest
from app.services.encounter_service import encounter_service
from app.models.encounter import EncounterCreate, Reference, CodeableConcept, Period

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class HL7Service:
    """Service for processing HL7 messages."""

    async def process_message(self, message_request: HL7MessageRequest, auth_header: str) -> Dict[str, Any]:
        """
        Process an HL7 message.

        Args:
            message_request: The HL7 message request
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        try:
            # Parse the HL7 message
            message_str = message_request.message
            parsed_message = hl7.parse(message_str)

            # Get the message type
            message_type = str(parsed_message.segment('MSH')[9][0])
            trigger_event = str(parsed_message.segment('MSH')[9][1])

            logger.info(f"Processing HL7 message of type {message_type}^{trigger_event}")

            # Process based on message type
            if message_type == "ADT":
                return await self.process_adt_message(parsed_message, message_str, auth_header)
            else:
                return {"status": "error", "message": f"Unsupported message type: {message_type}"}
        except Exception as e:
            logger.error(f"Error processing HL7 message: {str(e)}")
            return {"status": "error", "message": str(e)}

    async def process_adt_message(self, parsed_message: Any, raw_message: str, auth_header: str) -> Dict[str, Any]:
        """
        Process an ADT message.

        Args:
            parsed_message: The parsed HL7 message
            raw_message: The raw HL7 message
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        try:
            # Extract message details
            message_type = str(parsed_message.segment('MSH')[9][0])
            trigger_event = str(parsed_message.segment('MSH')[9][1])
            message_control_id = str(parsed_message.segment('MSH')[10])
            message_datetime_str = str(parsed_message.segment('MSH')[7])
            
            # Parse message datetime
            try:
                message_datetime = datetime.strptime(message_datetime_str, "%Y%m%d%H%M%S")
            except:
                message_datetime = datetime.now()

            # Extract patient details
            patient_id = str(parsed_message.segment('PID')[3][0])
            patient_id_type = str(parsed_message.segment('PID')[3][4]) if len(parsed_message.segment('PID')[3]) > 4 else None
            patient_id_authority = str(parsed_message.segment('PID')[3][3]) if len(parsed_message.segment('PID')[3]) > 3 else None
            patient_name_family = str(parsed_message.segment('PID')[5][0])
            patient_name_given = [str(parsed_message.segment('PID')[5][1])]
            
            # Extract visit details
            visit_number = str(parsed_message.segment('PV1')[19]) if len(parsed_message.segment('PV1')) > 19 else None
            visit_class = str(parsed_message.segment('PV1')[2])
            
            # Create ADT message model
            adt_message = ADTMessage(
                message_type=message_type,
                event_type=trigger_event,
                message_control_id=message_control_id,
                message_datetime=message_datetime,
                raw_message=raw_message,
                patient_id=patient_id,
                patient_id_type=patient_id_type,
                patient_id_authority=patient_id_authority,
                patient_name_family=patient_name_family,
                patient_name_given=patient_name_given,
                visit_number=visit_number,
                visit_class=visit_class
            )

            # Process based on trigger event
            if trigger_event == "A01":  # Admission
                return await self.process_admission(adt_message, auth_header)
            elif trigger_event == "A02":  # Transfer
                return await self.process_transfer(adt_message, auth_header)
            elif trigger_event == "A03":  # Discharge
                return await self.process_discharge(adt_message, auth_header)
            elif trigger_event == "A04":  # Registration
                return await self.process_registration(adt_message, auth_header)
            elif trigger_event == "A08":  # Update
                return await self.process_update(adt_message, auth_header)
            else:
                return {"status": "error", "message": f"Unsupported trigger event: {trigger_event}"}
        except Exception as e:
            logger.error(f"Error processing ADT message: {str(e)}")
            return {"status": "error", "message": str(e)}

    async def process_admission(self, adt_message: ADTMessage, auth_header: str) -> Dict[str, Any]:
        """
        Process an ADT-A01 (Admission) message.

        Args:
            adt_message: The ADT message
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        try:
            # Create an encounter from the ADT message
            encounter = EncounterCreate(
                status="in-progress",
                class_value={
                    "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                    "code": self._map_visit_class(adt_message.visit_class),
                    "display": self._get_visit_class_display(self._map_visit_class(adt_message.visit_class))
                },
                subject=Reference(
                    reference=f"Patient/{adt_message.patient_id}",
                    display=f"{adt_message.patient_name_given[0]} {adt_message.patient_name_family}"
                ),
                period=Period(
                    start=adt_message.message_datetime.isoformat()
                ),
                type=[
                    CodeableConcept(
                        coding=[
                            {
                                "system": "http://terminology.hl7.org/CodeSystem/v2-0007",
                                "code": "ADM",
                                "display": "Admission"
                            }
                        ],
                        text="Hospital admission"
                    )
                ]
            )

            # Create the encounter in the FHIR server
            created_encounter = await encounter_service.create_encounter(encounter, auth_header)

            return {
                "status": "success",
                "message": "ADT-A01 (Admission) message processed successfully",
                "encounter": created_encounter
            }
        except Exception as e:
            logger.error(f"Error processing admission: {str(e)}")
            return {"status": "error", "message": str(e)}

    async def process_transfer(self, adt_message: ADTMessage, auth_header: str) -> Dict[str, Any]:
        """
        Process an ADT-A02 (Transfer) message.

        Args:
            adt_message: The ADT message
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        # For now, just return a placeholder
        return {
            "status": "success",
            "message": "ADT-A02 (Transfer) message processed successfully",
            "details": "Transfer processing not fully implemented yet"
        }

    async def process_discharge(self, adt_message: ADTMessage, auth_header: str) -> Dict[str, Any]:
        """
        Process an ADT-A03 (Discharge) message.

        Args:
            adt_message: The ADT message
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        try:
            # Find the active encounter for this patient
            encounters = await encounter_service.get_patient_encounters(
                adt_message.patient_id,
                {"status": "in-progress"},
                auth_header
            )

            if not encounters:
                return {
                    "status": "warning",
                    "message": "No active encounter found for discharge"
                }

            # Update the encounter to mark it as finished
            encounter_id = encounters[0]["id"]
            updated_encounter = await encounter_service.update_encounter(
                encounter_id,
                {
                    "status": "finished",
                    "period": {
                        "end": adt_message.message_datetime.isoformat()
                    }
                },
                auth_header
            )

            return {
                "status": "success",
                "message": "ADT-A03 (Discharge) message processed successfully",
                "encounter": updated_encounter
            }
        except Exception as e:
            logger.error(f"Error processing discharge: {str(e)}")
            return {"status": "error", "message": str(e)}

    async def process_registration(self, adt_message: ADTMessage, auth_header: str) -> Dict[str, Any]:
        """
        Process an ADT-A04 (Registration) message.

        Args:
            adt_message: The ADT message
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        try:
            # Create an encounter from the ADT message
            encounter = EncounterCreate(
                status="arrived",
                class_value={
                    "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                    "code": self._map_visit_class(adt_message.visit_class),
                    "display": self._get_visit_class_display(self._map_visit_class(adt_message.visit_class))
                },
                subject=Reference(
                    reference=f"Patient/{adt_message.patient_id}",
                    display=f"{adt_message.patient_name_given[0]} {adt_message.patient_name_family}"
                ),
                period=Period(
                    start=adt_message.message_datetime.isoformat()
                ),
                type=[
                    CodeableConcept(
                        coding=[
                            {
                                "system": "http://terminology.hl7.org/CodeSystem/v2-0007",
                                "code": "REG",
                                "display": "Registration"
                            }
                        ],
                        text="Patient registration"
                    )
                ]
            )

            # Create the encounter in the FHIR server
            created_encounter = await encounter_service.create_encounter(encounter, auth_header)

            return {
                "status": "success",
                "message": "ADT-A04 (Registration) message processed successfully",
                "encounter": created_encounter
            }
        except Exception as e:
            logger.error(f"Error processing registration: {str(e)}")
            return {"status": "error", "message": str(e)}

    async def process_update(self, adt_message: ADTMessage, auth_header: str) -> Dict[str, Any]:
        """
        Process an ADT-A08 (Update) message.

        Args:
            adt_message: The ADT message
            auth_header: The authorization header for API calls

        Returns:
            The processing result
        """
        # For now, just return a placeholder
        return {
            "status": "success",
            "message": "ADT-A08 (Update) message processed successfully",
            "details": "Update processing not fully implemented yet"
        }

    def _map_visit_class(self, visit_class: str) -> str:
        """Map HL7 visit class to FHIR encounter class."""
        mapping = {
            "I": "IMP",  # Inpatient
            "O": "AMB",  # Outpatient
            "E": "EMER",  # Emergency
            "P": "AMB",  # Pre-admitted
            "R": "AMB",  # Recurring patient
            "B": "AMB",  # Obstetrics
            "C": "AMB",  # Commercial account
            "N": "AMB",  # Not applicable
            "U": "UNK",  # Unknown
        }
        return mapping.get(visit_class, "AMB")

    def _get_visit_class_display(self, visit_class: str) -> str:
        """Get display name for FHIR encounter class."""
        mapping = {
            "IMP": "inpatient",
            "AMB": "ambulatory",
            "EMER": "emergency",
            "VR": "virtual",
            "HH": "home health",
            "UNK": "unknown"
        }
        return mapping.get(visit_class, "ambulatory")

# Create a singleton instance
hl7_service = HL7Service()
