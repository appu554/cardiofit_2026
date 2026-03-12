import hl7
import hl7apy
from hl7apy.parser import parse_message
from hl7apy.core import Message
from typing import Dict, List, Any, Optional, Tuple, Union
from datetime import datetime
import uuid
import logging
from app.models.hl7 import ADTMessage
from app.models.fhir import Patient, Encounter, Observation, Identifier, HumanName, ContactPoint, Address

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

def parse_hl7_message(message_str: str) -> Tuple[Any, str]:
    """
    Parse an HL7 message string into a parsed message object.

    Args:
        message_str: The raw HL7 message string

    Returns:
        Tuple containing the parsed message and the message type
    """
    # First try with the simpler hl7 library which is more forgiving
    try:
        # Replace any Windows line endings with proper HL7 segment separators
        message_str = message_str.replace('\r\n', '\r').replace('\n', '\r')
        if not message_str.endswith('\r'):
            message_str += '\r'

        message = hl7.parse(message_str)

        # Extract message type from MSH segment
        msh = message.segment('MSH')
        if len(msh) > 9 and len(msh[9]) > 1:
            message_type = str(msh[9][0])
            event_type = str(msh[9][1])
            logger.info(f"Parsed HL7 message of type {message_type}^{event_type} using hl7 parser")
            return message, f"{message_type}^{event_type}"
        else:
            raise ValueError("MSH segment does not contain message type information")
    except Exception as e:
        logger.error(f"Error parsing with hl7 library: {str(e)}")

        # Try with hl7apy as a fallback
        try:
            # Try to determine the version from the message
            version_match = None
            for line in message_str.split('\r'):
                if line.startswith('MSH'):
                    parts = line.split('|')
                    if len(parts) > 12:
                        version_match = parts[11]
                        break

            # Use a supported version
            version = "2.3"  # Default to 2.3 which is widely supported
            if version_match in ["2.3", "2.3.1", "2.4", "2.5", "2.6"]:
                version = version_match

            message = parse_message(message_str, version=version)

            # Get message type and event
            message_type = message.msh.msh_9.msh_9_1.value
            event_type = message.msh.msh_9.msh_9_2.value

            logger.info(f"Parsed HL7 message of type {message_type}^{event_type} using hl7apy parser")
            return message, f"{message_type}^{event_type}"
        except Exception as fallback_error:
            logger.error(f"hl7apy parsing also failed: {str(fallback_error)}")

            # Last resort: manual parsing for basic information
            try:
                message_type = "UNKNOWN"
                event_type = "UNKNOWN"

                for line in message_str.split('\r'):
                    if line.startswith('MSH'):
                        parts = line.split('|')
                        if len(parts) > 9:
                            message_type_field = parts[8].split('^')
                            if len(message_type_field) > 1:
                                message_type = message_type_field[0]
                                event_type = message_type_field[1]
                                break

                if message_type != "UNKNOWN":
                    logger.info(f"Parsed HL7 message of type {message_type}^{event_type} using manual parsing")
                    # Create a simple dict-like object to represent the message
                    parsed_message = {"raw": message_str, "type": message_type, "event": event_type}
                    return parsed_message, f"{message_type}^{event_type}"
                else:
                    raise ValueError("Could not determine message type from MSH segment")
            except Exception as manual_error:
                logger.error(f"Manual parsing also failed: {str(manual_error)}")
                raise ValueError(f"Could not parse HL7 message: {str(e)}")

def extract_adt_data(message: Any) -> ADTMessage:
    """
    Extract data from an ADT message into our ADTMessage model.

    Args:
        message: The parsed HL7 message

    Returns:
        ADTMessage object with extracted data
    """
    try:
        # Initialize variables with default values
        message_control_id = "UNKNOWN"
        message_datetime = datetime.now()
        event_type = "UNKNOWN"
        patient_id = "UNKNOWN"
        patient_id_type = None
        patient_id_authority = None
        patient_name_family = "UNKNOWN"
        patient_name_given = ["UNKNOWN"]
        patient_dob = None
        patient_gender = None
        visit_number = None
        visit_class = None
        assigned_location = None
        admit_datetime = None
        discharge_datetime = None
        raw_message = str(message)

        # Using hl7apy
        if hasattr(message, 'msh') and hasattr(message.msh, 'msh_9'):
            # Extract message metadata
            message_control_id = message.msh.msh_10.value if hasattr(message.msh, 'msh_10') else "UNKNOWN"

            message_datetime_str = message.msh.msh_7.value if hasattr(message.msh, 'msh_7') else None
            if message_datetime_str:
                try:
                    message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d%H%M%S')
                except ValueError:
                    # Try alternative formats
                    try:
                        message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d')
                    except ValueError:
                        logger.warning(f"Could not parse message datetime: {message_datetime_str}")

            event_type = message.msh.msh_9.msh_9_2.value if hasattr(message.msh.msh_9, 'msh_9_2') else "UNKNOWN"

            # Extract patient information if PID segment exists
            if hasattr(message, 'pid'):
                if hasattr(message.pid, 'pid_3') and hasattr(message.pid.pid_3, 'pid_3_1'):
                    patient_id = message.pid.pid_3.pid_3_1.value
                    patient_id_type = message.pid.pid_3.pid_3_5.value if hasattr(message.pid.pid_3, 'pid_3_5') else None
                    patient_id_authority = message.pid.pid_3.pid_3_4.value if hasattr(message.pid.pid_3, 'pid_3_4') else None

                if hasattr(message.pid, 'pid_5') and hasattr(message.pid.pid_5, 'pid_5_1'):
                    patient_name_family = message.pid.pid_5.pid_5_1.value
                    patient_name_given = [message.pid.pid_5.pid_5_2.value] if hasattr(message.pid.pid_5, 'pid_5_2') else [""]
                    if hasattr(message.pid.pid_5, 'pid_5_3') and message.pid.pid_5.pid_5_3.value:
                        patient_name_given.append(message.pid.pid_5.pid_5_3.value)

                patient_dob_str = message.pid.pid_7.value if hasattr(message.pid, 'pid_7') else None
                if patient_dob_str:
                    try:
                        patient_dob = datetime.strptime(patient_dob_str, '%Y%m%d')
                    except ValueError:
                        logger.warning(f"Could not parse patient DOB: {patient_dob_str}")

                patient_gender = message.pid.pid_8.value if hasattr(message.pid, 'pid_8') else None

            # Extract visit information if PV1 segment exists
            if hasattr(message, 'pv1'):
                visit_number = message.pv1.pv1_19.value if hasattr(message.pv1, 'pv1_19') else None
                visit_class = message.pv1.pv1_2.value if hasattr(message.pv1, 'pv1_2') else None

                if hasattr(message.pv1, 'pv1_3'):
                    assigned_location = message.pv1.pv1_3.value

                admit_datetime_str = message.pv1.pv1_44.value if hasattr(message.pv1, 'pv1_44') else None
                if admit_datetime_str:
                    try:
                        admit_datetime = datetime.strptime(admit_datetime_str, '%Y%m%d%H%M%S')
                    except ValueError:
                        try:
                            admit_datetime = datetime.strptime(admit_datetime_str, '%Y%m%d')
                        except ValueError:
                            logger.warning(f"Could not parse admit datetime: {admit_datetime_str}")

                discharge_datetime_str = message.pv1.pv1_45.value if hasattr(message.pv1, 'pv1_45') else None
                if discharge_datetime_str:
                    try:
                        discharge_datetime = datetime.strptime(discharge_datetime_str, '%Y%m%d%H%M%S')
                    except ValueError:
                        try:
                            discharge_datetime = datetime.strptime(discharge_datetime_str, '%Y%m%d')
                        except ValueError:
                            logger.warning(f"Could not parse discharge datetime: {discharge_datetime_str}")

        # Using hl7 library
        elif hasattr(message, 'segment'):
            # Extract message metadata
            msh = message.segment('MSH')
            if len(msh) > 10:
                message_control_id = str(msh[10][0])

            if len(msh) > 7 and len(msh[7]) > 0:
                message_datetime_str = str(msh[7][0])
                try:
                    message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d%H%M%S')
                except ValueError:
                    # Try alternative formats
                    try:
                        message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d')
                    except ValueError:
                        logger.warning(f"Could not parse message datetime: {message_datetime_str}")

            if len(msh) > 9 and len(msh[9]) > 1:
                event_type = str(msh[9][1])

            # Extract patient information
            try:
                pid = message.segment('PID')

                if len(pid) > 3 and len(pid[3]) > 0:
                    patient_id = str(pid[3][0])
                    patient_id_type = str(pid[3][4]) if len(pid[3]) > 4 else None
                    patient_id_authority = str(pid[3][3]) if len(pid[3]) > 3 else None

                if len(pid) > 5 and len(pid[5]) > 0:
                    patient_name_family = str(pid[5][0])
                    patient_name_given = [str(pid[5][1])] if len(pid[5]) > 1 else [""]
                    if len(pid[5]) > 2:
                        patient_name_given.append(str(pid[5][2]))

                if len(pid) > 7 and len(pid[7]) > 0:
                    patient_dob_str = str(pid[7][0])
                    try:
                        patient_dob = datetime.strptime(patient_dob_str, '%Y%m%d')
                    except ValueError:
                        logger.warning(f"Could not parse patient DOB: {patient_dob_str}")

                if len(pid) > 8 and len(pid[8]) > 0:
                    patient_gender = str(pid[8][0])
            except Exception as pid_error:
                logger.warning(f"Error extracting PID data: {str(pid_error)}")

            # Extract visit information if PV1 segment exists
            try:
                if message.segments('PV1'):
                    pv1 = message.segment('PV1')

                    if len(pv1) > 19 and len(pv1[19]) > 0:
                        visit_number = str(pv1[19][0])

                    if len(pv1) > 2 and len(pv1[2]) > 0:
                        visit_class = str(pv1[2][0])

                    if len(pv1) > 3 and len(pv1[3]) > 0:
                        assigned_location = str(pv1[3][0])

                    if len(pv1) > 44 and len(pv1[44]) > 0:
                        admit_datetime_str = str(pv1[44][0])
                        try:
                            admit_datetime = datetime.strptime(admit_datetime_str, '%Y%m%d%H%M%S')
                        except ValueError:
                            try:
                                admit_datetime = datetime.strptime(admit_datetime_str, '%Y%m%d')
                            except ValueError:
                                logger.warning(f"Could not parse admit datetime: {admit_datetime_str}")

                    if len(pv1) > 45 and len(pv1[45]) > 0:
                        discharge_datetime_str = str(pv1[45][0])
                        try:
                            discharge_datetime = datetime.strptime(discharge_datetime_str, '%Y%m%d%H%M%S')
                        except ValueError:
                            try:
                                discharge_datetime = datetime.strptime(discharge_datetime_str, '%Y%m%d')
                            except ValueError:
                                logger.warning(f"Could not parse discharge datetime: {discharge_datetime_str}")
            except Exception as pv1_error:
                logger.warning(f"Error extracting PV1 data: {str(pv1_error)}")

        # Using dictionary (manual parsing)
        elif isinstance(message, dict):
            raw_message = message.get("raw", "")
            event_type = message.get("event", "UNKNOWN")

            # Parse the raw message to extract fields
            lines = raw_message.split('\r')
            for line in lines:
                if line.startswith('MSH'):
                    parts = line.split('|')
                    if len(parts) > 10:
                        message_control_id = parts[9]
                    if len(parts) > 7:
                        message_datetime_str = parts[6]
                        try:
                            message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d%H%M%S')
                        except ValueError:
                            try:
                                message_datetime = datetime.strptime(message_datetime_str, '%Y%m%d')
                            except ValueError:
                                logger.warning(f"Could not parse message datetime: {message_datetime_str}")

                elif line.startswith('PID'):
                    parts = line.split('|')
                    if len(parts) > 3:
                        id_parts = parts[3].split('^')
                        if id_parts:
                            patient_id = id_parts[0]
                            if len(id_parts) > 3:
                                patient_id_authority = id_parts[3]
                            if len(id_parts) > 4:
                                patient_id_type = id_parts[4]

                    if len(parts) > 5:
                        name_parts = parts[5].split('^')
                        if name_parts:
                            patient_name_family = name_parts[0]
                            patient_name_given = [name_parts[1]] if len(name_parts) > 1 else [""]
                            if len(name_parts) > 2:
                                patient_name_given.append(name_parts[2])

                    if len(parts) > 7 and parts[7]:
                        try:
                            patient_dob = datetime.strptime(parts[7], '%Y%m%d')
                        except ValueError:
                            logger.warning(f"Could not parse patient DOB: {parts[7]}")

                    if len(parts) > 8:
                        patient_gender = parts[8]

                elif line.startswith('PV1'):
                    parts = line.split('|')
                    if len(parts) > 2:
                        visit_class = parts[2]
                    if len(parts) > 3:
                        assigned_location = parts[3]
                    if len(parts) > 19:
                        visit_number = parts[19]

        # Create and return ADTMessage object
        return ADTMessage(
            message_type="ADT",
            event_type=event_type,
            message_control_id=message_control_id,
            message_datetime=message_datetime,
            raw_message=raw_message,
            patient_id=patient_id,
            patient_id_type=patient_id_type,
            patient_id_authority=patient_id_authority,
            patient_name_family=patient_name_family,
            patient_name_given=patient_name_given,
            patient_dob=patient_dob,
            patient_gender=patient_gender,
            visit_number=visit_number,
            visit_class=visit_class,
            assigned_location=assigned_location,
            admit_datetime=admit_datetime,
            discharge_datetime=discharge_datetime
        )
    except Exception as e:
        logger.error(f"Error extracting ADT data: {str(e)}")
        logger.error(f"Message: {message}")
        raise ValueError(f"Could not extract ADT data: {str(e)}")

def adt_to_fhir_resources(adt_message: ADTMessage) -> Dict[str, Any]:
    """
    Convert an ADT message to FHIR resources.

    Args:
        adt_message: The ADT message to convert

    Returns:
        Dictionary containing FHIR resources created from the ADT message
    """
    resources = {}

    # Create Patient resource
    patient = {
        "resourceType": "Patient",
        "id": str(uuid.uuid4()),
        "identifier": [
            {
                "system": adt_message.patient_id_authority or "http://example.org/identifier/mrn",
                "value": adt_message.patient_id,
                "type": {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/v2-0203",
                            "code": adt_message.patient_id_type or "MR",
                            "display": "Medical Record Number"
                        }
                    ]
                }
            }
        ],
        "active": True,
        "name": [
            {
                "family": adt_message.patient_name_family,
                "given": adt_message.patient_name_given,
                "use": "official"
            }
        ],
        # Add meta tags to indicate HL7 source
        "meta": {
            "tag": [
                {
                    "system": "http://clinicalsynthesishub.com/source",
                    "code": "hl7v2",
                    "display": "Created from HL7v2 message"
                },
                {
                    "system": "http://clinicalsynthesishub.com/hl7/message_type",
                    "code": adt_message.event_type,
                    "display": f"HL7 ADT-{adt_message.event_type}"
                }
            ]
        }
    }

    # Add gender if available
    if adt_message.patient_gender:
        gender_map = {
            'M': 'male',
            'F': 'female',
            'O': 'other',
            'U': 'unknown',
            'A': 'other',
            'N': 'other'
        }
        patient["gender"] = gender_map.get(adt_message.patient_gender, 'unknown')

    # Add birthDate if available
    if adt_message.patient_dob:
        patient["birthDate"] = adt_message.patient_dob.strftime('%Y-%m-%d')

    resources["Patient"] = patient

    # Create Encounter resource for ADT messages with visit information
    if adt_message.visit_number or adt_message.event_type in ['A01', 'A02', 'A03', 'A04', 'A05', 'A06', 'A07', 'A08']:
        # Map ADT event types to encounter status
        status_map = {
            'A01': 'in-progress',  # Admission
            'A02': 'in-progress',  # Transfer
            'A03': 'finished',     # Discharge
            'A04': 'in-progress',  # Registration
            'A05': 'planned',      # Pre-admission
            'A06': 'in-progress',  # Transfer outpatient to inpatient
            'A07': 'in-progress',  # Transfer inpatient to outpatient
            'A08': 'in-progress',  # Update patient information
            'A09': 'in-progress',  # Patient departing
            'A10': 'in-progress',  # Patient arriving
            'A11': 'cancelled',    # Cancel admission
            'A12': 'cancelled',    # Cancel transfer
            'A13': 'cancelled',    # Cancel discharge
            'A14': 'in-progress',  # Pending admission
            'A15': 'in-progress',  # Pending transfer
            'A16': 'in-progress',  # Pending discharge
        }

        # Map PV1-2 (patient class) to encounter class
        class_map = {
            'I': 'inpatient',
            'O': 'ambulatory',
            'E': 'emergency',
            'P': 'outpatient',
            'R': 'recurring',
            'B': 'outpatient',
            'C': 'outpatient',
            'N': 'outpatient',
            'U': 'outpatient'
        }

        encounter = {
            "resourceType": "Encounter",
            "id": str(uuid.uuid4()),
            "status": status_map.get(adt_message.event_type, 'unknown'),
            "class": {
                "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                "code": class_map.get(adt_message.visit_class, 'AMB'),
                "display": "ambulatory"
            },
            "subject": {
                "reference": f"Patient/{patient['id']}",
                "display": f"{adt_message.patient_name_family}, {' '.join(adt_message.patient_name_given)}"
            },
            # Add meta tags to indicate HL7 source
            "meta": {
                "tag": [
                    {
                        "system": "http://clinicalsynthesishub.com/source",
                        "code": "hl7v2",
                        "display": "Created from HL7v2 message"
                    },
                    {
                        "system": "http://clinicalsynthesishub.com/hl7/message_type",
                        "code": adt_message.event_type,
                        "display": f"HL7 ADT-{adt_message.event_type}"
                    },
                    {
                        "system": "http://clinicalsynthesishub.com/hl7/event_meaning",
                        "code": status_map.get(adt_message.event_type, 'unknown'),
                        "display": next((desc for code, status in status_map.items()
                                      if code == adt_message.event_type
                                      for desc in ["Admission", "Transfer", "Discharge", "Registration", "Pre-admission"]
                                      if desc.lower() in status), "Other")
                    }
                ]
            }
        }

        # Add identifiers if visit number is available
        if adt_message.visit_number:
            encounter["identifier"] = [
                {
                    "system": "http://example.org/identifier/visit",
                    "value": adt_message.visit_number
                }
            ]

        # Add period if admit/discharge dates are available
        period = {}
        if adt_message.admit_datetime:
            period["start"] = adt_message.admit_datetime.isoformat()
        if adt_message.discharge_datetime:
            period["end"] = adt_message.discharge_datetime.isoformat()
        if period:
            encounter["period"] = period

        # Add location if available
        if adt_message.assigned_location:
            encounter["location"] = [
                {
                    "status": "active",
                    "location": {
                        "reference": f"Location/{adt_message.assigned_location}",
                        "display": adt_message.assigned_location
                    }
                }
            ]

        resources["Encounter"] = encounter

    return resources



def process_hl7_message(message_str: str) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to FHIR resources.

    Args:
        message_str: The raw HL7 message string

    Returns:
        Dictionary containing FHIR resources created from the HL7 message
    """
    # Parse the message
    parsed_message, message_type = parse_hl7_message(message_str)

    # Process based on message type
    if message_type.startswith('ADT'):
        # Extract ADT data
        adt_message = extract_adt_data(parsed_message)

        # Convert to FHIR resources
        return adt_to_fhir_resources(adt_message)
    elif message_type.startswith('ORU'):
        # ORU messages are now handled by the Lab Microservice
        raise ValueError("ORU messages should be sent to the Lab Microservice")
    else:
        raise ValueError(f"Unsupported message type: {message_type}")
